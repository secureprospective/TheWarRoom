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
