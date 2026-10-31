# Terra data catalog — 2026-10-30

**INCOMPLETE.** Only the Franchise HQ mock and the origin/main files cited below were inspected. Rows are a partial inventory, not a screen-complete catalog. Sources marked unknown have not been investigated; unknown is not a claim that a new path is required. WR, LP, HM, CR, cross-screen entities and the new-path audit remain outstanding. Role variants not evident in the mock are unknown.

## HQ — Franchise HQ (partial)

| ID | Screen › Region › Element | Shows / does | Data shape | Source on origin/main (path:symbol) or NEW | Read/Write | Refresh | Role/state variants |
|---|---|---|---|---|---|---|---|
| HQ-001 | HQ › Rail › League name | Displays league name | string | unknown — not investigated | Read | per-load | unknown |
| HQ-002 | HQ › Rail › Week | Displays current week | integer | internal/app/mflref/contracts/lineup_api.go:LineupReading | Read | on-event | offseason unknown |
| HQ-003 | HQ › Rail › Live games | Displays count of live games | integer | unknown — not investigated | Read | live (stat feed) | no live games unknown |
| HQ-004 | HQ › Rail › Lineup lock notice | Displays lock day and time | timestamp | frontend/src/app/clock/useLeagueClock.ts:useLeagueClock | Read | on-event | unknown deadline possible |
| HQ-005 | HQ › Rail › Shortcuts | Displays ordered pinned pages | list<{page_id:string,label:string,order:int}> | unknown — not investigated | Read | per-load | user-specific; empty unknown |
| HQ-006 | HQ › Rail › Pin shortcut | Adds or removes page from shortcuts | {page_id:string,pinned:bool} | unknown — not investigated | Write | on-event | customize mode |
| HQ-007 | HQ › Header › Team name | Displays selected franchise name | {franchise_id:string,name:string} | frontend/src/app/data/contract.ts:Franchise | Read | per-load | missing name possible |
| HQ-008 | HQ › Header › Team record | Displays wins and losses | {wins:int,losses:int} | unknown — not investigated | Read | on-event | offseason unknown |
| HQ-009 | HQ › Header › Division standing | Displays division and rank | {division:string,rank:int} | unknown — not investigated | Read | on-event | tied and offseason unknown |
| HQ-010 | HQ › Header › Active roster count | Displays used and maximum active slots | {used:int,max:int} | unknown — not investigated | Read | on-event | full and over-limit unknown |
| HQ-011 | HQ › Header › Cap season | Labels cap season | integer | frontend/src/app/data/contract.ts:League | Read | per-load | offseason unknown |
| HQ-012 | HQ › Header › Cap room | Displays available cap | money_cents | internal/app/mflref/contracts/contracts_api.go:API.TargetContracts | Read | on-event | missing cap or franchise can fail |
| HQ-013 | HQ › Header › Cap maximum | Displays cap limit | money_cents | internal/app/mflref/contracts/contracts_api.go:API.TargetContracts | Read | on-event | configured cap can be absent |
| HQ-014 | HQ › Header › Cap committed | Displays committed salary and bar segment | money_cents | PARTIAL: internal/app/mflref/contracts/contracts_api.go:API.TargetContracts lacks mock-specific separation of committed and dead cap in its current cap-used figure | Read | on-event | unknown |
| HQ-015 | HQ › Header › Dead cap | Displays dead cap and bar segment | money_cents | PARTIAL: internal/app/mflref/contracts/contracts_api.go:API.TargetContracts provides signed salary adjustments but does not interpret descriptions as dead cap | Read | on-event | unknown |
| HQ-016 | HQ › Header › Lineup check | Shows counts of empty, bye and questionable starters | {empty:int,bye:int,questionable:int} | PARTIAL: internal/app/mflref/contracts/lineup_api.go:LineupReading lacks the mock's bye and questionable counts | Read | on-event | full, empty, bye, questionable; unreadable rules |
| HQ-017 | HQ › Header › Lock countdown | Displays time remaining until lineup lock | {deadline:timestamp,remaining_seconds:int} | frontend/src/app/clock/useLeagueClock.ts:useLeagueClock | Read | on-event | unknown deadline possible |
| HQ-018 | HQ › Lineup › Starter count | Displays number of starting slots | integer | internal/app/mflref/contracts/lineup_api.go:LineupReading | Read | on-event | zero if rules unreadable |
| HQ-019 | HQ › Lineup › Slot designation | Displays each slot position and whether empty | {slot:string,player_id:string|null} | PARTIAL: internal/app/mflref/contracts/lineup_api.go:LineupReading lacks the mock's explicit per-slot rendering contract | Read | on-event | empty slot |
| HQ-020 | HQ › Lineup › Starter identity | Shows starter position, name and NFL team | {player_id:string,position:string,name:string,team:string} | frontend/src/app/cards/playerModel.ts:playerCardModel | Read | on-event | unavailable player data |
| HQ-021 | HQ › Lineup › Starter opponent | Shows game opponent or bye | {opponent:string|null,is_bye:bool} | unknown — not investigated | Read | on-event | bye |
| HQ-022 | HQ › Lineup › Starter projection | Shows per-player projected points | decimal | unknown — not investigated | Read | on-event | projection unavailable unknown |
| HQ-023 | HQ › Lineup › Starter status | Shows LIVE, Q, BYE or OUT badge | enum{live,questionable,bye,out,none} | unknown — not investigated | Read | live (stat feed) | live, questionable, bye, out |
| HQ-024 | HQ › Lineup › Form chart | Shows five weekly points bars and projection marker | {weeks:list<{week:int,points:decimal|null}>,projection:decimal} | unknown — not investigated | Read | on-event | bye renders hatched bar |
| HQ-025 | HQ › Lineup › Form bar hover | Shows week and weekly points or bye | {week:int,points:decimal|null,is_bye:bool} | unknown — not investigated | Read | on-event | bye |
| HQ-026 | HQ › Lineup › Form average | Displays average of non-bye chart points | decimal | unknown — not investigated | Read | on-event | no scores unknown |
| HQ-027 | HQ › Lineup › Offence projection total | Sums effective starter projections for offence | decimal | unknown — not investigated | Read | on-event | bye and out treated as zero in mock |
| HQ-028 | HQ › Lineup › Defence projection total | Sums effective starter projections for defence | decimal | unknown — not investigated | Read | on-event | empty, bye and out treated as zero in mock |
| HQ-029 | HQ › Lineup › Total projection | Sums effective starter projections | decimal | unknown — not investigated | Read | on-event | empty, bye and out treated as zero in mock |
| HQ-030 | HQ › Lineup › Bench count | Displays nonstarting roster count | integer | internal/app/mflref/contracts/lineup_api.go:LineupReading | Read | on-event | empty bench |
| HQ-031 | HQ › Lineup › Bench position filters | Filters bench by ALL, QB, RB, WR, TE, OL, K, DT, DE, LB, CB or S | enum{ALL,QB,RB,WR,TE,OL,K,DT,DE,LB,CB,S} | PARTIAL: frontend/src/app/data/contract.ts:POSITIONS lacks OL | Read/Write | on-event | selected filter; no matches unknown |
| HQ-032 | HQ › Lineup › Bench player identity | Shows each bench player's position, name and NFL team | {player_id:string,position:string,name:string,team:string} | frontend/src/app/cards/playerModel.ts:playerCardModel | Read | on-event | unavailable player data |
| HQ-033 | HQ › Lineup › Bench opponent | Shows bench player's matchup or bye | {opponent:string|null,is_bye:bool} | unknown — not investigated | Read | on-event | bye |
| HQ-034 | HQ › Lineup › Bench projection | Displays and sorts by projected points | decimal | unknown — not investigated | Read | on-event | missing projection unknown |
| HQ-035 | HQ › Lineup › Bench form | Shows five scores, average, projection marker and hover values | {weeks:list<{week:int,points:decimal|null}>,projection:decimal,average:decimal} | unknown — not investigated | Read | on-event | bye |
| HQ-036 | HQ › Lineup › Move starter or bench player | Drag or select player and eligible slot; displaced starter may cascade | {player_id:string,slot:string,eligible:bool} | PARTIAL: frontend/src/app/commands/lineupEdit.ts:runLineupAction supports starter toggles but does not implement the mock's slot cascade | Read/Write | on-event | ineligible slot refused; empty slot; bench autofill |
| HQ-037 | HQ › Lineup › Change message | Displays number and recent descriptions of unsaved swaps | {count:int,recent:list<string>} | unknown — not investigated | Read | on-event | no changes |
| HQ-038 | HQ › Lineup › Discard changes | Reloads mock to discard edits | {franchise_id:string,baseline_player_ids:list<string>} | frontend/src/app/commands/lineupEdit.ts:runLineupAction | Write | on-event | unsaved edits |
| HQ-039 | HQ › Lineup › Check and save plan | Checks and drafts lineup plan | {franchise_id:string,starter_ids:list<string>} | frontend/src/app/commands/lineupEdit.ts:runLineupAction | Read/Write | on-event | missing lineup, failed freshness or unknown week prevents edit |
| HQ-040 | HQ › Lineup › Open MFL | Hands off ready lineup plan to MFL | {correlation_id:string,state:string} | frontend/src/app/commands/lineupEdit.ts:runLineupAction | Read/Write | on-event | handoff only when ready |
| HQ-041 | HQ › Matchup › Opponent | Displays owner and opposing franchise | {owner:string,franchise:string} | unknown — not investigated | Read | per-load | no matchup unknown |
| HQ-042 | HQ › Matchup › Our projection | Shows sum of effective starter projections | decimal | unknown — not investigated | Read | on-event | lineup edits |
| HQ-043 | HQ › Matchup › Opponent projection | Shows opposing projected total | decimal | unknown — not investigated | Read | on-event | unavailable unknown |
| HQ-044 | HQ › Matchup › Win probability | Shows percent and proportional bar | decimal_percent | unknown — not investigated | Read | live (stat feed) | unavailable unknown |
| HQ-045 | HQ › Matchup › Live player count | Shows number of user's players currently live | integer | unknown — not investigated | Read | live (stat feed) | no live players unknown |
| HQ-046 | HQ › Matchup › Position comparisons | Shows position, player names, points and point difference | list<{slot:string,ours:string,ours_points:decimal,theirs:string,theirs_points:decimal,delta:decimal}> | unknown — not investigated | Read | on-event | empty slot and bye shown |
| HQ-047 | HQ › Contracts › Contract count | Displays number of contracts | integer | internal/app/mflref/contracts/contracts_api.go:API.TargetContracts | Read | on-event | empty roster |
| HQ-048 | HQ › Contracts › Player identity | Displays each player's position, name and NFL team | {player_id:string,position:string,name:string,team:string} | internal/app/mflref/contracts/contracts_api.go:API.TargetContracts | Read | on-event | unknown names display ID |
| HQ-049 | HQ › Contracts › Annual salary cells | Shows salary by year and final-year marker | list<{year:int,salary_cents:int,is_final:bool}> | PARTIAL: internal/app/mflref/contracts/contracts_api.go:API.TargetContracts carries current annual salary forward; MFL does not report separate future-year salaries | Read | on-event | no contract in year |
| HQ-050 | HQ › Contracts › Free agency badge hover | Shows RFA or UFA on final contract year | enum{RFA,UFA} | frontend/src/app/data/contract.ts:CONTRACT_STATUSES | Read | on-event | contract statuses beyond RFA/UFA exist |
| HQ-051 | HQ › Contracts › Dead if cut | Shows calculated charge for each player | money_cents | PARTIAL: internal/app/mflref/contracts/contracts_api.go:API.TargetContracts has no calculated dead cap in contract table | Read | on-event | unknown |
| HQ-052 | HQ › Contracts › Player value | Displays dynasty value | number | unknown — not investigated | Read | on-event | missing value unknown |
| HQ-053 | HQ › Contracts › Cost per value | Displays salary divided by player value | decimal | unknown — not investigated | Read | on-event | zero or missing value unknown |
| HQ-054 | HQ › Contracts › Future cap runway | Shows year, committed and dead segments, and remaining room | list<{year:int,committed_cents:int,dead_cents:int,room_cents:int}> | PARTIAL: internal/app/mflref/contracts/contracts_api.go:API.TargetContracts does not project future adjustments or dead-cap charges | Read | on-event | unknown |
| HQ-055 | HQ › Release drawer › Player | Displays selected player and free-agency end status | {player_id:string,name:string,expiration_year:int,status:string} | internal/app/mflref/contracts/contracts_api.go:API.TargetPlayerContract | Read | on-event | missing player contract |
| HQ-056 | HQ › Release drawer › Release charge | Displays current-year dead cap | money_cents | internal/app/mflref/contracts/release.go:ReleasePlan | Read | on-event | cut vs buyout |
| HQ-057 | HQ › Release drawer › Prepare release | Creates release plan for selected what-if move | {franchise_id:string,move_id:string,confirmed:bool} | internal/app/mflref/contracts/release.go:Release | Read/Write | on-event | draft requires move; handoff requires ID |
| HQ-058 | HQ › What-if drawer › Modeled moves | Lists modelled cuts or buyouts and costs | list<{id:string,player_id:string,label:string,charge_cents:int}> | PARTIAL: internal/app/mflref/contracts/whatif.go:WhatIf supports cuts and buyouts but not all mock taxi, IR, sign, extension and trade options | Read/Write | on-event | stale; empty; eligibility unknown |
| HQ-059 | HQ › What-if drawer › Before/after figures | Shows modelled cap figures | list<{label:string,before:string,after:string,change:string}> | internal/app/mflref/contracts/whatif.go:WhatIfFigure | Read | on-event | stale or empty model |
| HQ-060 | HQ › Inspector › Player identity | Shows position, name, team and contract summary | {player_id:string,name:string,position:string,team:string,salary_cents:int,years:int,status:string} | frontend/src/app/cards/playerModel.ts:playerCardModel | Read | on-event | fields may be unavailable |
| HQ-061 | HQ › Inspector › Data freshness | Shows source and time since fetch | {source:string,fetched_at:timestamp,state:string} | frontend/src/app/cards/playerModel.ts:provenanceSlot | Read | on-event | live, stale, failed, fixture |

