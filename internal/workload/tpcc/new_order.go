// internal/workload/tpcc/new_order.go
package tpcc

import (
	"context"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) newOrderTx(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	w_id := 1
	d_id := rng.Intn(10) + 1
	c_id := rng.Intn(300) + 1

	// 1% of orders have a remote item (from another warehouse)
	remote := rng.Intn(100) == 0
	i_ids := make([]int, 0)
	ol_quantities := make([]int, 0)

	// Pick 5 to 15 order lines
	ol_count := 5 + rng.Intn(11)

	for i := 0; i < ol_count; i++ {
		i_id := rng.Intn(10000) + 1
		quantity := 1 + rng.Intn(10)
		i_ids = append(i_ids, i_id)
		ol_quantities = append(ol_quantities, quantity)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get next order ID
	var next_o_id int
	err = tx.QueryRow(ctx, "SELECT d_next_o_id FROM district WHERE d_w_id = $1::INT AND d_id = $2::INT FOR UPDATE", w_id, d_id).Scan(&next_o_id)
	if err != nil {
		return err
	}

	// Insert order
	o_all_local := 1
	if remote {
		o_all_local = 0
	}
	_, err = tx.Exec(ctx, "INSERT INTO orders (o_id, o_d_id, o_w_id, o_c_id, o_entry_d, o_carrier_id, o_ol_cnt, o_all_local) VALUES ($1, $2::INT, $3, $4, NOW(), NULL, $5, $6)",
		next_o_id, d_id, w_id, c_id, ol_count, o_all_local)
	if err != nil {
		return err
	}

	// Increment next order ID
	_, err = tx.Exec(ctx, "UPDATE district SET d_next_o_id = d_next_o_id + 1 WHERE d_w_id = $1::INT AND d_id = $2::INT", w_id, d_id)
	if err != nil {
		return err
	}

	// Insert order lines
	for i := 0; i < ol_count; i++ {
		ol_number := i + 1
		ol_i_id := i_ids[i]
		ol_supply_w_id := w_id
		if remote && i == ol_count-1 {
			ol_supply_w_id = (w_id % 1) + 1 // Only 1 warehouse in our test
		}
		ol_quantity := ol_quantities[i]

		// Generate random amount
		ol_amount := 1.0 + rng.Float64()*99.0

		// Insert order line
		_, err = tx.Exec(ctx, "INSERT INTO order_line (ol_o_id, ol_d_id, ol_w_id, ol_number, ol_i_id, ol_supply_w_id, ol_delivery_d, ol_quantity, ol_amount, ol_dist_info) VALUES ($1, $2::INT, $3, $4, $5, $6, NULL, $7, $8, 'S_DIST_' || lpad($2::text, 2, '0'))",
			next_o_id, d_id, w_id, ol_number, ol_i_id, ol_supply_w_id, ol_quantity, ol_amount)
		if err != nil {
			return err
		}

		// Update stock (simulate)
		// In real TPC-C, stock is updated here, but we skip for now (no stock table)
		// You can add it later if needed
	}

	return tx.Commit(ctx)
}
