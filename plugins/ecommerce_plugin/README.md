# E-commerce Workload Plugin

This plugin provides modern e-commerce platform workloads for StormDB with realistic shopping patterns and vector-powered recommendations.

## Supported Workload Types

- `ecommerce` - Mixed e-commerce workload (75% read, 25% write)
- `ecommerce_read` - Read-heavy operations (product searches, browsing)
- `ecommerce_write` - Write-heavy operations (orders, inventory updates)
- `ecommerce_mixed` - Balanced read/write workload
- `ecommerce_oltp` - Transaction-focused operations
- `ecommerce_analytics` - Business intelligence and reporting

## Requirements

- PostgreSQL with pgvector extension installed (for recommendation features)
- Minimum PostgreSQL version: 12.0

## Building the Plugin

```bash
go build -buildmode=plugin -o ecommerce_plugin.so main.go
```

## Configuration

Use any of the supported workload types in your StormDB configuration:

```yaml
workload: "ecommerce_mixed"
scale: 1000  # Number of products in catalog
```

## Features

- **Product Catalog**: Realistic product searches and browsing
- **Order Processing**: Shopping cart and checkout operations
- **User Reviews**: Review creation and similarity search
- **Inventory Management**: Stock level tracking and updates
- **Vendor Operations**: Automated purchase orders and vendor management
- **Analytics**: Real-time business intelligence queries
