# Frontend held reads — plan Part 3

## Resume here

Shared primitive: `frontend/src/app/data/query.ts`. **Part 3 is closed** (§ Part 9): **37 registry read
adapters / 19 lifecycle modules**; read-session end is wired where an identity really changes (the
commissioner seat) and proven by lifetime at the native account. What remains is listed in § Part 9
"Remains": live polling, one owner ruling (Pool G1; the trade builder moved in revamp B2), and one optional
registry ruling (read modes on desktop declarations). Do not replace UI/action state to inflate counts.

## Contract

- The Go command registry remains the operation/argument authority. `nativegen -surface doors`
  now emits `commands/reads.gen.ts`, using real registry desktop exposures and typed bindings.
  Its `ReadArguments<K>` selects the existing generated `DesktopArgumentMap`, not another manual
  argument interface. Runtime shape validation uses those same registry binding schemas.
- The projection is generated from the registry: every G0 desktop exposure whose modes are exactly
  `[read]`. Legacy adapters opt in with `heldRead` in `commandcatalog/desktop.go` (the 37 audited
  held reads, 28 of which had no mode before Part 10); shared commissioner reads carry their native
  read mode, so all 26 other G0 commissioner reads are in the catalog too (64 entries with
  `TargetTradeBuilderRead`). G0 alone
  still proves nothing (Comms refresh and browser openers are G0): mark a legacy adapter only after
  reading its handler. The frozen door fixture records the 28 added modes.
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
| Sources `datasources/state.ts` | sources/crosswalk/params joint view | MOVED (part 7); two queries, sources first then crosswalk+params together |
| Counterparties `counterparties/state.ts` | historical scope/details | MOVED; per-team last-scope cache/tokens and pending dedup; filters/sort unchanged |
| Market `market/state.ts` | lists/clock/stretches/profiles | MOVED; joint publication and per-kind last-scope cache |
| Historical Draft `market/draftState.ts` | DraftLab / pick-kind details | MOVED; pending details fenced by refresh, retry invalidates per-kind cache |
| Assets `assets/state.ts` | asset board/profile | MOVED (part 7); one-time listener and failed-cell wording kept |
| Now `now/state.ts` | week/team live-season view | MOVED (part 7); weeks captured before refresh, franchise sampled after |
| Live game `now/liveState.ts` | game/player live detail | OPEN; polling/revision lifetime, deliberately left with live polling |
| Research `research/state.ts` | research/status/player/news | MOVED (part 7); board read only; status poll, player, open stay outside |
| Offers `offers/state.ts` | pending offers, refresh/actions | MOVED (part 7); actions never cached or replayed |
| Pool `pool/state.ts` | scoped pool and player detail | BLOCKED; `mfl.pool.read`/`TargetPool` is G1 in `commandcatalog/desktop.go`, not an admissible held read |
| Personal `personal/state.ts` | personal board and local changes | MOVED (part 7); one private query per franchise |
| Contracts `contracts/state.ts` | contract/player readings | NOT a read store: filters + action dispatch; release board reads through the release action command |
| Draft room `draftroom/state.ts` | native draft + owner actions | MOVED (part 7); reset fences reads |
| Shell `shell/lineups.ts` | lineup/trades/alerts per franchise | MOVED (part 7); one query per franchise, pending unpublished; `roster.ts` is pure projection |
| Shell `shell/tradeBuilderState.ts` | builder reading and isolated preview | MOVED (revamp B2); read is `TargetTradeBuilderRead` on the shared query (publishing into the builder store), preview is `TargetTradeBuilderPreview`, save and plan actions stay `TargetTradeBuilder`; `busy` still serializes every action |
| Commissioner `commissioner/hostedRoom.ts` | hosted league bundle | MOVED (part 7); keyed by native-lineup scope |
| Commissioner `commissioner/proxy.ts` (`leagueState`) | leagues picker + core/landing/ops settled reads | MOVED (part 8); picker in pre-read step, bundle keyed by league; controls read once |
| Commissioner `commissioner/{actReadings,competitionReadings}.ts` | one-shot option reads for an act form | NOT a read store: write-admission inputs, revision-checked per open |
| Commands `commands/{moves,sessionMoves,mflKey}.ts` | moves journal / MFL key status | NOT a read store (part 9 audit, § Part 9) |
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

## Part 7 — next compound/hosted slice (design)

