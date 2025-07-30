// internal/workload/ecommerce_basic/ecommerce_basic.go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ECommerceBasicWorkload simulates a basic e-commerce platform with standard OLTP patterns
// This includes: users, products, orders, inventory, reviews, user sessions, product analytics
type ECommerceBasicWorkload struct {
	Mode string
}

// GetName returns the workload name
func (w *ECommerceBasicWorkload) GetName() string {
	return "ecommerce_basic_" + w.Mode
}

// Setup creates the basic e-commerce schema and loads sample data
func (w *ECommerceBasicWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🛍️ Setting up Basic E-Commerce workload...")

	// Check if schema already exists
	var tableCount int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('users', 'products', 'orders', 'inventory', 'reviews', 'user_sessions', 'product_analytics')
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	if tableCount == 7 {
		log.Printf("✅ Basic E-Commerce schema already exists")
	} else {
		log.Printf("📊 Creating Basic E-Commerce schema...")
		if err := w.createSchema(ctx, db); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		log.Printf("✅ Basic E-Commerce schema created successfully")
	}

	// Load sample data if tables are empty
	var userCount int64
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if userCount == 0 {
		log.Printf("📊 Generating Basic E-Commerce sample data...")
		if err := w.loadSampleData(ctx, db, cfg.Scale); err != nil {
			return fmt.Errorf("failed to load sample data: %w", err)
		}
	} else {
		log.Printf("✅ Basic E-Commerce data already exists (%d users)", userCount)
	}

	return nil
}

