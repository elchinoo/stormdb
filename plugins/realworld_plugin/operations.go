// internal/workload/realworld/operations.go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// READ OPERATIONS - Mix of indexed and non-indexed queries

// executeReadOperation performs various read operations
func (w *RealWorldWorkload) executeReadOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) (string, error) {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		// Simple indexed queries (40% of reads)
		w.getUserByEmail,        // Uses unique index on email
		w.getProductBySKU,       // Uses unique index on SKU
		w.getProductsByCategory, // Uses index on category
		w.getUserOrders,         // Uses index on user_id
		w.getProductReviews,     // Uses index on product_id

		// Complex joins (30% of reads)
		w.getOrderDetailsWithItems, // Multi-table join with indexes
		w.getUserActivitySummary,   // Complex join across multiple tables
		w.getProductAnalytics,      // Join with analytics data

		// Full table scans / non-indexed queries (20% of reads)
		w.searchProductsByName, // Full-text search
		w.findSimilarUsers,     // Complex query without good indexes
		w.getRecentActivity,    // Date range query possibly without index

		// Window functions and CTEs (10% of reads)
		w.getTopProductsByCategory, // Window functions for ranking
		w.getUserSpendingTrends,    // CTE with window functions
		w.getInventoryAnalysis,     // Complex CTE analysis
	}

	op := operations[rng.Intn(len(operations))]
	opName := ""

	switch rng.Intn(14) {
	case 0:
		opName = "get_user_by_email"
	case 1:
		opName = "get_product_by_sku"
	case 2:
		opName = "get_products_by_category"
	case 3:
		opName = "get_user_orders"
	case 4:
		opName = "get_product_reviews"
	case 5:
		opName = "get_order_details_with_items"
	case 6:
		opName = "get_user_activity_summary"
	case 7:
		opName = "get_product_analytics"
	case 8:
		opName = "search_products_by_name"
	case 9:
		opName = "find_similar_users"
	case 10:
		opName = "get_recent_activity"
	case 11:
		opName = "get_top_products_by_category"
	case 12:
		opName = "get_user_spending_trends"
	case 13:
		opName = "get_inventory_analysis"
	}

	return opName, op(ctx, db, rng)
}

// Simple indexed queries

