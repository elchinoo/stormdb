// internal/workload/ecommerce_basic/data_loader.go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/internal/progress"
	"github.com/jackc/pgx/v5/pgxpool"
)

// loadSampleData generates realistic sample data for the e-commerce platform
func (w *ECommerceBasicWorkload) loadSampleData(ctx context.Context, db *pgxpool.Pool, scale int) error {
	if scale <= 0 {
		scale = 1000 // Default scale
	}

	// Scale factors
	userCount := scale
	productCount := scale / 2
	orderCount := scale * 2
	reviewCount := scale
	sessionCount := scale * 3

	fmt.Printf("📊 Loading E-Commerce Basic sample data (scale=%d)...\n", scale)

	// Load users
	if err := w.loadUsers(ctx, db, userCount); err != nil {
		return err
	}

	// Load products
	if err := w.loadProducts(ctx, db, productCount); err != nil {
		return err
	}

	// Load inventory
	if err := w.loadInventory(ctx, db, productCount); err != nil {
		return err
	}

	// Load orders and order items
	if err := w.loadOrders(ctx, db, orderCount, userCount, productCount); err != nil {
		return err
	}

	// Load reviews
	if err := w.loadReviews(ctx, db, reviewCount, userCount, productCount); err != nil {
		return err
	}

	// Load user sessions
	if err := w.loadUserSessions(ctx, db, sessionCount, userCount); err != nil {
		return err
	}

	// Load product analytics
	if err := w.loadProductAnalytics(ctx, db, sessionCount*2, userCount, productCount); err != nil {
		return err
	}

	return nil
}

// loadUsers creates realistic user data
func (w *ECommerceBasicWorkload) loadUsers(ctx context.Context, db *pgxpool.Pool, count int) error {
	firstNames := []string{"John", "Jane", "Michael", "Sarah", "David", "Lisa", "Robert", "Mary", "James", "Jennifer",
		"William", "Elizabeth", "Richard", "Maria", "Joseph", "Susan", "Thomas", "Jessica", "Christopher", "Karen"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
		"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin"}
	countries := []string{"USA", "Canada", "UK", "Germany", "France", "Australia", "Japan", "Brazil", "India", "Mexico"}
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose"}

	// Create progress tracker
	userProgress := progress.NewTracker("👥 Loading users", count)

	batch := 100
	for i := 0; i < count; i += batch {
		remaining := count - i
		if remaining < batch {
			batch = remaining
		}

		values := make([]interface{}, 0, batch*16)
		query := "INSERT INTO users (email, username, first_name, last_name, date_of_birth, gender, country, city, postal_code, phone, last_login, account_status, preferences, loyalty_points, total_spent) VALUES "

		for j := 0; j < batch; j++ {
			userID := i + j + 1
			firstName := firstNames[rand.Intn(len(firstNames))]
			lastName := lastNames[rand.Intn(len(lastNames))]
			email := fmt.Sprintf("%s.%s%d@example.com", firstName, lastName, userID)
			username := fmt.Sprintf("%s%s%d", firstName, lastName, userID)

			birthYear := 1960 + rand.Intn(40)
			birthDate := fmt.Sprintf("%d-%02d-%02d", birthYear, rand.Intn(12)+1, rand.Intn(28)+1)

			gender := []string{"male", "female", "other"}[rand.Intn(3)]
			country := countries[rand.Intn(len(countries))]
			city := cities[rand.Intn(len(cities))]
			postalCode := fmt.Sprintf("%05d", rand.Intn(100000))
			phone := fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(10000))

			lastLogin := time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
			status := []string{"active", "inactive", "suspended"}[rand.Intn(3)]

			preferences := fmt.Sprintf(`{"newsletter": %t, "sms_notifications": %t, "language": "en", "currency": "USD", "categories_of_interest": ["electronics", "books", "clothing"]}`,
				rand.Intn(2) == 1, rand.Intn(2) == 1)

			loyaltyPoints := rand.Intn(10000)
			totalSpent := rand.Float64() * 50000

			if j > 0 {
				query += ", "
			}
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4, len(values)+5,
				len(values)+6, len(values)+7, len(values)+8, len(values)+9, len(values)+10,
				len(values)+11, len(values)+12, len(values)+13, len(values)+14, len(values)+15)

			values = append(values, email, username, firstName, lastName, birthDate, gender, country, city, postalCode, phone, lastLogin, status, preferences, loyaltyPoints, totalSpent)
		}

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert users batch: %w", err)
		}

		userProgress.Update(i + batch)
	}

	return nil
}

