# RESUME — TheWarRoom Stage 1 built and gated (2026-10-03)

Stage 1 (the measure store) is built, committed and live-gated on
`session/stage1-measure-store`. It is not merged and not pushed. The full record is in the plan
under Stage 1: the design table S1–S11, the gate table and the live gate. This file replaces
`RESUME-2026-10-03-stage1.md`.

## 1. What we are doing
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. Stage 1 is done; Stage 2 (league
  truth: the MFL refresh) is next.
- **Repo:** `~/work/TheWarRoom` on the Beelink, branch `session/stage1-measure-store`, cut from
  `main` at `98b915f`. No upstream yet: `git push -u origin session/stage1-measure-store`.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify` (the pre-push hook). The bloat baseline was ratcheted
  down to comment 19, provenance 7, tiny files 27, dupl 38.

## 2. Commits on the branch (oldest first)
| Commit | What |
|---|---|
| `bd7c8bf` | Stage 1 resume doc; README rewrite deferred |
| `8e0e5c9` | Stage 1: `internal/measures`, `internal/archive` and `internal/store/history` (history.db), params snapshots, rankings writing runs, the App rewired, `internal/output` deleted |
| `7f46782` | Fix: the RAS cohort was summed in map order, so scores drifted between runs |
| `a01498e` | Unknown command-line arguments exit 2 instead of opening the window; `-version` added |
| (next) | This resume and the plan's gate record |

The commits could not be split finer: pre-commit sees untracked files, so a partial commit fails
on packages that are not yet staged.

## 3. Gate
Every item passes; the plan has the table. Live on Claude-OS with `v0.5.0-126-g7f46782`:
1. board #1 written;
2. a second pass reported "No change";
3. `layer3.decay_rate` set to 0.04 wrote board #2, and both boards read back;
4. with MFL blocked, the board still ran and warned.

Evidence is in `~/fleet/runs/warroom-dataflow-2026-10-03/live-gate-stage1-2026-10-03/`.
`attempt1/` and the `A*` screenshots record the failed first attempt that caught the RAS drift.

## 4. Incident: the production app ran on the Beelink's live desktop
- **What happened:** at 17:39:46 Claude ran `build/bin/thewarroom -version` to read the build
  stamp. `main` ignored unknown arguments, so the production window opened on Christopher's
  desktop against the real `~/.config/TheWarRoom/`. It ran until about 17:46. A ScoreLeague ran
  in it at 17:44–17:45.
- **Effects on the real data:**
  - `thewarroom.db` migrated v2 → v3. This is the planned migration, and its backup is
    `thewarroom.db.premigration-20261003T223946Z`.
  - A real `history.db` (27 MB) now exists. It holds one board run from `v0.5.0-125`, a build
    with the RAS drift, plus its archived fetches.
  - That pass spent the day's one MFL players call.
- **Fixed:** `a01498e` (unknown arguments exit 2). CLAUDE.md now forbids running a production
  binary on the Beelink.
- **Waiting on Christopher:** keep `history.db` as it is, or move it aside so the real history
  starts with the merged build. Nothing has been touched since.
- **Side effect on the gate:** the gate's "fresh snapshot" was taken after the accidental
  migration, so this gate did not re-run v2 → v3. Stage 0's gate already proved that migration.

## 5. Next actions, in order
1. Christopher: approve the merge (rebase-merge; tag the branch tip `archive/<branch>` first),
   and decide on the real `history.db` (§4).
2. Push the branch, open the PR, merge, and confirm `main`.
3. Update Hermes task T368 (Stage 1) and add Stage 2's task.
4. Start Stage 2 from the plan.

## 6. Follow-ups logged, not done
- `DiscoverHost` makes a full `league` request on every fetch. One ScoreLeague logged it 7
  times. Cache discovery per client in Stage 2.
- MFL bodies differ on every fetch, so each ScoreLeague archives about 0.4 MB. That's fine at
  this scale; revisit if refreshes become frequent.
- Engine Admin's APPLY shows no confirmation (the value is stored).
- The frontend still has `name || franchiseID` fallbacks; screens read "Franchise 0014" until
  Stage 2 refreshes the rulebook.
- The Wails CLI is v2.12.0 while the module is v2.13.0.
- README rewrite after the last stage.
- Closed: `mfl.Client` now has `WithTransport`, the test seam that was missing.

## 7. Lessons (do not relearn)
- Never run `build/bin/thewarroom` on the Beelink. Read a binary's stamp with `strings` or
  `-version` (it exists now), and only on Claude-OS for production builds.
- Any float accumulated across a Go map's range is non-deterministic in its last bits. A
  scoring run hashes its inputs and scores, so this shows up as a new board.
- pre-commit stashes unstaged edits but not untracked files.
