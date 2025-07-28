// internal/workload/ecommerce/ecommerce.go
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

// ECommerceWorkload simulates a modern e-commerce platform with advanced features
// This includes: users, products, orders, inventory, reviews with vectors, vendor management, and automated stock control
type ECommerceWorkload struct {
	Mode string
}

// GetName returns the workload name
func (w *ECommerceWorkload) GetName() string {
	return "ecommerce_" + w.Mode
}

// Setup creates the e-commerce schema and loads sample data
func (w *ECommerceWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🛒 Setting up E-Commerce workload...")

	// Check if schema already exists
	var tableCount int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('users', 'products', 'orders', 'inventory', 'reviews', 'user_sessions', 'product_analytics', 'vendors', 'purchase_orders', 'purchase_order_items')
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	if tableCount == 10 {
		log.Printf("✅ E-Commerce schema already exists")
	} else {
		log.Printf("📊 Creating E-Commerce schema...")
		if err := w.createSchema(ctx, db); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		log.Printf("✅ E-Commerce schema created successfully")
	}

	// Load sample data if tables are empty
	var userCount int64
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if userCount == 0 {
		log.Printf("📊 Generating E-Commerce sample data...")
		if err := w.loadSampleData(ctx, db, cfg.Scale); err != nil {
			return fmt.Errorf("failed to load sample data: %w", err)
		}
	} else {
		log.Printf("✅ E-Commerce data already exists (%d users)", userCount)
	}

	return nil
}

