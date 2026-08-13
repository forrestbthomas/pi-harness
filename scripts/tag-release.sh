#!/usr/bin/env bash
# tag-release.sh <tag> — cut a release tag from the FETCHED remote main tip.
#
# Correct order (CONTRIBUTING.md / ROADMAP.md ritual #5): merge every release
# commit (incl. the CHANGELOG entry) to main via PR FIRST, then run this
# script. It tags exactly what `origin/main` points at, so the tag is always an
# ancestor of main — impossible to reproduce the v0.9.1/v0.9.2 mistake of
# tagging a local commit that squash-merging later rewrote.
#
# Usage: scripts/tag-release.sh v0.9.3 [remote]
set -euo pipefail

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 <tag> [remote]" >&2
  exit 2
fi
TAG="$1"
REMOTE="${2:-github}"

# Refuse to re-tag.
if git rev-parse --verify "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "error: tag ${TAG} already exists (local). Delete it first if you really mean to move it." >&2
  exit 1
fi

# Fetch the real main tip and refuse to tag anything else.
git fetch "${REMOTE}" main
MAIN_TIP="$(git rev-parse FETCH_HEAD)"

git tag -a "${TAG}" -m "${TAG}" "${MAIN_TIP}"
git push "${REMOTE}" "${TAG}"

echo "tag ${TAG} created at remote ${REMOTE}/main (${MAIN_TIP:0:12}) and pushed."
