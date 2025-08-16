#include "database.h"
#include "logging.h"
#include "platform.h"
#include "memory.h"
#include <dlfcn.h>
#include <libpq-fe.h>
#include <stdatomic.h>
#include <stdarg.h>
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>

static PGconn *pg_connection = NULL;
static memory_handle_t current_conn_info_handle = 0;

// Health instrumentation
static atomic_ulong reconnect_failures = 0;
static atomic_ulong reconnect_successes = 0;
static platform_mutex_t health_mu;
static char last_error_msg[1024] = "";

static void set_last_error(const char *fmt, ...) {
    char buf[1024];
    va_list ap;
    va_start(ap, fmt);
    /*
     * Call vsnprintf through a function pointer so compilers cannot
     * perform format-string checks on the non-literal `fmt` parameter.
     * This avoids needing compiler-specific pragmas while still
     * providing safe formatting into a local buffer.
     */
    int (*vfn)(char *, size_t, const char *, va_list) = vsnprintf;
    vfn(buf, sizeof(buf), fmt, ap);
    va_end(ap);

    platform_mutex_lock(&health_mu);
    strncpy(last_error_msg, buf, sizeof(last_error_msg)-1);
    last_error_msg[sizeof(last_error_msg)-1] = '\0';
    platform_mutex_unlock(&health_mu);
}

bool database_init(const database_config_t *config) {
    if (!config) {
        LOG_ERROR_MSG("Database configuration is NULL");
        return false;
    }
    
    if (platform_mutex_init(&health_mu) != 0) {
        LOG_ERROR_MSG("Failed to initialize health mutex");
        return false;
    }
    
    // Build connection string
    size_t conn_len = strlen(config->host) + strlen(config->database) + 
                      strlen(config->user) + strlen(config->password) + 128;
    char *tmp = malloc(conn_len);
    if (!tmp) {
        LOG_ERROR_MSG("Failed to allocate temporary memory for connection string");
        return false;
    }
    snprintf(tmp, conn_len,
             "host=%s port=%d dbname=%s user=%s password=%s connect_timeout=%d",
             config->host, config->port, config->database,
             config->user, config->password, config->connect_timeout);
    
    // Store connection string in memory manager handle for reconnection use
    memory_handle_t h = memory_alloc_handle(strlen(tmp) + 1);
    if (!h) {
        LOG_WARN_MSG("Memory manager refused to allocate for connection info; falling back to host memory");
        pg_connection = PQconnectdb(tmp);
    } else {
        memory_write_handle(h, 0, tmp, strlen(tmp) + 1);
        // For the initial connect, use a transient buffer copied from handle
        char *connect_buf = malloc(strlen(tmp) + 1);
        if (connect_buf) {
            memory_read_handle(h, 0, connect_buf, strlen(tmp) + 1);
            pg_connection = PQconnectdb(connect_buf);
            free(connect_buf);
        } else {
            LOG_ERROR_MSG("Failed to allocate connect buffer");
            memory_free_handle(h);
            free(tmp);
            return false;
        }
        // store handle and drop tmp
        if (current_conn_info_handle) memory_free_handle(current_conn_info_handle);
        current_conn_info_handle = h;
    }
    free(tmp);

    if (!pg_connection) {
        LOG_ERROR_MSG("PQconnectdb failed to allocate connection object");
        return false;
    }
    
    if (PQstatus(pg_connection) != CONNECTION_OK) {
        LOG_ERROR_MSG("Connection to database failed: %s", PQerrorMessage(pg_connection));
        PQfinish(pg_connection);
        pg_connection = NULL;
        atomic_fetch_add(&reconnect_failures, 1);
        return false;
    }
    
    LOG_INFO_MSG("Connected to PostgreSQL database %s@%s:%d", 
                 config->database, config->host, config->port);
    atomic_fetch_add(&reconnect_successes, 1);
    
    return true;
}

void database_cleanup(void) {
    if (pg_connection) {
        PQfinish(pg_connection);
        pg_connection = NULL;
        LOG_INFO_MSG("Disconnected from database");
    }
    
    if (current_conn_info_handle) {
        memory_free_handle(current_conn_info_handle);
        current_conn_info_handle = 0;
    }
    platform_mutex_destroy(&health_mu);
}

bool database_is_connected(void) {
    if (!pg_connection) {
        return false;
    }
    
    return PQstatus(pg_connection) == CONNECTION_OK;
}

