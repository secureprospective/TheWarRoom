# Handoff

- Baton: Bee, 2026-11-17.
- Previous: merged origin/main (#334 Personal plumbing; #335 audit fix) into PR #333.
- State: PR #333 head `ec8ff0a` is open, mergeable, and CI-green; #323 is closed with a link to #333. Queue reads/writes reuse `LeaguePin.Personal`/`StoredHost.Personal`; tokens stay limited to the issuer's seated franchise. No schema/cache change; embedded DraftRoom unchanged; ADP/tier unavailable.
- Tests: Go race tests pass for cmd/twr-server, accounts, app/commissioner, commandcatalog, leagueservice, serverhttp, and store/personal. Frontend: 122 files / 905 tests; typecheck and native generators pass.
- Next: Christopher/ClaudeBox review and merge decision.
- Blocked: tidy check flags a scratch folder outside this task's authorized write paths; left untouched.
- Tried/rejected: duplicate private-store wiring; request-selected franchise; local DraftRoom changes.
