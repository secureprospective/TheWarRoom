# Feature extraction: moving a feature out of `package main`

The earlier pure-service recipe remains below for existing adapters. New feature-bound APIs use
**Feature API template**, which moves bindings as well as construction out of package main.
Shared report/season contracts and a feature-owned Host keep transport and state separate.

## Feature API template

This is the current extraction pattern. The older sections below describe the earlier pure-service
stage; their root-owned binding/wire/state instructions do **not** apply to a feature API move.
Comms is the first instance: `internal/app/comms/api.go`, `internal/app/core/host_comms.go` and
`frontend/src/app/comms/api.ts`. Keep behavior, refusal order, wording and JSON unchanged.

1. **Inventory and freeze.** Record the root-file count, the feature tests' uncached runtime and
   every exported method on the bound pointer. Freeze reads, refreshes, hand-offs and denials as
   byte-comparison goldens in `testdata/wire/<feature>`. Call the public API in the tests; include
   populated and zero/nil/empty converter twins. Keep canonical fixtures in place and use a path
   helper (`commstest.RepoPath`), never copied goldens. Pin the zone through the port, not `time.Local`.
2. **Separate ownership before moving.** `App` has a **named** `core *core.State`, never an
   embedded State. Wails reflects every exported pointer method, including promoted helpers;
   embedding would publish lock/state helpers. Shared state and every shared mutex live in core.
   Root adapters temporarily use exported fields; feature packages never do. Data-directory
   paths go through `core.Paths`. Update all direct `App{...}` test fixtures with initialized core
   state, including zero-state refusal fixtures, in the same compiling commit.
