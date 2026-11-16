#!/usr/bin/env bash
# Run on the lab PVE host only. Python keeps secrets out of argv and printed evidence.
set -euo pipefail
exec python3 "$(dirname "$0")/security-sweep.py" "$@"
