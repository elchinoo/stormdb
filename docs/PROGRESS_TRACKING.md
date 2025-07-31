# Progress Tracking for Data Seeding Operations

## Overview

StormDB now includes comprehensive progress tracking for data seeding operations when using `--setup` and `--rebuild` flags. This feature provides real-time feedback with progress bars, completion percentages, insertion rates, and estimated time to completion (ETA).

## Features

### ✨ Visual Progress Bars
- **Real-time progress bars** with configurable width
- **Percentage completion** display
- **Current/Total counts** (e.g., "1,250/5,000")
- **Insertion rate** (items per second)
- **ETA calculation** based on current progress

### 🎯 Workload Coverage
Progress tracking is implemented across all major workloads:

- **TPC-C**: Warehouses → Districts → Customers
- **E-commerce**: Vendors → Users → Products → Orders → Reviews → Sessions
- **IMDB**: Actors → Movies → Cast Relationships → Comments → Viewing Logs → Voting History
- **Simple**: Basic table seeding

### 📊 Smart Batching
- **Batch progress tracking** for large datasets
- **Automatic display throttling** (updates every 100ms)
- **Memory-efficient** progress calculation

## Usage Examples

### Basic Setup with Progress Bars

```bash
# TPC-C workload setup - see progress for warehouses, districts, customers
./build/stormdb -c config/config_progress_demo_tpcc.yaml --setup

# E-commerce workload setup - see progress for vendors, users, products, orders
./build/stormdb -c config/config_progress_demo_ecommerce.yaml --setup

# IMDB workload setup - see progress for actors, movies, comments, logs
./build/stormdb -c config/config_progress_demo_imdb.yaml --setup
```

### Rebuild with Progress Tracking

```bash
# Complete rebuild - drops tables, recreates schema, loads data with progress bars
./build/stormdb -c config/config_progress_demo_tpcc.yaml --rebuild
```

## Progress Bar Examples

### TPC-C Data Seeding
```
📦 Loading warehouses: [████████████████████████████████████████████████] 5/5 (100.0%) (12.5/s) ✅ Completed in 400ms
🏢 Loading districts: [████████████████████████████████████████████████] 50/50 (100.0%) (125.0/s) ✅ Completed in 400ms
👥 Loading customers: [████████████████████████████████████████████████] 5000/5000 (100.0%) (2,500.0/s) ✅ Completed in 2.0s
```

### E-commerce Data Seeding
```
📦 Loading vendors: [████████████████████████████████████████████████] 25/25 (100.0%) (83.3/s) ✅ Completed in 300ms
👥 Loading users: [████████████████████████████████████████████████] 500/500 (100.0%) (1,000.0/s) ✅ Completed in 500ms
📦 Loading products: [████████████████████████████████████████████████] 250/250 (100.0%) (625.0/s) ✅ Completed in 400ms
📊 Loading orders: [████████████████████████████████████████████████] 1000/1000 (100.0%) (2,000.0/s) ✅ Completed in 500ms
```

### IMDB Data Seeding
```
👥 Loading actors: [████████████████████████████████████████████████] 500/500 (100.0%) (1,250.0/s) ✅ Completed in 400ms
🎬 Loading movies: [████████████████████████████████████████████████] 1000/1000 (100.0%) (2,000.0/s) ✅ Completed in 500ms
🔗 Creating movie-actor relationships: [████████████████████████████████████████████████] 1000/1000 (100.0%) (2,500.0/s) ✅ Completed in 400ms
📝 Loading user comments: [████████████████████████████████████████████████] 3000/3000 (100.0%) (3,750.0/s) ✅ Completed in 800ms
📺 Loading viewing logs: [████████████████████████████████████████████████] 2000/2000 (100.0%) (4,000.0/s) ✅ Completed in 500ms
🗳️ Loading voting history: [████████████████████████████████████████████████] 2000/2000 (100.0%) (4,000.0/s) ✅ Completed in 500ms
```

## Technical Implementation

### Progress Tracker API

```go
// Create a new progress tracker
tracker := progress.NewTracker("📦 Loading vendors", totalCount)

// Update progress (automatically displays if enough time has passed)
for i := 1; i <= totalCount; i++ {
    // ... do work ...
    tracker.Update(i)
}

// Manual completion (optional - Update() handles this automatically)
tracker.Finish()
```

### Batch Progress Tracking

```go
// For batch operations
batchTracker := progress.NewBatchTracker("👥 Loading users", totalItems, batchSize)

for batchNum := 1; batchNum <= totalBatches; batchNum++ {
    // ... process batch ...
    batchTracker.UpdateBatch(batchNum)
}
```

### Customization Options

```go
tracker := progress.NewTracker("Task", total).
    SetWidth(60).              // Custom progress bar width
    SetShowETA(false)          // Disable ETA display
```

## Benefits

### 🔍 **Visibility During Long Operations**
- **No more "hanging"** - always know what's happening
- **Accurate progress estimation** for planning
- **Rate monitoring** to identify performance issues

### 🚀 **Better User Experience**
- **Professional appearance** with unicode progress bars
- **Informative feedback** including rates and ETA
- **Non-intrusive updates** (throttled to avoid spam)

### 🛠️ **Development & Debugging**
- **Identify bottlenecks** in data loading
- **Monitor insertion rates** across different data types
- **Validate batch processing** efficiency

## Configuration Files

Three demo configurations are provided to showcase progress tracking:

- `config/config_progress_demo_tpcc.yaml` - TPC-C with 5 warehouses
- `config/config_progress_demo_ecommerce.yaml` - E-commerce with 500 users
- `config/config_progress_demo_imdb.yaml` - IMDB with 1,000 movies

These configurations use smaller scales to quickly demonstrate the progress bars without long wait times.

## Performance Impact

- **Minimal overhead**: Progress updates are throttled to every 100ms
- **Memory efficient**: Only stores current progress state
- **Non-blocking**: Display updates don't slow down data insertion
- **Automatic throttling**: Prevents terminal flooding during fast operations

## Future Enhancements

Planned improvements for progress tracking:

- **Multi-stage progress** for complex operations
- **Parallel progress tracking** for concurrent operations
- **Progress persistence** across interrupted operations
- **JSON progress output** for automated monitoring
- **Custom progress themes** and styling options