// createSchema creates all the tables for the basic e-commerce workload
func (w *ECommerceBasicWorkload) createSchema(ctx context.Context, db *pgxpool.Pool) error {
	schemas := []string{
		// Users table - customer data with demographics and preferences
		`CREATE TABLE users (
			user_id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(100) UNIQUE NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			date_of_birth DATE,
			gender VARCHAR(10),
			country VARCHAR(100),
			city VARCHAR(100),
			postal_code VARCHAR(20),
			phone VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP WITH TIME ZONE,
			account_status VARCHAR(20) DEFAULT 'active',
			preferences JSONB,
			loyalty_points INTEGER DEFAULT 0,
			total_spent DECIMAL(12,2) DEFAULT 0.00
		)`,

		// Products table - product catalog with categories and metadata
		`CREATE TABLE products (
			product_id SERIAL PRIMARY KEY,
			sku VARCHAR(100) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(100),
			subcategory VARCHAR(100),
			brand VARCHAR(100),
			price DECIMAL(10,2) NOT NULL,
			cost DECIMAL(10,2),
			weight_kg DECIMAL(8,3),
			dimensions JSONB, -- {width, height, depth}
			tags TEXT[],
			attributes JSONB, -- flexible product attributes
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT true,
			avg_rating DECIMAL(3,2) DEFAULT 0.00,
			review_count INTEGER DEFAULT 0,
			view_count INTEGER DEFAULT 0
		)`,

		// Inventory table - stock management
		`CREATE TABLE inventory (
			inventory_id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(product_id),
			warehouse_location VARCHAR(100),
			quantity_available INTEGER NOT NULL,
			quantity_reserved INTEGER DEFAULT 0,
			reorder_level INTEGER DEFAULT 10,
			last_restocked TIMESTAMP WITH TIME ZONE,
			supplier_id INTEGER,
			unit_cost DECIMAL(10,2),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// Orders table - customer orders
		`CREATE TABLE orders (
			order_id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(user_id),
			order_number VARCHAR(50) UNIQUE NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			total_amount DECIMAL(12,2) NOT NULL,
			shipping_cost DECIMAL(8,2) DEFAULT 0.00,
			tax_amount DECIMAL(8,2) DEFAULT 0.00,
			discount_amount DECIMAL(8,2) DEFAULT 0.00,
			payment_method VARCHAR(50),
			shipping_address JSONB,
			billing_address JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			shipped_at TIMESTAMP WITH TIME ZONE,
			delivered_at TIMESTAMP WITH TIME ZONE,
			notes TEXT
		)`,

		// Order items table - individual items in orders
		`CREATE TABLE order_items (
			item_id SERIAL PRIMARY KEY,
			order_id INTEGER REFERENCES orders(order_id),
			product_id INTEGER REFERENCES products(product_id),
			quantity INTEGER NOT NULL,
			unit_price DECIMAL(10,2) NOT NULL,
			total_price DECIMAL(12,2) NOT NULL,
			discount_applied DECIMAL(8,2) DEFAULT 0.00
		)`,

		// Reviews table - product reviews and ratings
		`CREATE TABLE reviews (
			review_id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(user_id),
			product_id INTEGER REFERENCES products(product_id),
			order_id INTEGER REFERENCES orders(order_id),
			rating INTEGER CHECK (rating >= 1 AND rating <= 5),
			title VARCHAR(255),
			content TEXT,
			helpful_votes INTEGER DEFAULT 0,
			total_votes INTEGER DEFAULT 0,
			is_verified_purchase BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// User sessions table - tracking user behavior
		`CREATE TABLE user_sessions (
			session_id_pk SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(user_id),
			session_id VARCHAR(255) UNIQUE,
			device_type VARCHAR(50),
			browser VARCHAR(100),
			operating_system VARCHAR(100),
			ip_address INET,
			started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			ended_at TIMESTAMP WITH TIME ZONE,
			duration_seconds INTEGER
		)`,

		// Product analytics table - tracking product interactions
		`CREATE TABLE product_analytics (
			analytics_id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(product_id),
			user_id INTEGER REFERENCES users(user_id),
			event_type VARCHAR(50), -- view, add_to_cart, purchase, wishlist
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(ctx, schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for performance
	indexes := []string{
		// Users indexes
		"CREATE INDEX idx_users_email ON users(email)",
		"CREATE INDEX idx_users_country_city ON users(country, city)",
		"CREATE INDEX idx_users_created_at ON users(created_at)",
		"CREATE INDEX idx_users_last_login ON users(last_login)",
		"CREATE INDEX idx_users_loyalty_points ON users(loyalty_points)",
		// "CREATE GIN INDEX idx_users_preferences ON users USING GIN(preferences)",

		// Products indexes
		"CREATE INDEX idx_products_sku ON products(sku)",
		// "CREATE INDEX idx_products_name ON products USING GIN(to_tsvector('english', name))",
		"CREATE INDEX idx_products_category ON products(category, subcategory)",
		"CREATE INDEX idx_products_brand ON products(brand)",
		"CREATE INDEX idx_products_price ON products(price)",
		"CREATE INDEX idx_products_rating ON products(avg_rating)",
		"CREATE INDEX idx_products_created_at ON products(created_at)",
		// "CREATE GIN INDEX idx_products_tags ON products USING GIN(tags)",
		// "CREATE GIN INDEX idx_products_attributes ON products USING GIN(attributes)",

		// Inventory indexes
		"CREATE INDEX idx_inventory_product_id ON inventory(product_id)",
		"CREATE INDEX idx_inventory_warehouse ON inventory(warehouse_location)",
		"CREATE INDEX idx_inventory_quantity ON inventory(quantity_available)",

		// Orders indexes
		"CREATE INDEX idx_orders_user_id ON orders(user_id)",
		"CREATE INDEX idx_orders_status ON orders(status)",
		"CREATE INDEX idx_orders_created_at ON orders(created_at)",
		"CREATE INDEX idx_orders_total_amount ON orders(total_amount)",
		"CREATE INDEX idx_orders_number ON orders(order_number)",

		// Order items indexes
		"CREATE INDEX idx_order_items_order_id ON order_items(order_id)",
		"CREATE INDEX idx_order_items_product_id ON order_items(product_id)",

		// Reviews indexes
		"CREATE INDEX idx_reviews_product_id ON reviews(product_id)",
		"CREATE INDEX idx_reviews_user_id ON reviews(user_id)",
		"CREATE INDEX idx_reviews_rating ON reviews(rating)",
		"CREATE INDEX idx_reviews_created_at ON reviews(created_at)",
		"CREATE INDEX idx_reviews_helpful ON reviews(helpful_votes)",

		// Sessions indexes
		"CREATE INDEX idx_sessions_user_id ON user_sessions(user_id)",
		"CREATE INDEX idx_sessions_started_at ON user_sessions(started_at)",
		"CREATE INDEX idx_sessions_device_type ON user_sessions(device_type)",

		// Analytics indexes
		"CREATE INDEX idx_analytics_product_id ON product_analytics(product_id)",
		"CREATE INDEX idx_analytics_user_id ON product_analytics(user_id)",
		"CREATE INDEX idx_analytics_event_type ON product_analytics(event_type)",
		"CREATE INDEX idx_analytics_created_at ON product_analytics(created_at)",
		"CREATE INDEX idx_analytics_search_query ON product_analytics(search_query)",
	}

	for _, index := range indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Continue creating other indexes
		}
	}

	return nil
}

// Cleanup drops all real-world tables
func (w *ECommerceBasicWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🧹 Cleaning up Real-World workload...")

	tables := []string{
		"product_analytics",
		"user_sessions",
		"reviews",
		"order_items",
		"orders",
		"inventory",
		"products",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	log.Printf("✅ Real-World cleanup complete")
	return nil
}

// Run executes the real-world workload based on the configured mode
func (w *ECommerceBasicWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🌍 Starting Real-World %s workload...", w.Mode)

	// Initialize worker-specific metrics tracking
	metrics.InitializeWorkerMetrics(cfg.Workers)

	// Initialize time-series metrics tracking
	metrics.InitializeTimeSeries(5 * time.Second) // 5-second buckets

	var wg sync.WaitGroup

	// Start real-time reporting (placeholder for future implementation)
	// stopReporting := w.startRealTimeReporter(ctx, cfg, metrics, start)
	// defer stopReporting()

	// Launch workers
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(ctx, db, cfg, metrics, workerID)
		}(i)
	}

	wg.Wait()

	// Finalize time-series collection for analysis
	metrics.FinalizeTimeSeries()

	return nil
}

// worker executes database operations based on the workload mode
func (w *ECommerceBasicWorkload) worker(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, workerID int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			opStart := time.Now()
			var err error
			var operation string

			switch w.Mode {
			case "read":
				operation, err = w.executeReadOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsRead, 1)
			case "write":
				operation = "write"
				err = w.executeWriteOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsModified, 1)
			case "mixed":
				if rng.Intn(100) < 75 { // 75% reads, 25% writes
					operation, err = w.executeReadOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsRead, 1)
				} else {
					operation = "write"
					err = w.executeWriteOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsModified, 1)
				}
			case "oltp":
				// OLTP workload: frequent small transactions
				if rng.Intn(100) < 60 { // 60% reads
					operation = "oltp_read"
					err = w.executeOLTPReadOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsRead, 1)
				} else { // 40% writes
					operation = "oltp_write"
					err = w.executeOLTPWriteOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsModified, 1)
				}
			case "analytics":
				// Analytics workload: complex analytical queries
				operation = "analytics"
				err = w.executeAnalyticsOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsRead, 10) // Analytics typically read many rows
			default:
				operation, err = w.executeReadOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsRead, 1)
			}

			elapsed := time.Since(opStart).Nanoseconds()

			// Record metrics
			metrics.Mu.Lock()
			metrics.TransactionDur = append(metrics.TransactionDur, elapsed)
			metrics.Mu.Unlock()

			metrics.RecordLatency(elapsed)

			if err != nil {
				atomic.AddInt64(&metrics.Errors, 1)
				metrics.Mu.Lock()
				metrics.ErrorTypes[fmt.Sprintf("%s: %s", operation, err.Error())]++
				metrics.Mu.Unlock()

				// Record worker-specific error and failed transaction
				metrics.RecordWorkerError(workerID)
				metrics.RecordWorkerTransaction(workerID, false, elapsed)
			} else {
				// Record query type for breakdown table
				switch operation {
				case "get_user_orders", "get_product_details", "get_product_reviews", "get_products_by_category", "get_top_products_by_category", "get_recent_activity", "get_product_analytics":
					metrics.RecordQuery("SELECT")
				case "write":
					// For write operations, randomly distribute between INSERT/UPDATE
					if rng.Intn(2) == 0 {
						metrics.RecordQuery("INSERT")
					} else {
						metrics.RecordQuery("UPDATE")
					}
				default:
					metrics.RecordQuery("SELECT") // Default to SELECT for other operations
				}

				// Record worker-specific transaction and query (this also handles global TPS recording)
				metrics.RecordWorkerTransaction(workerID, true, elapsed)
				metrics.RecordWorkerQuery(workerID, operation)
			} // Record time-series metrics
			metrics.RecordTimeSeriesTransaction(err == nil, elapsed, 1, 1)

			// Think time varies by workload type
			var thinkTime time.Duration
			switch w.Mode {
			case "oltp":
				thinkTime = time.Duration(rng.Intn(10)) * time.Millisecond // Fast OLTP
			case "analytics":
				thinkTime = time.Duration(rng.Intn(500)) * time.Millisecond // Slower analytics
			default:
				thinkTime = time.Duration(rng.Intn(50)) * time.Millisecond // Standard
			}
			time.Sleep(thinkTime)
		}
	}
}
