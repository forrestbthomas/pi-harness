#!/usr/bin/env bash
# install-skills.sh — durable install of curated agent skills for Pi.
#
# Clones the two curated skill collections into ~/.pi/agent/skills/ (a Pi
# global skill location that is auto-discovered in every session), then
# removes the temporary clone path from the settings "skills" arrays now
# that a durable clone exists.
#
# Idempotent: re-running updates both clones (git pull --ff-only).
# Requires git and network access. Run from anywhere:
#   bash scripts/install-skills.sh
#
# After running, verify with: pi-run config-check

set -euo pipefail

SKILLS_DIR="${HOME}/.pi/agent/skills"
# Derive the project settings path from the git root so this works from any
# clone location (no hardcoded user paths).
PROJECT_SETTINGS="$(git rev-parse --show-toplevel 2>/dev/null)/.pi/settings.json"
GLOBAL_SETTINGS="${HOME}/.pi/agent/settings.json"

if [ ! -f "$PROJECT_SETTINGS" ]; then
  echo "warning: not in a git clone of pi-harness (cannot locate .pi/settings.json); skipping project settings edit" >&2
  PROJECT_SETTINGS=""
fi

echo "== Skills install -> ${SKILLS_DIR} =="
mkdir -p "$SKILLS_DIR"

install_or_update() {
  local name="$1" url="$2" dir="${SKILLS_DIR}/$1"
  if [ -d "$dir/.git" ]; then
    echo "  updating ${name} ..."
    git -C "$dir" pull --ff-only
  else
    echo "  cloning ${name} (${url}) ..."
    git clone --depth 1 "$url" "$dir"
  fi
}

install_or_update "superpowers"  "https://github.com/obra/superpowers.git"
install_or_update "agent-skills" "https://github.com/addyosmani/agent-skills.git"

echo "== Dropping temporary agent-skills path from settings =="
node -e '
const fs = require("fs");
for (const f of process.argv.slice(1)) {
  if (!fs.existsSync(f)) { console.error("skip (missing): " + f); continue; }
  const cfg = JSON.parse(fs.readFileSync(f, "utf8"));
  const before = (cfg.skills || []).length;
  cfg.skills = (cfg.skills || []).filter(s => !String(s).includes("agent-skills"));
  if (cfg.skills.length !== before) {
    fs.writeFileSync(f, JSON.stringify(cfg, null, 2) + "\n");
    console.log("  updated " + f);
  } else {
    console.log("  no change   " + f);
  }
}
' ${PROJECT_SETTINGS:+ "$PROJECT_SETTINGS"} "$GLOBAL_SETTINGS"

echo "== Done. Skills are auto-discovered from ${SKILLS_DIR} =="
echo "Note: superpowers also exists as a plain copy at ~/.agents/skills (pre-existing)."
echo "      Pi keeps the first skill found, so either copy works; remove one to avoid duplicate-skill warnings."
echo "Run: pi-run config-check"