// getUserByEmail retrieves user by email (uses unique index)
func (w *RealWorldWorkload) getUserByEmail(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	email := fmt.Sprintf("user%d@example.com", userID)

	var firstName, lastName, country string
	var loyaltyPoints int

	err := db.QueryRow(ctx, `
		SELECT first_name, last_name, country, loyalty_points
		FROM users 
		WHERE email = $1`,
		email).Scan(&firstName, &lastName, &country, &loyaltyPoints)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// getProductBySKU retrieves product by SKU (uses unique index)
func (w *RealWorldWorkload) getProductBySKU(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1
	sku := fmt.Sprintf("SKU-%06d", productID)

	var name, brand, category string
	var price float64
	var avgRating *float64

	err := db.QueryRow(ctx, `
		SELECT name, brand, category, price, avg_rating
		FROM products 
		WHERE sku = $1`,
		sku).Scan(&name, &brand, &category, &price, &avgRating)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// getProductsByCategory retrieves products by category (uses index)
func (w *RealWorldWorkload) getProductsByCategory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	categories := []string{"Electronics", "Books", "Clothing", "Home & Garden", "Sports", "Beauty"}
	category := categories[rng.Intn(len(categories))]

	rows, err := db.Query(ctx, `
		SELECT product_id, name, brand, price, avg_rating
		FROM products 
		WHERE category = $1 AND is_active = true
		ORDER BY avg_rating DESC
		LIMIT 20`,
		category)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var productID int
		var name, brand string
		var price float64
		var avgRating *float64
		if err := rows.Scan(&productID, &name, &brand, &price, &avgRating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Complex join queries

// getOrderDetailsWithItems retrieves order with all items (multi-table join)
func (w *RealWorldWorkload) getOrderDetailsWithItems(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	orderID := rng.Intn(2000) + 1

	rows, err := db.Query(ctx, `
		SELECT 
			o.order_number,
			o.status,
			o.total_amount,
			o.created_at,
			u.first_name,
			u.last_name,
			u.email,
			oi.quantity,
			oi.unit_price,
			p.name,
			p.sku
		FROM orders o
		JOIN users u ON o.user_id = u.user_id
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		WHERE o.order_id = $1`,
		orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderNumber, status, firstName, lastName, email, productName, sku string
		var totalAmount, unitPrice float64
		var quantity int
		var createdAt time.Time
		if err := rows.Scan(&orderNumber, &status, &totalAmount, &createdAt, &firstName, &lastName, &email, &quantity, &unitPrice, &productName, &sku); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getUserActivitySummary gets comprehensive user activity (complex join)
func (w *RealWorldWorkload) getUserActivitySummary(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	var firstName, lastName string
	var totalSpent float64
	var orderCount, reviewCount, sessionCount int
	var lastOrderDate, lastLoginDate *time.Time

	err := db.QueryRow(ctx, `
		SELECT 
			u.first_name,
			u.last_name,
			u.total_spent,
			u.last_login,
			COALESCE(order_stats.order_count, 0) as order_count,
			COALESCE(order_stats.last_order_date, NULL) as last_order_date,
			COALESCE(review_stats.review_count, 0) as review_count,
			COALESCE(session_stats.session_count, 0) as session_count
		FROM users u
		LEFT JOIN (
			SELECT 
				user_id,
				COUNT(*) as order_count,
				MAX(created_at) as last_order_date
			FROM orders
			WHERE user_id = $1
			GROUP BY user_id
		) order_stats ON u.user_id = order_stats.user_id
		LEFT JOIN (
			SELECT 
				user_id,
				COUNT(*) as review_count
			FROM reviews
			WHERE user_id = $1
			GROUP BY user_id
		) review_stats ON u.user_id = review_stats.user_id
		LEFT JOIN (
			SELECT 
				user_id,
				COUNT(*) as session_count
			FROM user_sessions
			WHERE user_id = $1 AND started_at >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY user_id
		) session_stats ON u.user_id = session_stats.user_id
		WHERE u.user_id = $1`,
		userID).Scan(&firstName, &lastName, &totalSpent, &lastLoginDate, &orderCount, &lastOrderDate, &reviewCount, &sessionCount)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// Full table scan / non-indexed queries

// getUserOrders retrieves user's recent orders (uses index on user_id)
func (w *RealWorldWorkload) getUserOrders(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		SELECT order_id, order_number, status, total_amount, created_at
		FROM orders 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 10`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int
		var orderNumber, status string
		var totalAmount float64
		var createdAt time.Time
		if err := rows.Scan(&orderID, &orderNumber, &status, &totalAmount, &createdAt); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getProductReviews retrieves reviews for a product (uses index on product_id)
func (w *RealWorldWorkload) getProductReviews(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	rows, err := db.Query(ctx, `
		SELECT 
			r.rating,
			r.title,
			r.content,
			r.helpful_votes,
			r.created_at,
			u.first_name,
			u.last_name
		FROM reviews r
		JOIN users u ON r.user_id = u.user_id
		WHERE r.product_id = $1
		ORDER BY r.helpful_votes DESC, r.created_at DESC
		LIMIT 15`,
		productID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rating, helpfulVotes int
		var title, content, firstName, lastName string
		var createdAt time.Time
		if err := rows.Scan(&rating, &title, &content, &helpfulVotes, &createdAt, &firstName, &lastName); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getProductAnalytics retrieves product analytics data (join with analytics data)
func (w *RealWorldWorkload) getProductAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	rows, err := db.Query(ctx, `
		SELECT 
			pa.event_type,
			COUNT(*) as event_count,
			COUNT(DISTINCT pa.user_id) as unique_users,
			DATE_TRUNC('day', pa.created_at) as event_date
		FROM product_analytics pa
		WHERE pa.product_id = $1
		AND pa.created_at >= CURRENT_DATE - INTERVAL '30 days'
		GROUP BY pa.event_type, DATE_TRUNC('day', pa.created_at)
		ORDER BY event_date DESC, event_count DESC`,
		productID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var eventCount, uniqueUsers int
		var eventDate time.Time
		if err := rows.Scan(&eventType, &eventCount, &uniqueUsers, &eventDate); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getRecentActivity retrieves recent platform activity (date range query)
func (w *RealWorldWorkload) getRecentActivity(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	hoursBack := rng.Intn(72) + 1 // 1-72 hours back

	query := fmt.Sprintf(`
		SELECT 
			'order' as activity_type,
			o.order_id::text as activity_id,
			u.first_name || ' ' || u.last_name as user_name,
			'Order ' || o.order_number || ' for $' || o.total_amount as description,
			o.created_at
		FROM orders o
		JOIN users u ON o.user_id = u.user_id
		WHERE o.created_at >= NOW() - INTERVAL '%d hours'
		
		UNION ALL
		
		SELECT 
			'review' as activity_type,
			r.review_id::text as activity_id,
			u.first_name || ' ' || u.last_name as user_name,
			'Review for ' || p.name || ' (' || r.rating || ' stars)' as description,
			r.created_at
		FROM reviews r
		JOIN users u ON r.user_id = u.user_id
		JOIN products p ON r.product_id = p.product_id
		WHERE r.created_at >= NOW() - INTERVAL '%d hours'
		
		ORDER BY created_at DESC
		LIMIT 20`, hoursBack, hoursBack)

	rows, err := db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var activityType, activityID, userName, description string
		var createdAt time.Time
		if err := rows.Scan(&activityType, &activityID, &userName, &description, &createdAt); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getInventoryAnalysis performs complex CTE analysis on inventory
func (w *RealWorldWorkload) getInventoryAnalysis(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		WITH inventory_summary AS (
			SELECT 
				p.category,
				p.brand,
				COUNT(*) as product_count,
				SUM(i.quantity_available) as total_inventory,
				AVG(i.quantity_available) as avg_inventory,
				SUM(CASE WHEN i.quantity_available < i.reorder_level THEN 1 ELSE 0 END) as low_stock_count,
				SUM(p.price * i.quantity_available) as inventory_value
			FROM products p
			JOIN inventory i ON p.product_id = i.product_id
			WHERE p.is_active = true
			GROUP BY p.category, p.brand
		),
		category_totals AS (
			SELECT 
				category,
				SUM(inventory_value) as category_value,
				SUM(total_inventory) as category_inventory
			FROM inventory_summary
			GROUP BY category
		)
		SELECT 
			is_.category,
			is_.brand,
			is_.product_count,
			is_.total_inventory,
			is_.avg_inventory,
			is_.low_stock_count,
			is_.inventory_value,
			ROUND((is_.inventory_value / ct.category_value * 100)::numeric, 2) as category_value_percentage,
			RANK() OVER (PARTITION BY is_.category ORDER BY is_.inventory_value DESC) as brand_rank_in_category
		FROM inventory_summary is_
		JOIN category_totals ct ON is_.category = ct.category
		ORDER BY is_.category, brand_rank_in_category
		LIMIT 30`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var category, brand string
		var productCount, totalInventory, lowStockCount, brandRank int
		var avgInventory, inventoryValue, categoryValuePercentage float64
		if err := rows.Scan(&category, &brand, &productCount, &totalInventory, &avgInventory, &lowStockCount, &inventoryValue, &categoryValuePercentage, &brandRank); err != nil {
			return err
		}
	}
	return rows.Err()
}

// searchProductsByName performs full-text search on product names
func (w *RealWorldWorkload) searchProductsByName(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	searchTerms := []string{"smartphone", "laptop", "camera", "headphones", "watch", "shoes", "jacket", "table", "chair"}
	searchTerm := searchTerms[rng.Intn(len(searchTerms))]

	rows, err := db.Query(ctx, `
		SELECT product_id, name, brand, price, avg_rating
		FROM products 
		WHERE to_tsvector('english', name || ' ' || description) @@ plainto_tsquery('english', $1)
		ORDER BY avg_rating DESC
		LIMIT 15`,
		searchTerm)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var productID int
		var name, brand string
		var price float64
		var avgRating *float64
		if err := rows.Scan(&productID, &name, &brand, &price, &avgRating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// findSimilarUsers finds users with similar preferences (complex non-indexed query)
func (w *RealWorldWorkload) findSimilarUsers(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		SELECT 
			u2.user_id,
			u2.first_name,
			u2.last_name,
			u2.country,
			u2.total_spent
		FROM users u1
		JOIN users u2 ON u1.user_id != u2.user_id
		WHERE u1.user_id = $1
		AND u2.country = u1.country
		AND ABS(EXTRACT(YEAR FROM u2.date_of_birth) - EXTRACT(YEAR FROM u1.date_of_birth)) <= 5
		AND u2.account_status = 'active'
		ORDER BY ABS(u2.total_spent - u1.total_spent)
		LIMIT 10`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var similarUserID int
		var firstName, lastName, country string
		var totalSpent float64
		if err := rows.Scan(&similarUserID, &firstName, &lastName, &country, &totalSpent); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Window functions and CTE queries

// getTopProductsByCategory uses window functions for ranking
func (w *RealWorldWorkload) getTopProductsByCategory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT 
			category,
			name,
			brand,
			price,
			avg_rating,
			review_count,
			ROW_NUMBER() OVER (PARTITION BY category ORDER BY avg_rating DESC, review_count DESC) as category_rank,
			PERCENT_RANK() OVER (PARTITION BY category ORDER BY avg_rating) as rating_percentile,
			NTILE(4) OVER (PARTITION BY category ORDER BY price) as price_quartile
		FROM products
		WHERE is_active = true AND avg_rating >= 4.0
		ORDER BY category, category_rank
		LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var category, name, brand string
		var price, avgRating, ratingPercentile float64
		var reviewCount, categoryRank, priceQuartile int
		if err := rows.Scan(&category, &name, &brand, &price, &avgRating, &reviewCount, &categoryRank, &ratingPercentile, &priceQuartile); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getUserSpendingTrends uses CTEs with window functions for trend analysis
func (w *RealWorldWorkload) getUserSpendingTrends(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		WITH monthly_spending AS (
			SELECT 
				DATE_TRUNC('month', o.created_at) as month,
				SUM(o.total_amount) as monthly_total,
				COUNT(*) as order_count,
				AVG(o.total_amount) as avg_order_value
			FROM orders o
			WHERE o.user_id = $1 
			AND o.created_at >= CURRENT_DATE - INTERVAL '12 months'
			AND o.status IN ('completed', 'shipped', 'delivered')
			GROUP BY DATE_TRUNC('month', o.created_at)
		)
		SELECT 
			month,
			monthly_total,
			order_count,
			avg_order_value,
			LAG(monthly_total) OVER (ORDER BY month) as prev_month_total,
			SUM(monthly_total) OVER (ORDER BY month) as cumulative_spending,
			AVG(monthly_total) OVER (ORDER BY month ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) as rolling_3month_avg
		FROM monthly_spending
		ORDER BY month`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var month time.Time
		var monthlyTotal, avgOrderValue, cumulativeSpending, rolling3MonthAvg float64
		var prevMonthTotal *float64
		var orderCount int
		if err := rows.Scan(&month, &monthlyTotal, &orderCount, &avgOrderValue, &prevMonthTotal, &cumulativeSpending, &rolling3MonthAvg); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Write Operations for OLTP workloads

// executeWriteOperation performs various write operations
func (w *RealWorldWorkload) executeWriteOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.createUser,
		w.createProduct,
		w.createOrder,
		w.updateUserInfo,
		w.updateProductRating,
		w.updateInventory,
		w.createReview,
		w.logProductView,
	}

	operation := operations[rng.Intn(len(operations))]
	return operation(ctx, db, rng)
}

// executeOLTPReadOperation performs OLTP-style read operations (fast, indexed queries)
func (w *RealWorldWorkload) executeOLTPReadOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.getUserByEmail,
		w.getProductBySKU,
		w.getUserOrders,
		w.getProductReviews,
		w.getProductsByCategory,
	}

	operation := operations[rng.Intn(len(operations))]
	return operation(ctx, db, rng)
}

// executeOLTPWriteOperation performs OLTP-style write operations
func (w *RealWorldWorkload) executeOLTPWriteOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.createOrder,
		w.updateInventory,
		w.logProductView,
		w.updateUserInfo,
	}

	operation := operations[rng.Intn(len(operations))]
	return operation(ctx, db, rng)
}

// executeAnalyticsOperation performs analytics-style queries (complex, resource-intensive)
func (w *RealWorldWorkload) executeAnalyticsOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.getUserSpendingTrends,
		w.getTopProductsByCategory,
		w.getInventoryAnalysis,
		w.getRecentActivity,
		w.findSimilarUsers,
	}

	operation := operations[rng.Intn(len(operations))]
	return operation(ctx, db, rng)
}

