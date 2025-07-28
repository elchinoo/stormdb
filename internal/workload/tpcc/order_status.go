// internal/workload/tpcc/order_status.go
package tpcc

import (
	"context"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) orderStatusTx(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	w_id := 1
	d_id := rng.Intn(10) + 1

	var c_id int
	c_by := rng.Intn(100)
	if c_by < 60 {
		// 60%: by last name
		// Find a customer by last name (assume we know it)
		var last string
		switch rng.Intn(3) {
		case 0:
			last = "BROWN"
		case 1:
			last = "SMITH"
		default:
			last = "CUSTOMER"
		}
		// Find the middle customer by last name (simplified)
		rows, err := db.Query(ctx, "SELECT c_id FROM customer WHERE c_w_id = $1::INT AND c_d_id = $2::INT AND c_last = $3 ORDER BY c_id", w_id, d_id, last)
		if err != nil {
			return err
		}
		defer rows.Close()

		c_ids := []int{}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return err
			}
			c_ids = append(c_ids, id)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		if len(c_ids) > 0 {
			c_id = c_ids[len(c_ids)/2] // middle
		} else {
			c_id = rng.Intn(300) + 1
		}
	} else {
		// 40%: by customer ID
		c_id = rng.Intn(300) + 1
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get customer
	var c_balance float64
	err = tx.QueryRow(ctx, "SELECT c_balance FROM customer WHERE c_w_id = $1::INT AND c_d_id = $2::INT AND c_id = $3::INT", w_id, d_id, c_id).Scan(&c_balance)
	if err != nil {
		return err
	}

	// Get last order
	var o_id, o_carrier_id, o_entry_d int
	err = tx.QueryRow(ctx, "SELECT o_id, o_carrier_id, EXTRACT(EPOCH FROM o_entry_d)::INT FROM orders WHERE o_w_id = $1::INT AND o_d_id = $2::INT AND o_c_id = $3::INT ORDER BY o_id DESC LIMIT 1",
		w_id, d_id, c_id).Scan(&o_id, &o_carrier_id, &o_entry_d)
	if err != nil {
		// No order, still valid
		return tx.Commit(ctx)
	}

	// Get order lines
	rows, err := tx.Query(ctx, "SELECT ol_i_id, ol_supply_w_id, ol_quantity, ol_amount, EXTRACT(EPOCH FROM ol_delivery_d)::INT FROM order_line WHERE ol_w_id = $1::INT AND ol_d_id = $2::INT AND ol_o_id = $3::INT",
		w_id, d_id, o_id)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var ol_i_id, ol_supply_w_id, ol_quantity int
		var ol_amount float64
		var ol_delivery_d *int
		if err := rows.Scan(&ol_i_id, &ol_supply_w_id, &ol_quantity, &ol_amount, &ol_delivery_d); err != nil {
			return err
		}
		// Just read — no processing
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
