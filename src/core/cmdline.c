#include "cmdline.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>

static struct option long_options[] = {
    {"config",    required_argument, 0, 'c'},
    {"host",      required_argument, 0, 'h'},
    {"port",      required_argument, 0, 'p'},
    {"database",  required_argument, 0, 'd'},
    {"user",      required_argument, 0, 'u'},
    {"password",  required_argument, 0, 'P'},
    {"api-port",  required_argument, 0, 'a'},
    {"verbose",   no_argument,       0, 'v'},
    {"quiet",     no_argument,       0, 'q'},
    {"version",   no_argument,       0, 'V'},
    {"help",      no_argument,       0,  0 },
    {0, 0, 0, 0}
};

void cmdline_init_args(cmdline_args_t *args) {
    if (!args) return;
    
    memset(args, 0, sizeof(cmdline_args_t));
    
    // Set defaults
    args->config_file = strdup("stormdb.yaml");
    args->database_port = 0; // Use config default
    args->api_port = 0; // Use config default
    args->log_level = LOG_INFO;
    args->verbose = false;
    args->quiet = false;
    args->show_help = false;
    args->show_version = false;
}

void cmdline_free_args(cmdline_args_t *args) {
    if (!args) return;
    
    if (args->config_file) {
        free(args->config_file);
        args->config_file = NULL;
    }
    if (args->database_host) {
        free(args->database_host);
        args->database_host = NULL;
    }
    if (args->database_name) {
        free(args->database_name);
        args->database_name = NULL;
    }
    if (args->database_user) {
        free(args->database_user);
        args->database_user = NULL;
    }
    if (args->database_password) {
        free(args->database_password);
        args->database_password = NULL;
    }
}

void cmdline_print_help(const char *program_name) {
    printf("Usage: %s [OPTIONS]\n\n", program_name);
    printf("StormDB - PostgreSQL Performance Testing Tool\n\n");
    printf("Options:\n");
    printf("  -c, --config FILE     Configuration file path (default: stormdb.yaml)\n");
    printf("  -h, --host HOST       Database host (overrides config)\n");
    printf("  -p, --port PORT       Database port (overrides config)\n");
    printf("  -d, --database DB     Database name (overrides config)\n");
    printf("  -u, --user USER       Database user (overrides config)\n");
    printf("  -P, --password PASS   Database password (overrides config)\n");
    printf("  -a, --api-port PORT   API server port (overrides config)\n");
    printf("  -v, --verbose         Enable debug logging\n");
    printf("  -q, --quiet           Disable console output\n");
    printf("  -V, --version         Show version information\n");
    printf("      --help            Show this help message\n");
    printf("\n");
    printf("Examples:\n");
    printf("  %s                    # Start with default configuration\n", program_name);
    printf("  %s -c /etc/stormdb.yaml  # Use custom config file\n", program_name);
    printf("  %s -h localhost -p 5432  # Override database connection\n", program_name);
    printf("  %s -v                 # Enable verbose logging\n", program_name);
}

void cmdline_print_version(void) {
    printf("StormDB v%s\n", STORMDB_VERSION);
    printf("PostgreSQL Performance Testing Tool\n");
    printf("Built with C11 on %s %s\n", __DATE__, __TIME__);
}

bool cmdline_parse(int argc, char *argv[], cmdline_args_t *args) {
    if (!args) {
        fprintf(stderr, "Error: args parameter is NULL\n");
        return false;
    }
    
    cmdline_init_args(args);
    
    int c;
    int option_index = 0;
    
    while ((c = getopt_long(argc, argv, "c:h:p:d:u:P:a:vqV", long_options, &option_index)) != -1) {
        switch (c) {
            case 'c':
                if (args->config_file) free(args->config_file);
                args->config_file = strdup(optarg);
                break;
            case 'h':
                if (args->database_host) free(args->database_host);
                args->database_host = strdup(optarg);
                break;
            case 'p':
                args->database_port = atoi(optarg);
                if (args->database_port <= 0 || args->database_port > 65535) {
                    fprintf(stderr, "Error: Invalid port number: %s\n", optarg);
                    cmdline_free_args(args);
                    return false;
                }
                break;
            case 'd':
                if (args->database_name) free(args->database_name);
                args->database_name = strdup(optarg);
                break;
            case 'u':
                if (args->database_user) free(args->database_user);
                args->database_user = strdup(optarg);
                break;
            case 'P':
                if (args->database_password) free(args->database_password);
                args->database_password = strdup(optarg);
                break;
            case 'a':
                args->api_port = atoi(optarg);
                if (args->api_port <= 0 || args->api_port > 65535) {
                    fprintf(stderr, "Error: Invalid API port number: %s\n", optarg);
                    cmdline_free_args(args);
                    return false;
                }
                break;
            case 'v':
                args->verbose = true;
                args->log_level = LOG_DEBUG;
                break;
            case 'q':
                args->quiet = true;
                break;
            case 'V':
                args->show_version = true;
                return true;
            case 0:
                // Long option
                if (strcmp(long_options[option_index].name, "help") == 0) {
                    args->show_help = true;
                    return true;
                }
                break;
            case '?':
                // Invalid option
                cmdline_free_args(args);
                return false;
            default:
                fprintf(stderr, "Error: Unknown option\n");
                cmdline_free_args(args);
                return false;
        }
    }
    
    // Check for non-option arguments
    if (optind < argc) {
        fprintf(stderr, "Error: Unexpected argument: %s\n", argv[optind]);
        cmdline_free_args(args);
        return false;
    }
    
    return true;
}
