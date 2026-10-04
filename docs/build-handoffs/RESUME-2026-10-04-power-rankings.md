# RESUME — TheWarRoom, after the scouting work (2026-10-04 evening)

## 0. Next actions, in order
Nothing is running that needs you. The scouting robustness work is merged (PR #12).
Christopher's next focus (2026-10-04): **the additional things that should affect the power
rankings.**
1. Confirm the first real-data run. Ask Christopher whether he has launched
   `~/work/TheWarRoom/build/bin/thewarroom` (v0.5.0-192-ge922753) and clicked Score League. His
   real DB had model run #2 at eb4cbb4; the first launch on the new build loads contracts (11 MB,
   one fetch) and re-reads CFBD 2016–2025 from the archive (no CFBD calls). Verify read-only with
   `sqlite3 -readonly ~/.config/TheWarRoom/history.db "select run_id, engine from model_runs"`
   (expect a run at e922753) and
   `"select count(*) from observations where measure='context.contract_cap_pct'"` (~20,600).
2. **Power-ranking factors: establish from evidence, then present a matrix with a
   recommendation** (product calls are Christopher's). Work on branch
   `session/power-ranking-factors` (holds only this file). Today M2 is the roster sum (or top-N
   starters) of Now or Dyn PPG, z-blended (median/MAD) with results (all-play if MFL reports it,
   else points for ÷ the best), 60/40 default on This season; The franchise is roster alone
   (R8-1…R8-7). Candidates, with where the data is:
   - **Cap health:** cap space, dead cap, contract years left. The league mirror
     (`thewarroom.db` `league_mirror.snapshot` JSON `Players[]` carries `Salary`, `ContractYear`,
     `ContractStatus`, `FranchiseID`); `salaryadjustments` ingestion exists; the cap amount comes
     from the league export (`internal/ingestion/league`).
   - **Lineup-slot fit:** value the starters the league's lineup rules actually start, not a flat
     sum. The `starters` block (count, per-position limits, `idp_starters`) is in
     `internal/ingestion/league/types.go`; `rulebook` diff already reads `starters.count`.
   - **Depth beyond starters** (bye and injury cover), and **the roster's age profile** (Dyn
     already discounts age, so check for double counting).
   - **Draft picks owned:** not loaded yet (MFL `futureDraftPicks` export; would need ingestion).
   - **Strength of schedule:** `internal/ingestion/leagueschedule` exists.
   Test each the way the scouting work did where possible: does it predict the rest-of-season
   (or next-season) results on the 2021–2025 league history better than the current board?
   MFL weekly scores 2021–2025 are in the fitting DB (`~/scratch/twr-research/fit.db`).
3. Each change follows the usual path: session branch, live gate on Claude-OS against a
   backup-API snapshot of Christopher's real DB, PR, merge only on his "merge N".

## 1. State
- **main = `e922753`** (v0.5.0-192). Today's merges: #7–#11 (core stages 4–8, horizon, README,
  phase, identity guard) and **#12, scouting robustness** (Core Build Plan R9-1…R9-6 with the
  gate record):
  - Defenders are fitted and valued at MFL's position (`modelrun.AtLeaguePositions`;
    `cmd/fit -players <MFL players export>`).
  - DT reads each college defensive share; DE adds his best college season and the number of
    seasons.
  - The college team's QB hurries are stored; a stored CFBD season is re-read from its archive
    when it lacks a mapped measure.
  - NFL contracts (OTC via nflverse, Parquet, `parquet-go`) feed defensive survival; kept at DT,
    DE, LB and CB, dropped at S; loaded every 30 days.
- `build/bin/thewarroom` on the Beelink = **v0.5.0-192-ge922753** (main). **Caution:**
  `make build` writes this path, the binary Christopher launches. Before building an unmerged
  branch, copy the result elsewhere and restore main's binary (done once on 10-04; main's
  binary is reproducible from main).
- Task list: **T378** (Hermes, PARKED) is rewritten to "scouting audit done and merged … Next:
  power-ranking factors" (commit 7a83320).

