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
   Compare with the shared `checkGolden(t, feature, name, v)` (`draft_wire_test.go`), not a new
   copy; sample large fixtures (a dozen players per feed) so each golden stays reviewable.
2. **Shared types.** Reports are built as `internal/app/report` values (`report.Board`,
   `report.Report`, `report.Row`, `report.Choice`, `report.Cells`, `report.ScoreText`). Add what
   the feature needs there; never change a field name or JSON tag. A root helper the service
   needs (the report shell over a held feed, a shared wording) moves there too and the root copy
   becomes a one-line delegate (`reportShell` → `report.Shell`, which takes the feed's read state
   as `report.Feed`), so nothing is duplicated.
3. **Move the construction.** `git mv` the reading files into `internal/app/<feature>/` so review
   sees renames, then change only the package, the type names and the exported entry point
   (`archive.Build`). Implementation tests move in the same commit, because they call unexported
   helpers that no longer exist at root. Git's rename detection is by similarity, so a file whose
   identifiers all change (Market's `heldSeasonFeed` fields becoming `market.Held`) shows as a
   delete and an add; say so in the commit message rather than claiming a rename.
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
- **Shared wire envelopes.** `Reading[T]` and its statuses, used by many features, stay at root;
  the service twin is `report.Reading[T]` and `wireReading(r, convert)` maps one to the other. A
  request type such as `RawMarketScope` stays too; its service twin (`market.Scope`) has the
  same fields in the same order, so `market.Scope(raw)` converts it directly, and the root
  `Validate` delegates.
- A non-wire root type other features use (`assetModel`, `dynastyValue`) may become an alias of
  the service type, so callers keep compiling; its interface methods become exported, and every
  implementation (test fakes included) changes in the same commit.
- Code built on another feature's wire types that have no shared contract yet (the Draft Lab
  returns `Reading[MarketSurface]` through Market's helpers). It moves after that feature's
  extraction adds the contract; twins written early would be copies the later move must undo.

## Authorization seams

The Commissioner extraction (reorg PR 11) set the rules for a feature that decides who may act and
writes the league record.

- **Who is calling stays at root.** `commissionerPrincipal` reads the injected session; the
  adapter reads it, in the same place in the check order, before any service call. A service takes
  the principal as a plain value and never reads a session or a global to find the caller.
- **Every check keeps its place.** The scope check, `ready()`, the principal, then the call, in
  the order the binding had. A pure check may move into the service (`AppLeague`, the request
  twins' `Validate`) only if the adapter still calls it at the same point. A check that differs
  between bindings stays different: the people and ledger bindings keep their looser league checks
  inline rather than sharing the strict one.
- **Door policy stays at root.** The commissioner door's refusals that send owner acts to their
  own gated doors (trades, cuts and claims, bids, the starting seed, lineups, rookie picks) are
  safeguards, not wording, and stay in `TargetCommissionerExecute`.
- **Writes stay in one transaction.** A store call that authorizes, validates and appends
  (`ExecuteCommissionerWithFacts`, `PreviewCommissionerWithFacts`, the filtered ledger and
  digests, each with `commissionerAuthorizer`) stays at root as one call. Splitting it so a
  service checks and the store writes would open a gap between the check and the append. Only
  read-only replays (`Records`) and their readings move.
- **Freeze the denials, not only the wire.** Before the move, a golden of the exact words each
  refused caller reads: a stranger, an owner and a co-commissioner on commissioner acts, a wrong
  franchise, a closed or expired window, a stale version, an unreadable session, every bad scope,
  and every routed act (`testdata/wire/commissioner/denials.json`). Check it with a planted
  gate-skip mutant, and race two writes at one version to show exactly one lands.

The native league extraction (reorg PR 12) added the owner side.

- **The gate is the seam.** `native_gate.go` (the registry, `runNativeAction`, `ownerZone`) stays at
  root whole: an owner action's `Run` takes the gate's `Permit`, so the only way to start one is
  through the gate. The owner door (`submitOwnerAction`), each family's `seat` (which franchise a
  command acts for) and the permit-matches-command checks in `recordOwnerAction`, `runLineupSet` and
  `runDraftPick` stay beside it; they decide who acts, not how a value is shaped.
- **The store checks again.** Each owner write re-checks inside its own transaction, with the version
  check and the append: `ExecuteCommissionerWithFacts` asks the gate again, and `RecordDraftPick`
  checks the permit, the franchise's seat and the draft. The adapter's gate call is the early
  refusal, not the only one; neither moves.
- **What moves is shaped data.** The facts mapping, the lineup decode, the bid season and the starting
  snapshot's shape take held values as arguments. The reads of held state (`lookupMu`, `seasonMu`,
  `refreshMu`, `targetSnapshotMu`) stay in the root adapter that takes the lock.
