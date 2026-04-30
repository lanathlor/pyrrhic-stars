# Codex Online - Task Runner

mod server

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

# --- Web Export ---

# Download Godot export templates (needed once)
web-install-templates:
    #!/usr/bin/env bash
    set -euo pipefail
    FULL=$(godot --version)
    VERSION=$(echo "$FULL" | cut -d. -f1-3)
    STATUS=$(echo "$FULL" | cut -d. -f4)
    TEMPLATE_DIR="$HOME/.local/share/godot/export_templates/${VERSION}.${STATUS}"
    if [ -d "$TEMPLATE_DIR" ] && ls "$TEMPLATE_DIR"/web_* 1>/dev/null 2>&1; then
        echo "Export templates already installed at $TEMPLATE_DIR"
        exit 0
    fi
    echo "Downloading Godot ${VERSION}-${STATUS} export templates..."
    mkdir -p "$TEMPLATE_DIR"
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT
    curl -L --progress-bar \
        "https://github.com/godotengine/godot/releases/download/${VERSION}-${STATUS}/Godot_v${VERSION}-${STATUS}_export_templates.tpz" \
        -o "$TMPDIR/templates.tpz"
    unzip -q "$TMPDIR/templates.tpz" -d "$TMPDIR"
    cp "$TMPDIR"/templates/* "$TEMPLATE_DIR/"
    echo "Templates installed to $TEMPLATE_DIR"

# Export client as web build
web-export: web-install-templates
    mkdir -p build/web
    godot --headless --path client/ --export-release "Web" ../build/web/index.html

# Serve web build with required COOP/COEP headers
web-serve:
    GOPATH="{{justfile_directory()}}/.go" go run tools/webserve/main.go

# Export and serve in one command
web: web-export web-serve

# --- Assets ---

blender:
    blender --python-use-system-env blender/

blender-open file:
    blender --python-use-system-env "{{file}}"

# --- Setup ---

# Configure git hooks and local tooling
setup:
    git config core.hooksPath .githooks

# --- Utilities ---

fmt:
    cd server && goimports -w .

psql:
    psql postgres://codex:codex@localhost:5432/codex

redis-cli:
    redis-cli -p 6379
