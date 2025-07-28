# Distribution and Package Management

This document explains how StormDB packages are automatically generated and distributed via GitHub.

## 🚀 **Automated Distribution Process**

### **How It Works**

1. **Tag Creation**: When you push a git tag (e.g., `v0.1.0-alpha.2`), GitHub Actions automatically triggers
2. **Multi-Platform Builds**: Binaries are built for all supported platforms
3. **Package Creation**: Linux packages (DEB/RPM) are generated using FPM
4. **Docker Images**: Multi-architecture container images are built and pushed
5. **GitHub Release**: All artifacts are automatically attached to a GitHub release
6. **Distribution**: Users can download via multiple channels

### **Triggering a Release**

```bash
# Create and push a new version tag
git tag v0.1.0-alpha.3
git push origin v0.1.0-alpha.3
```

The GitHub Actions workflow (`.github/workflows/release.yml`) will automatically:
- Build binaries for Linux, macOS, Windows (multiple architectures)
- Create DEB packages for Ubuntu/Debian
- Create RPM packages for CentOS/Fedora/Amazon Linux
- Build and push Docker images to GitHub Container Registry and Docker Hub
- Generate SHA256 checksums for all artifacts
- Create a GitHub release with all downloadable assets

## 📦 **Available Distribution Channels**

### **1. GitHub Releases** (Primary)
- **URL**: https://github.com/elchinoo/stormdb/releases
- **Content**: All binaries, packages, and checksums
- **Access**: Direct download links for all platforms

### **2. Container Registries**
- **GitHub Container Registry**: `ghcr.io/elchinoo/stormdb`
- **Docker Hub**: `docker.io/elchinoo/stormdb`
- **Architectures**: linux/amd64, linux/arm64

### **3. Package Repositories** (Future)
- **APT Repository**: For Ubuntu/Debian (planned)
- **YUM Repository**: For CentOS/RHEL/Fedora (planned)
- **Homebrew**: For macOS (community contribution welcome)
- **Chocolatey**: For Windows (community contribution welcome)

## 🌐 **User Download Options**

### **Quick Install Scripts**
```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/elchinoo/stormdb/main/install.sh | bash

# Windows PowerShell
iwr https://raw.githubusercontent.com/elchinoo/stormdb/main/install.ps1 | iex
```

### **Direct Downloads**

#### **Binaries**
- Linux: `https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-linux-amd64.tar.gz`
- macOS: `https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-darwin-amd64.tar.gz`
- Windows: `https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-windows-amd64.zip`

#### **Linux Packages**
- DEB: `https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb_0.1.0-alpha.2_amd64.deb`
- RPM: `https://github.com/elchinoo/stormdb/releases/download/v0.1.0-alpha.2/stormdb-0.1.0-alpha.2.x86_64.rpm`

#### **Docker Images**
```bash
docker pull ghcr.io/elchinoo/stormdb:latest
docker pull ghcr.io/elchinoo/stormdb:v0.1.0-alpha.2
```

## 🔧 **Build Matrix**

The automated build system creates artifacts for:

### **Binary Platforms**
| OS | Architecture | Format | Example |
|----|-------------|--------|---------|
| Linux | amd64 | tar.gz | `stormdb-linux-amd64.tar.gz` |
| Linux | arm64 | tar.gz | `stormdb-linux-arm64.tar.gz` |
| Linux | 386 | tar.gz | `stormdb-linux-386.tar.gz` |
| macOS | amd64 | tar.gz | `stormdb-darwin-amd64.tar.gz` |
| macOS | arm64 | tar.gz | `stormdb-darwin-arm64.tar.gz` |
| Windows | amd64 | zip | `stormdb-windows-amd64.zip` |
| Windows | 386 | zip | `stormdb-windows-386.zip` |

### **Linux Packages**  
| Distribution | Architecture | Format | Example |
|-------------|-------------|--------|---------|
| Ubuntu/Debian | amd64 | DEB | `stormdb_0.1.0-alpha.2_amd64.deb` |
| Ubuntu/Debian | arm64 | DEB | `stormdb_0.1.0-alpha.2_arm64.deb` |
| CentOS/RHEL/Fedora | amd64 | RPM | `stormdb-0.1.0-alpha.2.x86_64.rpm` |
| CentOS/RHEL/Fedora | arm64 | RPM | `stormdb-0.1.0-alpha.2.aarch64.rpm` |

### **Container Images**
| Registry | Image | Architectures |
|----------|-------|--------------|
| GitHub Container Registry | `ghcr.io/elchinoo/stormdb` | linux/amd64, linux/arm64 |
| Docker Hub | `elchinoo/stormdb` | linux/amd64, linux/arm64 |

## 📊 **Download Statistics**

GitHub provides download statistics for releases:
- **Location**: https://github.com/elchinoo/stormdb/releases
- **Metrics**: Download counts per asset
- **API**: Available via GitHub REST API

## 🔐 **Security and Verification**

### **Checksums**
Every release includes `SHA256SUMS` file with checksums for all artifacts:
```bash
# Verify download integrity
sha256sum -c SHA256SUMS --ignore-missing
```

### **GPG Signatures** (Future)
Planning to add GPG signatures for enhanced security:
- Release artifacts will be signed
- Public key will be available in repository
- Verification instructions in documentation

### **Container Image Scanning**
Docker images are automatically scanned for vulnerabilities using:
- GitHub Security Advisory Database
- Trivy vulnerability scanner
- Regular base image updates

## 📈 **Usage Analytics**

### **Download Metrics**
- GitHub release download counts
- Container registry pull statistics
- Geographic distribution (if available)

### **Version Adoption**
- Most popular versions
- Update patterns
- Platform preferences

## 🛠️ **Maintenance**

### **Adding New Platforms**
To add support for a new platform:

1. **Update Build Matrix**: Modify `.github/workflows/release.yml`
2. **Test Cross-Compilation**: Ensure Go supports the target platform
3. **Update Documentation**: Add platform to compatibility matrices
4. **Package Format**: Consider platform-specific packaging needs

### **Deprecating Platforms**
When removing platform support:

1. **Announce Deprecation**: Give users advance notice
2. **Update CI**: Remove from build matrix
3. **Documentation**: Update compatibility information
4. **Final Release**: Clearly mark last supported version

### **Version Management**
- **Semantic Versioning**: Follow semver for predictable releases
- **Pre-releases**: Use alpha/beta/rc tags for testing
- **LTS Versions**: Consider long-term support versions
- **Security Updates**: Backport critical fixes to supported versions

## 🤝 **Community Contributions**

### **Package Managers**
We welcome community contributions for additional package managers:
- **Homebrew Formula**: For macOS users
- **Chocolatey Package**: For Windows users
- **Snap Package**: For Ubuntu users
- **Flatpak**: For Linux desktop users
- **Nix Package**: For NixOS users

### **Distribution Partnerships**
Interested in official packages? We're open to collaborating with:
- Linux distribution maintainers
- Package repository operators
- Container registry providers

## 📞 **Support**

### **Distribution Issues**
For problems with downloads or packages:
- **GitHub Issues**: https://github.com/elchinoo/stormdb/issues
- **Security Issues**: https://github.com/elchinoo/stormdb/security/advisories

### **Package Requests**
Request new distribution methods:
- **Feature Requests**: Use GitHub Discussions
- **Community Forum**: For general questions
- **Direct Contact**: For partnership inquiries

---

**Documentation**: See [INSTALLATION.md](INSTALLATION.md) for detailed installation instructions.
**Build Guide**: See [BUILD_AND_RELEASE.md](BUILD_AND_RELEASE.md) for development information.
