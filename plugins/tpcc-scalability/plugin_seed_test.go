package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/elchinoo/stormdb/core"
	_ "github.com/lib/pq"
)

// testLogger is a no-op logger for plugin tests
type testLogger struct{}

func (l testLogger) Debug(msg string, fields ...core.Field)              {}
func (l testLogger) Info(msg string, fields ...core.Field)               {}
func (l testLogger) Warn(msg string, fields ...core.Field)               {}
func (l testLogger) Error(msg string, fields ...core.Field)              {}
func (l testLogger) WithFields(fields ...core.Field) core.Logger         { return l }
func (l testLogger) WithPlugin(pluginName string) core.Logger            { return l }
func (l testLogger) WithStorage(storage core.StorageManager) core.Logger { return l }

// openTestDB connects to a Postgres instance defined by TEST_DB_DSN
func openTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=tpcc_test sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to ping test db: %v", err)
	}
	return db
}

// cleanupDB drops all TPC-C tables
func cleanupDB(t *testing.T, db *sql.DB) {
	plugin := &TPCCPlugin{db: db, logger: testLogger{}, cfg: &TPCCConfig{Scale: 1}}
	if err := plugin.dropTables(context.Background()); err != nil {
		t.Fatalf("dropTables failed: %v", err)
	}
	// drop dynamic tables to reset state
	for _, tbl := range []string{"stock", "item", "goods_receipt", "purchase_order", "supplier"} {
		if _, err := db.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tbl)); err != nil {
			t.Fatalf("failed to drop table %s: %v", tbl, err)
		}
	}
}

func TestPopulateSeedData(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	defer db.Close()
	// clean slate
	cleanupDB(t, db)
	// migrate schema
	plugin := &TPCCPlugin{db: db, logger: testLogger{}, cfg: &TPCCConfig{Scale: 2}}
	if err := plugin.setupSchema(ctx); err != nil {
		t.Fatalf("setupSchema failed: %v", err)
	}
	// populate seed
	if err := plugin.populateSeedData(ctx); err != nil {
		t.Fatalf("populateSeedData failed: %v", err)
	}
	// verify warehouses
	var wCount int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM warehouse`).Scan(&wCount)
	if err != nil {
		t.Fatalf("count warehouse failed: %v", err)
	}
	if wCount != 2 {
		t.Errorf("expected 2 warehouses, got %d", wCount)
	}
	// verify districts (2 warehouses * 10 districts)
	var dCount int
	err = db.QueryRowContext(ctx, `SELECT count(*) FROM district`).Scan(&dCount)
	if err != nil {
		t.Fatalf("count district failed: %v", err)
	}
	if dCount != 20 {
		t.Errorf("expected 20 districts, got %d", dCount)
	}
	// verify warehouse address fields
	var street, city, state, zip string
	row := db.QueryRowContext(ctx,
		`SELECT w_street_1, w_city, w_state, w_zip FROM warehouse WHERE w_id = $1`, 1)
	if err := row.Scan(&street, &city, &state, &zip); err != nil {
		t.Fatalf("failed to scan warehouse address: %v", err)
	}
	// trim padded zip
	zipTrim := strings.TrimSpace(zip)
	if street != "Street1" || city != "City" || state != "ST" || zipTrim != "00000" {
		t.Errorf("unexpected warehouse address: street=%s, city=%s, state=%s, zip=%s", street, city, state, zipTrim)
	}
	// verify district default next order ID
	var nextOID int
	err = db.QueryRowContext(ctx,
		`SELECT d_next_o_id FROM district WHERE d_w_id = $1 AND d_id = $2`, 1, 1).Scan(&nextOID)
	if err != nil {
		t.Fatalf("failed to scan district next_o_id: %v", err)
	}
	if nextOID != 3001 {
		t.Errorf("expected district next_o_id 3001, got %d", nextOID)
	}
}

// TestPopulateTPCCData verifies that populateTPCCData inserts items, stock, and customers
func TestPopulateTPCCData(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	defer db.Close()
	// clean and setup schema
	cleanupDB(t, db)
	plugin := &TPCCPlugin{db: db, logger: testLogger{}, cfg: &TPCCConfig{Scale: 1}}
	if err := plugin.setupSchema(ctx); err != nil {
		t.Fatalf("setupSchema failed: %v", err)
	}
	if err := plugin.populateSeedData(ctx); err != nil {
		t.Fatalf("populateSeedData failed: %v", err)
	}
	if err := plugin.populateTPCCData(ctx); err != nil {
		t.Fatalf("populateTPCCData failed: %v", err)
	}
	// verify at least one item
	var itemCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM item`).Scan(&itemCount); err != nil {
		t.Fatalf("count item failed: %v", err)
	}
	if itemCount <= 0 {
		t.Errorf("expected >0 items, got %d", itemCount)
	}
	// verify at least one stock row
	var stockCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM stock`).Scan(&stockCount); err != nil {
		t.Fatalf("count stock failed: %v", err)
	}
	if stockCount <= 0 {
		t.Errorf("expected >0 stock rows, got %d", stockCount)
	}
	// verify at least one customer
	var custCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM customer`).Scan(&custCount); err != nil {
		t.Fatalf("count customer failed: %v", err)
	}
	if custCount <= 0 {
		t.Errorf("expected >0 customers, got %d", custCount)
	}
}
