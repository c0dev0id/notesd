# Developer Guide

## Prerequisites

- Go 1.23+
- Node.js 18+ and npm (for web client)

## Project Structure

```
cli/
├── main.go                      # Entry point
├── internal/
│   ├── client/
│   │   └── client.go            # HTTP client, token storage, auto-refresh
│   └── cmd/
│       ├── root.go              # Root command, global setup
│       ├── login.go             # Login/register commands
│       ├── logout.go            # Logout command
│       ├── notes.go             # Notes subcommands (list/show/create/edit/delete)
│       ├── todos.go             # Todos subcommands (list/show/create/complete/delete)
│       └── search.go            # Search command
├── go.mod
├── go.sum
└── Makefile

server/
├── cmd/notesd/main.go           # Entry point
├── internal/
│   ├── api/
│   │   ├── api.go               # Router, helpers, RSA key management
│   │   ├── auth.go              # Register, login, refresh, logout handlers
│   │   ├── middleware.go        # JWT auth middleware, token issuance
│   │   ├── notes.go             # Notes CRUD + search handlers
│   │   ├── sync.go              # Sync pull/push handlers
│   │   ├── todos.go             # Todos CRUD + overdue handler
│   │   └── api_test.go          # HTTP-level integration tests
│   ├── config/
│   │   └── config.go            # TOML config loading ($HOME/.notesd.conf, $PWD/notesd.conf)
│   ├── database/
│   │   ├── database.go          # DB open, schema, timestamp helpers
│   │   ├── database_test.go     # Database unit tests
│   │   ├── notes.go             # Note SQL operations
│   │   ├── todos.go             # Todo SQL operations
│   │   ├── tokens.go            # Refresh token storage
│   │   └── users.go             # User SQL operations
│   └── model/
│       └── model.go             # Data types, request/response models, ID generation
├── go.mod
├── go.sum
├── Makefile
└── notesd.conf.example

web/
├── src/
│   ├── app.html                    # HTML template
│   ├── app.css                     # Tailwind + Tiptap styles
│   ├── lib/
│   │   ├── api.js                  # REST API client with auto-refresh
│   │   ├── db.js                   # Dexie.js IndexedDB offline storage
│   │   ├── sync.js                 # Background sync logic
│   │   ├── device.js               # Device ID helper
│   │   ├── stores/
│   │   │   └── auth.js             # Auth store (localStorage persistence)
│   │   └── components/
│   │       ├── Editor.svelte       # Tiptap rich text editor
│   │       ├── NoteList.svelte     # Note list sidebar
│   │       └── TodoItem.svelte     # Todo item component
│   └── routes/
│       ├── +layout.svelte          # App shell with nav, sync indicator
│       ├── +page.svelte            # Root redirect
│       ├── login/+page.svelte      # Login page
│       ├── register/+page.svelte   # Registration page
│       ├── notes/+page.svelte      # Notes split-pane view
│       └── todos/+page.svelte      # Todo list with filters
├── package.json
├── svelte.config.js
├── vite.config.js
├── tailwind.config.js
└── postcss.config.js
```

## Building

### Server

```sh
cd server
make build    # produces ./notesd binary
```

### CLI Client

```sh
cd cli
make build    # produces ./notesd binary
```

### Web Client

```sh
cd web
npm install
npm run build    # produces static site in build/
```

## Configuration

notesd reads TOML configuration files in order:

1. `$HOME/.notesd.conf` (global defaults)
2. `$PWD/notesd.conf` (local overrides)

Values from the local file override the global file. If neither exists, built-in
defaults are used. See `notesd.conf.example` for all options.

On first start, if the RSA private key file does not exist, notesd generates a
2048-bit key pair automatically.

## Running

```sh
cd server
make run      # builds and runs
```

Or:

```sh
cd server
cp notesd.conf.example notesd.conf   # adjust as needed
./notesd
```

The server listens on `127.0.0.1:8080` by default. Logs go to stderr.

### Web Client (development)

```sh
cd web
npm run dev
```

This starts a Vite dev server (default port 5173) that proxies `/api` requests
to the notesd server at `http://127.0.0.1:8080`.

## Testing

```sh
cd server
make test     # runs all tests with verbose output
```

Or:

```sh
cd server
go test -v ./...
```

Tests use temporary SQLite databases and auto-generated RSA keys. No external
services or test data fixtures are required.

## Dependencies

| Package | Purpose |
|---|---|
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| `github.com/golang-jwt/jwt/v5` | JWT token signing and validation |
| `golang.org/x/crypto` | bcrypt password hashing |
| `github.com/BurntSushi/toml` | TOML configuration parsing |

### CLI (`cli/`)

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI command framework |
| `golang.org/x/term` | Terminal password input without echo |

### Web (`web/`)

| Package | Purpose |
|---|---|
| `@sveltejs/kit` | SvelteKit framework |
| `svelte` | Svelte 5 UI compiler |
| `@tiptap/core` | Rich text editor framework |
| `@tiptap/starter-kit` | Tiptap essentials (bold, italic, headings, lists, etc.) |
| `@tiptap/extension-placeholder` | Editor placeholder text |
| `@tiptap/pm` | ProseMirror peer dependency |
| `dexie` | IndexedDB wrapper for offline storage |
| `tailwindcss` | Utility-first CSS framework |
| `vite` | Build tool and dev server |

## API Endpoints

### Authentication (public)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/register` | Create new user account |
| POST | `/api/v1/auth/login` | Authenticate and receive tokens |
| POST | `/api/v1/auth/refresh` | Exchange refresh token for new token pair |

### Authentication (protected)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/auth/logout` | Revoke all refresh tokens |

### Notes

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/notes` | List notes (supports `limit`, `offset`) |
| GET | `/api/v1/notes/:id` | Get single note |
| POST | `/api/v1/notes` | Create note |
| PUT | `/api/v1/notes/:id` | Update note (partial) |
| DELETE | `/api/v1/notes/:id` | Soft-delete note |
| GET | `/api/v1/notes/search?q=` | Search notes by title/content |

### Todos

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/todos` | List todos (supports `limit`, `offset`) |
| GET | `/api/v1/todos/:id` | Get single todo |
| POST | `/api/v1/todos` | Create todo |
| PUT | `/api/v1/todos/:id` | Update todo (partial) |
| DELETE | `/api/v1/todos/:id` | Soft-delete todo |
| GET | `/api/v1/todos/overdue` | List incomplete todos past due date |

### Sync

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/sync/changes?since=` | Get changes since timestamp (unix ms) |
| POST | `/api/v1/sync/push` | Push local changes with LWW resolution |

All protected endpoints require `Authorization: Bearer <access_token>` header.