// Individual write operations

// createUser creates a new user account
func (w *RealWorldWorkload) createUser(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	firstName := fmt.Sprintf("User%d", rng.Intn(10000))
	lastName := fmt.Sprintf("Test%d", rng.Intn(10000))
	email := fmt.Sprintf("user%d@example.com", rng.Intn(100000))
	countries := []string{"US", "CA", "GB", "DE", "FR", "JP", "AU"}
	country := countries[rng.Intn(len(countries))]

	_, err := db.Exec(ctx, `
		INSERT INTO users (first_name, last_name, email, country, date_of_birth, account_status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'active', NOW())`,
		firstName, lastName, email, country, time.Now().AddDate(-rng.Intn(50)-18, 0, 0))

	return err
}

// createProduct adds a new product
func (w *RealWorldWorkload) createProduct(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	categories := []string{"Electronics", "Books", "Clothing", "Home & Garden", "Sports", "Beauty"}
	brands := []string{"BrandA", "BrandB", "BrandC", "BrandD", "BrandE"}

	name := fmt.Sprintf("Product%d", rng.Intn(10000))
	category := categories[rng.Intn(len(categories))]
	brand := brands[rng.Intn(len(brands))]
	price := float64(rng.Intn(500) + 10)
	sku := fmt.Sprintf("SKU-%08d", rng.Intn(100000000))

	_, err := db.Exec(ctx, `
		INSERT INTO products (name, category, brand, price, sku, description, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW())`,
		name, category, brand, price, sku, fmt.Sprintf("Description for %s", name))

	return err
}

