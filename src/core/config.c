#include "config.h"
#include "logging.h"
#include <yaml.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>

static stormdb_config_t *current_config = NULL;
static char *current_config_path = NULL;

static log_level_t parse_log_level(const char *s) {
    if (!s) return LOG_INFO;
    char tmp[16];
    size_t n = strlen(s);
    if (n >= sizeof(tmp)) n = sizeof(tmp) - 1;
    for (size_t i = 0; i < n; i++) tmp[i] = (char)toupper((unsigned char)s[i]);
    tmp[n] = '\0';
    if (strcmp(tmp, "ERROR") == 0) return LOG_ERROR;
    if (strcmp(tmp, "WARN") == 0 || strcmp(tmp, "WARNING") == 0) return LOG_WARN;
    if (strcmp(tmp, "INFO") == 0) return LOG_INFO;
    if (strcmp(tmp, "DEBUG") == 0) return LOG_DEBUG;
    if (strcmp(tmp, "TRACE") == 0) return LOG_TRACE;
    return LOG_INFO;
}

void config_set_defaults(stormdb_config_t *config) {
    if (!config) return;
    
    // Database defaults
    strncpy(config->database.host, "localhost", sizeof(config->database.host) - 1);
    config->database.port = 5432;
    strncpy(config->database.database, "stormdb", sizeof(config->database.database) - 1);
    strncpy(config->database.user, "stormdb", sizeof(config->database.user) - 1);
    strncpy(config->database.password, "", sizeof(config->database.password) - 1);
    config->database.connect_timeout = 30;
    
    // API defaults
    strncpy(config->api.host, "0.0.0.0", sizeof(config->api.host) - 1);
    config->api.port = 8080;
    config->api.max_connections = 100;
    
    // Plugin defaults
    strncpy(config->plugin.plugin_dir, "./plugins", sizeof(config->plugin.plugin_dir) - 1);
    config->plugin.auto_load = true;
    
    // Daemon defaults
    strncpy(config->daemon.pid_file, "/tmp/stormdb.pid", sizeof(config->daemon.pid_file) - 1);
    config->daemon.user[0] = '\0';
    config->daemon.group[0] = '\0';
    
    // Logging defaults
    config->logging.level = LOG_INFO;
    config->logging.file[0] = '\0';
    config->logging.max_size = 104857600; // 100MB
    config->logging.max_files = 5;
    
    // Metrics defaults
    config->metrics.collection_interval = 1000;
    config->metrics.buffer_size = 10000;
    strncpy(config->metrics.export_format, "json", sizeof(config->metrics.export_format) - 1);
    
    // Memory defaults (256MB)
    config->memory.buffer_size_bytes = 268435456;
}