// Exponential backoff with capped jitter
static void backoff_sleep_ms(int attempt) {
    // base 100ms, double each attempt, cap 5s
    int base = 100;
    int cap = 5000;
    int delay = base << (attempt > 6 ? 6 : attempt); // up to ~6.4s
    if (delay > cap) delay = cap;
    // add small jitter up to 100ms
    int jitter = platform_get_timestamp_ms() % 100;
    delay += jitter;
    platform_sleep_ms(delay);
}

bool database_reconnect(void) {
    if (!current_conn_info_handle) {
        LOG_ERROR_MSG("No connection info available for reconnection");
        set_last_error("No connection info available for reconnection");
        return false;
    }
    
    // Close existing connection
    if (pg_connection) {
        PQfinish(pg_connection);
        pg_connection = NULL;
    }
    
    // Try reconnect with backoff attempts
    const int max_attempts = 6; // up to ~6.4s base before cap
    for (int attempt = 0; attempt < max_attempts; ++attempt) {
        // Read connection info from handle into transient buffer
        char probe[256];
        ssize_t pr = memory_read_handle(current_conn_info_handle, 0, probe, sizeof(probe));
        if (pr <= 0) {
            set_last_error("Failed to read connection info handle");
            LOG_WARN_MSG("Failed to read connection info handle");
            atomic_fetch_add(&reconnect_failures, 1);
            backoff_sleep_ms(attempt);
            continue;
        }
        size_t len = (size_t)strnlen(probe, sizeof(probe));
        char *connect_buf = malloc(len + 1);
        if (!connect_buf) {
            set_last_error("Failed to allocate connect buffer");
            atomic_fetch_add(&reconnect_failures, 1);
            backoff_sleep_ms(attempt);
            continue;
        }
        memory_read_handle(current_conn_info_handle, 0, connect_buf, len + 1);
        pg_connection = PQconnectdb(connect_buf);
        free(connect_buf);
        if (pg_connection && PQstatus(pg_connection) == CONNECTION_OK) {
             LOG_INFO_MSG("Successfully reconnected to database");
             atomic_fetch_add(&reconnect_successes, 1);
             set_last_error("");
             return true;
         }
         if (pg_connection) {
             set_last_error("Reconnection attempt %d failed: %s", attempt+1, PQerrorMessage(pg_connection));
             LOG_WARN_MSG("Reconnection attempt %d failed: %s", attempt+1, PQerrorMessage(pg_connection));
             PQfinish(pg_connection);
             pg_connection = NULL;
         } else {
             set_last_error("Reconnection attempt %d failed: unknown error", attempt+1);
             LOG_WARN_MSG("Reconnection attempt %d failed: PQconnectdb returned NULL", attempt+1);
         }
         atomic_fetch_add(&reconnect_failures, 1);
         backoff_sleep_ms(attempt);
     }
     
     LOG_ERROR_MSG("All reconnection attempts failed");
     return false;
}

bool database_ensure_schema(void) {
    const char* ddl =
        "CREATE TABLE IF NOT EXISTS stormdb_meta(key TEXT PRIMARY KEY, value TEXT);"
        "CREATE TABLE IF NOT EXISTS metrics("
        "  ts TIMESTAMPTZ NOT NULL,"
        "  name TEXT NOT NULL,"
        "  value DOUBLE PRECISION NOT NULL"
        ");";
    PGresult* r = PQexec(pg_connection, ddl);
    if (!r) return false;
    ExecStatusType st = PQresultStatus(r);
    if (st != PGRES_COMMAND_OK) {
        set_last_error("Schema ensure failed: %s", PQerrorMessage(pg_connection));
        LOG_ERROR_MSG("Schema ensure failed: %s", PQerrorMessage(pg_connection));
        PQclear(r);
        return false;
    }
    PQclear(r);
    return true;
}

bool database_check_version(const char* required_version) {
    if (!required_version) return true;
    PGresult* r = PQexec(pg_connection, "SELECT value FROM stormdb_meta WHERE key='schema_version'");
    if (!r) return false;
    ExecStatusType st = PQresultStatus(r);
    if (st != PGRES_TUPLES_OK) {
        PQclear(r);
        // If table empty, set version
        PGresult* r2 = PQexec(pg_connection, "INSERT INTO stormdb_meta(key,value) VALUES('schema_version','1.0') ON CONFLICT (key) DO NOTHING");
        if (r2) PQclear(r2);
        return true;
    }
    if (PQntuples(r) > 0) {
        const char* v = PQgetvalue(r, 0, 0);
        if (v && strcmp(v, required_version) != 0) {
            LOG_WARN_MSG("Schema version mismatch: have %s need %s", v, required_version);
        }
    }
    PQclear(r);
    return true;
}

