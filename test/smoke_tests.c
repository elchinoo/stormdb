#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>

// Forward-declare functions we will exercise via system calls

static int run_cmd(const char* cmd) {
    int rc = system(cmd);
    if (rc == -1) return -1;
#ifdef _WIN32
    return rc;
#else
    if (WIFEXITED(rc)) return WEXITSTATUS(rc);
    return rc;
#endif
}

static void assert_eq_int(const char* name, int a, int b) {
    if (a != b) {
        fprintf(stderr, "Assertion failed: %s expected %d got %d\n", name, b, a);
        exit(1);
    }
}

int main(void) {
    // --version should succeed
    int rc = run_cmd("./bin/stormdb-debug --version > /dev/null");
    assert_eq_int("version", rc, 0);

    // --help should succeed
    rc = run_cmd("./bin/stormdb-debug --help > /dev/null");
    assert_eq_int("help", rc, 0);

    // Load default sample config; allow missing DB since we are not connecting here
    // Just ensure the binary starts and immediately exits on --version

    printf("Smoke tests passed\n");
    return 0;
}