3. **Declare the port in the feature.** Add `type Host interface` in `<feature>/api.go`, listing
   only the reads/effects this API needs. `API` has named, unexported `host Host` and (for stateful features) `held *Held`
   fields. Stateless APIs use `New(Host)`; stateful APIs use `New(Host, *Held)`. Neither
   imports core. Create exactly one unbound holder per App in common `NewApp` construction,
   store it on a named unexported field and bind `sources.New(app.core, app.sourcesHeld)`.
   Update direct App fixtures too. Never embed or bind Held. Shared state stays in core;
   feature-private reports and their short data mutexes move to Held. Its Publish and Snapshot
   methods lock and deep-clone on both ingress and egress, preserving nil versus empty at
   every depth (including each signals load's Missing slice). Port signatures use lower-level value types or shared report/season
   contracts. Do not return a feature-owned Input from core: return lower-level pieces and
   assemble the Input in the API. A lower, genuinely shared type can move to report/season; a
   feature-only cache should stay with its feature instead of creating a core→feature cycle.
4. **Implement the host in `core/host_<feature>.go`.** Methods on `*State` take the owning lock,
   copy nested mutable slices/maps/pointers, unlock and return detached data. An opaque immutable
   lookup can be shared if its public API exposes no mutation. Preserve the existing lock order,
   refresh serialization, freshness capture before the refresh lock, key checks, last-good
   retention/clearing, archive fallback and event order. A callback into a remaining root-owned
   cache must document its caller-held lock; never lock the same mutex twice. Shared fallback
   and held-feed primitives already live in core; reuse them instead of cloning them.
   Operation serializers (LoadingSignals, RefreshMu) remain core/root with their original
   spans, lock order and deferred notifications after release. No holder data lock spans a
   Host call, callback, I/O, event or another holder lock. Where an existing critical section
   jointly captures shared and private state, retain its outer shared lock for every reader
   and writer. Capture through an unbound root composition adapter: core lock → holder
   snapshot → shared copy → unlock. Do not replace this with independent snapshots or call
   a core method that re-locks that mutex. Cross-feature reads use the reader's Host port,
   implemented by an unbound root adapter over the other feature's holder; never a sibling import.
5. **Move the wire boundary.** Move the public methods, Raw request validation, exported wire
   declarations and converters into `<feature>/api.go`, with the exact method names, arguments,
   JSON tags and nil behavior. Remove emptied root files. Compose only in `main.go` by adding
   `<feature>.New(app.core)` to Wails' Bind list. Update root callers such as cache refresh routing
   to call that API rather than restoring an App wrapper. The assignment checks the Host
   structurally. Do not expose helper methods on API: Wails will bind them too.
6. **Catalog and manifest.** Expose package-level `Commands() []catalog.Command`, **not an API
   method**, with stable registry-style IDs and each bound method name. Availability is desktop,
   server or both; browser-opening commands are desktop-only. Copy known role/gravity/version
   metadata; use `catalog.Unset` for genuinely unknown values. Unset must block future server
   publication; it changes no desktop authorization now. Add the bound pointer and its Commands
   to `TestBindingManifest` in `selfprotect_test.go`, then update
   `testdata/bindings/manifest.txt`: the old App list minus exactly the moved methods, plus the
   feature API methods. Runtime and TestBindingManifest use one `bindings(app)` function;
   reject unexpected bound types as well as methods, including a directly bound Held.
   Remaining App methods are temporarily catalog-exempt. Plant an extra
   exported helper and prove the test rejects both its surface and missing catalog entry.
7. **Frontend adapter and generation.** Run `make bindings` with the pinned Wails CLI. Generated
   model namespaces may change; review those diffs, never hand-edit the generated files. Only
   `frontend/src/app/<feature>/api.ts` imports that feature's generated bindings/models or touches
   `window.go.<feature>`. It owns event subscribe/unsubscribe too; state and components call it.
   Keep the no-window/no-go browser path honest and unchanged. Test all calls, event delivery,
   cleanup and disposal before an async runtime import resolves. Do not migrate other features'
   legacy main/App imports as part of this move.
8. **Guards, placement and proof.** Extend depguard with the feature's own subtree, report,
   season and catalog. Core may import no feature and no Wails; no feature may import core,
   including tests. `scripts/check-frontend-transport.sh` runs in `make lint` and checks generated
   feature imports, feature models and direct/bracket window.go accesses. Map generated Go
   packages explicitly to frontend folders (`comms→comms`, `sources→datasources`), rejecting
   unmapped packages. Reject moved main/App bindings outside the adapter, without migrating
   other features' legacy imports. Bite-test each new
   boundary with a compiling forbidden import/access, then restore it. Update SYSTEM_MAP,
   placement.tsv and the actual root-go-baseline in the move commit. Move fast unit/API tests
   beside the feature using a fake Host. Test core separately with `coretest.New(t).Build()`;
   it supplies isolated stores and a memory keyring, and `coretest.NewTransport` never opens a
   socket. Keep root cross-feature scenarios and a small composition/refresh golden where they
   still prove real wiring. Race an actual refresh publication against reads, show reads can
   complete during refresh I/O, and mutate inputs and snapshots to prove ingress/egress
   detachment. Exercise nested nil/empty slices, publication/read races, serializer/event order
   and joint-capture consistency. Sources' stateful example holds crosswalk and signals reports
   in `sources.Held`; shared season/lookup state and the whole-load serializer stay in core. Plant a missing
   clone/lock and prove the guards fail. Run build, lint, focused race tests, root Wire/Manifest,
   frontend tests/typecheck/build and binding freshness; CI runs the full make verify when a
   scoped extraction brief excludes the full root suite.

**Comms ownership and metadata.** Core holds transactions/boards and their locks, key access and
refresh coordination. `archiveCache`, `assetModel`, `control`, `research`, `race`, `now`, `liveGame`,
`crosswalk` and `signals` remain on App: their types belong to other feature packages or to the
still-root feature adapters. Core's archive callback accesses archiveCache under FactsMu; the
transaction callback publishes through the existing root fetcher under RefreshMu. No feature
imports core. The catalog records the current compound desktop `comms.read` command's ambient
roles (`gm,commish,admin`) and gravity (`G0`) for its read/refresh/open actions. The registry's social
page gravity is not a newly enforced binding policy. No wire/API version was previously declared,
so Version is Unset. This does not grant server publication or alter any permission check.

**Request scope — recorded, not built.** Commissioner still reads the session and the app-lifetime
context from shared state; this pilot does not change that. Server reuse needs a per-call scope
containing context, principal and league, passed explicitly into the operation. Never swap a
request's identity, league or context onto shared State: concurrent callers would overwrite each
other. Resolve that seam before publishing Commissioner or any identity-sensitive server route.

**Next-feature traps.** Shared wire envelopes still declared at root cannot be imported by a
feature. Move a genuinely shared contract lower without inventing a new JSON shape; otherwise
report the blocker. Feature-specific held types/caches remain root-owned until their own turn.
Callbacks must retain original transaction/lock spans. Existing direct App literals need core
initialization. Catalog Unset versions need a publication decision, not a guessed default. Wails
currently emits generic-model "Not found" diagnostics even on a successful generation; compare
fresh output and typecheck rather than hand-fixing its output.

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
The contracts and roster-action extraction (reorg PR 13) added the plans a GM drafts locally, before any move on MFL.

- **The request's franchise is the scope.** These bindings have no session or principal: the franchise
  in the request owns every plan, and a plan, bid, what-if move or player of another franchise is refused
  in the root adapter (`contractActAction`, `releaseAction`, `acquisitionPlayer`, `changeWhatIf`,
  `replayWhatIf`, `contractActFacts`, `lineupSelection`), in the same place as before. A service prices
  or words a value the root has already scoped (`ActFacts` takes the player the root found on the
  roster); it never looks a plan up by id.
- **Windows and locks are decided at root.** The coordinator phase's window refusal is computed in the
  adapter and passed in (`ActFacts` takes `window`); the lineup lock, the week match and the MFL host
  checks stay in `lineupRequest`; the release revalidation and buyout count stay beside the plan ledger.
- **Plan writes keep their lock.** Each draft, transition, invalidation and supersede is a plan-ledger
  call under `movesMu` (and `whatifMu` for the what-if store) in the root adapter; the read that refuses
  a second open plan (`refuseSecondContractPlan`) stays under the same lock as the write, so two
  simultaneous drafts land once (`TestContractsSimultaneousPlansLandOnce`, which fails without either lock).
- **Freeze the refusals.** `testdata/wire/contracts/denials.json`: every malformed request on every
  binding, each cross-franchise plan, move and player, an unheld franchise, each shut window (in-season
  tender and buyout and their hand-offs, a locked and an unknown lineup week), a stale what-if, a second
  open plan, an unconfirmed release and a fresh plan's rebase. Dropping the franchise check in
  `contractActAction` changes four entries.

### Trades seams

- The local MFL what-if desk has no native principal check; do not add one while extracting
  presentation. Native trade doors and commissioner overrides belong to Native/Commissioner
  and stay untouched. Local accept/exit ownership, plan franchise, intent, pending-offer and
  safe-URL checks stay at root, in their original order.
- Keep the builder's `whatifMu` → `refreshMu` span and each original `movesMu` span. Simulation
  construction, coordinator execution, live hand-off revalidation, supersession, plan/receipt
  saves and notifications remain root-owned; moving them would split the existing lifecycle.
  `trades.ProjectedTeam` receives detached simulated-player values after execution. It never
  writes a store. Shared what-if money formatting is an explicit function, not a copied rule.
- Catalog inputs are read under their existing owners. The pick-value callback preserves the
  original per-pick model/time read; the leg reader borrows only the local simulation's player
  lookup. Services neither acquire locks nor discover the caller.
- Freeze all three local bindings, with a sampled populated projection and fully populated
  converter twins, plus native non-owner/wrong-seat/override/window/deadline refusals and local
  accept/revoke refusals. Skipping the local revoke check fails the denial golden; dropping a
  counterparty name fails both wire guards. Removing `tradeState`'s season lock fails the
  held-publication race guard. Parallel service reads must leave held inputs unchanged.

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

### Counterparties Feature API move

- The frozen Lab has no readiness/authorization gate, shared mutex or refresh publication.
  `core.State.Lab` preserves the shared once-decoder and zero-state fallback; the host detaches
  only Trades (including nested leg/player pointers), TeamSeasons and Franchises plus the two
  timestamps the service reads. Do not add a live-feed lock or a startup wait to this path.
- Existing service payloads already have identical wire fields, so the API uses them directly
  through concrete `BoardReading` and `TeamCounterpartReading` report envelopes. Wails discovers
  their nested models; returning the generic `report.Reading[T]` directly omits the nested
  payload model classes. Envelope `value: any` remains the existing Wails limitation.
  There is no payload converter left to twin-test. Canonical root goldens still call the API.
- Shared asset names/percent formatting join Number/Count in report; Market delegates. Root
  Offers retains its own TeamOption wire declaration and provenance delegate; unrelated Draft
  wording assertions move to its existing test file. Neither scoped root file needs to remain.
- The registry has endpoint.open navigation verbs, not verbs for either backend read. Their
  command IDs, roles, gravity and version remain Unset, blocking future server publication.
  Counterparties has no runtime subscription or disposal lifecycle to migrate.

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
  root facade until that consumer moves: Now delegates the shared season view to `season.View` inside its service; the root keeps only
  the Race facade still used by integration tests. Remove obsolete helpers when the last production
  consumer moves, without changing the other feature's bindings. Retain test-only wrappers for
  unrelated root integration scenarios.
- Race guards refresh versus reads through real feed publication, verifies the power index is
  detached (a missing clone fails), and constructs with the root season lock held. Package tests
  concurrently build one snapshot and compare its serialized inputs before/after.
## Stateful features: Control, refresh and clock

- `App` remains the owner of `seasonMu`, `weekMu`, refresh serialization, cached feeds,
  credentials, timers and goroutine lifetimes. Services never acquire these locks or start
  workers. Keep refresh/status bindings at root when they schedule, publish or access a store.
- Take the owning lock, copy the reading's inputs, release it, then invoke the synchronous
  service. Copy nested slices and maps, not just their headers. Control's `WeekSnapshot` owns
  its game slice; `WindowSnapshot` owns its schedule map, week slices and matchup slices.
  The settings adapter clones the scoring-name map under `seasonMu`; `ActiveConfig` already
  returns a deep copy under the rulebook's own lock. Site news copies its raw item slice under
  `seasonMu` (each raw item contains only value strings).
- Keep publication and archive/network fallback in the original order at root. Control's
  settings/news adapters may refresh on demand before constructing a reading. Its service
  owns only report construction, bylaw/parameter wording, alert ordering/bye wording and
  week/window/gameday calculations. Wire declarations remain in `main`.
- Alert acquisition and cross-feature coordination stay at root: lineup/trade bindings,
  roster/injury reads, archive-only byes and schedule fallback. Only their independent
  presentation functions move. Likewise, week fetching/phase transitions and window writes
  remain with their existing root coordinators. Do not move a callback that still needs the
  lock or a transaction just to make the adapter look smaller.
- Keep root binding goldens and wire-twin checks. Run a concurrent refresh-versus-read test
  under `go test -race`; the Control test drives real week refresh and season schedule
  publication while multiple readers build clock/window snapshots. Also mutate held nested
  data after capture to prove detachment, and evaluate the service while the root locks are
  held to prove it does not acquire them. Removing either slice clone must fail the test.

- Now's `nowReadInput` copies schedule and first-score indexes plus feed metadata under
  `seasonMu` into `now.Snapshot`, `Held`, `FirstScores` and `Feed[T]`. It borrows immutable
  published weekly results, brackets, standings, lineups, projections and schedule payloads;
  lookup and copied rulebook config come from their existing owners after unlock. The root
  retains first-score archive discovery and schedule/bye refresh publication.
- Live game's root owns the polling adapter, `liveGameMu`, backoff, archive I/O, crosswalk
  reads and photo policy. `now.LiveSnapshot` receives board metadata and a detached detail
  index, never the adapter's fetcher, final store or scheduling state. Pure construction may
  run within the root's serialized demand-driven poll, but never under `seasonMu`. Starter
  selection captures immutable live/week payloads under that lock and calls the service after
  unlock. No timers or goroutines move into `internal/app/now`.
- Now's root tests read through actual `TargetNowRefresh` publication, guard detached indexes
  and construct while the season lock is held. Service tests build Now and Live snapshots
  concurrently and compare serialized inputs before/after. A planted JSON-tag rename fails
  the root goldens; a planted schedule-index deletion fails the immutability/race guard.
