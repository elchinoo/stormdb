#include "stormdb.h"
#include "logging.h"
#include "config.h"
#include "cmdline.h"
#include "pidfile.h"
#include "database.h"
#include "plugin.h"
#include "api.h"
#include "metrics.h"
#include "memory.h"

static volatile bool running = true;
static stormdb_config_t *global_config = NULL;
static cmdline_args_t g_args; // store parsed args for reload context

static void signal_handler(int sig) {
#ifdef PLATFORM_UNIX
    switch (sig) {
        case SIGINT:
        case SIGTERM:
            LOG_INFO_MSG("Received signal %d, shutting down gracefully", sig);
            running = false;
            break;
        case SIGHUP:
            LOG_INFO_MSG("Received SIGHUP, reloading configuration");
            if (config_reload()) {
                const stormdb_config_t *cfg = config_get();
                if (cfg) {
                    // Apply logging changes
                    logging_set_level(cfg->logging.level);
                    if (cfg->logging.file[0] != '\0') {
                        logging_set_file(cfg->logging.file);
                    }
                    // Apply rotation settings
                    logging_set_rotation(cfg->logging.max_size, cfg->logging.max_files);
                    LOG_INFO_MSG("Applied logging changes after reload");

                    // Restart API if port changed
                    static int last_api_port = 0;
                    if (last_api_port == 0) last_api_port = cfg->api.port;
                    if (cfg->api.port != last_api_port) {
                        LOG_INFO_MSG("API port changed from %d to %d, restarting", last_api_port, cfg->api.port);
                        if (!api_restart(cfg->api.port)) {
                            LOG_ERROR_MSG("API restart failed on reload");
                        }
                        last_api_port = cfg->api.port;
                    }

                    // Reconnect DB if needed and re-ensure schema/version
                    if (!database_is_connected()) {
                        if (!database_reconnect()) {
                            LOG_ERROR_MSG("Database reconnect failed after reload");
                        }
                    }
                    (void)database_ensure_schema();
                    (void)database_check_version("1.0");

                    // Optionally auto-load plugins on reload
                    if (cfg->plugin.auto_load && cfg->plugin.plugin_dir[0] != '\0') {
                        plugin_load_from_directory(cfg->plugin.plugin_dir);
                    }
                }
            } else {
                LOG_ERROR_MSG("Configuration reload failed");
            }
            break;
        default:
            LOG_WARN_MSG("Received unhandled signal %d", sig);
            break;
    }
#else
    // Windows console control handler
    LOG_INFO_MSG("Received shutdown signal, shutting down gracefully");
    running = false;
#endif
}

static void setup_signal_handlers(void) {
#ifdef PLATFORM_UNIX
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = signal_handler;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = SA_RESTART;
    
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGHUP, &sa, NULL);
    
    // Ignore SIGPIPE (broken connections)
    signal(SIGPIPE, SIG_IGN);
#else
    // Windows console control handler
    platform_signal_set_handler(SIGINT, signal_handler);
    platform_signal_set_handler(SIGTERM, signal_handler);
#endif
}

static void cleanup_and_exit(int exit_code) {
    LOG_INFO_MSG("Starting cleanup process");
    
    // Cleanup in reverse order of initialization
    plugin_system_cleanup();
    database_cleanup();
    
    if (global_config) {
        config_free(global_config);
        global_config = NULL;
    }
    
    pidfile_remove();
    logging_cleanup();
    
    LOG_INFO_MSG("Cleanup completed, exiting with code %d", exit_code);
    exit(exit_code);
}

