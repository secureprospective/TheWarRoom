# Handoff

- Baton: Bee, 2026-11-17.
- Previous: current main includes PRs #330 and #331 hosted player-usage facts.
- State: PR #333 https://github.com/secureprospective/TheWarRoom/pull/333 is open; PR #323 is closed with a link to #333. Hosted queue uses existing per-league `private_entries`; session/token franchise comes only from the authenticated seat. Embedded DraftRoom is unchanged; ADP/tier stay unavailable.
- Tests: queue privacy/replacement tests, server-pin tests, full race tests for app/commissioner, accounts, commandcatalog, leagueservice, serverhttp, personal store, and twr-server; nativegen; frontend 121 files / 904 tests; typecheck; `make lint`; pre-push verify all pass.
- CI: four race-test shards, package build/install, and release-tools pass. Blocking dependency audit flags CVE-2026-112366 in picomatch 4.0.7/4.0.4 via Vite. npm has no patched release; upstream PR #194 is open. No audit bypass or unreviewed patch.
- Next: Christopher/ClaudeBox security decision on the no-fix audit blocker; then re-run checks and update PR #333.
- Tried/rejected: no schema change, shared league cache, commissioner-wide view, local queue route, or dependency-audit bypass.
