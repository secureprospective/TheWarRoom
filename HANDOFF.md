# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #334 (https://github.com/secureprospective/TheWarRoom/pull/334) open. Hosted notes/watch reuse #325's private store; the authenticated seated franchise is derived server-side; HQ and War Room inspector use hosted actions; private reads are uncached. Commands are server-only, leaving the local desktop route untouched.
- Verified: `go test -race ./internal/commandcatalog ./internal/app/commissioner` passed after the release-matrix fixes; `make lint`, frontend typecheck, nativegen check, and four focused Vitest files (24 tests) passed. Latest catalog availability change also passes lint and nativegen check.
- Next: commit and push local fixes, rerun CI; close #313/#328 only after green CI.
- Blocked / escalation to ClaudeBox: CI dev-toolchain audit flags GHSA-fc5q-39xc-vjpf (`picomatch <=4.0.7`); npm reports 4.0.7 current with no patched release, upstream `micromatch/picomatch#194` open. Decide dependency remediation; do not suppress or mislabel the advisory.
- CI detail: remote head `7f967ad` failed release-matrix tests; local `a610a6f` fixes their fixtures/filters. A push attempt did not update remote; retry after the latest catalog metadata adjustment.
- Tried/rejected: no private-read cache, commissioner view, schema migration, request-selected franchise, local desktop route change, audit bypass, force push, or `--no-verify`.
