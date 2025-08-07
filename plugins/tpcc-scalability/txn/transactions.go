package txn

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// TransactionType defines supported txn codes
type TransactionType string

const (
	NewOrderTxn        TransactionType = "new_order"
	PaymentTxn         TransactionType = "payment"
	OrderStatusTxn     TransactionType = "order_status"
	DeliveryTxn        TransactionType = "delivery"
	StockLevelTxn      TransactionType = "stock_level"
	SupplierReorderTxn TransactionType = "supplier_reorder"
)

// MetricsRecorder interface for recording transaction metrics
type MetricsRecorder interface {
	RecordOperation(workerID int, operation string, latency time.Duration, rowsAffected int64, isError bool)
}

// ExecuteTransactionWithMetrics dispatches transactions with enhanced metrics tracking
func ExecuteTransactionWithMetrics(ctx context.Context, db *sql.DB, tt TransactionType, warehouseID, scale, workerID int, metrics MetricsRecorder) error {
	switch tt {
	case NewOrderTxn:
		return newOrderTransactionWithMetrics(ctx, db, warehouseID, workerID, metrics)
	case PaymentTxn:
		return paymentTransactionWithMetrics(ctx, db, warehouseID, workerID, metrics)
	case OrderStatusTxn:
		return orderStatusTransactionWithMetrics(ctx, db, warehouseID, workerID, metrics)
	case DeliveryTxn:
		return deliveryTransactionWithMetrics(ctx, db, warehouseID, workerID, metrics)
	case StockLevelTxn:
		return stockLevelTransactionWithMetrics(ctx, db, warehouseID, workerID, metrics)
	case SupplierReorderTxn:
		return supplierReorderTransactionWithMetrics(ctx, db, warehouseID, scale, workerID, metrics)
	default:
		return fmt.Errorf("unknown transaction type: %s", tt)
	}
}

// ExecuteTransaction dispatches to specific txn implementations (legacy)
func ExecuteTransaction(ctx context.Context, db *sql.DB, tt TransactionType, warehouseID, scale int) error {
	switch tt {
	case NewOrderTxn:
		return newOrderTransaction(ctx, db, warehouseID)
	case PaymentTxn:
		return paymentTransaction(ctx, db, warehouseID)
	case OrderStatusTxn:
		return orderStatusTransaction(ctx, db, warehouseID)
	case DeliveryTxn:
		return deliveryTransaction(ctx, db, warehouseID)
	case StockLevelTxn:
		return stockLevelTransaction(ctx, db, warehouseID)
	case SupplierReorderTxn:
		return supplierReorderTransaction(ctx, db, warehouseID, scale)
	default:
		return fmt.Errorf("unknown transaction type: %s", tt)
	}
}

