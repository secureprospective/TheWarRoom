# RESUME: TheWarRoom ring 0 build, head-brain session (2026-10-06, ~20:05 CDT)

## 1. What we are doing
- **Building ring 0 (the walking skeleton) of the approved UI target.** Christopher (10-06): "This
  will be a headbrain session... start with ring zero now. The coder will be GPT-6.1 Sol on Bee.
  You will spin Bee up with coding briefs... you review the code, fix whats not right, apply the
  code for me to review, rinse and repeat... in chunks or phases."
- **Where:**
  - repo `~/work/TheWarRoom`;
  - **worktree `~/work/TheWarRoom/.worktrees/ring-0`**, branch `session/ring-0` (from main 4cac98e).
    The branch has **no upstream and is not pushed**. `.worktrees/` is in `.git/info/exclude`.
  - The main checkout stays on the Lab branch (`session/league-history-pwa`), untouched (his
    rule: never mix the Lab and TheWarRoom).
- **Run directory:** `~/fleet/runs/warroom-ring0-2026-10-06/`
  - `PLAN.md` (phases + running log);
  - `dispatch.sh`, `shot.sh`;
  - `pN/` per phase (brief.md, REPORT.md, sessions/, shots/);
  - `data/` (read-only DB snapshots).

## 2. Agents and harnesses
- **Bee = pi + Sol 6.1**, provider openai-codex, model `gpt-6.1-sol`, **`--thinking medium`** (his call).
  - Launch: `~/fleet/runs/warroom-ring0-2026-10-06/dispatch.sh <phase> <brief.md>`, run in the
    background; I am re-invoked when it exits.
  - It runs under `nice 10` with `GOMAXPROCS=8`, and has a **120 s startup watchdog with one
    retry** (one run hung at startup on 10-06).
  - Session transcripts: `pN/sessions/*.jsonl`. These are the recovery channel.
- **Briefs:** `~/fleet/briefs/warroom-ring0-p{1,1b,2,3a,3b,3c,4,4-...-rulings,4r,5a}-*.md`.
  - Rules carried in every brief: no git, no GUI, no Beelink desktop, only the snapshot DBs plus
    test temp DBs, no dependency changes (unless named), the 110-character line gate, the bundle
    budget, `make lint VERSION=dev COMMIT=ring0`, Go with
    `GOFLAGS=-mod=readonly GOPROXY=off`, `-p 4`.
- **Claude** reviews the diff, fixes, renders, commits.
  - **Never commit while a Bee phase is running in the worktree:** the pre-commit hook stashes
    unstaged files.
- **Renders:**
  - **vite dev server** on 127.0.0.1:5199, started by Claude (PID 2640872), from the worktree's
    `frontend/`.
  - **Headless:** `shot.sh '<hash or ?query>' out.png [w] [h]` runs Chromium on Claude-OS through
    `ssh -R 5199`.
  - **Interactive desktop review on Claude-OS** (1280×800, Christopher logged in):
    - a persistent tunnel `ssh -f -N -R 5199:127.0.0.1:5199 claudeos`;
    - a Chromium `--app` window with `--user-data-dir=/tmp/twr-prof` on DISPLAY=:0;
    - driven with xdotool, captured with scrot into `/tmp/twr/`, then scp'd back.
    - **Never `pkill -f twr-prof` inside ssh** (it kills the ssh shell).
    - Claude-OS has **Caps Lock ON**; it is his, left alone.

## 3. Status (ring 0 phases)
| Phase | State | Commit |
|---|---|---|
| P1 data contract (one Go builder `internal/snapshot` → `cmd/fixtures` + `TargetSnapshot` binding; TS contract, parser, providers; vitest) | done | e867757 |
| P4r Sol's 442 rows reconciled into the registry | done | 9ac81d1 |
| P2 look (tokens, G0–G3, U0–U3, glyphs, Card, PlayerCard ×3 densities, dev specimen) | done; **approved by Christopher** ("looks good, i approve") | dcf2d8d |
| P3a command registry + `Act` + control scanner; shell, routes, harness switch | done | a7ab656 |
| P3b command bar, presets, my-franchise, Franchise HQ roster, inspector player view, 110-char gate | done | 046c9d0 |
| P3c performance (§18): density without re-render, fixture out of the bundle, bundle budget, CSS motion; polish | done | ae13374 |
| P4 endpoint registry normalized + generated + endpoint indexes + **≥2-routes gate test** + lazy harness | done | 4cb2541 |
| P5a Go league clock + move envelope + fixtures (reviewed and reworked by Claude, see §4) | done | this commit |
| **P5b UI: clock in the status strip / calendar panel, envelope rail in Franchise HQ › My moves** | **next: write the brief** | — |
| Ring 0 gate: Christopher's live review on Claude-OS (`make build`, backup-API snapshot) | after P5b | — |

