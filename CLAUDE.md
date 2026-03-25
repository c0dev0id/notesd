# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

**Server** (Go):
```sh
cd server && make build   # build ./notesd binary
cd server && make test    # run tests (-v)
cd server && make run     # build and run
cd server && make clean
```

**CLI** (Go):
```sh
cd cli && make build
cd cli && make clean
```

**Web** (SvelteKit + Vite):
```sh
cd web && npm install
cd web && npm run dev     # dev server with HMR; proxies /api → 127.0.0.1:8080
cd web && npm run build   # production build → build/
```

## Architecture

Three independent components sharing a REST API contract:

```
server/   Go REST API + SQLite — single binary daemon
cli/      Go CLI client (Cobra) — thin wrapper around the API
web/      SvelteKit 2 + Svelte 5 SPA — offline-first via IndexedDB
```

### Server (`server/`)

- **Entry:** `cmd/notesd/main.go` — loads config, opens DB, starts HTTP server with graceful shutdown
- **Packages:** `internal/api` (handlers, routing, JWT middleware), `internal/database` (SQLite CRUD), `internal/config` (TOML), `internal/model` (shared types)
- **Auth:** RSA-2048 JWT (RS256); access tokens 15 min, refresh tokens 30 days with rotation; bcrypt cost 12; rate-limited auth endpoints (20 req/min per IP)
- **Database:** Pure Go SQLite (`modernc.org/sqlite`, no CGO); WAL mode; soft deletes via `deleted_at` for sync propagation; timestamps as Unix milliseconds
- **Config loading:** built-in defaults → `$HOME/.notesd.conf` → `$PWD/notesd.conf` (later files override)

### CLI (`cli/`)

- `internal/client/client.go` — HTTP client with auto-refresh on 401
- `internal/cmd/` — Cobra command tree (login, logout, notes, todos, search)
- Stores session at `~/.notesd/session.json` (mode 0600) and config at `~/.notesd/config.toml`

### Web (`web/`)

- `src/lib/api.js` — REST client with auto-refresh on 401
- `src/lib/db.js` — Dexie.js (IndexedDB) for local storage
- `src/lib/sync.js` — background sync every 30 s: pull-then-push, LWW conflict resolution
- `src/lib/stores/auth.js` — auth state in localStorage
- Tiptap 2 (ProseMirror) for rich text; note content stored as JSON

### Sync model

All writes go to local storage first (IndexedDB on web, or directly to the server via CLI). Background sync uses Last-Write-Wins on `modified_at` millisecond timestamps. Soft deletes propagate via `deleted_at`.

## Conventions

**Commit style:** `area: description` (e.g. `server: add rate limiting`, `web: fix sync race`). 50–72 chars, imperative mood, lowercase after colon, no trailing period. Body explains *why*, wrapped at 72 chars.

**Code style:** KISS; performance over convenience; exhaust existing libraries before adding dependencies. Tests use AAA pattern (Arrange, Act, Assert) with verbose output.

**Note types:** `"note"` | `"todo_list"` — todos can be standalone or embedded in a note via `note_id` + `line_ref`.
