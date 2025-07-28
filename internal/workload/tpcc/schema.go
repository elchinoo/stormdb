// internal/workload/tpcc/schema.go
package tpcc

import (
	"context"
	"fmt"
	"log"

	"github.com/elchinoo/stormdb/pkg/types"

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

	// Now use cfg.Scale to determine how many rows to insert
	log.Printf("Seeding TPCC data with scale factor %d...", cfg.Scale)
	if err := t.loadInitialData(ctx, db, cfg.Scale); err != nil {
		return err
	}

	return nil
}

func (t *TPCC) loadInitialData(ctx context.Context, db *pgxpool.Pool, scale int) error {
	if scale <= 0 {
		scale = 1
	}

	log.Printf("🏗️  Seeding TPCC data with scale = %d warehouses", scale)

	for w := 1; w <= scale; w++ {
		// Insert warehouse
		_, err := db.Exec(ctx, "INSERT INTO warehouse (w_id, w_name, w_tax, w_ytd) VALUES ($1, $2, 0.1, 300000) ON CONFLICT (w_id) DO NOTHING",
			w, fmt.Sprintf("WH%d", w))
		if err != nil {
			return fmt.Errorf("failed to insert warehouse %d: %v", w, err)
		}

		// Each warehouse has 10 districts
		for d := 1; d <= 10; d++ {
			_, err := db.Exec(ctx, "INSERT INTO district (d_id, d_w_id, d_name, d_tax, d_ytd, d_next_o_id) VALUES ($1::INT, $2::INT, $3, 0.1, 30000, 3001) ON CONFLICT (d_w_id, d_id) DO NOTHING",
				d, w, fmt.Sprintf("District%d-%d", w, d))
			if err != nil {
				return fmt.Errorf("failed to insert district %d for warehouse %d: %v", d, w, err)
			}

			// Each district has 30000 customers
			for c := 1; c <= 30000; c++ {
				_, err := db.Exec(ctx, `
                    INSERT INTO customer (
                        c_id, c_d_id, c_w_id, c_first, c_last, c_since, c_credit, c_balance
                    ) VALUES ($1, $2, $3, $4, 'CUSTOMER', NOW(), 'GC', 0)
                    ON CONFLICT (c_w_id, c_d_id, c_id) DO NOTHING`,
					c, d, w, fmt.Sprintf("First%d", c))
				if err != nil {
					return fmt.Errorf("failed to insert customer %d in district %d, warehouse %d: %v", c, d, w, err)
				}
			}
		}
	}

	log.Printf("✅ Seeded %d warehouses, %d districts, %d customers", scale, scale*10, scale*10*30000)
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

	// Recreate schema
	if err := t.Setup(ctx, db, cfg); err != nil {
		return fmt.Errorf("failed to recreate schema: %w", err)
	}

	// Seed data only during rebuild
	if err := t.loadInitialData(ctx, db, cfg.Scale); err != nil {
		return fmt.Errorf("failed to load initial data: %w", err)
	}

	log.Printf("🌱 Seeded TPCC data with scale = %d", cfg.Scale)
	return nil
}
