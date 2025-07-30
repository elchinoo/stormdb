# Plugin Rename: realworld_plugin → ecommerce_basic_plugin

## Overview

The `realworld_plugin` has been renamed to `ecommerce_basic_plugin` to better reflect its purpose as a simplified e-commerce workload for basic OLTP testing patterns.

## Rationale

The original name "realworld" was misleading because:
- The full `ecommerce_plugin` is actually more "real-world" with advanced B2B features
- The "realworld" plugin provides basic e-commerce patterns without complex vendor management
- The new name clearly differentiates it as the simpler e-commerce implementation

## Changes Made

### 1. Plugin Directory and Files
- **Directory**: `plugins/realworld_plugin/` → `plugins/ecommerce_basic_plugin/`
- **Main File**: `realworld.go` → `ecommerce_basic.go`
- **Binary Output**: `realworld_plugin.so` → `ecommerce_basic_plugin.so`

### 2. Code Changes
- **Struct**: `RealWorldWorkload` → `ECommerceBasicWorkload`
- **Plugin Name**: `"realworld"` → `"ecommerce_basic"`
- **Workload Types**: All `realworld*` → `ecommerce_basic*`
  - `realworld` → `ecommerce_basic`
  - `realworld_read` → `ecommerce_basic_read`
  - `realworld_write` → `ecommerce_basic_write` 
  - `realworld_mixed` → `ecommerce_basic_mixed`
  - `realworld_oltp` → `ecommerce_basic_oltp`
  - `realworld_analytics` → `ecommerce_basic_analytics`

### 3. Configuration Files
- **Main Config**: `config/config_realworld.yaml` → `config/config_ecommerce_basic.yaml`
- **HCP Configs**: All `config_realworld_*.yaml` → `config_ecommerce_basic_*.yaml`
- **Script References**: Updated all HCP bash scripts to use `ecommerce_basic`

### 4. Build System
- **Makefile**: Updated to build `ecommerce_basic_plugin.so`
- **Build Scripts**: Updated `build_plugins.sh` references
- **Plugin Loading**: Updated plugin discovery and loading

### 5. Documentation
- **README.md**: Updated plugin descriptions and examples
- **Architecture diagrams**: Updated visual representations
- **Release notes**: Updated feature descriptions
- **HCP documentation**: Updated all testing guides

## Current Plugin Lineup

| Plugin | Purpose | Operations | Tables | Features |
|--------|---------|------------|--------|----------|
| **ecommerce_plugin.so** | Full-featured e-commerce | 20 | 10 | B2B, vendors, pgvector, AI |
| **ecommerce_basic_plugin.so** | Basic e-commerce OLTP | 14 | 7 | Standard patterns, B2C only |
| **imdb_plugin.so** | Movie database workloads | Various | Various | Entertainment data |
| **vector_plugin.so** | Vector similarity search | Various | Various | pgvector operations |

## Migration Guide

### For Existing Configurations
Replace workload references:
```yaml
# Old
workload: "realworld_mixed"

# New  
workload: "ecommerce_basic_mixed"
```

### For HCP Testing
Configuration files have been automatically updated:
- `hcp/config_ecommerce_basic_epas16.yaml`
- `hcp/config_ecommerce_basic_epas17.yaml` 
- `hcp/config_ecommerce_basic_pge16.yaml`
- `hcp/config_ecommerce_basic_pge17.yaml`

### For Build Scripts
The build system automatically includes the new plugin:
```bash
make build-all  # Builds ecommerce_basic_plugin.so
```

## Verification

The rename has been verified with:
- ✅ Plugin builds successfully
- ✅ Plugin loads and runs correctly (370+ TPS)
- ✅ All configuration files updated
- ✅ Documentation consistency maintained
- ✅ HCP testing infrastructure working

## Backward Compatibility

**Note**: This is a breaking change. Old `realworld` configurations will need to be updated to use `ecommerce_basic` workload types.

The change improves clarity and maintainability by having:
- **ecommerce_plugin**: Advanced e-commerce with B2B features
- **ecommerce_basic_plugin**: Simple e-commerce with standard OLTP patterns

This naming convention better reflects the actual capabilities and use cases of each plugin.
