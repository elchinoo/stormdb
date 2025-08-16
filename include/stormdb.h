#ifndef STORMDB_H
#define STORMDB_H

#include "platform.h"

// Application version
#define STORMDB_VERSION "1.0.0"
#define STORMDB_API_VERSION "1.0"

// Default configuration
#define DEFAULT_CONFIG_FILE "config/stormdb.yaml"
#ifdef PLATFORM_WINDOWS
    #define DEFAULT_PID_FILE PLATFORM_DEFAULT_PID_FILE
#else
    #define DEFAULT_PID_FILE "/tmp/stormdb.pid"
#endif
#define MAX_METRICS_QUEUE_SIZE 10000
#define MAX_PATH_LENGTH PLATFORM_MAX_PATH
#define MAX_NAME_LENGTH 256

// Exit codes
#define EXIT_SUCCESS 0
#define EXIT_FAILURE 1
#define EXIT_CONFIG_ERROR 2
#define EXIT_DATABASE_ERROR 3
#define EXIT_ALREADY_RUNNING 4

// Common structures
typedef struct {
    char *config_file;
    char *log_level;
    char *log_file;
    bool verbose;
    bool help;
    bool version;
    char *database_url;
    char *plugin_dir;
} cmdline_options_t;

#endif // STORMDB_H
