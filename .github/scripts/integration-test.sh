#!/bin/sh
# Integration test: starts a real notesd server and exercises notes-cli against it.
# Requires: bin/notesd and bin/notes-cli to exist (built by CI before this runs).
set -e

PASS=0
FAIL=0
NOTESD="$(pwd)/bin/notesd"
CLI="$(pwd)/bin/notes-cli"
TMPDIR="$(mktemp -d)"
SERVER_DIR="$TMPDIR/server"
DEV1_HOME="$TMPDIR/device1"
DEV2_HOME="$TMPDIR/device2"
SERVER_PORT=18080
SERVER_URL="http://127.0.0.1:$SERVER_PORT"
SERVER_PID=""

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$TMPDIR"
}
trap cleanup EXIT INT TERM

ok() {
    PASS=$((PASS + 1))
    printf "  PASS  %s\n" "$1"
}

fail() {
    FAIL=$((FAIL + 1))
    printf "  FAIL  %s\n" "$1"
    if [ -n "$2" ]; then
        printf "        %s\n" "$2"
    fi
}

assert_contains() {
    desc="$1"
    expected="$2"
    actual="$3"
    if printf '%s' "$actual" | grep -qF "$expected"; then
        ok "$desc"
    else
        fail "$desc" "expected to contain: $expected"
        printf "        got: %s\n" "$actual"
    fi
}

assert_not_contains() {
    desc="$1"
    unexpected="$2"
    actual="$3"
    if printf '%s' "$actual" | grep -qF "$unexpected"; then
        fail "$desc" "expected NOT to contain: $unexpected"
        printf "        got: %s\n" "$actual"
    else
        ok "$desc"
    fi
}

# ── server setup ──────────────────────────────────────────────────────────────

mkdir -p "$SERVER_DIR" "$DEV1_HOME" "$DEV2_HOME"

# Generate RSA key for JWT signing
openssl genrsa -out "$SERVER_DIR/notesd.key" 2048 2>/dev/null

cat > "$SERVER_DIR/.notesd.conf" <<EOF
[server]
listen = "127.0.0.1:$SERVER_PORT"

[database]
path = "$SERVER_DIR/notesd.db"

[auth]
private_key = "$SERVER_DIR/notesd.key"
access_token_expiry = "15m"
refresh_token_expiry = "720h"
EOF

# Start server
HOME="$SERVER_DIR" "$NOTESD" &
SERVER_PID=$!

# Wait for server to become ready (up to 10 seconds)
i=0
while [ $i -lt 20 ]; do
    if curl -sf "$SERVER_URL/api/v1/health" > /dev/null 2>&1; then
        break
    fi
    sleep 0.5
    i=$((i + 1))
done
if ! curl -sf "$SERVER_URL/api/v1/health" > /dev/null 2>&1; then
    echo "ERROR: server did not start within 10s"
    exit 1
fi

printf "Server started (pid %d)\n\n" "$SERVER_PID"

# ── helpers ───────────────────────────────────────────────────────────────────

# Run notes-cli as device 1
d1() { HOME="$DEV1_HOME" "$CLI" "$@"; }
# Run notes-cli as device 2
d2() { HOME="$DEV2_HOME" "$CLI" "$@"; }

# ── auth tests ────────────────────────────────────────────────────────────────

printf "=== Auth\n"

out=$(d1 register \
    --server "$SERVER_URL" \
    --email "user@test.local" \
    --password "hunter2" \
    --name "Test User" 2>&1) && ok "register new user" || fail "register new user" "$out"

out=$(d1 login \
    --server "$SERVER_URL" \
    --email "user@test.local" \
    --password "hunter2" \
    --device "device-1" 2>&1)
assert_contains "login device 1" "Logged in as" "$out"

# Device 2 logs in as the same user
out=$(d2 login \
    --server "$SERVER_URL" \
    --email "user@test.local" \
    --password "hunter2" \
    --device "device-2" 2>&1)
assert_contains "login device 2" "Logged in as" "$out"

