// internal/workload/ecommerce/data_loader.go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// loadSampleData generates realistic sample data for the e-commerce platform
func (w *ECommerceWorkload) loadSampleData(ctx context.Context, db *pgxpool.Pool, scale int) error {
	if scale <= 0 {
		scale = 1000 // Default scale
	}

	// Scale factors
	userCount := scale
	vendorCount := scale / 20 // 1 vendor per 20 users
	productCount := scale / 2 // 1 product per 2 users
	orderCount := scale * 2   // 2 orders per user
	reviewCount := scale      // 1 review per user
	sessionCount := scale * 3 // 3 sessions per user

	// Load vendors first (required for products)
	if err := w.loadVendors(ctx, db, vendorCount); err != nil {
		return err
	}

	// Load users
	if err := w.loadUsers(ctx, db, userCount); err != nil {
		return err
	}

	// Load products
	if err := w.loadProducts(ctx, db, productCount, vendorCount); err != nil {
		return err
	}

	// Load inventory
	if err := w.loadInventory(ctx, db, productCount, vendorCount); err != nil {
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
	if err := w.loadProductAnalytics(ctx, db, sessionCount, userCount, productCount); err != nil {
		return err
	}

	// Load some initial purchase orders
	if err := w.loadInitialPurchaseOrders(ctx, db, vendorCount, productCount); err != nil {
		return err
	}

	fmt.Printf("✅ E-Commerce sample data loaded successfully\n")
	return nil
}

// loadVendors generates vendor data
func (w *ECommerceWorkload) loadVendors(ctx context.Context, db *pgxpool.Pool, count int) error {
	fmt.Printf("📦 Loading %d vendors...\n", count)

	vendorNames := []string{
		"TechSupply Corp", "Global Electronics", "Fashion Forward Inc", "Home Essentials Ltd",
		"Sports Gear Co", "Beauty Products Inc", "Toy World Suppliers", "Book Distributors",
		"Auto Parts Plus", "Garden Supplies Co", "Kitchen Warehouse", "Office Solutions",
		"Pet Supply Central", "Music Instruments Co", "Craft Materials Inc", "Health Products Ltd",
		"Jewelry Suppliers", "Watch Company", "Shoe Distributors", "Clothing Manufacturers",
	}

	countries := []string{"USA", "Canada", "UK", "Germany", "France", "Italy", "Spain", "Brazil", "Japan", "China"}
	paymentTerms := []string{"Net 30", "Net 60", "2/10 Net 30", "COD", "Net 15"}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		vendorName := vendorNames[rand.Intn(len(vendorNames))] + fmt.Sprintf(" #%d", i)
		email := fmt.Sprintf("vendor%d@%s.com", i, strings.ToLower(strings.ReplaceAll(vendorName[:10], " ", "")))
		phone := fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(9000)+1000)
		country := countries[rand.Intn(len(countries))]
		paymentTerm := paymentTerms[rand.Intn(len(paymentTerms))]
		leadTime := rand.Intn(14) + 3 // 3-17 days
		minOrder := float64(rand.Intn(1000) + 100)
		rating := 3.0 + rand.Float64()*2.0 // 3.0-5.0

		address := fmt.Sprintf(`{"street": "%d Main St", "city": "Business City", "country": "%s", "postal_code": "%05d"}`,
			rand.Intn(9999)+1, country, rand.Intn(99999))

		batch = append(batch, []interface{}{
			vendorName, email, phone, address, paymentTerm, leadTime, minOrder, rating,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertVendorBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert vendor batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertVendorBatch inserts a batch of vendors
func (w *ECommerceWorkload) insertVendorBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO vendors (vendor_name, contact_email, contact_phone, address, payment_terms, lead_time_days, min_order_amount, rating)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadUsers generates user data
func (w *ECommerceWorkload) loadUsers(ctx context.Context, db *pgxpool.Pool, count int) error {
	fmt.Printf("👥 Loading %d users...\n", count)
	firstNames := []string{"Freya", "Sarah", "Michael", "Sofia", "Lucía", "Léa", "Júlia", "Paula", "Isabela", "Oliver", "Olivia", "Chloe", "William", "Arthur", "Zhi", "Yui", "Grace", "Ben", "Paul", "Emma", "Thomas", "Lucas", "Marie", "Haruto", "Linda", "Miguel", "Manuela", "Kim", "Alexander", "Ren", "Raphaël", "Jessica", "Leo", "Hugo", "Florence", "Isabella", "Riku", "Paul", "Susan", "Hina", "Yuna", "Michael", "Jade", "Carlos", "Clara", "Freddie", "Tommaso", "Riccardo", "Sakura", "Leo", "Anna", "Theo", "Guillaume", "Karen", "Maria", "Lukas", "Emily", "Davi", "Chloé", "Jules", "Paula", "Raphael", "Laura", "Juliette", "Maximilian", "Alina", "Zoe", "Felix", "Hannah", "Bernardo", "Ting", "Henry", "Chloe", "Ella", "Oscar", "Alice", "Isla", "Alice", "Noah", "Emma", "Gabriel", "Sophie", "Leo", "Shan", "Kiara", "Yuto", "Anna", "Thomas", "Alice", "Hannah", "Lucas", "Rin"}
	lastNames := []string{"Nascimento", "Brown", "Tanaka", "Wright", "Campbell", "King", "Takahashi", "Bertrand", "Gao", "Fernández", "Bernard", "Liu", "Almeida", "Wright", "Jones", "Nguyen", "Schneider", "Martin", "Costa", "Suzuki", "López", "Becker", "Richter", "Rossi", "Mendes", "Leroy", "Silva", "Fontaine", "Romano", "Hernández", "Wolf", "Lopez", "Gomes", "Sanchez", "Schäfer", "Meier", "Dupont", "Russo", "Zimmermann", "González", "Miller", "Ribeiro", "Dubois", "Yamada", "Rodríguez", "Anderson", "Souza", "Brown", "Dubois", "Thomas", "Costa", "Araújo", "Müller", "Walker", "Rizzo", "Moreno", "Nguyen", "Fischer", "Alves", "Moreau", "Hayashi", "Lima", "Pérez", "Romero", "Mancini", "Lopez", "Almeida", "Alonso", "Yamada", "Gutiérrez", "Santos", "Guo", "Schwarz", "Morgan", "Marino", "Liu", "Nakamura", "De Luca", "Lee", "Muñoz", "Kimura", "Ribeiro", "Meyer", "Carvalho", "Yamaguchi", "Hoffmann", "Martínez", "Thomas", "Silva", "Dubois", "Hernández", "Schultz", "Costa", "Sato", "Cheng", "Romero", "Costa"}
	countries := []string{"USA", "Canada", "UK", "Germany", "France", "Australia", "Japan", "Brazil", "India", "Mexico"}
	cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego", "Dallas", "Austin"}
	genders := []string{"M", "F", "Other"}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		email := fmt.Sprintf("user%d@example.com", i)
		username := fmt.Sprintf("%s%s%d", strings.ToLower(firstName), strings.ToLower(lastName), i)

		// Birth date (18-80 years old)
		birthYear := time.Now().Year() - (rand.Intn(62) + 18)
		birthDate := time.Date(birthYear, time.Month(rand.Intn(12)+1), rand.Intn(28)+1, 0, 0, 0, 0, time.UTC)

		gender := genders[rand.Intn(len(genders))]
		country := countries[rand.Intn(len(countries))]
		city := cities[rand.Intn(len(cities))]
		postalCode := fmt.Sprintf("%05d", rand.Intn(99999))
		phone := fmt.Sprintf("+1-%03d-%03d-%04d", rand.Intn(900)+100, rand.Intn(900)+100, rand.Intn(9000)+1000)

		// Random last login within last 30 days
		lastLogin := time.Now().AddDate(0, 0, -rand.Intn(30))

		loyaltyPoints := rand.Intn(10000)
		totalSpent := float64(rand.Intn(5000))

		preferences := fmt.Sprintf(`{"newsletter": %t, "sms_notifications": %t, "preferred_categories": ["%s", "%s"]}`,
			rand.Float32() < 0.7, rand.Float32() < 0.3,
			[]string{"Electronics", "Clothing", "Books", "Home"}[rand.Intn(4)],
			[]string{"Sports", "Beauty", "Toys", "Auto"}[rand.Intn(4)])

		batch = append(batch, []interface{}{
			email, username, firstName, lastName, birthDate, gender, country, city, postalCode, phone,
			lastLogin, preferences, loyaltyPoints, totalSpent,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertUserBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert user batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertUserBatch inserts a batch of users
func (w *ECommerceWorkload) insertUserBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO users (email, username, first_name, last_name, date_of_birth, gender, country, city, postal_code, phone, last_login, preferences, loyalty_points, total_spent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadProducts generates product data
func (w *ECommerceWorkload) loadProducts(ctx context.Context, db *pgxpool.Pool, count int, vendorCount int) error {
	fmt.Printf("📦 Loading %d products...\n", count)

	categories := []string{"Electronics", "Clothing", "Books", "Home", "Sports", "Beauty", "Toys", "Auto"}
	subcategories := map[string][]string{
		"Electronics": {"Smartphones", "Laptops", "Tablets", "Cameras", "Audio"},
		"Clothing":    {"Shirts", "Pants", "Dresses", "Shoes", "Accessories"},
		"Books":       {"Fiction", "Non-Fiction", "Educational", "Children", "Comics"},
		"Home":        {"Furniture", "Kitchen", "Bedding", "Decor", "Appliances"},
		"Sports":      {"Fitness", "Outdoor", "Team Sports", "Water Sports", "Winter Sports"},
		"Beauty":      {"Skincare", "Makeup", "Hair Care", "Fragrances", "Tools"},
		"Toys":        {"Educational", "Action Figures", "Dolls", "Games", "Building"},
		"Auto":        {"Parts", "Accessories", "Tools", "Fluids", "Electronics"},
	}

	brands := []string{"BrandA", "BrandB", "BrandC", "BrandD", "BrandE", "Premium", "Value", "Elite", "Classic", "Modern"}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		sku := fmt.Sprintf("SKU-%06d", i)
		category := categories[rand.Intn(len(categories))]
		subcategory := subcategories[category][rand.Intn(len(subcategories[category]))]
		brand := brands[rand.Intn(len(brands))]

		name := fmt.Sprintf("%s %s %s #%d", brand, category, subcategory, i)
		description := fmt.Sprintf("High-quality %s from %s. Perfect for your %s needs.", subcategory, brand, strings.ToLower(category))

		cost := float64(rand.Intn(200) + 10)
		margin := 60.0 + rand.Float64()*40.0 // 60-100% margin
		price := cost * (1 + margin/100.0)

		weight := rand.Float64() * 10.0 // 0-10 kg

		dimensions := fmt.Sprintf(`{"width": %.1f, "height": %.1f, "depth": %.1f}`,
			rand.Float64()*50+5, rand.Float64()*50+5, rand.Float64()*30+5)

		tags := fmt.Sprintf(`{"%s", "%s", "%s"}`,
			strings.ToLower(category), strings.ToLower(brand), strings.ToLower(subcategory))

		attributes := fmt.Sprintf(`{"color": "%s", "material": "%s", "warranty": "%d months"}`,
			[]string{"Black", "White", "Red", "Blue", "Green"}[rand.Intn(5)],
			[]string{"Plastic", "Metal", "Wood", "Fabric", "Glass"}[rand.Intn(5)],
			[]int{6, 12, 24, 36}[rand.Intn(4)])

		avgRating := 1.0 + rand.Float64()*4.0 // 1.0-5.0
		reviewCount := rand.Intn(1000)
		viewCount := rand.Intn(10000)
		vendorID := rand.Intn(vendorCount) + 1

		batch = append(batch, []interface{}{
			sku, name, description, category, subcategory, brand, price, cost, margin,
			weight, dimensions, tags, attributes, avgRating, reviewCount, viewCount, vendorID,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertProductBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert product batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertProductBatch inserts a batch of products
func (w *ECommerceWorkload) insertProductBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO products (sku, name, description, category, subcategory, brand, price, cost, margin_percent, weight_kg, dimensions, tags, attributes, avg_rating, review_count, view_count, vendor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadInventory generates inventory data
func (w *ECommerceWorkload) loadInventory(ctx context.Context, db *pgxpool.Pool, productCount int, vendorCount int) error {
	fmt.Printf("📦 Loading inventory for %d products...\n", productCount)

	warehouses := []string{"Main Warehouse", "East Coast", "West Coast", "Central", "International"}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= productCount; i++ {
		warehouse := warehouses[rand.Intn(len(warehouses))]
		quantityAvailable := rand.Intn(200) + 10 // 10-210
		quantityReserved := rand.Intn(quantityAvailable / 4)
		reorderLevel := rand.Intn(30) + 5                  // 5-35
		maxStockLevel := reorderLevel * (rand.Intn(5) + 3) // 3-7x reorder level

		// Some items recently restocked
		var lastRestocked *time.Time
		if rand.Float32() < 0.7 {
			restock := time.Now().AddDate(0, 0, -rand.Intn(30))
			lastRestocked = &restock
		}

		supplierID := rand.Intn(vendorCount) + 1
		unitCost := float64(rand.Intn(100) + 5)
		autoReorder := rand.Float32() < 0.8 // 80% have auto-reorder enabled

		batch = append(batch, []interface{}{
			i, warehouse, quantityAvailable, quantityReserved, reorderLevel, maxStockLevel,
			lastRestocked, supplierID, unitCost, autoReorder,
		})

		if len(batch) >= 100 || i == productCount {
			err := w.insertInventoryBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert inventory batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertInventoryBatch inserts a batch of inventory records
func (w *ECommerceWorkload) insertInventoryBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO inventory (product_id, warehouse_location, quantity_available, quantity_reserved, reorder_level, max_stock_level, last_restocked, supplier_id, unit_cost, auto_reorder)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadOrders generates order data
func (w *ECommerceWorkload) loadOrders(ctx context.Context, db *pgxpool.Pool, count int, userCount int, productCount int) error {
	fmt.Printf("🛒 Loading %d orders...\n", count)

	statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}
	paymentMethods := []string{"credit_card", "debit_card", "paypal", "apple_pay", "google_pay"}

	batch := make([][]interface{}, 0, 50)
	for i := 1; i <= count; i++ {
		userID := rand.Intn(userCount) + 1
		orderNumber := fmt.Sprintf("ORD-%d-%06d", time.Now().Year(), i)
		status := statuses[rand.Intn(len(statuses))]
		paymentMethod := paymentMethods[rand.Intn(len(paymentMethods))]

		// Order created within last 90 days
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(90))

		// Calculate shipping and tax
		subtotal := float64(rand.Intn(500) + 20)
		shippingCost := 5.99 + rand.Float64()*10.0
		taxAmount := subtotal * 0.08 // 8% tax
		discountAmount := 0.0
		if rand.Float32() < 0.2 { // 20% chance of discount
			discountAmount = subtotal * (0.05 + rand.Float64()*0.15) // 5-20% discount
		}
		totalAmount := subtotal + shippingCost + taxAmount - discountAmount

		// Addresses
		shippingAddress := fmt.Sprintf(`{"street": "%d Main St", "city": "Sample City", "state": "ST", "postal_code": "%05d", "country": "USA"}`,
			rand.Intn(9999)+1, rand.Intn(99999))
		billingAddress := shippingAddress

		// Set shipped/delivered dates for completed orders
		var shippedAt, deliveredAt *time.Time
		if status == "shipped" || status == "delivered" {
			shipped := createdAt.AddDate(0, 0, rand.Intn(5)+1)
			shippedAt = &shipped
		}
		if status == "delivered" {
			delivered := shippedAt.AddDate(0, 0, rand.Intn(7)+1)
			deliveredAt = &delivered
		}

		batch = append(batch, []interface{}{
			userID, orderNumber, status, totalAmount, shippingCost, taxAmount, discountAmount,
			paymentMethod, shippingAddress, billingAddress, createdAt, shippedAt, deliveredAt,
		})

		if len(batch) >= 50 || i == count {
			orderIDs, err := w.insertOrderBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert order batch: %w", err)
			}

			// Create order items for each order
			for j, orderID := range orderIDs {
				batchIndex := len(orderIDs) - len(batch) + j
				if batchIndex >= 0 && batchIndex < len(batch) {
					err := w.createOrderItems(ctx, db, orderID, productCount)
					if err != nil {
						return fmt.Errorf("failed to create order items: %w", err)
					}
				}
			}

			batch = batch[:0]
		}
	}

	return nil
}

// insertOrderBatch inserts a batch of orders and returns their IDs
func (w *ECommerceWorkload) insertOrderBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) ([]int, error) {
	query := `
		INSERT INTO orders (user_id, order_number, status, total_amount, shipping_cost, tax_amount, discount_amount, payment_method, shipping_address, billing_address, created_at, shipped_at, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING order_id`

	var orderIDs []int
	for _, row := range batch {
		var orderID int
		err := db.QueryRow(ctx, query, row...).Scan(&orderID)
		if err != nil {
			return nil, err
		}
		orderIDs = append(orderIDs, orderID)
	}
	return orderIDs, nil
}

// createOrderItems creates order items for an order
func (w *ECommerceWorkload) createOrderItems(ctx context.Context, db *pgxpool.Pool, orderID int, productCount int) error {
	numItems := rand.Intn(5) + 1 // 1-5 items per order

	for i := 0; i < numItems; i++ {
		productID := rand.Intn(productCount) + 1
		quantity := rand.Intn(3) + 1 // 1-3 of each item
		unitPrice := float64(rand.Intn(200) + 10)
		totalPrice := float64(quantity) * unitPrice
		discountApplied := 0.0
		if rand.Float32() < 0.1 { // 10% chance of item discount
			discountApplied = totalPrice * (rand.Float64() * 0.2) // Up to 20% discount
		}

		_, err := db.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price, discount_applied)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			orderID, productID, quantity, unitPrice, totalPrice, discountApplied)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadReviews generates review data
func (w *ECommerceWorkload) loadReviews(ctx context.Context, db *pgxpool.Pool, count int, userCount int, productCount int) error {
	fmt.Printf("⭐ Loading %d reviews...\n", count)

	reviewTitles := []string{
		"Great product!", "Excellent quality", "Not what I expected", "Amazing value",
		"Perfect for my needs", "Could be better", "Outstanding service", "Highly recommend",
		"Disappointed", "Exceeded expectations", "Good purchase", "Will buy again",
	}

	reviewContents := []string{
		"This product exceeded my expectations. The quality is outstanding and it arrived quickly.",
		"I'm very happy with this purchase. Great value for money and exactly as described.",
		"The product is okay but not as good as I hoped. The quality could be better.",
		"Fantastic product! Works perfectly and the customer service was excellent.",
		"I love this item. It's exactly what I was looking for and the price was great.",
		"Not impressed. The product doesn't match the description and quality is poor.",
		"Excellent quality and fast shipping. Would definitely recommend to others.",
		"The product is good but arrived damaged. Customer service was helpful though.",
		"Perfect! This is exactly what I needed and it works flawlessly.",
		"Average product. Nothing special but it does the job adequately.",
	}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		userID := rand.Intn(userCount) + 1
		productID := rand.Intn(productCount) + 1
		rating := rand.Intn(5) + 1
		title := reviewTitles[rand.Intn(len(reviewTitles))]
		content := reviewContents[rand.Intn(len(reviewContents))]

		// Generate random vector for content (1536 dimensions for OpenAI embeddings)
		vector := make([]float32, 1536)
		for j := range vector {
			vector[j] = rand.Float32()*2 - 1 // Random values between -1 and 1
		}

		vectorStr := "["
		for j, v := range vector {
			if j > 0 {
				vectorStr += ","
			}
			vectorStr += fmt.Sprintf("%.6f", v)
		}
		vectorStr += "]"

		helpfulVotes := rand.Intn(50)
		totalVotes := helpfulVotes + rand.Intn(20)
		isVerifiedPurchase := rand.Float32() < 0.8 // 80% are verified purchases

		// Review created within last 180 days
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(180))

		// Get a real order ID for verified purchases
		var orderID *int
		if isVerifiedPurchase {
			var oid int
			err := db.QueryRow(ctx, `
				SELECT o.order_id 
				FROM orders o 
				JOIN order_items oi ON o.order_id = oi.order_id 
				WHERE o.user_id = $1 AND oi.product_id = $2 
				ORDER BY RANDOM() 
				LIMIT 1`, userID, productID).Scan(&oid)
			if err == nil {
				orderID = &oid
			}
		}

		batch = append(batch, []interface{}{
			userID, productID, orderID, rating, title, content, vectorStr,
			helpfulVotes, totalVotes, isVerifiedPurchase, createdAt,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertReviewBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert review batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertReviewBatch inserts a batch of reviews
func (w *ECommerceWorkload) insertReviewBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO reviews (user_id, product_id, order_id, rating, title, content, content_vector, helpful_votes, total_votes, is_verified_purchase, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::vector, $8, $9, $10, $11)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			// If vector insertion fails, try without vector
			queryNoVector := `
				INSERT INTO reviews (user_id, product_id, order_id, rating, title, content, helpful_votes, total_votes, is_verified_purchase, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $8, $9, $10, $11)`
			_, err = db.Exec(ctx, queryNoVector, row...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// loadUserSessions generates user session data
func (w *ECommerceWorkload) loadUserSessions(ctx context.Context, db *pgxpool.Pool, count int, userCount int) error {
	fmt.Printf("🔗 Loading %d user sessions...\n", count)

	deviceTypes := []string{"Desktop", "Mobile", "Tablet"}
	browsers := []string{"Chrome", "Firefox", "Safari", "Edge", "Opera"}
	operatingSystems := []string{"Windows 10", "macOS", "Linux", "iOS", "Android"}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		userID := rand.Intn(userCount) + 1
		sessionID := fmt.Sprintf("sess_%d_%d", userID, i)
		deviceType := deviceTypes[rand.Intn(len(deviceTypes))]
		browser := browsers[rand.Intn(len(browsers))]
		os := operatingSystems[rand.Intn(len(operatingSystems))]

		// Random IP address
		ipAddress := fmt.Sprintf("%d.%d.%d.%d",
			rand.Intn(255)+1, rand.Intn(255), rand.Intn(255), rand.Intn(255))

		// Session started within last 30 days
		startedAt := time.Now().AddDate(0, 0, -rand.Intn(30))

		// 70% of sessions have ended
		var endedAt *time.Time
		var durationSeconds *int
		if rand.Float32() < 0.7 {
			duration := rand.Intn(3600) + 60 // 1 minute to 1 hour
			ended := startedAt.Add(time.Duration(duration) * time.Second)
			endedAt = &ended
			durationSeconds = &duration
		}

		batch = append(batch, []interface{}{
			userID, sessionID, deviceType, browser, os, ipAddress,
			startedAt, endedAt, durationSeconds,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertUserSessionBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert user session batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertUserSessionBatch inserts a batch of user sessions
func (w *ECommerceWorkload) insertUserSessionBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO user_sessions (user_id, session_id, device_type, browser, operating_system, ip_address, started_at, ended_at, duration_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadProductAnalytics generates product analytics data
func (w *ECommerceWorkload) loadProductAnalytics(ctx context.Context, db *pgxpool.Pool, count int, userCount int, productCount int) error {
	fmt.Printf("📊 Loading %d product analytics events...\n", count)

	eventTypes := []string{"view", "add_to_cart", "purchase", "wishlist", "search"}
	searchQueries := []string{
		"laptop", "phone", "book", "shirt", "shoes", "watch", "camera", "tablet",
		"headphones", "speaker", "keyboard", "mouse", "monitor", "desk", "chair",
	}

	batch := make([][]interface{}, 0, 100)
	for i := 1; i <= count; i++ {
		userID := rand.Intn(userCount) + 1
		productID := rand.Intn(productCount) + 1
		eventType := eventTypes[rand.Intn(len(eventTypes))]

		var searchQuery *string
		if eventType == "search" {
			query := searchQueries[rand.Intn(len(searchQueries))]
			searchQuery = &query
		}

		// Event within last 60 days
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(60))

		batch = append(batch, []interface{}{
			productID, userID, eventType, searchQuery, createdAt,
		})

		if len(batch) >= 100 || i == count {
			err := w.insertProductAnalyticsBatch(ctx, db, batch)
			if err != nil {
				return fmt.Errorf("failed to insert product analytics batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	return nil
}

// insertProductAnalyticsBatch inserts a batch of product analytics
func (w *ECommerceWorkload) insertProductAnalyticsBatch(ctx context.Context, db *pgxpool.Pool, batch [][]interface{}) error {
	query := `
		INSERT INTO product_analytics (product_id, user_id, event_type, search_query, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	for _, row := range batch {
		_, err := db.Exec(ctx, query, row...)
		if err != nil {
			return err
		}
	}
	return nil
}

// loadInitialPurchaseOrders creates some initial purchase orders
func (w *ECommerceWorkload) loadInitialPurchaseOrders(ctx context.Context, db *pgxpool.Pool, vendorCount int, productCount int) error {
	fmt.Printf("📋 Loading initial purchase orders...\n")

	// Create 10-20 purchase orders
	orderCount := rand.Intn(11) + 10

	for i := 1; i <= orderCount; i++ {
		vendorID := rand.Intn(vendorCount) + 1
		poNumber := fmt.Sprintf("PO-%d-%06d", time.Now().Year(), i)

		statuses := []string{"pending", "sent", "received"}
		status := statuses[rand.Intn(len(statuses))]

		// Order created within last 30 days
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(30))
		expectedDelivery := createdAt.AddDate(0, 0, rand.Intn(14)+3) // 3-17 days later

		var actualDelivery *time.Time
		if status == "received" {
			delivered := expectedDelivery.AddDate(0, 0, rand.Intn(5)-2) // -2 to +3 days from expected
			actualDelivery = &delivered
		}

		totalAmount := float64(rand.Intn(5000) + 500)
		taxAmount := totalAmount * 0.08
		shippingCost := 50.0 + rand.Float64()*200.0

		// Create purchase order
		var poID int
		err := db.QueryRow(ctx, `
			INSERT INTO purchase_orders (po_number, vendor_id, status, total_amount, tax_amount, shipping_cost, expected_delivery, actual_delivery, created_at, created_by, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING po_id`,
			poNumber, vendorID, status, totalAmount, taxAmount, shippingCost,
			expectedDelivery, actualDelivery, createdAt, "system",
			"Initial purchase order for inventory setup").Scan(&poID)
		if err != nil {
			return err
		}

		// Add purchase order items
		numItems := rand.Intn(5) + 2 // 2-6 items per PO
		for j := 0; j < numItems; j++ {
			productID := rand.Intn(productCount) + 1
			quantityOrdered := rand.Intn(100) + 10
			unitCost := float64(rand.Intn(200) + 5)
			totalCost := float64(quantityOrdered) * unitCost

			var quantityReceived int
			var receivedAt *time.Time
			if status == "received" {
				quantityReceived = quantityOrdered - rand.Intn(3) // Maybe some shortfall
				if actualDelivery != nil {
					receivedAt = actualDelivery
				}
			}

			_, err = db.Exec(ctx, `
				INSERT INTO purchase_order_items (po_id, product_id, quantity_ordered, quantity_received, unit_cost, total_cost, received_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				poID, productID, quantityOrdered, quantityReceived, unitCost, totalCost, receivedAt)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
