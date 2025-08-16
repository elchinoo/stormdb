#include "memory.h"
#include "platform.h"
#include <stdatomic.h>
#include <stdlib.h>

static _Atomic size_t g_in_use = 0;

bool memory_init(void) {
    g_in_use = 0;
    return true;
}

void memory_cleanup(void) {
}

void* memory_alloc(size_t size) {
    void* p = malloc(size);
    if (p) {
        g_in_use += size;
    }
    return p;
}

void memory_free(void* p) {
    if (p) free(p);
}

size_t memory_bytes_in_use(void) {
    return g_in_use;
}
