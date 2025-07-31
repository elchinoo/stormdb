// internal/workload/tpcc/schema.go
package tpcc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/elchinoo/stormdb/internal/progress"
	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS warehouse (
            w_id SERIAL PRIMARY KEY,
            w_name TEXT,
            w_street TEXT,
            w_city TEXT,
            w_state CHAR(2),
            w_zip CHAR(9),
            w_tax DECIMAL(4,4),
            w_ytd DECIMAL(12,2)
        )`,
		`CREATE TABLE IF NOT EXISTS district (
            d_id SMALLINT,
            d_w_id INT REFERENCES warehouse(w_id),
            d_name TEXT,
            d_street TEXT,
            d_city TEXT,
            d_state CHAR(2),
            d_zip CHAR(9),
            d_tax DECIMAL(4,4),
            d_ytd DECIMAL(12,2),
            d_next_o_id INT,
            PRIMARY KEY (d_w_id, d_id)
        )`,
		`CREATE TABLE IF NOT EXISTS customer (
            c_id SERIAL,
            c_d_id SMALLINT,
            c_w_id INT,
            c_first TEXT,
            c_middle CHAR(2),
            c_last TEXT,
            c_street TEXT,
            c_city TEXT,
            c_state CHAR(2),
            c_zip CHAR(9),
            c_phone CHAR(16),
            c_since TIMESTAMPTZ,
            c_credit CHAR(2),
            c_credit_lim DECIMAL(12,2),
            c_discount DECIMAL(4,4),
            c_balance DECIMAL(12,2),
            c_ytd_pay DECIMAL(12,2),
            c_payment_cnt INT,
            c_delivery_cnt INT,
            c_data TEXT,
            PRIMARY KEY (c_w_id, c_d_id, c_id)
        )`,
		`CREATE TABLE IF NOT EXISTS orders (
            o_id INT,
            o_d_id SMALLINT,
            o_w_id INT,
            o_c_id INT,
            o_entry_d TIMESTAMPTZ,
            o_carrier_id INT,
            o_ol_cnt INT,
            o_all_local INT,
            PRIMARY KEY (o_w_id, o_d_id, o_id)
        )`,
		`CREATE TABLE IF NOT EXISTS order_line (
            ol_o_id INT,
            ol_d_id SMALLINT,
            ol_w_id INT,
            ol_number INT,
            ol_i_id INT,
            ol_supply_w_id INT,
            ol_delivery_d TIMESTAMPTZ,
            ol_quantity INT,
            ol_amount DECIMAL(6,2),
            ol_dist_info CHAR(24),
            PRIMARY KEY (ol_w_id, ol_d_id, ol_o_id, ol_number)
        )`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	log.Printf("✅ TPCC schema created")

	// Check if data already exists to avoid duplicates
	var warehouseCount int64
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM warehouse").Scan(&warehouseCount)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %v", err)
	}

	// Only load data if tables are empty
	if warehouseCount == 0 {
		log.Printf("Seeding TPCC data with scale factor %d...", cfg.Scale)
		if err := t.loadInitialData(ctx, db, cfg.Scale); err != nil {
			return fmt.Errorf("failed to load initial data: %v", err)
		}
	} else {
		log.Printf("✅ TPCC data already exists (%d warehouses)", warehouseCount)
	}

	return nil
}

func (t *TPCC) loadInitialData(ctx context.Context, db *pgxpool.Pool, scale int) error {
	if scale <= 0 {
		scale = 1
	}

	// For small scale factors (likely tests), use reduced customer count to avoid timeouts
	customersPerDistrict := 30000
	if scale <= 2 {
		customersPerDistrict = 100 // Much smaller for testing
		log.Printf("🧪 Using reduced customer count (%d) for small scale testing", customersPerDistrict)
	}

	log.Printf("🏗️  Seeding TPCC data with scale = %d warehouses", scale)

	// Calculate total operations for progress tracking
	totalWarehouses := scale
	totalDistricts := scale * 10
	totalCustomers := scale * 10 * customersPerDistrict

	// Create progress trackers
	warehouseProgress := progress.NewTracker("📦 Loading warehouses", totalWarehouses)
	districtProgress := progress.NewTracker("🏢 Loading districts", totalDistricts)
	customerProgress := progress.NewTracker("👥 Loading customers", totalCustomers)

	// Load warehouses (small number, individual inserts are fine)
	for w := 1; w <= scale; w++ {
		_, err := db.Exec(ctx, "INSERT INTO warehouse (w_id, w_name, w_tax, w_ytd) VALUES ($1, $2, 0.1, 300000) ON CONFLICT (w_id) DO NOTHING",
			w, fmt.Sprintf("WH%d", w))
		if err != nil {
			return fmt.Errorf("failed to insert warehouse %d: %v", w, err)
		}
		warehouseProgress.Update(w)
	}

	// Load districts using COPY protocol for better performance
	if err := t.loadDistrictsBatch(ctx, db, scale, districtProgress); err != nil {
		return fmt.Errorf("failed to load districts: %v", err)
	}

	// Load customers using COPY protocol for maximum speed
	if err := t.loadCustomersBatch(ctx, db, scale, customersPerDistrict, customerProgress); err != nil {
		return fmt.Errorf("failed to load customers: %v", err)
	}

	log.Printf("✅ Seeded %d warehouses, %d districts, %d customers", scale, scale*10, scale*10*customersPerDistrict)
	return nil
}

// loadDistrictsBatch loads districts using batch insert for better performance
func (t *TPCC) loadDistrictsBatch(ctx context.Context, db *pgxpool.Pool, scale int, progress *progress.Tracker) error {
	// Prepare data for batch insert
	rows := make([][]interface{}, 0, scale*10)

	for w := 1; w <= scale; w++ {
		for d := 1; d <= 10; d++ {
			rows = append(rows, []interface{}{
				d,                                  // d_id
				w,                                  // d_w_id
				fmt.Sprintf("District%d-%d", w, d), // d_name
				0.1,                                // d_tax
				30000,                              // d_ytd
				3001,                               // d_next_o_id
			})
		}
	}

	// Use COPY for districts too for consistency and speed
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	copySource := pgx.CopyFromRows(rows)
	rowsAffected, err := conn.Conn().CopyFrom(ctx,
		pgx.Identifier{"district"},
		[]string{"d_id", "d_w_id", "d_name", "d_tax", "d_ytd", "d_next_o_id"},
		copySource)
	if err != nil {
		return fmt.Errorf("failed to COPY districts: %v", err)
	}

	progress.Update(len(rows))
	log.Printf("📈 COPY inserted %d district rows", rowsAffected)

	return nil
}

// loadCustomersBatch loads customers using COPY protocol for maximum performance
func (t *TPCC) loadCustomersBatch(ctx context.Context, db *pgxpool.Pool, scale int, customersPerDistrict int, progress *progress.Tracker) error {
	// Prepare data for COPY protocol
	rows := make([][]interface{}, 0, scale*10*customersPerDistrict)
	now := time.Now() // Use actual timestamp instead of "NOW()"

	for w := 1; w <= scale; w++ {
		for d := 1; d <= 10; d++ {
			for c := 1; c <= customersPerDistrict; c++ {
				rows = append(rows, []interface{}{
					c,                         // c_id
					d,                         // c_d_id
					w,                         // c_w_id
					fmt.Sprintf("First%d", c), // c_first
					"CUSTOMER",                // c_last
					now,                       // c_since - use actual timestamp
					"GC",                      // c_credit
					0,                         // c_balance
				})
			}
		}
	}

	// Get a connection from the pool for COPY
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	// Use COPY protocol for bulk insert
	copySource := pgx.CopyFromRows(rows)
	rowsAffected, err := conn.Conn().CopyFrom(ctx,
		pgx.Identifier{"customer"},
		[]string{"c_id", "c_d_id", "c_w_id", "c_first", "c_last", "c_since", "c_credit", "c_balance"},
		copySource)
	if err != nil {
		return fmt.Errorf("failed to COPY customers: %v", err)
	}

	progress.Update(len(rows))
	log.Printf("📈 COPY inserted %d customer rows", rowsAffected)

	return nil
}

// internal/workload/tpcc/schema.go
func (t *TPCC) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	tables := []string{"order_line", "orders", "customer", "district", "warehouse"}
	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}
	log.Printf("🗑️  Dropped TPCC tables")

	// Only recreate schema - data loading will happen in Setup()
	if err := t.Setup(ctx, db, cfg); err != nil {
		return fmt.Errorf("failed to recreate schema: %w", err)
	}

	log.Printf("🔧 TPCC schema recreated (data loading will happen in Setup)")
	return nil
}
