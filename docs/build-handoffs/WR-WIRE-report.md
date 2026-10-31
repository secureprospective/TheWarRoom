# WR-WIRE implementation report

## Disposition

Implemented and committed locally on `build/WR-wire`. No pull/push, backend changes, dependency changes,
HQ-file edits, or locked-mock edits. **Not fully accepted:** one inherited frontend test fails in
forbidden HQ files, and a disposable-profile visual/live acceptance run remains necessary.

## Commits

| Commit | Part |
|---|---|
| `90ae599a` | Shell, header, initial calendar, lazy navigation entry |
| `920e63d1` | Private current trade, shelves/trays, previews and native lineup delta |
| `dc745b80` | Targets, own bids, pool, draft and native entry |
| `ab3cc9bf` | Inspector, calendar, drawers and missing-region states |
| `e885344e` | Region/style tests, command-backed controls, scouting dialog, build fixes |
| `3e7e19ed` | Reject mismatched research before publishing; failing-mutant regression |

All commits carry the requested co-author/session trailers and `[AI-assisted]`; hooks were not bypassed.
This report and the handoff are committed separately after these implementation commits.

## Per mock region

The WR row groups below follow `tie-WR.md`, `terra-WR.md` and `DATA_FLOW_MAP.md`.
“Unavailable” means the region displays **Not available yet**, not mock numbers.

| Region / WR rows | Wired read / behavior | Unavailable or partial |
|---|---|---|
| Rail and tabs, 001–008, 354–359 | Existing War Room route lazily mounts v6; local desk/market/draft/plans tabs; existing navigation command door | Live/unread counts and shortcut customization have no suitable read. Native entry is in the server workroom. |
| Franchise table, 009–019 | `TargetSnapshot`; `TargetRace` with `topn` reads record, standing, roster value/rank, `starterN`; label is `Best {starterN} (any position)` | Competitive window, posture, needs/surplus and value-by-age histogram have no matching scoped read. Native Best-N is unavailable rather than substituting the MFL model. |
| Cap runway, 017–019 | Mirror `TargetContracts` current room and held future commitments; native `TargetNativeCapCheck` runway via the existing HQ read adapter | Mirror future room is unavailable: future commitment is not room, and today's cap is not assumed for future years. Three visible runway rows are a display horizon, not a league rule. |
| Calendar/deadline strip, 020–027 | Mirror `TargetClock`; native `TargetLeagueCalendar` (`calendar.read`) windows and trade deadline, viewer timezone | Private plan/due-decision counts are unavailable. No fabricated event dates. |
| Current builder and trays, 028–038 | Mirror `TargetTradeBuilderRead` and `TargetTradeBuilderPreview`; native `TargetNativeAssets` and private `trade.offer` **preview**; player/pick selection, partner switch, drag into trays, clear, inspector-to-trade | Only an in-memory current draft. No claim of autosave, saved versions or successful send. |
| Shelves/partner, 039–048 | Mirror builder assets provide contract and held base value; snapshot birthdates provide age. Public partner `TargetRace`/snapshot room, or native `standings.read`/`native.cap.check` | Native scoped base asset valuations and some player positions/ages are unavailable. Posture, needs, spare and lens-based shopping advice are unavailable. |
| Lenses/totals/verdict/tune, 049–089, 360–361 | Existing preview reasons and before/after cap/roster figures; native D33 returns both sides' before/after starters, changes, per-week/this-week deltas and notes | D28 lens totals/verdict and D32 tune suggestions are unavailable. No assertion that a successful preview means accepted or executed. |
| Lineup wording, D33 | `TargetNativeLineupDelta` (`lineupdelta.read`); benched = “Your/Their best lineup would bench …”; every unprojected identity is listed; incomplete aggregate deltas and unavailable player projections are not displayed as zero | No mirror D33 endpoint. Pick-only proposals have no player delta. The inspector's hypothetical replacement/need rows are not connected to the current delta; the full result is on the desk. |
| Inspector, 090–114 | Mirror `TargetResearchPlayer` identity/bio age, `TargetPlayerContract` year rows with inferred/observed provenance, `TargetAgeArc`, `TargetPersonalNotes`; native recorded contract, own `contract.quote` cut cost and `projections.read` floor/median/ceiling/games/availability | Lens-specific values/edge, inferred needs and extra drawers are unavailable. Native private notes/public bio/age arc have no suitable hosted read. Partner-player cut quote may be refused by owner scope. |
| Pocket, 115–127 | Drawer opens/closes, including Escape, with the mock's private-draft wording | Saved-version storage/recheck read does not exist. No fabricated saved count. |
| Negotiations, 128–150 | Mirror `TargetTrades` pending offers; native `TargetNativeTrades` (`trades.read`) viewer-scoped offers, assets, status and expiry | No dedicated round/thread/turn/reply read. No acceptance/send side effects. Existing offer reads are not presented as a negotiated history. |
| Bottom action bar, 151–155, 363–364 | Pull handles and current draft clear | Pocket save, send, private plan execution, bid commit totals and hosted draft queue actions are unavailable. |
| Plans, 156–195, 365 | Locked regions retained: list, ranked needs, event workspace, trade threads, pocket, market, draft, scenario cap and conflicts | D39 has no plan read. All these plan-specific values/actions remain unavailable. |
| Targets, 196–219 | Mirror `TargetPersonal(mine)` watch/player entries sorted by stored order and grouped by position; stars add/remove; lane up/down uses `TargetPersonalChange` action `swap` with `with`; drag into an existing matching lane | No hosted private-target store/read; native never substitutes local private lists. Need severity and estimated price are unavailable. |
| Pool, 196–212 | Mirror `TargetPool` player, position, age, minimum bid and engine text; native recorded free-agent pool plus `TargetNativeWaivers`; native projections hydrate positions in batches of at most 120 IDs | Native minimums/age/base model values lack the corresponding scoped pool read. Need/affordability filters are disabled rather than guessed. |
| Own bids, 220–254, 362–363 | Mirror `TargetBidPlans` owner-only plans in READY with their actual status; native `TargetNativeBids` uses **only `yourBid`/`youLead`**, own contract and identity in PLACED | Blind leadership is unknown, not “outbid”. Rival leaders, amounts, teams and counts are discarded. No aggregate committed/free budget is invented from contract prose. Compose/raise/place/confirmation interactions are not connected on this read-focused surface; existing workflows remain elsewhere. |
| Price model/recent wins, 206–208, 229–231, 255–263 | No rival input used | D35 winning-price/rivals/trend/probability model has no read. Recent-win model region remains unavailable. |
| Draft, 264–311 | Native `TargetNativeDraft` (`draft.read`) order, owners, traded origins, on-clock deadline, recorded picks and available players; real round/mine filters; mirror draft reports M-183/M-063 show recorded draft results on the order side | No invented tiers, ADP, simulated clock, mock-draft engine or hosted private queue. Mirror fantasy available-prospect board is unavailable; draft-result rows are not misrepresented as available prospects. |
| Scouting card, 312–353 | Native prospect click opens the v6 dialog; mirror generic private observations reuse D45 notes; focus/Escape handling; no saved-report claim for absent fields | D44 verdict/grade/rank/narratives/traits/game-context, comparisons and hosted queue have no matching read. No internal engine confidence is surfaced. |

