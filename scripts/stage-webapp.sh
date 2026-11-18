#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
target="$PWD/internal/serverhttp/webassets"
case "${1:-}" in
  "") ;;
  --clean)
    find "$target" -mindepth 1 -maxdepth 1 ! -name .keep -exec rm -rf -- {} +
    exit 0
    ;;
  *) echo 'usage: stage-webapp.sh [--clean]' >&2; exit 2 ;;
esac
source_dir=
for candidate in frontend/dist-web frontend/dist; do
  if [[ -f "$candidate/index.html" ]]; then
    source_dir="$candidate"
    break
  fi
done
if [[ -z "$source_dir" ]]; then
  echo 'No built browser app found; run the frontend build first.' >&2
  exit 1
fi
find "$target" -mindepth 1 -maxdepth 1 ! -name .keep -exec rm -rf -- {} +
cp -R "$source_dir/." "$target/"
[[ -s "$target/index.html" ]] || { echo 'Staged browser app has no index.html.' >&2; exit 1; }
echo "Staged $source_dir in internal/serverhttp/webassets"
