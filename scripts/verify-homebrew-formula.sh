#!/usr/bin/env bash
# verify-homebrew-formula.sh <tag> — verify a released Homebrew formula installs
# a working pi-run whose `version` matches the tag.
#
# Called from .github/workflows/release.yml after update-homebrew-formula.sh
# (REL-3). The formula push used to be fire-and-forget: a failed TAP_PUSH_TOKEN
# push or a bad formula shipped a tag whose brew install was silently broken.
# This step installs the tap formula and asserts the installed version.
#
# Usage: scripts/verify-homebrew-formula.sh v0.10.0
# Requires: a Homebrew on PATH (macOS runner or linuxbrew in CI).
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi
TAG="$1"

# --force-bottle avoids build-from-source surprises. The install prefix is
# managed by Homebrew; `brew install --prefix` is not a supported option on
# current Homebrew. Resolve the formula's actual opt prefix after installing.
brew install --force-bottle --formula forrestbthomas/tap/pi-run >/dev/null 2>&1 || {
  # Retry without --force-bottle (a bottle may not exist for the platform yet).
  brew install --formula forrestbthomas/tap/pi-run >/dev/null
}

BIN="$(brew --prefix pi-run)/bin/pi-run"
if [ ! -x "${BIN}" ]; then
  echo "::error::pi-run not installed at ${BIN} (brew install failed)" >&2
  exit 1
fi

VERSION="$("${BIN}" version 2>/dev/null | awk '{print $2}')"
if [ "${VERSION}" != "${TAG}" ]; then
  echo "::error::installed pi-run version ${VERSION} != tag ${TAG}" >&2
  exit 1
fi

echo "verified: brew pi-run version ${VERSION} == ${TAG}"
