# Codex Online - Task Runner

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

# --- Utilities ---

psql:
    psql postgres://codex:codex@localhost:5432/codex

redis-cli:
    redis-cli -p 6379
