# Ring 1 gap closure · 2026-10-07

Evidence tags cover the immediately preceding paragraph, list or table. Patterns substitute the
discovered host, loaded year, league ID and franchise ID; numeric dump filenames are not MFL
option numbers. Inference IDs are stable report references.

Path aliases: `run:` is `~/fleet/runs/warroom-ring1-2026-10-07/` on the Beelink (the MFL
exports and owner-page dumps, outside git); `brief:p0a` is
`~/fleet/briefs/warroom-ring1-p0a-gaps.md`. §8 records what Claude settled after this draft;
where §8 and an earlier inference disagree, §8 wins.

[seen: docs/ui/endpoint-registry.csv · M-027–M-034]

## §0 Rulings · Christopher, 2026-10-07

1. MFL login: an MFL API key stored in the OS keyring, never in a file or the database.

2. The season phase is automated (derived), not manual.

3. Waivers run on ProBoards, not MFL. TheWarRoom builds its own waiver code independently; that is
the direction.

4. DOT review applies to trades only, for now.

5. Windows and deadlines are in the computer's local time zone.

6. MFL's settings are the IR and taxi rules for now. Don't over-engineer: after MFL, the app reads
official NFL feeds.

7. Victory Points: the 2026 league export has no Victory Points setting (`standingsSort` is
`PCT,H2H,DIVPCT,CONFPCT,ALL_PLAY_PCT,PTS,PWR`), so they are not used.

8. Trade flow (confirmed later the same day): owners agree on ProBoards; one proposes on MFL
and the other accepts there; the DOT votes (three approvals); the commissioner approves on MFL
and the trade executes. The trade desk hands off propose, accept, reject and revoke to MFL's
trade page (O=05); DOT review is the stage between accepted and Landed; Landed is the public
`transactions` TRADE row.

[seen: brief:p0a
  Christopher's rulings (verbatim words; wrapped to 100 columns)]

The actual standingsSort string includes a trailing comma:
PCT,H2H,DIVPCT,CONFPCT,ALL_PLAY_PCT,PTS,PWR,. This does not add a Victory Points setting.

[seen: run:data/mfl-exports/league.json · league.standingsSort]

## §1 Act targets

Ready is app validation, not acceptance. A full matching observation is required for Landed;
partial evidence is Not verified. Unmapped targets stay Draft/Blocked. DOTReview is distinct from
Landed.

[seen: docs/ui/Target_UI_Spec_2026-10.md · §6]

Target is {Kind TargetKind, URL string}; mapped URLs require HTTPS. Existing Spec/ExpectedChange
and validateSpec require one subject player and roster status. AwaitDOT currently supports
trade.propose only; Observe reads roster freshness after hand-off. Trade and lineup evidence need
intent-specific shapes, not invented roster status values.

[seen: internal/envelope/envelope.go · Target, ExpectedChange, validateSpec, Await]

[seen: internal/envelope/ir.go · Observe, handedAt, IRPredicate]

The roadmap requires all seven Acts and export confirmation; the registry currently calls all six
relevant rows spec. This document recommends statuses only; it does not edit the registry.

[seen: docs/build-handoffs/UI_Target_Roadmap_2026-10.md · Ring 1]

[seen: docs/ui/endpoint-registry.csv · M-027, M-028, M-029, M-032, M-033, M-034]

### lineup.set · M-027

Open https://{host}/{year}/lineup?L={league}&FRANCHISE={franchise}. Registry O=02 is the legacy
navigation link; the captured working form is /lineup, not options?O=02. D1: use the captured
/lineup page; equivalence of the legacy URL is not proved by a link alone. Explicit week link:
https://{host}/{year}/lineup?L={league}&WEEK={week}&F={franchise}.

Captured URL: https://www47.myfantasyleague.com/2026/lineup?L=14432&FRANCHISE=0025. POST action
https://{host}/{year}/lineup. Hidden FRANCHISE=0025 selects the owner; LEAGUE_ID=14432, WEEK=4,
FORM=1 scope the submission. lineup_expires=1791172964 is present but its meaning is undocumented.
INITPROJSRC=mfl and PROJSRC select projection source; SUBMIT says Submit Partial Lineup.

Move fields repeat position+franchise names: QB0025, RB0025, WR0025, TE0025, PK0025, DT0025,
DE0025, LB0025, CB0025, S0025; values are player IDs. Hidden inputs hold existing selections,
named checkboxes add selections; unnamed checkboxes carry no named player payload. Example
WR0025=15754. No player-prefill GET parameter is seen; WEEK and F links choose scope, not a
drafted set of starters.

[seen: run:data/mfl-pages/0054-submit-lineup.json
  url, forms[container-wrap], clickables week links]

Public landing shape: liveScoring.week, matchup[].franchise[].id, players.player[].id and
status="starter". Match the exact drafted starter ID set, same league/franchise and requested
week, not just one player. Captured liveScoring contains week 4 starters but no submission
timestamp. Transactions contains no lineup type. These sources cannot prove when the lineup was
submitted.

[seen: run:data/mfl-exports/liveScoring.json
  liveScoring.week and matchup[].franchise[].players.player[]]

[seen: run:data/mfl-exports/transactions.json · all transaction.type values]

I1. Proposed landing requires a fresh post-hand-off, week-scoped liveScoring observation and a
pre-hand-off baseline that differs from the draft. An already matching baseline is not proof of a
new submission. M-027 supports verified for the captured week-4 form and landing shape only;
future-week coverage remains unverified. [inferred] Verify: fetch public liveScoring with W=5 and
compare with the week-5 lineup page before/after a controlled change.

### roster.ir · M-028

Open https://{host}/{year}/options?L={league}&O=18. Registry O=18 matches the captured URL (0366
is only the dump filename). POST action https://{host}/{year}/ir; LEAGUE_ID scopes league,
FRANCHISE_ID=0025 scopes owner. The opening URL has no franchise parameter; the logged-in owner is
reflected in the hidden field. Cookie selection mechanism itself is not dumped.

