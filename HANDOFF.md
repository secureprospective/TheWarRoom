# Handoff

- Baton: Bee, 2026-11-18. Bundle 5 PR #350 is open; its latest CI was pending at merge.
- State: D65 is implemented on `session/pulse-b5perm`. The updated base has the hosted Rules board and Control Room frame/Jobs/Situations lanes.
- Flow: native catalog → `history.Store` transaction → `timedCommandRole` → `AuthorizeRecords` / `mayRun` → ledger append → governance replay; `commish.people.read` exposes effective permissions. Grants are private ledger events; no table/migration.
- D65: six permission groups; whole-mutation-catalog classification; preview/execute permission command; replay/revocation; co-commissioners get six; vote close/tiebreak stay commissioner-only.
- Verified D65: touched Go race packages; frontend 123 files/910 tests; TypeScript; generator freshness; `make lint`; `make bloat`.
- Slop fixed in PR #350: removed name-substring rule-board classification and restored owner guidance. D65: no unrelated slop found in scope.
- Tried/rejected: letting Change rules bypass field-level C-only checks; existing `TestCommissionerRulesPermissions` rejects co-commissioner edits to C-only `league.teams`, so deputies retain that restriction.
- Open decisions untouched: snooze scope, deputy job visibility, written-request visibility, division names.
- Next: push D65 PR against `bundle/b5` with the mapping and filelen rationale; head brain reviews both PRs/CI.
- Blocked: no D65 blocker. Preset comparison, house rules and written requests still await hosted reads (PR #350).
