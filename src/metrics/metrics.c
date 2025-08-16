#include "metrics.h"
#include "logging.h"
#include "database.h"
#include "platform.h"
#include <stdlib.h>
#include <string.h>

typedef struct {
    metric_t *buf;
    size_t cap;
    size_t head;
    size_t tail;
    platform_mutex_t mu;
    bool running;
    platform_thread_t th;
    int collection_interval_ms;
    int health_push_counter;
} metrics_state_t;

static metrics_state_t g;

static void push_health_metrics(void) {
    metric_t m;
    unsigned long failures = database_get_reconnect_failures();
    unsigned long successes = database_get_reconnect_successes();
    int connected = database_is_connected() ? 1 : 0;
    uint64_t ts = platform_get_timestamp_us();

    // failures
    memset(&m, 0, sizeof(m));
    m.ts_us = ts;
    snprintf(m.name, sizeof(m.name), "stormdb.db.reconnect_failures");
    m.value = (double)failures;
    metrics_push(&m);
    // successes
    memset(&m, 0, sizeof(m));
    m.ts_us = ts;
    snprintf(m.name, sizeof(m.name), "stormdb.db.reconnect_successes");
    m.value = (double)successes;
    metrics_push(&m);
    // connected
    memset(&m, 0, sizeof(m));
    m.ts_us = ts;
    snprintf(m.name, sizeof(m.name), "stormdb.db.connected");
    m.value = (double)connected;
    metrics_push(&m);
}

static void* metrics_thread(void* arg) {
    (void)arg;
    LOG_INFO_MSG("Metrics thread started (cap=%zu)", g.cap);
    while (g.running) {
        // Pop one if available
        platform_mutex_lock(&g.mu);
        bool has = (g.head != g.tail);
        metric_t m;
        if (has) {
            m = g.buf[g.tail];
            g.tail = (g.tail + 1) % g.cap;
        }
        platform_mutex_unlock(&g.mu);

        if (has) {
            // Persist to DB; on failure, log and continue
            if (!database_insert_metric(m.ts_us, m.name, m.value)) {
                LOG_WARN_MSG("Failed to persist metric %s", m.name);
            }
        } else {
            platform_sleep_ms(50);
        }

        // Periodically push health metrics
        g.health_push_counter += 1;
        if (g.health_push_counter >= (1000 / (g.collection_interval_ms > 0 ? g.collection_interval_ms : 1000))) {
            push_health_metrics();
            g.health_push_counter = 0;
        }
    }
    LOG_INFO_MSG("Metrics thread exiting");
    return NULL;
}

bool metrics_init(size_t capacity) {
    memset(&g, 0, sizeof(g));
    if (capacity == 0) capacity = MAX_METRICS_QUEUE_SIZE;
    g.buf = (metric_t*)calloc(capacity, sizeof(metric_t));
    if (!g.buf) return false;
    g.cap = capacity;
    if (platform_mutex_init(&g.mu) != 0) return false;
    // default collection interval until configured externally
    g.collection_interval_ms = 1000;
    g.health_push_counter = 0;
    return true;
}

void metrics_cleanup(void) {
    if (g.buf) {
        free(g.buf);
        g.buf = NULL;
    }
    g.cap = g.head = g.tail = 0;
}

bool metrics_start(void) {
    if (g.running) return true;
    g.running = true;
    return platform_thread_create(&g.th, metrics_thread, NULL) == 0;
}

void metrics_stop(void) {
    if (!g.running) return;
    g.running = false;
    void* rv = NULL;
    platform_thread_join(g.th, &rv);
}

bool metrics_push(const metric_t* m) {
    if (!m || !g.buf || g.cap == 0) return false;
    platform_mutex_lock(&g.mu);
    size_t next = (g.head + 1) % g.cap;
    if (next == g.tail) {
        // overwrite oldest by advancing tail
        g.tail = (g.tail + 1) % g.cap;
    }
    g.buf[g.head] = *m;
    g.head = next;
    platform_mutex_unlock(&g.mu);
    return true;
}
