# RESUME — TheWarRoom, after the power-rankings work (2026-10-04 night)

## 0. Next actions, in order
Nothing is running that needs you. PR #13 (power rankings) is merged: main = `9f6f0a0`.
1. **Listen first: Christopher has "a very interesting take" to discuss. It is PLANNING ONLY.
   Build nothing until he approves it.** Ask for it if he doesn't open with it. Hand him
   questions as a matrix with a recommendation, and actions as numbered steps.
2. **Queued build (his ruling 2026-10-04): remove NFL contract data.** His words: "we dont want
   NFL contract data, means nothing to us, only in game contracts matter." Real-NFL contracts
   (OverTheCap via nflverse, Core Build Plan R9-4/R9-5, PR #12) come out. In-game MFL contracts
   stay, e.g. cap room on the power board. Do this only after item 1 and when he says go, unless
   he folds it into his plan. Footprint to remove:
   - `internal/ingestion/contracts/` (Parquet reader). `parquet-go` is used only there, so drop
     it from `go.mod` (`go mod tidy`).
   - `signals_app.go`: `loadContracts`, `contractsFeed`, `contractsFirstSeason`, the feed view
     (around lines 181, 190 and 417).
   - The model:
     - `model.Player.Contracts`, `contractPrfx`, `contractMeasures`, `Tenure`/`TenureAt`
       (`internal/model/data.go`).
     - `SurvivalRow`'s contract terms, `SurvivalTerms`, `ReadsContract`, the `Survives(…, t
       Tenure)` argument (`params.go`); `value.go` survives; `modelrun.inputNames` "contract".
     - The fit: `fit/survival.go`'s with/without-contract comparison (`survivalRows`, `Contract`,
       `LogLossNoContract`); `dynasty.go` and `boardcheck.go` pass tenure.
   - The registry: `measures.csv` `context.contract_*` (4 rows) and `source_fields.csv`
     `nflverse,contracts.*`; `measures_test.go` covers `contracts.SourceURL`.
     `make measure-dictionary`.
   - The docs:
     - `Approved_Sources.md`: v1.3 approved nflverse contracts; move it to not-adopted with his
       reason.
     - `SYSTEM_MAP.md` rows.
     - A plan ruling that supersedes R9-4/R9-5.
   - **Refit:**
     `go run ./cmd/fit -db ~/scratch/twr-research/fit.db -players ~/scratch/twr-research/players.json`.
     Survival goes back to age, draft pick, games and percentile; `fitted.json` and
     `docs/fit/Fit_Report.md` regenerate.
   - **Watch:**
     - History is append-only, so `context.contract_*` observations already stored stay put.
       Check that dropping the measures from the registry doesn't fail reading or validating
       stored rows; if it would, keep the rows as retired measures rather than deleting history.
       His real DB has 0 contract observations (he never ran the r9 build), but the gate
       snapshots and `fit.db` have them.
   - Then gate, PR, and merge only on "merge N".
3. Optional, low priority: he hasn't yet launched the build with the scouting work (contract count
   0 in his DB). Contracts no longer matter, but a Score League on v0.5.0-199 would refresh his
   model run with the DE/DT prior work. Verify read-only with `sqlite3 -readonly`.

## 1. State
- **main = `9f6f0a0`** (v0.5.0-199). PR #13 merged 2026-10-04 (Core Build Plan R10-1…R10-7,
  gate record PASS):
  - All-play is read from MFL's `all_play_wlt`. It never worked before: MFL never sent
    `all_play_w/l/t`.
  - This season counts the roster as the best legal lineup (`powerrankings.Lineup`) by default.
    The franchise view counts the whole roster by default. The toggle offers lineup / top N /
    whole roster in both.
  - Auto roster weight 4 ÷ (4 + weeks played); the slider overrides it, and the Auto button
    restores it.
  - Franchise score = z(Dyn roster) − 0.5·z(value-weighted age).
  - Context columns, never in the score: Age, Proj (projected record, σ 50), Luck, Cap room (dead
    cap in the tooltip). `m2service.Outlooks`, `state.Mirror.DeadCap`.
  - `GetPowerRankings(weight, autoWeight, aggMode, view)`; `aggMode` "" means the view's default.
- `build/bin/thewarroom` on the Beelink = **v0.5.0-199-g9f6f0a0** (main), rebuilt after the merge.
  **Caution:** `make build` writes this path. When building an unmerged branch, copy the result
  away and restore main's binary.
- The factor table and evidence: `docs/modules/M2_Power_Ranking_Factors.md`;
  `~/fleet/runs/warroom-power-rankings-2026-10-04/`:
  - Scripts: `analyze.py`, `run.py`, `run2.py`, `power2-4.py`, `scratch_power_test.go`.
  - Results: `results*.txt`, `values.csv` (49,629 player-weeks of real Now/Dyn).
  - MFL exports 2021–2026: `mfl-exports/`.
  - Screenshots: `live-gate/`.
- Task list T378 (Hermes) updated this session.
- Memory saved: `warroom-only-in-game-contracts-matter`.

## 2. Environment
- Go: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`.
- **Gates:**
  - `make lint`: bloat baseline comment 13, provenance 0, **tiny 22** (under 40 code lines counts
    as tiny, so fold small files into their homes), dupl 0; gocyclo 15.
  - `make verify` runs pre-push.
  - The pre-commit hook lints the whole tree with unstaged changes stashed but untracked files
    present, so commit code that depends on untracked files together.
