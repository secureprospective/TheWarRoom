# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-02, `build/ROSTER-1`.
APP-ROSTER-10 proof complete; worker-budget acceptance still fails. Local only.

## Where it stands
- `590f4caf0`: plain make build regenerated roster-history / transaction-follow bindings.
- `2ee267af8`: measured contract-history capture optimization, side-qualified and tested.
- Full importer: 3,338 appended, 17m50.423s → **10m43.485s**; final repeat **0**.
  Still **43.485s over** the unchanged ten-minute worker lifetime, before lineups.
- Refused: inherited **76 → 75** free; **107 → 77** kickoff-aware. Final repeats append
  **0** with identical refusal text. No full refusal clearance claimed.
- Both modes: **32/32** membership/status equality; Bills **W1–7 all recorded**,
  including kickoff-aware W7. Explicit archived W8: **32 appended / 0 refused**;
  repeat **0**. Actual commissioner jobs: **2 contracts / 0 unverified picks**.
- Final focused race batch: **52 test/subtest PASS**, zero fail/skip; final plain build
  and production commit hooks pass. Selective-capture mutant rejected; restored pass.
- Evidence: `docs/build-handoffs/APP-ROSTER-10-evidence.md`; every remaining exact reason:
  `docs/build-handoffs/APP-ROSTER-10-refusals.txt`. Logs/CPU profiles: `.tmp/roster10/`.
  Own disposable DBs/raw bodies/harnesses deleted after process/open-file checks.

## Next move / blocked on
- `internal/composition/migration_app.go:161`: ten-minute worker deadline still too
  short for the actual importer. Do not raise it. First atomic batch cannot expose
  partial progress. Full direct call is not automatic-worker acceptance.
- Final CPU: correction projection 30.42%, full-prefix event rereads 22.17%, governance
  15.29%, contract capture 13.69%, GC 20.69% (overlapping cumulative CPU, not wall shares).
  Target measured prefix rereads/projection next while preserving every ordinary gate.
- Lineup calls themselves now take 10m25s / 10m31s; repeats 3m15s / 3m30s. Expanded
  history regresses overall refresh cost; transaction acceleration alone is not success.
- Shared 74 W1–6 refusals lack held kickoff context. Free adds W7 All (04, currently IR).
  Locked adds W7 Kirk (15), Terrell (32), Miller (20): inactive/absent at first kickoff,
  activated/acquired October 25. Do not silently change the locked first-kickoff rule.
- Prior debt: durable off-roster blank-contract review; older leagues need ordinary
  enabling of transactions.follow; actual worker, independent review and GUI/Pulse proof.

## Tried and rejected
No timeout increase, fabricated kickoff/contract, skipped asset, direct ledger SQL
write, protected-file edit, remote or live MFL move. Capture-all mutant fails both
bounded capture and side isolation. Initial perf commit hit complexity; simplified
selection and passed the gate without bypass. Stopped pre-final attempts are not
reported as completed timings; transaction rollback to two records was verified.
