# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Release automation workflow for building and publishing releases
- Multi-platform binary builds (Linux, macOS, Windows) for both AMD64 and ARM64
- Automated GitHub release creation with checksums and release notes
- Docker multi-platform image publishing to GitHub Container Registry
- Enhanced release script with web UI build support

### Changed

- Improved release.sh script with README inclusion in archives

## [0.1.0] - 2024-01-01

### Added

- Initial release of Yunt Mail Server
- SMTP server with mail capture and relay support
- IMAP server for mail client support (Thunderbird, etc.)
- Modern Web UI admin panel
- REST API for full-featured integration
- Multi-user support with isolated mailboxes
- Multi-database support (SQLite, PostgreSQL, MySQL, MongoDB)
- Docker support with multi-platform images (AMD64, ARM64)
- MIME parser for email handling
- Configuration via YAML file or environment variables
