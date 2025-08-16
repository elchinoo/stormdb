#include "database.h"
#include "logging.h"
#include <dlfcn.h>
#include <libpq-fe.h>

static PGconn *pg_connection = NULL;
static char *current_conn_info = NULL;

bool database_init(const database_config_t *config) {
    if (!config) {
        LOG_ERROR_MSG("Database configuration is NULL");
        return false;
    }
    
    // Build connection string
    size_t conn_len = strlen(config->host) + strlen(config->database) + 
                      strlen(config->user) + strlen(config->password) + 128;
    char *conn_info = malloc(conn_len);
    if (!conn_info) {
        LOG_ERROR_MSG("Failed to allocate memory for connection string");
        return false;
    }
    
    snprintf(conn_info, conn_len,
             "host=%s port=%d dbname=%s user=%s password=%s connect_timeout=%d",
             config->host, config->port, config->database,
             config->user, config->password, config->connect_timeout);
    
    // Connect to database
    pg_connection = PQconnectdb(conn_info);
    if (PQstatus(pg_connection) != CONNECTION_OK) {
        LOG_ERROR_MSG("Connection to database failed: %s", PQerrorMessage(pg_connection));
        PQfinish(pg_connection);
        pg_connection = NULL;
        free(conn_info);
        return false;
    }
    
    // Store connection info for reconnection
    current_conn_info = conn_info;
    
    LOG_INFO_MSG("Connected to PostgreSQL database %s@%s:%d", 
                 config->database, config->host, config->port);
    
    return true;
}

void database_cleanup(void) {
    if (pg_connection) {
        PQfinish(pg_connection);
        pg_connection = NULL;
        LOG_INFO_MSG("Disconnected from database");
    }
    
    if (current_conn_info) {
        free(current_conn_info);
        current_conn_info = NULL;
    }
}

bool database_is_connected(void) {
    if (!pg_connection) {
        return false;
    }
    
    return PQstatus(pg_connection) == CONNECTION_OK;
}

bool database_reconnect(void) {
    if (!current_conn_info) {
        LOG_ERROR_MSG("No connection info available for reconnection");
        return false;
    }
    
    // Close existing connection
    if (pg_connection) {
        PQfinish(pg_connection);
        pg_connection = NULL;
    }
    
    // Reconnect
    pg_connection = PQconnectdb(current_conn_info);
    if (PQstatus(pg_connection) != CONNECTION_OK) {
        LOG_ERROR_MSG("Reconnection to database failed: %s", PQerrorMessage(pg_connection));
        PQfinish(pg_connection);
        pg_connection = NULL;
        return false;
    }
    
    LOG_INFO_MSG("Successfully reconnected to database");
    return true;
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
    if (!ok) LOG_ERROR_MSG("Insert metric failed: %s", PQerrorMessage(pg_connection));
    PQclear(r);
    return ok;
}
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
    
    // Create our result wrapper
    database_result_t *result = malloc(sizeof(database_result_t));
    if (!result) {
        LOG_ERROR_MSG("Failed to allocate memory for database result");
        PQclear(pg_result);
        return NULL;
    }
    
    result->pg_result = pg_result;
    result->row_count = PQntuples(pg_result);
    result->col_count = PQnfields(pg_result);
    result->current_row = 0;
    
    LOG_DEBUG_MSG("Query executed successfully, %d rows returned", result->row_count);
    return result;
}

void database_free_result(database_result_t *result) {
    if (result) {
        if (result->pg_result) {
            PQclear(result->pg_result);
        }
        free(result);
    }
}

const char* database_get_value(database_result_t *result, int row, int col) {
    if (!result || !result->pg_result) {
        return NULL;
    }
    
    if (row < 0 || row >= result->row_count || col < 0 || col >= result->col_count) {
        LOG_WARN_MSG("Database result index out of bounds: row %d, col %d", row, col);
        return NULL;
    }
    
    return PQgetvalue(result->pg_result, row, col);
}

const char* database_get_column_name(database_result_t *result, int col) {
    if (!result || !result->pg_result) {
        return NULL;
    }
    
    if (col < 0 || col >= result->col_count) {
        return NULL;
    }
    
    return PQfname(result->pg_result, col);
}
