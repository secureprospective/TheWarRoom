# TheWarRoom debloat — resume point (written 2026-11-09 by Tom on the Beelink)

Read first: `~/Downloads/TheWarRoom-DEBLOAT-HANDOFF.md`, then `bone-list.md` here (all three revisions
bind), then `semantic-pot.md`.

## Rulings from Christopher this session (override the handoff where they differ)

1. Keep every engine (fitted model, model runs, values, power rankings, measures, params, signal
   pipelines, `internal/engine/*`). Do not break them.
2. Project moved from MFL mirror to MFL→Native import. The mirror (`mflref`, `transactions`, what-if
   desk, Now/Market/Race) stays ONLY as the solo-owner framework, kept simple: trim, never delete.
3. The MFL→Native import must be flawless for any league size or type. Its tests are in the keep set.
4. PWA-centric: the React app served by `twr-server` is the main interface. Desktop stays first-class.
   Where desktop-only and hosted paths duplicate a job, protect the hosted one.
5. AI/CLI-first: an agent does everything through the CLI; league shape and rules are data; screens and
   minigames are outputs. Measure the bone by what `cmd/twr` links (86k production Go lines on 11-09).
6. The 32-franchise cap stays for now (scope).
7. The end-to-end read of the import and `internal/commissioner` is deferred to its own session
   ("it's a mess"). Do not reshape core ledger code before that read and a checkpoint with him.
8. Wording drift goes in `semantic-pot.md`, not fixed inline.

## State

