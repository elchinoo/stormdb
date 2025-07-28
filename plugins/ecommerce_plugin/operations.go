// internal/workload/ecommerce/operations.go
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
func (w *ECommerceWorkload) executeReadOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) (string, error) {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		// Simple indexed queries (40% of reads)
		w.getUserByEmail,            // Uses unique index on email
		w.getProductBySKU,           // Uses unique index on SKU
		w.getProductsByCategory,     // Uses index on category
		w.getUserOrders,             // Uses index on user_id
		w.getProductReviews,         // Uses index on product_id
		w.getVendorProducts,         // Uses index on vendor_id
		w.getPurchaseOrdersByVendor, // Uses index on vendor_id

		// Complex joins (30% of reads)
		w.getOrderDetailsWithItems, // Multi-table join with indexes
		w.getUserActivitySummary,   // Complex join across multiple tables
		w.getProductAnalytics,      // Join with analytics data
		w.getInventoryStatus,       // Join inventory with products and vendors
		w.getVendorPerformance,     // Complex vendor analysis

		// Full table scans / non-indexed queries (20% of reads)
		w.searchProductsByName,  // Full-text search
		w.findSimilarUsers,      // Complex query without good indexes
		w.getRecentActivity,     // Date range query possibly without index
		w.searchReviewsByVector, // Vector similarity search using pgvector

		// Window functions and CTEs (10% of reads)
		w.getTopProductsByCategory, // Window functions for ranking
		w.getUserSpendingTrends,    // CTE with window functions
		w.getInventoryAnalysis,     // Complex CTE analysis
		w.getStockControlReport,    // Stock control analytics
	}

	op := operations[rng.Intn(len(operations))]
	opName := ""

	switch rng.Intn(20) {
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
		opName = "get_vendor_products"
	case 6:
		opName = "get_purchase_orders_by_vendor"
	case 7:
		opName = "get_order_details_with_items"
	case 8:
		opName = "get_user_activity_summary"
	case 9:
		opName = "get_product_analytics"
	case 10:
		opName = "get_inventory_status"
	case 11:
		opName = "get_vendor_performance"
	case 12:
		opName = "search_products_by_name"
	case 13:
		opName = "find_similar_users"
	case 14:
		opName = "get_recent_activity"
	case 15:
		opName = "search_reviews_by_vector"
	case 16:
		opName = "get_top_products_by_category"
	case 17:
		opName = "get_user_spending_trends"
	case 18:
		opName = "get_inventory_analysis"
	case 19:
		opName = "get_stock_control_report"
	}

	return opName, op(ctx, db, rng)
}

// Simple indexed queries

