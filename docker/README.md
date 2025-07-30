# StormDB Package Testing with Docker

This directory contains Docker-based testing infrastructure for building and testing StormDB packages across multiple Linux distributions.

## Overview

The package testing system provides:
- **Automated package building** for DEB and RPM formats
- **Multi-distribution testing** (Ubuntu, Debian, CentOS)
- **Installation verification** on each target platform
- **Package linting** and validation
- **Comprehensive test reporting**

## Quick Start

### Test All Distributions
```bash
# Build and test packages on all supported distributions
make test-packages
```

### Test Specific Distribution
```bash
# Test only Ubuntu DEB package
make test-packages-ubuntu

# Test only Debian DEB package
make test-packages-debian

# Test only CentOS RPM package
make test-packages-centos
```

### Verbose Testing
```bash
# Run tests with detailed output
make test-packages-verbose
```

### Complete Release Testing
```bash
# Build packages and test them before release
make release-test
```

## Manual Usage

You can also run the test script directly:

```bash
# Test all distributions
./docker/test-packages.sh

# Test specific distributions
./docker/test-packages.sh --distro ubuntu --distro debian

# Skip initial build (use existing artifacts)
./docker/test-packages.sh --skip-build

# Verbose output
./docker/test-packages.sh --verbose

# Show help
./docker/test-packages.sh --help
```

## Docker Services

The system uses Docker Compose with the following services:

### ubuntu-deb
- **Base**: Ubuntu 22.04
- **Purpose**: Build and test DEB packages on Ubuntu
- **Package Type**: DEB
- **Tools**: dpkg, lintian, postgresql-client

### debian-deb
- **Base**: Debian 12
- **Purpose**: Build and test DEB packages on Debian
- **Package Type**: DEB
- **Tools**: dpkg, lintian, postgresql-client

### centos-rpm
- **Base**: CentOS 8
- **Purpose**: Build and test RPM packages on CentOS
- **Package Type**: RPM
- **Tools**: rpm-build, rpmlint, postgresql

## What Gets Tested

For each distribution, the testing process:

1. **Builds the container** with all necessary dependencies
2. **Compiles StormDB** binary and plugins
3. **Creates the package** (DEB or RPM)
4. **Installs the package** using system package manager
5. **Verifies installation**:
   - Binary is accessible (`stormdb --help`)
   - All plugins are installed
   - Configuration files are in place
   - Man page is available
   - Directory structure follows FHS
6. **Runs package linting** (lintian for DEB, rpmlint for RPM)
7. **Generates test report**

## Directory Structure

```
docker/
├── Dockerfile.ubuntu      # Ubuntu container definition
├── Dockerfile.debian      # Debian container definition
├── Dockerfile.centos      # CentOS container definition
├── docker-compose.yml     # Multi-service orchestration
├── test-packages.sh       # Main test orchestration script
├── test-results/          # Generated test reports and logs
│   ├── ubuntu-test.log    # Ubuntu test output
│   ├── debian-test.log    # Debian test output
│   ├── centos-test.log    # CentOS test output
│   ├── ubuntu-build.log   # Ubuntu build output
│   ├── debian-build.log   # Debian build output
│   ├── centos-build.log   # CentOS build output
│   └── summary.txt        # Overall test summary
└── README.md              # This file
```

## Test Results

After running tests, check the results:

```bash
# View test summary
cat docker/test-results/summary.txt

# View individual test logs
cat docker/test-results/ubuntu-test.log
cat docker/test-results/debian-test.log
cat docker/test-results/centos-test.log
```

## Package Validation

Each test verifies:

### Installation Structure
- ✅ Binary at `/usr/bin/stormdb`
- ✅ Plugins at `/usr/lib/stormdb/plugins/`
- ✅ Main config at `/etc/stormdb/config_tpcc.yaml`
- ✅ Examples at `/etc/stormdb/examples/`
- ✅ Man page at `/usr/share/man/man1/stormdb.1.gz`
- ✅ Documentation at `/usr/share/doc/stormdb/`

### Functionality
- ✅ Binary executes without errors
- ✅ Help system works
- ✅ Man page is accessible
- ✅ All plugins are loadable
- ✅ Configuration files are valid

### Package Quality
- ✅ Package metadata is complete
- ✅ Dependencies are correctly specified
- ✅ No linting errors (warnings acceptable)
- ✅ FHS compliance

## Troubleshooting

### Container Build Failures
```bash
# Check build logs
cat docker/test-results/ubuntu-build.log

# Rebuild specific container
docker-compose -f docker/docker-compose.yml build ubuntu-deb --no-cache
```

### Package Installation Failures
```bash
# Check test logs for installation errors
cat docker/test-results/ubuntu-test.log

# Run container interactively for debugging
docker-compose -f docker/docker-compose.yml run --rm ubuntu-deb /bin/bash
```

### Missing Dependencies
- Ensure Docker and docker-compose are installed
- Verify internet connectivity for package downloads
- Check disk space for container builds

## Integration with CI/CD

This testing system can be integrated into GitHub Actions or other CI/CD systems:

```yaml
# Example GitHub Actions usage
- name: Test packages
  run: make test-packages

- name: Upload test results
  uses: actions/upload-artifact@v3
  if: always()
  with:
    name: package-test-results
    path: docker/test-results/
```

## Performance Notes

- **First run**: Takes longer due to container building
- **Subsequent runs**: Faster due to Docker layer caching
- **Build context**: Optimized with `.dockerignore`
- **Parallel testing**: Docker Compose runs tests concurrently

## Contributing

When adding new distributions:

1. Create new `Dockerfile.{distro}` following existing patterns
2. Add service to `docker-compose.yml`
3. Update `test-packages.sh` script
4. Add corresponding Makefile target
5. Test thoroughly and update documentation

## Security Considerations

- Containers run with standard user privileges
- No network access required during testing
- Build context excludes sensitive files via `.dockerignore`
- Package installation tested in isolated environments
