#!/usr/bin/env bash
# check-stub-plan-paths.sh — every source-file path in a stub plan markdown file
# must resolve to a tracked file (repo root, frontend/src/app/, or bare .go/.ts/.tsx).
set -euo pipefail
cd "$(dirname "$0")/.."

if (( $# != 1 )) || [[ -z $1 ]]; then
  echo "usage: check-stub-plan-paths.sh <plan-markdown-path>" >&2
  exit 2
fi

plan=$1

exec python3 - "$plan" <<'PY'
import os
import re
import subprocess
import sys

plan_path = sys.argv[1]

try:
    with open(plan_path, "r", encoding="utf-8") as f:
        lines = f.readlines()
except OSError as exc:
    print(f"check-stub-plan-paths: cannot read plan file {plan_path}: {exc}", file=sys.stderr)
    sys.exit(1)

try:
    ls_out = subprocess.check_output(["git", "ls-files", "-z"])
except (OSError, subprocess.CalledProcessError) as exc:
    print(f"check-stub-plan-paths: git ls-files failed: {exc}", file=sys.stderr)
    sys.exit(1)

tracked_list = [p for p in ls_out.decode("utf-8", errors="strict").split("\0") if p]
tracked_set = set(tracked_list)
tracked_go_basenames = {os.path.basename(p) for p in tracked_list if p.endswith(".go")}
tracked_app_ts_basenames = {
    os.path.basename(p)
    for p in tracked_list
    if p.startswith("frontend/src/app/") and (p.endswith(".ts") or p.endswith(".tsx"))
}

EXT_RE = re.compile(r"[A-Za-z0-9_.*\[\]-]\.(?:ts|tsx|go|mjs)$")
PLAIN_RE = re.compile(r"^(?:~/|/|https?://|\./)?[A-Za-z0-9_./*?\[\]-]+\.(?:ts|tsx|go|mjs)$")
BACKTICK_RE = re.compile(r"`([^`]+)`")
MD_LINK_RE = re.compile(r"\[[^\]]*\]\(([^)]+)\)")

candidates = set()
for line in lines:
    def take_backtick(match: re.Match[str]) -> str:
        span = match.group(1).strip()
        if EXT_RE.search(span) and not any(ch in span for ch in "+,;()"):
            candidates.add(span)
        return " "

    def take_link(match: re.Match[str]) -> str:
        dest = match.group(1).strip()
        dest = re.sub(r'\s+["\'].*$', "", dest)
        dest = dest.split("#", 1)[0].split("?", 1)[0].strip("<>")
        if EXT_RE.search(dest):
            candidates.add(dest)
        return " "

    without_backticks = BACKTICK_RE.sub(take_backtick, line)
    without_links = MD_LINK_RE.sub(take_link, without_backticks)
    for raw_tok in without_links.split():
        tok = raw_tok.lstrip("('\"<").rstrip(".,:;)'\">")
        if PLAIN_RE.fullmatch(tok):
            candidates.add(tok)

checked = 0
missing = []

for raw in sorted(candidates):
    p = raw
    # Drop true globs.
    if any(ch in p for ch in ("*", "?", "[")):
        continue
    # Skip paths outside the repo.
    if p.startswith("~/") or p.startswith("/") or p.startswith("http://") or p.startswith("https://"):
        continue
    # Normalize leading ./ prefixes.
    while p.startswith("./"):
        p = p[2:]
    if not p:
        continue

    checked += 1
    if p in tracked_set or f"frontend/src/app/{p}" in tracked_set:
        continue
    if "/" not in p:
        if p.endswith(".go") and p in tracked_go_basenames:
            continue
        if (p.endswith(".ts") or p.endswith(".tsx")) and p in tracked_app_ts_basenames:
            continue
    missing.append(p)

if checked == 0:
    print(f"check-stub-plan-paths: no recognized source paths found in {plan_path}", file=sys.stderr)
    sys.exit(1)

if missing:
    print(f"check-stub-plan-paths: missing tracked source file(s) in {plan_path}:", file=sys.stderr)
    for m in missing:
        print(f"  {m}", file=sys.stderr)
    sys.exit(1)

print(f"check-stub-plan-paths: {checked} path(s) in {plan_path} resolved to tracked files")
PY
