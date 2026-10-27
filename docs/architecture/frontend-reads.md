# Frontend held reads — plan Part 3

## Resume here

Shared primitive: `frontend/src/app/data/query.ts`. **Nine handwritten lifecycle implementations
now use it, serving ten store consumers**: Archive, Race, Comms, Draft reports, Market reports,
Control, Sources cache scopes, historical Market, historical Draft and Counterparties. The registry
projection contains 17 audited read adapters. Writes, polling and session invalidation are unchanged.
Next: Sources' compound sources/crosswalk/params view (ordered initial read, parallel independent
subreads and partial failures), then Assets and Now held reads. Leave live polling and write-plan
state for their separate invalidation/lifetime requirements. Do not replace UI/action state merely
to inflate migration counts.

## Contract

- The Go command registry remains the operation/argument authority. `nativegen -surface doors`
  now emits `commands/reads.gen.ts`, using real registry desktop exposures and typed bindings.
  Its `ReadArguments<K>` selects the existing generated `DesktopArgumentMap`, not another manual
  argument interface. Runtime shape validation uses those same registry binding schemas.
- Legacy desktop declarations do **not** consistently publish `Modes: [read]`: G0 includes
  Comms refresh and browser opening. Therefore the generator has a conservative 17-adapter
  migration list of audited held reads. Every entry must exist in the registry, have G0 and have
  no conflicting declared modes. Do not infer purity from gravity, method spelling or availability.
  Expand this list only after reading its handler. No registry role/mode/schema/binding changed.
- The projection is **not** a new command door. Existing feature `api.ts` adapters still perform
  transport access and parse/validate results. There is no generic Wails reflection, HTTP fallback,
  capability grant or response-schema invention. Result type comes from the typed parsed reader
  port; the Go binding registry currently has argument shapes, not frontend parsed-result schemas.
- `createReadQuery(command, {initial, read, accept})` owns the Zustand read lifecycle and hook.
  Wrappers keep requests (filters/week/team), local presentation (Comms limit), actions and server
  echoes. `accept` publishes only the latest successful parsed response. Errors preserve the last
  good view; pending/error bookkeeping cannot be overwritten by old successes/failures/finally.
- `load(args | lazyArgs, {before?, policy?})` defaults to **revalidate**, preserving the old stores'
  repeated-read behavior. `before` is an explicitly supplied refresh action, never inferred or run
  on mount by the query tool. Its failure stops the read; lazy arguments are sampled afterward,
  as Race/Comms previously did. A superseded pre-read action cannot publish or start a stale read.
- Keys contain registry logical ID, qualified adapter and all validated positional arguments.
  Object property order is normalized; array order is retained. Leading-zero IDs, season, team,
  viewer identity and scope values are not coerced. Inputs are detached before asynchronous reads.
- Each query instance owns a bounded last-success cache (32 entries by default, one for historical details). There is no process-global or
  cross-provider cache. `cache-first` is explicit opt-in; only historical details opt in, preserving their pre-existing
  one-last-scope-per-kind/team caches. Main/compound views still always revalidate.
  Explicit refresh clears cached scopes before the action; failure cannot turn an old entry into
  fresh evidence. No TTL, timer, automatic retry, write replay or in-flight coalescing is introduced.
- `invalidate()` clears cached values and fences pending publication, retaining the displayed
  last-good view. `reset()` also clears private/displayed data and restores initial selections.
  Neither cancels a transport promise (the existing Wails calls have no AbortSignal contract).
  Reset the instance when its account/league/provider lifetime ends; that integration is still
  required before extending cache-first beyond the pre-existing historical detail caches.
- Existing feature event adapters and UI mount subscriptions stay unchanged in this batch.
  Next migrations must route events/actions to explicit invalidation/revalidation without duplicate
  listeners or refresh loops. No new global event listener or eager request was installed.

## Migration inventory

Status describes the **read lifecycle**, not every line in a feature's store. Paths below are
relative to `frontend/src/app/`. Query-neutral/action-only state is explicitly out of scope.

