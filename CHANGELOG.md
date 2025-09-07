# Changelog

All notable changes to the Iris Provider SLog project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-09-06

### Added
- Initial release of iris-provider-slog
- Complete SLog provider implementation for Iris logging library
- Bridge between Iris and Go's structured logging (slog) ecosystem
- Support for all Iris log levels with proper slog level mapping
- Structured attribute forwarding from Iris fields to slog
- Configurable slog handler support (JSON, Text, or custom handlers)
- Thread-safe concurrent operations
- Comprehensive test coverage including integration tests
- Complete documentation with usage examples
- Production-ready implementation with quality assurance

### Features
- **SLog Integration**: Seamless integration with Go's slog package
- **Level Mapping**: Iris levels mapped to appropriate slog levels
- **Attribute Support**: Full support for structured logging attributes
- **Handler Flexibility**: Works with any slog.Handler implementation
- **Performance**: Optimized for high-throughput logging scenarios
- **Compatibility**: Full compatibility with slog ecosystem

### Quality Assurance
- 100% test coverage with integration tests
- All Go quality tools passing (go vet, staticcheck, gosec)
- Race condition testing
- Comprehensive documentation
- Production-ready codebase