activate0025 values are player IDs to activate from IR (16622, 16692, 15831, 17211). drop0025
selects a roster drop, NOT an IR placement. Every active row shows Not-Eligible in Deactivate; no
deactivation checkbox/name is captured. No honored player-prefill GET parameter is seen.

[seen: run:data/mfl-pages/0366-ir-placement.json
  url, forms fields, text and tables Deactivate/Activate]

Public landing shape: rosters.franchise[].id and player[].id with status="INJURED_RESERVE" for
placement, "ROSTER" for activation. Transactions type="IR", franchise, deactivated or activated
comma-separated IDs, timestamp. Capture shows both lists and optional by_commish. No transaction
ID is present.

[seen: run:data/mfl-exports/rosters.json · rosters.franchise[].player[].status]

[seen: run:data/mfl-exports/transactions.json · type IR rows]

I2. Proposed IR landing matches same league/franchise/player and requested status in a
post-hand-off fetch; when corroborating an event, require type IR, the requested ID in the
appropriate list and timestamp >= hand-off, excluding baseline events. Registry must stay spec for
placement: its move-bearing deactivation field is missing. Activation form and landing source are
seen. [inferred] Verify: capture an eligible-player IR page and a controlled placement/activation
pair.

DraftIR currently leaves this target Unmapped because O=18 was spec-only; do not confuse the
existing permissive IRCheck note with proved eligibility.

[seen: internal/envelope/draft.go · DraftIR]

[seen: internal/envelope/ir.go · IRCheck]

### roster.taxi · M-029

Open https://{host}/{year}/options?L={league}&O=98. Registry O=98 matches capture. POST action
https://{host}/{year}/taxi_squad; LEAGUE_ID and FRANCHISE_ID=0025 scope the move. No franchise
selector is in this URL; hidden owner reflects the logged-in session, whose cookie mechanics are
not captured.

promote0025 selects player IDs to promote (17492, 17483, 17353, 17672, 17602, 17598). drop0025
drops players from roster; it does not demote. Active rows show Cannot be demoted or Locked, and
no demote field is captured. No honored player-prefill GET parameter is seen.

[seen: run:data/mfl-pages/0064-taxi-squad.json · url, forms fields, tables and text]

Public landing shape: rosters.franchise[].id, player[].id, status="TAXI_SQUAD" for demotion or
"ROSTER" for promotion. Transactions type="TAXI", franchise, demoted/promoted ID lists, timestamp.

[seen: run:data/mfl-exports/rosters.json · player.status]

[seen: run:data/mfl-exports/transactions.json · type TAXI rows]

I3. Proposed taxi predicate follows I2 freshness/baseline rules, with type TAXI and
demoted/promoted lists. Registry stays spec for demotion: no move-bearing demotion field is seen.
Promotion form and landing source are seen. [inferred] Verify: capture an unlocked eligible taxi
demotion form and one demotion/promotion pair.

### trade.propose · M-033

