# Version Management

This document explains how to manage StormDB product and plugin API versions.

## Current Version: 1.0.0

## Version Location

- `include/stormdb.h`

```c
#define STORMDB_VERSION "1.0.0"
#define STORMDB_API_VERSION "1.0"
```

The `--version` output includes the compile date/time.

## Updating Versions

Edit the defines in `include/stormdb.h`:

- `STORMDB_VERSION`: Semantic version string (Major.Minor.Patch)
- `STORMDB_API_VERSION`: Plugin API compatibility (Major.Minor)

### Version Numbering

- Major: Breaking changes to the product or interfaces
- Minor: Backward-compatible feature updates
- Patch: Backward-compatible bug fixes

### Plugin API Versioning

We recommend validating plugin API versions as follows (to be enforced by the plugin system):

- Plugin API Major must match Core API Major
- Plugin API Minor must be less than or equal to Core API Minor

If you introduce breaking plugin API changes, bump `STORMDB_API_VERSION` Major and document required changes for plugin authors.
