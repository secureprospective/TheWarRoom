# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #334 (https://github.com/secureprospective/TheWarRoom/pull/334) open. Hosted notes/watch use #325's private store; seated franchise is derived server-side; HQ and War Room inspector use hosted actions; local desktop UI stays on its existing route; private reads are uncached; agent tokens are issuer-seat-only. Desktop service adapters and release-matrix exclusions now account for these private workflows.
- Verified: Focused race tests pass for Wails parity, release admission, private owner/privacy/cache/token behavior, plus the affected leagueservice/commandcatalog/serverhttp packages. `make lint`, nativegen checks, frontend typecheck and four Vitest files (24 tests) pass. First full CI found catalog-matrix fixtures; updated locally and verified.
- Next: Commit/push current fixes, check CI again; close #313/#328 only after green CI.
- Blocked on: CI dev-toolchain audit flags GHSA-fc5q-39xc-vjpf (`picomatch <=4.0.7`; npm reports no patched version, and 4.0.7 is current). Upstream fix PR `micromatch/picomatch#194` is still open. Do not suppress audit or mislabel a patched version; escalate the dependency decision to ClaudeBox.
- Tried/rejected: No private-read cache, commissioner view, schema migration, request-selected franchise, local desktop route change, or dependency-audit bypass.