Open the non-mutating desk https://{host}/{year}/options?L={league}&O=05. Builder candidate
https://{host}/{year}/options?L={league}&O=05&FRANCHISE={franchise} uses FRANCHISE as counterparty
(captured 0001), not the logged-in owner. Registry O=05 matches both dumps.

Desk POST action "options" resolves to https://{host}/{year}/options. Hidden LEAGUE_ID=14432,
OPTION=05, FRANCHISE=0025 coexist with a select also named FRANCHISE containing 31 counterparties.
The dump omits option values, so duplicate-field server semantics are not known. This is a
counterparty chooser, NOT the proposal submission form. Builder dump has only chrome player_search
GET form and the text Commissioner Access Required. No proposal action, method, player/pick fields
or player-prefill URL is seen.

[seen: run:data/mfl-pages/0060-trades.json · url and forms[container-wrap]]

[seen: run:data/mfl-pages/0096-trade-builder.json · url, forms, text Commissioner Access Required]

Public transactions exposes completed type="TRADE", franchise/franchise2,
franchise1_gave_up/franchise2_gave_up, timestamp, expires, comments and optional by_commish. It
exposes no offer ID and no pending proposal type. A proposal is not a completed transfer.

[seen: run:data/mfl-exports/transactions.json · type TRADE rows]

I4. Registry stays spec: proposal form and authenticated pendingTrades shape are missing. Proposed
submission observation matches pending offer identity, both franchises and both full asset sets;
it can enter DOT review only with actual submission evidence, not just page opening. A public
completed TRADE with those assets may eventually prove execution, not initial proposal acceptance.
[inferred] Verify: capture permitted owner proposal form and pendingTrades before/after proposing;
agree lifecycle semantics with Claude.

### trade.accept · M-033

Open https://{host}/{year}/options?L={league}&O=05. Captured desk says TRADES OFFERED TO YOU
(NONE); no accept control, form action/method, trade-ID input or honored accept-prefill URL is
seen. Owner 0025 comes from session/hidden FRANCHISE, not a proved accept franchise selector.

[seen: run:data/mfl-pages/0060-trades.json · text and forms]

I5. Registry stays spec: incoming-offer accept form is missing. Proposed execution predicate
requires a new public type TRADE event with both franchise IDs, both complete gave_up sets
(players and FP pick tokens), timestamp >= hand-off, excluding baseline events; corroborate each
player destination in public rosters. Acceptance awaiting DOT is not yet executed/Landed. expires
is not acceptance time. [inferred] Verify: capture incoming-offer page, authenticated
pendingTrades through approval, and matching public execution event.

[seen: run:data/mfl-exports/transactions.json · TRADE field shape (predicate proposal is I5)]

[seen: run:data/mfl-exports/rosters.json · franchise[].id and player[].id (corroboration shape)]

### trade.reject · M-033

Open https://{host}/{year}/options?L={league}&O=05. No reject form, action/method, trade-ID field
or prefilled reject URL is captured because incoming offers are absent. Session owner is 0025 on
this desk. Public transactions has no reject type.

[seen: run:data/mfl-pages/0060-trades.json · text TRADES OFFERED TO YOU (NONE), forms]

[seen: run:data/mfl-exports/transactions.json · all type values]

I6. Registry stays spec: reject controls and authenticated rejection evidence are missing.
pendingTrades is the required auth source to investigate, not a known rejection schema. Offer
disappearance alone could be rejection, revoke, expiration or acceptance, so never treat it as
conclusive. [inferred] Verify: capture reject form and auth export before/after rejection, with
explicit outcome or independent authoritative outcome evidence.

### trade.revoke · M-034

Open the safe desk https://{host}/{year}/options?L={league}&O=05. Registry gives no O= for revoke.
Actual captured link: trade_response?L=14432&TRADE_ID=1526&ACTION=revoke, resolving to
https://{host}/{year}/trade_response?L={league}&TRADE_ID={trade}&ACTION=revoke. This is a GET
action link, not a form; there is no form action/method to record for revoke. TRADE_ID identifies
the offer, ACTION selects revoke and L scopes league; franchise is absent from this link and owner
is session-scoped on the desk.

This link is trade-specific (1526), not proof of a prefilled confirmation page. The dump does not
show whether opening it mutates immediately. No public revoke type exists.

[seen: run:data/mfl-pages/0060-trades.json · clickables revoke, text offered-by-you, forms]

[seen: run:data/mfl-exports/transactions.json · all type values]

