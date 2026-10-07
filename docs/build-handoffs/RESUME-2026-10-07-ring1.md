# RESUME: TheWarRoom, ring 0 done → ring 1 planning (2026-10-07, ~08:15 CDT)

> Written by `/compact-safe` on ClaudeBox (CT105). The session continues after compaction:
> Christopher says "we are back", then start at §9 item 1.

## 1. What we are doing
- Building the approved UI target ring by ring. **Ring 0 (the walking skeleton) is done, live-gated
  ("pass", Christopher, 10-07) and merged: PR #14, rebase merge, main `3d01f33`.**
- **Next: ring 1 planning** (Christopher, 10-07: "after compaction we will work on ring 1").
- Same working model as ring 0 (his call, 10-07): **Claude is head brain; Sol 6.1 on Bee at
  medium thinking does the token-heavy coding, in small chunks.** Claude reviews every diff,
  applies what is good, fixes what is bad, reworks through Sol when the list is long. His laws:
  "excellence is the standard, not the goal", "snappy lightning fast UI/UX is law", "slop is
  banned". Watch Sol's long packed lines.
- **Where:** Beelink `chris@192.168.1.191` (the active repo). Repo `~/work/TheWarRoom`.
  - **Worktree `~/work/TheWarRoom/.worktrees/ring-0` is now on branch `session/ring-1`** (from
    main `3d01f33`, clean, not pushed). The directory name is historical: it was kept so the
    shared pre-commit hook and `node_modules` stay as they are. Do not `mv` it.
  - The main checkout stays on `session/league-history-pwa` (the Lab). Never mix the Lab and
    TheWarRoom.
  - `session/ring-0` is kept on origin: the commit hashes cited in ring 0 docs live there (the
    rebase merge rewrote them on main: 34399fc→50a9504, 1e5638a→4c72a2f, f22be20→64b2f73,
    f28e065→3d01f33).

## 2. Agents and harnesses
- **Bee = pi + Sol**: `--provider openai-codex --model gpt-6.1-sol --thinking medium`.
  - Ring 0 launcher: `~/fleet/runs/warroom-ring0-2026-10-06/dispatch.sh <phase> <brief>`. It has
    the run dir and worktree hard-coded; for ring 1, copy it to a new run dir
    (`~/fleet/runs/warroom-ring1-2026-10-07/`) and change `RUN` (WT path is unchanged).
  - Run it from CT105 with `ssh chris@192.168.1.191 '<dispatch.sh> pN <brief>'` as a background
    Bash; Claude is re-invoked when it exits. `nice 10`, `GOMAXPROCS=8`, 120 s startup watchdog.
  - Transcripts in `<run>/pN/sessions/*.jsonl` are the recovery channel.
- **Briefs** live in `~/fleet/briefs/warroom-ring0-*.md`. The pattern that worked:
  - every brief cites the P1 brief's **Where you work / Hard rules / Quality bar**;
  - plus a **Code standard** block (110-char gate, one statement per line, one field/prop per line
    past ~90 chars, no nested ternaries, comments say why);
  - plus a **Law** block (§18 budgets, honest data, live never falls back to fixtures);
  - plus exact gates and a six-heading REPORT.md.
  - Rework briefs are short numbered lists, each item with the test that proves it.
- **Claude's review loop per chunk:** read REPORT, read the diff, scan for added lines over 100
  chars, render on Claude-OS at 1280×800 when it is UI, run `make verify` itself, commit.
- **Renders:** start vite from the worktree `frontend/` detached
  (`setsid nohup nice -n 10 pnpm exec vite --host 127.0.0.1 --port 5199 --strictPort`), then
  `~/fleet/runs/warroom-ring0-2026-10-06/shot.sh '#/hq/my-moves' out.png 1280 800`. Headless
  cannot open overlays (calendar panel, inspector subject): those need the live gate or xdotool.

## 3. Status
| Item | State |
|---|---|
| Ring 0 P1–P5a | done 10-06 (see `docs/build-handoffs/RESUME-2026-10-06-ring0.md`) |
| P5b-1 Go `TargetDraftIR` / `TargetMoves` | done (main 50a9504) |
| P5b-2 clock strip + calendar panel | done (main 4c72a2f) |
| P5b-3 IR Act zone + My moves rails | done (main 64b2f73) |
| Ring 0 live gate on Claude-OS | **passed** 10-07 |
| Roadmap / HANDOFF / ring 0 resume §0 | updated (main 3d01f33) |
| Ring 1 | **not started: plan first** |

Gates at main 3d01f33: `make verify` green (Go race suite, golangci-lint 0, bloat at baseline,
188 frontend tests); entry chunk 318,816 / 325,000 bytes.

## 4. Artifacts
- Gate binary `~/work/TheWarRoom/.worktrees/ring-0/build/bin/thewarroom` (built from f22be20),
  sha256 `bcc1d9d78f8aea8fb4366bba87725b4ad4fe49ccaa6b24e538783e20a887cd4f`; copy on Claude-OS
  `~/warroom-ring0/thewarroom`.