// createSchema creates all the tables for the e-commerce workload
func (w *ECommerceWorkload) createSchema(ctx context.Context, db *pgxpool.Pool) error {
	// First check if pgvector extension is available
	log.Printf("📦 Checking for pgvector extension...")
	if err := w.enablePgVector(ctx, db); err != nil {
		log.Printf("⚠️  Warning: pgvector extension not available, review vectors will be disabled: %v", err)
	}

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

		// Vendors table - supplier management
		`CREATE TABLE vendors (
			vendor_id SERIAL PRIMARY KEY,
			vendor_name VARCHAR(255) NOT NULL,
			contact_email VARCHAR(255),
			contact_phone VARCHAR(50),
			address JSONB,
			payment_terms VARCHAR(100),
			lead_time_days INTEGER DEFAULT 7,
			min_order_amount DECIMAL(10,2) DEFAULT 0.00,
			rating DECIMAL(3,2) DEFAULT 5.00,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
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
			margin_percent DECIMAL(5,2) DEFAULT 60.00,
			weight_kg DECIMAL(8,3),
			dimensions JSONB, -- {width, height, depth}
			tags TEXT[],
			attributes JSONB, -- flexible product attributes
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT true,
			avg_rating DECIMAL(3,2) DEFAULT 0.00,
			review_count INTEGER DEFAULT 0,
			view_count INTEGER DEFAULT 0,
			vendor_id INTEGER REFERENCES vendors(vendor_id)
		)`,

		// Inventory table - stock management with automated reorder points
		`CREATE TABLE inventory (
			inventory_id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES products(product_id),
			warehouse_location VARCHAR(100),
			quantity_available INTEGER NOT NULL,
			quantity_reserved INTEGER DEFAULT 0,
			reorder_level INTEGER DEFAULT 10,
			max_stock_level INTEGER DEFAULT 100,
			last_restocked TIMESTAMP WITH TIME ZONE,
			supplier_id INTEGER REFERENCES vendors(vendor_id),
			unit_cost DECIMAL(10,2),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			auto_reorder BOOLEAN DEFAULT true
		)`,

		// Purchase Orders table - orders to vendors
		`CREATE TABLE purchase_orders (
			po_id SERIAL PRIMARY KEY,
			po_number VARCHAR(50) UNIQUE NOT NULL,
			vendor_id INTEGER REFERENCES vendors(vendor_id),
			status VARCHAR(50) DEFAULT 'pending', -- pending, sent, received, cancelled
			total_amount DECIMAL(12,2) NOT NULL,
			tax_amount DECIMAL(8,2) DEFAULT 0.00,
			shipping_cost DECIMAL(8,2) DEFAULT 0.00,
			expected_delivery DATE,
			actual_delivery DATE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			created_by VARCHAR(100), -- system or user who created the PO
			notes TEXT
		)`,

		// Purchase Order Items table - individual items in purchase orders
		`CREATE TABLE purchase_order_items (
			po_item_id SERIAL PRIMARY KEY,
			po_id INTEGER REFERENCES purchase_orders(po_id),
			product_id INTEGER REFERENCES products(product_id),
			quantity_ordered INTEGER NOT NULL,
			quantity_received INTEGER DEFAULT 0,
			unit_cost DECIMAL(10,2) NOT NULL,
			total_cost DECIMAL(12,2) NOT NULL,
			received_at TIMESTAMP WITH TIME ZONE
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

		// Reviews table - product reviews and ratings with vector content
		`CREATE TABLE reviews (
			review_id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(user_id),
			product_id INTEGER REFERENCES products(product_id),
			order_id INTEGER REFERENCES orders(order_id),
			rating INTEGER CHECK (rating >= 1 AND rating <= 5),
			title VARCHAR(255),
			content TEXT,
			content_vector vector(1536), -- pgvector column for semantic search (OpenAI embeddings size)
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
			search_query VARCHAR(255),
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

		// Vendors indexes
		"CREATE INDEX idx_vendors_name ON vendors(vendor_name)",
		"CREATE INDEX idx_vendors_rating ON vendors(rating)",
		"CREATE INDEX idx_vendors_active ON vendors(is_active)",

		// Products indexes
		"CREATE INDEX idx_products_sku ON products(sku)",
		"CREATE INDEX idx_products_category ON products(category, subcategory)",
		"CREATE INDEX idx_products_brand ON products(brand)",
		"CREATE INDEX idx_products_price ON products(price)",
		"CREATE INDEX idx_products_rating ON products(avg_rating)",
		"CREATE INDEX idx_products_created_at ON products(created_at)",
		"CREATE INDEX idx_products_vendor_id ON products(vendor_id)",

		// Inventory indexes
		"CREATE INDEX idx_inventory_product_id ON inventory(product_id)",
		"CREATE INDEX idx_inventory_warehouse ON inventory(warehouse_location)",
		"CREATE INDEX idx_inventory_quantity ON inventory(quantity_available)",
		"CREATE INDEX idx_inventory_reorder ON inventory(quantity_available, reorder_level) WHERE auto_reorder = true",
		"CREATE INDEX idx_inventory_supplier_id ON inventory(supplier_id)",

		// Purchase Orders indexes
		"CREATE INDEX idx_purchase_orders_vendor_id ON purchase_orders(vendor_id)",
		"CREATE INDEX idx_purchase_orders_status ON purchase_orders(status)",
		"CREATE INDEX idx_purchase_orders_created_at ON purchase_orders(created_at)",
		"CREATE INDEX idx_purchase_orders_expected_delivery ON purchase_orders(expected_delivery)",

		// Purchase Order Items indexes
		"CREATE INDEX idx_po_items_po_id ON purchase_order_items(po_id)",
		"CREATE INDEX idx_po_items_product_id ON purchase_order_items(product_id)",

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

	// Create vector index for reviews if pgvector is available
	vectorIndex := `CREATE INDEX idx_reviews_content_vector ON reviews USING ivfflat (content_vector vector_cosine_ops) WITH (lists = 100)`
	if _, err := db.Exec(ctx, vectorIndex); err != nil {
		log.Printf("Warning: Failed to create vector index (pgvector might not be available): %v", err)
	}

	// Create triggers for automated stock control
	if err := w.createStockControlTriggers(ctx, db); err != nil {
		log.Printf("Warning: Failed to create stock control triggers: %v", err)
	}

	return nil
}

// enablePgVector attempts to enable the pgvector extension
func (w *ECommerceWorkload) enablePgVector(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	return err
}

// createStockControlTriggers creates triggers for automated stock management
func (w *ECommerceWorkload) createStockControlTriggers(ctx context.Context, db *pgxpool.Pool) error {
	// Function to check stock and create purchase orders
	stockControlFunction := `
	CREATE OR REPLACE FUNCTION check_stock_and_reorder()
	RETURNS TRIGGER AS $$
	DECLARE
		vendor_record RECORD;
		po_number VARCHAR(50);
		new_po_id INTEGER;
		reorder_quantity INTEGER;
		new_unit_cost DECIMAL(10,2);
		new_price DECIMAL(10,2);
	BEGIN
		-- Only proceed if auto_reorder is enabled and quantity is at or below reorder level
		IF NEW.auto_reorder = true AND NEW.quantity_available <= NEW.reorder_level THEN
			-- Get vendor information
			SELECT v.* INTO vendor_record 
			FROM vendors v 
			WHERE v.vendor_id = NEW.supplier_id AND v.is_active = true;
			
			IF vendor_record.vendor_id IS NOT NULL THEN
				-- Calculate reorder quantity (difference between max stock and current stock)
				reorder_quantity := GREATEST(NEW.max_stock_level - NEW.quantity_available, 10);
				
				-- Generate PO number
				po_number := 'PO-' || to_char(NOW(), 'YYYYMMDD') || '-' || LPAD(nextval('purchase_orders_po_id_seq')::text, 6, '0');
				
				-- Simulate price increase (5% chance of 10-30% increase)
				new_unit_cost := NEW.unit_cost;
				IF random() < 0.05 THEN
					new_unit_cost := NEW.unit_cost * (1.1 + random() * 0.2); -- 10-30% increase
				END IF;
				
				-- Create purchase order
				INSERT INTO purchase_orders (
					po_number, vendor_id, status, total_amount, 
					expected_delivery, created_at, created_by, notes
				) VALUES (
					po_number, vendor_record.vendor_id, 'pending', 
					new_unit_cost * reorder_quantity,
					CURRENT_DATE + INTERVAL '1 day' * vendor_record.lead_time_days,
					NOW(), 'system', 
					'Auto-generated reorder for product ID: ' || NEW.product_id
				) RETURNING po_id INTO new_po_id;
				
				-- Create purchase order item
				INSERT INTO purchase_order_items (
					po_id, product_id, quantity_ordered, unit_cost, total_cost
				) VALUES (
					new_po_id, NEW.product_id, reorder_quantity, new_unit_cost, 
					new_unit_cost * reorder_quantity
				);
				
				-- If cost increased, update product cost and price
				IF new_unit_cost > NEW.unit_cost THEN
					-- Update inventory cost
					NEW.unit_cost := new_unit_cost;
					
					-- Calculate new selling price (ensure at least 60% margin)
					SELECT margin_percent INTO new_price FROM products WHERE product_id = NEW.product_id;
					new_price := new_unit_cost * (1 + COALESCE(new_price, 60.00) / 100.0);
					
					-- Update product price
					UPDATE products 
					SET price = new_price, cost = new_unit_cost, updated_at = NOW()
					WHERE product_id = NEW.product_id;
				END IF;
				
				RAISE NOTICE 'Auto-generated purchase order % for product % (quantity: %)', po_number, NEW.product_id, reorder_quantity;
			END IF;
		END IF;
		
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;`

	if _, err := db.Exec(ctx, stockControlFunction); err != nil {
		return fmt.Errorf("failed to create stock control function: %w", err)
	}

	// Create trigger
	stockControlTrigger := `
	CREATE OR REPLACE TRIGGER trigger_stock_reorder
		BEFORE UPDATE ON inventory
		FOR EACH ROW
		EXECUTE FUNCTION check_stock_and_reorder();`

	if _, err := db.Exec(ctx, stockControlTrigger); err != nil {
		return fmt.Errorf("failed to create stock control trigger: %w", err)
	}

	// Function to handle purchase order receipts
	receiptFunction := `
	CREATE OR REPLACE FUNCTION handle_purchase_receipt()
	RETURNS TRIGGER AS $$
	BEGIN
		-- When quantity_received is updated, update inventory
		IF NEW.quantity_received > OLD.quantity_received THEN
			UPDATE inventory 
			SET quantity_available = quantity_available + (NEW.quantity_received - OLD.quantity_received),
				last_restocked = NOW(),
				updated_at = NOW()
			WHERE product_id = NEW.product_id;
			
			-- Mark as received if fully received
			IF NEW.quantity_received >= NEW.quantity_ordered THEN
				NEW.received_at := NOW();
				
				-- Update PO status if all items are received
				UPDATE purchase_orders 
				SET status = 'received', actual_delivery = CURRENT_DATE, updated_at = NOW()
				WHERE po_id = NEW.po_id 
				AND NOT EXISTS (
					SELECT 1 FROM purchase_order_items 
					WHERE po_id = NEW.po_id 
					AND quantity_received < quantity_ordered
				);
			END IF;
		END IF;
		
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;`

	if _, err := db.Exec(ctx, receiptFunction); err != nil {
		return fmt.Errorf("failed to create receipt function: %w", err)
	}

	receiptTrigger := `
	CREATE OR REPLACE TRIGGER trigger_purchase_receipt
		BEFORE UPDATE ON purchase_order_items
		FOR EACH ROW
		EXECUTE FUNCTION handle_purchase_receipt();`

	if _, err := db.Exec(ctx, receiptTrigger); err != nil {
		return fmt.Errorf("failed to create receipt trigger: %w", err)
	}

	return nil
}

// Cleanup drops all e-commerce tables
func (w *ECommerceWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🧹 Cleaning up E-Commerce workload...")

	// Drop triggers first
	triggers := []string{
		"DROP TRIGGER IF EXISTS trigger_stock_reorder ON inventory",
		"DROP TRIGGER IF EXISTS trigger_purchase_receipt ON purchase_order_items",
		"DROP FUNCTION IF EXISTS check_stock_and_reorder()",
		"DROP FUNCTION IF EXISTS handle_purchase_receipt()",
	}

	for _, trigger := range triggers {
		if _, err := db.Exec(ctx, trigger); err != nil {
			log.Printf("Warning: Failed to drop trigger/function: %v", err)
		}
	}

	tables := []string{
		"product_analytics",
		"user_sessions",
		"reviews",
		"order_items",
		"orders",
		"purchase_order_items",
		"purchase_orders",
		"inventory",
		"products",
		"vendors",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	log.Printf("✅ E-Commerce cleanup complete")
	return nil
}

// Run executes the e-commerce workload based on the configured mode
func (w *ECommerceWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🛒 Starting E-Commerce %s workload...", w.Mode)

	// Initialize per-worker metrics tracking
	metrics.InitializeWorkerMetrics(cfg.Workers)

	var wg sync.WaitGroup

	// Launch workers
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(ctx, db, cfg, metrics, workerID)
		}(i)
	}

	wg.Wait()
	return nil
}