// newOrderTransactionWithMetrics implements the New-Order transaction with enhanced metrics
func newOrderTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID int, workerID int, metrics MetricsRecorder) error {
	// Use Read Committed with explicit locking for order ID generation
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	dID := rand.Intn(10) + 1
	var oID int

	// Get next order ID atomically using sequence-based function
	selectStart := time.Now()
	if err := tx.QueryRowContext(ctx,
		"SELECT get_next_order_id($1, $2)",
		warehouseID, dID).Scan(&oID); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	cID := rand.Intn(3000) + 1
	entryD := time.Now()
	olCnt := rand.Intn(11) + 5

	// INSERT into order
	insertStart := time.Now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO "order" (o_w_id,o_d_id,o_id,o_c_id,o_entry_d,o_ol_cnt,o_all_local)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		warehouseID, dID, oID, cID, entryD, olCnt, 1); err != nil {
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)

	// INSERT into new_order
	insertStart = time.Now()
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO new_order (no_w_id,no_d_id,no_o_id) VALUES ($1,$2,$3)",
		warehouseID, dID, oID); err != nil {
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)

	// Process order lines
	var totalInserts, totalUpdates, totalSelects int64
	for lineNum := 1; lineNum <= olCnt; lineNum++ {
		itemID := rand.Intn(100000) + 1
		qty := rand.Intn(10) + 1

		// SELECT item price
		selectStart := time.Now()
		var price float64
		if err := tx.QueryRowContext(ctx,
			"SELECT i_price FROM item WHERE i_id=$1", itemID).Scan(&price); err != nil {
			metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)
		totalSelects++

		amount := price * float64(qty)

		// INSERT order_line
		insertStart := time.Now()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO order_line (ol_w_id,ol_d_id,ol_o_id,ol_number,ol_i_id,ol_supply_w_id,ol_delivery_d,ol_quantity,ol_amount,ol_dist_info)
			VALUES ($1,$2,$3,$4,$5,$6,NULL,$7,$8,$9)`,
			warehouseID, dID, oID, lineNum, itemID, warehouseID, qty, amount, "DIST"); err != nil {
			metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)
		totalInserts++

		// UPDATE stock
		updateStart := time.Now()
		if _, err := tx.ExecContext(ctx,
			`UPDATE stock SET s_quantity=s_quantity-$1, s_ytd=s_ytd+$1, s_order_cnt=s_order_cnt+1
			WHERE s_w_id=$2 AND s_i_id=$3`,
			qty, warehouseID, itemID); err != nil {
			metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)
		totalUpdates++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// paymentTransactionWithMetrics implements the Payment transaction with enhanced metrics
func paymentTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID int, workerID int, metrics MetricsRecorder) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	dID := rand.Intn(10) + 1
	amount := rand.Float64() * 5000.0

	// UPDATE warehouse
	updateStart := time.Now()
	if _, err := tx.ExecContext(ctx,
		"UPDATE warehouse SET w_ytd = w_ytd + $1 WHERE w_id=$2", amount, warehouseID); err != nil {
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)

	// UPDATE district
	updateStart = time.Now()
	if _, err := tx.ExecContext(ctx,
		"UPDATE district SET d_ytd = d_ytd + $1 WHERE d_w_id=$2 AND d_id=$3", amount, warehouseID, dID); err != nil {
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)

	cID := rand.Intn(3000) + 1

	// UPDATE customer
	updateStart = time.Now()
	if _, err := tx.ExecContext(ctx,
		`UPDATE customer SET c_balance = c_balance - $1, c_ytd_payment = c_ytd_payment + $1, c_payment_cnt = c_payment_cnt + 1
		 WHERE c_w_id=$2 AND c_d_id=$3 AND c_id=$4`,
		amount, warehouseID, dID, cID); err != nil {
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)

	// SELECT warehouse name
	selectStart := time.Now()
	var wName string
	if err := tx.QueryRowContext(ctx,
		"SELECT w_name FROM warehouse WHERE w_id=$1", warehouseID).Scan(&wName); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	// SELECT district name
	selectStart = time.Now()
	var dName string
	if err := tx.QueryRowContext(ctx,
		"SELECT d_name FROM district WHERE d_w_id=$1 AND d_id=$2", warehouseID, dID).Scan(&dName); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	hData := fmt.Sprintf("%s    %s", wName, dName)
	now := time.Now()

	// INSERT history
	insertStart := time.Now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO history (h_c_id,h_c_d_id,h_c_w_id,h_d_id,h_w_id,h_date,h_amount,h_data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		cID, dID, warehouseID, dID, warehouseID, now, amount, hData); err != nil {
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// orderStatusTransactionWithMetrics implements the Order-Status transaction with enhanced metrics
func orderStatusTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID int, workerID int, metrics MetricsRecorder) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	dID := rand.Intn(10) + 1
	cID := rand.Intn(3000) + 1

	// SELECT customer balance
	selectStart := time.Now()
	var cBalance float64
	if err := tx.QueryRowContext(ctx,
		"SELECT c_balance FROM customer WHERE c_w_id=$1 AND c_d_id=$2 AND c_id=$3",
		warehouseID, dID, cID).Scan(&cBalance); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	// SELECT latest order
	selectStart = time.Now()
	var oID int
	var oEntryD time.Time
	var oCarrierID sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT o_id, o_entry_d, o_carrier_id FROM "order" WHERE o_w_id=$1 AND o_d_id=$2 AND o_c_id=$3 ORDER BY o_entry_d DESC LIMIT 1`,
		warehouseID, dID, cID).Scan(&oID, &oEntryD, &oCarrierID); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	// SELECT order lines
	selectStart = time.Now()
	rows, err := tx.QueryContext(ctx,
		`SELECT ol_i_id, ol_supply_w_id, ol_quantity, ol_amount, ol_delivery_d FROM order_line WHERE ol_w_id=$1 AND ol_d_id=$2 AND ol_o_id=$3`,
		warehouseID, dID, oID)
	if err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 0, true)
		return err
	}
	defer rows.Close()

	var rowCount int64
	for rows.Next() {
		var itemID, supplyWID, qty int
		var amount float64
		var deliveryDate sql.NullTime
		if err := rows.Scan(&itemID, &supplyWID, &qty, &amount, &deliveryDate); err != nil {
			rows.Close()
			metrics.RecordOperation(workerID, "select", time.Since(selectStart), rowCount, true)
			return err
		}
		rowCount++
	}
	if err := rows.Err(); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), rowCount, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), rowCount, false)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// deliveryTransactionWithMetrics implements the Delivery transaction with enhanced metrics
func deliveryTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID int, workerID int, metrics MetricsRecorder) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	carrierID := rand.Intn(10) + 1
	now := time.Now()

	var totalDeletes, totalUpdates, totalSelects int64

	for dID := 1; dID <= 10; dID++ {
		// SELECT from new_order with row-level locking to prevent concurrent access
		selectStart := time.Now()
		var oID int
		if err := tx.QueryRowContext(ctx,
			`SELECT no_o_id FROM new_order WHERE no_w_id=$1 AND no_d_id=$2 ORDER BY no_o_id FOR UPDATE SKIP LOCKED LIMIT 1`,
			warehouseID, dID).Scan(&oID); err != nil {
			if err == sql.ErrNoRows {
				metrics.RecordOperation(workerID, "select", time.Since(selectStart), 0, false)
				continue
			}
			metrics.RecordOperation(workerID, "select", time.Since(selectStart), 0, true)
			return err
		}
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)
		totalSelects++

		// DELETE from new_order
		deleteStart := time.Now()
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM new_order WHERE no_w_id=$1 AND no_d_id=$2 AND no_o_id=$3`,
			warehouseID, dID, oID); err != nil {
			metrics.RecordOperation(workerID, "delete", time.Since(deleteStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "delete", time.Since(deleteStart), 1, false)
		totalDeletes++

		// UPDATE order
		updateStart := time.Now()
		if _, err := tx.ExecContext(ctx,
			`UPDATE "order" SET o_carrier_id=$1 WHERE o_w_id=$2 AND o_d_id=$3 AND o_id=$4`,
			carrierID, warehouseID, dID, oID); err != nil {
			metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)
		totalUpdates++

		// UPDATE order_line
		updateStart = time.Now()
		if _, err := tx.ExecContext(ctx,
			`UPDATE order_line SET ol_delivery_d=$1 WHERE ol_w_id=$2 AND ol_d_id=$3 AND ol_o_id=$4`,
			now, warehouseID, dID, oID); err != nil {
			metrics.RecordOperation(workerID, "update", time.Since(updateStart), 0, true)
			return err
		}
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)
		totalUpdates++

		// SELECT customer ID
		selectStart = time.Now()
		var cID int
		if err := tx.QueryRowContext(ctx,
			`SELECT o_c_id FROM "order" WHERE o_w_id=$1 AND o_d_id=$2 AND o_id=$3`,
			warehouseID, dID, oID).Scan(&cID); err != nil {
			metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)
		totalSelects++

		// SELECT total amount
		selectStart = time.Now()
		var totalAmt float64
		if err := tx.QueryRowContext(ctx,
			`SELECT SUM(ol_amount) FROM order_line WHERE ol_w_id=$1 AND ol_d_id=$2 AND ol_o_id=$3`,
			warehouseID, dID, oID).Scan(&totalAmt); err != nil {
			metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)
		totalSelects++

		// UPDATE customer
		updateStart = time.Now()
		if _, err := tx.ExecContext(ctx,
			`UPDATE customer SET c_balance=c_balance+$1, c_delivery_cnt=c_delivery_cnt+1 WHERE c_w_id=$2 AND c_d_id=$3 AND c_id=$4`,
			totalAmt, warehouseID, dID, cID); err != nil {
			metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, true)
			return err
		}
		metrics.RecordOperation(workerID, "update", time.Since(updateStart), 1, false)
		totalUpdates++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// stockLevelTransactionWithMetrics implements the Stock-Level transaction with enhanced metrics
func stockLevelTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID int, workerID int, metrics MetricsRecorder) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	dID := rand.Intn(10) + 1
	threshold := 10

	// SELECT next order ID
	selectStart := time.Now()
	var nextOID int
	if err := tx.QueryRowContext(ctx,
		"SELECT d_next_o_id FROM district WHERE d_w_id=$1 AND d_id=$2",
		warehouseID, dID).Scan(&nextOID); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), 1, false)

	startOID := nextOID - 20
	if startOID < 1 {
		startOID = 1
	}

	// SELECT stock level count
	selectStart = time.Now()
	query := `SELECT COUNT(DISTINCT s.s_i_id) FROM stock s
