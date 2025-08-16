#include "memory.h"
#include "logging.h"
#include "platform.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <errno.h>
#include <inttypes.h>

typedef struct mem_block_s {
    memory_handle_t handle;
    size_t size;
    void *data; // NULL if swapped out
    off_t swap_offset; // offset in swap file
    bool dirty;
    struct mem_block_s *prev, *next; // LRU list
} mem_block_t;

static mem_block_t **g_table = NULL;
static size_t g_table_cap = 0;
static memory_handle_t g_next_handle = 1;

static void *g_buffer = NULL;
static size_t g_buffer_size = 0; // configured limit
static size_t g_buffer_used = 0;

static platform_mutex_t g_mu;
static int g_swap_fd = -1;
static off_t g_swap_next_offset = 0;

// LRU head (most recent) and tail (least recent)
static mem_block_t *g_lru_head = NULL;
static mem_block_t *g_lru_tail = NULL;

static bool ensure_table(void) {
    if (!g_table) {
        g_table_cap = 1024;
        g_table = calloc(g_table_cap, sizeof(mem_block_t*));
        if (!g_table) return false;
    }
    return true;
}

static void lru_promote(mem_block_t *b) {
    if (!b || g_lru_head == b) return;
    // unlink
    if (b->prev) b->prev->next = b->next;
    if (b->next) b->next->prev = b->prev;
    if (g_lru_tail == b) g_lru_tail = b->prev;
    // insert at head
    b->prev = NULL;
    b->next = g_lru_head;
    if (g_lru_head) g_lru_head->prev = b;
    g_lru_head = b;
    if (!g_lru_tail) g_lru_tail = b;
}

static void lru_insert(mem_block_t *b) {
    b->prev = NULL; b->next = g_lru_head;
    if (g_lru_head) g_lru_head->prev = b;
    g_lru_head = b;
    if (!g_lru_tail) g_lru_tail = b;
}

static void lru_remove(mem_block_t *b) {
    if (!b) return;
    if (b->prev) b->prev->next = b->next;
    if (b->next) b->next->prev = b->prev;
    if (g_lru_head == b) g_lru_head = b->next;
    if (g_lru_tail == b) g_lru_tail = b->prev;
    b->prev = b->next = NULL;
}

static bool open_swap_file(void) {
    if (g_swap_fd >= 0) return true;
    char tmp[PATH_MAX];
    snprintf(tmp, sizeof(tmp), "%s/stormdb_swap_%d.bin", platform_get_current_directory(), (int)getpid());
    g_swap_fd = open(tmp, O_RDWR | O_CREAT, 0600);
    if (g_swap_fd < 0) {
        LOG_ERROR_MSG("Failed to open swap file %s: %s", tmp, strerror(errno));
        return false;
    }
    // unlink so file is removed when process exits
    unlink(tmp);
    return true;
}

static bool swap_out(mem_block_t *b) {
    if (!b || !b->data) return true;
    if (!open_swap_file()) return false;
    // write at next offset
    off_t off = g_swap_next_offset;
    ssize_t w = pwrite(g_swap_fd, b->data, b->size, off);
    if (w < 0 || (size_t)w != b->size) {
        LOG_ERROR_MSG("Swap out failed: %s", strerror(errno));
        return false;
    }
    b->swap_offset = off;
    g_swap_next_offset += b->size;
    free(b->data);
    b->data = NULL;
    if (b->dirty) b->dirty = false;
    g_buffer_used -= b->size;
    return true;
}

static bool swap_in(mem_block_t *b) {
    if (!b || b->data) return true;
    if (!open_swap_file()) return false;
    if (g_buffer_size - g_buffer_used < b->size) {
        // evict until we have space
        while (g_lru_tail && (g_buffer_size - g_buffer_used < b->size)) {
            if (!swap_out(g_lru_tail)) return false;
            lru_remove(g_lru_tail);
        }
    }
    b->data = malloc(b->size);
    if (!b->data) return false;
    ssize_t r = pread(g_swap_fd, b->data, b->size, b->swap_offset);
    if (r < 0 || (size_t)r != b->size) {
        LOG_ERROR_MSG("Swap in failed: %s", strerror(errno));
        free(b->data);
        b->data = NULL;
        return false;
    }
    g_buffer_used += b->size;
    lru_insert(b);
    return true;
}

bool memory_init_with_limit(size_t buffer_size_bytes) {
    if (buffer_size_bytes == 0) return false;
    g_buffer_size = buffer_size_bytes;
    if (!ensure_table()) return false;
    if (platform_mutex_init(&g_mu) != 0) return false;
    open_swap_file(); // best-effort
    return true;
}