- **Freeze an owner's refusals.** `testdata/wire/native/denials.json`: a stranger and an owner acting
  for another franchise on every owner act and desk, an owner on the commissioner's starting snapshot
  and overrides, each shut window, a passed deadline, a locked lineup player, a pick out of turn, an
  unreadable session and each malformed request. Skipping the gate's refusal in `runNativeAction`
  changes six entries (the store's second gate still refuses, in other words).

### Offers, Personal, Pool and Bids seams

- Local bid plans and native league bids are different doors. Do not add native authorization
  to a local what-if plan, or move native permission checks into its presentation service.
  Preserve every manual hand-off, withdrawal, conditional-drop and commissioner override path.
- Keep `movesMu` around the entire local load/check/save/log/notify sequence, including live
  revalidation at hand-off and the original reopen/rebase order when revising. Pure parsing and
  wording can move; the root invokes them at the original point in that sequence.
- Offer observation still runs under the caller's `seasonMu`, with the same pending trades and
  ending evidence in one store operation. Calendar classification and disclosure are pure;
  transaction coordination and archive/live reads remain root-owned.

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
- **Staging and the hook.** The pre-commit hook stashes unstaged changes and lints the staged
  tree. A `git mv` stages the old content at the new path, so stage the rewritten file (or
  unstage the rename) before committing an unrelated earlier step, or the hook sees a
  `package main` file inside the service directory.
- **fillWire** handles strings, floats, bools and ints; a twin with another scalar kind needs a
  case added there.
- **Tiny files.** Each new non-test file needs at least 40 lines of code, or it counts against
  the bloat ratchet's `tiny_files` (baseline 22). Put the package doc in a real file.
- **Store the held value as before.** `archive.Read` returns the zero `Snapshot` when nothing is
  verified; the adapter only replaces the held copy when the result is a real snapshot, as the
  old code did.

## Counterparties additions

- Evidence-wrapped screens reuse `report.Reading[T]`; the root retains its Wails wire
  `Reading[T]` and converts the envelope along with the payload. Keep all nested named wire
  types at root, not only the board. Explicit converters preserve nil versus empty slices.
- Shared shell vocabulary can be passed as named formatting functions with the data loader and
  build stamp. This avoids importing another feature or copying its formatting implementation.
- Move implementation tests with construction; retain unrelated cross-feature assertions at
  root. A populated wire-twin test complements fixture goldens by exercising nullable fields.
- Placement and SYSTEM_MAP changes that remove old paths must accompany the move commit:
  those gates reject references to deleted files. Finish the narrative and root ratchet next.

## Comms additions

- Held season exports pass as `report.Feed[T]` snapshots; `report.Shell`, `report.Time` and
  `report.ProseName` own existing presentation vocabulary. Root helpers delegate, rather than
  copying formatters into each feature. `Feed` does not transfer ownership of held state.
- Shared-lock reads and refresh publication can remain root callbacks. Capture freshness
  metadata before taking the existing refresh lock, then pass it to the service under that
  lock, preserving the original ordering. Authentication, key removal and archive fallback
  stay with their owner.
- Preserve cross-feature scenarios in their original root files. A test-only fixture facade
  can delegate to the extracted service without moving another feature's test or duplicating
  production construction. Shared fixture loading lives in `commstest`.
- Mutants must run in isolation: restore the source before starting a full-suite compilation,
  otherwise an unrelated test process can compile the planted mutation.

## Sources additions

- A new root binding golden temporarily needs a placement row and one ratchet slot in its own
  commit. Lower the ratchet with the moved/deleted paths; do not leave the temporary increase.
- Stateful compilation can move synchronously behind the existing root `factsMu.TryLock`; its
  scheduler and cache warmers stay root. Likewise, keep `loadingSignals`, held-report publication
  and the deferred health notification in their original order around the service call.
- Pass live-state reads as explicit callbacks when they originally followed store writes. In
  crosswalk linking, moving roster/name snapshots before link writes changes ordering; invoke
  those readers in the service only after the links are written and read back.
- Split borrowed history capabilities by operation (health, facts, imports, signals, crosswalk),
  rather than creating one oversized store interface. Settings shared with photos and
  cross-feature refresh routing remain with their root owner.
- Preserve root integration scenarios that exercise shared state or another feature. Move pure
  implementation assertions beside the service; a test-only bridge can keep an unrelated root
  scenario intact without adding a production builder back to the adapter.

## Scoring additions

- Inventory the live methods after the purge, not the pre-purge brief. Do not restore retired
  bindings just to golden-test them. When none survive, freeze the root entry point consumed by
  the live feature (Scoring's power board feeds Race), including empty and populated results.
- Keep scheduling, dirty-flag handoff, shared directory reads and event publication with their
  root locks. Move the synchronous model pass; pass shared snapshots as callbacks at the original
  read point, rather than taking them before validation or runner construction.
- Service, root adapter and implementation-test moves are one compiling commit when they depend
  on each other. Placement references to deleted paths and the lowered ratchet accompany it;
  narrative and recipe notes can follow separately. A feature-specific result needs wire twins,
  not a duplicate of the shared report contracts when it does not return report tables.

## Stateful features

- `App` owns `seasonMu`, refresh/news/archive coordination, held caches, status, timers and
  goroutines. A feature service never acquires the season lock or starts background work.
  Keep refresh/status bindings at root when they depend on that ownership.
- Reading adapter: lock → copy the mutable indexes and read metadata into a detached input →
  unlock → build the service snapshot → call the synchronous pure constructor → convert the
  result to the root-declared wire types. Copy payloads too if their owner mutates them in place;
  immutable published payloads may be borrowed. If a copy is too expensive or a call must retain
  the lock, leave that operation at root and report the exception rather than hiding the lock.
- Research's `researchInput` captures feed/news indexes, depth links, injuries and schedule
  indexes under `seasonMu`; directory lookup, detached roster view and copied rulebook config
  come from their existing owners. `research.Snapshot`, `Held` and `Feed[T]` have no live `App`
  references. Published feed slices and the lookup are immutable; refresh replaces them.
  `Now` is captured by the adapter, not read by the service.
- Keep a shared calculation with its existing owner. Research's snapshot receives finished-week
  scores from the Market-owned root aggregation and cut charges from the shared bid rule;
  neither implementation is copied into the service. Archive hydration, fallback and batched
  news refresh remain root because they coordinate cache publication and refresh state.
- Root race tests must exercise refresh versus reading, not only parallel constructors.
  Research reads while a news fetch is blocked, then reads through publication; a detached-index
  guard checks feed/news/link/schedule maps and builds with the root season lock held. Package
  tests run parallel constructors against one snapshot and assert that its inputs are unchanged.
  Run both root and service under `go test -race`; plant a missing copy or wrong refresh lock to
  prove the guards reject it. Keep wire goldens and populated converter twins at root, pure
  construction assertions beside the service, and shared raw fixtures in `researchtest`.

- Race's `raceReadInput` captures feed/live metadata, kickoff-derived `LiveUnplayed` and a copied
  power-board index under `seasonMu`. Published payloads/rows are immutable; refresh replaces
  them. `raceInput.snapshot` converts power rows and creates live-week overlays after unlock,
  using Now's existing helper rather than copying another feature's calculation. The service's
  `Snapshot`, `Held` and `Feed[T]` own no locks, timers, callbacks or goroutines.
- A feature may already expose package-private construction to another feature. Keep a small
  root facade until that consumer moves: Race's `raceView` preserves Now's fields and vocabulary
  while delegating calculations to `race.View`. Do not edit the other feature merely to finish
  deleting a root file. Retain test-only wrappers for unrelated root integration scenarios.
- Race guards refresh versus reads through real feed publication, verifies the power index is
  detached (a missing clone fails), and constructs with the root season lock held. Package tests
  concurrently build one snapshot and compare its serialized inputs before/after.
