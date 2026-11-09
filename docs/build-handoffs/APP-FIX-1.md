# APP-FIX-1

Base: `930002e6`, branch `build/FIX-1`. Local only; no desktop launch or remote operation.

## Migration finding and correction

The failing audit test fabricates v4 from a fully initialized current database. It
removed v5–v11 objects, reset `user_version=4`, but retained all three v13 tables.
The replay therefore encountered an existing `live_evidence` table. This was not
DDL duplicated in adoption or a production migration running twice.

Traced flow: `Store.Initialize` → `Pools.Migrate` → `migrateStep` →
`addLiveEvidence`; each step's DDL and version marker commit in one transaction.
`adoptHistory` contains no live DDL. Completed versions are skipped on reopen.

Fix: remove the live board, chronology and evidence in dependency order when the
test reconstructs v4. Keep the production migration strict: `IF NOT EXISTS`
would hide an inconsistent schema, not fix this fixture.

Added `TestHistoryLiveMigrationRetriesAtomicallyAndReopens`: adopt an unversioned
legacy database through v12, fail after live DDL, observe v12 with no live objects,
retry to v13, insert evidence/chronology/board rows, close/reopen and initialize
twice, verify retained rows and append-only update/delete guards.
All five brief-listed failures were reproduced on the base before changes.
Production source footprint unchanged; test source +63 physical lines.

## Real-history copy proof

Copied only `history.db` and `history.db-wal` into the permitted scratch area;
source file sizes/mtimes stayed stable during copying. SQLite was opened only on
the copies. Initial copy: v12, no live objects, integrity `ok`.

A disposable Go harness called the real `db.Open`, `history.New`, `Initialize`,
`RestoreLiveStats` and `VerifyAudit`. First open: v13, three live tables,
pre-v12 migration backup created, audit size 16,776. Second open: v13, three live
tables, same audit size, no additional migration backup.

Afterward: integrity `ok`, zero foreign-key violations. Counts unchanged:
2,789 archive bodies; 1,193,235 observations; 16,776 league events and audit rows.
Deleted the copied database, WAL and generated backup, and removed the harness.
No real config/data was changed.

## Golden review

Recursive JSON comparison found no removals, changed existing values, reordered
arrays or changed array lengths. Only the following intended additions:

| Test | Exact fixture delta |
| --- | --- |
| `TestNativeResultsStandingsWire` | `native-results-standings-wire.json`: 64 starter `fieldSources` maps (`tackle: ""` from fixture evidence), +192 lines. |
| `TestFrozenDoorMetadata` | `doors-before.json`: only three hashes, for stat correct/approve/reject. Removing the new optional `items` argument from each current schema reproduces its old hash exactly. This is the existing batch-stat addition from `c9655954`; no permissions/modes/roles changed. |
| `TestNativeLeagueWireMatchesGolden` | Only `testdata/wire/native/results.json`: 32 starters gain `source: "final · nflverse"` and `fieldSources`, +1,568 lines. All other goldens exercised by this test unchanged. |
| `TestNativeScoreWireFixtureMatchesGo` | `player-score-wire.json`: five readings gain the same `source` and `fieldSources`, +245 lines. Two unavailable readings unchanged. |

Source labels/maps are the existing provenance feature from `138803c7`.
No score, standings, receipt or other behavior change was accepted into a golden.

## Verification

- `go test -list . ./internal/store/history ./internal/composition` ran first.
- All five requested tests plus the new migration test: PASS under `-race -p 2`.
- `internal/store/history`: PASS, 318 top-level tests +197 subtests; four skips.
- `internal/composition`: PASS, 604 top-level tests +243 subtests; zero skips,
  908.6 seconds. The first combined run used Go's default ten-minute timeout;
  after history completed, composition was explicitly stopped by PID and restarted
  with `-race -p 2 -timeout 60m` rather than risk that timeout interrupting the
  longer suite. Totals above count only completed runs, not the interrupted run.
- `make build VERSION=dev COMMIT=fix-1 BUILD_DATE=fix-1`: PASS; binary built,
  deliberately not launched. Discarded only build-generated Wails runtime modes.
- `golangci-lint run ./internal/store/history/...`: PASS, zero issues.
- `git diff --check`: PASS. Normal pre-commit gates required on the local commit.

## Debt / review limits

No production migration defect found; no production migration change warranted.
Manually lowered version markers with newer objects still present remain invalid
input, not a supported downgrade. GUI verification was deliberately not attempted.
