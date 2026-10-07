# RESUME: TheWarRoom ring 1, mid phase 1 (2026-10-07, ~09:30 CDT)

> Written by `/compact-safe` on ClaudeBox (CT105). The session continues: Christopher says
> "we are back", then check §1b (in flight) and start at §9 item 1.

## 1. What we are doing
- Ring 1 of the UI target (roadmap `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` › Ring 1):
  the frequent loops end to end with the first live Acts. Plan approved 10-07, phases:
  **0 gaps ✔ · 1 live season data (1a ✔ 1b ✔ 1c running, 1d briefed, 1e next) · 2 hand-off +
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
- **Sol phase 1c** (frontend: week + lineup lock in strip and calendar, `target:clock` event).
  Started 09:21 CDT, timeout 3 h. Alive check:
  `ssh chris@192.168.1.191 'ls -la ~/fleet/runs/warroom-ring1-2026-10-07/p1c/sessions/'`
  (transcript mtime advancing) or `cat …/p1c/sentinel` (exists when done).
  Output: `…/p1c/REPORT.md`; transcript `…/p1c/sessions/*.jsonl` is the recovery channel.
  The Claude background-task notification will NOT survive compaction: poll the sentinel.
- **Uncommitted in the worktree:** Sol's 1c edits (frontend clock/data files) AND Claude's
  ruling-8 edit to `docs/build-handoffs/Ring1_Gap_Closure.md` (§0 item 8, trade flow).
  Commit the doc with the 1c review, or on its own.
- Ring 0 gate app still runs on Claude-OS (`pgrep -x thewarroom`): Christopher's window, leave it.

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
| p0a gap closure doc | committed df94d4e (+ ruling 8 uncommitted) |
| p1a nflschedule / leagueweek / clock lineup lock | committed f5822cd |
| p1b week refresh worker, derived phase, TargetClock | committed 5687c7c |
| wailsjs models (`Reading.week?`) | committed 96e2da1 |
| p1c frontend week + lock | **Sol running** |
| p1d MFL key (Go) | brief ready; **add go-keyring dep first** (§9) |
| `make verify` at 96e2da1 | green; entry chunk 318,816 / 325,000 |

## 4. Artifacts
- Run dir `~/fleet/runs/warroom-ring1-2026-10-07/`: `PLAN.md` (log + rulings), `data/`
  (`mfl-exports/*.json` with `MANIFEST.txt`; `mfl-pages/*.json` incl. 0366 IR page and 0160
  league settings O=26), `panel-key/` (three answers + `TRIAGE.md`, the key-storage spec),
  `p0a/`…`p1c/`, fix scripts `p1a_fix.py`, `p1b_fix.py`, `ruling8.py`.
- Export sha256 prefixes: league c781e007, rosters 176d5b37, liveScoring 05432539,
  liveScoring-w5 d66d8faa, nflSchedule 2c4bef56, nflSchedule-w5 a81c99df, transactions
  4297d2d6, rules ec783af5, injuries 341ac653, nflByeWeeks 4ddb0e2e.

## 5. Current bug
None open.

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
- Committed on `session/ring-1` (not pushed): 38be4e3, df94d4e, f5822cd, 5687c7c, 96e2da1.
- Uncommitted: Sol's p1c work (in progress) and the ruling-8 doc edit.
- Task list on Hermes: no new items this window (T406 ring 1 already open).

## 9. Next actions
1. **Check p1c** (sentinel). Review REPORT + diff (strip copy at 1280 px, no re-render between
   readings, latest-wins hook, stale chip, note in panel, no churn), render on Claude-OS
   (`shot.sh '#/hq/my-moves' out.png 1280 800` against a vite dev server, see ring 0 resume §2),
   `make verify`, commit together with the ruling-8 doc edit.
2. **Add the dependency:** in the worktree, `go get github.com/zalando/go-keyring@v0.2.8`
   (bumps godbus to v5.2.2), confirm `make verify` green, commit; then dispatch
   `p1d ~/fleet/briefs/warroom-ring1-p1d-mfl-key.md`. Review the leak gate line by line;
   `go mod tidy` at review; `wails generate module` after (new bindings).
3. **Brief p1e** (frontend): Settings surface for the key (registered command, password field,
   one-way `SetMFLKey`, status only), then Christopher enters his key (Help → Developer's API on
   MFL) on the live app; capture a real `pendingTrades` / `calendar` body to close Gap Closure
   §7.3–7.6, 7.13.
4. **Phase 1 rest:** liveScoring + transactions + pendingTrades into the snapshot (Sourced).
5. Then phases 2–7 per §1. Live gate on Claude-OS; push/merge only on Christopher's go.

## 10. Environment
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
- No MFL auth exists yet; every trade surface depends on p1d/p1e and Christopher's key.
- No ring 1 Act has landed; the gate envelope (lineup.set) is phase 4.
