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

# 1. Node + pi. pi-run requires an nvm-managed Node installation even if a
# different Node or pi binary happens to be available on PATH.
if [ -s "$HOME/.nvm/nvm.sh" ]; then
  # shellcheck disable=SC1091
  . "$HOME/.nvm/nvm.sh"
fi
if ! command -v nvm >/dev/null 2>&1; then
  echo "  nvm not found — install it first:"
  echo "    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash"
  echo "  then re-run this bootstrap."
  exit 1
fi

echo "  [global] Ensuring Node ${NODE_VERSION} via nvm ..."
echo "  [global] This may install Node in your nvm-managed global toolchain."
nvm install "$NODE_VERSION"

NVM_NODE_BIN="$(dirname "$(nvm which current)")"
if [ ! -x "$NVM_NODE_BIN/pi" ]; then
  echo "  [global] Installing the Pi coding agent with npm -g ..."
  echo "  [global] This installs pi in the active nvm-managed Node toolchain."
  # The REAL Pi coding agent: the plain `pi` npm package is an unrelated
  # legacy CLI (a digits-of-pi calculator) that prints "3" for every
  # invocation — pi-run would silently eval garbage. --ignore-scripts is the
  # README-documented install form.
  npm install -g --ignore-scripts @earendil-works/pi-coding-agent
else
  echo "  pi CLI already installed for the active nvm Node"
fi
# Sanity check: the coding agent reports a semver (e.g. 0.84.1); the legacy
# impostor prints "3". Refuse to proceed on a wrong CLI.
if ! "$NVM_NODE_BIN/pi" --version 2>/dev/null | grep -Eq '^[0-9]+\.[0-9]+\.'; then
  echo "  ERROR: $NVM_NODE_BIN/pi is not the Pi coding agent " \
       "(pi --version: $($NVM_NODE_BIN/pi --version 2>&1 | head -1))" >&2
  exit 1
fi

# 2. Build pi-run
echo "  Building bin/pi-run ..."
(cd "$ROOT" && go build -ldflags "-X github.com/forrestthomas1/pi-harness/internal/cli.Version=dev" -o bin/pi-run ./cmd/pi-run)

# 3. Python venv + deps
echo "  Setting up eval/.venv ..."
(cd "$ROOT" && bin/pi-run setup)

# 4. PATH and key guidance
echo ""
echo "  bootstrap complete. Add bin/ to your PATH:"
echo "    export PATH=\"$ROOT/bin:\$PATH\""
echo "  or run: $ROOT/bin/pi-run install"
echo "  verify: $ROOT/bin/pi-run version"
echo ""
echo "== Provide an API key: =="
echo "  export OPENAI_API_KEY=sk-...          # or"
echo "  export OPENROUTER_API_KEY=sk-or-v1-... # or"
echo "  export DEEPSEEK_API_KEY=sk-...        # then:"
echo "  $ROOT/bin/pi-run chat"
echo ""
echo "Secret manager (optional): pi-run resolves keys env-first, then from a configured secret manager (PI_SECRET_BACKEND: bitwarden via bw_get, 1password via op, or env-only)."
