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

## CLI Client (`cli/`)

Thin command-line wrapper around the server REST API. Uses Cobra for command
structure. No local database — all operations are direct API calls.

### Package Structure

- `main.go` — Entry point
- `internal/client/` — HTTP client with token management and auto-refresh
- `internal/cmd/` — Cobra command definitions (login, notes, todos, search)

### Authentication Flow

1. User runs `notesd login`, provides server URL, email, password
2. Client calls `POST /api/v1/auth/login`, stores tokens in `~/.notesd/session.json`
3. Subsequent commands attach the access token as `Authorization: Bearer` header
4. On 401 response, the client automatically refreshes using the stored refresh token
5. `notesd logout` revokes server-side tokens and deletes the local session

### Configuration

- `~/.notesd/config.toml` — Server URL, device ID
- `~/.notesd/session.json` — Access and refresh tokens (file mode 0600)

## Web Client (`web/`)

Single-page application built with SvelteKit 2 and Svelte 5. Runs entirely in
the browser with offline-first capabilities.

### Package Structure

- `src/lib/api.js` — HTTP client for server REST API, auto-refresh on 401
- `src/lib/stores/auth.js` — Svelte writable store with localStorage persistence
- `src/lib/db.js` — Dexie.js IndexedDB wrapper for offline CRUD and sync
- `src/lib/sync.js` — Background sync (30s interval), pull then push
- `src/lib/device.js` — Device ID generation and persistence
- `src/lib/components/Editor.svelte` — Tiptap rich text editor with toolbar
- `src/lib/components/NoteList.svelte` — Sidebar note list with selection
- `src/lib/components/TodoItem.svelte` — Todo item with checkbox and delete
- `src/routes/` — SvelteKit page routes (login, register, notes, todos)

### Offline Architecture

1. All data is written to IndexedDB (via Dexie.js) first
2. Background sync runs every 30 seconds when authenticated
3. Sync pulls server changes since last sync timestamp, then pushes local changes
4. Conflict resolution follows LWW — same as server side
5. Auth tokens stored in localStorage for persistence across tabs

### Build

Built with Vite 5 and deployed as a static site via `@sveltejs/adapter-static`.
In development mode, Vite proxies `/api` requests to `http://127.0.0.1:8080`.