// worker executes database operations based on the workload mode
func (w *ECommerceWorkload) worker(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, workerID int) {
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

			if err != nil {
				atomic.AddInt64(&metrics.Errors, 1)
				metrics.Mu.Lock()
				metrics.ErrorTypes[fmt.Sprintf("%s: %s", operation, err.Error())]++
				metrics.Mu.Unlock()

				// Record worker-specific metrics for failed transaction
				metrics.RecordWorkerTransaction(workerID, false, elapsed)
				metrics.RecordWorkerError(workerID)
			} else {
				// Record worker-specific metrics for successful transaction
				metrics.RecordWorkerTransaction(workerID, true, elapsed)

				// For now, estimate queries per transaction based on operation type
				switch operation {
				case "write", "oltp_write":
					// Typical write operations have INSERT + SELECT
					metrics.RecordWorkerQuery(workerID, "INSERT")
					metrics.RecordWorkerQuery(workerID, "SELECT")
				case "analytics":
					// Complex analytics queries
					metrics.RecordWorkerQuery(workerID, "SELECT")
					metrics.RecordWorkerQuery(workerID, "SELECT")
					metrics.RecordWorkerQuery(workerID, "SELECT")
				default:
					// Read operations typically have one SELECT
					metrics.RecordWorkerQuery(workerID, "SELECT")
				}

				// Add some variation for mixed operations
				if w.Mode == "mixed" && operation == "write" {
					metrics.RecordWorkerQuery(workerID, "UPDATE") // Mixed write might include updates
				}
			}

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