I7. Never use the revoke action link as an automatic envelope hand-off target without confirming
it is non-mutating; open the desk instead. M-034 stays spec: link is seen but form/confirmation
and authenticated outcome are not. pendingTrades disappearance is ambiguous as in I6. [inferred]
Verify: capture revoke confirmation/response and authenticated before/after outcome evidence
without assuming link behavior.

## §2 Transaction catalogue

Captured transaction count: 770. The following are all observed type values, not a claim about
every MFL type. Examples retain IDs and raw stamps only; trailing commas delimit lists.

[seen: run:data/mfl-exports/transactions.json · transactions.transaction[]; aggregate count]

| type | count | ID-only example payload |
| --- | ---: | --- |
| FREE_AGENT | 460 | franchise=0020; transaction=\|16289, |
| LOAD_ROSTERS | 112 | franchise=0008; transaction=\|14904, |
| TRADE | 109 | franchise=0019; franchise2=0018; see below |
| IR | 47 | franchise=0024; deactivated=16696,17166,; activated=empty |
| TAXI | 42 | franchise=0018; demoted=17762,; promoted=17469, |

[seen: run:data/mfl-exports/transactions.json · counts by type and first row of each type]

Every type has franchise, timestamp, type. FREE_AGENT and LOAD_ROSTERS add transaction and
optional by_commish. IR adds activated, deactivated and optional by_commish. TAXI adds demoted and
promoted. TRADE adds franchise2, franchise1_gave_up, franchise2_gave_up, expires, comments and
optional by_commish. These are unions of captured fields, not guarantees that optional fields
always exist.

TRADE example: timestamp=1790999418; expires=1791417600; franchise=0019; franchise2=0018;
franchise1_gave_up=17434,15385,16751,FP_0019_2027_1,FP_0019_2027_2,FP_0019_2028_1,;
franchise2_gave_up=16213,13447,. No trade/offer ID is present in these rows. IR example
timestamp=1791338104; TAXI example timestamp=1790901568. No lineup-related type occurs in this
sample.

[seen: run:data/mfl-exports/transactions.json · field unions and example rows by type]

IR activated removes from IR; deactivated places on IR. TAXI promoted removes from taxi; demoted
places on taxi. The activation/promotion direction is visible in the page headings and matches
exported roster statuses; simultaneous nonempty lists must be treated separately, not as one
player move.

[seen: run:data/mfl-pages/0366-ir-placement.json
  SELECT PLAYERS TO ACTIVATE versus DEACTIVATE/DROP]

[seen: run:data/mfl-pages/0064-taxi-squad.json · SELECT PLAYERS TO PROMOTE versus DEMOTE/DROP]

[seen: run:data/mfl-exports/transactions.json · IR/TAXI list pairs]

I8. Interpret timestamp, expires and kickoff as Unix seconds: 1790900100 maps to 2026-10-02 00:15
UTC (Oct 1 8:15 p.m. EDT), agreeing with captured week-4 Thursday labels. Epoch stamps are
zone-independent; display deadlines in computer local time, never parse them as local wall time.
Unit is strongly corroborated, not explicitly documented in exports. [inferred] Verify: obtain MFL
API timestamp documentation or compare a controlled event time and its export stamp.

[seen: run:data/mfl-exports/nflSchedule.json · matchup kickoff=1790900100]

[seen: run:data/mfl-pages/0054-submit-lineup.json · week-4 Thursday kickoff labels]

## §3 Current week and lineup lock

On the Wednesday capture, nflSchedule.week="4" and liveScoring.week="4" identify the returned
scoring/schedule week, not necessarily the upcoming lineup week. All 16 schedule matchups have
gameSecondsRemaining="0". Kickoffs run from 1790900100 to 1791245700 (Oct 2 00:15 UTC to Oct 6
00:15 UTC under I8). Neither export contains week-5 games. Manifest times are Oct 7 08:38–08:40
CDT, not a week rollover record.

[seen: run:data/mfl-exports/nflSchedule.json · week, all matchup gameSecondsRemaining and kickoff]

[seen: run:data/mfl-exports/liveScoring.json · liveScoring.week]

[seen: run:data/mfl-exports/MANIFEST.txt · fetch stamps and URLs (no week arguments)]

The lineup form also says WEEK=4 and LINEUP FOR WEEK 4. Its SELECT WEEK links explicitly include
WEEK=5&F=0025 through WEEK=17. Thus the page supports choosing a different lineup week; nothing
here proves that MFL rolled a default week on Tuesday. Week 5 starts Thursday is brief context,
not a captured week-5 kickoff.

[seen: run:data/mfl-pages/0054-submit-lineup.json · forms.WEEK, text and clickables week links]

