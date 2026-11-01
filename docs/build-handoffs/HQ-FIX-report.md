# HQ-FIX implementation report — 2026-10-31

**Partial delivery, not locked-mock acceptance.** Local `build/HQ-wire`; no pull/push,
MFL submission, deployment, production database access or production app execution.
The locked mock is unchanged. No dependencies or bound methods added.

## Commits and letter status

A–C were inherited: `b8334d46`, `cebe0f69`, `207f44f3` (plus the research golden follow-up).
Their board/header tests were exercised again.

| Letter | Status | Implementation / remaining limit |
|---|---|---|
| A | Partial | Inline taxi/IR, capacities and held return weeks render. Rookie year 1 is known; complete veteran NFL-year/taxi-time metadata still needs a read over existing model/eligibility inputs. Missing compact metadata displays an em dash, with an unavailable title. |
| B | Partial | MFL division names and native human alignment values resolve. Native settings containing only ledger IDs do not provide a human name; the ID is not displayed. |
| C | Partial | Five finished-week rows/chart, bye hatch, projection marker and average work for MFL. Native hosted form has no wired read. |
| D | Partial — `7773ad0e` | Compatible target checks, starter swaps, bounded displaced-player cascade, weakest-slot autoplacement and projection-backed autofill. Membership changes use existing MFL draft/check/save/handoff. Native uses existing `createHostedRoom`, revision/evidence-fenced preview and explicit hold confirmation. Discard restores membership. **Named slot assignments are only presentation state, not persisted:** neither lineup command carries slots. Pure starter swaps therefore do not create a membership plan. Unknown projections are not invented to choose autofill. |
| E | Partial — `8ef09dd2` | Actual scores, projections, position deltas and an existing-model win probability. The held Now preview calls `powerrankings.WinProbability` only for a complete projected pair. No new prediction engine or binding. Native matchup reads remain unwired. |
| F | Partial — `d73f4d9e` | Mounted MFL HQ owns a scoped existing live poll; visibility pause/cleanup retained. Rail live-game count, LIVE badge and live-starter count use held game states. Q/BYE/OUT retain existing facts. Individual on-field participation is **not** inferred from possession. No B3 branch was merged. Native hosted live evidence remains unwired. |
| G | Partial — `0e4736cc` | Existing MFL inspector notes, contract, usage, arc and actions retained. Native Bench changes draft membership; To IR opens real hosted reserve preview. Native schedules and selected cut prices render. Native notes/watch, age arc/usage/dynasty/PPG and add-to-trade remain unwired; local MFL private lists are never borrowed. |
| H | Partial — `aa87e66e` | Native reserve activate/promote uses hosted preview/confirmation; reserve verdict notes are wired. Native What-if selects a player and reads its real cut quote with before/after room and dead cost. Annual native schedules use the existing contract pricer schedule via a thin cap-read extension. Existing MFL contract/move/stacked-what-if paths retained. Hosted stacked simulation/local plan persistence, complete season ledger, classified dead-player table and modeled decision suggestions remain open. |
| I | Implemented — `d50265f8` | MFL export display name carried through held config into existing settings report and rail. Lock format is `Locks Sun 12:00`, from the actual clock instant. Old persisted configs require the normal MFL refresh to acquire the newly retained name. Includes regenerated schedule models from H. |

A final verification correction formats the affected source, restricts skill FLEX to RB/WR/TE,
and labels surplus skill/defensive slots FLEX/D·FLX instead of pinning them to an incumbent's position.
The existing probability bar and live-count placement now use the real held model/read values.
No engine/store behavior was rewritten.

## Go added

- `internal/app/now/nfl_reading.go`: thin existing-model probability in the existing preview summary.
- `internal/nativecheck/contract_price.go`: `ExistingContractSchedule` delegates to the existing
  pricer's annual schedule; does not price or record an action.
- `internal/commissioner/native_cap.go`: additive per-player annual schedules in existing cap read.
- `internal/ingestion/league/{types.go,league.go}`: retain the export's display name.
- `internal/app/control/reading.go`: expose that name in the existing settings report.
- Tests in Now, native runway and Control; regenerated Wails models. No new bound method, endpoint,
  package, migration, dependency, authority or command-catalog/parity-count change.

