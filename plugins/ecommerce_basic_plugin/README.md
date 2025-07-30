# E-commerce Basic Workload Plugin

This plugin provides basic e-commerce platform workloads for StormDB with standard OLTP patterns and simplified business logic.

## Supported Workload Types

- `ecommerce_basic` - Mixed basic e-commerce workload 
- `ecommerce_basic_read` - Read-heavy basic e-commerce queries
- `ecommerce_basic_write` - Write-heavy basic e-commerce operations
- `ecommerce_basic_mixed` - Balanced read/write workload
- `ecommerce_basic_oltp` - OLTP-focused transactional workload
- `ecommerce_basic_analytics` - Analytics-focused reporting workload

## Building the Plugin

```bash
go build -buildmode=plugin -o ecommerce_basic_plugin.so main.go
```

## Configuration

Use any of the supported workload types in your StormDB configuration:

```yaml
workload: "ecommerce_basic_mixed"
scale: 1000  # Dataset size parameter
```

## Workload Characteristics

- **Basic E-commerce Model**: Simple B2C e-commerce without vendor/supplier complexity
- **7 Core Tables**: users, products, orders, inventory, reviews, user_sessions, product_analytics
- **Standard OLTP Patterns**: User management, product catalog, order processing, inventory tracking
- **Simplified Analytics**: Basic reporting without advanced features like vector search
- **Performance Testing**: Ideal for testing standard PostgreSQL performance without extensions
