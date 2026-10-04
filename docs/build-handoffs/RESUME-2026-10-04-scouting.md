# RESUME — TheWarRoom, after the core (2026-10-04 afternoon)

## 0. Next actions, in order
Nothing running that needs you. Core Stages 4–8 are merged and Christopher is using the app.
The next focus (Christopher, 2026-10-04): **(a) make sure the scouting profile and its weights are
as robust as we think; (b) the additional things that should affect the power rankings.**
1. Branch: `session/scouting-robustness` (holds only this file). Read
   `docs/build-handoffs/Core_Build_Plan_2026-10.md` Stages 5–8 and R6-*/R7-*/R8-* first.
2. **Scouting audit — establish from evidence, then design.** There are two scouting systems:
   - **Today's board, L4** (`internal/engine/l4/settings.go`, `rubric.go`): film × RAS × breakout,
     three capped S-curves; the breakout composite weights BreakoutAge, SchoolTier, CollegeShare
     and **AgeTrajectory** per position (hand-set curves). Caps/weights are Admin params
     (`l4.*@POS`, R7) but **hand-set, never fitted**. Film is coverage-only at CB/S (PFR advanced
     defense); Madden is out (R8). L3 age decay + L4 AgeTrajectory = age counted twice on the
     board (R7-6 left this until the board is retired).
   - **The model's prior** (`internal/model`, fitted by `cmd/fit`): ridge on log pick, entry age,
     8 combine tests, last college season production, with missing indicators. Prior R² holdout
     0.21–0.43 by position, K ≈ 0, DT 40% train vs 3% 2025 (`docs/fit/Fit_Report.md`).
   - Questions to answer with data (holdout 2025 from ≤2024, as in Stage 6/7): does each L4
     component predict next-season league percentile on its own and beyond the fitted prior? Are
     the hand-set caps/weights near what a fit would choose? Do correlated signals double count
     (plan Stage 6 note: L4 multiplies film × RAS × breakout as if independent)? Coverage of each
     signal on the roster (Signals console). Then a recommendation matrix for Christopher:
     fit L4's weights / fold L4 into the prior and retire it / keep as is.
3. **Power rankings — what else should move them.** Today: roster sum (or top-N starters) of Now
   or Dyn, z-blended (median/MAD) with results (all-play when MFL reports it, else points for ÷
   best), 60/40 default; the franchise view is roster alone. Candidates to evaluate (product calls
   are Christopher's): cap health (cap space, dead cap, contracts' years left — the ledger has it),
   positional scarcity / starting-lineup slot fit instead of a flat sum, depth beyond starters,
   age profile of the roster (Dyn partly covers it), draft picks owned (not in history yet),
   strength of schedule. Present as a matrix with a recommendation.
4. Each change: session branch, live gate on Claude-OS against a backup-API snapshot of
   Christopher's real DB, PR, merge only on his "merge N".

## 1. State
- **main = `eb4cbb4`** (v0.5.0-184). Merged today: PR #7 (Stages 4–8), #8 (dynasty horizon
  3 seasons / 0.75, Christopher's call; Admin shows pinned overrides + Reset, R8-7), #9 (README:
  voice/emoji/vision kept + core facts + live screenshots; ChatGPT is the second opinion), #10
  (M2 phase from MFL standings, not the model run), #11 (identity guard: a history record whose
  rookie season is >2 from MFL's draft year is another player's — set aside, valued on MFL's
  facts; DynastyProcess gave MFL 17767 James Thompson Jr. a 1953 WR's gsis/PFR ids).
- `build/bin/thewarroom` on the Beelink = v0.5.0-184-geb4cbb4. Christopher launches it with
  `~/work/TheWarRoom/build/bin/thewarroom &`. His real DB: signals loaded (2026-10-04 13:04),
  model run #1 at ec985a4 (17767 at 0); he relaunches + Score League to get the fixed run.
- CFBD key on the Beelink: `~/.config/cfbd/api_key` (600), exported from `~/.profile` and
  `~/.bashrc`. Merge method: fast-forward main to the PR head (`git push origin
  origin/<branch>:refs/heads/main`), same as all stage PRs; there is no CI, `make verify` is the
  pre-push gate.

## 2. Environment
- Go: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`. Gates `make lint`,
  `make verify`. Bloat baseline comment 13, provenance 0, tiny 22, dupl 0.
- **Claude-OS** (`ssh claudeos`, leave it running): `~/.config/TheWarRoom` now holds a snapshot
  of **Christopher's real DBs** (taken 14:22 with `sqlite3 .backup`), with model run #2 at
  eb4cbb4; the old gate DB is kept at `~/.config/TheWarRoom.gate-2026-10-04`. App
  `~/twr-gate/thewarroom-s8` (v0.5.0-184) is running. Launch:
  `cd ~/twr-gate && . ./cfbd.env && DISPLAY=:0 setsid -f ./thewarroom-s8 > s8gate.out 2>&1`.
  Kill the app before scp'ing a new binary (text file busy). Copy with `scp -l 40000`.
  Unmaximized: Assets (172,302) Pulse (172,360) Control (172,533); Engine Admin tab (513,123),
  Admin filter (497,175); Score League (395,126); Pulse "The franchise" chip (547,241).
- Snapshot of the real DBs also at `~/scratch/twr-snap-2026-10-04/` (321 MB) for analysis.
- Real DB on the Beelink: `~/.config/TheWarRoom/{thewarroom,history,whatif}.db` — read only with
  `sqlite3 -readonly`; never run the production binary on the Beelink yourself.
- MFL players response is archived in history `raw_archive` (gzip; fetch_log url
  `TYPE=players&DETAILS=1`): MFL birth dates, draft_year/round/pick.
- Evidence: `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-{stage4..8,horizon,identity}-2026-10-04/`.

## 3. Refuted / settled — do not retest
- Birth date is not an identity test: MFL's own birth dates are sometimes placeholders
  (Josephs "2026-09-08", several "2005-01-01"); rookie season vs MFL draft year separates cleanly
  (1,429 equal, 10 within one, the mislink 48 away).
- Athleticism × age interaction: tested and rejected (R6-9).
- A model edit must not write a board run (R8-6, fixed); an Admin override equal to the default
  still pins (R8-7, Reset added).
- Christopher's rulings: horizon 3 / 0.75; CFBD key set; all-play — he turns it on in MFL and it
  moves only This season; README voice stays fun/emoji/vision-forward, facts true.

## 4. Open, not started
- DynastyProcess upstream issue for the 17767 bad ids: draft only on Christopher's OK.
- Commissioner League Controls shows OFFSEASON mid-season (deferred suite; observed only).
- Transact roster pane narrower than its table (periphery).
- Known model limits: in-season k_now overweights the current season; a rookie who sits keeps
  his draft-day debut chance; kickers.
