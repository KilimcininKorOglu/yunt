# Feature 1: Project Foundation & Configuration

**Feature ID:** F001  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.1.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature establishes the fundamental project structure, configuration system, and development environment for Yunt. It includes setting up the Go module structure, implementing a flexible configuration loader that supports both YAML files and environment variables, establishing logging infrastructure, and creating the CLI framework. This foundation is critical as all other features depend on these core components.

The configuration system will support multiple deployment scenarios (development, production, Docker) and provide sensible defaults while allowing full customization. The project structure follows Go best practices with clear separation between internal packages, command entry points, and external dependencies.

## Goals

- Establish a clean, scalable Go project structure following best practices
- Implement a robust configuration system supporting YAML and environment variables
- Create a production-ready logging infrastructure with multiple output formats
- Build a CLI framework for server management and administrative tasks
- Set up development tooling (Makefile, build scripts, linting)

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] Project builds successfully with `go build`
- [ ] Configuration can be loaded from YAML and environment variables
- [ ] CLI commands execute successfully
- [ ] Logging outputs correctly in text and JSON formats
- [ ] Code passes linting and formatting checks

## Tasks

### T001: Initialize Go Module and Project Structure

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Create the initial Go module and establish the complete directory structure for the Yunt project. This includes creating all necessary directories for internal packages, command entry points, configuration files, scripts, and documentation.

#### Technical Details

- Initialize Go module with `go mod init yunt`
- Create directory structure matching the PRD architecture
- Set up `.gitignore` for Go projects
- Create placeholder `README.md` with project description
- Initialize `Makefile` with basic targets (build, test, clean)
- Create directory structure: `cmd/yunt/`, `internal/{config,domain,repository,service,smtp,imap,api,parser}`, `web/`, `webui/`, `configs/`, `scripts/`

#### Files to Touch

- `go.mod` (new)
- `go.sum` (new)
- `.gitignore` (update)
- `README.md` (new)
- `Makefile` (new)
- `cmd/yunt/main.go` (new)
- `internal/config/.gitkeep` (new)
- `internal/domain/.gitkeep` (new)
- `internal/repository/.gitkeep` (new)
- `internal/service/.gitkeep` (new)
- `internal/smtp/.gitkeep` (new)
- `internal/imap/.gitkeep` (new)
- `internal/api/.gitkeep` (new)
- `internal/parser/.gitkeep` (new)
- `configs/.gitkeep` (new)
- `scripts/.gitkeep` (new)

#### Dependencies

- None

#### Success Criteria

- [ ] Go module initialized successfully
- [ ] All directories created according to PRD structure
- [ ] `go mod tidy` runs without errors
- [ ] `.gitignore` includes Go-specific patterns
- [ ] README.md contains project overview
- [ ] Makefile has build, test, and clean targets

---

### T002: Implement Configuration Structure and Loader

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Implement the complete configuration structure with support for YAML files and environment variable overrides. The configuration system must handle all server components (SMTP, IMAP, API), database settings, authentication, logging, and admin user settings.

#### Technical Details

- Define Go structs matching the PRD configuration schema
- Use `spf13/viper` for YAML parsing and environment variable binding
- Implement configuration validation logic
- Support nested configuration structures
- Provide sensible defaults for all settings
- Implement environment variable override with `YUNT_` prefix
- Add configuration merge logic (file → env vars → defaults)

#### Files to Touch

- `internal/config/config.go` (new)
- `internal/config/loader.go` (new)
- `internal/config/defaults.go` (new)
- `internal/config/validation.go` (new)
- `configs/yunt.example.yaml` (new)

#### Dependencies

- T001 (project structure must exist)

#### Success Criteria

- [ ] All configuration structs defined with proper tags
- [ ] YAML configuration file loads successfully
- [ ] Environment variables override YAML settings
- [ ] Default values apply when not specified
- [ ] Configuration validation catches invalid settings
- [ ] Unit tests for configuration loading pass
- [ ] Example configuration file is complete and documented

---

### T003: Set Up Logging Infrastructure

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1 day

#### Description

Implement a structured logging system using `zerolog` that supports multiple output formats (text, JSON), configurable log levels, and output destinations (stdout, file). The logging system should be easily accessible throughout the application.

#### Technical Details

- Implement logger initialization based on configuration
- Support log levels: debug, info, warn, error
- Support output formats: text (for development), JSON (for production)
- Support output destinations: stdout, file
- Implement log rotation for file output
- Create logger factory/singleton pattern
- Add context-aware logging capabilities
- Implement request ID tracking for HTTP requests

#### Files to Touch

- `internal/config/logger.go` (new)
- `internal/config/logger_test.go` (new)

#### Dependencies

- T002 (configuration system must be ready)

#### Success Criteria