Sources uses two shared queries on one view: sources publishes first, then crosswalk/parameters
publish together after their parallel reads. The busy/action/outcome guard stays outside both;
a refused action's error survives successful reads. Assets keeps its existing one-time listener
and failed-cell wording while retrying. Now captures weeks before refresh and samples franchise
afterward. Personal owns one private query per franchise; no write is cached or replayed. Native
draft uses its registry scope and existing refusal formatter; reset fences outstanding reads.

Hosted room will key the whole bundle by the registry's native-lineup scope (league, franchise,
week, read-at), while still reading assets first to derive season and reading the remaining views
in parallel. Existing hosted API/nativeEnvelope validates each actual registry request; parsers
and cross-reading league/revision/football checks remain in the room. Only the held-read lifecycle
moves to the shared query, not preview/confirm, the generation fence for writes, or canChange.
Refresh failure must retain the last bundle but set stale/read-only before pending clears. Dispose
must fence the query as well as writes and clear private readings. No cache-first or eager request.

## Part 7 — close

Moved this part: Sources joint view, Assets, Now, Offers, Personal, Draft room, hosted league
bundle, shell held readings (lineup/trades/alerts) and the Research board. **30 registry read
adapters / 18 lifecycle modules** on the shared tool (Part 6 ended at 17 / 9).

Shell readings: one query per franchise inside `createReadings`; the wrapper publishes only
non-pending states so subscribers see exactly the old notifications (success, or last reading +
error). Research: refresh/news are `before` steps only when requested, so a plain scope change still
calls the reader synchronously and an older reply is fenced; a superseded refresh/news load now
skips its own board read (its result was discarded before, so only the call is saved). The status
poll stays in the wrapper and a poll failure still survives the read's success (new test passes on
the pre-migration store and fails on an `accept` that drops it).

Blocked without a registry ruling: Pool (`offers.TargetPool`/`TargetBidPlans` are G1 MFL reads;
the generator admits only G0 held reads). The trade builder was split in revamp B2 once its read
stopped writing `whatif.db` (projections run on a throwaway scratch file).

