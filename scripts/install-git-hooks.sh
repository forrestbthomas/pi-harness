#!/usr/bin/env bash
# install-git-hooks.sh — point this repo's git at the shipped hooks.
#
# Uses core.hooksPath (repo-local config), so updates to scripts/git-hooks/
# propagate without re-running this script, and no files are copied into .git/.
set -euo pipefail

cd "$(dirname "$0")/.."
git config core.hooksPath scripts/git-hooks
echo "git hooks installed: core.hooksPath -> scripts/git-hooks ($(git config core.hooksPath))"
echo "  pre-commit: blocks sensitive artifact paths; runs gitleaks --staged when installed"
