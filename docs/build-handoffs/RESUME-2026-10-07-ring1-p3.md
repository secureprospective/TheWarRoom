# RESUME: TheWarRoom ring 1, phase 3a running (2026-10-07, ~12:40 CDT)

> Written by `/compact-safe` on ClaudeBox (CT105). The session continues: Christopher says
> "we are back", then check §1b (in flight) and start at §9 item 1.
> Supersedes `RESUME-2026-10-07-ring1-p1.md` (kept for history).

## 1. What we are doing
- Ring 1 of the UI target (roadmap `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` › Ring 1):
  the frequent loops end to end with the first live Acts. Plan approved 10-07, phases:
  **0 gaps ✔ · 1 live season data ✔ (1a–1g) · 2 hand-off + landing ✔ (2a, 2b) · 3 HQ
  lineup/roster (3a running, 3b next) · 4 lineup.set · 5 trade desk · 6 Home + League Pulse Now ·
  7 gate.**
- Working model (Christopher): Claude is head brain; **Sol (gpt-6.1-sol) on Bee at medium** codes
  in small chunks; Claude reviews every line, fixes what is bad (all-or-nothing Python edit
  scripts scp'd to the run dir), proves each fix with a test that fails without it, runs
  `make verify`, commits. Laws: "excellence is the standard, not the goal", "snappy lightning
  fast UI/UX is law", "slop is banned".
- **Where:** Beelink `chris@192.168.1.191`, worktree `~/work/TheWarRoom/.worktrees/ring-0`, branch
  **`session/ring-1`** (never `mv` the worktree). **Not pushed.** `origin/main` is 3d01f33.

### 1b. In flight
- **Sol phase 3a** (Go: pure `internal/lineup` with `ParseRules`/`Check`, binding
  `TargetLineup(franchiseID)` in `lineup_app.go`; brief `~/fleet/briefs/warroom-ring1-p3a-lineup.md`).
  Started 12:31:35 CDT, timeout 3 h. pi pid 436468. Alive: transcript mtime
  `~/fleet/runs/warroom-ring1-2026-10-07/p3a/sessions/2026-10-07T17-31-35-656Z_ring1-p3a-20261007T173135Z.jsonl`;
  done: `…/p3a/sentinel`; output `…/p3a/REPORT.md`. The background notification does NOT survive
  compaction: poll the sentinel. Uncommitted in the worktree: only Sol's 3a files
  (`internal/lineup/`, `lineup_app.go`, more to come).
- **Claude-OS runs ring 1 at fd5ceab** (not yet 2a/2b; `~/warroom-ring1/thewarroom`, sha256
  dc5b31bb97edc307…, built `VERSION=ring1-dev COMMIT=fd5ceab`, pid 7557). Previous binary kept as
  `thewarroom.da0c0112`. Christopher's MFL key connected. Live check passed 11:49 (§3).

## 2. Agents and harnesses
- Dispatch: `ssh chris@192.168.1.191 '~/fleet/runs/warroom-ring1-2026-10-07/dispatch.sh <phase>
  ~/fleet/briefs/<brief>.md'` as a background Bash (it notifies on exit). Sentinel, REPORT,
  transcript in `<run>/<phase>/`. Briefs inherit `~/fleet/briefs/warroom-ring1-common.md`.
  Briefs this window: `p1f-season-feeds`, `p1g-season-wiring`, `p2a-move-log`,
  `p2b-handoff-watcher`, `p3a-lineup` (scratchpad copies too).
- Sol confines Go/lint caches to the run dir (`<phase>/go-cache`, `lint-cache`, `cache`): delete
  after review (chmod -R u+w first). Sol's pi tool also leaves `/tmp/pi-bash-*.log` on the
  Beelink (RAM): delete after review.
- Panels: non-Claude only (`panel.sh`: Nemotron 3 Ultra + GLM 5.3 + Sol).
- Review loop: read REPORT §4–6, read the whole diff, fix with a script, prove fixes with tests
  that fail against Sol's original (swap the original file back in, run, restore), `wails generate
  module -tags webkit2_41` when bindings change, then **chmod 644** the three
  `frontend/wailsjs/runtime/*` files (the generator flips them to 755), `make verify`, commit with
  `-F` message file. Sol's touched files trip the 110-char gate on pre-existing long lines, so
  rewraps in touched files are legitimate; rewraps in untouched files are churn: revert.

## 3. Status
| Item | State |
|---|---|
| p1f season-feed parsers (`internal/leaguefeed`, transactions, livescoring, pendingtrades.Fetch) | committed 69b814c |
| p1g held feeds, `TargetSeason`, `target:season`, key-change refresh, shutdown-cancel fix | committed fd5ceab |
| live check of 1g on Claude-OS (11:49) | **passed**: transactions, liveScoring W=5, pendingTrades (APIKEY=REDACTED) on www47, ~1 s apart, after nflSchedule; 0 unredacted APIKEY rows ever; run.log 0 apikey |
| p2a move store (`internal/store/moves`, append-only triggers), `envelope.Restore`, per-feed Observation | committed 1bc4773 |
| p2b `TargetHandOff`, `TargetCheckMoves`, landing watcher, `target:moves` | committed 155dac2 |
| p3a lineup legality + `TargetLineup` | **Sol running** |
| `make verify` at 155dac2 | green; entry chunk 322,536 / 325,000 |

## 4. Artifacts
- Run dir `~/fleet/runs/warroom-ring1-2026-10-07/`: `PLAN.md`, `data/mfl-exports/*.json` +
  `MANIFEST.txt` (league.json has `starters`: count 21, iop 8, idp 12, per-position limits),
  `data/mfl-auth/pendingTrades-2026-10-07.json` (league-private, never in the repo), fix scripts
  `p1f_fix.py`, `p1g_fix.py`, `p1g_tstype.py`, `p2a_fix.py`, `p2b_fix.py`, `season_check.py`
  (read-only fetch_log check, also on Claude-OS `~/warroom-ring1/`).
- Claude-OS: `~/warroom-ring1/launch.sh`, `run.log`, `season_check.py`. Ring 0 binary
  `~/warroom-ring0/`. Data backup `~/.config/TheWarRoom.pre-ring1-20261007`.

## 5. Current bug
None open. Watch: entry chunk headroom 2,464 bytes (3b adds frontend code: keep it lazy).

## 6. Refuted / settled: do not retest
- **Build stamp matters:** `make build VERSION=dev` makes the app open `*-dev.db` databases (by
  design, `dbFileName`). Live builds for Claude-OS use **`VERSION=ring1-dev`**. A dev build ran
  there for ~1 min at 11:48 on dev DBs; real DBs untouched; stray `thewarroom-dev.db`,
  `whatif-dev.db` left in `~/.config/TheWarRoom` (harmless).
- "first MFL refresh ready" in the startup log is a no-op when the mirror is populated; it is not
  evidence of a fetch. Check `fetch_log` (history.db, `fetched_at` is UTC RFC3339 with Z).
- Wails cannot type `playerid.PlayerID` (emits an undefined namespace): every bound field of that
  type needs `ts_type:"string"` / `"string[]"`. `time.Time` generates as `any` (accepted).
- `Mirror.AsOf` is the last roster **content change**, not the last fetch; the watcher uses
  `rostersCheckedAt` (set on a successful `refreshLeagueLocked`). `TargetSnapshot` still shows
  AsOf (pre-existing, not changed).
- My brief was wrong that every franchise has 21 starters: week 5 has 19–21 (partial lineups).
  liveScoring default week on 10-07 was week 4.
- Earlier rulings and refutations in `RESUME-2026-10-07-ring1-p1.md` §6 still hold.

## 7. Decisions
- **Christopher, 10-07:** key in OS keyring, never file/DB, no session-only fallback; phase
  automated; waivers on ProBoards; DOT review trades only; local time; MFL settings are the rules
  (ruling 6); Victory Points unused; trade flow ProBoards → MFL propose/accept → DOT → commissioner
  → Landed = public TRADE row; panels non-Claude.
- **Claude, this window:**
  - pendingTrades never falls back to the archive; no key → no request at all.
  - `TargetSeason`/`TargetLineup` never take `refreshMu`; bindings never wait on MFL.
  - A failed feed is no evidence (`ErrStaleObservation`, no transition), not "Not verified".
  - Hand-off saves before opening the browser; opens only https `*.myfantasyleague.com`.
  - Watcher: 60 s passes for 30 min after a hand-off, else after hourly season refresh; only the
    declared feeds. **Deferred to the phase 4 brief:** skip re-fetching a season feed fetched
    within the last minute *and* after the latest hand-off (the hourly season refresh wakes the
    watcher, which would fetch the same feeds again).
  - Lineup legality lives once, in Go (`internal/lineup`), from the rulebook's MFL starters;
    QB+RB+WR+TE ≤ 8 (PK outside), 12 defenders, 21 total; partial is legal but not full.

## 8. Ledger state
- Committed on `session/ring-1` this window: 69b814c, fd5ceab, 1bc4773, 155dac2 (+ this resume).
  Earlier: 38be4e3 … ee9f4cf, 749d6e1. **Nothing pushed.**
- Uncommitted: Sol's 3a work in progress.
- Task list (Hermes T406): updated in this compact-safe.

## 9. Next actions
1. **Check p3a** (sentinel). Review every line: rules parsed from the rulebook (never
   hard-coded), PK mapping reused from `internal/normalize` (not copied), `short`/`over`/`unknown`
   problem kinds, no `refreshMu`, no network, slices never null, `ts_type` on PlayerID fields.
   Fix, prove, `wails generate`, chmod 644, `make verify`, commit. Reap `p3a/` caches and
   `/tmp/pi-bash-*`.
2. **Brief 3b** (frontend): season + lineup contracts and strict parsers (lazy, out of the entry
   chunk), provider `lineup(franchiseId)` and `onSeasonChange` (`target:season`, the
   `onClockChange` pattern), HQ › Lineup and roster: replace the `lineup · ring 1` placeholder
   (`frontend/src/app/shell/FranchiseHQ.tsx`) with starters by position, bench, the legality line
   with problems, lineup provenance; IR/taxi groups stay. No layout shift; no re-render outside
   the section on season events.
3. Deploy to Claude-OS (`make build VERSION=ring1-dev COMMIT=<sha>`, swap with SIGTERM, launch)
   and have Christopher look at his lineup in HQ.
4. **Phase 4 `lineup.set`**: mapped target (`/lineup?L=&FRANCHISE=`, Gap Closure §1 M-027), draft
   + pre-flight via `internal/lineup.Check`, lineup predicate (exact starter set, requested week,
   post-hand-off liveScoring, baseline must differ), the deferred season-feed skip rule. **The
   ring 1 gate envelope: Christopher submits on MFL live.**
5. Phases 5–7. Push/merge only on Christopher's go.

## 10. Environment
- Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH
  GOMEMLIMIT=1500MiB GOMAXPROCS=8`; `nice -n 10 make verify VERSION=dev COMMIT=ring1`.
- Claude-OS from the Beelink: `ssh claudeos`; deploy pattern in §9.3 (old pid SIGTERM, wait,
  `mv`, `bash launch.sh`). `season_check.py <utc-since>` prints fetches without key values.
- Compact-safe 12:36: reaped ~7.7 GB (p1g cache, p2a/p2b go/lint caches); filing gate PASS.
- claudeos-browser / claudeos-desktop MCP servers fail to connect; ssh works.

## 11. Honest status
- Season feeds, the watcher and the hand-off are proven by tests and (feeds only) one live run;
  **no move has been handed off or landed live**: only `roster.ir` exists and its drafts stay
  blocked. Phase 4 is the first live hand-off.
- `TargetSeason` data is not shown in any UI yet (3b is the first consumer).
- Ring 1 remaining after 3a: 3b, 4, 5, 6, 7 gate.
