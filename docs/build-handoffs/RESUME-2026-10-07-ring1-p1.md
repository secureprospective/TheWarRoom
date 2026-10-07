# RESUME: TheWarRoom ring 1, mid phase 1 (2026-10-07, updated ~10:35 CDT)

> Written by `/compact-safe` on ClaudeBox (CT105). The session continues: Christopher says
> "we are back", then check §1b (in flight) and start at §9 item 1.

## 1. What we are doing
- Ring 1 of the UI target (roadmap `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` › Ring 1):
  the frequent loops end to end with the first live Acts. Plan approved 10-07, phases:
  **0 gaps ✔ · 1 live season data (1a–1e ✔, 1f running, 1g next) · 2 hand-off +
  landing watcher · 3 HQ lineup/roster + IR/taxi · 4 lineup.set · 5 trade desk · 6 Home + League
  Pulse Now · 7 gate.**
- Working model (Christopher): Claude is head brain; **Sol (gpt-6.1-sol) on Bee at medium** does
  the coding in small chunks; Claude reviews every diff, fixes what is bad, commits. Laws:
  "excellence is the standard, not the goal", "snappy lightning fast UI/UX is law", "slop is
  banned". Watch Sol's churn (it rewraps/rewords unrelated lines) and packed lines.
- **Where:** Beelink `chris@192.168.1.191`, worktree `~/work/TheWarRoom/.worktrees/ring-0` on
  branch **`session/ring-1`** (directory name is historical; never `mv` it). Not pushed.
  `origin/main` is 3d01f33 (ring 0 merged); the local `main` ref is stale, compare against
  `origin/main`.

### 1b. In flight
- **Sol phase 1f** (Go, pure: `internal/leaguefeed` + ingestion `transactions`, `livescoring`,
  `pendingtrades.Fetch`; brief `~/fleet/briefs/warroom-ring1-p1f-season-feeds.md`). Started
  10:32 CDT, timeout 3 h. Alive: transcript mtime in `…/p1f/sessions/`; done: `…/p1f/sentinel`.
  Output `…/p1f/REPORT.md`. Background notification does NOT survive compaction: poll.
  pi pid 315765 (timeout wrapper, 10800 s); transcript
  `…/p1f/sessions/2026-10-07T15-32-58-855Z_ring1-p1f-20261007T153258Z.jsonl` (recovery channel).
  Expect Sol to create `.phase-tmp/` or run-dir caches again: delete after review, never commit.
- **Uncommitted in the worktree:** only Sol's 1f work.
- **Claude-OS runs the ring 1 build** (`~/warroom-ring1/thewarroom`, sha256 da0c0112…, built at
  ee9f4cf; launch `bash ~/warroom-ring1/launch.sh`; log `~/warroom-ring1/run.log`). Ring 0 binary
  kept at `~/warroom-ring0/`. Data backup before ring 1: `~/.config/TheWarRoom.pre-ring1-20261007`.
  **Christopher's MFL key is connected** (keyring "Login", verified 10:23:40 CDT); live strip
  shows `Regular season · Wk 5 · Lineup lock`; the app recorded OFFSEASON → REGULAR_SEASON.

## 2. Agents and harnesses
- Dispatch: `ssh chris@192.168.1.191 '~/fleet/runs/warroom-ring1-2026-10-07/dispatch.sh <phase>
  ~/fleet/briefs/<brief>.md'` as a background Bash. Sentinel + REPORT + transcript in
  `<run>/<phase>/`.
- Briefs: `~/fleet/briefs/warroom-ring1-common.md` (rules every brief inherits: where, hard
  rules, law, code standard, report format) + `warroom-ring1-p0a-gaps.md`, `p1a-week.md`,
  `p1b-week-wiring.md`, `p1c-clock-week-ui.md`, **`p1d-mfl-key.md` (written, not dispatched)**.
  Scratchpad copies under the session scratchpad.
- **Expert panels run on non-Claude models** (Christopher 10-07; memory
  `feedback_panel_models_not_claude`): `~/fleet/runs/warroom-ring1-2026-10-07/panel.sh <brief>
  <outdir>` runs Nemotron 3 Ultra + GLM 5.3 (pi provider `nvidia`) + Sol in parallel.
