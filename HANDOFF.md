# Handoff

## Baton
Bee — 2026-10-27, `session/mode1-mirror`. Settings F2b implemented; ready for ClaudeBox review. No push.

## Where it stands
- App-native Create + `NativeStart` append atomically, using the same reason and clock. Seed failure rolls both back.
- Source alone selects automatic start; MFL-import Create remains one record and waits for held capture.
- Original Create receipts/retries stay stable; start keys append `.native-start`. Hosted starts use recipient tokens and inherit credential-admitted creation, while retaining seed domain checks.
- `seed` job exists only before a start: native “Start the league”, import “Record starting assets”. Launcher/tray follow that job; no wire-shape change or binding generation needed.
- Renamed native start source/tests; references updated. Placement/shard weights index packages and had no old filename.
- Low-level single-record creation remains available for historical fixtures; older native Default32 and empty leagues can still start manually.

## Evidence
- PASS: requested test inventory and package tests; race batch for commissioner, history, app/commissioner and leagueservice; targeted composition creation/calendar/schedule/restore/wire tests.
- PASS: build all, vet internal, lint (0 issues), TypeScript, 31 frontend files / 426 tests; UI detector returned no findings.
- Failing-first creation table proved native leagues previously lacked starts. Table now covers exact Default32 seed bytes, open bids, empty starts, source/preset independence, jobs and retries.
- Injected seed insert failure leaves no Create; retry after removing failure records both. Hosted admission/revocation regression passes with both records.

## Next move
ClaudeBox reviews and runs the live gate on a throwaway profile. Create Default32: 32 unassigned O-Line lots, open bids, no starting-assets launcher/tray. Create empty: started, no lots; unset rules still need rulings. Older unstarted native: “Start the league”. MFL import: held-capture action until started.

## Blocked on
No implementation blocker. GUI verification and independent review not performed here. No production databases or live MFL calls used.

## Tried and rejected
- Create then seed separately: the store can make both atomic.
- Using ledger count for hosted seed Expected: recipient concurrency requires its opaque token.
- Requiring a separate hosted seed capability: automatic start is internally derived from already-admitted Create, not another user operation.

## Debt left
Live GUI verification remains. Existing generic seed receipt/ledger descriptions still refer to held data even for native starts; unchanged by this brief. Earlier unrelated debt remains in git history.