[seen: brief:p0a · §3 week-5 Thursday context]

I9. Proposed lineup target week is the explicitly selected week. To suggest an upcoming week,
advance from a completed schedule week only when that whole week is complete and a next-week
schedule is fetched; never blindly use liveScoring.week or week+1. Before the first game, retain
the selected week; during a week, retain it for remaining unlocked players. After endWeek show no
next league lineup. [inferred] Verify: one week-5 lineup-page capture validates WEEK=5; one
nflSchedule W=5 fetch supplies actual future dates.

league.lockout="No" and partialLineupAllowed="YES"; the page offers Submit Partial Lineup and many
hidden existing starter fields. This proves partial submission is exposed, not the precise server
meaning of lockout. Page rows reflect Locked on taxi while the IR page has a different capture
context. No page fetch timestamp is supplied in the dumps.

[seen: run:data/mfl-exports/league.json · lockout, partialLineupAllowed]

[seen: run:data/mfl-pages/0054-submit-lineup.json · forms and Submit Partial Lineup]

[seen: run:data/mfl-pages/0064-taxi-squad.json · text Locked; top-level keys lack capture timestamp]

I10. Proposed locks are per-player NFL-team kickoff for the selected week, not one global first
kickoff. Resolve player to NFL team through an authoritative player source; bye/FA/missing team
remains unknown rather than assigning an invented deadline. league settings and partial form
suggest rolling locks but do not prove eligibility after kickoff or the semantics of lockout=No.
[inferred] Verify: one league lineup-settings/help capture defining lockout and
partialLineupAllowed closes semantics; nflSchedule W=5 closes missing kickoff times; an
unlocked/locked owner lineup capture validates enforcement.

leagueclock currently accepts season, phase and commissioner events, leaves rule windows unknown
and does not derive a week. Dated schedule facts belong in an adapter-fed extension, not domain
imports of raw MFL shapes.

[seen: internal/leagueclock/clock.go · Inputs, Clock, unknownWindows, package comment]

## §4 Phase derivation · proposal, not a new ruling

domain.Phase defines OFFSEASON, REGULAR_SEASON, PLAYOFFS; the loaded season owns its offseason and
rollover is PLAYOFFS(N) → OFFSEASON(N+1). League settings give startWeek=1,
lastRegularSeasonWeek=13, endWeek=17; rulebook regular season ends week 13.

[seen: internal/domain/phase.go · Phase constants and season ownership comment]

[seen: run:data/mfl-exports/league.json · startWeek, lastRegularSeasonWeek, endWeek]

[seen: docs/league-rules/Official_Rulebook.md · §5]

I11. Proposed pure derivation for a validated league week w: before startWeek → OFFSEASON;
startWeek <= w <= lastRegularSeasonWeek → REGULAR_SEASON; lastRegularSeasonWeek < w <= endWeek →
PLAYOFFS; after endWeek → OFFSEASON of N+1, subject to rollover approval. Require valid ordered
bounds and dated week evidence; unknown week is unknown, never the seed phase as a fallback. Week
13→14 changes phase; before week 1 is offseason; after week 17 must not imply an ordinary
same-season append. [inferred] Verify: Claude approves boundary/season semantics; fetch dated
weeks 1/14/17 and season-end evidence, including completion of final games.

The log seeds OFFSEASON, orders latest by seq and forbids update/delete via triggers.
AppendPhaseTransition reads committed phase, rejects no-op and permits corrections. RolloverSeason
is only legal from PLAYOFFS, advances year and roster/contract snapshots and expires contracts.
CurrentPhase errors on missing or invalid history.

[seen: internal/store/state/season_phase.go
  DDL, seedInitialPhase, AppendPhaseTransition, RolloverSeason, CurrentPhase]

phase_gate is default-deny; AdvancePhase is legal in every phase, RolloverSeason only PLAYOFFS,
Buyout only OFFSEASON. SIGN additionally checks its window, TRADE its deadline. Gates read
committed state before mutation, not the planned phase in the same transaction.

[seen: internal/transactions/phase_gate.go · phasePolicy and gatePhase]

I12. Proposed owner: transaction coordinator invokes an automated deriver after each successful
scoped league/schedule refresh and before exposing phase-dependent Acts. If validated desired
phase differs, append through existing AdvancePhase in its own write transaction, then refresh
readers; equal phase is a no-op outside AppendPhaseTransition. Note should record derived phase,
league/year/week, source fetch instant and bounds. Never update/delete history or bypass
gatePhase. Startup/late refresh may jump directly to current phase but must record evidence, not
replay guessed transitions. Serialize derivation to avoid competing appends. [inferred] Verify:
Claude approves coordinator wiring, stale-source policy, manual-correction handling and tests for
startup/13→14/concurrent refresh.

