# Changelog

All notable changes to Go-Cache Framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Code generator (`go-cache-gen`) for automatic annotation scanning and registration
- `proxy.SimpleDecorate()` - One-liner initialization API
- `proxy.SimpleDecorateWithManager()` - Simple decoration with custom cache manager
- `proxy.SimpleDecorateWithError()` - Simple decoration with error handling
- `proxy.SimpleDecorateWithManagerAndError()` - Full-featured decoration API
- Multi-level cache support (L1 Memory + L2 Redis)
- Prometheus metrics integration
- OpenTelemetry distributed tracing
- Cache protection mechanisms (penetration, breakdown, avalanche)
- SpEL expression evaluation based on `expr` engine
- YAML configuration support

### Changed
- **Breaking**: Renamed `AutoDecorate` interface methods to avoid conflicts
- Improved code generator output with detailed summary
- Enhanced annotation parser to support more parameters

### Fixed
- Circular import issue between `pkg/core` and `pkg/proxy`
- Code generator path handling for nested directories

---

## [0.2.0] - 2026-03-13

### Added
- P2 Core Features implementation
- Automatic code generation for annotation metadata
- Simplified initialization API
- Multi-level cache backend (HybridBackend)
- Metrics and tracing support
- Comprehensive cache protection mechanisms

### Changed
- Updated README with new SimpleDecorate API
- Reorganized package structure for better separation of concerns
- Improved test coverage

### Fixed
- Various bug fixes in proxy interception logic
- Fixed TTL calculation edge cases

---

## [0.1.0] - 2026-03-01

### Added
- Initial release
- Basic cache annotations (@cacheable, @cacheput, @cacheevict)
- Memory and Redis backends
- SpEL expression support for dynamic cache keys
- Method interception via reflection
- Basic cache manager

---

## Legend
- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security improvements

---

**Generated**: 2026-03-13