- [ ] Logger initializes with configuration settings
- [ ] Text format outputs readable logs
- [ ] JSON format outputs valid JSON
- [ ] Log levels filter correctly
- [ ] File output creates log files in correct location
- [ ] Unit tests for logger initialization pass
- [ ] Logger is thread-safe

---

### T004: Create CLI Framework with Cobra

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Build a command-line interface using `spf13/cobra` to support server operations and administrative tasks. Implement commands for starting the server, managing users, running migrations, and checking system health.

#### Technical Details

- Set up Cobra command structure
- Implement `serve` command with flags for config file and service selection
- Implement `migrate` command for database migrations
- Implement `user` subcommands: create, list, delete
- Implement `messages` subcommand: delete-all
- Implement `version` command with build info
- Implement `health` command for system checks
- Add global flags: --config, --log-level, --verbose
- Implement proper error handling and exit codes

#### Files to Touch

- `cmd/yunt/main.go` (update)
- `cmd/yunt/cmd_root.go` (new)
- `cmd/yunt/cmd_serve.go` (new)
- `cmd/yunt/cmd_migrate.go` (new)
- `cmd/yunt/cmd_user.go` (new)
- `cmd/yunt/cmd_messages.go` (new)
- `cmd/yunt/cmd_version.go` (new)
- `cmd/yunt/cmd_health.go` (new)

#### Dependencies

- T002 (configuration must be available)
- T003 (logging must be available)

#### Success Criteria

- [ ] `yunt --help` displays all commands
- [ ] `yunt serve --help` shows serve options
- [ ] Config file flag works correctly
- [ ] Version command displays build information
- [ ] Commands have proper error handling
- [ ] Exit codes are correct (0 for success, non-zero for errors)
- [ ] All commands are documented

---

### T005: Implement Build System and Scripts

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create build scripts and Makefile targets for development workflow, including building binaries, running tests, linting code, and generating releases. Support cross-compilation for multiple platforms.

#### Technical Details

- Expand Makefile with comprehensive targets
- Create `scripts/build.sh` for production builds
- Create `scripts/release.sh` for multi-platform releases
- Add version injection using ldflags
- Support CGO_ENABLED for SQLite
- Implement targets: build, test, test-coverage, lint, fmt, clean, install
- Add pre-commit checks
- Generate binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

#### Files to Touch

- `Makefile` (update)
- `scripts/build.sh` (new)
- `scripts/release.sh` (new)
- `.golangci.yml` (new)

#### Dependencies

- T001 (project structure)

#### Success Criteria

- [ ] `make build` creates working binary
- [ ] `make test` runs all tests
- [ ] `make lint` checks code quality
- [ ] `make fmt` formats code
- [ ] Build scripts generate versioned binaries
- [ ] Cross-compilation works for all target platforms
- [ ] Binary size is reasonable (< 30MB for single platform)

---

### T006: Add Core Dependencies and Go Module Management

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 0.5 days

#### Description

Add all required Go dependencies to `go.mod` and ensure proper version constraints. This includes dependencies for SMTP, IMAP, HTTP routing, database drivers, authentication, and utilities.

#### Technical Details

- Add dependencies listed in PRD section 12
- Use specific versions (not latest) for stability
- Run `go mod tidy` to clean up
- Verify all dependencies download correctly
- Document any version-specific requirements
- Check for security vulnerabilities using `go list`

#### Files to Touch

- `go.mod` (update)
- `go.sum` (update)

#### Dependencies

- T001 (project structure must exist)

#### Success Criteria

- [ ] All dependencies from PRD are added
- [ ] `go mod verify` passes
- [ ] `go mod tidy` makes no changes
- [ ] No dependency conflicts
- [ ] Build succeeds with all dependencies
- [ ] Dependency versions are pinned appropriately

---

## Performance Targets

- Configuration loading: < 10ms
- Logger initialization: < 5ms
- CLI command startup: < 100ms
- Binary size: < 30MB (single platform, no web UI)

## Risk Assessment

| Risk                              | Probability | Impact | Mitigation                                      |
|-----------------------------------|-------------|--------|-------------------------------------------------|
| Dependency version conflicts      | Medium      | Medium | Pin specific versions, test thoroughly          |
| Configuration complexity          | Low         | Medium | Comprehensive validation and examples           |
| Cross-platform build issues       | Low         | Low    | Use CGO only where necessary, test on platforms |
| CLI usability issues              | Low         | Low    | Follow standard CLI conventions, get feedback   |

## Notes

- Configuration system should be extensible for future features
- Logging should be structured from the start for production use
- CLI should follow UNIX conventions and be intuitive
- Build system should support CI/CD integration from day one
- Consider using `golangci-lint` for comprehensive code quality checks