- Worktree `~/work/TheWarRoom/.worktrees/debloat`, branch `session/debloat-1109`, off `main` 3a2843de4
  (PR #220 merged). Not pushed. No PR.
- Commits: 2d3b2ecb6 (batch 1), 1970f5703 (batch 2). Production code untouched except: one
  `//nolint:gosec` in `internal/composition/app.go`, and three test-diagnostic functions removed from
  `store/history/season_score_cache.go` and `reading_flight.go`.

| Area (k lines) | Before | Now |
|---|---|---|
| Go production | 141.6 | 139.9 |
| Go tests | 136.6 | 94.1 |
| Go test data | 133.4 | 61.2 |
| Frontend source | 72.8 | 72.5 |
| Frontend tests | 22.0 | 10.8 |
| Frontend JSON | 101.3 | 23.8 |
| Docs (`.md` + `docs/`) | 101.6 | 50.8 |
| All tracked | 725.4 | 467.7 |

Measure with `tools/twr-measure.sh` from the worktree root.

## Checked so far

- `go build ./...`, `go vet ./...`, `golangci-lint run ./...` (0 issues) at 1970f5703.
- Tests run: composition, app/..., store/history, serverhttp, commissioner/... ; frontend tsc and
  vitest (98 files, 756 tests) after batch 1.
- Full `go test -count=1 ./...` (no race) passed at 1970f5703, exit 0.
- NOT yet run: `make verify`, the race suite, `make build`, any lab parity or speed proof, Claude-OS.

## Tools (source in `tools/`, build with `go build`)

- `twr-extract` + `loop.sh`: after deleting test files, pulls only the helpers the survivors need into
  `helpers_test.go`. `loop.sh <worktree> <pkgdir> <pkgname> <undefined names...>`.
- `twr-prune <regexp> files...`: deletes test functions whose source matches.
- `twr-unused`: pipe golangci-lint output in; deletes declarations reported unused.

## Next, in order

1. Remaining Go tests (94k → ~40k): `internal/app/*` (research, now, mflref, control, commissioner
   release matrices), `ingestion`, `mflref`, `serverhttp`, `leagueservice`. Same method as batch 2.
2. `internal/commissioner/testdata` (29k): the `*wire*.json` files double as inputs to backup, transfer
   and recovery tests. Cut only files no surviving test opens.
3. Research out of the app (ruling 10-28): `internal/app/research`, `researchfeeds`, `injuries`,
   `frontend/src/app/research`, their composition adapters.
4. Frontend: old endpoint registry (`app/registry`, `docs/ui/endpoint-registry.csv` and its Go test
   readers), specimen, and why shipping code imports 23.8k lines of `data/fixtures` JSON (`data/provider.ts`).
5. Dead code and stray: bloat ledger §4 (~600 lines), `store/playeridentity`, `seams`, `teamneeds`,
   migration assistant, fixtures compiled into release binaries (gate with a build tag).
6. Gates: comment share 6→5, tiny files 41→24, native forks 19→one seam, boundary inventory review.
   Lower each baseline in the commit that earns it.
7. Docs: rewrite README and SYSTEM_MAP to match the tree (README has dead links to deleted docs);
   `docs/ui/data-catalog`, `docs/ui/showcase` review.
8. Step 5 of the handoff: `make verify`, `make build`, lab parity and speed (keep `deploy/lab` until
   then), PWA at phone width, desktop on Claude-OS, mirror click-through, then PR.

## Open with Christopher

- Mirror check: he offered to click through a solo-owner screen; ask him for one specific screen when
  the mirror trim happens.
- Import coverage by league size and type: unknown until the deferred read.

## In flight

Nothing. No background jobs, dispatches, VMs or lab CTs were started by this session.

## Environment and traps (learned this session; do not rediscover)

- Go env for every command: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$HOME/vendor/gh/bin:$PATH
  GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false"
  GOLANGCI_LINT_CACHE=$PWD/.golangci-cache`. Frontend deps are installed in the debloat worktree.
- The main checkout `~/work/TheWarRoom` sits on `session/league-history-pwa` (the Legacy NFL Lab). Never touch it.
- The commit hook runs golangci-lint over everything. Deleting tests leaves unused helpers and unused
  production test-diagnostics; clear them with `twr-unused` in a loop before committing.
- gosec G118 fired on `composition/app.go` only because a deleted test was the one direct caller of
  `WorkCancel()`; production cancels it through `workerGroup().Stop`. Already annotated.
- Some tests test the tests (glob for other `_test.go` files, `rootTestSource`). Delete them on sight.
- Frontend "test kits" are not named `.test.`: `commissionerStub.ts`, `roomTestKit.tsx`,
  `draftHarness.ts` (all deleted). Check non-test importers of fixtures before assuming code is shipped.
- Frontend reachability: the native mount reaches ~94% of the frontend because native screens wrap the
  shared HQ/War Room components with `native ?` forks. The frontend does not split by import graph.
- `twr-server` links none of `internal/app`, `internal/composition`, `internal/mflref`,
  `internal/transactions` (61 packages vs the desktop's 118).
- `internal/commissioner/testdata/*wire*.json` are read by offsite, leaguetransfer, leaguerecovery and
  `cmd/twr-server` stand-up tests as inputs. Not bulk-deletable.
- Lab host reachable from the Beelink: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@192.168.1.182`;
  parity scripts are in `/root/stress256`. Never touch CT 110.
- Not checked and not ruled out: whether any CI job or `make verify` step reads a deleted doc, mock or
  fixture. The first `make verify` on this branch is the test of that.

## Honest status

About 35% of tracked lines are gone, almost all of it tests, test data and docs. Production Go is down
1.2%. The handoff's definition of done (50% of code, gates at target, lab proof, PR) is not met. The
structural production cut depends on the deferred import and commissioner read.

## Update 2026-11-09 evening — batch 3 (commit 2f32cc00a)

- Method changed for Go tests: per-test coverage, repo-wide (`tools/twr-cover/collect.py`, ~20 min;
  data in `cover/tests.pkl`), then `select.py` (keep set pinned by package and name; any other test
  survives only if it executes a statement no kept test executes). Result list: `cover/cut.tsv`.
- Cut 295 of 2,301 tests; covered statements 45,826 before and after. Go tests 94.1k → 85.2k.
  Build, vet, lint clean; full `go test -count=1 ./...` passed. Commits now: 2d3b2ecb6, 1970f5703,
  69c3fdce6 (docs), 2f32cc00a.
- FINDING: the ~40k Go-test target is not reachable without losing coverage. Even with the ledger
  packages unpinned (name pins only) the rule frees ~18k lines (`cover/cut-nameonly.tsv`), i.e. ~75k.
  Further test reduction must come from deleting production code (its tests go with it), not quota.
- Frontend fixture finding: `frontend/src/app/data/fixtures/snapshot.json` (23.6k lines) is example
  data for `FixtureProvider` (`data/provider.ts`), chosen when the app runs in a browser with no
  Wails (`selectProvider`). ~30 files reference it, mostly tests. Not yet changed.
- Next action 1 is DONE. Resume at 2 (commissioner testdata), then 3 (research out).

## Update 2026-11-09 night — after batches 4–6 (head f66e816ee)

Christopher agreed: stop cutting Go tests by quota; go through the remaining items in order.

- Item 2 DONE (e72c82566): test data. Ran the suite under strace (`go test -exec "strace -f ..."`), deleted
  the 51 data files nothing opens/embeds/names (~21k lines). Kept SOURCE.md notes and generator scripts.
- Item 3 NOT A CUT: the research *screen* is already gone. What is still named "research" is the player
  feed behind Home tiles, HQ and War Room inspectors, roster facts and offers. See semantic pot 14.
- Item 5 DONE (c09449301): `deadcode` over all mains. Removed unreachable funcs + `seams`,
  `store/playeridentity`, `teamneeds`. KEPT on purpose: `.sealed` interface markers, secret-redacting
  String/GoString/MarshalJSON in `accounts`, engine/ledger/leaguerules accessors, test support
  (`hostedfixture`, `NewFixtureChanges`, `NewMemory`, `ReadCandidate`), `composition.runFlag` (restore
  tests), crosswalk/leaguearchive/app-scoring accessors, and `leaguemigration/assistant.go` (handoff says
  cut, but it is the one-ask AI import path; left for the deferred import read).
- Item 4 PARTLY: specimen page deleted (814a9f8f4). NOT cut, both are live and need rebuild not deletion:
  (a) `frontend/src/app/registry` (6.6k generated) is used by the solo shell, command search, reports and
  commissioner controls; `docs/ui/endpoint-registry.csv` is embedded in the desktop binary, read by
  `twr-server --preview-registry`, and is in the offsite backup set. (b) `FixtureProvider` +
  `snapshot.json` (23.6k): example data when no Wails; ~20 solo-owner tests use it as their data.
- Gates (f4b8ec71d): placement and boundary inventories fixed; SYSTEM_MAP stale paths removed.
  bloat.sh still at baseline (comment 6, tiny files 41); native forks 19. Not lowered yet.
- MISTAKE CAUGHT: batch 1 deleted six handoff docs that project CLAUDE.md / AGENTS.md / README still
  point to (Hardening_Plan = "the current plan"). Restored in f66e816ee; semantic pot 15.
- Numbers (k lines): all 437.5 (from 725.4, -40%); Go prod 139.4; Go tests 84.5; testdata 40.0;
  frontend 72.3; frontend tests 10.8; docs 52.2.
- `make verify` first full run launched on f66e816ee; log `verify-f66e816ee.log` in this dir.

Next: read the verify log and fix what it finds; then item 6 (comment share 6→5, tiny files 41→24,
native forks → one seam; these are edits, not deletions), item 7 README rewrite, item 8 (make build, lab
parity/speed, PWA phone width, Claude-OS desktop, mirror click-through with Christopher), then PR.

## Update 2026-11-09 late — head a16b27114

- `make verify` PASSED (exit 0) on f66e816ee: log `verify-f66e816ee.log`. Rerun once before the PR
  (tree has changed since: restored tests, docs).
- SECOND MISTAKE CAUGHT (8a9d52671): the coverage rule cut tests whose value is not coverage: repo
  gates (`internal/seams` future-seams gate from the 11-08 binding direction, binding manifest, tzdata,
  no-MFL-submit source scan), opt-in `TestLive_*`, race-detector tests, never/only/refuses invariants.
  80 restored into `guards_test.go` files; `internal/seams` and the per-feature `Commands()` lists are
  back. LESSON: zero-coverage or invariant-named tests are never "duplicates". `select.py` needs that pin
  before any reuse.
- Also restored: `docs/audit/CLIENT-SERVER-RULINGS-2026-11-05.md` and `...TOM-SECOND-OPINION...` (the
  governing plan cites them).
- Docs realigned (a16b27114): North_Star 2.1, CLAUDE.md 3.3, AGENTS.md, README rewritten (import first,
  AI/CLI first, PWA, measured speed, the trim). NOT reviewed yet: `docs/roadmap/Vision_2026.md`,
  `Roadmap_and_Open_Questions.md`, `docs/architecture/system-narrative.md`, `docs/Owner_Install_Guide.md`.
- Note from `docs/RESUME-2026-11-08.md`: Christopher's 11-09 test rule was "a test that is done is
  deleted, target ~3/4 cut, no proving coverage first". This session he agreed to stop quota cuts.
- Next: rerun `make verify`; ratchets (comments, tiny files, native forks); `make build`; lab parity and
  speed; PWA phone width; Claude-OS desktop; mirror screen with Christopher; push; PR.