JOIN order_line ol ON s.s_w_id=ol.ol_w_id AND s.s_i_id=ol.ol_i_id
WHERE s.s_w_id=$1 AND ol.ol_d_id=$2 AND ol.ol_o_id >= $3 AND ol.ol_o_id < $4 AND s.s_quantity < $5`
	var count int
	if err := tx.QueryRowContext(ctx, query,
		warehouseID, dID, startOID, nextOID, threshold).Scan(&count); err != nil {
		metrics.RecordOperation(workerID, "select", time.Since(selectStart), 0, true)
		return err
	}
	metrics.RecordOperation(workerID, "select", time.Since(selectStart), int64(count), false)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// supplierReorderTransactionWithMetrics implements the Supplier-Reorder extension with enhanced metrics
func supplierReorderTransactionWithMetrics(ctx context.Context, db *sql.DB, warehouseID, scale int, workerID int, metrics MetricsRecorder) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	itemID := rand.Intn(100000) + 1
	suppWID := rand.Intn(scale) + 1
	qty := rand.Intn(100) + 1
	now := time.Now()

	// INSERT purchase_order
	insertStart := time.Now()
	var poID int
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO purchase_order (po_w_id,po_i_id,po_supp_w_id,po_qty,po_order_date) VALUES ($1,$2,$3,$4,$5) RETURNING po_id`,
		warehouseID, itemID, suppWID, qty, now).Scan(&poID); err != nil {
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)

	// INSERT goods_receipt
	insertStart = time.Now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO goods_receipt (gr_po_id,gr_receipt_date) VALUES ($1,$2)`,
		poID, now); err != nil {
		metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, true)
		return err
	}
	metrics.RecordOperation(workerID, "insert", time.Since(insertStart), 1, false)

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
func newOrderTransaction(ctx context.Context, db *sql.DB, warehouseID int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	dID := rand.Intn(10) + 1
	var oID int
	// Get next order ID atomically using sequence-based function
	if err := tx.QueryRowContext(ctx,
		"SELECT get_next_order_id($1, $2)",
		warehouseID, dID).Scan(&oID); err != nil {
		tx.Rollback()
		return err
	}
	cID := rand.Intn(3000) + 1
	entryD := time.Now()
	olCnt := rand.Intn(11) + 5
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO "order" (o_w_id,o_d_id,o_id,o_c_id,o_entry_d,o_ol_cnt,o_all_local)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		warehouseID, dID, oID, cID, entryD, olCnt, 1); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO new_order (no_w_id,no_d_id,no_o_id) VALUES ($1,$2,$3)",
		warehouseID, dID, oID); err != nil {
		tx.Rollback()
		return err
	}
	for lineNum := 1; lineNum <= olCnt; lineNum++ {
		itemID := rand.Intn(100000) + 1
		qty := rand.Intn(10) + 1
		var price float64
		if err := tx.QueryRowContext(ctx,
			"SELECT i_price FROM item WHERE i_id=$1", itemID).Scan(&price); err != nil {
			tx.Rollback()
			return err
		}
		amount := price * float64(qty)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO order_line (ol_w_id,ol_d_id,ol_o_id,ol_number,ol_i_id,ol_supply_w_id,ol_delivery_d,ol_quantity,ol_amount,ol_dist_info)
			VALUES ($1,$2,$3,$4,$5,$6,NULL,$7,$8,$9)`,
			warehouseID, dID, oID, lineNum, itemID, warehouseID, qty, amount, "DIST"); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE stock SET s_quantity=s_quantity-$1, s_ytd=s_ytd+$1, s_order_cnt=s_order_cnt+1
			WHERE s_w_id=$2 AND s_i_id=$3`,
			qty, warehouseID, itemID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// paymentTransaction implements the Payment transaction
func paymentTransaction(ctx context.Context, db *sql.DB, warehouseID int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	dID := rand.Intn(10) + 1
	amount := rand.Float64() * 5000.0
	if _, err := tx.ExecContext(ctx,
		"UPDATE warehouse SET w_ytd = w_ytd + $1 WHERE w_id=$2", amount, warehouseID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE district SET d_ytd = d_ytd + $1 WHERE d_w_id=$2 AND d_id=$3", amount, warehouseID, dID); err != nil {
		tx.Rollback()
		return err
	}
	cID := rand.Intn(3000) + 1
	if _, err := tx.ExecContext(ctx,
		`UPDATE customer SET c_balance = c_balance - $1, c_ytd_payment = c_ytd_payment + $1, c_payment_cnt = c_payment_cnt + 1
		 WHERE c_w_id=$2 AND c_d_id=$3 AND c_id=$4`,
		amount, warehouseID, dID, cID); err != nil {
		tx.Rollback()
		return err
	}
	var wName, dName string
	if err := tx.QueryRowContext(ctx,
		"SELECT w_name FROM warehouse WHERE w_id=$1", warehouseID).Scan(&wName); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.QueryRowContext(ctx,
		"SELECT d_name FROM district WHERE d_w_id=$1 AND d_id=$2", warehouseID, dID).Scan(&dName); err != nil {
		tx.Rollback()
		return err
	}
	hData := fmt.Sprintf("%s    %s", wName, dName)
	now := time.Now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO history (h_c_id,h_c_d_id,h_c_w_id,h_d_id,h_w_id,h_date,h_amount,h_data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		cID, dID, warehouseID, dID, warehouseID, now, amount, hData); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// orderStatusTransaction implements the Order-Status transaction
func orderStatusTransaction(ctx context.Context, db *sql.DB, warehouseID int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	dID := rand.Intn(10) + 1
	cID := rand.Intn(3000) + 1
	var cBalance float64
	if err := tx.QueryRowContext(ctx,
		"SELECT c_balance FROM customer WHERE c_w_id=$1 AND c_d_id=$2 AND c_id=$3",
		warehouseID, dID, cID).Scan(&cBalance); err != nil {
		tx.Rollback()
		return err
	}
	var oID int
	var oEntryD time.Time
	var oCarrierID sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT o_id, o_entry_d, o_carrier_id FROM "order" WHERE o_w_id=$1 AND o_d_id=$2 AND o_c_id=$3 ORDER BY o_entry_d DESC LIMIT 1`,
		warehouseID, dID, cID).Scan(&oID, &oEntryD, &oCarrierID); err != nil {
		tx.Rollback()
		return err
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT ol_i_id, ol_supply_w_id, ol_quantity, ol_amount, ol_delivery_d FROM order_line WHERE ol_w_id=$1 AND ol_d_id=$2 AND ol_o_id=$3`,
		warehouseID, dID, oID)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var itemID, supplyWID, qty int
		var amount float64
		var deliveryDate sql.NullTime
		if err := rows.Scan(&itemID, &supplyWID, &qty, &amount, &deliveryDate); err != nil {
			rows.Close()
			tx.Rollback()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// deliveryTransaction implements the Delivery transaction
func deliveryTransaction(ctx context.Context, db *sql.DB, warehouseID int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	carrierID := rand.Intn(10) + 1
	now := time.Now()
	for dID := 1; dID <= 10; dID++ {
		var oID int
		// Use row-level locking to prevent concurrent access
		if err := tx.QueryRowContext(ctx,
			`SELECT no_o_id FROM new_order WHERE no_w_id=$1 AND no_d_id=$2 ORDER BY no_o_id FOR UPDATE SKIP LOCKED LIMIT 1`,
			warehouseID, dID).Scan(&oID); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM new_order WHERE no_w_id=$1 AND no_d_id=$2 AND no_o_id=$3`,
			warehouseID, dID, oID); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE "order" SET o_carrier_id=$1 WHERE o_w_id=$2 AND o_d_id=$3 AND o_id=$4`,
			carrierID, warehouseID, dID, oID); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE order_line SET ol_delivery_d=$1 WHERE ol_w_id=$2 AND ol_d_id=$3 AND ol_o_id=$4`,
			now, warehouseID, dID, oID); err != nil {
			tx.Rollback()
			return err
		}
		var cID int
		if err := tx.QueryRowContext(ctx,
			`SELECT o_c_id FROM "order" WHERE o_w_id=$1 AND o_d_id=$2 AND o_id=$3`,
			warehouseID, dID, oID).Scan(&cID); err != nil {
			tx.Rollback()
			return err
		}
		var totalAmt float64
		if err := tx.QueryRowContext(ctx,
			`SELECT SUM(ol_amount) FROM order_line WHERE ol_w_id=$1 AND ol_d_id=$2 AND ol_o_id=$3`,
			warehouseID, dID, oID).Scan(&totalAmt); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE customer SET c_balance=c_balance+$1, c_delivery_cnt=c_delivery_cnt+1 WHERE c_w_id=$2 AND c_d_id=$3 AND c_id=$4`,
			totalAmt, warehouseID, dID, cID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// stockLevelTransaction implements the Stock-Level transaction
func stockLevelTransaction(ctx context.Context, db *sql.DB, warehouseID int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	dID := rand.Intn(10) + 1
	threshold := 10
	var nextOID int
	if err := tx.QueryRowContext(ctx,
		"SELECT d_next_o_id FROM district WHERE d_w_id=$1 AND d_id=$2",
		warehouseID, dID).Scan(&nextOID); err != nil {
		tx.Rollback()
		return err
	}
	startOID := nextOID - 20
	if startOID < 1 {
		startOID = 1
	}
	query := `SELECT COUNT(DISTINCT s.s_i_id) FROM stock s
JOIN order_line ol ON s.s_w_id=ol.ol_w_id AND s.s_i_id=ol.ol_i_id
WHERE s.s_w_id=$1 AND ol.ol_d_id=$2 AND ol.ol_o_id >= $3 AND ol.ol_o_id < $4 AND s.s_quantity < $5`
	if err := tx.QueryRowContext(ctx, query,
		warehouseID, dID, startOID, nextOID, threshold).Scan(new(int)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// supplierReorderTransaction implements the Supplier-Reorder extension
func supplierReorderTransaction(ctx context.Context, db *sql.DB, warehouseID, scale int) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	itemID := rand.Intn(100000) + 1
	suppWID := rand.Intn(scale) + 1
	qty := rand.Intn(100) + 1
	now := time.Now()
	var poID int
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO purchase_order (po_w_id,po_i_id,po_supp_w_id,po_qty,po_order_date) VALUES ($1,$2,$3,$4,$5) RETURNING po_id`,
		warehouseID, itemID, suppWID, qty, now).Scan(&poID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO goods_receipt (gr_po_id,gr_receipt_date) VALUES ($1,$2)`,
		poID, now); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
