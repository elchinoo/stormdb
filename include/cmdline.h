#ifndef CMDLINE_H
#define CMDLINE_H

#include "stormdb.h"
#include "logging.h"

// Command line arguments structure
typedef struct {
    char *config_file;
    char *database_host;
    int database_port;
    char *database_name;
    char *database_user;
    char *database_password;
    int api_port;
    log_level_t log_level;
    bool verbose;
    bool quiet;
    bool show_help;
    bool show_version;
} cmdline_args_t;

// Command line parsing functions
bool cmdline_parse(int argc, char *argv[], cmdline_args_t *args);
void cmdline_init_args(cmdline_args_t *args);
void cmdline_free_args(cmdline_args_t *args);
void cmdline_print_help(const char *program_name);
void cmdline_print_version(void);

#endif // CMDLINE_H
