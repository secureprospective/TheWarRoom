# TheWarRoom pickup: where the client/server build stands (11-23, 9:10 PM CT)
**Date:** 2026-11-23
**Branch:** session/resume-1123e (this note); the work is on the branches named below
**Status:** ready to pick up. Scope is TheWarRoom only (the hosted server, the PWA and the Lab). Nothing else.

## Read first
1. This note.
2. `docs/resume/RESUME-rehearsal.md` on this branch. It holds the dates, rulings, gates and refuted hypotheses.
3. `docs/specs/mfl-helper-small.md` and `docs/specs/mfl-helper-hosted-design.md` on branch `session/mfl-helper-build`.

## What changed tonight
- **Merged to main** with Christopher's OK:
  - #479 line-budget guard
  - #483 practice live replay
  - #484 real drafts Tue Dec 1; practice drafts Sun 11-29 on the Lab
  - #485 MFL helper spec
  - #486 owner-door draft and Tuesday report card on the real binary; agent tokens see NFL games; official weeks read final
- **#487 scores read 14.5x faster** (2,155,410 → 148,687 ns/op). It failed CI on
  `TestEncryptedExportRoundtripFrozenReplayParity`.
  - Cause: the test compared the stored reading, which now carries a hidden encode cache, with a fresh
    replay using `reflect.DeepEqual`. What owners are sent is identical.
  - Fix: commit b9cba8672 compares the sent JSON instead. A planted wrong winner still fails the test.
    It passes locally, and it fails on #487 without the fix.
  - A detached script pushes the fix, waits for CI and squash-merges on green (his OK: "merge those pr when green").
    It survives this session closing.
    - Log: `~/fleet/runs/twr-merge-486-487-1123/run.log`. The final line is `#487 MERGED` or `#487 NOT merged`.
    - Script: `run487.sh` in the same folder.
- **Lab** auto-released v66 (efa544de4) at 8:30 PM. The status check was clean. It will release #487 by itself once merged.
- **Small MFL helper (T710 step 1):**
  - Rulings recorded in the spec (commit ea65a796e):
    - The owner types the MFL league number and picks their team. Plans are private and no MFL password is asked for.
    - The league is a normal seat with its rules pulled from MFL as settings.
    - Data is fetched only when an owner asks, at most once every 5 minutes, read only.
  - Design committed: `docs/specs/mfl-helper-hosted-design.md`.
    - The checks, plans and links all exist already but are reachable only through the desktop.
    - The hosted server has no MFL data path today.
    - Three build lanes:
      - A: read-only mirror, refresh, free-agent list with values
      - B: bid and lineup checks, plans
      - C: lazy panel in NativeFranchiseHQ
  - No code was written yet. Lane A was started, then stopped at close, with nothing committed.

## Next, in order
1. Check run.log for `#487 MERGED`. If it says NOT merged, read the failing CI job, fix it on `session/results-read-speed-3`, push, and merge on green (his OK stands).
2. Build MFL helper lane A on `session/mfl-helper-build`. The brief is the design §1–3 and §5A. Use Gemini Neon when the quota is back (~11 PM CT), otherwise Sonnet medium.
   - Every Neon build gets an Argon review: no sends to MFL, no other owner's data, no branch on league source.
   - After A, lanes B and C. Open a PR; it is not merged without his OK.
3. Rerun the Lab load test on the release holding #487 (`~/fleet/runs/twr-stage3-load-1123/` has the previous numbers).
4. standings.read speed: same method as #487, based on origin/main after #487 merges.
5. Prepare the Sunday 11-29 practice drafts on the Lab (copies of the leagues, `cycle.sh`, a seat link for him; keep the long pick clock).

## Verification
- `go test ./internal/leaguetransfer -run TestEncryptedExportRoundtripFrozenReplayParity` passes on b9cba8672.
  - It fails without the fix.
  - It passes on main.
  - A planted break is caught.
- The full gate is CI on #487 (running).

## Open items / what to check
- **Season server (Acer laptop, 192.168.1.162):** no route to host at 8:34 PM, probably off or asleep. It is not needed before the Saturday 11-28 setup, and nothing was done to it. Christopher: power it on before Saturday.
- **The overnight watcher was stopped at close.** Nobody is watching the Lab tonight except its own auto-release timer on Hermes.
- **Cleanup when merged:** worktrees for season-box-live, season-box-draft, mfl-helper-spec, results-read-speed-2, resume-1123d and resume-1123e.
