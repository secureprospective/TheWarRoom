#!/usr/bin/env bash
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT=${SHAPES_SCRIPT:-$HERE/shapes.sh}
work=$(mktemp -d "${TMPDIR:-/tmp}/twr-shapes.XXXXXX")
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/bin" "$work/kit"
touch "$work/kit/lab-commish.json"
cat >"$work/bin/python3" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$FAKE_PYTHON_LOG"
if [ "${FAIL_LEAGUE:-}" != "" ] && [ "$*" = "-I labcommish.py revoke $FAIL_LEAGUE" ]; then
  exit 23
fi
SH
cat >"$work/bin/ssh" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$FAKE_SSH_LOG"
SH
chmod +x "$work/bin/python3" "$work/bin/ssh"

export PATH="$work/bin:$PATH"
export TWR_LAB_KIT="$work/kit"
export FAKE_PYTHON_LOG="$work/python.log"
export FAKE_SSH_LOG="$work/ssh.log"

bash "$SCRIPT" release
for n in 10 12 16; do
  grep -Fx -- "-I labcommish.py revoke native.rehtest-$n" "$FAKE_PYTHON_LOG" >/dev/null
done
[ "$(wc -l <"$FAKE_PYTHON_LOG")" -eq 3 ]
[ ! -s "$FAKE_SSH_LOG" ] || { echo 'release unexpectedly contacted SSH' >&2; exit 1; }

: >"$FAKE_PYTHON_LOG"
if FAIL_LEAGUE=native.rehtest-12 bash "$SCRIPT" release >/dev/null 2>&1; then
  echo 'release unexpectedly succeeded after helper revoke failed' >&2
  exit 1
else
  rc=$?
  [ "$rc" -eq 23 ] || { echo "release returned $rc after helper revoke failure, want 23" >&2; exit 1; }
fi
[ "$(wc -l <"$FAKE_PYTHON_LOG")" -eq 2 ]
[ ! -s "$FAKE_SSH_LOG" ] || { echo 'failed revoke unexpectedly contacted SSH' >&2; exit 1; }
echo 'shapes release offline checks passed'
