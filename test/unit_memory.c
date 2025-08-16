#include <stdio.h>
#include <string.h>
#include <assert.h>
#include "memory.h"

int main(void) {
    if (!memory_init_with_limit(1024)) {
        fprintf(stderr, "memory_init_with_limit failed\n");
        return 2;
    }

    // Allocate a handle of 800 bytes
    memory_handle_t h1 = memory_alloc_handle(800);
    assert(h1 != 0);
    char pattern[800];
    for (int i = 0; i < 800; ++i) pattern[i] = (char)(i & 0xFF);
    ssize_t w = memory_write_handle(h1, 0, pattern, sizeof(pattern));
    assert(w == (ssize_t)sizeof(pattern));

    // Allocate another handle to force eviction
    memory_handle_t h2 = memory_alloc_handle(800);
    assert(h2 != 0 && h2 != h1);
    char pattern2[800];
    memset(pattern2, 0xAA, sizeof(pattern2));
    ssize_t w2 = memory_write_handle(h2, 0, pattern2, sizeof(pattern2));
    assert(w2 == (ssize_t)sizeof(pattern2));

    // Read back h1 (may have been swapped out and swapped back in)
    char buf[800];
    ssize_t r = memory_read_handle(h1, 0, buf, sizeof(buf));
    assert(r == (ssize_t)sizeof(buf));
    assert(memcmp(buf, pattern, sizeof(buf)) == 0);

    // Clean up
    memory_free_handle(h1);
    memory_free_handle(h2);
    memory_cleanup();

    printf("unit_memory PASS\n");
    return 0;
}