stormdb_config_t* config_load(const char *config_file) {
    if (!config_file) {
        LOG_ERROR_MSG("Configuration file path is NULL");
        return NULL;
    }
    
    FILE *file = fopen(config_file, "r");
    if (!file) {
        LOG_ERROR_MSG("Failed to open configuration file: %s", config_file);
        return NULL;
    }
    
    stormdb_config_t *config = malloc(sizeof(stormdb_config_t));
    if (!config) {
        LOG_ERROR_MSG("Failed to allocate memory for configuration");
        fclose(file);
        return NULL;
    }
    
    // Set defaults first
    config_set_defaults(config);
    
    yaml_parser_t parser;
    yaml_document_t document;
    
    if (!yaml_parser_initialize(&parser)) {
        LOG_ERROR_MSG("Failed to initialize YAML parser");
        free(config);
        fclose(file);
        return NULL;
    }
    
    yaml_parser_set_input_file(&parser, file);
    
    if (!yaml_parser_load(&parser, &document)) {
        LOG_ERROR_MSG("Failed to parse YAML configuration file");
        yaml_parser_delete(&parser);
        free(config);
        fclose(file);
        return NULL;
    }
    
    // Parse YAML configuration
    yaml_node_t *root = yaml_document_get_root_node(&document);
    if (root && root->type == YAML_MAPPING_NODE) {
        yaml_node_pair_t *pair;
        for (pair = root->data.mapping.pairs.start; pair < root->data.mapping.pairs.top; pair++) {
            yaml_node_t *key_node = yaml_document_get_node(&document, pair->key);
            yaml_node_t *value_node = yaml_document_get_node(&document, pair->value);
            
            if (key_node->type == YAML_SCALAR_NODE && value_node->type == YAML_MAPPING_NODE) {
                const char *section = (const char *)key_node->data.scalar.value;
                
                if (strcmp(section, "database") == 0) {
                    // Parse database configuration
                    yaml_node_pair_t *db_pair;
                    for (db_pair = value_node->data.mapping.pairs.start; db_pair < value_node->data.mapping.pairs.top; db_pair++) {
                        yaml_node_t *db_key = yaml_document_get_node(&document, db_pair->key);
                        yaml_node_t *db_value = yaml_document_get_node(&document, db_pair->value);
                        
                        if (db_key->type == YAML_SCALAR_NODE && db_value->type == YAML_SCALAR_NODE) {
                            const char *key = (const char *)db_key->data.scalar.value;
                            const char *value = (const char *)db_value->data.scalar.value;
                            
                            if (strcmp(key, "host") == 0) {
                                strncpy(config->database.host, value, sizeof(config->database.host) - 1);
                            } else if (strcmp(key, "port") == 0) {
                                config->database.port = atoi(value);
                            } else if (strcmp(key, "database") == 0) {
                                strncpy(config->database.database, value, sizeof(config->database.database) - 1);
                            } else if (strcmp(key, "user") == 0) {
                                strncpy(config->database.user, value, sizeof(config->database.user) - 1);
                            } else if (strcmp(key, "password") == 0) {
                                strncpy(config->database.password, value, sizeof(config->database.password) - 1);
                            } else if (strcmp(key, "connect_timeout") == 0) {
                                config->database.connect_timeout = atoi(value);
                            }
                        }
                    }
                } else if (strcmp(section, "api") == 0) {
                    // Parse API configuration
                    yaml_node_pair_t *api_pair;
                    for (api_pair = value_node->data.mapping.pairs.start; api_pair < value_node->data.mapping.pairs.top; api_pair++) {
                        yaml_node_t *api_key = yaml_document_get_node(&document, api_pair->key);
                        yaml_node_t *api_value = yaml_document_get_node(&document, api_pair->value);
                        
                        if (api_key->type == YAML_SCALAR_NODE && api_value->type == YAML_SCALAR_NODE) {
                            const char *key = (const char *)api_key->data.scalar.value;
                            const char *value = (const char *)api_value->data.scalar.value;
                            
                            if (strcmp(key, "host") == 0) {
                                strncpy(config->api.host, value, sizeof(config->api.host) - 1);
                            } else if (strcmp(key, "port") == 0) {
                                config->api.port = atoi(value);
                            } else if (strcmp(key, "max_connections") == 0) {
                                config->api.max_connections = atoi(value);
                            }
                        }
                    }
                }
                // TODO: Parse other sections (plugin, daemon, logging, metrics)
            }
        }
    }
    
    yaml_document_delete(&document);
    yaml_parser_delete(&parser);
    fclose(file);
    
    // Remember the config file path for reloads
    if (current_config_path) {
        free(current_config_path);
        current_config_path = NULL;
    }
    current_config_path = strdup(config_file);

    current_config = config;
    LOG_INFO_MSG("Configuration loaded successfully from %s", config_file);
    
    return config;
}

void config_free(stormdb_config_t *config) {
    if (config) {
        free(config);
        if (config == current_config) {
            current_config = NULL;
        }
    }
}

const stormdb_config_t *config_get(void) {
    return current_config;
}

bool config_init(void) {
    return true;
}

void config_cleanup(void) {
    if (current_config) {
        config_free(current_config);
        current_config = NULL;
    }
}