Next slice: `commissioner/proxy.ts` leagueState. Proposed shape, not built: a leagues query (no
args) publishing the picker, then one bundle query keyed by `[league]` whose read runs the three
settled groups (core, landing, ops) and returns them whole; failure must also clear the core
readings (today's catch) — the query's `failure` is string-only, so either the wrapper resets
readings on error or the bundle read never rejects (settled groups already word their failures).
Controls stay read-once. Then provider/session invalidation design (brief step 2) once reads close.

## Part 8 — commissioner league state

One query on `commissioner.TargetCommissionerLedger` with Rules/People/Jobs/Notices/Digests as
companions, keyed by `[selected league]`. `choose` clears readings itself when another league is
named, then loads with a `before` step that reads the leagues and resolves/validates the
selection; the lazy argument function (sampled only for the current request) publishes the picker
and returns the league. The bundle read runs the three settled groups unchanged, so only the
picker/selection can reject; `failure` clears the six readings (hosted-room precedent) and keeps
the picker. Pending stays true from start to final publication (no flicker between stages).
Execute/forget/outcome stay in the wrapper on the same store. Difference: a superseded choice no
longer issues its discarded readings.

## Part 8 — provider/session invalidation (design)

A read session is one account/league/provider lifetime. `createReadSession()` returns
`join(query, 'shared' | 'private')` → leave, and `end()`. Shared members are invalidated (displayed
last-good view kept; caches dropped; old-session replies fenced). Private members are reset to their
initial state because the view held another identity's data. `end()` snapshots members, so a query
joining during end belongs to the next session. No listener, timer, transport, retry or reread is
installed; the code that changes identity calls `end()` and then its existing reread.

Not wired yet (next part). Owner of the session: one instance created beside the live provider
(`data/provider.ts` `getProvider`) and passed to feature constructors that already take a
provider — no new module global inside query.ts. Join table:

| Boundary (caller) | End when | Then |
|---|---|---|
| MFL key (`commands/registry.ts` `requestMFL`) | key league/season changes, or connected → absent | existing shell/now rereads |
| Native account (`control/NativeAccount.tsx`) | enter, forget, revoke-device, revoke-all accepted | its existing view reread |
| Commissioner seat (`commissioner/LocalSignIn.tsx`) | sign-in accepted | `leagueState.choose()` |

Private: Personal, shell lineups/trades/alerts, Offers, Draft room, hosted room, commissioner
leagueState. Shared: Archive, Race, Comms, reports, Control, Sources, Research, Assets, Now, Market,
Counterparties and historical details. `target:season` is not a session boundary (features already
reread on it). Ending without a following reread would blank private screens: wiring must test the
pair together. Tests exist for the primitive (3, three mutants rejected); wiring tests do not.

## Part 9 — session wiring and close

**Session ownership refined.** One app-wide session (the Part 8 sketch) would reset every private screen
at any boundary, and most have no reread at that moment. Each owner of an identity now owns its session,
created inside that owner (no module global, no listener):

| Boundary | Evidence | Wiring |
|---|---|---|
| Commissioner seat (`LocalSignIn` → `leagueState.changeSeat()`) | another seat may see other leagues, roles, controls | WIRED. `createLeagueState` joins its league query to a seat session as **private**; `changeSeat` ends it (readings, picker, read-once controls and outcome cleared; old-seat replies fenced) then `choose(previous selection)`. Selection survives exactly as the old `choose()` did. |
| Native account enter/forget/revoke-device/revoke-all | `NativeAccount.run` already drops `view.franchise` before admission | ALREADY ENDED by lifetime: `HostedLeague` unmounts → `room.dispose()` (generation++, `query.invalidate()`, private readings cleared), remount → `room.refresh()`. A second session object would duplicate dispose. Startup never carries a franchise (Go `serverclient` `status` sets only the message). |
| MFL key connect/forget (`requestMFL`) | league is the constant `ingestion.LeagueID`; season changes arrive by `target:season`; held reads are keyed by explicit franchise, not by key; key-dependent Go data (pending trades) is already fenced by `KeyGeneration` | NOT WIRED. No held read changes identity at the key boundary; ending would blank lineups/trades/alerts and reread identical data. Revisit only if a read becomes key-scoped. |

Tests (pair = end + reread together): `leagueState.test.ts` seat change (old-seat reply fenced, readings
cleared at once, same league and controls reread); `localSignIn.test.tsx` asserts an accepted sign-in calls
`changeSeat`; `nativeAccount.test.tsx` four cases (each action closes the open room before admission;
only `enter` with an assigned franchise reopens). Mutants rejected, each restored: no `seat.end()`;
`shared` membership; no reread; reread without the previous selection; `LocalSignIn` back to `choose()`;
`NativeAccount` without the pre-admission view clear (all four cases fail).

**Commands audit (brief step 2).** Not split, because neither is a held-read store:
- `moves.ts`/`sessionMoves.ts`: `moves` is one receipt journal written by both the `TargetMoves` read and
  draft/hand-off actions, merged audit-monotonically ("audits only grow"); per-franchise loads dedupe an
  in-flight load (the query tool revalidates instead). Moving the read part would change dedup and could
  roll back a hand-off. `drafting/draftErrors` are action state; `demo` is the fixture stand-in.
- `mflKey.ts`: one status written by read, connect and forget under one `pending` guard that serializes
  them; the read is the keyring check, not a held reading. Splitting would break that serialization.

**Command bar flags (brief step 3).** The bar's IDs are already generated from the registry
(`commandbar.gen.ts`, `Browser.Searchable`, no args); none are reads. The 37-adapter hand list in
`frontend/scripts/nativegen/reads.go` was replaced in Part 10 by registry read modes (above).

Gates: `tsc --noEmit` passes. Full frontend suite once (vitest, 2 threads): 141 files / 1,706 tests, 1
failure: the readability gate (`controls.test.ts`, >110-char lines) caught two new test lines. Both were
shortened (a test title and a comment, no logic change); that gate plus the two edited files then pass
(3 files / 41 tests). The full suite was not rerun for that string-only change.

### Remains (exact)

1. Live game `now/liveState.ts` — polling/revision lifetime; stays with live polling.
2. Pool `pool/state.ts` — owner ruling: `offers.TargetPool`/`TargetBidPlans` are G1.
3. Done in revamp B2: trade builder read/preview/save split; the read is on the shared query.
4. Done in Part 10: registry read modes replace the generator list.
5. `cache-first` stays limited to historical details; extending it needs the session wiring above per owner.
Not targets: contracts/commissioner action and option-read modules, moves journal, MFL key status,
navigation/sort/presentation stores (inventory table).