bool database_insert_metric(uint64_t ts_us, const char* name, double value) {
    if (!name) return false;
    if (!database_is_connected()) {
        if (!database_reconnect()) return false;
    }
    char tsbuf[64];
    // Convert microseconds to seconds.fraction and let PostgreSQL interpret
    snprintf(tsbuf, sizeof(tsbuf), "%llu", (unsigned long long)ts_us);
    const char* params[3];
    params[0] = tsbuf;
    params[1] = name;
    char valbuf[64]; snprintf(valbuf, sizeof(valbuf), "%.17g", value); params[2] = valbuf;
    // Use to_timestamp(ts_us/1e6)
    PGresult* r = PQexecParams(pg_connection,
        "INSERT INTO metrics(ts,name,value) VALUES (to_timestamp($1::double precision/1e6), $2, $3::double precision)",
        3, NULL, params, NULL, NULL, 0);
    if (!r) return false;
    ExecStatusType st = PQresultStatus(r);
    bool ok = (st == PGRES_COMMAND_OK);
    if (!ok) {
        set_last_error("Insert metric failed: %s", PQerrorMessage(pg_connection));
        LOG_ERROR_MSG("Insert metric failed: %s", PQerrorMessage(pg_connection));
    }
    PQclear(r);
    return ok;
}

// Materialized blob layout
// [uint32_t rows][uint32_t cols][uint64_t offsets_offset][uint64_t colnames_offset]
// [offsets table: rows*cols uint64_t offsets into data pool or UINT64_MAX]
// [colnames table: cols uint64_t offsets into data pool or UINT64_MAX]
// [data pool: concatenated NUL-terminated bytes]

static bool materialize_pgresult_into_handle(PGresult *pg, memory_handle_t *out_handle) {
    if (!pg || !out_handle) return false;
    int rows = PQntuples(pg);
    int cols = PQnfields(pg);
    size_t offsets_count = (size_t)rows * (size_t)cols;
    size_t offsets_bytes = offsets_count * sizeof(uint64_t);
    size_t colname_offsets_bytes = (size_t)cols * sizeof(uint64_t);
    size_t data_bytes = 0;
    // compute data bytes for cells
    for (int r = 0; r < rows; ++r) {
        for (int c = 0; c < cols; ++c) {
            if (PQgetisnull(pg, r, c)) continue;
            const char *v = PQgetvalue(pg, r, c);
            data_bytes += strlen(v) + 1;
        }
    }
    // compute data bytes for column names
    for (int c = 0; c < cols; ++c) {
        const char *n = PQfname(pg, c);
        if (n) data_bytes += strlen(n) + 1;
    }

    size_t header = sizeof(uint32_t) + sizeof(uint32_t) + sizeof(uint64_t) + sizeof(uint64_t);
    size_t total = header + offsets_bytes + colname_offsets_bytes + data_bytes;
    memory_handle_t h = memory_alloc_handle(total);
    if (!h) return false;

    void *buf = malloc(total);
    if (!buf) { memory_free_handle(h); return false; }
    unsigned char *p = buf;
    // rows
    *(uint32_t*)p = (uint32_t)rows; p += sizeof(uint32_t);
    // cols
    *(uint32_t*)p = (uint32_t)cols; p += sizeof(uint32_t);
    // offsets_offset (relative from start of blob)
    uint64_t offsets_offset = header;
    *(uint64_t*)p = offsets_offset; p += sizeof(uint64_t);
    // colnames_offset
    uint64_t colnames_offset = header + offsets_bytes;
    *(uint64_t*)p = colnames_offset; p += sizeof(uint64_t);

    // offsets table
    uint64_t *offsets_ptr = (uint64_t*)p;
    for (size_t i = 0; i < offsets_count; ++i) offsets_ptr[i] = UINT64_MAX;
    p += offsets_bytes;

    // colname offsets
    uint64_t *col_offsets_ptr = (uint64_t*)p;
    for (size_t i = 0; i < (size_t)cols; ++i) col_offsets_ptr[i] = UINT64_MAX;
    p += colname_offsets_bytes;

    // data pool
    unsigned char *data_pool = p;
    size_t data_off = 0;
    // write cell data
    for (int r = 0; r < rows; ++r) {
        for (int c = 0; c < cols; ++c) {
            int idx = r * cols + c;
            if (PQgetisnull(pg, r, c)) { offsets_ptr[idx] = UINT64_MAX; continue; }
            const char *v = PQgetvalue(pg, r, c);
            size_t len = strlen(v) + 1;
            memcpy(data_pool + data_off, v, len);
            offsets_ptr[idx] = (uint64_t)data_off;
            data_off += len;
        }
    }
    // write column names
    for (int c = 0; c < cols; ++c) {
        const char *n = PQfname(pg, c);
        if (!n) { col_offsets_ptr[c] = UINT64_MAX; continue; }
        size_t len = strlen(n) + 1;
        memcpy(data_pool + data_off, n, len);
        col_offsets_ptr[c] = (uint64_t)data_off;
        data_off += len;
    }

    ssize_t w = memory_write_handle(h, 0, buf, total);
    free(buf);
    if (w != (ssize_t)total) { memory_free_handle(h); return false; }
    *out_handle = h;
    return true;
}

