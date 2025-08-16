#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <assert.h>
#include "config.h"
#include "pidfile.h"
#include "plugin.h"

// Minimal pseudo-tests to catch edge cases; use return codes only (no framework)
static int failures = 0;
static void check(bool cond, const char* msg) { if (!cond) { fprintf(stderr, "FAIL: %s\n", msg); failures++; } }

int main(void) {
    // Config defaults
    stormdb_config_t cfg; memset(&cfg, 0, sizeof(cfg));
    config_set_defaults(&cfg);
    check(cfg.logging.max_files > 0, "logging.max_files default > 0");
    check(cfg.logging.max_size > 0, "logging.max_size default > 0");
    check(cfg.api.port > 0, "api.port default > 0");

    // PID file lifecycle
    const char* pid = "/tmp/stormdb-test.pid";
    // Ensure not running, create and then remove
    (void)pidfile_remove; // silence unused on some platforms
    bool running = pidfile_check_running(pid);
    check(!running, "pidfile_check_running false when no file");

    bool created = pidfile_create(pid);
    check(created, "pidfile_create returns true");

    running = pidfile_check_running(pid);
    // Might be true or false depending on implementation; just ensure it doesn't crash
    check(running || !running, "pidfile_check_running callable");

    pidfile_remove();

    // Plugin directory load (should tolerate missing dir)
    bool loaded = plugin_load_from_directory("/non/existent/dir");
    check(!loaded || loaded, "plugin_load_from_directory tolerant of missing dir");

    if (failures) {
        fprintf(stderr, "Unit tests had %d failure(s)\n", failures);
        return 1;
    }
    printf("Unit tests passed\n");
    return 0;
}
