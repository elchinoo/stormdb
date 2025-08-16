#ifndef STORMDB_METRICS_H
#define STORMDB_METRICS_H

#include "stormdb.h"
#include "config.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    uint64_t ts_us;
    char name[64];
    double value;
    // reserved for tags/labels in future
} metric_t;

bool metrics_init(size_t capacity);
void metrics_cleanup(void);
bool metrics_start(void);
void metrics_stop(void);

// Non-blocking push; overwrites oldest on full and returns true. Returns false on error.
bool metrics_push(const metric_t* m);

#endif // STORMDB_METRICS_H
