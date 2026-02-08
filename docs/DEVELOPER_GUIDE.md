# Developer Guide

## Prerequisites

- Go 1.23+

## Project Structure

```
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
```

## Building

```sh
cd server
make build    # produces ./notesd binary
```

Or directly:

```sh
cd server
go build -o notesd ./cmd/notesd
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
