# Ring 4 (commissioner side) — handoff to ClaudeBox
**Date:** 2026-10-19
**Branch:** `session/ring-4-integ2` (worktree `~/work/TheWarRoom/.worktrees/ring-4-integ2` on the Beelink)
**Status:** build complete for ring 4 scope; live walk on Claude-OS half done; PR not opened. Ready for the
last walk, then the PR and Christopher's go.

Written by the Beelink Claude Code session that built ring 4 (Christopher calls it "Tom" in skill names).
Christopher is transferring the session and its goal to ClaudeBox. Everything you need is in this file;
the long history is in the resume documents listed under "History", but you should not need them.

---

## 1. The goal and what "done" means

**Goal (Christopher, verbatim):** "finish ring 4". Ring 4 is the TheWarRoom commissioner side, native and
app-owned (spec `docs/ui/Target_UI_Spec_2026-10.md` §16).

**Scope (spec §16.11):** census groups A, B, C, D, E and G except CM-055, plus the commissioner-facing §15
carry items (mail log M-093, commissioner news M-095 via CM-052, waivers M-024/M-156/M-157 via CM-012 and
CM-035, cache tools M-137 via CM-054). Group F (CM-046–CM-049), CM-050 (awards) and CM-055 (engine
parameters) are **Later**, not ring 4. The controls reading confirms this: everything is built except
CM-048, CM-049, CM-050, CM-055. (CM-046/047 bracket acts were built anyway in c11c; harmless extra.)

**Done =**
1. Every ring-4 act works live on Claude-OS against a seeded app league (walk in §5 below).
2. `make verify` exit 0 on the branch tip.
3. PR opened from `session/ring-4-integ2` to `main` with the body built from `PR-notes.md` (§6).
4. Christopher is asked **one plain question** (does he accept the calls listed in PR-notes) and gives his go.
5. Squash-merge on his go (`~/.local/bin/gh pr merge <n> --squash`; main needs linear history).
6. Hermes task **T434** updated (`ssh hermes '~/work/command-center/bin/cc task check T434'` once merged;
   until then edit the line in place to the current state).

---

## 2. Where everything is

| What | Where |
|---|---|
| Repo (main checkout — **never touch**) | `~/work/TheWarRoom` on the Beelink |
| Integration worktree (all work lands here) | `~/work/TheWarRoom/.worktrees/ring-4-integ2`, branch `session/ring-4-integ2` |
| Branch history | 95+ commits ahead of `main`; latest are the c11 merges and walk fixes (see §3) |
| Run directory (briefs, Sol phases, logs, walk tools, shots) | `~/fleet/runs/warroom-ring4-2026-10-13/` |
| Integration logs (verify runs, builds, PR notes) | `~/fleet/runs/warroom-ring4-2026-10-13/integ2/` |
| **PR notes (the calls Christopher may overrule)** | `~/fleet/runs/warroom-ring4-2026-10-13/integ2/PR-notes.md` |
| Sol briefs for ring 4 phases | `~/fleet/runs/warroom-ring4-2026-10-13/par/*.full.md` |
| Walk screenshots | `~/fleet/runs/warroom-ring4-2026-10-13/shots/` (latest: `r4m-*`) |
| Product rulings for the commissioner side | `~/fleet/runs/warroom-commish-k3-2026-10-13/` (`K4-outline-draft.md`, rulings) |
| Spec | `docs/ui/Target_UI_Spec_2026-10.md` §16 (in the repo) |
| Test VM | Claude-OS (`ssh claudeos` from the Beelink), app dir `~/warroom-ring1/`, log `~/warroom-ring1/run.log` |
| Task list | Hermes `~/work/command-center/Task_list.md`, item **T434** (line ~542, `<!-- id:T434 -->`) |

Other ring-4 worktrees (`ring-4-c1-spine` … `ring-4-c11c`) are finished phase branches, all merged into
integ2. Leave them; remove them only after the squash-merge, through `git worktree remove` (never `mv`).

---

## 3. State at handoff

### Last commits on `session/ring-4-integ2`
```
<final commit of this handoff>   docs: ring 4 handoff to ClaudeBox
<walk-fix commit>                Starting assets preview: cap totals, typed reason keeps the preview
70c47845 Preview names each repeated pair of sources once
04d28574 Merge c11c: commissioner market, session identity, schedule and bracket acts
87abce55 Starting assets preview sums up each franchise
9fc3952d Commissioner market readings, session identity, schedule and bracket acts
33629f10 Starting assets preview shows before the reason is typed
7ee54213 Merge c11a: real previews for every commissioner act
e1318abf Merge c11b: the commissioner act tray
```
Run `git -C ~/work/TheWarRoom/.worktrees/ring-4-integ2 log --oneline -5` for the exact tip; §8 records
whether the push to origin finished.

