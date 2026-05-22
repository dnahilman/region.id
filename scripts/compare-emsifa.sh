#!/usr/bin/env bash
# Generate region.id output with --no-links and diff against the emsifa
# upstream static/api tree. Any difference is a backward-compat regression.
#
# Usage:  scripts/compare-emsifa.sh [path-to-emsifa-static-api]
#
# Default emsifa path: ../api-wilayah-indonesia/static/api
set -euo pipefail

EMSIFA_DIR="${1:-../api-wilayah-indonesia/static/api}"
OUT_DIR="${OUT_DIR:-/tmp/region-compare}"

if [[ ! -d "$EMSIFA_DIR" ]]; then
  echo "emsifa static/api directory not found: $EMSIFA_DIR" >&2
  exit 1
fi

echo "==> Building region binary"
go build -o ./region ./cmd/region

echo "==> Generating to $OUT_DIR (--no-links, --force)"
rm -rf "$OUT_DIR"
./region generate --data ./data --out "$OUT_DIR" --no-links --force

echo "==> Diffing $OUT_DIR/api vs $EMSIFA_DIR"
if diff -r "$OUT_DIR/api" "$EMSIFA_DIR" --brief > /tmp/region-diff.txt; then
  echo "BYTE-IDENTICAL — backward compatibility preserved."
  exit 0
fi

# diff exits non-zero when there are differences; show a summary.
echo
echo "Differences detected:"
head -50 /tmp/region-diff.txt
echo
echo "Full diff at /tmp/region-diff.txt"
exit 1
