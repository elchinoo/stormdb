#ifndef STORMDB_DATABASE_H
#define STORMDB_DATABASE_H

#include "stormdb.h"
#include "config.h"
#include <stdbool.h>
#include <libpq-fe.h>

// Database result wrapper
typedef struct {
    PGresult *pg_result;
    int row_count;
    int col_count;
    int current_row;
} database_result_t;

// Database functions
bool database_init(const database_config_t *config);
void database_cleanup(void);
bool database_is_connected(void);
bool database_reconnect(void);
// Schema/version management
bool database_ensure_schema(void);
bool database_check_version(const char* required_version);

// Query execution
database_result_t* database_execute_query(const char *query, const char **params, int param_count);
void database_free_result(database_result_t *result);

// Result handling
const char* database_get_value(database_result_t *result, int row, int col);
const char* database_get_column_name(database_result_t *result, int col);

// Metrics persistence (simple example table metrics(ts TIMESTAMPTZ, name TEXT, value DOUBLE PRECISION))
bool database_insert_metric(uint64_t ts_us, const char* name, double value);

#endif // STORMDB_DATABASE_H
