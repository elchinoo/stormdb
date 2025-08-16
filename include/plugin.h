#ifndef STORMDB_PLUGIN_H
#define STORMDB_PLUGIN_H

#include "stormdb.h"
#include <stdbool.h>
#include <dlfcn.h>
#include <dirent.h>
#include <limits.h>

// Plugin information structure
typedef struct {
    char name[64];
    char version[16];
    char author[64];
    char description[256];
} plugin_info_t;

// Plugin function types
typedef bool (*plugin_get_info_func)(plugin_info_t *info);
typedef bool (*plugin_init_func)(void);
typedef void (*plugin_cleanup_func)(void);
typedef bool (*plugin_execute_func)(const char *input, char *output, size_t output_size);

// Plugin structure
typedef struct {
    char path[PATH_MAX];
    void *handle;
    plugin_info_t info;
    
    // Plugin functions
    plugin_init_func init;
    plugin_cleanup_func cleanup;
    plugin_execute_func execute;
} plugin_t;

// Plugin system functions
bool plugin_system_init(void);
void plugin_system_cleanup(void);

// Plugin loading/unloading
bool plugin_load(const char *path, plugin_t *plugin);
void plugin_unload(plugin_t *plugin);
bool plugin_load_from_directory(const char *directory);

// Plugin management
bool plugin_register(plugin_t *plugin);
plugin_t* plugin_find_by_name(const char *name);
plugin_t* plugin_get_all(size_t *count);

// Plugin execution
bool plugin_execute_by_name(const char *name, const char *input, char *output, size_t output_size);

#endif // STORMDB_PLUGIN_H