Gates at 4cb2541:
- `make verify VERSION=dev COMMIT=ring0` passes: 101 frontend tests, the Go race suite, lint.
- Entry chunk 314,166 bytes against a budget of 325,000.

## 4. In flight (Step 1)
- **Nothing is running from Bee.** P5a exited 0 at 19:50; Claude reviewed it and reworked it:
  - **Clock:** the MFL schedule input is dropped (its weeks have no dates, and fetching it went to
    the network on every call). `TargetClock` (now in `target_app.go`) reads only the what-if
    store's phase log and commissioner calendar. Deadlines sort soonest first, undated last.
    There is no `currentWeek` and no `source` field.
  - **One clock builder:** `snapshot.BuildClock(ctx, at, season, ClockSource, provenance)`.
    `cmd/fixtures` gained **`-whatif`** and reads `data/whatif-snapshot.db`, a backup-API copy of
    `~/.config/TheWarRoom/whatif.db` taken at 19:52. The fixture clock reads **OFFSEASON, with no
    deadlines** (the real what-if store has 0 calendar events) and all seven windows unknown.
  - **Envelope:**
    - DOT, bid and waiver stages now follow hand-off, and Landed is terminal (Sol had Landed → DOT).
    - The spec validator no longer hard-codes IR.
    - Old evidence returns `ErrStaleObservation` and changes nothing (Sol degraded the state to
      not_verified).
    - An unmapped target keeps the check's own note.
    - `Observation.At` is gone; the time comes from the roster fetch.
    - The transition table is hand-written in the test (144 pairs, 47 transitions).
  - TS: `parseReceipt` is exported (5b needs it). PHASES has no `''`, and rosterStatus is any
    roster status.
  - `wails generate module` was run (only `go/` diffs kept). `make verify` passes: 116 frontend
    tests, lint 0 issues, bloat at baseline; the entry chunk is 314,177 / 325,000 bytes.
- **Fixture regen command:** `go run ./cmd/fixtures -db data/thewarroom-snapshot.db
  -history data/history-snapshot.db -whatif data/whatif-snapshot.db -out
  frontend/src/app/data/fixtures`, with paths under the run directory.
- **Long-lived helpers:**
  - vite (PID 2640872);
  - the ssh tunnel (`ssh -f -N -R 5199…`);
  - the Chromium review window on Claude-OS.
  - Leave them running until the ring 0 gate, then reap them.

## 5. Current bug
None open.

## 6. Refuted / settled: do not retest
- **Firefox headless `--screenshot`** fires before lazy imports resolve and captures "Loading…".
  Use Chromium on Claude-OS (`--virtual-time-budget`).
- **Command bar "not opening"** was a focus race, now fixed. Visibility was transitioning, so
  `focus()` failed; the hidden input also kept focus after close. Don't re-debug it.
- **Bee "hung" (P4 attempt 2):** pi sat idle in epoll with no sockets and no session file. A smoke
  call `pi -p … "Reply with exactly: OK"` answered in 11 s. It was transient; the watchdog now
  covers it.
- **Host load:** "claudebox almost crashed the host" (19:05). Evidence:
  - the Beelink load was 37 on 16 cores from **Bee's `go test -race ./...`**, not Claudebox;
  - Proxmox load was 0.5.
  - Fixed with nice, `GOMAXPROCS=8` and `-p 4`.
- **The mirror has no player names.** They come from the archived MFL players export in
  history.db (the snapshot at `data/history-snapshot.db`). Fixtures join on that.
- **`@types/node` 26 breaks TypeScript 4.9.** It is pinned to 20.11.30 through `pnpm.overrides`.
- **gosec G306 wants 0600 on the fixture write.** That is fine: git stores the file as 644.

