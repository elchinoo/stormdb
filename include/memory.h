#ifndef STORMDB_MEMORY_H
#define STORMDB_MEMORY_H

#include <stddef.h>
#include <stdbool.h>
#include <stdint.h>

// Opaque handle used to reference allocated blocks managed by the memory manager.
// Callers must use the read/write APIs to access contents; raw pointers are not
// supported to enable swapping to disk when memory pressure requires it.
typedef uint64_t memory_handle_t;

// Initialize memory manager with given buffer size (bytes). Returns false on failure.
bool memory_init_with_limit(size_t buffer_size_bytes);
void memory_cleanup(void);

// Allocate/Free a handle of given size. Returns 0 on failure.
memory_handle_t memory_alloc_handle(size_t size);
void memory_free_handle(memory_handle_t h);

// Read/write contents of a handle at an offset. Returns number of bytes read/written
// or -1 on error.
ssize_t memory_read_handle(memory_handle_t h, size_t offset, void *dst, size_t n);
ssize_t memory_write_handle(memory_handle_t h, size_t offset, const void *src, size_t n);

// Helpers
size_t memory_bytes_in_use(void);
size_t memory_limit_bytes(void);

#endif // STORMDB_MEMORY_H
