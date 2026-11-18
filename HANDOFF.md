# Handoff

- Baton: Bee, 2026-11-18; lane `pulse-b7door`, base `bundle/b7`.
- State: implementation stopped at the brief's migration fallback. No server routes or behavior changed. Blocker recorded in `docs/build-handoffs/BROWSER-DOOR-MIGRATION-BLOCKER-2026-11-18.md`.
- Evidence: `account_sessions` has no league field; `league_directory` has no per-account recency; desktop device label is fixed and not a league. A memory-only selector loses its choice on restart, and ledger mutation history misses read-only league use.
- Next: review the draft PR and decide whether to add persistent per-account or per-session last-used league state to the server account database. Resume browser session routes only after that migration decision.
- Blocked on: migration approval and recency ownership choice.
- Tried and rejected: in-memory last-league map (not durable); inferring use from league events (misses reads); reusing the device label (changes its existing meaning).
- Hygiene: `tidy-check.sh` found the parallel `b7web` launcher (pid 797562); left its run and scratch untouched because they are not this lane's.
