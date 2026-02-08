# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Server implementation with REST API (`/api/v1/`)
- User registration and authentication (JWT RS256, bcrypt)
- Refresh token rotation and revocation
- Notes CRUD with rich text content storage
- Todo CRUD with due dates and completion tracking
- Overdue todos endpoint
- Full-text search on notes (title and content)
- Sync pull/push endpoints with LWW conflict resolution
- Soft deletes via tombstones for sync propagation
- SQLite database with WAL mode (pure Go driver, no CGO)
- TOML configuration (`$HOME/.notesd.conf`, `$PWD/notesd.conf`)
- Auto-generated RSA key pair on first run
- Graceful server shutdown on SIGINT/SIGTERM
- Database and API test suites (30 tests)
