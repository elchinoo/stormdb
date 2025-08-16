#ifndef CONFIG_H
#define CONFIG_H

#include "stormdb.h"
#include "logging.h"

// Database configuration structure
typedef struct {
    char host[256];
    int port;
    char database[64];
    char user[64];
    char password[256];
    int connect_timeout;
} database_config_t;

// API configuration structure
typedef struct {
    char host[256];
    int port;
    int max_connections;
} api_config_t;

// Plugin configuration structure
typedef struct {
    char plugin_dir[512];
    bool auto_load;
} plugin_config_t;

// Daemon configuration structure
typedef struct {
    char pid_file[512];
    char user[64];
    char group[64];
} daemon_config_t;

// Logging configuration structure
typedef struct {
    log_level_t level;
    char file[512];
    size_t max_size;
    int max_files;
} logging_config_t;

// Metrics configuration structure
typedef struct {
    int collection_interval;
    int buffer_size;
    char export_format[32];
} metrics_config_t;

// Main configuration structure
typedef struct {
    database_config_t database;
    api_config_t api;
    plugin_config_t plugin;
    daemon_config_t daemon;
    logging_config_t logging;
    metrics_config_t metrics;
} stormdb_config_t;

// Configuration functions
bool config_init(void);
void config_cleanup(void);
stormdb_config_t* config_load(const char *config_file);
void config_free(stormdb_config_t *config);
bool config_reload(void);
const stormdb_config_t *config_get(void);
// Note: config_reload() reloads from the last path passed to config_load()

// Utility functions
void config_set_defaults(stormdb_config_t *config);

#endif // CONFIG_H
