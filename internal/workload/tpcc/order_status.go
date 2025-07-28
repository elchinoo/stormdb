// internal/workload/tpcc/order_status.go
package tpcc

import (
	"context"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) orderStatusTx(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	wID := 1
	dID := rng.Intn(10) + 1

	var cID int
	cBy := rng.Intn(100)
	if cBy < 60 {
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
		rows, err := db.Query(ctx, "SELECT c_id FROM customer WHERE c_w_id = $1::INT AND c_d_id = $2::INT AND c_last = $3 ORDER BY c_id", wID, dID, last)
		if err != nil {
			return err
		}
		defer rows.Close()

		cIDs := []int{}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return err
			}
			cIDs = append(cIDs, id)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		if len(cIDs) > 0 {
			cID = cIDs[len(cIDs)/2] // middle
		} else {
			cID = rng.Intn(300) + 1
		}
	} else {
		// 40%: by customer ID
		cID = rng.Intn(300) + 1
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Get customer
	var cBalance float64
	err = tx.QueryRow(ctx, "SELECT c_balance FROM customer WHERE c_w_id = $1::INT AND c_d_id = $2::INT AND c_id = $3::INT", wID, dID, cID).Scan(&cBalance)
	if err != nil {
		return err
	}

	// Get last order
	var oID, oCarrierID, oEntryD int
	err = tx.QueryRow(ctx, "SELECT o_id, o_carrier_id, EXTRACT(EPOCH FROM o_entry_d)::INT FROM orders WHERE o_w_id = $1::INT AND o_d_id = $2::INT AND o_c_id = $3::INT ORDER BY o_id DESC LIMIT 1",
		wID, dID, cID).Scan(&oID, &oCarrierID, &oEntryD)
	if err != nil {
		// No order, still valid
		return tx.Commit(ctx)
	}

	// Get order lines
	rows, err := tx.Query(ctx, "SELECT ol_i_id, ol_supply_w_id, ol_quantity, ol_amount, EXTRACT(EPOCH FROM ol_delivery_d)::INT FROM order_line WHERE ol_w_id = $1::INT AND ol_d_id = $2::INT AND ol_o_id = $3::INT",
		wID, dID, oID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var olIID, olSupplyWID, olQuantity int
		var olAmount float64
		var olDeliveryD *int
		if err := rows.Scan(&olIID, &olSupplyWID, &olQuantity, &olAmount, &olDeliveryD); err != nil {
			return err
		}
		// Just read — no processing
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
