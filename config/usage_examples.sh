#!/bin/bash

# Example: Using the new consolidated configuration templates
# This script demonstrates how to quickly set up different workload scenarios

echo "=== StormDB New Configuration Templates - Usage Examples ==="
echo

# Example 1: Quick TPC-C test
echo "Example 1: Quick TPC-C Test"
echo "  cp workload_tpcc.yaml my_tpcc_test.yaml"
echo "  # Edit database connection in my_tpcc_test.yaml"
echo "  # Run: ./pgstorm -config config/my_tpcc_test.yaml"
echo

# Example 2: E-commerce read-heavy workload
echo "Example 2: E-commerce Read-Heavy Test"
echo "  cp workload_ecommerce.yaml my_ecommerce_read_test.yaml"
echo "  # In my_ecommerce_read_test.yaml:"
echo "  #   1. Comment out the default 'ecommerce_mixed' section"
echo "  #   2. Uncomment the 'ecommerce_read' example section"
echo "  #   3. Configure database connection"
echo "  # Run: ./pgstorm -config config/my_ecommerce_read_test.yaml"
echo

# Example 3: Progressive scaling with analytics
echo "Example 3: Progressive Scaling with Full Analytics"
echo "  cp workload_tpcc.yaml my_progressive_tpcc.yaml"
echo "  # In my_progressive_tpcc.yaml:"
echo "  #   1. Uncomment the 'results_backend' section"
echo "  #   2. Uncomment the 'test_metadata' section"
echo "  #   3. Uncomment the 'progressive scaling' example"
echo "  #   4. Configure database connections"
echo "  # Run: ./pgstorm -config config/my_progressive_tpcc.yaml"
echo

# Example 4: Vector similarity testing
echo "Example 4: Vector Similarity Search"
echo "  cp workload_pgvector.yaml my_vector_test.yaml"
echo "  # Default is cosine similarity - just configure database and run"
echo "  # Run: ./pgstorm -config config/my_vector_test.yaml"
echo

echo "=== Key Benefits of New Templates ==="
echo "✅ One file per workload type instead of 39 separate files"
echo "✅ Multiple examples in each template with full context"
echo "✅ Comprehensive documentation and comments"
echo "✅ Consistent structure across all workload types"
echo "✅ Easy to switch between different scenarios"
echo "✅ All new features included (database backend, progressive scaling, etc.)"
echo

echo "=== Next Steps ==="
echo "1. Test the new templates with your workloads"
echo "2. Create custom configurations by copying and modifying templates"
echo "3. Once satisfied, run './cleanup_helper.sh' to see what old files can be removed"
echo "4. Consider backing up old configs before cleanup"
echo

echo "For more information, see: config/README.md"
