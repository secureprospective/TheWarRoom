# HQ-WIRE implementation report

Local branch `build/HQ-wire`; no pull, push, merge, deployment or production app execution.
The locked v10 mock is unchanged. Go added: **none**. Dependencies added: **none**.

## Commits

1. `35cfd303` — shell and franchise plate.
2. `a421341e` — starter/bench board and held player context.
3. `48ffff78` — drawers, notes, moves, native reads and runway.
4. The final verification commit includes command/bundle gate preservation, mapping and interaction
   tests, native cut quotes, read-failure handling, this report and the handoff.

## Region coverage

| Mock region | Wired reads / honest unavailable portion |
|---|---|
| Rail and tabs | HQ navigation, read week, clock lock; native league name from rules. MFL league name and live-game count: Not available yet. Customize/pin persistence unavailable. |
| Franchise plate | TargetSnapshot / native assets; TargetRace standings / TargetNativeStandings, settings-derived active allowance. |
| Cap strip | MFL snapshot + TargetContracts cap/room/limit; native TargetNativeCapCheck.runway committed/dead/room/limit. MFL committed/dead split unavailable; adjustments are not assumed dead money. |
| Lineup check and countdown | Existing TargetLineup, draft check, TargetClock; native lineup verdict and read kickoff times. |
| Starters and bench | Existing LineupRoster/LineupEditor/lineups and native lineup read; settings-derived rows and counts. M-046 projections, D13 opponent/usage, M-154 byes, research injury status and finished-week Form. Missing projections are not zero. |
| Totals / this-week matchup | Complete player projection sums; TargetNow X-03 names/entities, M-018 complete opponent totals, actual player projection deltas. Win probability and on-field indicators unavailable. |
| Contracts table | TargetContracts and TargetPlayerContract, annual inference titles, RFA/UFA markers, asset-board dynasty value and salary/value. Native current contracts and selected-player cut quote; individual future native annual schedules unavailable. |
| Cap runway / dead / decisions | Native cap runway and current dead total; partial MFL future commitments retain the existing inference. MFL future room/dead, itemized classified dead charges and decision/deadline recommendations unavailable. |
| Player inspector | Selected-player identity and contract; asset board; D13 usage/matchup; D26 TargetAgeArc; D45 TargetPersonalNotes with add/remove/reread. Native selected cut cost uses contract.quote. No hosted counterpart for local personal/model/usage reads is substituted. |
| Player actions | Existing MFL bench draft, eligible IR plan, private Watch and Add to trade (existing local builder, not proposal/submission). Native editing/planning actions remain disabled on this HQ surface; existing hosted workflows remain reachable. |
| IR / taxi | Roster buckets, salary, capacities, MFL eligibility candidates, D15 returnSeason/returnWeek/reason, derived weeks until return; native reserve verdict notes and counted IR pay. Taxi value uses the asset board. Typed NFL tenure/taxi time-left and several native eligibility/injury fields unavailable. |
| What-if / release | Existing MFL what-if session, backend-authored before/after figures and local release-plan hold. Native read-only cut quote supplies current/next dead cost and room after. Native stacking/persisted plans unavailable. |
| My moves | TargetMoves plus existing bid/release/contract plans; TargetNativeMoves, including native needs_you. Four lanes and queue counts; unavailable rather than guessed totals on partial read failure. Season ledger unavailable: neither consumed move shape proves complete season membership. |

## Unwired items

- MFL league display name; live-game count/poll lifecycle; live on-field player state and win probability.
- Exact saved slot assignments, compatible-target-only drag placement and mock cascade/autofill.
  The existing membership editor is reused, not replaced by a second lineup engine.
- Five-week form window: the existing player research Form read exposes only three finished weeks.
- MFL committed/dead classification, complete future cap-room/dead rows and classified dead-player ledger.
- Native individual future annual salary schedule; quotes are loaded on player selection, not
  speculatively for every row. Other players' unrequested cut quotes remain unavailable until selected.
- Typed NFL year / taxi time left; native active-roster reserve-entry candidates and hosted injury returns.
- Native hosted private notes/watch/age arc/usage, saved local plans and stacked what-if. No local
  MFL fallback is used for a native hosted franchise.
- Complete season transaction ledger; native move records have no season field and include historical
  records. Decision ordering/recommendations, replacement suggestions, release alternatives,
  custom shortcuts and add-drawer preferences are not invented.

These are explicit gaps, not a claim that no implementation exists anywhere in the fleet.

## Evidence

- `pnpm exec tsc --noEmit`: PASS.
- Full `pnpm test`: **1,638 PASS, 144 files** on the final production source.
- `make frontend-transport`: PASS.
- `GOTOOLCHAIN=go1.26.10 make build VERSION=dev COMMIT=HQ-WIRE BUILD_DATE=HQ-WIRE`: PASS.
- Bundle self-test: PASS; **148,925 eager bytes / 321,818**. Deliberate budget, fixture, rail and
  commissioner leaks rejected; lazy rail/form checks pass through the real dependency graph.
- Static CSP and headless Firefox CSP probes: PASS; inline scripts, event handlers, foreign scripts
  and eval rejected. Wails generic-model warnings remain nonfatal.
- Headless fixture-only board/cap inspection found and confirmed the fix for overlapping empty
  numeric cells. This is not a live Wails click-through, pixel-diff approval or real-league proof.
- No Go race suite/full make verify claimed: this slice changes no Go; backend compilation passed.
- Logs/captures are local under `.tmp/hq/` (ignored).

Settings consumed and state/failure ownership: `docs/architecture/hq-wire.md`.
The settings tests change roster, IR/taxi capacities, starter groups and cap values and observe changed
view output. Missing read values stay missing instead of taking the mock's sample values.

## Suspected issues / debt

- Native standings fixtures expose division ledger identifiers instead of human division names.
  Reported; no backend change.
- Existing cap inconsistencies remain exactly as documented in `docs/architecture/cap-runway.md`:
  release penalties charged in the cut year, rollover charge re-dating, unclassified imported
  adjustments and expired/pending-contract differences between older current-cap and runway reads.
- MFL projection reports have name-only identities; duplicate names deliberately remain unavailable.
- The root snapshot is an existing startup reading; full roster refresh and hosted server push/poll
  lifecycle are not redesigned here. Explicit hosted refresh remains in the existing workflows.
- Full locked-interaction parity and a disposable-profile Wails click-through are still required.
  This report does not label the remaining behavior gaps complete.

Acceptance check: open HQ for a mirror and a native league on disposable profiles; compare actual
names/counts/cap to their reads, change starter/squad/cap settings and reopen/refresh, then verify notes,
quote selection and move lanes without submitting anything to MFL.

Handwritten affected production footprint versus `329daea5`: 33 files, 1,871 → 3,768 physical lines; 79,744 → 208,221 Unicode code points including LF. Tests/docs/generated files excluded; no reduction claimed.
