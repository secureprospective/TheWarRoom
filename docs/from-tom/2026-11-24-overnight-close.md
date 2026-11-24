# Overnight 11-23/24 close: faster reads, My MFL moves live, `twr workflow`
**Date:** 2026-11-24
**Branch:** session/resume-1124 (this note and the resume doc; the code all went to main through PRs)
**Status:** Done. All 7 PRs merged on Christopher's OK. Nothing open from this run.

## What changed for owners and helpers

**Phones and the PWA are faster.** Measured on the Lab with 62 devices for 15 minutes. All 20,609 requests were answered, with no errors.

| | Before | After |
|---|---|---|
| A typical request | 95 ms | 39 ms |
| Scores and standings | ~130 ms | ~40 ms (3.2x faster) |
| First load of the league directory on a device | 1.8 s | 0.27 s |
| The slowest 5% of requests | 462 ms | 316 ms (target was 200 ms: **not met**) |

**My MFL moves** (T710) is in the hosted app, and the Lab carries it as v71. An owner whose league still lives on MFL can:
- refresh a read-only copy of the MFL league;
- pick their MFL team and see its roster;
- see free agents with values from the model run;
- check a waiver bid (cap room after the bid, one drop at most) or a lineup, in plain words;
- save the move as a private plan, open its MFL page, then mark it done or withdraw it.

Nothing is ever sent to MFL, and no MFL password is asked for.

**AI helpers and the terminal** can run any league command with `twr workflow <command> --json '{…}'`, or the MCP
tool `league_workflow`. When the league refuses, the helper sees the league's own words. This is PR #493.

**Sunday 11-29 practice drafts** have one script each, run in this order:
1. `scripts/lab/cycle.sh`
2. `seat-link.sh native.rehtest-default` (claim your seat)
3. `open-drafts.sh`

## PRs merged

| PR | What | Main commit |
|---|---|---|
| #487 | Scores read 14.5x faster in the store (a kept week encoded once) | 9e749162f |
| #491 | Heavy reads stop re-hashing 125k outside player ids per request (was ~70% of Lab CPU) | 8b08e6da9 |
| #492 | The "last seen" stamp never makes a request wait (25 ms bound) | cc88e745c |
| #488 | Standings read 2.3x faster (ledger read once, a kept season encoded once) | 6aeb64500 |
| #490 | One-command Sunday practice drafts (guarded to native.rehtest-* only) | faa12dd1e |
| #489 | My MFL moves on the hosted app | a99d18c84 |
| #493 | `twr workflow` + MCP `league_workflow` | 60ebd21fe |

## Why
These were Tom's pickup-note steps (docs/from-tom/2026-11-23-twr-pickup.md):
1. Confirm #487.
2. Build the MFL helper.
3. Rerun the Lab load test.
4. Make standings faster.
5. Prepare the Sunday drafts.

Overnight work added two more: the CPU and stall fixes the profile found (#491, #492), and the CLI gap the live MFL check found (#493).

## How it was built and checked
- **Builders:**
  - Claude Sonnet: MFL lanes A–E.
  - Gemini Neon High: fix pass F, the live Lab checks, a second-model review.
  - Luna high: lane H (panel fixes) and lane G (`twr workflow`).
- **Review:** I reviewed every lane line by line. I planted one break per lane, and the tests caught every one.
- **The MFL helper's path to working:**
  - It passed every unit test but **failed on the real Lab server**: all 12 commands were refused. The host wrappers dropped the MFL hooks, and admission did not know the commands.
  - Fix pass F fixed this and added an end-to-end test through the real server handler. The test fails on the old code.
  - Live Lab checks, run as a signed-in owner:
    - mfllive2 at 49da38235: all 12 commands passed.
    - mfllive3 at b46ae7d9c: 18/19. A two-drop bid answered 400.
    - mfllive4 at 541fb783f: the fix passed.
  - Reports are in `~/fleet/runs/mfllive{2,3,4}-1124/RESULT.md`.
- **Gemini's adversarial review** flagged two problems as merge-blocking. One was real and fixed: the size cap on the MFL league export. The other wasn't a blocker, but it led to a real fix: owner reads no longer take the database writer to create tables.
- **Lane G review:** I caught a regression in Luna's first version, where a refused `twr preview` lost its message. I fixed it with a test that failed first.

## Verification
- CI 7/7 green on every merged PR.
- Load tests: `~/fleet/runs/overnight-twr-1123/load/RESULT.md` (v68) and `load2/RESULT.md` (v70).
- Lab CT 260 now runs v71 (a99d18c84, #489). twr-status reported all OK at 06:51 CT.
- Season server: not touched.
- Filing gate PASS.
- tidy-check: only other agents' entries are left.

## Open items: what Christopher or Claude should check
1. **#493 merged** (60ebd21fe) after main was merged in following #489 (line-budget conflict).
   - The push failed twice first, on a stale golangci-lint cache from deleted worktrees. That was not a code problem; it's fixed and saved as a memory note.
   - Main now holds both #489 and #493, so a helper can run `twr workflow mflmoves.pool.read --json '{"sort":"minimum"}'` once the Lab releases 60ebd21fe.
2. **The slowest 5% of reads is ~300 ms** (target 200). It is on every read type, not one slow path, so the likely cause is the Lab's 2 cores during score-update bursts. This is **not proven**.
   - Next: profile one burst (the method is in `~/fleet/runs/profile-1123/`), or try more cores on CT 260.
3. **MFL money units are mixed in the JSON:** bid amounts go in as dollars, and salaries come out as cents. Settle this before more helpers build on it.
4. **Helper tokens** for My MFL moves also need `native.directory.read`. The desktop and `cycle.sh` already mint it.
5. **Values** in My MFL moves come from the house model run, not each league's own scoring.
6. **Rule for new hosted commands** (saved as a memory note): every one needs a test through `internal/serverhttp`'s real handler, plus a live Lab check. A business rule must never sit in `Validate()`, because hosted decode turns any Validate error into 400.

## Where everything is
- **Full night log:** `~/fleet/runs/overnight-twr-1123/REPORT.md` (summary on top).
- **Resume doc:** `docs/resume/RESUME-overnight-1124.md` on this branch, with a copy at Claudebox `/root/TheWarRoom-RESUME.md`.
- **Briefs:** `~/fleet/runs/overnight-twr-1123/briefs/`.
- **Profiles:** `~/fleet/runs/profile-1123/`.
- **Lab kit (secrets, never print):** `~/scratch/commish-1123/` (has KEEP.md).
- **Worktrees removed:** 13, all clean and pushed. The branches stay on GitHub.

## Final state at close (07:25 CT)
- **Main is at 60ebd21fe** (#493). The Hermes autorelease timer will put it on the Lab as the next release.
  - Lab check at close: v71 (a99d18c84), all OK.
- **Watchers:** all stopped. No job of this run is still running.
- **Task list:** T1820, T1821 and T1822 are checked.
