#!/usr/bin/env bash
unset GOLANGCI_LINT_CACHE
set -euo pipefail
script=$(realpath scripts/golangci-cache.sh)
tmp=$(mktemp -d)
mkdir -p "$tmp/bin" "$tmp/a" "$tmp/b"
printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$GOLANGCI_LINT_CACHE"\n' > "$tmp/bin/golangci-lint"
chmod +x "$tmp/bin/golangci-lint"
for d in "$tmp/a" "$tmp/b"; do git -C "$d" init -q; done
PATH="$tmp/bin:$PATH" HOME="$tmp/home" XDG_CACHE_HOME="$tmp/cache" bash -c 'cd "$1"; "$2" run' _ "$tmp/a" "$script" > "$tmp/ca"
PATH="$tmp/bin:$PATH" HOME="$tmp/home" XDG_CACHE_HOME="$tmp/cache" bash -c 'cd "$1"; "$2" run' _ "$tmp/b" "$script" > "$tmp/cb"
[[ $(cat "$tmp/ca") != $(cat "$tmp/cb") ]]
explicit="$tmp/explicit"; PATH="$tmp/bin:$PATH" GOLANGCI_LINT_CACHE="$explicit" bash -c 'cd "$1"; "$2" run' _ "$tmp/a" "$script" > "$tmp/ce"
[[ $(cat "$tmp/ce") == "$explicit" ]]
echo 'golangci cache test passed'