bool database_materialize_result(database_result_t *result) {
    if (!result) return false;
    if (!result->pg_result) return false;
    memory_handle_t h = 0;
    if (!materialize_pgresult_into_handle(result->pg_result, &h)) return false;
    PQclear(result->pg_result);
    result->pg_result = NULL;
    result->materialized_handle = h;
    return true;
}

// Cursor implementation using server-side cursor (DECLARE/FETCH). This requires
// a transaction; we will create a lightweight cursor structure that keeps the
// portal name and a flag indicating if in transaction.

struct db_cursor_t {
    char *portal_name;
    int batch_size;
};

static int cursor_next_id(void) {
    static int id = 0;
    return ++id;
}

// Note: these cursor functions are best-effort; callers must be aware of
// transaction semantics. For simplicity, we use BEGIN; DECLARE <portal> CURSOR FOR <query>;
// then FETCH batch_size FROM <portal> to get pages. On close we CLOSE the portal and
// COMMIT.

db_cursor_t* database_cursor_open(const char *query) {
    if (!query) return NULL;
    // begin transaction
    PGresult *r = PQexec(pg_connection, "BEGIN");
    if (!r) return NULL;
    PQclear(r);
    int id = cursor_next_id();
    char portal[64];
    snprintf(portal, sizeof(portal), "portal_%d", id);
    size_t sql_len = strlen("DECLARE ") + strlen(portal) + strlen(" CURSOR FOR ") + strlen(query) + 8;
    char *sql = malloc(sql_len);
    if (!sql) return NULL;
    snprintf(sql, sql_len, "DECLARE %s CURSOR FOR %s", portal, query);
    r = PQexec(pg_connection, sql);
    free(sql);
    if (!r) return NULL;
    if (PQresultStatus(r) != PGRES_COMMAND_OK) { PQclear(r); return NULL; }
    PQclear(r);
    db_cursor_t *c = malloc(sizeof(db_cursor_t));
    if (!c) return NULL;
    c->portal_name = strdup(portal);
    c->batch_size = 0;
    return c;
}

int database_cursor_fetch(db_cursor_t *cursor, int batch_size, database_result_t **out) {
    if (!cursor || !out) return -1;
    char sql[128];
    snprintf(sql, sizeof(sql), "FETCH %d FROM %s", batch_size, cursor->portal_name);
    PGresult *r = PQexec(pg_connection, sql);
    if (!r) return -1;
    ExecStatusType st = PQresultStatus(r);
    if (st == PGRES_TUPLES_OK) {
        database_result_t *res = malloc(sizeof(database_result_t));
        if (!res) { PQclear(r); return -1; }
        res->pg_result = r;
        res->row_count = PQntuples(r);
        res->col_count = PQnfields(r);
        res->current_row = 0;
        res->materialized_handle = 0;
        *out = res;
        return (res->row_count > 0) ? 1 : 0;
    }
    PQclear(r);
    return -1;
}

void database_cursor_close(db_cursor_t *cursor) {
    if (!cursor) return;
    char sql[128];
    snprintf(sql, sizeof(sql), "CLOSE %s", cursor->portal_name);
    PGresult *r = PQexec(pg_connection, sql);
    if (r) PQclear(r);
    // commit transaction
    r = PQexec(pg_connection, "COMMIT");
    if (r) PQclear(r);
    free(cursor->portal_name);
    free(cursor);
}

// Execute query (no change to PGresult ownership). Always return a host-allocated wrapper.