// getUserByEmail retrieves user by email (uses unique index)
func (w *ECommerceWorkload) getUserByEmail(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
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
func (w *ECommerceWorkload) getProductBySKU(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1
	sku := fmt.Sprintf("SKU-%06d", productID)

	var name, category, brand string
	var price float64

	err := db.QueryRow(ctx, `
		SELECT name, category, brand, price
		FROM products 
		WHERE sku = $1`,
		sku).Scan(&name, &category, &brand, &price)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// getProductsByCategory retrieves products by category (uses index)
func (w *ECommerceWorkload) getProductsByCategory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	categories := []string{"Electronics", "Clothing", "Books", "Home", "Sports", "Beauty", "Toys"}
	category := categories[rng.Intn(len(categories))]

	rows, err := db.Query(ctx, `
		SELECT product_id, name, price, avg_rating
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
		var name string
		var price, rating float64
		if err := rows.Scan(&productID, &name, &price, &rating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getUserOrders retrieves user's orders (uses index)
func (w *ECommerceWorkload) getUserOrders(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
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

// getProductReviews retrieves product reviews (uses index)
func (w *ECommerceWorkload) getProductReviews(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	rows, err := db.Query(ctx, `
		SELECT r.review_id, r.rating, r.title, r.content, r.helpful_votes, u.username
		FROM reviews r
		JOIN users u ON r.user_id = u.user_id
		WHERE r.product_id = $1
		ORDER BY r.helpful_votes DESC, r.created_at DESC
		LIMIT 20`,
		productID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewID, rating, helpfulVotes int
		var title, content, username string
		if err := rows.Scan(&reviewID, &rating, &title, &content, &helpfulVotes, &username); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getVendorProducts retrieves products from a specific vendor
func (w *ECommerceWorkload) getVendorProducts(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	vendorID := rng.Intn(50) + 1

	rows, err := db.Query(ctx, `
		SELECT p.product_id, p.name, p.price, p.cost, v.vendor_name
		FROM products p
		JOIN vendors v ON p.vendor_id = v.vendor_id
		WHERE p.vendor_id = $1 AND p.is_active = true
		ORDER BY p.name
		LIMIT 50`,
		vendorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var productID int
		var name, vendorName string
		var price, cost float64
		if err := rows.Scan(&productID, &name, &price, &cost, &vendorName); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getPurchaseOrdersByVendor retrieves purchase orders for a vendor
func (w *ECommerceWorkload) getPurchaseOrdersByVendor(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	vendorID := rng.Intn(50) + 1

	rows, err := db.Query(ctx, `
		SELECT po_id, po_number, status, total_amount, expected_delivery, created_at
		FROM purchase_orders
		WHERE vendor_id = $1
		ORDER BY created_at DESC
		LIMIT 20`,
		vendorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var poID int
		var poNumber, status string
		var totalAmount float64
		var expectedDelivery time.Time
		var createdAt time.Time
		if err := rows.Scan(&poID, &poNumber, &status, &totalAmount, &expectedDelivery, &createdAt); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Complex joins

// getOrderDetailsWithItems retrieves order with all items (multi-table join)
func (w *ECommerceWorkload) getOrderDetailsWithItems(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	orderID := rng.Intn(2000) + 1

	rows, err := db.Query(ctx, `
		SELECT o.order_number, o.status, o.total_amount, 
		       oi.quantity, oi.unit_price, p.name AS product_name,
		       u.username, u.email
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		JOIN users u ON o.user_id = u.user_id
		WHERE o.order_id = $1`,
		orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderNumber, status, productName, username, email string
		var totalAmount, unitPrice float64
		var quantity int
		if err := rows.Scan(&orderNumber, &status, &totalAmount, &quantity, &unitPrice, &productName, &username, &email); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getUserActivitySummary retrieves comprehensive user activity (complex join)
func (w *ECommerceWorkload) getUserActivitySummary(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	var username, email string
	var totalSpent, avgOrderValue float64
	var orderCount, reviewCount int

	err := db.QueryRow(ctx, `
		SELECT u.username, u.email, u.total_spent,
		       COUNT(DISTINCT o.order_id) AS order_count,
		       COUNT(DISTINCT r.review_id) AS review_count,
		       COALESCE(AVG(o.total_amount), 0) AS avg_order_value
		FROM users u
		LEFT JOIN orders o ON u.user_id = o.user_id
		LEFT JOIN reviews r ON u.user_id = r.user_id
		WHERE u.user_id = $1
		GROUP BY u.user_id, u.username, u.email, u.total_spent`,
		userID).Scan(&username, &email, &totalSpent, &orderCount, &reviewCount, &avgOrderValue)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// getProductAnalytics retrieves product interaction analytics
func (w *ECommerceWorkload) getProductAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	rows, err := db.Query(ctx, `
		SELECT pa.event_type, COUNT(*) AS event_count,
		       DATE_TRUNC('day', pa.created_at) AS event_date
		FROM product_analytics pa
		WHERE pa.product_id = $1
		  AND pa.created_at > NOW() - INTERVAL '30 days'
		GROUP BY pa.event_type, DATE_TRUNC('day', pa.created_at)
		ORDER BY event_date DESC, event_count DESC`,
		productID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var eventCount int
		var eventDate time.Time
		if err := rows.Scan(&eventType, &eventCount, &eventDate); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getInventoryStatus retrieves inventory status with product and vendor info
func (w *ECommerceWorkload) getInventoryStatus(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT p.name, p.sku, i.quantity_available, i.reorder_level,
		       v.vendor_name, i.warehouse_location,
		       CASE WHEN i.quantity_available <= i.reorder_level THEN 'LOW' ELSE 'OK' END AS status
		FROM inventory i
		JOIN products p ON i.product_id = p.product_id
		LEFT JOIN vendors v ON i.supplier_id = v.vendor_id
		WHERE i.quantity_available <= i.reorder_level * 2
		ORDER BY i.quantity_available ASC
		LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, sku, vendorName, warehouseLocation, status string
		var quantityAvailable, reorderLevel int
		if err := rows.Scan(&name, &sku, &quantityAvailable, &reorderLevel, &vendorName, &warehouseLocation, &status); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getVendorPerformance analyzes vendor performance
func (w *ECommerceWorkload) getVendorPerformance(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT v.vendor_name, v.rating,
		       COUNT(DISTINCT po.po_id) AS total_orders,
		       SUM(po.total_amount) AS total_value,
		       COALESCE(AVG(po.actual_delivery - po.expected_delivery), 0) AS avg_delay_days
		FROM vendors v
		LEFT JOIN purchase_orders po ON v.vendor_id = po.vendor_id
		WHERE po.created_at > NOW() - INTERVAL '90 days'
		GROUP BY v.vendor_id, v.vendor_name, v.rating
		HAVING COUNT(DISTINCT po.po_id) > 0
		ORDER BY v.rating DESC, total_value DESC
		LIMIT 20`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var vendorName string
		var rating, totalValue, avgDelayDays float64
		var totalOrders int
		if err := rows.Scan(&vendorName, &rating, &totalOrders, &totalValue, &avgDelayDays); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Full table scans / non-indexed queries

// searchProductsByName performs text search on product names
func (w *ECommerceWorkload) searchProductsByName(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	searchTerms := []string{"laptop", "phone", "book", "shirt", "shoe", "watch", "camera", "tablet"}
	searchTerm := searchTerms[rng.Intn(len(searchTerms))]

	rows, err := db.Query(ctx, `
		SELECT product_id, name, price, avg_rating
		FROM products
		WHERE name ILIKE '%' || $1 || '%' AND is_active = true
		ORDER BY avg_rating DESC, view_count DESC
		LIMIT 20`,
		searchTerm)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var productID int
		var name string
		var price, avgRating float64
		if err := rows.Scan(&productID, &name, &price, &avgRating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// findSimilarUsers finds users with similar purchasing patterns
func (w *ECommerceWorkload) findSimilarUsers(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		WITH user_categories AS (
			SELECT DISTINCT p.category
			FROM orders o
			JOIN order_items oi ON o.order_id = oi.order_id
			JOIN products p ON oi.product_id = p.product_id
			WHERE o.user_id = $1
		)
		SELECT DISTINCT u.user_id, u.username, u.country,
		       COUNT(DISTINCT p.category) AS common_categories
		FROM users u
		JOIN orders o ON u.user_id = o.user_id
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		WHERE u.user_id != $1
		  AND p.category IN (SELECT category FROM user_categories)
		GROUP BY u.user_id, u.username, u.country
		HAVING COUNT(DISTINCT p.category) >= 2
		ORDER BY common_categories DESC
		LIMIT 10`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var similarUserID, commonCategories int
		var username, country string
		if err := rows.Scan(&similarUserID, &username, &country, &commonCategories); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getRecentActivity retrieves recent system activity
func (w *ECommerceWorkload) getRecentActivity(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT 'order' AS activity_type, created_at, user_id, total_amount::text AS details
		FROM orders
		WHERE created_at > NOW() - INTERVAL '24 hours'
		UNION ALL
		SELECT 'review' AS activity_type, created_at, user_id, rating::text AS details
		FROM reviews
		WHERE created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var activityType, details string
		var userID int
		var createdAt time.Time
		if err := rows.Scan(&activityType, &createdAt, &userID, &details); err != nil {
			return err
		}
	}
	return rows.Err()
}

// searchReviewsByVector performs vector similarity search on review content
func (w *ECommerceWorkload) searchReviewsByVector(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Generate a random 1536-dimensional vector for similarity search
	vector := make([]float32, 1536)
	for i := range vector {
		vector[i] = rand.Float32()*2 - 1 // Random values between -1 and 1
	}

	vectorStr := "["
	for j, v := range vector {
		if j > 0 {
			vectorStr += ","
		}
		vectorStr += fmt.Sprintf("%.6f", v)
	}
	vectorStr += "]"

	// Try vector search first
	rows, err := db.Query(ctx, `
		SELECT r.review_id, r.title, r.content, r.rating,
		       r.content_vector <-> $1::vector AS distance
		FROM reviews r
		WHERE r.content_vector IS NOT NULL
		ORDER BY r.content_vector <-> $1::vector
		LIMIT 10`, vectorStr)

	if err != nil {
		// Fallback to text search if vector search fails
		searchTerms := []string{"great", "good", "excellent", "poor", "bad", "amazing", "terrible", "fantastic"}
		searchTerm := searchTerms[rng.Intn(len(searchTerms))]

		rows, err = db.Query(ctx, `
			SELECT review_id, title, content, rating, 0.0 AS distance
			FROM reviews
			WHERE content ILIKE '%' || $1 || '%'
			ORDER BY helpful_votes DESC
			LIMIT 10`,
			searchTerm)
		if err != nil {
			return err
		}
	}
	defer rows.Close()

	for rows.Next() {
		var reviewID, rating int
		var title, content string
		var distance float64
		if err := rows.Scan(&reviewID, &title, &content, &rating, &distance); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Window functions and CTEs

// getTopProductsByCategory uses window functions for ranking
func (w *ECommerceWorkload) getTopProductsByCategory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT category, name, price, avg_rating, rank
		FROM (
			SELECT category, name, price, avg_rating,
			       ROW_NUMBER() OVER (PARTITION BY category ORDER BY avg_rating DESC, review_count DESC) AS rank
			FROM products
			WHERE is_active = true
		) ranked_products
		WHERE rank <= 3
		ORDER BY category, rank`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var category, name string
		var price, avgRating float64
		var rank int
		if err := rows.Scan(&category, &name, &price, &avgRating, &rank); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getUserSpendingTrends analyzes user spending patterns with CTEs
func (w *ECommerceWorkload) getUserSpendingTrends(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		WITH monthly_spending AS (
			SELECT DATE_TRUNC('month', created_at) AS month,
			       SUM(total_amount) AS monthly_total
			FROM orders
			WHERE user_id = $1
			GROUP BY DATE_TRUNC('month', created_at)
		),
		spending_with_trend AS (
			SELECT month, monthly_total,
			       LAG(monthly_total) OVER (ORDER BY month) AS prev_month_total,
			       CASE 
			         WHEN LAG(monthly_total) OVER (ORDER BY month) IS NULL THEN 0
			         ELSE ((monthly_total - LAG(monthly_total) OVER (ORDER BY month)) / LAG(monthly_total) OVER (ORDER BY month)) * 100
			       END AS growth_rate
			FROM monthly_spending
		)
		SELECT month, monthly_total, prev_month_total, growth_rate
		FROM spending_with_trend
		ORDER BY month DESC
		LIMIT 12`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var month time.Time
		var monthlyTotal, growthRate float64
		var prevMonthTotal *float64
		if err := rows.Scan(&month, &monthlyTotal, &prevMonthTotal, &growthRate); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getInventoryAnalysis performs complex inventory analysis with CTEs
func (w *ECommerceWorkload) getInventoryAnalysis(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		WITH inventory_status AS (
			SELECT i.product_id, p.name, p.category, i.quantity_available, 
			       i.reorder_level, i.max_stock_level,
			       CASE 
			         WHEN i.quantity_available <= i.reorder_level THEN 'LOW'
			         WHEN i.quantity_available >= i.max_stock_level * 0.8 THEN 'HIGH'
			         ELSE 'NORMAL'
			       END AS stock_status
			FROM inventory i
			JOIN products p ON i.product_id = p.product_id
		),
		category_summary AS (
			SELECT category, stock_status, COUNT(*) AS product_count
			FROM inventory_status
			GROUP BY category, stock_status
		)
		SELECT category, stock_status, product_count,
		       SUM(product_count) OVER (PARTITION BY category) AS total_in_category,
		       ROUND((product_count::numeric / SUM(product_count) OVER (PARTITION BY category)) * 100, 2) AS percentage
		FROM category_summary
		ORDER BY category, stock_status`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var category, stockStatus string
		var productCount, totalInCategory int
		var percentage float64
		if err := rows.Scan(&category, &stockStatus, &productCount, &totalInCategory, &percentage); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getStockControlReport generates stock control and purchase order analytics
func (w *ECommerceWorkload) getStockControlReport(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		WITH stock_alerts AS (
			SELECT i.product_id, p.name, p.sku, i.quantity_available, 
			       i.reorder_level, v.vendor_name,
			       CASE WHEN i.quantity_available <= i.reorder_level THEN 'REORDER_NEEDED' ELSE 'OK' END AS alert_status
			FROM inventory i
			JOIN products p ON i.product_id = p.product_id
			LEFT JOIN vendors v ON i.supplier_id = v.vendor_id
			WHERE i.auto_reorder = true
		),
		pending_pos AS (
			SELECT po.vendor_id, COUNT(*) AS pending_orders, SUM(po.total_amount) AS pending_value
			FROM purchase_orders po
			WHERE po.status = 'pending'
			GROUP BY po.vendor_id
		)
		SELECT sa.name, sa.sku, sa.quantity_available, sa.reorder_level,
		       sa.vendor_name, sa.alert_status,
		       COALESCE(pp.pending_orders, 0) AS pending_orders,
		       COALESCE(pp.pending_value, 0) AS pending_value
		FROM stock_alerts sa
		LEFT JOIN pending_pos pp ON sa.product_id IN (
			SELECT poi.product_id FROM purchase_order_items poi 
			JOIN purchase_orders po ON poi.po_id = po.po_id 
			WHERE po.vendor_id = pp.vendor_id AND po.status = 'pending'
		)
		WHERE sa.alert_status = 'REORDER_NEEDED'
		ORDER BY sa.quantity_available ASC
		LIMIT 20`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, sku, vendorName, alertStatus string
		var quantityAvailable, reorderLevel, pendingOrders int
		var pendingValue float64
		if err := rows.Scan(&name, &sku, &quantityAvailable, &reorderLevel, &vendorName, &alertStatus, &pendingOrders, &pendingValue); err != nil {
			return err
		}
	}
	return rows.Err()
}

// WRITE OPERATIONS

// executeWriteOperation performs various write operations
func (w *ECommerceWorkload) executeWriteOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// All write operations enabled - po_id ambiguity fixed
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.createNewOrder,         // Insert new order and order items
		w.insertProductReview,    // Insert new product review
		w.updateInventory,        // Update stock levels
		w.updateUserProfile,      // Update user information
		w.updateProductPricing,   // Update product prices
		w.insertProductAnalytics, // Track product interactions
		w.updateVendorRating,     // Update vendor performance rating
		w.processOrderShipment,   // Update order status to shipped
		w.createPurchaseOrder,    // Create purchase order to vendor
		w.receivePurchaseOrder,   // Process purchase order receipt (FIXED)
	}

	op := operations[rng.Intn(len(operations))]
	return op(ctx, db, rng)
}

// createNewOrder creates a new customer order
func (w *ECommerceWorkload) createNewOrder(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	orderNumber := fmt.Sprintf("ORD-%d-%d-%d", time.Now().UnixNano(), rng.Intn(1000000), userID)

	// Start transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create order
	var orderID int
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, order_number, total_amount, payment_method, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING order_id`,
		userID, orderNumber, 99.99, "credit_card").Scan(&orderID)
	if err != nil {
		return err
	}

	// Add order items
	numItems := rng.Intn(3) + 1
	for i := 0; i < numItems; i++ {
		productID := rng.Intn(500) + 1
		quantity := rng.Intn(3) + 1
		unitPrice := float64(rng.Intn(100) + 10)
		totalPrice := float64(quantity) * unitPrice

		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, productID, quantity, unitPrice, totalPrice)
		if err != nil {
			return err
		}

		// Update inventory
		_, err = tx.Exec(ctx, `
			UPDATE inventory 
			SET quantity_available = quantity_available - $1,
			    updated_at = NOW()
			WHERE product_id = $2`,
			quantity, productID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// updateInventory updates inventory levels
func (w *ECommerceWorkload) updateInventory(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1
	adjustment := rng.Intn(20) - 10 // -10 to +10

	_, err := db.Exec(ctx, `
		UPDATE inventory 
		SET quantity_available = GREATEST(quantity_available + $1, 0),
		    updated_at = NOW()
		WHERE product_id = $2`,
		adjustment, productID)
	return err
}

// insertProductReview inserts a new product review
func (w *ECommerceWorkload) insertProductReview(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	productID := rng.Intn(500) + 1
	rating := rng.Intn(5) + 1

	reviews := []string{
		"Great product, highly recommended!",
		"Good value for money.",
		"Not what I expected, but okay.",
		"Excellent quality and fast shipping.",
		"Could be better, but works fine.",
	}
	content := reviews[rng.Intn(len(reviews))]

	// Generate a fake vector (in practice, this would be generated by an embedding model)
	// For demonstration, we'll use a simple approach
	vector := make([]float32, 1536)
	for i := range vector {
		vector[i] = rand.Float32()*2 - 1 // Random values between -1 and 1
	}

	vectorStr := "["
	for i, v := range vector {
		if i > 0 {
			vectorStr += ","
		}
		vectorStr += fmt.Sprintf("%.6f", v)
	}
	vectorStr += "]"

	_, err := db.Exec(ctx, `
		INSERT INTO reviews (user_id, product_id, rating, title, content, content_vector, is_verified_purchase)
		VALUES ($1, $2, $3, $4, $5, $6::vector, $7)`,
		userID, productID, rating, "Review Title", content, vectorStr, rng.Float32() < 0.7)
	return err
}

// updateUserProfile updates user information
func (w *ECommerceWorkload) updateUserProfile(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	loyaltyPoints := rng.Intn(1000)

	_, err := db.Exec(ctx, `
		UPDATE users 
		SET loyalty_points = $1, last_login = NOW()
		WHERE user_id = $2`,
		loyaltyPoints, userID)
	return err
}

// createPurchaseOrder creates a purchase order to a vendor
func (w *ECommerceWorkload) createPurchaseOrder(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	vendorID := rng.Intn(50) + 1
	poNumber := fmt.Sprintf("PO-%d-%d-%d", time.Now().UnixNano(), rng.Intn(1000000), vendorID)

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create purchase order
	var poID int
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_orders (po_number, vendor_id, total_amount, status, expected_delivery, created_by)
		VALUES ($1, $2, $3, 'pending', $4, 'system')
		RETURNING po_id`,
		poNumber, vendorID, 1000.00, time.Now().AddDate(0, 0, 7)).Scan(&poID)
	if err != nil {
		return err
	}

	// Add purchase order items
	numItems := rng.Intn(3) + 1
	for i := 0; i < numItems; i++ {
		productID := rng.Intn(500) + 1
		quantity := rng.Intn(50) + 10
		unitCost := float64(rng.Intn(50) + 5)
		totalCost := float64(quantity) * unitCost

		_, err = tx.Exec(ctx, `
			INSERT INTO purchase_order_items (po_id, product_id, quantity_ordered, unit_cost, total_cost)
			VALUES ($1, $2, $3, $4, $5)`,
			poID, productID, quantity, unitCost, totalCost)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// receivePurchaseOrder processes receipt of purchase order items
func (w *ECommerceWorkload) receivePurchaseOrder(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Find a pending purchase order item with explicit column references
	var poItemID, quantityOrdered int
	err := db.QueryRow(ctx, `
		SELECT poi.po_item_id, poi.quantity_ordered
		FROM purchase_order_items poi
		WHERE poi.quantity_received < poi.quantity_ordered
		ORDER BY RANDOM()
		LIMIT 1`).Scan(&poItemID, &quantityOrdered)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // No pending items
		}
		return err
	}

	// Receive partial or full quantity
	quantityReceived := rng.Intn(quantityOrdered) + 1

	_, err = db.Exec(ctx, `
		UPDATE purchase_order_items
		SET quantity_received = quantity_received + $1
		WHERE po_item_id = $2`,
		quantityReceived, poItemID)

	return err
}

// updateProductPricing updates product prices
func (w *ECommerceWorkload) updateProductPricing(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	// Get current cost
	var currentCost float64
	err := db.QueryRow(ctx, `SELECT COALESCE(cost, 0) FROM products WHERE product_id = $1`, productID).Scan(&currentCost)
	if err != nil {
		return err
	}

	// Calculate new price with at least 60% margin
	if currentCost > 0 {
		newPrice := currentCost * 1.6 // 60% margin minimum

		_, err = db.Exec(ctx, `
			UPDATE products 
			SET price = $1, updated_at = NOW()
			WHERE product_id = $2`,
			newPrice, productID)
		return err
	}

	return nil
}

// insertProductAnalytics tracks product interactions
func (w *ECommerceWorkload) insertProductAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1
	productID := rng.Intn(500) + 1

	eventTypes := []string{"view", "add_to_cart", "purchase", "wishlist"}
	eventType := eventTypes[rng.Intn(len(eventTypes))]

	_, err := db.Exec(ctx, `
		INSERT INTO product_analytics (product_id, user_id, event_type)
		VALUES ($1, $2, $3)`,
		productID, userID, eventType)
	return err
}

// updateVendorRating updates vendor performance rating
func (w *ECommerceWorkload) updateVendorRating(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	vendorID := rng.Intn(50) + 1
	newRating := 1.0 + rand.Float64()*4.0 // Rating between 1.0 and 5.0

	_, err := db.Exec(ctx, `
		UPDATE vendors 
		SET rating = $1, updated_at = NOW()
		WHERE vendor_id = $2`,
		newRating, vendorID)
	return err
}

// processOrderShipment updates order status to shipped
func (w *ECommerceWorkload) processOrderShipment(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Find a pending order
	var orderID int
	err := db.QueryRow(ctx, `
		SELECT order_id
		FROM orders
		WHERE status = 'pending'
		ORDER BY RANDOM()
		LIMIT 1`).Scan(&orderID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // No pending orders
		}
		return err
	}

	_, err = db.Exec(ctx, `
		UPDATE orders 
		SET status = 'shipped', shipped_at = NOW(), updated_at = NOW()
		WHERE order_id = $1`,
		orderID)

	return err
}

// OLTP OPERATIONS

// executeOLTPReadOperation performs OLTP-style read operations
func (w *ECommerceWorkload) executeOLTPReadOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Fast, indexed lookups typical in OLTP systems
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.getUserByEmail,
		w.getProductBySKU,
		w.getUserOrders,
		w.getInventoryByProduct,
		w.getOrderDetails,
	}

	op := operations[rng.Intn(len(operations))]
	return op(ctx, db, rng)
}

// executeOLTPWriteOperation performs OLTP-style write operations
func (w *ECommerceWorkload) executeOLTPWriteOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Fast, small transactions typical in OLTP systems
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.updateInventory,
		w.updateUserProfile,
		w.insertProductAnalytics,
		w.updateOrderStatus,
	}

	op := operations[rng.Intn(len(operations))]
	return op(ctx, db, rng)
}

// getInventoryByProduct gets inventory for a specific product
func (w *ECommerceWorkload) getInventoryByProduct(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	productID := rng.Intn(500) + 1

	var quantityAvailable, reorderLevel int
	var warehouseLocation string

	err := db.QueryRow(ctx, `
		SELECT quantity_available, reorder_level, warehouse_location
		FROM inventory
		WHERE product_id = $1`,
		productID).Scan(&quantityAvailable, &reorderLevel, &warehouseLocation)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// getOrderDetails gets details for a specific order
func (w *ECommerceWorkload) getOrderDetails(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	orderID := rng.Intn(2000) + 1

	var orderNumber, status string
	var totalAmount float64
	var createdAt time.Time

	err := db.QueryRow(ctx, `
		SELECT order_number, status, total_amount, created_at
		FROM orders
		WHERE order_id = $1`,
		orderID).Scan(&orderNumber, &status, &totalAmount, &createdAt)

	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// updateOrderStatus updates order status
func (w *ECommerceWorkload) updateOrderStatus(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	orderID := rng.Intn(2000) + 1
	statuses := []string{"pending", "processing", "shipped", "delivered"}
	status := statuses[rng.Intn(len(statuses))]

	_, err := db.Exec(ctx, `
		UPDATE orders 
		SET status = $1, updated_at = NOW()
		WHERE order_id = $2`,
		status, orderID)
	return err
}

// ANALYTICS OPERATIONS

// executeAnalyticsOperation performs analytics-style operations
func (w *ECommerceWorkload) executeAnalyticsOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Complex analytical queries
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.getInventoryAnalysis,
		w.getUserSpendingTrends,
		w.getTopProductsByCategory,
		w.getVendorPerformance,
		w.getStockControlReport,
		w.getSalesAnalytics,
		w.getCustomerSegmentation,
	}

	op := operations[rng.Intn(len(operations))]
	return op(ctx, db, rng)
}

// getSalesAnalytics performs sales analytics
func (w *ECommerceWorkload) getSalesAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		SELECT DATE_TRUNC('month', o.created_at) AS month,
		       p.category,
		       COUNT(DISTINCT o.order_id) AS total_orders,
		       SUM(oi.quantity) AS total_quantity,
		       SUM(oi.total_price) AS total_revenue,
		       AVG(oi.unit_price) AS avg_unit_price
		FROM orders o
		JOIN order_items oi ON o.order_id = oi.order_id
		JOIN products p ON oi.product_id = p.product_id
		WHERE o.created_at > NOW() - INTERVAL '12 months'
		GROUP BY DATE_TRUNC('month', o.created_at), p.category
		ORDER BY month DESC, total_revenue DESC
		LIMIT 100`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var month time.Time
		var category string
		var totalOrders, totalQuantity int
		var totalRevenue, avgUnitPrice float64
		if err := rows.Scan(&month, &category, &totalOrders, &totalQuantity, &totalRevenue, &avgUnitPrice); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getCustomerSegmentation performs customer segmentation analysis
func (w *ECommerceWorkload) getCustomerSegmentation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	rows, err := db.Query(ctx, `
		WITH customer_metrics AS (
			SELECT u.user_id, u.country,
			       COUNT(DISTINCT o.order_id) AS total_orders,
			       SUM(o.total_amount) AS total_spent,
			       AVG(o.total_amount) AS avg_order_value,
			       MAX(o.created_at) AS last_order_date,
			       COALESCE(EXTRACT(EPOCH FROM NOW() - MAX(o.created_at))/86400.0, 0) AS days_since_last_order
			FROM users u
			LEFT JOIN orders o ON u.user_id = o.user_id
			GROUP BY u.user_id, u.country
		),
		customer_segments AS (
			SELECT *,
			       CASE 
			         WHEN total_orders = 0 THEN 'Inactive'
			         WHEN total_orders = 1 THEN 'New'
			         WHEN days_since_last_order <= 30 AND total_spent >= 500 THEN 'VIP'
			         WHEN days_since_last_order <= 30 THEN 'Active'
			         WHEN days_since_last_order <= 90 THEN 'At Risk'
			         ELSE 'Churned'
			       END AS segment
			FROM customer_metrics
		)
		SELECT segment, country, COUNT(*) AS customer_count,
		       AVG(total_spent) AS avg_total_spent,
		       AVG(avg_order_value) AS avg_order_value
		FROM customer_segments
		GROUP BY segment, country
		ORDER BY customer_count DESC
		LIMIT 50`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var segment, country string
		var customerCount int
		var avgTotalSpent, avgOrderValue float64
		if err := rows.Scan(&segment, &country, &customerCount, &avgTotalSpent, &avgOrderValue); err != nil {
			return err
		}
	}
	return rows.Err()
}
