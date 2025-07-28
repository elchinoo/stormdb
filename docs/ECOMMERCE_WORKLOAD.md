# E-Commerce Workload Plugin Documentation

## Overview

The E-Commerce workload plugin simulates a modern e-commerce platform with advanced features including automated vendor management, pgvector-powered similarity search, and intelligent stock control systems. This plugin provides comprehensive e-commerce workloads for realistic performance testing scenarios.

## Plugin Architecture

The E-Commerce workload is implemented as a **plugin** in StormDB's modular architecture:

- **Plugin Location**: `plugins/ecommerce_plugin/`
- **Binary Output**: `build/plugins/ecommerce_plugin.so`
- **Requirements**: PostgreSQL 12+ with pgvector extension
- **Auto-loading**: Automatically discovered when plugins are enabled

## Key Features

### 🛒 Comprehensive E-Commerce Schema
- **Users**: Customer profiles with demographics and purchasing history
- **Vendors**: Supplier management with performance tracking
- **Products**: Rich product catalog with vendor relationships
- **Inventory**: Real-time stock management with automated reordering
- **Orders**: Complete order processing with multi-item support
- **Reviews**: Product reviews with vector embeddings for similarity search

### 📦 Automated Stock Control System
- **Smart Reordering**: Automatic purchase order generation when stock falls below threshold
- **Cost Tracking**: Monitors vendor cost changes and adjusts selling prices
- **Price Management**: Maintains minimum 60% profit margin automatically
- **Vendor Integration**: Seamless purchase order processing and receipt tracking

### 🔍 Vector-Powered Search (pgvector)
- **Review Similarity**: Semantic search through product reviews using vector embeddings
- **Content Matching**: Advanced similarity search for customer recommendations
- **Fallback Support**: Graceful degradation to text search when pgvector unavailable

### 📊 Advanced Analytics
- **Customer Segmentation**: Behavioral analysis and user categorization
- **Sales Analytics**: Revenue trends and product performance metrics
- **Inventory Analytics**: Stock turnover and reorder pattern analysis
- **Vendor Performance**: Supplier rating and delivery performance tracking

## Database Schema

### Core Tables

#### users
```sql
- user_id (SERIAL PRIMARY KEY)
- email, username (UNIQUE)
- demographics (name, birth_date, gender, location)
- account_status, preferences (JSONB)
- loyalty_points, total_spent
- created_at, last_login
```

#### vendors
```sql
- vendor_id (SERIAL PRIMARY KEY)
- vendor_name, contact_info
- address (JSONB), payment_terms
- lead_time_days, min_order_amount
- rating, is_active
- created_at, updated_at
```

#### products
```sql
- product_id (SERIAL PRIMARY KEY)
- sku (UNIQUE), name, description
- category, subcategory, brand
- price, cost, margin_percent
- attributes (JSONB), tags (TEXT[])
- avg_rating, review_count, view_count
- vendor_id (FK to vendors)
```

#### inventory
```sql
- inventory_id (SERIAL PRIMARY KEY)
- product_id (FK), warehouse_location
- quantity_available, quantity_reserved
- reorder_level, max_stock_level
- supplier_id (FK to vendors)
- auto_reorder (triggers purchase orders)
- unit_cost, updated_at
```

### Stock Control Tables

#### purchase_orders
```sql
- po_id (SERIAL PRIMARY KEY)
- po_number (UNIQUE), vendor_id (FK)
- status (pending/sent/received/cancelled)
- total_amount, tax_amount, shipping_cost
- expected_delivery, actual_delivery
- created_by, notes
```

#### purchase_order_items
```sql
- po_item_id (SERIAL PRIMARY KEY)
- po_id (FK), product_id (FK)
- quantity_ordered, quantity_received
- unit_cost, total_cost
- received_at
```

### Customer Operations

#### orders
```sql
- order_id (SERIAL PRIMARY KEY)
- user_id (FK), order_number (UNIQUE)
- status, total_amount, shipping_cost
- tax_amount, discount_amount
- payment_method, addresses (JSONB)
- shipped_at, delivered_at
```

#### reviews
```sql
- review_id (SERIAL PRIMARY KEY)
- user_id (FK), product_id (FK), order_id (FK)
- rating, title, content
- content_vector (vector(1536)) -- pgvector column
- helpful_votes, total_votes
- is_verified_purchase
```

## Workload Modes