## Shared entities

Partial HQ-only evidence; cross-screen usage not inspected.

- Franchise/team: HQ-007–HQ-010, HQ-041.
- Player: HQ-019–HQ-026, HQ-032–HQ-036, HQ-046, HQ-048–HQ-053, HQ-055–HQ-058, HQ-060.
- Contract: HQ-047–HQ-051, HQ-055–HQ-056, HQ-060.
- Lineup: HQ-016–HQ-019, HQ-027–HQ-030, HQ-036–HQ-040.
- Move/plan: HQ-036–HQ-040, HQ-057–HQ-059.

## New data paths

Not audited. PARTIAL rows identified so far:

- Cap and contract accounting: HQ-014, HQ-015, HQ-049, HQ-051, HQ-054.
- Lineup and position handling: HQ-016, HQ-019, HQ-031, HQ-036.
- What-if actions: HQ-058.

No NEW classification is asserted: uninvestigated rows remain unknown.

## Doubts

1. Catalog is incomplete: HQ still lacks many elements in contracts, release, roster drawers, matchup, moves, inspector and navigation; the other four mocks were not read.
2. Uninvestigated source rows may already be served by origin/main; no NEW determination is justified yet.
3. Role/permission, empty, locked and offseason variants require code and contextual document inspection.
4. The brief's concrete shapes are target shapes inferred from mock displays, not verified API wire contracts for every row.
5. Cross-screen shared entities and backend new-path groupings remain incomplete.
