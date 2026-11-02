# Handoff

## Baton
Bee → Christopher / ClaudeBox — 2026-11-01, `build/ROSTER-1`.
APP-ROSTER-6 remains incomplete. Local commits only; no remote contact.

## Where it stands
- `c2fabb074`: ordinary `contracts.unknown_acquisition` setting, no default;
  ordinary current-draft-slot corrections and regenerated Wails models.
- `3c616b695`: checked transaction translation, stable requests and atomic
  ordinary roster/pick correction replay. All 895 archive rows plan into 779
  transaction groups / 1,600 commands; LOAD_ROSTERS is baseline evidence only.
- Copied-profile replay stops at the first February cut: player 15239 is absent
  from the November seed. No transaction appended on either attempt/copy.
- Refusals unchanged: schedule-free 76; kickoff-aware 107. Roster equality
  remains 28/32 (0003/0015/0018/0025 differ). Bills 1–7 recorded schedule-free;
  kickoff-aware Bills 7 still refuses. Explicit archived week 8: 0 refusals
  without kickoffs, 32 pre-seed refusals with held kickoffs; not a live check.
- Final targeted race batch: 144 PASS; subsequent overlapping focused rerun:
  12 PASS. Plain make build PASS, including frontend tsc; binary not launched.
  Commit hooks passed. No full suite, independent review or GUI acceptance.
- Evidence and exact remaining refusals:
  `docs/build-handoffs/APP-ROSTER-6-evidence.md` and
  `docs/build-handoffs/APP-ROSTER-6-refusals.txt`.
- Disposable backup-API database copies and proof harness removed after checking
  open files/processes. Ignored text logs remain in `.tmp/roster6/`.
  Pre-existing untracked `.golangci-cache/` left alone.

## Next move / blocked on
- Implement ordinary unknown-contract admission and commissioner review intake;
  **do not await a default-policy answer**. The brief authorizes unknown state
  until configured. No salary or years was invented here, but that state is not
  represented/admitted yet. Guard: `internal/commissioner/correction_projector.go:46`.
- Implement inverse-consistent generic pre-observation projection; current refusal
  remains at `internal/commissioner/native_reading.go:40`.
- Reserve groups need ordinary prepared atomic execution, not a batch allowlist
  bypass (`internal/store/history/migration_import.go:94`).
- Add transaction-follow setting, migration actor and automatic import/sync wiring.
  Requested starting ref has neither `schedule.follow` nor `syncPlayedLineups`;
  protected composition files/Pulse were untouched. Resolve integration base.
- Prove successful historical replay/repeat-zero, both kickoff modes, all 32
  rosters and Bills 1–7. Full transaction-import time is still unmeasured because
  replay stops on its first transaction; timings in evidence are early refusals.

## Tried and rejected
- No removal of pre-seed guards without reconstruction, fabricated contract,
  unprepared reserve outcome, raw ledger SQL, backdated receipt clock, real MFL
  request or move. No remote contact, forced commit or full-suite run.
- Respect heavy STOPs and the shared linter lock; no other agent was interrupted.
- Copied-profile repeat-zero after initial refusal is not successful replay proof.
  Real-store successful post-observation trade repeat-zero is separately tested.