int main(int argc, char *argv[]) {
    // Parse command line arguments first (before logging init)
    if (!cmdline_parse(argc, argv, &g_args)) {
        fprintf(stderr, "Failed to parse command line arguments\n");
        return EXIT_FAILURE;
    }
    
    // Handle help and version early
    if (g_args.show_help) {
        cmdline_print_help(argv[0]);
        cmdline_free_args(&g_args);
        return EXIT_SUCCESS;
    }
    
    if (g_args.show_version) {
        cmdline_print_version();
        cmdline_free_args(&g_args);
        return EXIT_SUCCESS;
    }
    
    // Initialize logging
    if (!logging_init(g_args.log_level)) {
        fprintf(stderr, "Failed to initialize logging\n");
        cmdline_free_args(&g_args);
        return EXIT_FAILURE;
    }
    
    LOG_INFO_MSG("StormDB starting up");
    if (g_args.verbose) {
        LOG_DEBUG_MSG("Debug logging enabled");
    }
    
    // Setup signal handlers
    setup_signal_handlers();
    
    // Load configuration
    global_config = config_load(g_args.config_file);
    // Initialize memory subsystem
    if (!memory_init()) {
        LOG_ERROR_MSG("Failed to initialize memory subsystem");
        cleanup_and_exit(EXIT_FAILURE);
    }

    if (!global_config) {
        LOG_ERROR_MSG("Failed to load configuration from %s", g_args.config_file);
        cleanup_and_exit(EXIT_FAILURE);
    }
    
    // Override config with command line arguments
    if (g_args.database_host) {
        strncpy(global_config->database.host, g_args.database_host, sizeof(global_config->database.host) - 1);
    }
    if (g_args.database_port > 0) {
        global_config->database.port = g_args.database_port;
    }
    if (g_args.api_port > 0) {
        global_config->api.port = g_args.api_port;
    }

    // Apply logging from config and quiet flag
    if (g_args.quiet) {
        logging_set_level(LOG_ERROR);
    } else {
        logging_set_level(global_config->logging.level);
    }
    if (global_config->logging.file[0] != '\0') {
        logging_set_file(global_config->logging.file);
    }
    logging_set_rotation(global_config->logging.max_size, global_config->logging.max_files);
    
    // Create PID file
    if (global_config->daemon.pid_file[0] != '\0') {
        if (!pidfile_create(global_config->daemon.pid_file)) {
            LOG_ERROR_MSG("Failed to create PID file, another instance may be running");
            cleanup_and_exit(EXIT_FAILURE);
        }
    }
    
    // Initialize database connection
    if (!database_init(&global_config->database)) {
        LOG_ERROR_MSG("Failed to initialize database connection");
        cleanup_and_exit(EXIT_FAILURE);
    }
    // Ensure schema/version non-fatally
    if (!database_ensure_schema()) {
        LOG_WARN_MSG("Database schema ensure failed");
    } else {
        (void)database_check_version("1.0");
    }
    
    LOG_INFO_MSG("Database connection established");
    
    // Initialize plugin system
    if (!plugin_system_init()) {
        LOG_ERROR_MSG("Failed to initialize plugin system");
        cleanup_and_exit(EXIT_FAILURE);
    }
    
    // Load plugins from directory
    if (global_config->plugin.auto_load && global_config->plugin.plugin_dir[0] != '\0') {
        plugin_load_from_directory(global_config->plugin.plugin_dir);
    }
    
    LOG_INFO_MSG("StormDB initialization completed successfully");
    LOG_INFO_MSG("API server listening on port %d", global_config->api.port);
    LOG_INFO_MSG("Press Ctrl+C to stop the server");

    // Start API thread
    if (!api_start(global_config->api.port)) {
        LOG_ERROR_MSG("Failed to start API thread");
        cleanup_and_exit(EXIT_FAILURE);
    }

    // Start metrics consumer thread
    if (!metrics_init((size_t)global_config->metrics.buffer_size)) {
        LOG_ERROR_MSG("Failed to initialize metrics subsystem");
        cleanup_and_exit(EXIT_FAILURE);
    }
    if (!metrics_start()) {
        LOG_ERROR_MSG("Failed to start metrics thread");
        cleanup_and_exit(EXIT_FAILURE);
    }
    
    // Main event loop
    while (running) {
        // Check database connection
        if (!database_is_connected()) {
            LOG_WARN_MSG("Database connection lost");
            if (!database_reconnect()) {
                LOG_ERROR_MSG("Failed to reconnect to database");
                cleanup_and_exit(EXIT_FAILURE);
            }
        }
        
        // Sleep for a short period to avoid busy waiting
        usleep(100000); // 100ms
    }
    
    LOG_INFO_MSG("Main loop exited, beginning shutdown");
    api_stop();
    metrics_stop();
    metrics_cleanup();
    cmdline_free_args(&g_args);
    cleanup_and_exit(EXIT_SUCCESS);
    
    return EXIT_SUCCESS; // Never reached
}