### Verification
- verify23 (c11c merge, 04d28574): **VERIFY_EXIT=0**, root race run 2416 s, vitest 1345/1345, bundle
  317,518 B (cap 321,818).
- verify24 (walk-fix commit): see §8 for the result recorded at handoff.
- Every fix in this session has a test that failed against the old code (mutant planted and restored).

### What the c11 phases built (this session)
- **c11a** — a real preview (`TargetCommissionerPreview`) for every built act: who is affected, before →
  after with sources, blockers. Built flags are honest.
- **c11b** — one table-driven act tray (`CommissionerActTray.tsx`, `actTable.ts`) for every built control,
  plus the starting-assets tray (`SeedTray.tsx`).
- **c11c** — `TargetCommissionerMarket`: the seven commissioner waiver/bid/trade acts pick targets from a
  reading under the act's own permission; undecided blind bids carry only player + why it waits (never
  bidders or amounts); processed claims and decided bids only. Self acts use `TargetCommissionerActor`
  (signed-in person). Schedule generate/edit/publish and bracket acts have named pickers.

### Walk findings fixed this session (all committed, all in PR-notes.md)
1. Starting-assets tray opened on "The preview does not match this league version" (Go seed preview
   refused until a reason was typed, with no target). Fixed in Go (`RawNativeSeedRequest.ValidatePreview`)
   and in the tray (show Go's refusal, not a mismatch).
2. Starting-assets preview listed every player three times (thousands of rows). Now per franchise:
   roster by status with salary, draft picks by season, cap charges as "N cap charges totalling $X"
   (`internal/commissioner/preview_native.go` `seedPreviewValues`).
3. "League version: N" removed from every tray and the ledger header.
4. Preview sources repeated on every row; each distinct source pair now named once with its count
   (`PreviewReview.tsx`).
5. Typing the starting-assets reason re-read the multi-second preview on every keystroke and collapsed
   the tray. The last preview now stays; Go is asked again after a 400 ms pause; confirm waits for the
   exact request's preview (`SeedTray.tsx`, `seedPreviewPause`).

### Known, deliberately left
- Leagues created before the app required a season (the Claude-OS test leagues "Twelve team test 10-14",
  "Twelve team fixed 10-14", "Walk test 10-13") show a raw error chain on the League page. New leagues
  always record a season (`creation.go` refuses Season 0), so a real commissioner cannot reach this.
  Noted in PR-notes.

---

## 4. Next actions, in order

1. **Confirm the tip is verified.** If §8 says verify24 passed and the walk-fix commit exists, go on.
   Otherwise: run verify (env in §7), read any FAIL, fix, rerun, commit.
2. **Build and deploy** the tip to Claude-OS as `r4n` (commands in §7). Wait for `value model fitted` in
   `~/warroom-ring1/run.log` before opening trays: the first preview waits on the launch refresh (~20 s).
3. **Walk** (§5) on the league "Ring 4 walk 10-18". Screenshot every step into `shots/r4n-*`. Look at
   each screenshot (read the image) before claiming a step works.
4. **Fix what the walk finds** the same way: failing test first, fix, plant a mutant (one change per
   mutant, restore it), frontend-only change → frontend half of verify; Go change → full verify. Commit
   with `git commit -F` and the trailer. Add each finding to PR-notes.md "Walk findings".
5. **Watch for:** the act tray (`useChangePreview.ts`) may have the same collapse-on-every-keystroke
   behaviour the seed tray had (`current = review?.body === body ? review : undefined` drops the preview
   while the next one loads). Check it on a G3 act with a typed reason; fix the same way if it jumps.
   The seed preview takes several seconds on the full league; if it is slow outside launch, profile
   `PreviewCommissionerWithFacts` for the seed path.
6. **Clean up:** `go clean -cache` by hand when no builder or verify runs (the go-build cache is large);
   log it in the three ledgers (§9).
7. **PR:** write the body from `integ2/PR-notes.md` (calls 1–13, engineering notes, walk findings; add a
   short "c11" section: honest built flags, real previews, act tray, market reading, seed summary). End
   the body with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`. Open it with
   `~/.local/bin/gh pr create --base main --head session/ring-4-integ2`.
8. **Ask Christopher one plain question** (pick-from options): accept the calls in the PR as written, or
   name the ones to change. Squash-merge only on his go. Then update T434 and clear caches.

---

## 5. The walk (Claude-OS)

Tools live in `~/fleet/runs/warroom-ring4-2026-10-13/` and run from the Beelink:
- `./shot.sh <name>` → `shots/<name>.png` (read it to see it)
- `./click.sh x y`; keys: `ssh claudeos 'DISPLAY=:0 xdotool key End'`; typing:
  `ssh claudeos 'DISPLAY=:0 xdotool mousemove X Y click 1 type --delay 20 "text"'`
- Hold-to-fire: `ssh claudeos 'DISPLAY=:0 xdotool mousemove X Y mousedown 1 sleep 1.5 mouseup 1'`
- `./unstick-pointer.sh` if clicks stop registering (the spice tablet holds button 1)
- Coordinates (1920×1080): Control Room nav (87,384); League tab (528,86); league picker (382,327) when
  an error banner shows, (382,217) when not; "Ring 4 walk 10-18" is the 2nd option; a blocked job such as
  "Waiver processing" at (266,414) opens the starting-assets tray; tray Close (1880,101).

Steps (the walk league has a season but **no starting assets yet** — record them first):
1. **Starting assets:** open the tray, wait for the summary, scroll to the bottom (click in the tray, End
   a few times), type a reason, wait ~1 s for the re-check, hold "Record starting assets". Expect a
   receipt and the jobs strip to change from "Record starting assets first."
2. **A correction** (Season ops → correct a roster/contract): preview, reason, record, then the ledger.
3. **Rollover dry run** (Season ops → rollover → "Check the next season without recording").
4. **Stat approval queue** job → open an item.
5. **Draft room** (Draft group): open, check the clock and the pick picker.
6. **Market acts:** waivers process/override, bids process/override, trade review/appeal/override. A fresh
   league has no claims, bids or trades, so expect empty pickers with honest words; to see real choices,
   create an owner cut/bid/trade on the walk league first if the UI offers it.
7. **Schedule:** generate a draft, publish it, edit a published game. **Bracket** acts (extra scope).
8. **Jobs strip** after each act; **preview words** (no "League version", no CM-ids, no G1/G2/G3).

---

## 6. Christopher's standing constraints (keep every one)

- **Never send email or any outgoing message** without his explicit per-message permission. Drafts are fine.
- **Never open myfantasyleague.com league 14432 or any commissioner page, never submit to MFL.** Do not
  click "Read data again" on Claude-OS.
- **No work on main.** Branches `session/<desc>`. Never `--no-verify`. **Squash-merge only, only on his go.**
- Never touch `~/work/TheWarRoom` (main checkout). **Never `mv` a git repo or worktree.**
- Never commit in a worktree while a builder works there. Never commit `.cache/`, `.phase-tmp/`, `.tmp/`.
- Commits: `git commit -F <file>` ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
  PR bodies end with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- **Log every move or deletion** on the Beelink in `~/MOVED.md`, `~/archive/MANIFEST.md` and CT105
  `/root/.claude/backbone/context.md` (same one-line dated format as the existing entries).
- Use `scp`, not rsync. No `ssh -n` with a heredoc (writes an empty file). No `pkill -f` over ssh; no
  `pgrep`/`pkill -f` whose pattern matches its own shell.
- Free tools only. No GLM. Per-deploy approval for live sites (Claude-OS is the test VM, not live).
- Never run two golangci-lint at once. Never show confidence scores. Root go tests use `-timeout 45m`.
- Don't ask Christopher questions mid-build; ask the one PR question at the end. Two Sols at once is fine;
  avoid Tom (Beelink Claude) as builder unless Sol is down. Protect context.
- Clear caches at merge. When a usage limit hits, wait and continue.
- Every builder report gets one reviewer-planted mutant and a live click-through before Christopher clicks.

---

## 7. Commands (run on the Beelink, from the integ2 worktree)

```sh
export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8 \
  GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1 \
  GOLANGCI_LINT_CACHE=$HOME/fleet/runs/warroom-ring4-2026-10-13/integ2/lint-cache
cd ~/work/TheWarRoom/.worktrees/ring-4-integ2

# full verify (~45 min; the root package race run is ~40 of it). Detached so it survives the session:
nohup sh -c 'make verify VERSION=dev COMMIT=ring4 BUILD_DATE=ring4; echo VERIFY_EXIT=$?' \
  > ~/fleet/runs/warroom-ring4-2026-10-13/integ2/verifyNN.log 2>&1 &
# done when: grep VERIFY_EXIT .../verifyNN.log

# frontend half only (when only frontend files changed since a passing verify):
cd frontend && pnpm tsc --noEmit && pnpm test && pnpm run build && node scripts/bundle-budget.mjs

# after a merge or a Go binding change:
wails generate module && git checkout HEAD -- frontend/wailsjs/runtime
TWR_WRITE_WIRE=1 go test -count=1 -timeout 45m -run 'Wire|TestScheduleEditOwnerRevertRestart|TestTheDraftRoomWireIsWhatGoSends|TestDraftReadingSurvivesReopeningAndFixesTheWire|TestNativeDraftEngineWire' . ./internal/commissioner/ ./internal/store/history/
LEAGUERULES_WRITE_GOLDEN=1 go test -count=1 ./internal/leaguerules

# build and deploy to Claude-OS:
make build VERSION=dev COMMIT=ring4 BUILD_DATE=ring4
gzip -kf build/bin/thewarroom
scp build/bin/thewarroom.gz claudeos:warroom-ring1/thewarroom.new.gz
ssh claudeos 'cd ~/warroom-ring1 && gunzip -f thewarroom.new.gz && chmod +x thewarroom.new && bash deploy.sh r4n'
```

Repo rules the tests enforce: frontend controls must be `Act` (raw `<button>`/`onClick` fail
`controls.test.ts`); app source lines ≤110 chars (JSON fixtures exempt); entry bundle ≤321,818 B. Run the
**whole** vitest suite, not just `src/app/commissioner` — repo-wide scans live elsewhere.

Sol dispatch, if a builder is needed: `WT=<worktree> nohup ./dispatch-wt.sh <phase> par/<brief>.full.md >
<phase>.launch.log 2>&1 &` from the run directory. Alive = mtime of `<phase>/sessions/*.jsonl`; done =
non-empty `<phase>/sentinel`; report `<phase>/REPORT.md`. Sol never runs git.

---

## 8. Recorded at handoff (filled in at the moment of transfer)

See the "Handoff state" block appended at the end of this file.

---

## 9. History (only if you need the why)

- `~/fleet/runs/warroom-ring4-2026-10-13/RESUME.md` — full dated log of ring 4 (newest on top).
- `docs/build-handoffs/RESUME-2026-10-18-ring4-build.md` (+ the 10-13 plan and 10-14 build resumes).
- CT105 `/root/TheWarRoom-ring4-build-RESUME.md` — copy of the run-dir RESUME at 2026-10-18 22:20.
- Phase reports: `~/fleet/runs/warroom-ring4-2026-10-13/<phase>/REPORT.md` (c11a-preview, c11b-acttray,
  c11c-market, and earlier c1…c10 phases).

---

## Handoff state (2026-10-19 ~07:00 CT, written when Christopher cleared the Beelink session)

**Not yet committed** in `~/work/TheWarRoom/.worktrees/ring-4-integ2` (on disk, tested, mutants killed):
- `internal/commissioner/preview_native.go` — `seedChargeWords`: cap charges as "N cap charges totalling $X".
- `commissioner_preview_acts_test.go` — asserts the cap-charge summary.
- `frontend/src/app/commissioner/PreviewReview.tsx` — "1 value" wording.
- `frontend/src/app/commissioner/tray.test.tsx` — covers "1 value".
- `frontend/src/app/commissioner/SeedTray.tsx` — reason typing keeps the preview, re-asks Go after `seedPreviewPause` (400 ms).
- `frontend/src/app/commissioner/actTray.test.tsx` — new test for that; the old seed test waits the pause.
- This file (`docs/from-tom/…`).

**In flight:** `make verify` → `~/fleet/runs/warroom-ring4-2026-10-13/integ2/verify24.log` (nohup, started ~06:43,
survives the session). It started before the SeedTray edit, so its tsc step ran on the older SeedTray.

**Your first steps:**
1. `grep VERIFY_EXIT ~/fleet/runs/warroom-ring4-2026-10-13/integ2/verify24.log` (wait if absent and a `make verify`
   process is alive). If 0: run the frontend half (§7) to cover the SeedTray edit. If not 0: read the FAIL, fix, rerun.
2. Commit the code: message "Starting assets preview: cap totals, typed reason keeps the preview" (body: what the
   bullets above say), trailer `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`, via `git commit -F`.
3. Commit this file: "docs: ring 4 handoff to ClaudeBox".
4. The branch has **never been pushed** (no upstream). Push with `git push -u origin session/ring-4-integ2`; the
   pre-push hook runs `make verify` (~45 min), so run it detached and do not run a second verify at the same time.
5. Continue at §4 step 2 (deploy r4n, walk).

T434 on Hermes was updated at 06:55 (commit 4158f6b) to say the work is transferred and where this file is.