- **Bindings:** `wails generate module -tags webkit2_41` regenerates `frontend/wailsjs`. Revert the
  mode-only changes it makes under `frontend/wailsjs/runtime`.
- **Merge:** `git push origin origin/<branch>:refs/heads/main` (fast-forward), on his "merge N" or
  equivalent.
- **Claude-OS** (`ssh claudeos`; leave it running): `~/twr-gate/thewarroom-r10c`
  (v0.5.0-198) is running on a backup-API snapshot of his DBs taken 18:49, in
  `~/.config/TheWarRoom`. Earlier gate data: `~/.config/TheWarRoom.gate-r9b`, `.gate-2026-10-04`,
  `.snap-1422`.
  - Launch: `cd ~/twr-gate && . ./cfbd.env && (DISPLAY=:0 setsid -f ./<bin> > <log> 2>&1)`.
  - Resize: `xdotool windowmove $W 0 25 windowsize $W 1280 775`; Pulse is at (45,362).
  - Stop it with **`pkill -x <name>`**. `pkill -f <pattern>` matches the ssh shell's own command
    line and kills it (exit 144/255; happened twice).
- **MFL history API:** `https://www47.myfantasyleague.com/<year>/export?TYPE=<t>&L=14432&JSON=1`
  serves past seasons (weeklyResults per week with every roster, starters and scores;
  leagueStandings; rosters with salaries; draftResults; futureDraftPicks; schedule). It throttles
  with a 4-byte "No" body when more than about 2 requests run in parallel: fetch one at a time
  and validate the body is JSON.
- Real DB: `~/.config/TheWarRoom/{thewarroom,history,whatif}.db`. Read only with
  `sqlite3 -readonly` or the backup API. Never run the production binary on the Beelink.
  Fitting DB: `~/scratch/twr-research/fit.db`.

## 3. Refuted / settled — do not retest
- **Power factors, Legacy NFL 2021–2025, held-out seasons:**
  - Not inputs: depth (bench) as an input; start/sit efficiency (season to season r 0.23); recent
    form; head-to-head record (worst signal); remaining schedule as strength (it lives only in
    Proj); draft picks owned (no signal within 2 seasons); expiring contracts.
  - Capping the franchise age term scores worse on every held-out season. The Dolphins at 9th
    (youngest roster, 23.3) is a known, evidenced effect.
  - On the real Dyn, the whole roster beats the lineup for the franchise view.
- Scouting (2026-10-04 earlier): see `RESUME-2026-10-04-power-rankings.md` §3. Do not retest.
- **Christopher's rulings:**
  - Horizon 3 seasons at 0.75.
  - FantasyPros is a benchmark only.
  - A source joins `Approved_Sources.md` only after it passes.
  - Power-ranking calls 1A / 2B / 3B / 4B.
  - **NFL contract data is out; only in-game contracts matter.**

## 4. Open, not started
- The NFL contracts removal (§0 item 2).
- His planning topic (§0 item 1).
- DynastyProcess issue for 17767's bad ids: only on his OK.
- League Controls shows OFFSEASON mid-season (deferred suite).
- The Transact roster pane is narrower than its table.
- Model limits: kickers aren't modeled, so the K slot counts 0 for everyone; a rookie who sits
  keeps his draft-day debut chance.
- The franchise view's "Top N" chip shows "N" (StarterN is echoed only for top N and lineup).
  Cosmetic.
