#!/usr/bin/env bash
# check-root-ratchet.sh — the root-file ratchet. Counts the *.go files (code and tests) in the repo
# root and fails if there are more than docs/architecture/root-go-baseline.txt allows. Features
# move out to internal/app/<feature>/, so the baseline only ever goes down.
set -euo pipefail
cd "$(dirname "$0")/.."

baseline_file=docs/architecture/root-go-baseline.txt
baseline=$(grep -v '^#' "$baseline_file" | tr -d '[:space:]')
[[ $baseline =~ ^[0-9]+$ ]] || { echo "check-root-ratchet: $baseline_file holds no count" >&2; exit 1; }

shopt -s nullglob
files=(*.go)
count=${#files[@]}

if (( count > baseline )); then
  echo "check-root-ratchet: the root holds $count *.go files, over the baseline of $baseline." >&2
  echo "  New code goes in internal/app/<feature>/ behind a thin root adapter (SYSTEM_MAP.md)." >&2
  exit 1
fi
if (( count < baseline )); then
  echo "check-root-ratchet: the root holds $count *.go files, under the baseline of $baseline."
  echo "  Lock it in: write $count to $baseline_file in the same commit."
  exit 0
fi
echo "check-root-ratchet: $count root *.go files, at the baseline"
