# Architecture

## Overview

notesd follows an offline-first architecture with opportunistic synchronization.
Clients store all data locally and sync with the server when a connection is
available.

## Components

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Web Client │  │   Android   │  │  CLI Client  │
│  (SvelteKit)│  │  (Kotlin)   │  │    (Go)      │
└──────┬──────┘  └──────┬──────┘  └──────┬───────┘
       │                │                │
       └────────────────┼────────────────┘
                        │ REST API (JSON)
                        │ /api/v1/
                ┌───────┴────────┐
                │  notesd server │
                │     (Go)      │
                └───────┬────────┘
                        │
                ┌───────┴────────┐
                │    SQLite      │
                └────────────────┘
```

## Server (`server/`)

Single Go binary serving a REST API over HTTP.

### Package Structure

- `cmd/notesd/` — Entry point, server startup, graceful shutdown
- `internal/api/` — HTTP handlers, routing, JWT middleware
- `internal/config/` — TOML configuration loading
- `internal/database/` — SQLite operations, schema, CRUD
- `internal/model/` — Shared data types and request/response models

### Data Flow

1. HTTP request arrives at the router (`net/http` ServeMux)
2. Auth middleware validates JWT access token, injects user ID into context
3. Handler reads request, calls database layer
4. Database layer executes parameterized SQL against SQLite
5. Handler writes JSON response

### Sync Strategy

- **Conflict resolution:** Last-Write-Wins (LWW) based on `modified_at` timestamps
  with millisecond precision (UTC)
- **Soft deletes:** Records are marked with `deleted_at` rather than removed,
  allowing deletions to propagate via sync
- **Pull:** `GET /api/v1/sync/changes?since=<unix_ms>` returns all changes
  (including deletions) since the given timestamp
- **Push:** `POST /api/v1/sync/push` accepts batches of notes and todos;
  the server applies LWW and returns conflicts where the server version wins

### Authentication

- JWT tokens signed with RS256 (RSA)
- Access tokens: 15 minute expiry
- Refresh tokens: 30 day expiry, rotated on use
- Refresh token hashes stored in database for revocation

### Database

- SQLite with WAL mode for concurrent read performance
- Foreign keys enforced
- Timestamps stored as INTEGER (Unix milliseconds)
- Indexes on `user_id`, `modified_at`, `deleted_at`, `due_date`