// loadProducts creates realistic product data
func (w *ECommerceBasicWorkload) loadProducts(ctx context.Context, db *pgxpool.Pool, count int) error {
	categories := []string{"Electronics", "Books", "Clothing", "Home & Garden", "Sports", "Beauty", "Toys", "Automotive"}
	brands := []string{"Samsung", "Apple", "Nike", "Adidas", "Sony", "Dell", "HP", "Canon", "KitchenAid", "IKEA"}
	adjectives := []string{"Premium", "Ultimate", "Professional", "Deluxe", "Classic", "Modern", "Vintage", "Advanced", "Smart", "Eco-Friendly"}
	nouns := []string{"Smartphone", "Laptop", "Headphones", "Camera", "Watch", "Shoes", "Jacket", "Chair", "Table", "Keyboard"}

	batch := 100
	for i := 0; i < count; i += batch {
		remaining := count - i
		if remaining < batch {
			batch = remaining
		}

		values := make([]interface{}, 0, batch*16)
		query := "INSERT INTO products (sku, name, description, category, subcategory, brand, price, cost, weight_kg, dimensions, tags, attributes, avg_rating, review_count, view_count) VALUES "

		for j := 0; j < batch; j++ {
			productID := i + j + 1
			sku := fmt.Sprintf("SKU-%06d", productID)

			adjective := adjectives[rand.Intn(len(adjectives))]
			noun := nouns[rand.Intn(len(nouns))]
			brand := brands[rand.Intn(len(brands))]
			name := fmt.Sprintf("%s %s %s", brand, adjective, noun)

			description := fmt.Sprintf("High-quality %s from %s. Features advanced technology and premium materials for the ultimate user experience.", noun, brand)

			category := categories[rand.Intn(len(categories))]
			subcategory := fmt.Sprintf("%s Accessories", category)

			price := 19.99 + rand.Float64()*980.01     // $19.99 to $1000
			cost := price * (0.4 + rand.Float64()*0.3) // 40-70% of price

			weight := 0.1 + rand.Float64()*49.9 // 0.1kg to 50kg

			dimensions := fmt.Sprintf(`{"width": %.1f, "height": %.1f, "depth": %.1f}`,
				rand.Float64()*50+1, rand.Float64()*50+1, rand.Float64()*50+1)

			tags := fmt.Sprintf(`{"%s", "%s", "bestseller", "premium"}`, category, brand)

			attributes := fmt.Sprintf(`{"color": "black", "warranty": "%d years", "material": "premium", "energy_rating": "A+"}`,
				1+rand.Intn(5))

			avgRating := 1.0 + rand.Float64()*4.0 // 1.0 to 5.0
			reviewCount := rand.Intn(1000)
			viewCount := rand.Intn(10000)

			if j > 0 {
				query += ", "
			}
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4, len(values)+5,
				len(values)+6, len(values)+7, len(values)+8, len(values)+9, len(values)+10,
				len(values)+11, len(values)+12, len(values)+13, len(values)+14, len(values)+15)

			values = append(values, sku, name, description, category, subcategory, brand, price, cost, weight, dimensions, tags, attributes, avgRating, reviewCount, viewCount)
		}

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert products batch: %w", err)
		}
	}

	return nil
}

// loadInventory creates inventory records for products
func (w *ECommerceBasicWorkload) loadInventory(ctx context.Context, db *pgxpool.Pool, productCount int) error {
	warehouses := []string{"North America - East", "North America - West", "Europe - Central", "Asia Pacific", "South America"}

	batch := 100
	for i := 1; i <= productCount; i += batch {
		remaining := productCount - i + 1
		if remaining < batch {
			batch = remaining
		}

		values := make([]interface{}, 0, batch*8)
		query := "INSERT INTO inventory (product_id, warehouse_location, quantity_available, quantity_reserved, reorder_level, last_restocked, supplier_id, unit_cost) VALUES "

		for j := 0; j < batch && i+j <= productCount; j++ {
			productID := i + j
			warehouse := warehouses[rand.Intn(len(warehouses))]
			quantityAvailable := rand.Intn(1000) + 10
			quantityReserved := rand.Intn(quantityAvailable / 10)
			reorderLevel := 10 + rand.Intn(50)
			lastRestocked := time.Now().Add(-time.Duration(rand.Intn(90)) * 24 * time.Hour)
			supplierID := rand.Intn(100) + 1
			unitCost := 5.0 + rand.Float64()*495.0

			if j > 0 {
				query += ", "
			}
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4,
				len(values)+5, len(values)+6, len(values)+7, len(values)+8)

			values = append(values, productID, warehouse, quantityAvailable, quantityReserved, reorderLevel, lastRestocked, supplierID, unitCost)
		}

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert inventory batch: %w", err)
		}
	}

	return nil
}