// createOrder creates a new order with items
func (w *RealWorldWorkload) createOrder(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userID := rng.Intn(1000) + 1
	orderNumber := fmt.Sprintf("ORD-%d-%d", time.Now().Unix(), rng.Intn(10000))

	var orderID int
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, order_number, status, total_amount, created_at)
		VALUES ($1, $2, 'pending', 0, NOW())
		RETURNING order_id`,
		userID, orderNumber).Scan(&orderID)
	if err != nil {
		return err
	}

	// Add 1-5 items to the order
	itemCount := rng.Intn(5) + 1
	totalAmount := 0.0

	for i := 0; i < itemCount; i++ {
		productID := rng.Intn(500) + 1
		quantity := rng.Intn(3) + 1
		unitPrice := float64(rng.Intn(200) + 10)
		totalPrice := float64(quantity) * unitPrice

		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, productID, quantity, unitPrice, totalPrice)
		if err != nil {
			return err
		}

		totalAmount += totalPrice
	}

	// Update order total
	_, err = tx.Exec(ctx, `
		UPDATE orders SET total_amount = $1 WHERE order_id = $2`,
		totalAmount, orderID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// updateUserInfo updates user information
func (w *RealWorldWorkload) updateUserInfo(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	spentAmount := float64(rng.Intn(1000))

	_, err := db.Exec(ctx, `
		UPDATE users 
		SET total_spent = total_spent + $1, last_login = NOW()
		WHERE user_id = $2`,
		spentAmount, userID)

	return err
}

// updateProductRating updates product average rating
func (w *RealWorldWorkload) updateProductRating(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1
	newRating := float64(rng.Intn(5)) + 1.0 + (float64(rng.Intn(10)) / 10.0)

	_, err := db.Exec(ctx, `
		UPDATE products 
		SET avg_rating = (COALESCE(avg_rating, 0) * COALESCE(review_count, 0) + $1) / (COALESCE(review_count, 0) + 1),
		    review_count = COALESCE(review_count, 0) + 1
		WHERE product_id = $2`,
		newRating, productID)

	return err
}

// updateInventory updates product inventory
func (w *RealWorldWorkload) updateInventory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1
	quantityChange := rng.Intn(20) - 10 // -10 to +9

	_, err := db.Exec(ctx, `
		UPDATE inventory 
		SET quantity_available = GREATEST(0, quantity_available + $1),
		    last_updated = NOW()
		WHERE product_id = $2`,
		quantityChange, productID)

	return err
}

// createReview creates a product review
func (w *RealWorldWorkload) createReview(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	productID := rng.Intn(500) + 1
	rating := rng.Intn(5) + 1
	titles := []string{"Great product!", "Not bad", "Amazing quality", "Could be better", "Excellent value"}
	title := titles[rng.Intn(len(titles))]

	_, err := db.Exec(ctx, `
		INSERT INTO reviews (user_id, product_id, rating, title, content, helpful_votes, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW())`,
		userID, productID, rating, title, fmt.Sprintf("Review content for product %d", productID))

	return err
}

// logProductView logs a product view event for analytics
func (w *RealWorldWorkload) logProductView(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	productID := rng.Intn(500) + 1
	eventTypes := []string{"view", "add_to_cart", "purchase", "wishlist_add"}
	eventType := eventTypes[rng.Intn(len(eventTypes))]

	_, err := db.Exec(ctx, `
		INSERT INTO product_analytics (user_id, product_id, event_type, created_at)
		VALUES ($1, $2, $3, NOW())`,
		userID, productID, eventType)

	return err
}