I13. Do not automatically execute RolloverSeason merely because an old export remains week 17 or
the calendar reaches January: it has irreversible contract side effects. Separate pure proposed
phase from approved year rollover, use the existing standalone rollover transaction, and never
regress to OFFSEASON(N) from next-season evidence. First boot after season end without PLAYOFFS
history is an unresolved bootstrap case. [inferred] Verify: Claude chooses season-completion
source and rollover authorization/bootstrap policy.

## §5 Rules for ring 1

| Rule | Value | Source |
| --- | --- | --- |
| Total starters | 21; iop_starters=8; idp_starters=12 | league.starters |
| QB / PK | 1 / 1 | league.starters.position |
| RB / WR / TE | 1–3 / 2–5 / 1–3 | league.starters.position |
| DT / DE / LB / CB / S | each 2–4 | league.starters.position |
| Roster size | 80 | league.rosterSize |
| Position roster limits | all ten positions 0-0 (literal) | league.rosterLimits |
| IR slots | 12 | league.injuredReserve |
| Taxi slots | 8 | league.taxiSquad |
| IR/taxi salary included | 100 / 100 | league.includeIRWithSalary/includeTaxiWithSalary |
| Taxi contract-year included | 100 | league.includeTaxiWithContractYear |
| Trade expiry default | 7 days | league.defaultTradeExpirationDays |
| Maximum waiver rounds | 4 | league.maxWaiverRounds |
| Partial lineup / lockout | YES / No | league.partialLineupAllowed/lockout |

[seen: run:data/mfl-exports/league.json · named fields above; all position rows]

IR eligibility: active rows are Not-Eligible in the Deactivate column, not Drop. Their drop
checkboxes still exist, so Not-Eligible means placement is unavailable for that row, not that
every action is unavailable. Current IR players have activation controls and IR labels. injuries
export marks those four players status="IR". Full allowed status list (including PUP) is absent
from league export and page settings.

[seen: run:data/mfl-pages/0366-ir-placement.json
  Deactivate header, Not-Eligible rows, drop and activate fields]

[seen: run:data/mfl-exports/injuries.json · injury IDs 16622, 16692, 15831, 17211]

Taxi eligibility: page shows Cannot be demoted or Locked on active players; promotion controls
exist for six current taxi players. These snapshots do not expose the experience threshold,
demotion dates or whether promotion permanently removes eligibility. The league export gives
capacity only, not an eligibility algorithm.

[seen: run:data/mfl-pages/0064-taxi-squad.json · tables, text and forms]

[seen: run:data/mfl-exports/league.json · taxiSquad and absence of eligibility fields]

I14. Proposed lineup legality checks total 21 and all position ranges together; do not equate
iop_starters=8 with eight including PK (that would conflict with 21 and 12 defense). Proposed
interpretation is eight offensive skill starters plus one PK and 12 defense. Numeric ranges alone
also permit flex distributions the rulebook may prohibit. Do not interpret rosterLimits 0-0 as
zero allowable players or assume how 80 counts IR/taxi. [inferred] Verify: one league
lineup/roster settings capture defines iop_starters and 0-0, roster counting and flex constraints;
compare known legal lineups.

