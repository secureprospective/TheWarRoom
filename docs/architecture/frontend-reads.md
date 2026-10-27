# Frontend held reads — plan Part 3

## Resume here

Shared primitive: `frontend/src/app/data/query.ts`. Four handwritten lifecycle implementations
now use it, serving **five store consumers**: Archive, Race, Comms, Draft reports and Market reports
(the last two already shared a factory). Selection/actions and public wrapper methods are preserved.
Next: migrate Control's simple held reads and Sources cache-scope reads, then the scoped historical
Market/Draft/Counterparties reads. Leave live polling and write-plan state until their separate
invalidation/lifetime requirements are covered. Do not replace UI/action state merely to inflate
migration counts.

## Contract

- The Go command registry remains the operation/argument authority. `nativegen -surface doors`
  now emits `commands/reads.gen.ts`, using real registry desktop exposures and typed bindings.
  Its `ReadArguments<K>` selects the existing generated `DesktopArgumentMap`, not another manual
  argument interface. Runtime shape validation uses those same registry binding schemas.
- Legacy desktop declarations do **not** consistently publish `Modes: [read]`: G0 includes
  Comms refresh and browser opening. Therefore the generator has a conservative five-adapter
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
- Each query instance owns a bounded 32-entry last-success cache. There is no process-global or
  cross-provider cache. `cache-first` is explicit opt-in; none of the migrated wrappers opt in yet.
  Explicit refresh clears cached scopes before the action; failure cannot turn an old entry into
  fresh evidence. No TTL, timer, automatic retry, write replay or in-flight coalescing is introduced.
- `invalidate()` clears cached values and fences pending publication, retaining the displayed
  last-good view. `reset()` also clears private/displayed data and restores initial selections.
  Neither cancels a transport promise (the existing Wails calls have no AbortSignal contract).
  Reset the instance when its account/league/provider lifetime ends before enabling cache-first.
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
| Control `control/state.ts` | league/settings/news/accounting reads | OPEN; independent readings and action outcomes |
| Sources `datasources/cacheState.ts` | TargetCacheScopes; clear outcomes | OPEN; clear is an action, not a cached read |
| Sources `datasources/state.ts` | sources/crosswalk/params joint view | OPEN; preserve read-before-parallel-subreads and partial results |
| Counterparties `counterparties/state.ts` | historical scope/details | OPEN; scoped detail cache and persisted filters |
| Market `market/state.ts` | lists/clock/stretches/profiles | OPEN; joint publication and per-profile scope |
| Historical Draft `market/draftState.ts` | DraftLab / pick-kind details | OPEN; scope revision plus per-kind cache |
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
selections, labels/refusals and last-good views; no browser/network/write action merely from reading.


### Regression checkpoint

Affected batch: 11 files / 101 tests pass, plus TypeScript no-emit. Two one-at-a-time actual-source
mutations were rejected (late publication without revision fencing; stale same-scope cache after
explicit refresh). Both restored before the final passing run. No production mutation committed.