## Unwired items

- D27 needs/surplus/pick equivalence; D28 posture/lens; D29 saved versions/pocket; D30 negotiation rounds;
  D32 tune-it; D35 bid market model; D39 plans; D43 hosted queue; D44 structured scouting reports.
- Competitive-window model, value-by-age histogram, private attention counts and customizable shortcuts.
- Native Best-N, hosted private targets/notes, native age curve/bio and scoped shelf base values.
- Mirror future cap-room values and mirror per-trade lineup delta.
- Inspector replacement/need explanation is still empty even when the desk has a native D33 result.
- Bid editing/placement/confirmation, aggregate committed/free bar and trade send are deliberately not
  inferred from old mock interactions. Current proposal previews and owner bid reads are wired; these
  remaining controls are disabled. This is residual frontend integration debt, not a backend absence claim.

## Settings read

No preset checks or literal league caps, roster limits, slot counts or contract lengths were added.

- Mirror settings adapter: roster size, IR size, taxi size, weekly starter total, offense/defense starter
  counts and positional min/max. The header uses the read starter total and Race's `starterN`.
- Native shared rule reader: `roster.limits`, `roster.reserveExcluded`, `ir.slots`, `taxi.slots`,
  `lineup.positions`, `lineup.groups` total/offense/defense bounds.
- Cap, room, contract expiration/years, calendar/deadlines, draft rounds/owners and projection scoring
  come from existing authoritative reads. The frontend does not recalculate their business rules.
- A test changes the starter setting from 8 to 13 and observes the rendered Best-N label changing.

## Go added

**None.** Engines, stores, models and services are unchanged.
`commands/native.gen.ts` was regenerated from the existing Go catalog to expose the already-built
`lineupdelta.read` and `projections.read` schemas to the normal hosted transport.