| Consumer / path | Held commands / distinguishing state | Status / next constraint |
|---|---|---|
| Archive `archive/state.ts` | TargetArchive; season/team/other | MOVED; server echoes and local validation retained |
| Race `race/state.ts` | TargetRace; week/team/mine, optional explicit refresh | MOVED; same read/refresh order |
| Comms `comms/state.ts` | TargetComms; team/kind, local pagination, explicit open/refresh | MOVED; open/more are not reads |
| Draft reports `draftreports/state.ts` | TargetDraftReports; refresh-before-read | MOVED |
| Market reports `reports/marketState.ts` | TargetMarketReports; shared report factory | MOVED; explicitly supplies its own registry key, never Draft's |
| Control `control/state.ts` | league/settings/news/accounting reads | MOVED; joint league publication, independent news; accounting open remains an action |
| Sources `datasources/cacheState.ts` | TargetCacheScopes; clear outcomes | MOVED; clear/busy/outcomes remain action state; forget fences reads |
| Sources `datasources/state.ts` | sources/crosswalk/params joint view | OPEN; preserve read-before-parallel-subreads and partial results |
| Counterparties `counterparties/state.ts` | historical scope/details | MOVED; per-team last-scope cache/tokens and pending dedup; filters/sort unchanged |
| Market `market/state.ts` | lists/clock/stretches/profiles | MOVED; joint publication and per-kind last-scope cache |
| Historical Draft `market/draftState.ts` | DraftLab / pick-kind details | MOVED; pending details fenced by refresh, retry invalidates per-kind cache |
| Assets `assets/state.ts` | asset board/profile | OPEN; selection and detail identity |
| Now `now/state.ts` | week/team live-season view | OPEN; refresh and held snapshot interplay |
| Live game `now/liveState.ts` | game/player live detail | OPEN; polling/revision lifetime |
| Research `research/state.ts` | research/status/player/news | OPEN; cancellation/pending and news refresh |
| Offers `offers/state.ts` | pending offers, refresh/actions | OPEN; no automatic plan execution |
| Pool `pool/state.ts` | scoped pool and player detail | OPEN; persisted filter and multi-part results |
| Personal `personal/state.ts` | personal board and local changes | OPEN; write outcomes separate from reads |
| Contracts `contracts/state.ts` | contract/player readings | OPEN; plan/actions and local previews separate |
| Draft room `draftroom/state.ts` | native draft + owner actions | OPEN; hosted permissions and native scope |
| Shell `shell/lineups.ts`, `shell/roster.ts` | lineups/readings/roster snapshots | OPEN; shared provider and franchise identity |
| Shell `shell/tradeBuilderState.ts` | builder reading and isolated preview | OPEN; local plan writes never replay |
| Commissioner `commissioner/{actReadings,competitionReadings,hostedRoom,proxy}.ts` | hosted/native readings | OPEN; preserve credential/provider/league lifetime |
| Commands `commands/{moves,sessionMoves,mflKey}.ts` | moves/key/session state | OPEN audit; split read state from action state first |
| `contracts/{contractActState,releases}.ts`, `commissioner/{backupState,surfaceActions,controls}.ts` | action results / previews / controls | MIXED; migrate only audited reads, not mutation admission |
| `shell/state.ts`, `reports/sortState.ts`, `clock/windowState.ts`, search/specimen/ticker | navigation/sort/edit/action/presentation | NOT read-store migration targets |

This is a responsibility inventory, not an asserted exact count of Zustand stores: several modules
contain multiple readers, some use custom subscriptions, and several similarly named state files
are action-only. The initial filesystem scan found 25 production `*state*.ts` / `*State*.ts` paths;
that filename count is not the migration denominator.

## Exercised evidence

- Existing Archive/Race/Comms/Draft reports suites pass unchanged behavior assertions: held-data
  retention, request ordering, server echoes, validation, stale response exclusion, explicit browser
  opening and event cleanup. Market reports' shared factory now explicitly uses its own command ID.
- New query tests exercise registry-key identity/full scope, canonical objects, runtime and compile-
  time argument rejection, last-good/error retry, bounded/private caches, cache-first opt-in,
  refresh-before-lazy-scope, refresh refusal, late success/failure/finally fencing, invalidate/reset.