- Backup-API DB copies `~/fleet/runs/warroom-ring0-2026-10-06/gate-db/`:
  - `history.db` sha256 `c3f30c1c…8842013`;
  - `thewarroom.db` `d04e0f20…2b998a4`;
  - `whatif.db` `854b6b52…cca78`.
  The same files are on Claude-OS `~/.config/TheWarRoom/` (the app's launch refresh has since
  written to the Claude-OS copies). The older Claude-OS data is at
  `~/.config/TheWarRoom.pre-ring0-gate-20261007`.
- Screenshots: `~/fleet/runs/warroom-ring0-2026-10-06/p5b2/shots/`, `p5b3/shots/` (gate-1.png is
  the live Wails app).

## 5. Current bug
None open.

## 6. Refuted / settled: do not retest
- **Wails cannot type `[]playerid.PlayerID`.** It emits an undefined `playerid` TS namespace and
  tsc fails. Fixed with `ts_type:"string[]"` / `ts_type:"string"` tags on the envelope fields.
  Never hand-edit `frontend/wailsjs/`. After `wails generate module`, restore
  `frontend/wailsjs/runtime` (mode-only diffs).
- **Commit hooks need the toolchain on PATH:** export
  `PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH` in the same ssh command as
  `git commit`. Without it ifaceguard says "go: No such file" and golangci-lint falls back to the
  pre-commit Go, which panics in buildir on stdlib "poll". Neither is a code defect.
- **Commit messages with apostrophes break `ssh '…'` quoting:** scp a message file and use
  `git commit -F`.
- **`pkill -f` inside ssh** matches its own shell if the pattern appears unbracketed anywhere on
  the command line. Kill by PID.
- **GitHub merge commits are disabled on this repo**; only rebase or squash.
- **The 30 s clock interval** showed a stale minute. The ticker now reschedules on minute flips,
  1 s inside the last hour, and runs no timer for passed or undated deadlines.
- **A 1280 px header overflows** when the strip carries a sentence: keep strip copy short, and keep
  a shrink and ellipsis rule on the strip's direct `.act` child.
- **The rail must not scroll horizontally** (it hid the current stage): stages are `flex: 1 1 0`.
- Ring 0 entries in `RESUME-2026-10-06-ring0.md` §6 still hold.

## 7. Decisions
**Christopher:**
- Sol on Bee at medium; Claude is head brain (10-06, reaffirmed 10-07).
- The ring 0 work passed (10-07).
- Push and merge granted 10-07 for PR #14.

**Claude's engineering calls (in code; carry them into ring 1):**
- A ring 0 IR draft's MFL target is **unmapped** (registry M-028 is spec-only), so every live
  draft ends `blocked`. Ring 1 verifies O=18 and maps it, so an envelope can reach Landed
  (ring 1 gate).
- Moves live in their own store; the shell store carries no domain data.
- Lazy chunks are prefetched on idle after the snapshot.
- No client copy of rules that Go checks own.
- The demo envelope shows only in fixture mode.
- The move audit log stays in memory until the first live Act persists it (ring 1).

## 8. Ledger state
- TheWarRoom: everything merged; `session/ring-1` is a fresh branch with this resume committed on
  it (not pushed).
- Run directory: `~/fleet/runs/warroom-ring0-2026-10-06/PLAN.md` has the 10-07 log.

## 9. Next actions
1. **Plan ring 1 with Christopher:** read `docs/build-handoffs/UI_Target_Roadmap_2026-10.md`
   › Ring 1 and `docs/ui/Target_UI_Spec_2026-10.md`. Propose phases sized for Sol chunks, with
   gap closure first:
   - league rules (allRules / By-Laws);
   - verify the MFL IR page O=18 (M-028) and other Act targets;
   - a dated current-week source (MFL nflSchedule / liveScoring);
   - his answers on waivers, DOT, windows, taxi/IR and Victory Points.
   Get his go on the plan before any brief. Plan mode beats act-fast.
2. **Set up the ring 1 run dir:** copy `dispatch.sh` and `shot.sh` and set `RUN`, and start
   `PLAN.md`.
3. **Then chunked briefs → Sol → review → commit**, as in ring 0. Live gate on Claude-OS, then
   push/merge only on his go.

## 10. Environment
- Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH
  GOMEMLIMIT=1500MiB GOMAXPROCS=8`; run `make verify VERSION=dev COMMIT=<tag>` under
  `nice -n 10`.
- `gh` is at `~/.local/bin/gh` on the Beelink, so add it to PATH.
- Claude-OS (`ssh claudeos` from the Beelink, Debian 13, 1280×800, Caps Lock ON, his):
  - it has no Go, Wails or pnpm, so binaries are built on the Beelink and copied over;
  - WebKitGTK 4.1 is present;
  - **the ring 0 gate app is still running there** (`pgrep -x thewarroom`). It is his review
    window: leave it unless he says otherwise.
- The claudeos-browser and claudeos-desktop MCP servers failed to connect this session; ssh,
  scrot and xdotool work.

## 11. Honest status
- Ring 0 is proven live: the snapshot, the clock over IPC, drafts and the rail.
- The calendar panel and inspector Act zone were checked by tests and by Christopher's live
  review, not by Claude's own screenshots.
- No §18 profiler timings exist yet; the architecture gates enforce them.
- Ring 1 has no plan yet. Its first live Acts touch MFL, so they need verified targets and
  Christopher's rules answers before coding.
