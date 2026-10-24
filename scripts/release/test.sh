#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
# The lock contains Linux CPython 3.13 wheels for the offline release workstation.
python3.13 -m venv .venv
.venv/bin/python -m pip install --require-hashes --only-binary=:all: -r requirements.txt
.venv/bin/python -m unittest discover -s . -v
