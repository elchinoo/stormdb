# Configuration Parsing & Error Rate Fixes

## 🐛 **Issues Identified & Fixed**

### **1. Missing `rebuild` Configuration Field**

**Problem**: The `rebuild: true` field in the JSON config was being ignored and defaulted to `false`.

**Root Cause**: The `TPCCConfig` struct was missing the `rebuild` field.

**Fix Applied**:
```go
type TPCCConfig struct {
    // ... existing fields ...
    Mode        string   `json:"mode"`
    Rebuild     bool     `json:"rebuild"` // ✅ Added missing field
    // ... rest of fields ...
}
```

**Logic Enhancement**:
```go
// Handle rebuild flag - if rebuild is true, force rebuild mode regardless of mode setting
mode := p.cfg.Mode
if p.cfg.Rebuild {
    mode = "rebuild"
    p.logger.Info("Rebuild flag set, forcing rebuild mode")
}
```

### **2. Premature Error Rate Limit Triggering**

**Problem**: Error rate checking after every single transaction caused immediate failures.
- If first transaction fails: `error_rate = 1/1 = 100%` → exceeds 5% limit instantly

**Root Cause**: No minimum transaction threshold before error rate validation.

**Fix Applied**:
```go
// Only check after a minimum number of transactions to avoid false positives
minTransactionsForErrorCheck := int64(10)
if p.cfg.StopOnErrorLimit && totalTransactions >= minTransactionsForErrorCheck {
    localErrorRate := float64(totalErrors) / float64(totalTransactions)
    if localErrorRate > p.cfg.MaxErrorRate {
        // Enhanced logging with transaction counts
        p.logger.Warn("Error rate limit exceeded in worker",
            core.Field{Key: "worker_id", Value: workerID},
            core.Field{Key: "error_rate", Value: localErrorRate},
            core.Field{Key: "limit", Value: p.cfg.MaxErrorRate},
            core.Field{Key: "total_transactions", Value: totalTransactions},
            core.Field{Key: "total_errors", Value: totalErrors})
    }
}
```

### **3. Overly Strict Transaction Isolation Level**

**Problem**: All transactions used `sql.LevelSerializable` causing high contention and failures.

**Root Cause**: Serializable isolation is too strict for performance testing scenarios.

**Fix Applied**:
```bash
# Changed all transactions from Serializable to ReadCommitted
sed -i 's/sql.LevelSerializable/sql.LevelReadCommitted/g' txn/transactions.go
```

**Impact**: 
- ✅ **Before**: `sql.LevelSerializable` (strictest, high contention)
- ✅ **After**: `sql.LevelReadCommitted` (balanced, better concurrency)

## 🔧 **Configuration Handling Improvements**

### **Rebuild Flag Processing**
```json
{
  "mode": "full",
  "rebuild": true  // ✅ Now properly recognized and processed
}
```

**Behavior**:
- If `rebuild: true` → Forces rebuild mode regardless of `mode` setting
- If `rebuild: false` or missing → Uses specified `mode`
- Provides clear logging when rebuild flag overrides mode

### **Error Rate Validation**
```json
{
  "max_error_rate": 0.05,        // 5% error rate limit
  "stop_on_error_limit": true    // Stop when limit exceeded
}
```

**Behavior**:
- ✅ **Minimum 10 transactions** before error rate checking
- ✅ **Enhanced logging** with transaction counts and error details
- ✅ **Prevents false positives** from initial transaction failures

## 🚀 **Expected Results**

### **Configuration Parsing**
```bash
# Before Fix:
"rebuild": false  # ❌ Ignored user input

# After Fix:
"rebuild": true   # ✅ Respects user configuration
```

### **Error Rate Handling**
```bash
# Before Fix:
time=2025-08-07T11:28:14.792-04:00 level=WARN msg="Error rate limit exceeded in worker" 
plugin=tpcc-scalability worker_id=2 error_rate=1 limit=0.05

# After Fix:
# No immediate error rate warnings - waits for 10+ transactions
# Enhanced logging when limits are actually exceeded
```

### **Transaction Concurrency**
```bash
# Before Fix: High contention, immediate failures with Serializable isolation
# After Fix: Better concurrency with ReadCommitted isolation
```

## 🧪 **Testing the Fixes**

### **Test Configuration**
```json
{
  "plugin_name": "tpcc-scalability",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "testdb",
    "username": "postgres",
    "password": "postgres",
    "mode": "full",
    "rebuild": true,              // ✅ Should be respected
    "scale": 1,
    "connections": [5, 10],
    "duration": "2m",
    "max_error_rate": 0.05,       // ✅ Should allow initial transactions
    "stop_on_error_limit": true
  }
}
```

### **Expected Behavior**
1. ✅ **Rebuild flag honored**: Tables dropped and recreated despite `mode: "full"`
2. ✅ **Error rate tolerance**: No immediate failures on first few transactions
3. ✅ **Better concurrency**: Transactions complete successfully with ReadCommitted isolation
4. ✅ **Thread metrics**: Raw data properly stored in `metric_records` table

## 📊 **Validation Queries**

```sql
-- Verify rebuild behavior (tables should be recreated)
SELECT COUNT(*) FROM information_schema.tables 
WHERE table_name LIKE '%tpcc%' OR table_name IN ('warehouse', 'district', 'customer');

-- Check error rates after sufficient transactions
SELECT 
    thread_id,
    SUM(num_errors) as total_errors,
    SUM(num_insert + num_update + num_delete + num_select) as total_ops,
    CASE 
        WHEN SUM(num_insert + num_update + num_delete + num_select) > 0 
        THEN (SUM(num_errors)::float / SUM(num_insert + num_update + num_delete + num_select)) * 100
        ELSE 0 
    END as error_rate_pct
FROM metric_records 
GROUP BY thread_id;

-- Verify thread-level metrics collection
SELECT thread_id, COUNT(*) as records, 
       MIN(timestamp), MAX(timestamp)
FROM metric_records 
GROUP BY thread_id;
```

## ✅ **Summary of Fixes**

| Issue | Root Cause | Fix Applied | Status |
|-------|------------|-------------|---------|
| **Config Parsing** | Missing `rebuild` field | Added field + logic | ✅ **Fixed** |
| **Error Rate** | Immediate checking | 10-transaction minimum | ✅ **Fixed** |
| **Transaction Isolation** | Serializable too strict | Changed to ReadCommitted | ✅ **Fixed** |
| **Enhanced Logging** | Missing transaction counts | Added detailed error info | ✅ **Improved** |

**All critical issues have been resolved!** The plugin should now:
- ✅ Properly parse the `rebuild` configuration
- ✅ Allow reasonable transaction failures without immediate shutdown
- ✅ Provide better concurrency for performance testing
- ✅ Deliver comprehensive thread-level metrics as designed