void memory_cleanup(void) {
    platform_mutex_lock(&g_mu);
    // free all blocks
    if (g_table) {
        for (size_t i = 0; i < g_table_cap; ++i) {
            mem_block_t *b = g_table[i];
            while (b) {
                mem_block_t *n = b->next;
                if (b->data) free(b->data);
                free(b);
                b = n;
            }
        }
        free(g_table);
        g_table = NULL;
    }
    if (g_swap_fd >= 0) {
        close(g_swap_fd);
        g_swap_fd = -1;
    }
    platform_mutex_unlock(&g_mu);
    platform_mutex_destroy(&g_mu);
}

memory_handle_t memory_alloc_handle(size_t size) {
    if (size == 0) return 0;
    platform_mutex_lock(&g_mu);
    if (!ensure_table()) { platform_mutex_unlock(&g_mu); return 0; }
    // Evict until we have space
    while (g_buffer_size - g_buffer_used < size) {
        if (!g_lru_tail) break;
        if (!swap_out(g_lru_tail)) break;
        lru_remove(g_lru_tail);
    }
    mem_block_t *b = calloc(1, sizeof(mem_block_t));
    if (!b) { platform_mutex_unlock(&g_mu); return 0; }
    b->handle = g_next_handle++;
    b->size = size;
    b->data = malloc(size);
    if (!b->data) { free(b); platform_mutex_unlock(&g_mu); return 0; }
    b->swap_offset = -1;
    b->dirty = false;
    g_buffer_used += size;
    lru_insert(b);
    // insert into hash table (simple mod)
    size_t idx = b->handle % g_table_cap;
    b->next = (mem_block_t*)g_table[idx];
    g_table[idx] = b;
    platform_mutex_unlock(&g_mu);
    return b->handle;
}

void memory_free_handle(memory_handle_t h) {
    if (h == 0) return;
    platform_mutex_lock(&g_mu);
    if (!g_table) { platform_mutex_unlock(&g_mu); return; }
    size_t idx = h % g_table_cap;
    mem_block_t *prev = NULL;
    mem_block_t *b = g_table[idx];
    while (b) {
        if (b->handle == h) break;
        prev = b; b = b->next;
    }
    if (!b) { platform_mutex_unlock(&g_mu); return; }
    // remove from table
    if (prev) prev->next = b->next; else g_table[idx] = b->next;
    lru_remove(b);
    if (b->data) { g_buffer_used -= b->size; free(b->data); }
    free(b);
    platform_mutex_unlock(&g_mu);
}

ssize_t memory_read_handle(memory_handle_t h, size_t offset, void *dst, size_t n) {
    if (h == 0 || !dst) return -1;
    platform_mutex_lock(&g_mu);
    if (!g_table) { platform_mutex_unlock(&g_mu); return -1; }
    size_t idx = h % g_table_cap;
    mem_block_t *b = g_table[idx];
    while (b) { if (b->handle == h) break; b = b->next; }
    if (!b) { platform_mutex_unlock(&g_mu); return -1; }
    if (offset >= b->size) { platform_mutex_unlock(&g_mu); return 0; }
    size_t toread = n;
    if (offset + toread > b->size) toread = b->size - offset;
    if (!b->data) {
        if (!swap_in(b)) { platform_mutex_unlock(&g_mu); return -1; }
    }
    memcpy(dst, (char*)b->data + offset, toread);
    lru_promote(b);
    platform_mutex_unlock(&g_mu);
    return (ssize_t)toread;
}

ssize_t memory_write_handle(memory_handle_t h, size_t offset, const void *src, size_t n) {
    if (h == 0 || !src) return -1;
    platform_mutex_lock(&g_mu);
    if (!g_table) { platform_mutex_unlock(&g_mu); return -1; }
    size_t idx = h % g_table_cap;
    mem_block_t *b = g_table[idx];
    while (b) { if (b->handle == h) break; b = b->next; }
    if (!b) { platform_mutex_unlock(&g_mu); return -1; }
    if (offset >= b->size) { platform_mutex_unlock(&g_mu); return 0; }
    size_t towrite = n;
    if (offset + towrite > b->size) towrite = b->size - offset;
    if (!b->data) {
        if (!swap_in(b)) { platform_mutex_unlock(&g_mu); return -1; }
    }
    memcpy((char*)b->data + offset, src, towrite);
    b->dirty = true;
    lru_promote(b);
    platform_mutex_unlock(&g_mu);
    return (ssize_t)towrite;
}

size_t memory_bytes_in_use(void) {
    return g_buffer_used;
}

size_t memory_limit_bytes(void) {
    return g_buffer_size;
}
