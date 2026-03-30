# Codex Online - Task Runner

# --- Docker Compose ---

up:
    docker compose up -d --build

down:
    docker compose down

logs *args:
    docker compose logs -f {{args}}

up-infra:
    docker compose up -d redis postgres

# --- Client ---

godot:
    godot --path client/

godot-editor:
    godot --editor --path client/

client-test:
    godot --headless --path client/ -s addons/gdUnit4/bin/GdUnitCmdTool.gd -a res://tests

client-test-suite suite:
    godot --headless --path client/ -s addons/gdUnit4/bin/GdUnitCmdTool.gd -a res://tests/{{suite}}.gd

# Bot plays the game (watch it go)
client-bot:
    godot --path client/ -- --bot --capture

# Vanguard bot plays the game
client-bot-vanguard:
    godot --path client/ -- --bot --capture --class=vanguard

# Headless bot run for CI (default 30s, override with duration=N)
client-e2e duration="30":
    godot --path client/ -- --bot --capture --e2e --duration={{duration}}

# Headed bot run (watch the bot + get e2e result)
client-e2e-visual duration="30":
    godot --path client/ -- --bot --capture --e2e --duration={{duration}}

# Vanguard e2e
client-e2e-vanguard duration="30":
    godot --path client/ -- --bot --capture --e2e --class=vanguard --duration={{duration}}

# Remote control mode
client-remote:
    godot --path client/ -- --remote --capture

# --- Assets ---

blender:
    blender --python-use-system-env blender/

blender-open file:
    blender --python-use-system-env "{{file}}"

# --- Server (local dev) ---

server-gateway:
    cd server && go run ./cmd/gateway/

server-zone:
    cd server && go run ./cmd/zone/

server-chat:
    cd server && go run ./cmd/chat/

server-test:
    cd server && go test ./...

server-build:
    cd server && go build ./cmd/gateway/ ./cmd/zone/ ./cmd/chat/

# --- Utilities ---

fmt:
    cd server && goimports -w .

lint:
    cd server && go vet ./...

psql:
    psql postgres://codex:codex@localhost:5432/codex

redis-cli:
    redis-cli -p 6379
