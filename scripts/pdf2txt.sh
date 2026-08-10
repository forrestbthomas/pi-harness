#!/usr/bin/env bash
# pdf2txt — extract text from a PDF using the eval venv's pypdf.
#
# Pi's built-in `read` tool cannot extract text from PDFs (it fails with
# "get_content ... Not found"). Use this helper via the `bash` tool, then
# `read` the produced .txt file. Example:
#
#   bash scripts/pdf2txt.sh /path/to/file.pdf /tmp/file.txt
#   read /tmp/file.txt
#
# Usage:
#   scripts/pdf2txt.sh <file.pdf> [output.txt]
#   - with output.txt: writes extracted text to that file (stdout stays clean)
#   - without: prints extracted text to stdout
#
# HARNESS env var overrides the repo root (default: the parent of this script).

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(dirname "$HERE")"
HARNESS="${HARNESS:-$ROOT}"
PY="$HARNESS/eval/.venv/bin/python"

if [ "$#" -lt 1 ]; then
  echo "usage: pdf2txt.sh <file.pdf> [output.txt]" >&2
  exit 2
fi
if [ ! -x "$PY" ]; then
  echo "pdf2txt: eval/.venv python not found ($PY) - run 'make install' first" >&2
  exit 1
fi

exec "$PY" - "$@" <<'PYEOF'
import sys
from pathlib import Path
from pypdf import PdfReader

args = sys.argv[1:]
src, out = args[0], (args[1] if len(args) > 1 else None)

reader = PdfReader(src)
pages = []
for i, page in enumerate(reader.pages, 1):
    text = page.extract_text() or ""
    pages.append(f"--- page {i} ---\n{text}")
content = "\n\n".join(pages)

if out:
    Path(out).write_text(content, encoding="utf-8")
else:
    sys.stdout.write(content)
PYEOF
