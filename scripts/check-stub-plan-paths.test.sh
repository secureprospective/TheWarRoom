#!/usr/bin/env bash
# check-stub-plan-paths.test.sh — fixture tests for scripts/check-stub-plan-paths.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

script=scripts/check-stub-plan-paths.sh
fixtures=scripts/testdata/check-stub-plan-paths

tmp_unreadable=$(mktemp)
cleanup() {
  chmod 600 "$tmp_unreadable" 2>/dev/null || true
  rm -f "$tmp_unreadable"
}
trap cleanup EXIT

expect_pass() {
  local label=$1 file=$2 expected_substr=$3
  local out
  out=$("$script" "$file" 2>&1)
  if ! grep -qF "$expected_substr" <<<"$out"; then
    echo "FAIL [$label]: expected '$expected_substr', got: $out" >&2
    exit 1
  fi
  echo "PASS [$label]: $out"
}

expect_fail() {
  local label=$1 file=$2 expected_substr=$3
  local out rc
  set +e
  out=$("$script" "$file" 2>&1)
  rc=$?
  set -e
  if (( rc == 0 )); then
    echo "FAIL [$label]: exited 0, expected non-zero: $out" >&2
    exit 1
  fi
  if [[ -n $expected_substr ]] && ! grep -qF "$expected_substr" <<<"$out"; then
    echo "FAIL [$label]: output did not contain '$expected_substr': $out" >&2
    exit 1
  fi
  echo "PASS [$label]: failed (rc=$rc) as expected"
}

expect_pass "valid" "$fixtures/valid.md" "5 path(s)"
expect_pass "dot-slash-valid" "$fixtures/dot-slash-valid.md" "2 path(s)"

expect_fail "missing" "$fixtures/missing.md" "retired/DeletedScreen.tsx"
expect_fail "missing-table" "$fixtures/missing.md" "commissioner/RetiredControlRoomJobs.tsx"
expect_fail "markdown-link-missing" "$fixtures/markdown-link-missing.md" "retired/Deleted.tsx"
expect_fail "missing-gen" "$fixtures/missing-gen.md" "retired/Deleted.gen.ts"
expect_fail "space-backtick-missing" "$fixtures/space-backtick-missing.md" "retired/Old Views homeTiles.tsx"
expect_fail "zero-match" "$fixtures/zero-match.md" "no recognized source paths"

chmod 000 "$tmp_unreadable"
if [[ ! -r $tmp_unreadable ]]; then
  expect_fail "unreadable-plan" "$tmp_unreadable" "cannot read plan file"
else
  echo "SKIP [unreadable-plan]: running as root/unrestricted, chmod 000 still readable"
fi
expect_fail "nonexistent-plan" "$fixtures/does-not-exist.md" "cannot read plan file"

set +e
noarg_out=$("$script" 2>&1)
noarg_rc=$?
set -e
if (( noarg_rc == 0 )); then
  echo "FAIL [no-arg]: exited 0, expected non-zero: $noarg_out" >&2
  exit 1
fi
echo "PASS [no-arg]: failed (rc=$noarg_rc) as expected"