- Read-only MFL page capture: on Claude-OS `MFLMAP_RUN=<run> python3 ~/bin/mflmap.py go <url>
  <label>` (via `ssh claudeos` from the Beelink); it drives Christopher's logged-in Brave
  (franchise 0025, Arizona Cardinals) and refuses submits.
- Review loop per chunk: read REPORT, read the diff, fix with an all-or-nothing Python edit
  script scp'd to the run dir (never inline nested heredocs), `make verify`, commit with `-F`
  message file and PATH exported.

## 3. Status
| Item | State |
|---|---|
| p0a gap closure doc | committed df94d4e; ruling 8 in 14f3fc2 |
| p1a nflschedule / leagueweek / clock lineup lock | committed f5822cd |
| p1b week refresh worker, derived phase, TargetClock | committed 5687c7c |
| wailsjs models (`Reading.week?`) | committed 96e2da1 |
| p1c frontend week + lock | committed 9ad8322 (+ test fix 07f8328) |
| go-keyring v0.2.8 | committed b0751c1 |
| p1d MFL key (Go: keyring store, keyed requests, bindings, leak gate) | committed 892be7b |
| p1e connect MFL UI (Control Room › App) | committed 8395160 |
| strip wraps instead of clipping | committed 099ca61 |
| window maximised 1280x800, surface background | committed ee9f4cf |
| **live key test on Claude-OS** | **passed**: Connected; fetch_log has 0 unredacted APIKEY; log clean |
| p1f season feeds (Go parsers) | **Sol running** |
| `make verify` at ee9f4cf | green; entry chunk 322,536 / 325,000 (2,464 headroom) |

## 4. Artifacts
- Run dir `~/fleet/runs/warroom-ring1-2026-10-07/`: `PLAN.md` (log + rulings), `data/`
  (`mfl-exports/*.json` with `MANIFEST.txt`; `mfl-pages/*.json` incl. 0366 IR page and 0160
  league settings O=26), `panel-key/` (three answers + `TRIAGE.md`, the key-storage spec),
  `p0a/`…`p1c/`, fix scripts `p1a_fix.py`, `p1b_fix.py`, `ruling8.py`.
- Export sha256 prefixes: league c781e007, rosters 176d5b37, liveScoring 05432539,
  liveScoring-w5 d66d8faa, nflSchedule 2c4bef56, nflSchedule-w5 a81c99df, transactions
  4297d2d6, rules ec783af5, injuries 341ac653, nflByeWeeks 4ddb0e2e.

## 5. Current bug
None open. Watch: entry chunk headroom is 2,464 bytes; new eager code must be justified.
Real pendingTrades body: `~/fleet/runs/…/data/mfl-auth/pendingTrades-2026-10-07.json` (sha256
daa78e9d…), **league-private, never in the repo**. Shape: one trade = object (array when
several), `will_give_up`/`will_receive` from the OFFERING team's side, trailing commas,
epoch-second strings; it does not prove the key's franchise. Pick tokens (MFL docs):
`DP_r_p` current year, round/pick one less than actual; `FP_fid_year_round` actual round;
`BB_x` blind-bid dollars.

## 6. Refuted / settled: do not retest
- **"nflSchedule goes to the league host":** refuted. `mfl.Client.Do` sends any request without
  `L` to the api host; `TestFetchUsesAPIHostWithoutLeague` covers it. (MFL does reject
  nflSchedule on www47: "must go to api.myfantasyleague.com".)
- **go-keyring "unmaintained since 2021 / no test double"** (Nemotron): false. Proxy: v0.2.8
  released 2026-03-23, `MockInit`/`MockInitWithError`, linux build needs no cgo, unlocks via the
  Secret Service prompt; no context support (wrap with timeout). 99designs/keyring last
  released 2022-12 and ships a file backend.
- **"MFL league ID changes each season"** (GLM): false for Legacy NFL (14432 in 2025 and 2026).
- The archive transport already redacts query keys containing "key" (`originURL`); but
  `mfl.Client` wraps `*url.Error` (full URL) with `%w`: keyed requests must scrub (p1d brief B).