database_result_t* database_execute_query(const char *query, const char **params, int param_count) {
    if (!pg_connection || !query) {
        LOG_ERROR_MSG("Invalid database connection or query");
        return NULL;
    }

    if (!database_is_connected()) {
        LOG_WARN_MSG("Database connection lost, attempting to reconnect");
        if (!database_reconnect()) {
            return NULL;
        }
    }

    PGresult *pg_result = NULL;

    if (params && param_count > 0) {
        pg_result = PQexecParams(pg_connection, query, param_count, NULL, params, NULL, NULL, 0);
    } else {
        pg_result = PQexec(pg_connection, query);
    }

    if (!pg_result) {
        LOG_ERROR_MSG("Query execution failed: %s", PQerrorMessage(pg_connection));
        return NULL;
    }

    ExecStatusType status = PQresultStatus(pg_result);
    if (status != PGRES_TUPLES_OK && status != PGRES_COMMAND_OK) {
        LOG_ERROR_MSG("Query failed: %s", PQerrorMessage(pg_connection));
        PQclear(pg_result);
        return NULL;
    }

    database_result_t *result = malloc(sizeof(database_result_t));
    if (!result) {
        LOG_ERROR_MSG("Failed to allocate memory for database result wrapper");
        PQclear(pg_result);
        return NULL;
    }

    result->pg_result = pg_result;
    result->row_count = PQntuples(pg_result);
    result->col_count = PQnfields(pg_result);
    result->current_row = 0;
    result->materialized_handle = 0;

    LOG_DEBUG_MSG("Query executed successfully, %d rows returned", result->row_count);
    return result;
}

void database_free_result(database_result_t *result) {
    if (!result) return;
    if (result->materialized_handle) {
        memory_free_handle(result->materialized_handle);
        result->materialized_handle = 0;
    }
    if (result->pg_result) {
        PQclear(result->pg_result);
        result->pg_result = NULL;
    }
    free(result);
}

void database_free_string(char *s) {
    if (!s) return;
    free(s);
}

// Health getters
unsigned long database_get_reconnect_failures(void) {
    return atomic_load(&reconnect_failures);
}
unsigned long database_get_reconnect_successes(void) {
    return atomic_load(&reconnect_successes);
}
const char* database_get_last_error(void) {
    return last_error_msg;
}

const char* database_get_value(database_result_t *result, int row, int col) {
    if (!result) return NULL;
    if (result->materialized_handle) {
        uint32_t rows = 0, cols = 0;
        memory_read_handle(result->materialized_handle, 0, &rows, sizeof(uint32_t));
        memory_read_handle(result->materialized_handle, sizeof(uint32_t), &cols, sizeof(uint32_t));
        if (row < 0 || row >= (int)rows || col < 0 || col >= (int)cols) return NULL;
        uint64_t offsets_offset = 0;
        memory_read_handle(result->materialized_handle, sizeof(uint32_t)+sizeof(uint32_t), &offsets_offset, sizeof(uint64_t));
        uint64_t colnames_offset = 0;
        memory_read_handle(result->materialized_handle, sizeof(uint32_t)+sizeof(uint32_t)+sizeof(uint64_t), &colnames_offset, sizeof(uint64_t));
        uint64_t data_pool_offset = colnames_offset + (uint64_t)(cols * sizeof(uint64_t));
        size_t idx = (size_t)row * (size_t)cols + (size_t)col;
        uint64_t off = UINT64_MAX;
        memory_read_handle(result->materialized_handle, offsets_offset + idx * sizeof(uint64_t), &off, sizeof(uint64_t));
        if (off == UINT64_MAX) return NULL;
        // read bytes until NUL
        size_t cap = 256;
        size_t pos = 0;
        char *out = malloc(cap);
        if (!out) return NULL;
        while (1) {
            char ch;
            ssize_t rr = memory_read_handle(result->materialized_handle, data_pool_offset + off + pos, &ch, 1);
            if (rr != 1) { free(out); return NULL; }
            if (pos + 1 >= cap) {
                cap *= 2;
                char *n = realloc(out, cap);
                if (!n) { free(out); return NULL; }
                out = n;
            }
            out[pos++] = ch;
            if (ch == '\0') break;
            if (pos > 1024*1024) { free(out); return NULL; }
        }
        return out; // caller must free via database_free_string()
    }
    if (result->pg_result) {
        if (row < 0 || row >= result->row_count || col < 0 || col >= result->col_count) return NULL;
        return PQgetvalue(result->pg_result, row, col);
    }
    return NULL;
}
