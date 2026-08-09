#!/usr/bin/env bash
# bootstrap.sh — one command from a fresh clone to a working harness.
#
# 1. Ensures Node (via nvm) + the `pi` CLI are available.
# 2. Builds bin/pi-run from source.
# 3. Runs `pi-run setup` (creates eval/.venv, installs deps, refreshes model catalogs).
# 4. Prints how to provide an API key (plain env var first; Bitwarden optional).
#
# Idempotent: safe to re-run.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(dirname "$HERE")"
# Use the latest nvm-installed node by default; PI_NODE_VERSION overrides.
NODE_VERSION="${PI_NODE_VERSION:-latest}"

echo "== pi-harness bootstrap =="

# 1. Node + pi
if command -v node >/dev/null 2>&1 && command -v pi >/dev/null 2>&1; then
  echo "  node + pi already on PATH"
else
  echo "  Ensuring Node ${NODE_VERSION} via nvm ..."
  if [ -s "$HOME/.nvm/nvm.sh" ]; then
    # shellcheck disable=SC1091
    . "$HOME/.nvm/nvm.sh"
  fi
  if ! command -v node >/dev/null 2>&1; then
    echo "  Installing node ${NODE_VERSION} via nvm ..."
    nvm install "$NODE_VERSION"
  fi
  if ! command -v pi >/dev/null 2>&1; then
    echo "  Installing pi CLI (npm global) ..."
    npm install -g pi
  fi
fi

# 2. Build pi-run
echo "  Building bin/pi-run ..."
(cd "$ROOT" && go build -ldflags "-X github.com/forrestthomas1/pi-harness/internal/cli.Version=dev" -o bin/pi-run ./cmd/pi-run)

# 3. Python venv + deps
echo "  Setting up eval/.venv ..."
(cd "$ROOT" && bin/pi-run setup)

# 4. Key guidance
echo ""
echo "== Done. Provide an API key: =="
echo "  export OPENAI_API_KEY=sk-...          # or"
echo "  export OPENROUTER_API_KEY=sk-or-v1-... # or"
echo "  export DEEPSEEK_API_KEY=sk-...        # then:"
echo "  pi-run chat"
echo ""
echo "Bitwarden (optional): pi-run also resolves keys via ~/bin/bw_get (BW_GET override)."