## Shared files changed

- `frontend/src/app/shell/TargetApp.tsx`: one lazy import and War Room route mount.
- `frontend/src/app/commissioner/HostedLeague.tsx`: lazy native War Room and server-workroom entry.
- `frontend/src/app/offers/api.ts`: existing private-list binding adapter for target `swap`.
- `frontend/src/app/commissioner/api.test.ts`: expected existing binding count 48 → 50; asserts both new reads.
- `frontend/src/app/commands/native.gen.ts`: generated catalog freshness, no manual schema edits.

No `HQ*.tsx`, `hq*.ts`, `hq.css` or `FranchiseHQ*` file changed. Existing shared HQ read/format/action
helpers are reused read-only. The handoff is updated separately from feature source.

Read-module paths were disambiguated for the existing case-insensitive build gate:
`warroomTrade.ts` → `warroomTradeReads.ts`, `warroomMarket.ts` → `warroomMarketReads.ts`,
`warroomInspector.ts` → `warroomInspectorReads.ts`; draft transport lives in `warroomDraftReads.ts`.

## Tests and build evidence

| Check | Result |
|---|---|
| War Room region + stylesheet tests | **24 PASS**, 2 files |
| Full `pnpm test` | **1,664 PASS / 1 FAIL**, 146 files; inherited HQ line-length gate only |
| `pnpm exec tsc --noEmit` | **PASS** |
| `make build VERSION=dev COMMIT=WR-wire BUILD_DATE=WR-wire`, Go 1.26.10 | **PASS**, packaged desktop binary; not launched |
| Frontend transport guard | **PASS** |
| Native generated catalog check | **PASS** |
| Bundle budget | **PASS**, 148,961 eager bytes / 321,818 limit |
| Static CSP + headless Firefox CSP probe | **PASS** |
| `git diff --check` | **PASS** |

Tests cover shelves/select/partner changes, native private preview (never execute), complete and incomplete
D33 results, correct benched wording, names for missing projections, real read-to-render fields, stars and
swap persistence, native private-read isolation, own/rival bid filtering, blind bid wording, calendar,
draft filters, empty plans/pocket, offer scope, dated-note persistence and slow-selection fencing.
The CSS test compares every mock rule/declaration against the scoped stylesheet, including overrides.
A deliberately restored validate-after-assignment research mutant failed its regression test; restoring
validate-before-publish passed. No mutant remains.

The inherited failing test is `commands/controls.test.ts` → “forbids lines longer than 110 characters”:
`FranchiseHQ.tsx:116`, `HQBoard.tsx:15,60,109,147`. These lines exist at base `a3ea876b` and are unchanged.
The gate was not weakened and the protected files were not edited. The full-suite requirement is therefore
**blocked**, not claimed green.

## Suspected issues / unverified behavior

- Existing native roster parsers omit some optional fields present in backend types (for example,
  `heldContractStatus` and compensatory/correction pick metadata). Such readings can be rejected; no
  backend or other-agent parser was changed here.
- D33 has no returned ledger revision: matching league/season/week/franchises is checked, but atomic
  agreement with a separately read preview/catalog across concurrent league changes is unproven.
- Wails emits existing unresolved-generic binding warnings during generation; packaging still exits 0.
- Native entry from the protected HQ screen requires returning to its server workroom, then choosing
  War Room. Direct HQ-to-War routing was not changed in another agent's files.
- Browser CSP testing is not a visual or live Wails feature gate. Exact final pixel/interaction acceptance
  on real MFL/native reads is **unverified**. No live profile, MFL move or production binary was opened.

## Debt and review handoff

Keep the complete 900-line mock CSS together: its responsibility is the locked v6 visual contract.
It is scoped to `wr-v6`; animation names are scoped too. No new design system is introduced.
Handwritten affected production source, including shared edited files, base `a3ea876b` → `3e7e19ed`:
993 → 2,959 physical lines; 36,926 → 190,933 Unicode code points including LF;
36,975 → 191,093 UTF-8 bytes. Tests/docs/generated catalogs are excluded. This is added UI, not a debloat claim.

Next acceptance step: disposable MFL and native profiles. Select a real partner, add/remove player and
pick assets, inspect preview legality, verify both D33 sides and missing-projection names, star/swap a
mirror target, open native own bids and confirm no rival disclosure, then check draft filters and
inspector notes. Compare all four workspaces/drawers against v6 at the intended desktop size. Do not
interpret the passing build or stylesheet-rule test as proof of that visual/live gate.
