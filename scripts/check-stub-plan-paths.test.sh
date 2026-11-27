#!/usr/bin/env bash
# check-stub-plan-paths.test.sh — fixture tests for scripts/check-stub-plan-paths.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

script=scripts/check-stub-plan-paths.sh
fixtures=scripts/testdata/check-stub-plan-paths

valid_out=$(STUB_PLAN="$fixtures/valid.md" "$script" 2>&1)
if ! grep -q '4 path(s)' <<<"$valid_out"; then
  echo "FAIL: valid fixture expected 4 checked paths, got: $valid_out" >&2
  exit 1
fi

set +e
missing_out=$(STUB_PLAN="$fixtures/missing.md" "$script" 2>&1)
rc=$?
set -e
if (( rc == 0 )); then
  echo "FAIL: missing fixture exited 0, expected non-zero" >&2
  exit 1
fi
for expected in 'retired/DeletedScreen\.tsx' 'commissioner/RetiredControlRoomJobs\.tsx'; do
  if ! grep -q "$expected" <<<"$missing_out"; then
    echo "FAIL: missing fixture output did not name $expected: $missing_out" >&2
    exit 1
  fi
done

echo "PASS: check-stub-plan-paths.test.sh (valid passed 4 paths, missing failed naming both retired paths)"