## Evidence and settings

Logs and captures are under `.tmp/hq/`; they are ignored, not committed.
- Full frontend: **1,652 PASS / 148 files** (`frontend-verified.log`); TypeScript PASS.
- Required Go composition/serverhttp/commandcatalog race suites: **3 packages PASS**, using the
  project's 60-minute timeout; composition took 784.926s (`go-verified.log`).
- Changed-feature race suites also PASS: Now; nativecheck; commissioner and commissioner/market;
  app/commissioner and app/commissioner/native; Control; league ingestion (8 additional packages).
- `make build VERSION=dev COMMIT=HQ-FIX BUILD_DATE=HQ-FIX`: PASS (`build-verified.log`).
- Transport boundary, bundle self-test and static/browser CSP: PASS. Eager graph **148,925 / 321,818 B**.
- Four stale cap wire fixtures were regenerated. JSON review proves their only changed paths are
  additive `contracts` fields; tests/assertions were not weakened.
- The first composition run hit Go's default 10-minute limit; rerun uses Makefile's 60-minute limit.
  A load-sensitive pre-existing rule-screen test timed out while Go was busy; isolated and complete
  frontend reruns passed without changing the test or its timeout.
- Full `make verify`, authenticated Wails/native round trips and exact pixel acceptance are not claimed.

Region tests cover reads → rendered values, compatible/rejected placement and preserved slots,
swaps/cascade/autofill, native membership/revision/football arguments, reserve preview arguments,
promote callbacks, discard without execution, quote selection isolation, notes reread, and clock/name.
Settings congruence changes position allowance, lineup groups, reserve capacities, cap amounts,
and the schedule's league season. Existing settings remain the owners: MFL position min/max,
starter/IOP/IDP counts, roster/IR/taxi sizes and cap; native `lineup.positions`, `lineup.groups`,
`roster.limits`, `roster.reserveExcluded`, `ir.slots`, `taxi.slots`, contract horizon and resolved season.
Annual salary/expiration come from recorded contracts. Win probability delegates its model constants
rather than introducing screen constants.

## Browser proof and remaining visual differences

Firefox headless captures: `app-1920x1080.png`, `app-2550x1352.png`, and matching `mock-*` files.
The real app route uses the repository's recorded MFL `snapshot.json` for Arizona (0025), with a
read-only recorded-binding bridge for available supplemental reads. The lineup golden's normalized
`INSTANT` is rehydrated with its test assertion timestamp. This is **recorded-read rendering**, not
an authenticated Wails/live-league acceptance test. Missing/mismatched fixture reads throw rather
than using the mock's sample data. The unrelated Buffalo-view Now/Race fixtures are not substituted
for Arizona. The temporary harness can never execute a backend command.

Region comparison at both sizes:
- Rail/work/inspector widths match the mock: 224px rail, 316px inspector, and 1380/2010px work.
  The real plate rounds to 92px versus 91px and tray to 51px versus 50px; the board is about 3px
  shorter at both sizes. Those remaining geometry differences are not called pixel parity. Actual league, week, roster counts,
  player names and capacities differ appropriately from mock examples. Longer unavailable text
  expands the rail header and plate's auto columns; the cap strip starts farther right.
- Board has the league's 21 starters/17 bench, not mock 19/22. Real settings produce different
  positional minima and surplus FLEX rows; no sample OL or invented gap is inserted.
- Unavailable projections/form/opponents/live/scores in this recorded bridge remain visibly absent;
  populated-read rendering is separately tested. No win-probability bar/sample percentage is faked.
- The extra real reserve rows occupy bench height; at 1080 some IR content requires its pull tab.
  Both pull tabs remain reachable. Complete tenure/return metadata is not in these captures.
- Inspector lacks matching captured selected-player model/private/contract reads; empty/error states
  appear instead of mock PPG/value/arc/notes. Real contract year count is retained. Current team-action
  admission is unavailable in the bridge, so guarded controls are disabled, not force-enabled.
