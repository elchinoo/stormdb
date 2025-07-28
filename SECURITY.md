# Security Policy

## Supported Versions

We actively support the following versions of StormDB with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| 0.9.x   | :x:                |
| < 0.9   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these guidelines:

### Do Not
- **Do not** file a public GitHub issue for security vulnerabilities
- **Do not** discuss the vulnerability in public forums or social media
- **Do not** attempt to exploit the vulnerability against systems you don't own

### Do
1. **Email us directly** at security@stormdb.org (GPG key available upon request)
2. **Provide detailed information** about the vulnerability:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact assessment
   - Suggested fix (if available)
3. **Allow reasonable time** for us to respond and fix the issue before disclosure

### Response Timeline
- **Initial Response**: Within 48 hours of report
- **Assessment**: Within 7 days of report
- **Fix Timeline**: Critical issues within 30 days, others within 90 days
- **Disclosure**: Coordinated disclosure after fix is available

## Security Best Practices

### When Using StormDB

#### Database Security
- Use dedicated test databases, never production data
- Create isolated database users with minimal privileges
- Use connection encryption (SSL/TLS) when testing over networks
- Rotate database credentials regularly

#### Configuration Security
```yaml
# Use environment variables for sensitive data
database:
  host: ${STORMDB_DB_HOST}
  username: ${STORMDB_DB_USER}
  password: ${STORMDB_DB_PASSWORD}
  sslmode: require  # Enable SSL/TLS
```

#### Network Security
- Run tests in isolated network environments
- Use firewalls to restrict database access
- Monitor network traffic during testing
- Use VPNs for remote testing scenarios

#### Plugin Security
- Only load plugins from trusted sources
- Verify plugin checksums before loading
- Use code signing for custom plugins
- Review plugin source code when possible

### For Developers

#### Code Security
- Follow secure coding practices
- Validate all user inputs
- Use parameterized queries to prevent SQL injection
- Implement proper error handling without information leakage

#### Dependency Management
- Regularly update dependencies using `make deps-upgrade`
- Monitor for security advisories with `make vuln-check`
- Use dependency scanning in CI/CD pipelines
- Pin dependency versions in production builds

#### Build Security
- Verify build integrity with checksums
- Use multi-stage Docker builds to minimize attack surface
- Scan container images for vulnerabilities
- Sign release artifacts

## Security Tools Integration

### Automated Security Scanning
```bash
# Run security analysis
make security

# Check for known vulnerabilities
make vuln-check

# Full security audit
make validate-full
```

### CI/CD Security
Our CI/CD pipeline includes:
- **SAST**: Static Application Security Testing with gosec
- **Dependency Scanning**: Vulnerability checks with govulncheck
- **Container Scanning**: Docker image vulnerability scanning
- **Supply Chain Security**: Signed commits and verified builds

## Incident Response

If a security incident occurs:

1. **Immediate Response**: Isolate affected systems
2. **Assessment**: Determine scope and impact
3. **Notification**: Inform affected users within 72 hours
4. **Remediation**: Deploy fixes and patches
5. **Post-Incident**: Review and improve security measures

## Contact Information

- **Security Team**: security@stormdb.org
- **General Issues**: https://github.com/elchinoo/stormdb/issues
- **Security Advisories**: https://github.com/elchinoo/stormdb/security/advisories

## Recognition

We appreciate security researchers who responsibly disclose vulnerabilities. With your permission, we'll acknowledge your contribution in:
- Security advisories
- Release notes
- Hall of fame on our website

Thank you for helping keep StormDB secure!