### Mixed Mode (ecommerce_mixed)
- **75% Read Operations**: Product searches, user queries, analytics
- **25% Write Operations**: Orders, reviews, inventory updates
- **Realistic Production Load**: Balanced workload simulating real e-commerce traffic

### Read Mode (ecommerce_read)
- **Product Catalog Browsing**: Category-based product searches
- **User Account Queries**: Order history, profile information
- **Analytics Queries**: Sales reports, inventory status
- **Vector Searches**: Review similarity and recommendations

### Write Mode (ecommerce_write)
- **Order Processing**: New orders with multiple items
- **Inventory Updates**: Stock level changes and adjustments
- **Review Submissions**: Product reviews with vector embeddings
- **Purchase Orders**: Vendor ordering and receipt processing

### Analytics Mode (ecommerce_analytics)
- **Customer Segmentation**: Complex behavioral analysis
- **Sales Analytics**: Revenue trends and product performance
- **Inventory Analytics**: Stock control and vendor performance
- **Complex Queries**: CTEs, window functions, multi-table joins

### OLTP Mode (ecommerce_oltp)
- **High-Frequency Transactions**: Fast, indexed lookups
- **Real-Time Operations**: Inventory checks, user authentication
- **Low Latency**: Optimized for transactional workloads

## Automated Stock Control

### Trigger-Based Automation
The workload includes PostgreSQL triggers that automatically:

1. **Monitor Inventory Levels**: Check stock against reorder thresholds
2. **Generate Purchase Orders**: Create orders to vendors when stock is low
3. **Update Costs and Prices**: Adjust selling prices when vendor costs change
4. **Process Receipts**: Update inventory when purchase orders are received

### Cost and Pricing Logic
- **Automatic Price Updates**: When vendor costs increase, selling prices adjust automatically
- **Profit Margin Protection**: Ensures minimum 60% markup on all products
- **Cost Tracking**: Maintains historical cost data for analysis

### Purchase Order Workflow
1. **Low Stock Detection**: Trigger fires when quantity ≤ reorder_level
2. **Vendor Selection**: Uses existing supplier relationships
3. **Order Generation**: Creates purchase order with optimal quantity
4. **Receipt Processing**: Updates inventory when goods are received
5. **Cost Analysis**: Adjusts product pricing if costs have changed

## Vector Search Integration

### pgvector Support
- **Review Embeddings**: 1536-dimensional vectors (OpenAI embedding size)
- **Similarity Search**: Cosine similarity for content matching
- **Indexing**: IVFFlat index for efficient vector queries
- **Fallback**: Graceful degradation to text search when pgvector unavailable

### Vector Operations
```sql
-- Find similar reviews
SELECT review_id, content, rating,
       content_vector <-> $1::vector AS distance
FROM reviews 
WHERE content_vector IS NOT NULL
ORDER BY content_vector <-> $1::vector
LIMIT 10;
```

## Configuration Examples

### Basic E-Commerce Mixed Workload
```yaml
host: "localhost"
port: 5432
database: "stormdb"
workload: "ecommerce_mixed"
workers: 10
duration: "60s"
scale: 1000
```

### High-Performance Analytics
```yaml
workload: "ecommerce_analytics"
workers: 3
duration: "5m"
scale: 5000
```

## Performance Considerations

### Indexing Strategy
- **Primary Keys**: All tables have efficient primary key indexes
- **Foreign Keys**: Comprehensive foreign key indexing for joins
- **Search Indexes**: Category, brand, and text search optimization
- **Vector Indexes**: IVFFlat indexes for vector similarity search
- **Composite Indexes**: Multi-column indexes for complex queries

### Query Optimization
- **Read Operations**: Mix of indexed and full-table scan queries
- **Write Operations**: Batched inserts and transaction optimization
- **Analytics**: CTEs and window functions for complex analysis
- **Vector Queries**: Optimized similarity search with distance functions

### Scalability Features
- **Batch Processing**: Efficient bulk data loading
- **Connection Pooling**: Optimized database connection management
- **Transaction Management**: Proper isolation levels and rollback handling
- **Error Handling**: Comprehensive error tracking and reporting

## Monitoring and Metrics

### Built-in Metrics
- **Transaction Rates**: TPS and QPS tracking
- **Latency Distribution**: Histogram-based latency analysis
- **Error Tracking**: Categorized error reporting
- **Operation Breakdown**: Per-operation type metrics

### Stock Control Monitoring
- **Reorder Alerts**: Automatic low-stock notifications
- **Purchase Order Status**: Real-time PO processing metrics
- **Cost Variance**: Price change impact analysis
- **Vendor Performance**: Delivery and quality metrics