- Tray contains real save/error/status wording, not the mock's simulated change history. Contracts,
  move lanes, quote drawers and native confirmation are tested, but not pixel-approved click-throughs.
- Full locked-mock acceptance, saved named-slot reload, actual hosted/mirror round-trip click-through,
  authenticated probability refresh and individual on-field indicators remain unverified/open.

## Missing-read searches — absence versus unwired

Exact searches and bounded output: `.tmp/hq/search-evidence.txt`. **A source hit is not absence.**
Existing but unwired values below are delivery gaps, not permitted “data does not exist” claims.

1. Hosted private notes/watch, scoped age arc/usage and saved stacked plans:
   `rg -n -i 'note|watch|age.?arc|usage|what.?if|lineup.*plan' internal/commandcatalog/workflow_commands.go internal/app/commissioner internal/commissioner internal/model internal/powerrankings internal/modelrun internal/m2service`
   No equivalent hosted private/scoped read was wired. D45 `TargetPersonalNotes`/change require
   `personalSnapshot(mine)` from `OffersSnapshotNames` (local mirror franchise identity). D26
   `TargetAgeArc` uses the local season/model source, not a hosted league scope. D13
   `TargetPlayerUsage` captures the local research season/weekly history/matchup. Their existence
   is not authority to label local results as another hosted league's results.
2. Named lineup assignments:
   `rg -n 'starters|Slots|slots|Assignments' internal/app/mflref/contracts/lineup* internal/mflref/lineup internal/commissioner/lineup*`
   Existing commands persist starter IDs, not named slot assignments. This remains an unsupported
   command shape, not implemented persistence.
3. Individual current on-field state:
   `rg -n -i 'on.?field|participation|snap|drive|possession' internal/app/now internal/ingestion/livegame`
   Possession/play participants are held, but are not an individual participation census.
4. Veteran NFL year/taxi time:
   `rg -n 'Experience\(|Taxi.*Years|MaximumYears|EntryYear' internal/model/data.go internal/app/market/age_arc.go internal/app/research internal/commissioner/native_reserve* internal/nativecheck/reserve*`
   `model.Player.Experience` exists. **Unwired**, not missing in the repo; no substitute years invented.
5. Suggestions/decision ordering/ledger:
   `rg -n -i 'recommend|decision.*due|replacement|season.*ledger' internal/app/* internal/commissioner internal/model internal/powerrankings internal/modelrun internal/m2service`
   No complete HQ decision/replacement recommendation model or complete season-membership move
   reading was established. Other feature models/ledger records exist; a complete HQ ledger needs
   a properly scoped read, not treating all returned history as one season.
6. Explicit existing reads traced:
   `rg -n 'TargetPersonalNotes|personalSnapshot|OffersSnapshotNames|TargetAgeArc|AgeArcSource|TargetPlayerUsage|UsageSource' internal/app/offers/personal_api.go internal/app/core/host_offers.go internal/app/market/api.go internal/composition/asset_board.go internal/app/research/api.go`

## Suspected issues and debt

- Imported cap classification/rollover/release issues remain those in `docs/architecture/cap-runway.md`;
  none was fixed or papered over. Signed MFL adjustments are not relabeled dead money.
- Settings IOP is a skill-player cap excluding PK, while the visual offence region includes PK;
  the recorded real lineup shows nine offence-region players beside the read's eight. The screen
  has not silently rewritten the backend's definition to match the mock.
- MFL projection names are not stable IDs; unique exact-name joins remain deliberately conservative.
- Native contract end markers, per-row dynasty/value ratios and classified dead-player rows remain open.
- Win probability describes the held saved-lineup pair, not a newly recomputed draft forecast. Opponent
  projection completeness is still conservatively gated by the existing report-wide missing count.
- Native current-contract schedule is the existing flat salary schedule. No reconstructed historical
  restructure schedule is invented.
- Native lineup save is a server change after explicit confirmation, **not** a persisted local plan.
- Named slots, richer native reads/actions, complete region parity and production-runtime proof are
  still delivery debt. This report must not be treated as “everything functional.”
