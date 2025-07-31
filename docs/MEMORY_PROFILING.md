#!/bin/bash

# Memory Profiling Guide for StormDB
# This script demonstrates how to use the new memory profiling feature

echo "🔍 StormDB Memory Profiling Guide"
echo "================================="
echo ""

echo "1. Start StormDB with profiling enabled:"
echo "   ./build/stormdb --config hcp/config_tpcc_epas17.yaml --rebuild --scale 50 --profile"
echo ""

echo "2. The profiling server will start on port 6060 (default) and show:"
echo "   📊 Access profiling at: http://localhost:6060/debug/pprof/"
echo "   💾 Memory profile: http://localhost:6060/debug/pprof/heap"
echo "   ⚡ CPU profile: http://localhost:6060/debug/pprof/profile"
echo ""

echo "3. Monitor memory usage in real-time:"
echo "   📈 Memory: Alloc=123MB, TotalAlloc=456MB, Sys=789MB, NumGC=10"
echo ""

echo "4. Analyze memory usage with go tool pprof:"
echo "   # Heap memory analysis"
echo "   go tool pprof http://localhost:6060/debug/pprof/heap"
echo ""
echo "   # CPU profiling (30 second sample)"
echo "   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30"
echo ""

echo "5. Common pprof commands once connected:"
echo "   (pprof) top10         # Show top 10 memory consumers"
echo "   (pprof) list          # Show functions using most memory"
echo "   (pprof) web           # Generate SVG graph (requires graphviz)"
echo "   (pprof) png           # Generate PNG graph"
echo "   (pprof) help          # Show all commands"
echo ""

echo "6. Monitor memory during progressive scaling:"
echo "   Watch the memory stats output to see when OOM occurs:"
echo "   📈 Memory: Alloc=2048MB, TotalAlloc=3000MB, Sys=2100MB, NumGC=50"
echo ""

echo "7. Alternative profiling port:"
echo "   ./build/stormdb --profile --profile-port 8080 ..."
echo ""

echo "Example complete command:"
echo "./build/stormdb --config hcp/config_tpcc_epas17.yaml --rebuild --scale 50 --profile --workers 10 --duration 5m"
