# Pyrrhic Stars - Task Runner

mod server
mod client

# --- Docker Compose ---

up:
    docker compose up -d --build

down:
    docker compose down

logs *args:
    docker compose logs -f {{args}}

up-infra:
    docker compose up -d redis postgres

# Bring up auth (Kratos) and its dependencies for local development
up-auth:
    docker compose up -d kratos

kratos-logs:
    docker compose logs -f kratos

# --- Web ---

# Export and serve in one command
web:
    just client web-export
    GOPATH="{{justfile_directory()}}/.go" go run tools/webserve/main.go

# --- Assets ---

blender:
    blender --python-use-system-env blender/

blender-open file:
    blender --python-use-system-env "{{file}}"

# --- Setup ---

# Configure git hooks and local tooling (pre-commit runs lint + test on staged files)
setup:
    git config core.hooksPath .githooks

# --- E2E Scenarios ---

# Run e2e scenario tests against a live server
# Usage: just e2e test_connect_hub
#         just e2e test_zone_cycle,test_connect_hub
#         just e2e test_zone_cycle --headed
e2e scenarios *flags:
    #!/usr/bin/env bash
    set -euo pipefail
    # Kill stale servers from previous runs
    fuser -k 7777/tcp 2>/dev/null || true
    fuser -k 7778/udp 2>/dev/null || true
    sleep 0.5
    cd server && GOPATH="$(cd .. && pwd)/.go" go build -o bin/gateway ./cmd/gateway && cd ..
    ROOT="$(pwd)"
    cd server
    CODEX_DEV=1 \
      GOPATH="$ROOT/.go" \
      CODEX_MOBS_DIR="$ROOT/shared/mobs" \
      CODEX_ENCOUNTERS_DIR="$ROOT/shared/encounters" \
      CODEX_LEVELS_DIR="$ROOT/shared/levels" \
      CODEX_ITEMS_DIR="$ROOT/shared/items" \
      bin/gateway &
    SERVER_PID=$!
    trap "kill $SERVER_PID 2>/dev/null; wait $SERVER_PID 2>/dev/null || true" EXIT
    for i in $(seq 1 40); do
        if bash -c "echo > /dev/tcp/127.0.0.1/7777" 2>/dev/null; then
            echo "[e2e] Server ready"
            break
        fi
        if [ "$i" -eq 40 ]; then
            echo "[e2e] Server failed to start"
            exit 1
        fi
        sleep 0.5
    done
    HEADED=""
    for flag in {{flags}}; do
        case "$flag" in --headed) HEADED=yes ;; esac
    done
    cd "$ROOT/client"
    set +e
    if [ -z "$HEADED" ]; then
        godot --headless -- --e2e-scenarios={{scenarios}}
    else
        godot -- --e2e-scenarios={{scenarios}}
    fi
    GODOT_EXIT=$?
    exit $GODOT_EXIT

# --- Utilities ---

psql:
    psql postgres://codex:codex@localhost:5432/codex

redis-cli:
    redis-cli -p 6379
