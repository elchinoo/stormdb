#include <stdio.h>
#include <stdlib.h>
#include <libpq-fe.h>
#include <string.h>
#include <unistd.h>

int main(void) {
    const char *conninfo = "host=127.0.0.1 port=54320 dbname=stormdb_test user=stormdb password=stormdb123 connect_timeout=10";
    PGconn *conn = PQconnectdb(conninfo);
    if (PQstatus(conn) != CONNECTION_OK) {
        fprintf(stderr, "Connection failed: %s\n", PQerrorMessage(conn));
        PQfinish(conn);
        return 2;
    }

    const char *ddl =
        "CREATE TABLE IF NOT EXISTS stormdb_meta(key TEXT PRIMARY KEY, value TEXT);"
        "CREATE TABLE IF NOT EXISTS metrics("
        "  ts TIMESTAMPTZ NOT NULL," 
        "  name TEXT NOT NULL," 
        "  value DOUBLE PRECISION NOT NULL" 
        ");";
    PGresult *r = PQexec(conn, ddl);
    if (!r || PQresultStatus(r) != PGRES_COMMAND_OK) {
        fprintf(stderr, "Schema DDL failed: %s\n", PQerrorMessage(conn));
        if (r) PQclear(r);
        PQfinish(conn);
        return 3;
    }
    PQclear(r);

    /* Insert a test metric */
    r = PQexecParams(conn,
        "INSERT INTO metrics(ts,name,value) VALUES (now(), $1, $2::double precision)",
        2, NULL, (const char*[]){"integration.test.metric", "42.0"}, NULL, NULL, 0);
    if (!r || PQresultStatus(r) != PGRES_COMMAND_OK) {
        fprintf(stderr, "Insert failed: %s\n", PQerrorMessage(conn));
        if (r) PQclear(r);
        PQfinish(conn);
        return 4;
    }
    PQclear(r);

    /* Query count */
    r = PQexecParams(conn, "SELECT COUNT(*) FROM metrics WHERE name=$1", 1, NULL, (const char*[]){"integration.test.metric"}, NULL, NULL, 0);
    if (!r || PQresultStatus(r) != PGRES_TUPLES_OK) {
        fprintf(stderr, "Query failed: %s\n", PQerrorMessage(conn));
        if (r) PQclear(r);
        PQfinish(conn);
        return 5;
    }

    char *count_str = PQgetvalue(r, 0, 0);
    long count = strtol(count_str, NULL, 10);
    PQclear(r);
    PQfinish(conn);

    if (count > 0) {
        printf("INTEGRATION TEST PASS: found %ld rows\n", count);
        return 0;
    } else {
        fprintf(stderr, "INTEGRATION TEST FAIL: expected >0 rows, got %ld\n", count);
        return 6;
    }
}