rules.json is scoring positionRules, not roster/eligibility policy. The read-only archive
allRules.json is a global scoring-event dictionary (e.g. #P Passing TDs), not league roster rules.
nflByeWeeks supplies team.id/bye_week/year; injuries supplies injury status by id, week and
timestamp, not league IR acceptance policy.

[seen: run:data/mfl-exports/rules.json · rules.positionRules]

[seen: ~/work/TheWarRoom/league-archive/raw/allRules.json
  allRules.rule shortDescription/abbreviation]

[seen: run:data/mfl-exports/nflByeWeeks.json · nflByeWeeks.team, year]

[seen: run:data/mfl-exports/injuries.json · injuries.injury, week, timestamp]

### Rulebook versus MFL: observed discrepancies only

Roster: rulebook active minimum 35 / maximum 48, excluding IR/taxi; MFL rosterSize=80 and position
limits 0-0. Different counting units are possible; export does not settle them. Lineup: rulebook
says offense 9 including kicker, defense 12; export says iop_starters=8, idp_starters=12,
count=21. Raw labels disagree; kicker accounting remains open, not a proved 20-starter rule.

[seen: docs/league-rules/Official_Rulebook.md · §2 and §3]

[seen: run:data/mfl-exports/league.json · rosterSize, rosterLimits, starters]

I15. Rulebook flex slots imply TE max 3 requires WR >=3: TE=3 consumes TE/WR and offensive flex,
leaving RB/WR flex. Export ranges permit RB=3, WR=2, TE=3 with QB=1 and PK=1 (nine offense), which
lacks a legal rulebook flex allocation. This is a settings expressiveness discrepancy, not proof
MFL accepts that combination. [inferred] Verify: capture MFL full flex settings or validate that
distribution on the owner form without submitting a real change.

[seen: docs/league-rules/Official_Rulebook.md · §3 named offensive flex slots]

[seen: run:data/mfl-exports/league.json · starters.position ranges]

Trade procedure: rulebook requires Proboards posting/acceptance and 3 DOT approvals/vetoes, with
no withdrawal after acceptance; MFL desk exposes a pending offer/revoke link. These are different
workflows, not proof that MFL settings abolish DOT or the week-9 deadline. Waivers: rulebook uses
Proboards, while MFL exposes maxWaiverRounds=4; that setting does not govern TheWarRoom waiver
implementation under ruling 3.

[seen: docs/league-rules/Official_Rulebook.md · §8 and §14]

[seen: run:data/mfl-pages/0060-trades.json · pending offered-by-you and revoke link]

[seen: run:data/mfl-exports/league.json · maxWaiverRounds]

No observed settings contradiction for taxi capacity 8. Rulebook eligibility (<=3 NFL years) and
IR/PUP eligibility cannot be compared with missing MFL eligibility settings. Rulebook gives no IR
slot number or trade-expiry default; 12 IR slots and 7 days are additional MFL facts, not
disagreements. No conflicts are resolved here; ruling 6 controls IR/taxi for now.

[seen: docs/league-rules/Official_Rulebook.md · §2 and §14]

[seen: run:data/mfl-exports/league.json · taxiSquad, injuredReserve, defaultTradeExpirationDays]

## §6 Auth · Claude probe, 2026-10-07; not re-probed

Public without login: league, rosters, liveScoring, transactions, rules; league-independent
nflSchedule, nflByeWeeks, injuries. Login required: pendingTrades, calendar. Error: "API requires
logged in user … MFL_USER_ID cookie or APIKEY". These are supplied probe facts, not conclusions
from these public files alone.

[seen: brief:p0a
  Known facts (Claude probed MFL on 2026-10-07)]

I16. Proposed pendingTrades consumers: Home pending-trade alert and Trade Floor offer list,
trade.propose observation, accept-awaiting-review, reject/revoke investigation and DOT stages.
Proposed calendar consumers: Home seasonal card/alert tray and trade-window deadlines through
leagueclock. calendar is not required to see public roster transfers; neither auth export is
supplied, so fields, timezone labels and negative-outcome predicates remain unknown. [inferred]
Verify: fetch one authenticated pendingTrades sample and calendar sample, then capture controlled
trade lifecycle outcomes.

[seen: docs/build-handoffs/UI_Target_Roadmap_2026-10.md · Ring 1 Home and Trade Floor surfaces]

## §7 Open points

1. Eligible IR placement field and full MFL status policy: one eligible-owner IR page plus IR
settings/help; do not infer a field name from activated/deactivated export names. [inferred]

2. Taxi demotion field, eligibility and lock window: one unlocked eligible-owner taxi page and
taxi settings/help. Current form proves promotion only. [inferred]

3. Proposal form/permissions: capture through the actual owner counterparty chooser; current
FRANCHISE=0001 URL returns Commissioner Access Required. Explain duplicate FRANCHISE encoding.
[inferred]

4. Accept/reject controls: incoming-offer page with action/method, trade ID and confirmation;
absence today is not proof those Acts are unsupported. [inferred]

5. Auth lifecycle: pendingTrades samples around propose/accept/reject/revoke/expire/DOT approval.
If outcomes are not explicit, another authoritative source is required; absence is never
sufficient. [inferred]

6. Revoke safety: inspect its confirmation/response before choosing any action URL as a hand-off.
Until then use the desk, not ACTION=revoke. [inferred]

7. Upcoming lineup: week-5 owner page, public liveScoring W=5 and nflSchedule W=5. These
separately settle selection, landing coverage and dated per-game deadlines. [inferred]

8. Default-week rollover: one contemporaneous no-week export/page capture settles only that
instant; a Tuesday-before/after pair is required to establish a Tuesday rollover rule. [inferred]

9. Lineup locks: MFL lineup-settings/help defining lockout and partialLineupAllowed, plus a
kickoff-boundary capture. Bye/FA and player-to-team source need explicit handling. [inferred]

10. Roster/lineup semantics: full roster and starter settings defining 80, 0-0, IR/taxi counting,
iop_starters and flex. Compare rulebook allocation with MFL legality. [inferred]

11. Automatic phase ownership: Claude approves I11–I13, source freshness, correction precedence,
transition notes and rollover/bootstrap policy before implementation. [inferred]

12. Timestamp units: MFL documentation or controlled timed event comparison confirms I8; local
display follows ruling 5 regardless of server display labels. [inferred]

13. DOT timing and authenticated calendar fields: sample calendar and approved trade lifecycle; no
invented deadlines from week-only or UI labels. [inferred]

## §8 Settled after the draft · Claude, 2026-10-07

Two more sources closed most of §7: the owner-visible league settings page and two week-5
exports, fetched 2026-10-07 at 08:5x CDT.

[seen: run:data/mfl-pages/0160-league-settings.json · options?L=14432&O=26, captured 2026-10-04]

[seen: run:data/mfl-exports/nflSchedule-w5.json, liveScoring-w5.json · MANIFEST.txt]

| Point | Settled fact | Closes |
| --- | --- | --- |
| Lineup locks | "Players Are Locked At Kickoff Of Their Game"; partial lineups YES | I10, §7.9 |
| Bye starters | "Should owners be allowed to submit players on bye as starters? Yes" | I10 |
| Offense cap | "Maximum Number of Starting QB/RB/WR/TE Players: 8"; PK is outside it | I14 |
| Defense total | "Total Number of Starting Individual Defensive Players: 12" | I14 |
| Roster limits | every position "0-No Limit"; 80 roster spots | I14, §7.10 |
| IR eligibility | classified IR; not Suspended, holdout or Reserve/COVID-19 | §7.1 (policy) |
| IR violation | does not block lineup submission | §7.1 |
| Taxi eligibility | "Players with less than 3 years of experience" | §7.2 (policy) |
| Trade approval | "When An Owner Accepts A Trade: The trade requires approval from the | I4, I5 |
| | commissioner." Invalid-roster trades are refused. Default expiry 7 days. | |
| Add/drop, waivers | "Add/Drop System: None"; "Waivers Processed Automatically: No" | ruling 3 |
| Time zone | MFL displays ET; the app shows local time (ruling 5), epoch stamps (I8) | §7.12 |
| Week-5 schedule | nflSchedule W=5: 15 games, first kickoff Thu 2026-10-08 19:15 CDT | §7.7 |
| Week-5 landing | liveScoring W=5 already lists franchise 0025's 21 saved starters | I1, §7.7 |

Lineup legality for ring 1 is therefore fully specified: QB 1, RB 1–3, WR 2–5, TE 1–3 with
QB+RB+WR+TE at most 8; PK 1; DT, DE, LB, CB and S 2–4 each with exactly 12 defensive starters;
21 starters in a full lineup, fewer allowed (partial). The rulebook's named flex slots (I15) are
not MFL's model; MFL settings rule (ruling 6), so the app checks the ranges and the two caps.

**Claude's engineering calls (carry into phases 1–7):**
1. **Phase (I11, I12) approved.** Pure derivation from the dated league week and the league
   bounds; the coordinator appends through `AdvancePhase` after a successful refresh,
   serialized, with the evidence in the note. Unknown week stays unknown.
2. **Rollover (I13): never automatic.** `RolloverSeason` expires contracts and cannot be undone.
   The app proposes it with evidence; a person confirms. Christopher's "automate" covers phase
   transitions inside a season.
3. **Revoke (I7):** hand off to the trade desk (O=05), never to the `ACTION=revoke` link.
4. **Lineup is the first Act to land.** Form, URL and a future-week landing source are all seen,
   and a lineup change is reversible until kickoff. It is the ring 1 gate envelope.
5. **IR placement and taxi demotion stay `spec`** until a page shows an eligible row's field.
   Activation and promotion are seen and can be mapped.

**Still open:** §7.3–7.6 and §7.13 (trade proposal, accept and reject controls, revoke
confirmation, the authenticated `pendingTrades` and `calendar` shapes); they need the API key.
§7.8 (week rollover instant) is designed around: the app reads the week from the schedule it
fetched, never from a default.