- Generator tests reject missing opted-in adapters and an adapter changed to execute mode, exclude
  refresh/open methods and check generated-door freshness/drift. The existing freshness gate covers
  the new projection automatically. Transport guard still permits backend access only in api.ts.

No new dependency, generated Wails binding, endpoint, UI layout, permission, role or schema change.
Deferred live test on a disposable profile: choose Archive/Race teams and weeks, Comms filters,
refresh Draft/Market reports, cause a read failure and retry. Passing means unchanged accepted
selections, labels/refusals and last-good views; no new implicit browser/write action, and existing
refresh ports retain their previous fetch/admission behavior.


### Regression checkpoint

Affected batch: 11 files / 101 tests pass, plus TypeScript no-emit. Two one-at-a-time actual-source
mutations were rejected (late publication without revision fencing; stale same-scope cache after
explicit refresh). Both restored before the final passing run. No production mutation committed.

Final importer/readability extension: 14 files / 312 tests pass. The source readability test first
caught two new >110-character query/test lines; split them without changing behavior or the gate.


## Part 6 — compound and scoped reads

Flow: existing feature reader/parser → registry-validated query → wrapper-owned view. No new
transport call or parser moved into the query tool. Control's league bundle uses settings as its
primary read plus By-laws/Accounting companion registry IDs; Market's bundle uses lists plus
clock/stretches IDs. All companions validate the same positional scope and participate in the key;
publication stays all-or-nothing after the existing Promise.all (Market still words each failed
subread separately). Control's independent news query shares its wrapper's view, not its cache;
search and open-accounting remain outside the query lifecycle. Sources clear is not a query and
cannot be replayed; its busy/outcome guard is unchanged.

`arguments` optionally projects a parsed frontend scope to the existing typed Go binding tuple.
Both the projection and original full scope are keyed, so Market's null/all versus empty-list UI
selections do not collapse even though its existing api.ts normalizes both to Go arrays. The reader
and accept port receive detached original inputs; validation still uses registry argument schemas.
Overloads require a typed projection for a differing reader input; ordinary queries retain the
registry tuple directly. No second response schema, route or capability is generated.

Historical detail queries have one entry per kind/team, matching the old signature maps rather than
introducing 32-scope history reuse. `fence()` stops pending publication but retains the last success;
Market/Draft call it when refreshing their main view, preserving their old revision spans. Retry
clears Draft/Counterparties detail caches. Counterparties retains its different original lifetime:
main refresh alone does not cancel a team read; a newer request for that same team fences the older
scope, different teams remain independent, and an already-pending identical panel is not duplicated.
The teamPending map observes the detail query status rather than implementing another read lifecycle.
Unavailable/failed parsed readings remain worded values (and cacheable where they were before), not
silently converted to generic errors. Existing global provider/session lifetimes remain unchanged;
this is not permission to enable cache-first for other reads before lifetime integration.

New tests cover Control's no-half-league publication/failure retention, Sources forget during a
read, compound Market publication, per-kind last-scope eviction, detail fences, Counterparties
same-scope pending dedup/per-team independence/latest scope/provider isolation, shared Control
view fields, companion validation and typed input projection preserving null versus empty scopes.
Three sequential actual-source mutations were rejected by their intended assertions: removing the
detail revision fence; collapsing original scope identity; replacing one-entry detail eviction with
32 entries. Restored between attempts; no mutant or scratch file remains. Skipped tests in focused
mutation runs are not counted as passes. Final targeted batch and production checks are recorded
in app-boundary.md. No UI layout/event listener, write/polling/session invalidation or dependency change.


Follow-up source review preserved Sources' successful-list error clearing even if a clear action
failed while that read was pending; the action's per-scope outcome remains displayed separately.
A real DOM/state regression covers that interleaving. Generator admission now individually rejects
execute mode, non-read gravity and removal for each of the 12 newly migrated registry adapters
(36 negative cases), not merely for the original Comms adapter.


Final source review shares the existing Market null-to-array binding projection between provider
and query keys (`marketScopeInput`); UI selections remain distinct in the cache key. Reader ports
retain their original method receiver through closures; a receiver-dependent Control regression
covers this. The final candidate's TypeScript check passes; the intermediate receiver-fixture typing
failure and immediate repair are recorded explicitly in app-boundary.md.
