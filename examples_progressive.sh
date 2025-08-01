#!/bin/bash

# StormDB Progressive Scaling Examples
# This script demonstrates various progressive scaling test scenarios

set -e

echo "🚀 StormDB Progressive Scaling Examples"
echo "======================================="

# Ensure stormdb binary exists
if [ ! -f "./pgstorm" ]; then
    echo "Building StormDB..."
    go build -o pgstorm ./cmd/pgstorm
fi

# Example 1: Basic Linear Progression
echo
echo "📊 Example 1: Basic Linear Progression"
echo "--------------------------------------"
echo "Running 5-band linear test with IMDB workload..."
echo "Command: ./pgstorm progressive --config config/config_imdb_mixed.yaml --strategy linear --bands 5 --test-duration 1m --verbose"
echo
echo "This will test with evenly distributed connection increases:"
echo "- Band 1: ~10 connections"  
echo "- Band 2: ~25 connections"
echo "- Band 3: ~40 connections"
echo "- Band 4: ~55 connections"
echo "- Band 5: ~70 connections"
echo
echo "Each band runs for 1 minute with 30s warmup periods."
echo "Press Enter to run or Ctrl+C to skip..."
read -r

./pgstorm progressive \
  --config config/config_imdb_mixed.yaml \
  --strategy linear \
  --bands 5 \
  --test-duration 1m \
  --warmup-time 30s \
  --verbose || echo "✗ Test failed or was interrupted"

echo
echo "✅ Example 1 completed!"

# Example 2: Exponential Scaling with Analysis
echo
echo "📈 Example 2: Exponential Scaling with Advanced Analysis"
echo "--------------------------------------------------------"
echo "Running exponential progression with comprehensive mathematical analysis..."
echo "Command: ./pgstorm progressive --config config/config_ecommerce_mixed.yaml --strategy exponential --bands 6 --enable-analysis --output results.json"
echo
echo "This will test with exponential connection increases:"
echo "- Band 1: ~5 connections"
echo "- Band 2: ~10 connections"  
echo "- Band 3: ~20 connections"
echo "- Band 4: ~40 connections"
echo "- Band 5: ~80 connections"
echo "- Band 6: ~160 connections"
echo
echo "Includes full mathematical analysis with queueing theory, derivatives, and regression."
echo "Press Enter to run or Ctrl+C to skip..."
read -r

./pgstorm progressive \
  --config config/config_ecommerce_mixed.yaml \
  --strategy exponential \
  --bands 6 \
  --min-connections 5 \
  --max-connections 160 \
  --test-duration 90s \
  --enable-analysis \
  --output results_exponential.json \
  --verbose || echo "✗ Test failed or was interrupted"

echo
echo "✅ Example 2 completed! Results saved to results_exponential.json"

# Example 3: Custom Fibonacci with HTML Report
echo
echo "🔢 Example 3: Fibonacci Progression with HTML Report"
echo "---------------------------------------------------"
echo "Running Fibonacci-based progression with HTML report generation..."
echo "Command: ./pgstorm progressive --config config/config_tpcc.yaml --strategy fibonacci --bands 7 --report report.html"
echo
echo "This will test with Fibonacci-based connection increases:"
echo "- Band 1: ~8 connections"
echo "- Band 2: ~13 connections"
echo "- Band 3: ~21 connections"
echo "- Band 4: ~34 connections"
echo "- Band 5: ~55 connections"
echo "- Band 6: ~89 connections"
echo "- Band 7: ~144 connections"
echo
echo "Generates an HTML report with charts and detailed analysis."
echo "Press Enter to run or Ctrl+C to skip..."
read -r

./pgstorm progressive \
  --config config/config_tpcc.yaml \
  --strategy fibonacci \
  --bands 7 \
  --min-connections 8 \
  --max-connections 144 \
  --test-duration 2m \
  --warmup-time 45s \
  --cooldown-time 20s \
  --enable-analysis \
  --report report.html \
  --verbose || echo "✗ Test failed or was interrupted"

echo
echo "✅ Example 3 completed! HTML report saved to report.html"

# Example 4: High-Intensity Load Testing
echo
echo "⚡ Example 4: High-Intensity Load Testing"
echo "----------------------------------------"
echo "Running high-intensity test to find system limits..."
echo "Command: ./pgstorm progressive --config config/config_showcase_all_features.yaml --strategy exponential --bands 8 --max-connections 500"
echo
echo "This aggressive test will:"
echo "- Start with 20 connections"
echo "- Scale exponentially to 500 connections"
echo "- Use longer test periods (3 minutes per band)"
echo "- Include early termination on excessive errors"
echo
echo "⚠️  WARNING: This test may saturate your database!"
echo "Ensure your PostgreSQL instance can handle high connection counts."
echo "Press Enter to run or Ctrl+C to skip..."
read -r

./pgstorm progressive \
  --config config/config_showcase_all_features.yaml \
  --strategy exponential \
  --bands 8 \
  --min-workers 10 \
  --max-workers 200 \
  --min-connections 20 \
  --max-connections 500 \
  --test-duration 3m \
  --warmup-time 60s \
  --enable-analysis \
  --output results_high_intensity.json \
  --report report_high_intensity.html \
  --verbose || echo "✗ Test failed or was interrupted (this is expected if system limits were reached)"

echo
echo "✅ Example 4 completed! Check results_high_intensity.json for detailed analysis."

# Summary
echo
echo "🎉 Progressive Scaling Examples Complete!"
echo "========================================"
echo
echo "Generated files:"
echo "- results_exponential.json: Exponential test results with mathematical analysis"
echo "- results_high_intensity.json: High-intensity load test results"
echo "- report.html: Fibonacci test HTML report with visualizations"
echo "- report_high_intensity.html: High-intensity test HTML report"
echo
echo "📊 Key Insights to Look For:"
echo "- Linear Scalability Score (>0.8 is excellent)"
echo "- Optimal connection ranges from scalability analysis"
echo "- Queueing theory metrics (utilization factor <0.9 recommended)"
echo "- Error rates and health scores across bands"
echo "- Performance breakpoints and bottleneck indicators"
echo
echo "📚 Next Steps:"
echo "1. Review the generated reports and JSON results"
echo "2. Analyze scalability scores and optimal configurations"
echo "3. Use insights to tune your PostgreSQL configuration"
echo "4. Run targeted tests around identified optimal ranges"
echo "5. Integrate progressive testing into your CI/CD pipeline"
echo
echo "For detailed documentation, see: docs/PROGRESSIVE_SCALING_GUIDE.md"
echo "Happy testing! 🚀"
