# DEB Package Installation Issues and Fixes

## 🐛 Issues Identified in v0.1.0-alpha.29

### 1. **Postinstall Script Bash Syntax Errors**
```
/var/lib/dpkg/info/stormdb.postinst: 43: [[: not found
/var/lib/dpkg/info/stormdb.postinst: 48: [[: not found
/var/lib/dpkg/info/stormdb.postinst: 53: [[: not found
```

**Root Cause:** Script uses `[[ ]]` bash-specific syntax but may run with `/bin/sh` (dash) on some Ubuntu systems.

**Fix Applied:** Changed all `[[ ]]` to POSIX-compliant `[ ]` syntax.

### 2. **Missing Configuration File**
```
ls: cannot access '/var/lib/stormdb/config/stormdb.yaml': No such file or directory
```

**Root Cause:** The postinstall script only copies config if `/etc/stormdb` exists, but package creation might not include config files properly.

**Fix Applied:** 
- Enhanced postinstall script to create default config if none exists
- Improved Makefile and CI to properly package config files

### 3. **Incorrect Directory Structure**
```
drwxr-xr-x 2 stormdb stormdb 4096 Jul 29 06:25 {config,logs,plugins}/
```

**Root Cause:** CI script used brace expansion incorrectly in `mkdir -p package-deb/{usr/local/bin,usr/local/lib/stormdb/plugins,etc/stormdb,etc/systemd/system,var/lib/stormdb/config}` creating a literal directory named `{config,logs,plugins}`.

**Fix Applied:** 
- Separated mkdir commands in CI workflow
- Updated both Makefile and CI to create proper directory structure

## 🔧 Fixes Applied

### 1. **Enhanced Postinstall Script** (`scripts/postinstall.sh`)
```bash
# Fixed POSIX compatibility
if [ -d /usr/local/lib/stormdb/plugins ]; then
    chmod -R 755 /usr/local/lib/stormdb/plugins
fi

# Added fallback config creation
if [ ! -f /var/lib/stormdb/config/stormdb.yaml ]; then
    # Create a basic config if none exists
    cat > /var/lib/stormdb/config/stormdb.yaml << 'EOF'
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: ""
  dbname: "postgres"
  sslmode: "disable"
# ... etc
EOF
fi
```

### 2. **Fixed CI Package Creation** (`.github/workflows/ci.yml`)
```yaml
# Create package structure
mkdir -p package-deb/usr/local/bin
mkdir -p package-deb/usr/local/lib/stormdb/plugins
mkdir -p package-deb/etc/stormdb
mkdir -p package-deb/etc/systemd/system
mkdir -p package-deb/var/lib/stormdb/config
mkdir -p package-deb/var/lib/stormdb/logs
mkdir -p package-deb/var/lib/stormdb/plugins
```

### 3. **Enhanced Makefile** (`Makefile`)
Updated `release-package-deb` target to:
- Create proper directory structure
- Include plugins in package
- Add systemd service file
- Include postgresql-client dependency

## 🚑 Emergency Fix for Existing Installations

For systems with the broken v0.1.0-alpha.29 package:

### **Quick Fix Script**
```bash
# Download and run the fix script
curl -O https://raw.githubusercontent.com/elchinoo/stormdb/main/scripts/fix_deb_installation.sh
sudo bash fix_deb_installation.sh
```

### **Manual Fix Steps**
1. **Fix directory structure:**
```bash
sudo rm -rf "/var/lib/stormdb/{config,logs,plugins}"
sudo mkdir -p /var/lib/stormdb/{config,logs,plugins}
sudo chown -R stormdb:stormdb /var/lib/stormdb
```

2. **Create default config:**
```bash
sudo -u stormdb tee /var/lib/stormdb/config/stormdb.yaml << 'EOF'
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: ""
  dbname: "postgres"
  sslmode: "disable"

workload:
  type: "basic"
  duration: "30s"
  workers: 4

metrics:
  enabled: true
  interval: "5s"
EOF
```

3. **Test installation:**
```bash
stormdb --help
stormdb -c /var/lib/stormdb/config/stormdb.yaml --help
```

## 🛡️ Prevention Measures

### **Testing Checklist for Future Releases**
- [ ] Test DEB package on fresh Ubuntu 24.04 LTS
- [ ] Verify postinstall script with both bash and dash
- [ ] Check directory structure after installation
- [ ] Verify configuration file creation
- [ ] Test systemd service functionality
- [ ] Validate binary permissions and PATH

### **Improved Package Testing**
```bash
# Test package in clean container
docker run -it ubuntu:24.04 bash
apt update && apt install -y ./stormdb_*.deb
ls -la /var/lib/stormdb/
stormdb --help
```

## 📋 Post-Fix Validation

After applying fixes, verify:
1. ✅ No bash syntax errors in postinstall
2. ✅ Configuration file exists and is readable  
3. ✅ Directory structure is correct
4. ✅ Binary is executable and in PATH
5. ✅ SystemD service can be enabled/started
6. ✅ User `stormdb` exists with proper permissions

## 🚀 Future Package Releases

These fixes will be included in the next release (v0.1.0-alpha.30+) to ensure:
- POSIX-compliant postinstall scripts
- Proper directory structure creation
- Reliable configuration file setup
- Better error handling and fallbacks