out=$(d1 register \
    --server "$SERVER_URL" \
    --email "user@test.local" \
    --password "hunter2" \
    --name "Duplicate" 2>&1) && fail "duplicate register should fail" || ok "duplicate register rejected"

# ── notes CRUD (device 1) ─────────────────────────────────────────────────────

printf "\n=== Notes CRUD\n"

out=$(d1 notes create --title "First Note" --content "Hello world" 2>&1)
assert_contains "create note" "Created note" "$out"
NOTE1_ID=$(printf '%s' "$out" | grep -oE '[0-9a-f-]{36}')

out=$(d1 notes list 2>&1)
assert_contains "list shows note" "First Note" "$out"

out=$(d1 notes show "$NOTE1_ID" 2>&1)
assert_contains "show note title" "First Note" "$out"
assert_contains "show note content" "Hello world" "$out"

# ── sync (device 1 → server) ──────────────────────────────────────────────────

printf "\n=== Sync device 1 → server\n"

out=$(d1 sync 2>&1)
assert_contains "sync pushes note" '"pushed": 1' "$out"

# ── sync (server → device 2) ─────────────────────────────────────────────────

printf "\n=== Sync server → device 2\n"

out=$(d2 sync 2>&1)
assert_contains "sync pulls note to device 2" '"pulled": 1' "$out"

out=$(d2 notes list 2>&1)
assert_contains "device 2 sees note" "First Note" "$out"

# ── multi-device: device 2 creates a note ────────────────────────────────────

printf "\n=== Multi-device\n"

out=$(d2 notes create --title "Device 2 Note" --content "From second device" 2>&1)
assert_contains "device 2 create note" "Created note" "$out"
NOTE2_ID=$(printf '%s' "$out" | grep -oE '[0-9a-f-]{36}')

out=$(d2 sync 2>&1)
assert_contains "device 2 sync pushes" '"pushed": 1' "$out"

out=$(d1 sync 2>&1)
assert_contains "device 1 sync pulls device 2 note" '"pulled": 1' "$out"

out=$(d1 notes list 2>&1)
assert_contains "device 1 sees device 2 note" "Device 2 Note" "$out"

# ── todos ─────────────────────────────────────────────────────────────────────

printf "\n=== Todos\n"

out=$(d1 todos create "Buy milk" 2>&1)
assert_contains "create todo" "Created todo" "$out"
TODO_ID=$(printf '%s' "$out" | grep -oE '[0-9a-f-]{36}')

out=$(d1 todos list 2>&1)
assert_contains "list todos" "Buy milk" "$out"

out=$(d1 todos complete "$TODO_ID" 2>&1)
assert_contains "complete todo" "Completed:" "$out"

out=$(d1 todos list 2>&1)
assert_contains "completed todo in list" "Buy milk" "$out"

# Sync todos to device 2
d1 sync > /dev/null 2>&1
d2 sync > /dev/null 2>&1

out=$(d2 todos list 2>&1)
assert_contains "device 2 sees completed todo" "Buy milk" "$out"

# ── delete propagation ────────────────────────────────────────────────────────

printf "\n=== Delete propagation\n"

out=$(d1 notes delete "$NOTE1_ID" 2>&1)
assert_contains "delete note" "Deleted note" "$out"

out=$(d1 notes list 2>&1)
assert_not_contains "deleted note gone from device 1" "First Note" "$out"

d1 sync > /dev/null 2>&1
d2 sync > /dev/null 2>&1

out=$(d2 notes list 2>&1)
assert_not_contains "deleted note gone from device 2 after sync" "First Note" "$out"

# ── search ────────────────────────────────────────────────────────────────────

printf "\n=== Search\n"

out=$(d1 search "Device 2" 2>&1)
assert_contains "search finds note" "Device 2 Note" "$out"

out=$(d1 search "xyzzy_nonexistent" 2>&1)
assert_not_contains "search empty result" "Device 2 Note" "$out"

# ── summary ───────────────────────────────────────────────────────────────────

printf "\n"
printf "Results: %d passed, %d failed\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
