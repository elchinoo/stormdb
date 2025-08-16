#ifndef STORMDB_MEMORY_H
#define STORMDB_MEMORY_H

#include <stddef.h>
#include <stdbool.h>
#include <stdint.h>

bool memory_init(void);
void memory_cleanup(void);
void* memory_alloc(size_t size);
void memory_free(void* p);
size_t memory_bytes_in_use(void);

#endif // STORMDB_MEMORY_H