## 2. Environment
- Go: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`. Gates `make lint`
  (bloat baseline comment 13, provenance 0, tiny 22, dupl 0: under 40 lines counts as tiny, so
  fold small files into their homes) and `make verify` (pre-push). Merge = fast-forward main
  (`git push origin origin/<branch>:refs/heads/main`).
- **Claude-OS** (`ssh claudeos`; leave it running): `~/twr-gate/thewarroom-r9b`
  (v0.5.0-191-g5d013c5) is running on a backup-API snapshot of Christopher's real DBs taken
  17:08, holding model runs 3–4 (run #2 is his eb4cbb4 run). Older copies:
  `~/.config/TheWarRoom.snap-1422`, `~/.config/TheWarRoom.gate-2026-10-04`. Launch:
  `cd ~/twr-gate && . ./cfbd.env && DISPLAY=:0 setsid -f ./<binary> > <log> 2>&1`. Kill the app
  before scp'ing a binary. Unmaximized: Assets (172,302), Pulse (172,360), Control (172,533),
  Score League (395,126), The franchise chip (547,241), Control tabs at y=123 (Signals 733).
- **Snapshots on the Beelink:** `~/scratch/twr-snap-r9/` (17:08, his real DBs). Fitting DB:
  `~/scratch/twr-research/fit.db`: the stage-4 history plus the team hurries and contracts, with
  weekly MFL points for 2021–2025. Christopher's real DB lacks weekly points, so fit only from
  fit.db. Refit:
  `go run ./cmd/fit -db ~/scratch/twr-research/fit.db -players ~/scratch/twr-research/players.json`
  (players.json = MFL players export, sha256 596cd73c…; deterministic).
- Research ledger: `~/fleet/runs/warroom-scouting-research-2026-10-04/RESEARCH.md`; harnesses
  and derived tables in `harness/`; gate evidence in `live-gate-r9/`. Harnesses run by copying
  into `internal/model/fit` with `-tags scratch` (never commit them).
- Real DB: `~/.config/TheWarRoom/{thewarroom,history,whatif}.db`. Read only with
  `sqlite3 -readonly`. Never run the production binary on the Beelink.

## 3. Refuted / settled — do not retest
- Scouting, all on the 2023–2025 rolling holdouts (positions as MFL lists them):
  - **Not adopted:** PFR pressures, hurries and missed tackles (pressures predict next-season
    sacks better than sacks, r 0.59 vs 0.55, but add nothing once the fantasy percentile is
    known); participation pass and run snap shares, including for part-timers; stat-crew tackle
    adjustment (venue factor year-to-year r 0.25); CFBD recruiting ratings; the SackSEER
    explosion index on top of the shipped DE inputs (loses 2025); separate college shares at
    LB, CB and S (mixed by season).
  - **The DE "Now" dips for hot starts are the fit's own rules:** recency tied Marcel
    (0.210 vs 0.209) so the simpler one is kept, and k_now rose 3.7 → 5.6 with the better DE
    prior. This is not a bug.
- Settled earlier: birth date is not an identity test; athleticism × age was rejected (R6-9);
  a model edit writes no board run (R8-6); an override equal to the default still pins (R8-7).
- Christopher's rulings: horizon 3 seasons at 0.75; FantasyPros rankings are a benchmark only,
  never an input; a source joins `Approved_Sources.md` only after it passes (contracts did,
  v1.3); README stays fun and emoji-friendly, with true facts.

## 4. Open, not started
- DynastyProcess issue for 17767's bad ids: draft only on Christopher's OK.
- Commissioner League Controls shows OFFSEASON mid-season (deferred suite; observed only).
- Transact roster pane narrower than its table (periphery).
- Model limits: kickers; a rookie who sits keeps his draft-day debut chance; the top-tier
  redraft benchmark still trails the experts at DE, LB and CB.
- The contracts window (30 days) means an offseason signing can take up to a month to count.
  Revisit if Christopher wants it faster (costs archive size: 11 MB a load).
