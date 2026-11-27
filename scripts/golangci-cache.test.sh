#!/usr/bin/env bash
unset GOLANGCI_LINT_CACHE
set -euo pipefail
script=$(realpath scripts/golangci-cache.sh)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin" "$tmp/a" "$tmp/b"
printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$GOLANGCI_LINT_CACHE"\n' > "$tmp/bin/golangci-lint"
chmod +x "$tmp/bin/golangci-lint"
for d in "$tmp/a" "$tmp/b"; do git -C "$d" init -q; done
PATH="$tmp/bin:$PATH" HOME="$tmp/home" XDG_CACHE_HOME="$tmp/cache" bash -c 'cd "$1"; "$2" run' _ "$tmp/a" "$script" > "$tmp/ca"
PATH="$tmp/bin:$PATH" HOME="$tmp/home" XDG_CACHE_HOME="$tmp/cache" bash -c 'cd "$1"; "$2" run' _ "$tmp/b" "$script" > "$tmp/cb"
[[ $(cat "$tmp/ca") != $(cat "$tmp/cb") ]]
explicit="$tmp/explicit"; PATH="$tmp/bin:$PATH" GOLANGCI_LINT_CACHE="$explicit" bash -c 'cd "$1"; "$2" run' _ "$tmp/a" "$script" > "$tmp/ce"
[[ $(cat "$tmp/ce") == "$explicit" ]]

lint_recipe=$(awk '/^lint:/{f=1;next} f && /^[^\t]/{exit} f{print}' Makefile)
if ! grep -q 'scripts/golangci-cache\.sh' <<<"$lint_recipe" || grep -qE '(^|[[:space:]])golangci-lint([[:space:]]|$)' <<<"$lint_recipe"; then
  echo 'FAIL: Makefile lint target must invoke scripts/golangci-cache.sh' >&2
  exit 1
fi
if ! grep -q 'scripts/golangci-cache\.sh run' scripts/bloat.sh || grep -v '^[[:space:]]*#' scripts/bloat.sh | grep -qE '(^|[[:space:](])golangci-lint([[:space:]]|$)'; then
  echo 'FAIL: scripts/bloat.sh must invoke scripts/golangci-cache.sh' >&2
  exit 1
fi
if ! grep -q 'entry: scripts/golangci-cache\.sh' .pre-commit-config.yaml; then
  echo 'FAIL: .pre-commit-config.yaml must invoke scripts/golangci-cache.sh' >&2
  exit 1
fi
echo 'golangci cache test passed'