## Getting Started

### Prerequisites
1. **PostgreSQL 12+**: Database server
2. **pgvector Extension**: Optional for vector features
3. **Go 1.19+**: For building from source

### Quick Start
```bash
# Build the application
go build -o stormdb ./cmd/stormdb

# Run mixed workload demo
./stormdb -config=./config/config_ecommerce_mixed.yaml

# Run interactive demo
./demo_ecommerce_workload.sh
```

### Schema Setup
The workload automatically:
1. Creates all required tables and indexes
2. Sets up stock control triggers and functions
3. Loads realistic sample data based on scale factor
4. Enables pgvector extension if available

## Sample Queries

### Customer Analysis
```sql
-- Customer segmentation
WITH customer_metrics AS (
  SELECT u.user_id, COUNT(o.order_id) as order_count,
         SUM(o.total_amount) as total_spent
  FROM users u LEFT JOIN orders o ON u.user_id = o.user_id
  GROUP BY u.user_id
)
SELECT 
  CASE 
    WHEN order_count = 0 THEN 'Inactive'
    WHEN total_spent > 1000 THEN 'VIP'
    ELSE 'Regular'
  END as segment,
  COUNT(*) as customer_count
FROM customer_metrics
GROUP BY 1;
```

### Inventory Analysis
```sql
-- Stock control report
SELECT p.name, i.quantity_available, i.reorder_level,
       v.vendor_name, po.status as po_status
FROM inventory i
JOIN products p ON i.product_id = p.product_id
LEFT JOIN vendors v ON i.supplier_id = v.vendor_id
LEFT JOIN purchase_orders po ON po.vendor_id = v.vendor_id
WHERE i.quantity_available <= i.reorder_level
ORDER BY i.quantity_available ASC;
```

### Vector Similarity
```sql
-- Find similar product reviews
SELECT r1.content, r2.content,
       r1.content_vector <-> r2.content_vector as similarity
FROM reviews r1, reviews r2
WHERE r1.review_id != r2.review_id
   AND r1.content_vector IS NOT NULL
   AND r2.content_vector IS NOT NULL
ORDER BY similarity ASC
LIMIT 10;
```

## Migration from RealWorld

The E-Commerce workload is a complete replacement for the previous "realworld" workload with the following improvements:

### Enhanced Features
- **Vendor Management**: Complete supplier relationship tracking
- **Automated Stock Control**: Intelligent reordering system
- **Vector Search**: Advanced similarity matching capabilities
- **Comprehensive Analytics**: Enhanced reporting and segmentation

### Schema Changes
- **Added Tables**: vendors, purchase_orders, purchase_order_items
- **Enhanced Columns**: Vector columns, margin tracking, automated flags
- **New Indexes**: Vector indexes, composite performance indexes
- **Triggers**: Automated stock control and pricing triggers

### Backward Compatibility
- All existing "realworld" configurations continue to work
- New "ecommerce" configurations provide enhanced functionality
- Gradual migration path available through configuration updates

## Troubleshooting

### Common Issues

#### pgvector Not Available
```
Warning: pgvector extension not available, review vectors will be disabled
```
**Solution**: Install pgvector extension or ignore warning for text-only search

#### Stock Control Triggers Failed
```
Warning: Failed to create stock control triggers
```
**Solution**: Check PostgreSQL permissions for creating functions and triggers

#### Connection Issues
```
Error: failed to connect to database
```
**Solution**: Verify database configuration and network connectivity

### Performance Tuning

#### High Latency
- Increase connection pool size
- Check index usage with EXPLAIN ANALYZE
- Consider partitioning for large datasets

#### Memory Usage
- Adjust vector index parameters
- Optimize query complexity
- Consider connection pooling configuration

## Support and Contributing

### Documentation
- Configuration examples in `./config/config_ecommerce_*.yaml`
- Interactive demo script: `./demo_ecommerce_workload.sh`
- Plugin documentation in `./plugins/ecommerce_plugin/README.md`
- Schema documentation in this README

### Plugin Development
- Plugin source code in `./plugins/ecommerce_plugin/`
- Plugin build instructions: `make build-ecommerce-plugin`
- Test configurations in `./config/`
- Plugin development guide in `./docs/PLUGIN_DEVELOPMENT.md`

---

*The E-Commerce workload provides a comprehensive, production-realistic simulation of modern e-commerce platforms with advanced features for testing PostgreSQL performance at scale.*