// Additional loading methods would go here (loadOrders, loadReviews, etc.)
// For brevity, I'll implement the key ones and you can extend as needed

func (w *ECommerceBasicWorkload) loadOrders(ctx context.Context, db *pgxpool.Pool, orderCount, userCount, productCount int) error {
	statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled", "returned"}

	batch := 50
	for i := 0; i < orderCount; i += batch {
		currentBatch := batch
		if i+batch > orderCount {
			currentBatch = orderCount - i
		}

		// First, create orders
		orderValues := make([]interface{}, 0, currentBatch*6)
		orderPlaceholders := make([]string, 0, currentBatch)

		for j := 0; j < currentBatch; j++ {
			userID := rand.Intn(userCount) + 1
			orderNumber := fmt.Sprintf("ORD-%d-%06d", time.Now().Year(), i+j+1)
			status := statuses[rand.Intn(len(statuses))]
			createdAt := time.Now().AddDate(0, 0, -rand.Intn(365))

			placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
				len(orderValues)+1, len(orderValues)+2, len(orderValues)+3,
				len(orderValues)+4, len(orderValues)+5, len(orderValues)+6)
			orderPlaceholders = append(orderPlaceholders, placeholder)

			orderValues = append(orderValues, userID, orderNumber, status, 0.0, createdAt, createdAt)
		}

		orderQuery := fmt.Sprintf(`
			INSERT INTO orders (user_id, order_number, status, total_amount, created_at, updated_at)
			VALUES %s RETURNING order_id`, strings.Join(orderPlaceholders, ", "))

		rows, err := db.Query(ctx, orderQuery, orderValues...)
		if err != nil {
			return fmt.Errorf("failed to insert orders batch: %w", err)
		}

		var orderIDs []int
		for rows.Next() {
			var orderID int
			if err := rows.Scan(&orderID); err != nil {
				rows.Close()
				return err
			}
			orderIDs = append(orderIDs, orderID)
		}
		rows.Close()

		// Create order items for each order
		for _, orderID := range orderIDs {
			itemCount := rand.Intn(5) + 1 // 1-5 items per order
			totalAmount := 0.0

			for k := 0; k < itemCount; k++ {
				productID := rand.Intn(productCount) + 1
				quantity := rand.Intn(3) + 1
				unitPrice := float64(rand.Intn(500) + 10)
				totalPrice := float64(quantity) * unitPrice

				_, err := db.Exec(ctx, `
					INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price)
					VALUES ($1, $2, $3, $4, $5)`,
					orderID, productID, quantity, unitPrice, totalPrice)
				if err != nil {
					return fmt.Errorf("failed to insert order item: %w", err)
				}

				totalAmount += totalPrice
			}

			// Update order total
			_, err := db.Exec(ctx, `
				UPDATE orders SET total_amount = $1 WHERE order_id = $2`,
				totalAmount, orderID)
			if err != nil {
				return fmt.Errorf("failed to update order total: %w", err)
			}
		}
	}

	return nil
}

