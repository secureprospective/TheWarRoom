# Lab out of TheWarRoom: cut map (2026-11-04, read-only scope, nothing moved)

Ruling (SOAK-RULINGS batch 1): lab, labcompile and facts move to their own multi-league repo (~/work/LeagueLab). Not deleted.
Size: internal/lab + internal/facts + internal/labcompile = 15,568 lines of Go (tests included).

## Who still imports it (non-test files)
| Importer | Files | What it uses | Already ruled? |
|---|---:|---|---|
| app/sources | 6 | lab.Data/Curated/NoteReadings/FrozenGaps/CheckExceptions, facts.CheckResult/ResolvedInput | Research split (10-28): leaves the app |
| app/archive | 2 | facts.Compile/ParseCurated/ResolvedInput, lab.CheckExceptions/FrozenGaps | Research split: leaves |
| app/report | 2 | lab.Interval/DataSHA, facts.Facts | Research split: leaves |
| app/comms | 3 | facts.RosterWeekColumns, NflGameColumns, Team, Position | **Ruled 11-04: removed** (not in the mocks, never built). Done in c7013152 |
| app/core | 4 | lab.Load/Data (LabData, LabFacts hooks), facts column types via host_comms | Wiring only; goes when its callers go |
| composition | 5 | lab.Load (app.go:102 LabData), lab_app.go, sources_runtime.go | Wiring only |
| labcompile | all | facts (14 files) | Moves with Lab |
| tools/labparity | - | Lab parity goldens (Makefile line 177) | Moves with Lab |

## Proposed order (each step builds and shrinks the tree)
1. ~~Comms~~ done: removed whole (c7013152). 68 files, -2,904 lines; the hourly refresh no longer fetches MFL messageBoard/polls.
2. Remove app/sources, app/archive and app/report from the app, with their core and composition wiring (Research split). Data files kept.
3. Copy lab, labcompile, facts and tools/labparity into ~/work/LeagueLab with history (git subtree split / filter-repo), then delete them from TWR.
4. Prove: `go list -deps` shows no lab/facts import; binary size and source line counts recorded before and after.

