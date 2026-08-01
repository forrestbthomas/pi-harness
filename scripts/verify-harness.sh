#!/usr/bin/env bash
# verify-harness.sh — validate the Pi harness OpenRouter setup.
#
# Safe to run any time. Prints presence/state info only; NEVER prints key values.
# Exits non-zero if a required check fails. No network calls are required for
# the checks themselves (the offline model-list check is informational).
#
# Usage: bash scripts/verify-harness.sh

set -uo pipefail

HARNESS="/Users/forrestthomas/Projects/harness"
PASS=0
FAIL=0

note() { printf '%s\n' "$*"; }
ok()   { PASS=$((PASS + 1)); printf '  [ok]   %s\n' "$*"; }
bad()  { FAIL=$((FAIL + 1)); printf '  [FAIL] %s\n' "$*"; }

note "== Pi harness verification =="

# --- 1. node + pi CLI ------------------------------------------------------
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
if nvm use v22.19.0 >/dev/null 2>&1; then
  ok "node $(node -v 2>/dev/null) (nvm v22.19.0)"
else
  bad "nvm use v22.19.0 failed"
fi

if command -v pi >/dev/null 2>&1; then
  ok "pi CLI present: $(pi --version 2>/dev/null | head -n1)"
else
  bad "pi CLI not found on PATH (after nvm use v22.19.0)"
fi

# --- 2. JSON validity ------------------------------------------------------
for f in \
  "$HARNESS/.pi/settings.json" \
  "$HOME/.pi/agent/settings.json" \
  "$HOME/.pi/agent/models.json"; do
  if [ -f "$f" ] && node -e 'JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"))' "$f" 2>/dev/null; then
    ok "valid JSON: $f"
  else
    bad "invalid or missing JSON: $f"
  fi
done

# --- 3. API key presence (presence only, never the value) -------------------
if [ -n "${OPENROUTER_API_KEY:-}" ]; then
  ok "OpenRouter key present (env OPENROUTER_API_KEY)"
elif [ -n "$("${BW_GET:-$HOME/bin/bw_get}" OPENROUTER_API_KEY 2>/dev/null)" ]; then
  ok "OpenRouter key present (Bitwarden via bw_get)"
elif [ -n "${OPENAI_API_KEY:-}" ]; then
  ok "OpenAI key present (fallback path: pi --provider openai)"
else
  bad "no API key found. Unlock Bitwarden ('bw unlock'), export OPENROUTER_API_KEY, or export OPENAI_API_KEY."
fi

# --- 3b. Bitwarden vault status (informational) --------------------------------
note "Bitwarden vault: $("${BW_GET:-$HOME/bin/bw_get}" --status 2>/dev/null || echo unknown)"

# --- 4. skills --------------------------------------------------------------
SP_COUNT=$(find "$HOME/.agents/skills" -maxdepth 2 -name SKILL.md 2>/dev/null | wc -l | tr -d ' ')
if [ "${SP_COUNT:-0}" -ge 14 ] 2>/dev/null; then
  ok "superpowers skills: ${SP_COUNT} SKILL.md files under ~/.agents/skills"
else
  bad "superpowers skills missing or incomplete under ~/.agents/skills (found ${SP_COUNT:-0})"
fi

if [ -d "$HOME/Projects/tmp/agent-skills/skills" ]; then
  ok "Addy Osmani agent-skills clone present (wired via settings 'skills' array)"
else
  bad "Addy Osmani agent-skills clone missing: $HOME/Projects/tmp/agent-skills/skills"
fi

if [ -d "$HOME/.pi/agent/skills/agent-skills" ] && [ -d "$HOME/.pi/agent/skills/superpowers" ]; then
  ok "durable skills present under ~/.pi/agent/skills (installed via scripts/install-skills.sh)"
fi

# --- 5. default model resolvable (informational, offline) --------------------
if command -v pi >/dev/null 2>&1; then
  if pi --offline --list-models 2>/dev/null | grep -q "openai/gpt-4o"; then
    ok "default model openai/gpt-4o present in model list"
  else
    note "  [info] openai/gpt-4o not in the offline model list; run 'pi update --models' once with network to refresh catalogs"
  fi
fi

note "== results: ${PASS} passed, ${FAIL} failed =="
[ "$FAIL" -eq 0 ]