func (w *ECommerceBasicWorkload) loadReviews(ctx context.Context, db *pgxpool.Pool, reviewCount, userCount, productCount int) error {
	titles := []string{
		"Great product!", "Not what I expected", "Amazing quality", "Could be better",
		"Excellent value", "Perfect for my needs", "Would recommend", "Disappointed",
		"Outstanding service", "Good but overpriced", "Fantastic experience", "Just okay",
	}

	batch := 100
	for i := 0; i < reviewCount; i += batch {
		currentBatch := batch
		if i+batch > reviewCount {
			currentBatch = reviewCount - i
		}

		values := make([]interface{}, 0, currentBatch*7)
		placeholders := make([]string, 0, currentBatch)

		for j := 0; j < currentBatch; j++ {
			userID := rand.Intn(userCount) + 1
			productID := rand.Intn(productCount) + 1
			rating := rand.Intn(5) + 1
			title := titles[rand.Intn(len(titles))]
			content := fmt.Sprintf("This is a review for product %d. Rating: %d stars. %s", productID, rating, title)
			helpfulVotes := rand.Intn(20)
			createdAt := time.Now().AddDate(0, 0, -rand.Intn(365))

			placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4,
				len(values)+5, len(values)+6, len(values)+7)
			placeholders = append(placeholders, placeholder)

			values = append(values, userID, productID, rating, title, content, helpfulVotes, createdAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO reviews (user_id, product_id, rating, title, content, helpful_votes, created_at)
			VALUES %s`, strings.Join(placeholders, ", "))

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert reviews batch: %w", err)
		}
	}

	return nil
}

func (w *ECommerceBasicWorkload) loadUserSessions(ctx context.Context, db *pgxpool.Pool, sessionCount, userCount int) error {
	devices := []string{"desktop", "mobile", "tablet"}
	browsers := []string{"Chrome", "Firefox", "Safari", "Edge", "Opera"}
	operatingSystems := []string{"Windows", "macOS", "Linux", "iOS", "Android"}

	batch := 100
	for i := 0; i < sessionCount; i += batch {
		currentBatch := batch
		if i+batch > sessionCount {
			currentBatch = sessionCount - i
		}

		values := make([]interface{}, 0, currentBatch*9)
		placeholders := make([]string, 0, currentBatch)

		for j := 0; j < currentBatch; j++ {
			userID := rand.Intn(userCount) + 1
			sessionID := fmt.Sprintf("sess_%d_%d_%d", userID, time.Now().Unix(), rand.Intn(10000))
			deviceType := devices[rand.Intn(len(devices))]
			browser := browsers[rand.Intn(len(browsers))]
			os := operatingSystems[rand.Intn(len(operatingSystems))]
			ipAddress := fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255))
			startedAt := time.Now().AddDate(0, 0, -rand.Intn(90))
			duration := rand.Intn(3600) + 60 // 1 minute to 1 hour
			endedAt := startedAt.Add(time.Duration(duration) * time.Second)

			placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4,
				len(values)+5, len(values)+6, len(values)+7, len(values)+8, len(values)+9)
			placeholders = append(placeholders, placeholder)

			values = append(values, userID, sessionID, deviceType, browser, os, ipAddress, startedAt, endedAt, duration)
		}

		query := fmt.Sprintf(`
			INSERT INTO user_sessions (user_id, session_id, device_type, browser, operating_system, ip_address, started_at, ended_at, duration_seconds)
			VALUES %s`, strings.Join(placeholders, ", "))

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert user sessions batch: %w", err)
		}
	}

	return nil
}

func (w *ECommerceBasicWorkload) loadProductAnalytics(ctx context.Context, db *pgxpool.Pool, eventCount, userCount, productCount int) error {
	eventTypes := []string{"view", "add_to_cart", "remove_from_cart", "purchase", "wishlist_add", "wishlist_remove", "compare", "share"}

	batch := 200
	for i := 0; i < eventCount; i += batch {
		currentBatch := batch
		if i+batch > eventCount {
			currentBatch = eventCount - i
		}

		values := make([]interface{}, 0, currentBatch*4)
		placeholders := make([]string, 0, currentBatch)

		for j := 0; j < currentBatch; j++ {
			userID := rand.Intn(userCount) + 1
			productID := rand.Intn(productCount) + 1
			eventType := eventTypes[rand.Intn(len(eventTypes))]
			createdAt := time.Now().AddDate(0, 0, -rand.Intn(30)) // Last 30 days

			placeholder := fmt.Sprintf("($%d, $%d, $%d, $%d)",
				len(values)+1, len(values)+2, len(values)+3, len(values)+4)
			placeholders = append(placeholders, placeholder)

			values = append(values, userID, productID, eventType, createdAt)
		}

		query := fmt.Sprintf(`
			INSERT INTO product_analytics (user_id, product_id, event_type, created_at)
			VALUES %s`, strings.Join(placeholders, ", "))

		if _, err := db.Exec(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to insert product analytics batch: %w", err)
		}
	}

	return nil
}
