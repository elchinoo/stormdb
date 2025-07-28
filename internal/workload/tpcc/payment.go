// internal/workload/tpcc/payment.go
package tpcc

import (
	"context"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (t *TPCC) paymentTx(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	wID := 1
	dID := rng.Intn(10) + 1
	cID := rng.Intn(300) + 1

	amount := 10.0 + rng.Float64()*90.0

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Update warehouse
	_, err = tx.Exec(ctx, "UPDATE warehouse SET w_ytd = w_ytd + $1 WHERE w_id = $2::INT", amount, wID)
	if err != nil {
		return err
	}

	// Update district
	_, err = tx.Exec(ctx, "UPDATE district SET d_ytd = d_ytd + $1 WHERE d_w_id = $2::INT AND d_id = $3::INT", amount, wID, dID)
	if err != nil {
		return err
	}

	// Update customer
	_, err = tx.Exec(ctx, "UPDATE customer SET c_balance = c_balance - $1, c_ytd_pay = c_ytd_pay + $1, c_payment_cnt = c_payment_cnt + 1 WHERE c_w_id = $2::INT AND c_d_id = $3::INT AND c_id = $4::INT",
		amount, wID, dID, cID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