## 7. Decisions
**Christopher (10-06):**
- the coder is Sol on Bee and Claude is head-brain;
- Bee runs at medium thinking;
- he approved P1–P2;
- **"snappy and fast is law"** (memory `warroom-snappy-and-fast-is-law`: the §18 budgets are
  failing gates; watch Sol's code density);
- Sol's packed long lines were a good catch;
- this is his first time coding with GPT, so keep an eye on it.

**Claude's engineering calls:**
- **Data:** one builder serves the fixture and live outlets. Fixtures come from real snapshots and
  are never invented. Live never falls back to fixtures.
- **Code location:** the target UI lives in `frontend/src/app/`; the harness is `components/`,
  lazy-loaded behind `harness.open`.
- **Commands:**
  - every control is an `Act` verb;
  - a scanner forbids raw buttons and handlers;
  - the 110-character gate is a test;
  - density is a root attribute that React never re-renders on.
- **Speed:** the bundle budget runs in `make verify`.
- **Registry:**
  - placement uses a closed vocabulary (spec §4 node › workspace, or a surface);
  - a merged row has `merged_into` ids or `merged_place`;
  - the 11 place-merges and L-34 are ruled in `warroom-ring0-p4-endpoint-registry-rulings.md`.
  - **Provisional:** S-01 nav rail → Home › Seasonal card; S-10 → Control Room › App.
- **My franchise** is app-local in localStorage, under `thewarroom.target.my-franchise`.
- **Envelope:** persistence (a SQLite audit table) comes with the first live Act in ring 1; ring 0
  keeps the audit log in memory.
- **League windows** are not in the data. They are shown as `unknown` until gap closure captures
  the rules.

## 8. Ledger state
- `session/ring-0` holds the commits above (e867757 … 4cb2541), all through the pre-commit hook.
  Nothing is pushed or merged, and that needs his go.
- This RESUME is committed as `docs/build-handoffs/RESUME-2026-10-06-ring0.md` with the P5a
  commit, and kept on CT105 (`/root/TheWarRoom-Ring0-RESUME.md`) and in the run directory.
- The Lab's uncommitted rounds 4–9b remain parked on the main checkout (do not touch).

## 9. Next actions
1. **Write and dispatch P5b** (the UI):
   - **Clock:** the status strip shows the phase and the next deadline as a U0–U3 countdown. One
     timer, no React re-render storm (CSS/data-attribute, like density). With no deadlines it says
     so plainly.
   - **Calendar panel:** lists deadlines, and the seven windows as "unknown · gap closure".
   - **Envelope rail** in Franchise HQ › My moves replays `envelope-demo.json`, loaded with a
     dynamic import.
   - **`roster.ir` draft verb** from the inspector:
     - a Go binding (for example `TargetDraftIR(franchise, player)`) runs `envelope.New` +
       `IRCheck` against the current snapshot, keeps the result in the in-memory log, and returns
       a receipt;
     - in plain-browser fixture mode the verb is disabled, with a reason;
     - the MFL hand-off is not wired in ring 0.
   - Hold-to-fire is only for G3, and none exist in ring 0.
2. **Ring 0 gate:**
   - `make build` on Claude-OS against a backup-API snapshot of the live DB (R12);
   - Christopher reviews live;
   - update the `UI_Target_Roadmap` ring 0 status and spec revision notes.
3. Ask Christopher before any push or merge.
4. **Then:** ring 1 planning, plus gap closure (league rules from allRules / By-Laws; the mflmap
   owner-form capture; his answers on waivers, DOT, windows, taxi/IR and Victory Points). A dated
   source for the current week (for example MFL nflSchedule / liveScoring) belongs to ring 1.
5. **Gap to note:** the Go code has 595 existing lines over 110 characters, so there is no Go
   line gate. New files are kept within 110 by review.

## 10. Environment
- Toolchain:
  `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8`; run
  `make verify` under `nice -n 10`.
- A fresh worktree needs `cd frontend && pnpm install --frozen-lockfile && pnpm build` before any
  push (the ifaceguard lint embeds `frontend/dist`).
- Claude-OS: libvirt `Claude-OS` under qemu:///session, reached with `ssh claudeos`. Leave it
  running.
- Proxmox 200 is the business; it was not touched.

## 11. Honest status
- **Rendered and checked on Claude-OS:** P2–P4 on fixtures in a plain browser through vite.
- **Not yet run:**
  - **Wails/live mode** (`TargetSnapshot` over IPC, the lazy harness inside Wails, the fixture
    chunk not fetched in Wails); the ring 0 gate on Claude-OS covers these;
  - the §18 timings, measured by a profiler; only the architecture gates enforce them so far.
- **ETA to the ring 0 gate:** P5a review, plus P5b (one Bee run and a review), plus the live gate.
  That is roughly 1–2 hours of wall time.
