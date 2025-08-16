#include "plugin.h"
#include "logging.h"
#include "platform.h"

static plugin_t *loaded_plugins = NULL;
static size_t plugin_count = 0;
static size_t plugin_capacity = 0;

bool plugin_system_init(void) {
    LOG_INFO_MSG("Initializing plugin system");
    return true;
}

void plugin_system_cleanup(void) {
    // Unload all plugins
    for (size_t i = 0; i < plugin_count; i++) {
        plugin_unload(&loaded_plugins[i]);
    }
    
    if (loaded_plugins) {
        free(loaded_plugins);
        loaded_plugins = NULL;
    }
    
    plugin_count = 0;
    plugin_capacity = 0;
    
    LOG_INFO_MSG("Plugin system cleanup completed");
}

bool plugin_load(const char *path, plugin_t *plugin) {
    if (!path || !plugin) {
        LOG_ERROR_MSG("Invalid plugin path or plugin structure");
        return false;
    }
    
    // Open shared library
    plugin->handle = platform_dlopen(path);
    if (!plugin->handle) {
        LOG_ERROR_MSG("Failed to load plugin %s: %s", path, platform_dlerror());
        return false;
    }
    
    // Get plugin info function
    (void)platform_dlerror(); // Clear any existing error if applicable
    void *get_info_ptr = platform_dlsym(plugin->handle, "plugin_get_info");
    plugin_get_info_func get_info = *(plugin_get_info_func*)&get_info_ptr;
    if (!get_info) {
    LOG_ERROR_MSG("Plugin %s missing required function 'plugin_get_info': %s", path, platform_dlerror());
    platform_dlclose(plugin->handle);
        return false;
    }
    
    // Get plugin functions
    void *init_ptr = platform_dlsym(plugin->handle, "plugin_init");
    plugin->init = *(plugin_init_func*)&init_ptr;
    
    void *cleanup_ptr = platform_dlsym(plugin->handle, "plugin_cleanup");
    plugin->cleanup = *(plugin_cleanup_func*)&cleanup_ptr;
    
    void *execute_ptr = platform_dlsym(plugin->handle, "plugin_execute");
    plugin->execute = *(plugin_execute_func*)&execute_ptr;
    
    if (!plugin->init || !plugin->cleanup || !plugin->execute) {
        LOG_ERROR_MSG("Plugin %s missing required functions", path);
    platform_dlclose(plugin->handle);
        return false;
    }
    
    // Get plugin information
    if (!get_info(&plugin->info)) {
        LOG_ERROR_MSG("Failed to get plugin info for %s", path);
    platform_dlclose(plugin->handle);
        return false;
    }
    
    // Initialize plugin
    if (!plugin->init()) {
        LOG_ERROR_MSG("Failed to initialize plugin %s", plugin->info.name);
    platform_dlclose(plugin->handle);
        return false;
    }
    
    // Store plugin path
    strncpy(plugin->path, path, sizeof(plugin->path) - 1);
    plugin->path[sizeof(plugin->path) - 1] = '\0';
    
    // Add to loaded plugins list
    if (!plugin_register(plugin)) {
        plugin->cleanup();
        dlclose(plugin->handle);
        return false;
    }
    
    LOG_INFO_MSG("Loaded plugin: %s v%s by %s", 
                 plugin->info.name, plugin->info.version, plugin->info.author);
    
    return true;
}

void plugin_unload(plugin_t *plugin) {
    if (!plugin || !plugin->handle) {
        return;
    }
    
    LOG_INFO_MSG("Unloading plugin: %s", plugin->info.name);
    
    // Cleanup plugin
    if (plugin->cleanup) {
        plugin->cleanup();
    }
    
    // Close shared library
    platform_dlclose(plugin->handle);
    
    // Clear plugin structure
    memset(plugin, 0, sizeof(plugin_t));
}

bool plugin_register(plugin_t *plugin) {
    if (!plugin) {
        return false;
    }
    
    // Expand plugin array if needed
    if (plugin_count >= plugin_capacity) {
        size_t new_capacity = plugin_capacity == 0 ? 8 : plugin_capacity * 2;
        plugin_t *new_plugins = realloc(loaded_plugins, new_capacity * sizeof(plugin_t));
        if (!new_plugins) {
            LOG_ERROR_MSG("Failed to expand plugin array");
            return false;
        }
        loaded_plugins = new_plugins;
        plugin_capacity = new_capacity;
    }
    
    // Copy plugin to array
    memcpy(&loaded_plugins[plugin_count], plugin, sizeof(plugin_t));
    plugin_count++;
    
    return true;
}

plugin_t* plugin_find_by_name(const char *name) {
    if (!name) {
        return NULL;
    }
    
    for (size_t i = 0; i < plugin_count; i++) {
        if (strcmp(loaded_plugins[i].info.name, name) == 0) {
            return &loaded_plugins[i];
        }
    }
    
    return NULL;
}

plugin_t* plugin_get_all(size_t *count) {
    if (count) {
        *count = plugin_count;
    }
    return loaded_plugins;
}

bool plugin_execute_by_name(const char *name, const char *input, char *output, size_t output_size) {
    plugin_t *plugin = plugin_find_by_name(name);
    if (!plugin) {
        LOG_ERROR_MSG("Plugin not found: %s", name);
        return false;
    }
    
    return plugin->execute(input, output, output_size);
}

bool plugin_load_from_directory(const char *directory) {
    if (!directory) {
        LOG_ERROR_MSG("Plugin directory path is NULL");
        return false;
    }
    
    DIR *dir = opendir(directory);
    if (!dir) {
        LOG_WARN_MSG("Failed to open plugin directory %s: %s", directory, strerror(errno));
        return false;
    }
    
    struct dirent *entry;
    bool loaded_any = false;
    
    while ((entry = readdir(dir)) != NULL) {
        // Skip hidden files and directories
        if (entry->d_name[0] == '.') {
            continue;
        }
        
    // Check for shared library extension (platform-specific)
    const char *ext = strrchr(entry->d_name, '.');
#ifdef PLATFORM_MACOS
    const char *wanted_ext = ".dylib";
#else
    const char *wanted_ext = ".so";
#endif
    if (!ext || strcmp(ext, wanted_ext) != 0) {
        continue;
    }
        
        // Build full path
        char plugin_path[PATH_MAX];
        snprintf(plugin_path, sizeof(plugin_path), "%s/%s", directory, entry->d_name);
        
        // Load plugin
        plugin_t plugin;
        memset(&plugin, 0, sizeof(plugin));
        
        if (plugin_load(plugin_path, &plugin)) {
            loaded_any = true;
        }
    }
    
    closedir(dir);
    
    if (loaded_any) {
        LOG_INFO_MSG("Loaded plugins from directory: %s", directory);
    } else {
        LOG_WARN_MSG("No plugins loaded from directory: %s", directory);
    }
    
    return loaded_any;
}

