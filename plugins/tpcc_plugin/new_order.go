// internal/workload/tpcc/new_order.go
package main

import (
	"context"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) newOrderTx(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	wID := 1
	dID := rng.Intn(10) + 1
	cID := rng.Intn(300) + 1

	// 1% of orders have a remote item (from another warehouse)
	remote := rng.Intn(100) == 0
	iIDs := make([]int, 0)
	olQuantities := make([]int, 0)

	// Pick 5 to 15 order lines
	olCount := 5 + rng.Intn(11)

	for i := 0; i < olCount; i++ {
		iID := rng.Intn(10000) + 1
		quantity := 1 + rng.Intn(10)
		iIDs = append(iIDs, iID)
		olQuantities = append(olQuantities, quantity)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Get next order ID
	var nextOID int
	err = tx.QueryRow(ctx, "SELECT d_next_o_id FROM district WHERE d_w_id = $1::INT AND d_id = $2::INT FOR UPDATE", wID, dID).Scan(&nextOID)
	if err != nil {
		return err
	}

	// Insert order
	oAllLocal := 1
	if remote {
		oAllLocal = 0
	}
	_, err = tx.Exec(ctx, "INSERT INTO orders (o_id, o_d_id, o_w_id, o_c_id, o_entry_d, o_carrier_id, o_ol_cnt, o_all_local) VALUES ($1, $2::INT, $3, $4, NOW(), NULL, $5, $6)",
		nextOID, dID, wID, cID, olCount, oAllLocal)
	if err != nil {
		return err
	}

	// Increment next order ID
	_, err = tx.Exec(ctx, "UPDATE district SET d_next_o_id = d_next_o_id + 1 WHERE d_w_id = $1::INT AND d_id = $2::INT", wID, dID)
	if err != nil {
		return err
	}

	// Insert order lines
	for i := 0; i < olCount; i++ {
		olNumber := i + 1
		olIID := iIDs[i]
		olSupplyWID := wID
		if remote && i == olCount-1 {
			olSupplyWID = wID // Fixed: avoid x % 1 which is always 0
		}
		olQuantity := olQuantities[i]

		// Generate random amount
		olAmount := 1.0 + rng.Float64()*99.0

		// Insert order line
		_, err = tx.Exec(ctx, "INSERT INTO order_line (ol_o_id, ol_d_id, ol_w_id, ol_number, ol_i_id, ol_supply_w_id, ol_delivery_d, ol_quantity, ol_amount, ol_dist_info) VALUES ($1, $2::INT, $3, $4, $5, $6, NULL, $7, $8, 'S_DIST_' || lpad($2::text, 2, '0'))",
			nextOID, dID, wID, olNumber, olIID, olSupplyWID, olQuantity, olAmount)
		if err != nil {
			return err
		}

		// Update stock (simulate)
		// In real TPC-C, stock is updated here, but we skip for now (no stock table)
		// You can add it later if needed
	}

	return tx.Commit(ctx)
}
