# Contributing to StormDB

Thank you for your interest in contributing to StormDB! This document provides guidelines and information for contributors.

## 🚀 Quick Start for Contributors

### 1. Development Setup

```bash
# Fork and clone the repository
git clone https://github.com/yourusername/stormdb.git
cd stormdb

# Install development tools
make dev-tools

# Install dependencies
make deps

# Build and test
make build-all
make test
```

### 2. Development Workflow

```bash
# Create a feature branch
git checkout -b feature/awesome-feature

# Make your changes
# ... edit files ...

# Run pre-commit checks
make pre-commit

# Commit with conventional commits format
git commit -m "feat: add awesome feature"

# Push and create pull request
git push origin feature/awesome-feature
```

## 🎯 Ways to Contribute

### 💻 Code Contributions
- **Bug fixes**: Fix reported issues
- **New features**: Implement requested features
- **Performance improvements**: Optimize existing code
- **Plugin development**: Create new workload plugins
- **Documentation**: Improve code documentation

### 🔌 Plugin Development
- **New workloads**: Industry-specific benchmarks
- **Enhanced metrics**: Custom performance measurements
- **Integration plugins**: Connect with monitoring systems
- **Community plugins**: Share useful workloads

### 📚 Documentation
- **User guides**: Help new users get started
- **API documentation**: Document interfaces and functions
- **Examples**: Provide configuration examples
- **Tutorials**: Create step-by-step guides

### 🧪 Testing
- **Unit tests**: Improve test coverage
- **Integration tests**: Test database interactions
- **Performance tests**: Validate benchmark accuracy
- **Platform testing**: Test on different OS/architectures

### 🐛 Bug Reports
- **Issue reporting**: Report bugs with detailed information
- **Reproduction cases**: Provide minimal test cases
- **Environment details**: Include system information
- **Log analysis**: Share relevant log files

## 📋 Development Guidelines

### Code Style

#### Go Code Standards
```bash
# Format code
make fmt

# Run linting
make lint

# Static analysis
make vet
```

#### Commit Message Format
We use [Conventional Commits](https://conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(plugins): add redis workload plugin
fix(metrics): correct latency calculation
docs(readme): update installation instructions
test(integration): add PostgreSQL 16 tests
```

### Code Quality

#### Test Requirements
- **Unit tests**: All new code must have unit tests
- **Integration tests**: Database interactions must be tested
- **Coverage**: Maintain >80% test coverage
- **Performance tests**: Benchmarks for performance-critical code

```bash
# Run all tests
make test-all

# Generate coverage report
make test-coverage

# Run benchmarks
make benchmark
```

#### Code Review Criteria
- **Functionality**: Code works as intended
- **Testing**: Adequate test coverage
- **Documentation**: Code is well documented
- **Performance**: No unnecessary performance degradation
- **Security**: No security vulnerabilities introduced
- **Style**: Follows project coding standards

### Plugin Development Guidelines

#### Plugin Structure
```
plugins/my_plugin/
├── main.go              # Plugin entry point
├── operations.go        # Workload operations
├── data_loader.go       # Data setup (optional)
├── go.mod              # Plugin dependencies
├── go.sum              # Dependency checksums
├── README.md           # Plugin documentation
└── examples/           # Configuration examples
    └── config_my_plugin.yaml
```

#### Plugin Interface Implementation
```go
package main

import (
    "context"
    "github.com/elchinoo/stormdb/pkg/plugin"
    "github.com/elchinoo/stormdb/pkg/types"
    
    "github.com/jackc/pgx/v5/pgxpool"
)

type MyPlugin struct{}

func (p *MyPlugin) GetMetadata() *plugin.PluginMetadata {
    return &plugin.PluginMetadata{
        Name:        "my_plugin",
        Version:     "1.0.0",
        Description: "Description of my plugin",
        Author:      "Your Name",
        WorkloadTypes: []string{"my_workload", "my_workload_read"},
    }
}

func (p *MyPlugin) CreateWorkload(workloadType string) (plugin.Workload, error) {
    switch workloadType {
    case "my_workload":
        return &MyWorkload{}, nil
    case "my_workload_read":
        return &MyWorkloadRead{}, nil
    default:
        return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
    }
}

func (p *MyPlugin) Initialize() error   { return nil }
func (p *MyPlugin) Cleanup() error     { return nil }

// Plugin entry point
var Plugin MyPlugin
```

#### Plugin Testing
```bash
# Test plugin compilation
cd plugins/my_plugin
go build -buildmode=plugin -o ../../build/plugins/my_plugin.so *.go

# Test plugin loading
./stormdb --config examples/config_my_plugin.yaml --setup

# Run plugin tests
make plugins-test
```

## 🔄 Pull Request Process

### 1. Before Submitting

```bash
# Ensure all checks pass
make validate-full

# Update documentation if needed
# Update CHANGELOG.md if adding features

# Rebase on latest main
git fetch origin
git rebase origin/main
```

### 2. Pull Request Template

When creating a pull request, please include:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows the project's style guidelines
- [ ] Self-review of code completed
- [ ] Code is commented, particularly in hard-to-understand areas
- [ ] Corresponding changes to documentation made
- [ ] No new warnings introduced
```

### 3. Review Process

1. **Automated Checks**: CI pipeline runs tests and quality checks
2. **Code Review**: Maintainers review code and provide feedback
3. **Testing**: Manual testing on different environments
4. **Approval**: At least one maintainer approval required
5. **Merge**: Squash and merge to main branch

## 🎖️ Recognition

### Contributors
All contributors are recognized in:
- **README.md**: Contributors section
- **Release notes**: Feature acknowledgments
- **GitHub**: Contributor statistics

### Maintainers
Consistent contributors may be invited to become maintainers with:
- **Commit access**: Direct push to repository
- **Release management**: Create and manage releases
- **Issue triage**: Manage issues and pull requests
- **Community support**: Help other contributors

## 📞 Getting Help

### Communication Channels
- **GitHub Issues**: Technical discussions and bug reports
- **GitHub Discussions**: General questions and feature requests
- **Email**: For sensitive topics (maintainers@stormdb.org)

### Development Support
- **Documentation**: Check docs/ directory
- **Examples**: Review existing plugins and configurations
- **Code Review**: Request feedback early and often
- **Mentoring**: Experienced contributors available to help

## 📚 Resources

### Development Resources
- [Go Development Guide](https://golang.org/doc/effective_go.html)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Plugin Development Guide](docs/PLUGIN_DEVELOPMENT.md)
- [Architecture Overview](ARCHITECTURE.md)

### Testing Resources
- [Go Testing](https://golang.org/pkg/testing/)
- [Testify Framework](https://github.com/stretchr/testify)
- [PostgreSQL Testing](docs/TESTING.md)

## 📄 License Agreement

By contributing to StormDB, you agree that your contributions will be licensed under the MIT License. You also confirm that:

- You have the right to submit the contribution
- Your contribution is your original work or properly attributed
- You grant the project maintainers a perpetual license to use your contribution

---

Thank you for contributing to StormDB! 🎉
