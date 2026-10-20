# Feature extraction: moving a feature out of `package main`

The recipe the Archive extraction followed (reorg PR 3). Wails keeps binding one `App`; a
feature's root files shrink to adapters and its logic moves to `internal/app/<feature>/`.

## Steps, one commit each

1. **Freeze the wire first.** Add a root test that JSON-encodes the feature's binding results for
   fixture inputs and compares them byte for byte to `testdata/wire/<feature>/*.json`
   (`TWR_WRITE_WIRE=1` writes them). Call the bound method itself at least once. Commit it passing
   before anything moves; it must pass unchanged after every later step. Keep clocks and the
   time zone out of the goldens: call bound methods on a fresh app and build held states in UTC.
   A new root test file counts against the root ratchet until the move deletes emptied files.
2. **Shared types.** Reports are built as `internal/app/report` values (`report.Board`,
   `report.Report`, `report.Row`, `report.Choice`, `report.Cells`, `report.ScoreText`). Add what
   the feature needs there; never change a field name or JSON tag. A root helper the service
   needs (the report shell over a held feed, a shared wording) moves there too and the root copy
   becomes a one-line delegate (`reportShell` → `report.Shell`, which takes the feed's read state
   as `report.Feed`), so nothing is duplicated.
3. **Move the construction.** `git mv` the reading files into `internal/app/<feature>/` so review
   sees renames, then change only the package, the type names and the exported entry point
   (`archive.Build`). Implementation tests move in the same commit, because they call unexported
   helpers that no longer exist at root.
4. **Move the read.** Data access becomes a function that takes its dependencies explicitly: a
   small interface over the store (`archive.History`), a `Source` struct of names and readers,
   and the value the caller already holds. No globals, no goroutines, no opening databases. What
   only another feature's root code knows comes in as a function value (`draft.Values` takes the
   measure meanings `describeMeasures` gives).
5. **Map and ratchet.** Update `placement.tsv` (each new package gets a row), the `SYSTEM_MAP.md`
   row and the narrative paragraph, and lower `root-go-baseline.txt`.

## What stays at root

- The bound method, with its name and signature: `ready()`, the timeout, the locks, then the
  service call. Each adapter is a few lines.
- The **wire types** (`ArchiveBoard`, `Report`, `ReportRow`, …). Wails names a TypeScript model
  after the package that declares the Go type, by reflection, and a type alias does not change
  that. Moving a wire type, or aliasing it, renames `main.X` in `frontend/wailsjs/go/models.ts`.
  So the service returns its own types and the adapter converts them (`wireBoard`,
  `wireChoices` in `report_reading.go`; a per-feature converter such as `buildArchive`).
- **Held state and its lock.** The service is stateless: the held value stays on `App` (the
  archive snapshot, under `factsMu`). The caller passes it in and stores what comes back.
- Anything that reads another feature's held state (`archiveOpponent` reads the season feeds
  under `seasonMu`). It moves when that state has one owner, not before.
- Code built on another feature's wire types that have no shared contract yet (the Draft Lab
  returns `Reading[MarketSurface]` through Market's helpers). It moves after that feature's
  extraction adds the contract; twins written early would be copies the later move must undo.

## Tests

- Root keeps the wire golden and `TestReportWireTwinsMatch`, which fills every field of a
  `report.Board` and checks the converted wire value encodes identically. A field the converter
  drops, or a tag that drifts between the twins, fails it.
- Unit tests live beside the code. A fixture both packages need goes in a
  `<feature>test` package (`archivetest.Facts`, `drafttest.Held`), not copied: a copied loader
  trips the `dupl` count in `make lint`'s bloat ratchet. The fixture package must not import the
  service, or the service's own tests form an import cycle: return plain values (`drafttest.Draft`)
  and let each side wrap them in its own types.
- Check each new test with a planted mutant.

## Pitfalls hit

- **Import names.** `app.go` already imports `internal/archive` (the HTTP response recorder), so
  root imports the service as `archiveapp`. Use `<feature>app` for every service import at root.
- **Exported fields.** The held value's fields become exported (`Snapshot.SHA`), so every root
  reader of the old private fields changes in the same commit (`comms_app.go`, `asset_board.go`).
- **Lint at a new home.** Returning a service error unchanged needs
  `//nolint:wrapcheck // the service's errors already start "<feature>:"`, so the text the
  frontend shows is unchanged. Test helpers may not use package-level `var`s
  (`gochecknoglobals`), and `exhaustive` rejects a `switch` over `reflect.Kind` even with a
  `default`.
- **Parallel lanes.** The pre-commit hook runs golangci-lint, which refuses to start while
  another lane's lint holds its lock ("parallel golangci-lint is running"). A private
  `GOLANGCI_LINT_CACHE` alone did not clear it; a private `TMPDIR` as well did.
- **Tiny files.** Each new non-test file needs at least 40 lines of code, or it counts against
  the bloat ratchet's `tiny_files` (baseline 22). Put the package doc in a real file.
- **Store the held value as before.** `archive.Read` returns the zero `Snapshot` when nothing is
  verified; the adapter only replaces the held copy when the result is a real snapshot, as the
  old code did.
