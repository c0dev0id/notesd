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
- CLI client with Cobra command framework
- CLI commands: login, register, logout, notes (list/show/create/edit/delete),
  todos (list/create/show/complete/delete), search
- CLI note editing via `$EDITOR`
- CLI token auto-refresh on expiry
- CLI config and session storage in `~/.notesd/`
- Web client (SvelteKit 2, Svelte 5, Tailwind CSS)
- Rich text editor with Tiptap (bold, italic, headings, lists, quotes, code)
- Offline storage with Dexie.js (IndexedDB)
- Background sync with 30-second interval
- Login and registration pages
- Split-pane notes view with debounced auto-save
- Todo management with filters (all, active, completed, overdue)
- CORS middleware on server for web client support
