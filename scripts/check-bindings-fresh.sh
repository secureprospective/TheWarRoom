#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

scripts/check-wails-version.sh
# A private copy preserves hand edits and isolates Wails' fixed-name temporary executable
# from generators in sibling worktrees. The bindings build tag needs no display or GTK.
scratch_root="$HOME/logs/TheWarRoom/reorg"
mkdir -p "$scratch_root"
scratch=$(mktemp -d "$scratch_root/bindings-fresh.XXXXXX")
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/project" "$scratch/tmp"
git ls-files --cached --others --exclude-standard -z |
  tar --exclude='frontend/wailsjs/*' --null -T - -cf - |
  tar -xf - -C "$scratch/project"
# main embeds this directory, but the binding generator never reads the UI assets.
mkdir -p "$scratch/project/frontend/dist"
printf 'bindings-only\n' > "$scratch/project/frontend/dist/index.html"
(
  cd "$scratch/project"
  TMPDIR="$scratch/tmp" wails generate module -nocolour
)
if ! diff -ru frontend/wailsjs "$scratch/project/frontend/wailsjs"; then
  echo 'check-bindings-fresh: stale bindings; run make bindings and commit frontend/wailsjs' >&2
  exit 1
fi
echo 'check-bindings-fresh: generated bindings match frontend/wailsjs'