bool config_reload(void) {
    if (!current_config_path) {
        LOG_ERROR_MSG("Cannot reload: no stored configuration path");
        return false;
    }

    stormdb_config_t *new_cfg = NULL;
    FILE *file = fopen(current_config_path, "r");
    if (!file) {
        LOG_ERROR_MSG("Failed to open configuration file for reload: %s", current_config_path);
        return false;
    }

    new_cfg = malloc(sizeof(stormdb_config_t));
    if (!new_cfg) {
        fclose(file);
        LOG_ERROR_MSG("Failed to allocate memory for new configuration");
        return false;
    }

    // Start from defaults
    config_set_defaults(new_cfg);

    yaml_parser_t parser;
    yaml_document_t document;
    if (!yaml_parser_initialize(&parser)) {
        LOG_ERROR_MSG("Failed to initialize YAML parser for reload");
        free(new_cfg);
        fclose(file);
        return false;
    }
    yaml_parser_set_input_file(&parser, file);
    if (!yaml_parser_load(&parser, &document)) {
        LOG_ERROR_MSG("Failed to parse YAML configuration file during reload");
        yaml_parser_delete(&parser);
        free(new_cfg);
        fclose(file);
        return false;
    }

    yaml_node_t *root = yaml_document_get_root_node(&document);
    if (root && root->type == YAML_MAPPING_NODE) {
        yaml_node_pair_t *pair;
        for (pair = root->data.mapping.pairs.start; pair < root->data.mapping.pairs.top; pair++) {
            yaml_node_t *key_node = yaml_document_get_node(&document, pair->key);
            yaml_node_t *value_node = yaml_document_get_node(&document, pair->value);
            if (!key_node || !value_node) continue;
            if (key_node->type != YAML_SCALAR_NODE) continue;
            const char *section = (const char *)key_node->data.scalar.value;

            if (strcmp(section, "database") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *db_pair;
                for (db_pair = value_node->data.mapping.pairs.start; db_pair < value_node->data.mapping.pairs.top; db_pair++) {
                    yaml_node_t *db_key = yaml_document_get_node(&document, db_pair->key);
                    yaml_node_t *db_value = yaml_document_get_node(&document, db_pair->value);
                    if (!db_key || !db_value) continue;
                    if (db_key->type == YAML_SCALAR_NODE && db_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)db_key->data.scalar.value;
                        const char *value = (const char *)db_value->data.scalar.value;
                        if (strcmp(key, "host") == 0) {
                            strncpy(new_cfg->database.host, value, sizeof(new_cfg->database.host) - 1);
                        } else if (strcmp(key, "port") == 0) {
                            new_cfg->database.port = atoi(value);
                        } else if (strcmp(key, "database") == 0) {
                            strncpy(new_cfg->database.database, value, sizeof(new_cfg->database.database) - 1);
                        } else if (strcmp(key, "user") == 0) {
                            strncpy(new_cfg->database.user, value, sizeof(new_cfg->database.user) - 1);
                        } else if (strcmp(key, "password") == 0) {
                            strncpy(new_cfg->database.password, value, sizeof(new_cfg->database.password) - 1);
                        } else if (strcmp(key, "connect_timeout") == 0) {
                            new_cfg->database.connect_timeout = atoi(value);
                        }
                    }
                }
            } else if (strcmp(section, "api") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *api_pair;
                for (api_pair = value_node->data.mapping.pairs.start; api_pair < value_node->data.mapping.pairs.top; api_pair++) {
                    yaml_node_t *api_key = yaml_document_get_node(&document, api_pair->key);
                    yaml_node_t *api_value = yaml_document_get_node(&document, api_pair->value);
                    if (!api_key || !api_value) continue;
                    if (api_key->type == YAML_SCALAR_NODE && api_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)api_key->data.scalar.value;
                        const char *value = (const char *)api_value->data.scalar.value;
                        if (strcmp(key, "host") == 0) {
                            strncpy(new_cfg->api.host, value, sizeof(new_cfg->api.host) - 1);
                        } else if (strcmp(key, "port") == 0) {
                            new_cfg->api.port = atoi(value);
                        } else if (strcmp(key, "max_connections") == 0) {
                            new_cfg->api.max_connections = atoi(value);
                        }
                    }
                }
            } else if (strcmp(section, "plugin") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *pl_pair;
                for (pl_pair = value_node->data.mapping.pairs.start; pl_pair < value_node->data.mapping.pairs.top; pl_pair++) {
                    yaml_node_t *pl_key = yaml_document_get_node(&document, pl_pair->key);
                    yaml_node_t *pl_value = yaml_document_get_node(&document, pl_pair->value);
                    if (!pl_key || !pl_value) continue;
                    if (pl_key->type == YAML_SCALAR_NODE && pl_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)pl_key->data.scalar.value;
                        const char *value = (const char *)pl_value->data.scalar.value;
                        if (strcmp(key, "plugin_dir") == 0) {
                            strncpy(new_cfg->plugin.plugin_dir, value, sizeof(new_cfg->plugin.plugin_dir) - 1);
                        } else if (strcmp(key, "auto_load") == 0) {
                            new_cfg->plugin.auto_load = (strcmp(value, "true") == 0 || strcmp(value, "1") == 0 || strcasecmp(value, "yes") == 0);
                        }
                    }
                }
            } else if (strcmp(section, "daemon") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *dm_pair;
                for (dm_pair = value_node->data.mapping.pairs.start; dm_pair < value_node->data.mapping.pairs.top; dm_pair++) {
                    yaml_node_t *dm_key = yaml_document_get_node(&document, dm_pair->key);
                    yaml_node_t *dm_value = yaml_document_get_node(&document, dm_pair->value);
                    if (!dm_key || !dm_value) continue;
                    if (dm_key->type == YAML_SCALAR_NODE && dm_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)dm_key->data.scalar.value;
                        const char *value = (const char *)dm_value->data.scalar.value;
                        if (strcmp(key, "pid_file") == 0) {
                            strncpy(new_cfg->daemon.pid_file, value, sizeof(new_cfg->daemon.pid_file) - 1);
                        } else if (strcmp(key, "user") == 0) {
                            strncpy(new_cfg->daemon.user, value, sizeof(new_cfg->daemon.user) - 1);
                        } else if (strcmp(key, "group") == 0) {
                            strncpy(new_cfg->daemon.group, value, sizeof(new_cfg->daemon.group) - 1);
                        }
                    }
                }
            } else if (strcmp(section, "logging") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *lg_pair;
                for (lg_pair = value_node->data.mapping.pairs.start; lg_pair < value_node->data.mapping.pairs.top; lg_pair++) {
                    yaml_node_t *lg_key = yaml_document_get_node(&document, lg_pair->key);
                    yaml_node_t *lg_value = yaml_document_get_node(&document, lg_pair->value);
                    if (!lg_key || !lg_value) continue;
                    if (lg_key->type == YAML_SCALAR_NODE && lg_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)lg_key->data.scalar.value;
                        const char *value = (const char *)lg_value->data.scalar.value;
                        if (strcmp(key, "level") == 0) {
                            new_cfg->logging.level = parse_log_level(value);
                        } else if (strcmp(key, "file") == 0) {
                            strncpy(new_cfg->logging.file, value, sizeof(new_cfg->logging.file) - 1);
                        } else if (strcmp(key, "max_size") == 0) {
                            new_cfg->logging.max_size = (size_t)strtoull(value, NULL, 10);
                        } else if (strcmp(key, "max_files") == 0) {
                            new_cfg->logging.max_files = atoi(value);
                        }
                    }
                }
            } else if (strcmp(section, "metrics") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *mt_pair;
                for (mt_pair = value_node->data.mapping.pairs.start; mt_pair < value_node->data.mapping.pairs.top; mt_pair++) {
                    yaml_node_t *mt_key = yaml_document_get_node(&document, mt_pair->key);
                    yaml_node_t *mt_value = yaml_document_get_node(&document, mt_pair->value);
                    if (!mt_key || !mt_value) continue;
                    if (mt_key->type == YAML_SCALAR_NODE && mt_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)mt_key->data.scalar.value;
                        const char *value = (const char *)mt_value->data.scalar.value;
                        if (strcmp(key, "collection_interval") == 0) {
                            new_cfg->metrics.collection_interval = atoi(value);
                        } else if (strcmp(key, "buffer_size") == 0) {
                            new_cfg->metrics.buffer_size = atoi(value);
                        } else if (strcmp(key, "export_format") == 0) {
                            strncpy(new_cfg->metrics.export_format, value, sizeof(new_cfg->metrics.export_format) - 1);
                        }
                    }
                }
            } else if (strcmp(section, "memory") == 0 && value_node->type == YAML_MAPPING_NODE) {
                yaml_node_pair_t *m_pair;
                for (m_pair = value_node->data.mapping.pairs.start; m_pair < value_node->data.mapping.pairs.top; m_pair++) {
                    yaml_node_t *m_key = yaml_document_get_node(&document, m_pair->key);
                    yaml_node_t *m_value = yaml_document_get_node(&document, m_pair->value);
                    if (!m_key || !m_value) continue;
                    if (m_key->type == YAML_SCALAR_NODE && m_value->type == YAML_SCALAR_NODE) {
                        const char *key = (const char *)m_key->data.scalar.value;
                        const char *value = (const char *)m_value->data.scalar.value;
                        if (strcmp(key, "buffer_size_bytes") == 0) {
                            new_cfg->memory.buffer_size_bytes = (size_t)strtoull(value, NULL, 10);
                        }
                    }
                }
            }
        }
    }

    yaml_document_delete(&document);
    yaml_parser_delete(&parser);
    fclose(file);

    // Swap in the new config
    if (current_config) {
        free(current_config);
    }
    current_config = new_cfg;
    LOG_INFO_MSG("Configuration reloaded from %s", current_config_path);
    return true;
}
