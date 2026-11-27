#!/usr/bin/env bash
set -euo pipefail

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT=${SHAPES_SCRIPT:-$HERE/shapes.sh}
ORIG_PATH=$PATH
REAL_PYTHON=$(command -v python3)
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

setup_bin="$work/setup-bin"
setup_kit="$work/setup-kit"
mkdir -p "$setup_bin" "$setup_kit/bin"
touch "$setup_kit/lab-commish.json" "$work/ssh.conf"
printf 'https://example.invalid\n' >"$setup_kit/server.url"
cat >"$setup_bin/ssh" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat >"$setup_bin/python3" <<SH
#!/usr/bin/env bash
if [ "\${1:-}" = "-I" ] && [ "\${2:-}" = "labcommish.py" ] && [ "\${3:-}" = "helper" ]; then
  printf 'helper-%s\n' "\$4" >"\$TWR_LAB_KIT/helper-\$4.txt"
  exit 0
fi
if [ "\${1:-}" = "-I" ] && [ "\${2##*/}" = "schedule.py" ]; then
  out=""
  shift 2
  while [ \$# -gt 0 ]; do
    if [ "\$1" = "--out" ]; then out=\$2; shift 2; else shift; fi
  done
  printf '{}\n' >"\$out/schedule-generate.json"
  printf '{}\n' >"\$out/schedule-publish.json"
  exit 0
fi
exec "$REAL_PYTHON" "\$@"
SH
cat >"$setup_kit/bin/twr" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat >"$setup_kit/apply.sh" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$TWR_LAB_KIT/apply.log"
SH
chmod +x "$setup_bin/ssh" "$setup_bin/python3" "$setup_kit/bin/twr" "$setup_kit/apply.sh"

PATH="$setup_bin:$ORIG_PATH" TWR_LAB_KIT="$setup_kit" LAB_SSH_CONF="$work/ssh.conf" bash "$SCRIPT" setup
"$REAL_PYTHON" - "$setup_kit" <<'PY'
import json, pathlib, sys

kit = pathlib.Path(sys.argv[1])
calls = [line.split() for line in (kit / "apply.log").read_text().splitlines() if line.strip()]
assert len(calls) == 3, f"expected 3 apply.sh calls, got {len(calls)}"

for n, args in zip((10, 12, 16), calls):
    names = [pathlib.Path(a).name for a in args]
    assert "draft-expire.json" in names, f"shape-{n}: draft-expire.json missing from apply.sh args: {names}"
    assert names.index("draft-expire.json") < names.index("draft-open.json"), (
        f"shape-{n}: draft-expire.json must be applied before draft-open.json: {names}"
    )
    w = kit / f"shape-{n}"
    expire = json.loads((w / "draft-expire.json").read_text())
    assert expire["command"] == "commish.rules.set", expire
    assert expire["args"] == {"field": "draft.onExpire", "value": "autopick"}, expire
    assert expire["effective"] == "now", expire
    assert expire["league"] == f"native.rehtest-{n}", expire
    reqs = [json.loads((w / fn).read_text())["request"] for fn in ("draft-clock.json", "draft-expire.json", "draft-open.json")]
    assert len(set(reqs)) == 3 and all(reqs), f"shape-{n}: duplicate or empty request keys: {reqs}"
PY
echo 'shapes setup offline checks passed'