- MFL public without login: league, rosters, liveScoring, transactions, rules, nflSchedule,
  nflByeWeeks, injuries. Login required: pendingTrades, calendar.
- liveScoring `W=5` already lists franchise 0025's 21 saved starters: future-week lineup
  landing source exists. Week 5 first kickoff 2026-10-09 00:15 UTC (Thu 19:15 CDT); unplayed
  games report `gameSecondsRemaining` 3600, final ones 0.
- Ring 0 settled items (resume `RESUME-2026-10-07-ring1.md` §6) still hold.

## 7. Decisions
**Christopher, 10-07:** (1) MFL API key in the OS keyring, never file/DB; (2) phase automated;
(3) waivers run on ProBoards, TheWarRoom builds waiver code natively; (4) DOT review trades only;
(5) local time zone; (6) MFL settings are the IR/taxi rules, don't over-engineer (NFL feeds after
MFL); (7) Victory Points not used; (8) **trade flow:** agree on ProBoards → propose/accept on MFL
→ DOT 3 approvals → commissioner approves on MFL → Landed = public TRADE row; (9) key panel
verdict adopted: go-keyring, fail closed, **no session-only key**; (10) panels non-Claude.
**Claude's calls (Gap Closure §8):** phase derived inside a season, **rollover never
automatic**; revoke hands off to the desk (O=05), never `ACTION=revoke`; **lineup.set is the
ring 1 gate envelope**; IR placement and taxi demotion stay `spec` (no eligible row seen);
TargetClock never touches the network; a missing/failed schedule degrades to `stale`, never
`fail`.

## 8. Ledger state
- Committed on `session/ring-1` (not pushed): 38be4e3, df94d4e, f5822cd, 5687c7c, 96e2da1,
  14f3fc2, bae1b29, 9ad8322, b0751c1, 07f8328, 892be7b, 8395160, 099ca61, ee9f4cf.
- Uncommitted: Sol's p1f work (in progress).
- Task list on Hermes: no new items this window (T406 ring 1 already open).

## 9. Next actions
1. **Check p1f** (sentinel), review every line (asset parsing against MFL docs, unknown
   transaction kinds kept, no private data in testdata), `make verify`, commit.
2. **Brief p1g** (Go wiring, the week pattern): background cache of transactions, liveScoring
   (lineup week) and pendingTrades (keyed, only when a key is stored); binding `TargetSeason()`
   returning Sourced sections, no network; Wails event `target:season`; stale/fail honest notes;
   refresh hourly with the week worker under `refreshMu`, spaced ≥1 s.
3. Frontend contract for the season feeds when the first consumer lands (phase 3 HQ lineup,
   phase 5 trade desk shows the real pending offer, phase 6 League Pulse feed).
4. Then phases 2–7 per §1. Push/merge only on Christopher's go.

## 10. Environment
- Compact-safe 10:35: reaped 2.6 GB of Go/lint caches from p1a/p1b/p1c/p1e run dirs (reports,
  transcripts, logs, shots kept). Filing gate PASS.

- Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH
  GOMEMLIMIT=1500MiB GOMAXPROCS=8`; `nice -n 10 make verify VERSION=dev COMMIT=ring1`.
- MFL curl needs `-A "TheWarRoom/dev"`; league host `www47`; space requests ≥1–2 s.
- Auto-mode classifier failed with "no verdict" repeatedly at session start; Christopher left
  auto mode. If it recurs, ask him to leave auto mode rather than retrying 10 times.
- claudeos-browser / claudeos-desktop MCP servers fail to connect; ssh + mflmap + scrot work.

## 11. Honest status
- The week, lineup lock and derived phase are tested with real MFL files but **never run in the
  live app**: Wails event delivery, the hourly ticker and the first real AdvancePhase on the
  what-if store are unproven until the next live run.
- MFL auth works live (one real keyed pendingTrades export, clean archive). The trade desk
  itself is phase 5.
- No ring 1 Act has landed; the gate envelope (lineup.set) is phase 4.
