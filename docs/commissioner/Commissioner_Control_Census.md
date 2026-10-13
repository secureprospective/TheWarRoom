# Commissioner control census (K4, spec rev 4 §16.7)

Built by Claude, 2026-10-13, from: GLM census (`glm-findings.md` Step 1, K-001..K-045), Sol census (`bee-findings.md` §1,
K-001..K-104), `k3-triage.md`, `league-facts-rulings-2026-10-13.md`, `K4-outline-draft.md` (agreed sections),
`docs/league-rules/Official_Rulebook.md` v2026 + `Math_Rules_Reference.md`.

ID prefix `CM-` = commissioner control (human-facing, one per tray). Sol and GLM IDs are cited as
`Sol K-nnn` / `GLM K-nnn`; those numbers are not reused. Each verb is `commish.<domain>.<verb>`.
Every change also carries `expected:Version request:Id reason:Reason effective:Instant` (Sol), and
competitive rule changes default `effective=next_rollover` (agreed §16.4).
Roles: C commissioner · CC co-commissioner · DOT. "C only" = co-commissioner blocked (ruling 3 list).
Governs = owner-registry IDs (brief §7), carried from GLM and checked against brief §7 by K3 triage.

The source passes and triage are in the Beelink run dir `~/fleet/runs/warroom-commish-k3-2026-10-13/` (not in this
public repo). Every group was signed off by Christopher on 2026-10-13, in the order A to G.

## A. Constitution (the league's rules; offseason, effective next rollover by default)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Legacy default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-001 | League shape | How many teams, conferences, divisions; how do teams compete; how are first rosters built? | `league.teams n:Int` · `league.conferences n:Int` · `league.divisions n:Int` · `league.format kind:Enum(h2h,…)` · `league.initial_rosters kind:Enum(draft,…)` | G2 | C only | 32 · 2 · 8 · head-to-head · by draft (brief §5) | M-091, M-097, M-030, M-101, M-011, M-014 | Sol K-001,002,003,005,006 · GLM K-001 |
| CM-002 | Division alignment | Which team sits in which division? | `league.alignment set:Map<Franchise,Division>` | G2 | C only | Current alignment carried at setup | M-002, M-011, M-012, M-013, M-014 | Sol K-004 · GLM K-002 |
| CM-003 | Roster limits | How small and how large may a roster be? Any position minimums? | `roster.limits min:Int max:Int` · `roster.position_bounds set:Map<Pos,Range>` | G2 | C | 18-80, IR/taxi excluded, no position bounds (fact R) | M-030, M-101, M-027, M-036 | Sol K-022,023,024 · GLM K-007 |
| CM-004 | Lineup | How many start, and at which positions? | `lineup.bounds set:Map<Pos,{min:Int,max:Int}>` · `lineup.group_bounds set:Map<Group(offense,defense,total),{min:Int,max:Int}>` · `lineup.invalid_penalty kind:Enum(zero_top_starter,zero_slot,none)` · `lineup.missed_flag on:Bool` | G2 | C | Starting values = today's ranges: QB 1 · RB 1-3 · WR 2-5 · TE 1-3 · offense ≤8 + PK 1 · DT/DE/LB/CB/S 2-4 each · 12 IDP · 21 total (brief §5). Q-A1 RULED. Invalid lineup: zero the highest-scoring starter (rulebook §3); a missed lineup flags the team to the commissioner, removal stays a human G3 act via CM-019 (added 2026-10-13) | M-009, M-010, M-027, M-030, M-101 | Sol K-025..030 · GLM K-006 |
| CM-005 | Dynasty and carry | Do rosters carry year to year, and who carries? | `league.dynasty on:Bool` · `league.carry policy:Enum(all,keepers)` `keepers:Int?` | G2 | C only | Dynasty, full carry (brief §5) | M-091, M-097, M-030, M-080 | Sol K-035,036 · GLM K-008 |
| CM-006 | Scoring | What is each stat worth? | `scoring.set stat:StatId pts:Decimal` · `scoring.unset stat:StatId` · `scoring.bonus stat:StatId at:Decimal pts:Decimal` · `scoring.preset.apply name:Id` | G2 | C | Rulebook §4 as the "Legacy NFL" preset; true-position tackle split, TFL direct, stacking long-play bonuses (Math §13) | M-001, M-003, M-005, M-037, M-040, M-072, M-076, M-077, M-002 | Sol K-043 · GLM K-004 · fact 1 |
| CM-007 | Standings rules | How are teams ranked and ties broken? | `standings.criteria order:List<Criterion>` (no cutoff verb: standings count through the last regular-season week, set in CM-015) | G2 | C | win% › H2H › div% › conf% › all-play% › points › power rank (brief §5); cutoff = last regular-season week (week 13 for Legacy). Q-A2 RULED | M-002, M-074, M-161, M-170, M-008 | Sol K-047,048 · GLM K-005 |
| CM-008 | Salary cap | What is the cap, and is going over blocked or flagged? | `cap.set amount:Money season:Season` · `cap.enforcement mode:Enum(hard,soft)` · `cap.reserve_counts ir:Pct taxi:Pct` | G2 | C | $125M; IR and taxi count 100%; all money snaps to $10k (rulebook §1, Math §0); enforcement is a setting: hard blocks any move ending over the cap; soft allows it, flags the team and sets a compliance deadline; conditional drops work in both. Legacy default hard (rulebook §1). Q-A3 RULED | M-078, M-031, C-06.3, C-06.5, C-06.6 | Sol K-051,052,053 · GLM K-009 |
| CM-009 | Contract actions | Which contract actions exist, and at what numbers? | `contract.action.set action:Enum(tag,extension,restructure,buyout,rfa_tender,resign,retire,death,cap_relief) on:Bool params:ActionParams` | G2 | C | Every rulebook number as the preset: dead cap 35% (50% if restructured), tag top-5 avg / 120% floor / 2 consecutive, extension 150% + position floors / 1 per season / ≤6 yrs, restructure tiers / 1 per year, buyout 60/75/90% / 2 per offseason / 1-yr disallowed, RFA tenders + 110% + re-sign +0/15/30/45%, retirement 30%, death 0, max length 4, flat salaries, min salaries by experience (rulebook §6-§13, Math §1-§9) | M-078, M-031, C-06.3, C-06.5, C-06.6 | Sol K-054,057..060 · GLM K-009 · fact 5 |
| CM-010 | IR | How many IR spots, and who qualifies? | `ir.slots n:Int` · `ir.eligible designations:Set<NflStatus>` · `ir.must_leave_when_healthy on:Bool` | G2 | C | 12 spots; NFL IR/PUP/NFI/Out/suspended; leave when healthy (fact 4a) | M-028, M-029, M-030, M-149 | Sol K-038,040 · GLM K-010 |
| CM-011 | Taxi squad | How many taxi spots, and who qualifies? | `taxi.slots n:Int` · `taxi.eligible max_years:Int basis:Enum(calendar_years,career_games)` | G2 | C | 8 spots; 3 years and under (fact 4b; rulebook §2 counts by career games) | M-029, M-028, M-030 | Sol K-039,041 · GLM K-011 |
| CM-012 | Waivers | When does a player hit waivers, for how long, and who gets priority? | `waivers.trigger on:Enum(cut)` · `waivers.duration d:Duration` · `waivers.method kind:Enum(reverse_standings,rolling,…)` | G2 | C | On cut only; 24h; reverse standings (brief §6, fact 2) | M-024, M-156, M-157, M-194, C-06.7, M-036, M-056 | Sol K-061,062,063,067 · GLM K-012,013 |
| CM-013 | Free-agent bids | How do UFA and RFA bids work? | `bids.mode process:Enum(ufa,rfa) kind:Enum(blind,open)` · `bids.formula year_mult:List<Decimal>` · `bids.limits max_active:Int one_year_max:Money tiers:List<Tier>` · `bids.increment min:Decimal snipe_after:Duration snipe_min:Decimal` · `bids.phase_rules set:PhaseRules` | G2 | C | Bid points = salary × 1.0/1.2/1.4/1.6; 30 active; $12M 1-yr max; $12-24M ≥2 yrs; >$24M ≥3 yrs; +0.1 step; snipe after 20h needs +1.0; multi-year closes end wk 13; after wk 12 playoff teams 1-yr only; playoff 1-yr cap lifted (rulebook §6); mode per process (fact 3; UDFA follows UFA) | M-024, M-156, C-06.7 | Sol K-068 · GLM K-013 |
| CM-014 | Trade rules | Who reviews trades, what can be traded, and when do offers lapse? | `trade.review route:Enum(dot,commish,none)` · `trade.dot seats:Int(4..10) decide_at:Int` · `trade.future years:Int rounds:Range` · `trade.roster_gate on:Bool` · `trade.offer_expiry d:Duration` · `trade.parties max:Int` | G2 | C | DOT review, 3 approves or 3 vetoes (rulebook §14), DOT 4-10 seats (fact 7a); picks 2 years ahead, rounds 1-5; invalid rosters refused; offers expire 7 days (brief §5); up to 3 parties (ruling 4); comp picks not tradable (§14) | M-033, M-068, M-153, X-04, M-199, C-06.2 | Sol K-071..075 · GLM K-014 |
| CM-015 | Schedule | Who plays whom, each week? | `schedule.generate weeks:Int pattern:Enum` · `schedule.edit week:Week game:Id home:Franchise away:Franchise` · `schedule.publish` | G2 | C | Head-to-head through wk 13 regular season (rulebook §5); pattern division-weighted | M-011, M-012, M-013, M-003, M-005, M-015 | Sol K-078,079 · GLM K-015 |
| CM-016 | Playoff rules | Who qualifies, how are they seeded, which weeks? | `playoffs.preset name:Id` · `playoffs.graph set:Graph` · `playoffs.seeding policy:SeedPolicy reseed:Bool` · `playoffs.weeks start:Week final:Week` · `playoffs.home_bonus pts:Decimal` | G2 | C | 7 per conference (4 division winners + 3 wild cards), #1 bye, NFL-style reseed (facts 11a-b); weeks 14-17 (brief §5); +3 home field (rulebook §5) | M-014, M-090, M-011 | Sol K-093,094,095,097 · GLM K-016 |
| CM-017 | House rules | What else does the league rule that the settings don't cover? | `rules.text.add id:Id text:Text` · `rules.text.remove id:Id` | G1 | C | Rulebook imported as text (read-only reference) | M-097, M-185 | Sol K-007 · GLM K-003 |

### Rulings for group A
- Q-A1 RULED 2026-10-13: min and max only, per position and per group; the commissioner/league decides
  the rest. Not coded to MFL or Legacy structures; no flex-slot model. (Was: lineup model. Rulebook §3 uses flex slots (1 QB, 1 RB, 2 WR, 1 TE, RB/WR flex, TE/WR flex,
  offensive flex, K; 2 each IDP + 2 defensive flex). MFL uses per-position ranges with an 8-player cap.
  They are not the same: MFL allows 3 RB + 3 TE + 2 WR; the rulebook does not (one offensive flex can't
  cover both). Defense is equivalent.)
- Q-A2 RULED 2026-10-13: standings follow the season end; no separate cutoff setting. (Was: standings cutoff. MFL counts standings through week 17; the rulebook ends the regular season at
  week 13 with playoffs 14-17.)
- Q-A3 RULED 2026-10-13: hard or soft is a commissioner setting. (Was: cap enforcement. MFL is set to a soft cap; rulebook §1 says "under cap at all times. No exceptions."
  The conditional-drop memory covers over-cap bids.)

**Group A SIGNED OFF 2026-10-13.**

## B. People (seats, abilities, roles, succession)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-018 | Franchise identity | What is each franchise called? | `franchise.edit id:Franchise name:Text abbr:Text` | G2 | CC | Current 32 identities at setup | M-091, M-110, M-023, M-105 | Sol K-008 · GLM K-017 |
| CM-019 | Owner seats | Who holds each franchise, and how do owners join or leave? | `seat.invite franchise:Franchise handle:Text` (G2) · `seat.invite.revoke invite:Id` (G2) · `seat.remove franchise:Franchise` (G3) · `seat.replace franchise:Franchise handle:Text` (G3) | G2/G3 | CC | Invite is single-use, bound to one franchise; removing an owner keeps the franchise, contracts and picks; invites stay private (ruling 5) | M-091, M-110, M-032 | Sol K-009,010,011,014 · GLM K-018,020 |
| CM-020 | Owner abilities and visibility | What may each owner do, and who can read what? | `abilities.set franchise:Franchise\|all ability:Ability on:Bool` · `visibility.set report:Id audience:Enum(all,commish)` | G2 | CC | All 32 identical: lineups, add/drop, drop-only, trades, pick trades, IR, taxi, articles, board, chat, polls on; customize franchise / home page off (brief §5). Survivor and NFL pool tagged `deferred` (pools are future-native) | M-091, M-024, M-027, M-033 | Sol K-012,013 · GLM K-019 |
| CM-021 | Roles | Who is commissioner, co-commissioner, or on the DOT? | `role.grant handle:Text role:Enum(commish,cocommish,dot)` · `role.revoke handle:Text role:Enum` | G3 | C only | The app assigns nobody; creator is first commissioner (fact 8a); last commissioner can't be revoked; DOT 4-10 seats (fact 7a): granting an 11th is refused, and with fewer than 4 the trade-review control shows "Needs a ruling: DOT has n of 4 seats" and trades wait (ruling 6); grants private (ruling 5) | M-091 | Sol K-015 · GLM K-021 |
| CM-022 | Succession | What happens when the commissioner leaves, goes quiet, or loses the league's confidence? | `succession.stepdown successor:Handle?` · `succession.takeover` (CC, after inactivity) · `succession.inactivity in_season:Duration offseason:Duration` · `succession.voteout.open` · `succession.election.open` · `vote.cast ballot:Id choice:Id` | G3 | see rows | Co-commissioner takes over first, else owner election (fact 8c); inactivity 14 days in season / 30 offseason, adjustable, all owners notified; vote-out needs two-thirds of franchises, locked (§16.3); every step on the ledger. a co-commissioner or a DOT member opens a vote-out or election (Q-B1 RULED) | M-091 | new (facts 8a-c, §16.3) |

### Rulings for group B
- Q-B1 RULED 2026-10-13: only a co-commissioner or a DOT member can open a vote-out or an election.

**Group B SIGNED OFF 2026-10-13.**

## C. Calendar (league time, windows, deadlines)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-023 | League windows | When does each window open and close, and what does it open? | `calendar.window.set id:Id start:Date end:Date opens:Set<Action>` · `calendar.window.move id:Id start:Date` · `calendar.window.remove id:Id` | G2 | CC | Legacy dates (brief §1): contract window Mar 1-Apr 30 (actions chosen by the commissioner, §16.6) · RFA tenders May 1-3 · UFA May 3-10 · re-sign May 10-15 · open FA from May 19 · rookie draft from Jul 14 · cut day and buyouts Aug 25-Sep 1; re-dated a year at rollover | M-096, C-04, S-04, C-06.7, M-183 | Sol K-018 · GLM K-041 |
| CM-024 | League clock | Which timezone runs the league, and where does a shared date split? | `calendar.timezone tz:Tz` · `calendar.boundary side:Enum(start_of_day,end_of_day)` | G2 | CC | US Eastern, daylight saving followed; windows change at midnight league time; split side set by the commissioner (facts 12a-b). Owners always see their own local time | M-096, C-04, S-04 | Sol K-021 · facts 12a-b |
| CM-025 | Trade deadline | When do trades stop? | `calendar.trade_deadline at:Instant` | G2 | CC | Week 9, exact moment set by the commissioner, on the calendar with a countdown (rulebook §14, fact 12c) | M-033, M-153, M-096 | Sol K-020 · brief §6 |
| CM-026 | Window exception | May this one action happen outside its window? | `calendar.exception action:Action franchise:Franchise until:Instant` | G3 | CC | None; one named action, one franchise, time-boxed, typed reason, visible to all owners (rulings 5, 8) | M-096, C-06.7 | Sol K-019 |

**Group C SIGNED OFF 2026-10-13.**

## D. Season ops (running the year; mostly overrides on automated rules)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-027 | Season rollover | Can I move the league forward a year safely? | `season.rollover to:Season dry_run:Bool` | G3 | C only | One act: preview, snapshot, before/after. Contracts age a year; expiring players marked pending, not released (9b); IR and taxi back to active (9a); cap resets, dead cap carried; calendar re-dated; rule changes waiting for next rollover take effect (§16.4); champions archived. Picks are not issued here (9c) | M-030, M-078, M-068, M-063, M-096, C-04, M-032, M-080, M-082..M-085, M-105, M-114, M-104 | Sol K-037 · GLM K-028 |
| CM-028 | Roster and asset corrections | Which roster move, provenance or pick ownership was recorded wrong? | `correct.roster franchise:Franchise player:Player action:Enum(add,remove)` · `correct.transaction id:Id` · `correct.provenance player:Player via:Enum` · `correct.pick pick:Id owner:Franchise` | G3 | CC | Compensating ledger entries only, never deletes (triage settled #4) | M-030, M-101, M-032, M-079, M-078, M-068 | Sol K-032,033,092 · GLM K-029 |
| CM-029 | Lineup correction | Which lineup should count, despite the lock? | `lineup.correct franchise:Franchise week:Week set:Lineup` | G3 | CC | None; rescore follows | M-027, M-003, M-005 | Sol K-031 |
| CM-030 | Player position | The feed has the wrong position; what is right? | `player.position id:Player pos:Pos` · `player.position.clear id:Player` | G2 | CC | Feed rules; override only when the feed is wrong | M-037, M-079, M-030, M-101, M-009, M-048 | Sol K-034 · GLM K-030 |
| CM-031 | Stat corrections | The NFL changed a stat; do we accept it? | `stat.queue.approve id:Id` · `stat.queue.reject id:Id` · `stat.correct player:Player week:Week stat:StatId value:Decimal source:Text` | G2 | CC | Feed changes queue in the alert tray, 1-2 clicks to approve (ruling 7, new feature beyond the brief); manual entry when the feed is silent; pre-flight flags any result flip | M-088, M-037, M-003, M-002 | Sol K-044 · GLM K-033 |
| CM-032 | Score adjustments | Which player or team score is wrong? | `score.player week:Week player:Player delta:Decimal` (G2, escalates to G3 if it flips a result) · `score.franchise week:Week franchise:Franchise delta:Decimal` (G3) | G2/G3 | CC | Deltas only, inverse delta to undo | M-037, M-003, M-005, M-072, M-002, M-071, M-007 | Sol K-045,046 · GLM K-031,032 |
| CM-033 | Standings override | What record or full order must stand? | `standings.record.adjust franchise:Franchise w:Int l:Int t:Int` · `standings.override order:List<Franchise>` · `standings.override.clear` | G3 | C only | Computed standings always shown; override stored with reason (brief §6) | M-002, M-090, M-014 | Sol K-049,050 · GLM K-034 |
| CM-034 | Contract and cap override | Which contract, salary or cap hit is wrong, or earns relief? | `contract.edit player:Player years:Int salary:Money` · `cap.adjust franchise:Franchise amount:Money season:Season` · `cap.relief player:Player amount:Money` | G2 (G3 if it changes a pending trade's cap compliance) | CC | None; human escape hatch; cap relief appeal for career-ending/recurring injury or suspension (rulebook §13) | M-078, M-031, C-06.5, C-06.6 | Sol K-055,056 · GLM K-035 |
| CM-035 | Waiver processing | Run the wire now, set the order by hand, or fix one award? | `waivers.process window:Id` · `waivers.order.set order:List<Franchise>` · `waivers.order.auto` · `waivers.award.override claim:Id winner:Franchise` | G2 | CC | The app resolves all claims at window close (brief §6); these are the manual trigger and overrides | M-156, M-157, M-024, M-032, M-079, M-056 | Sol K-064,065,066 · GLM K-036,037 |
| CM-036 | Bid processing | Award cleared players' bids now, or fix one award? | `bids.process player:Player` · `bids.award.override bid:Id winner:Franchise` | G2 | CC | The app resolves bids once waivers clear (brief §6). Corrects Sol K-069, which made this human-confirmed | M-024, M-156, C-06.7 | Sol K-069,070 · GLM K-013 |
| CM-037 | Trade review | Does this trade stand? | `trade.review id:Id decision:Enum(approve,veto)` (DOT) · `trade.recuse id:Id` (auto) · `trade.appeal id:Id` (party, once) · `trade.appeal.rule id:Id decision:Enum` (C; CC if C is party) · `trade.override id:Id decision:Enum` (C; G3, typed reason, visible to all owners; Q-D1 RULED) | G3 | DOT | 3 approves or 3 vetoes (rulebook §14); party members recused automatically; fail closed if nobody eligible (ruling 4); one appeal to the commissioner (fact 7b) | M-153, X-04, M-033, M-199, M-032 | Sol K-076,077 · GLM K-038 |
| CM-038 | Reserve placement exception | Should this player go on IR or taxi despite failing the rule? | `reserve.exception player:Player bucket:Enum(ir,taxi) until:Instant?` | G3 | CC | None | M-028, M-029, M-030 | Sol K-042 |
| CM-039 | Rule exception and reversal | Override one automated decision here, or undo one recorded change? | `rule.exception rule:RuleRef scope:Scope until:Instant?` · `correction.revert entry:Id` | G3 | CC | Every automated rule keeps a manual override that is itself a command (full-control memory); revert writes a compensating entry | ledger-wide | Sol K-103,104 |

### Rulings for group D
- Q-D1 RULED 2026-10-13: yes. The commissioner can overturn any DOT decision at any time, as G3 with a typed reason, visible to all owners.

**Group D SIGNED OFF 2026-10-13.**

## E. Rookie draft (the commissioner tray also lives in the Draft room, ruling 2)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-040 | Draft setup | How is the draft run, how many rounds, and what do picks cost? | `draft.mode kind:Enum(slow,live)` · `draft.clock d:Duration` · `draft.on_expire kind:Enum(skip,autopick)` · `draft.timeouts n:Int\|unlimited` · `draft.rounds n:Int` · `draft.max_picks n:Int` · `draft.pricing set:Map<Round,List<SlotPrice>> lengths:Map<Round,Years>` | G2 | CC | Slow draft, 6-hour clock, expired clock = skip, unlimited timeouts (brief §5); 5 rounds; max 15 picks per team, extra picks forfeited (rulebook §15); slot prices and lengths per rulebook §15 (R1 5 yrs; R2-5 3 yrs + RFA) | M-183, M-176, M-179, M-063, M-064, M-103 | Sol K-080..083 · GLM K-022 |
| CM-041 | Draft order | Who picks where, each round? | `draft.order.auto source:Enum(standings)` · `draft.order.set round:Int\|all order:List<Franchise>` · `draft.order.pattern kind:Enum(linear,snake,…)` · `draft.order.tiebreak rule:TieRule` | G2 (G3 once the clock runs) | CC | Worst team first in round 1 (brief §3); later-round pattern and tie rule have no default and show "Needs a ruling" until the commissioner sets them (Q-E1 RULED) | M-063, M-064, M-102, M-176, M-183 | Sol K-084 · GLM K-023 |
| CM-042 | Draft start | When does the draft begin? | `draft.start at:Instant` (one event shared with the calendar) | G1 | CC | From Jul 14 (brief §1) | M-096, C-04, S-04, M-176, M-183 | Sol K-089 · GLM K-024 |
| CM-043 | Live draft tray | Skip, assign, insert, change a pick, or work the clock? | `draft.skip pick:Id` · `draft.assign pick:Id player:Player` · `draft.insert after:Id franchise:Franchise player:Player` · `draft.change pick:Id player:Player` · `draft.timer action:Enum(pause,resume,set) d:Duration?` | G3 | CC | Expired clock auto-skips, manually overridable (triage take #4); insertion is always typed by a human; changing selection mid-draft warns instead of retargeting (Sol tray anatomy) | M-183, M-063, M-064, M-030, M-176 | Sol K-085,086,087,091 · GLM K-025 |
| CM-044 | Revert draft | Which draft checkpoint must we restore? | `draft.revert to:Id` | G3 | CC | None; ledger keeps both states | M-183, M-063, M-064, M-030 | Sol K-088 · GLM K-026 |
| CM-045 | Close draft | Is the draft done? Seal it and issue new picks. | `draft.archive season:Season` | G2 | CC | Auto at the last pick; undrafted rookies go to the UFA pool (fact 13); picks for two seasons out are issued and tradable at once (fact 9c) | M-063, M-064, M-066, M-065, M-068 | Sol K-090 · GLM K-027 |

### Rulings for group E
- Q-E1 RULED 2026-10-13: the later-round pattern and the tie rule are commissioner settings with no default; both show "Needs a ruling" until set.

**Group E SIGNED OFF 2026-10-13.**

## F. Playoffs and history

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-046 | Bracket and seeds | Build the brackets; seed a slot by hand? | `bracket.build preset:Id` · `bracket.edit node:Id team:Franchise` · `bracket.seed slot:Int team:Franchise` · `bracket.preset.save name:Id shape:Graph` | G2 (G3 once playoffs begin) | CC | Built from CM-016 rules: 7 per conference, #1 bye, auto-seeded from final standings, NFL-style reseed; per-slot manual override (triage settled #8, facts 11a-b) | M-014, M-090, M-011, M-003 | Sol K-096 · GLM K-039 |
| CM-047 | Advancement override | Which team should advance despite the computed result? | `bracket.advance game:Id team:Franchise` | G3 | CC | None | M-014, M-090 | Sol K-098 |
| CM-048 | Champion title | What season does the title name? | `title.set season:Season label:Text` | G1 | CC | Year-stamped automatically (fixes MFL's stale "2025 Super Bowl Champion" label, brief §5) | M-104, M-114 | Sol K-102 · GLM K-016 |
| CM-049 | Franchise history link | Which historical franchise does this record belong to? | `history.link record:Id franchise:Franchise` | G2 | CC | Links carried from the archive at setup | M-105, M-114, M-104 | Sol K-099 · GLM K-028 merge |
| CM-050 | League awards | Which awards exist, and who wins them? | `awards.define set:AwardSet` · `awards.grant award:Id season:Season franchise:Franchise` | G1 | CC | **Tagged `deferred`**: awards, pools and mini games are a later, native War Room feature (memory warroom-pools-are-future-native). The command exists so CLI/chat refuses with a real message | M-081, M-104, M-114, M-105 | Sol K-100,101 · GLM K-040 |

**Group F SIGNED OFF 2026-10-13.**

## G. League ops (comms, data, engine, the ledger)

| ID | Control | Plain question | Verbs (typed args) | G | Min role | Default (cite) | Governs | Sources |
|---|---|---|---|---|---|---|---|---|
| CM-051 | Notifications | What does the league get told, and where do commissioner alerts go? | `notify.set event:Event channel:Channel on:Bool` · `notify.route alert:Alert to:Handle` | G1 | CC | Transaction digests on; deadline reminders on (calendar-driven); a plain-sentence digest per G2/G3 change (§16.9) | M-120, M-125, M-126, M-127, C-05 | Sol K-016 · GLM K-042 |
| CM-052 | Official notices | What official notice or newsletter should owners see? | `notice.publish text:Text pin:Bool` · `newsletter.publish week:Week` | G1 | CC | Weekly newsletter in season; M-095 commissioner news and M-093 mail log are the owner-read surfaces (spec §15 ring-4 carry) | M-095, M-093, C-05 | Sol K-017 · GLM K-042 |
| CM-053 | What's New and league notes | What changed in the app or the league? | `whatsnew.post text:Text pin:Bool` | G1 | CC | App release notes post automatically; commissioner may pin league notes | M-141, C-05 | GLM K-045 |
| CM-054 | Data refresh and cache | Pull fresh data now, or clear a stale cache? | `data.refresh scope:Scope` · `cache.clear scope:Scope` | G1 | CC | Scheduled refresh; health on the status strip; lives in Control Room › Data and sources | S-09, M-137, C-07, S-12 | GLM K-044 |
| CM-055 | Engine parameters | Tune the valuation and market engine? | `engine.set param:ParamId value:Decimal` · `engine.revert to:Version` | G2 | C only | One commissioner-only slot; the parameter list is unknown and nothing is invented; versioned; lives in Control Room › Data and sources (triage take #6) | C-09, L-01, L-02, L-03, L-14, L-15 | GLM K-043 |
| CM-056 | Ledger (read) | What changed, who did it, and why? | `ledger.read filter:Filter` | G0 | Owner (filtered) | Every entry, append-only; owners see all league-state changes with who/what/why; role grants and invites hidden (ruling 5) | ledger-wide | triage settled #4 · GLM Step 4 |

Dormant owner-registry rows tagged `deferred` (CLI/chat refuses with a real message): pools and survivor
M-172, M-173, M-198; accounting M-098, M-197 (triage settled #10).

## Totals
56 human-facing controls (CM-001..CM-056) in 7 groups: A 17 · B 5 · C 4 · D 13 · E 6 · F 5 · G 6.
Typed verbs: 144 distinct (Sol's 104 atomic verbs plus succession, appeal, override and window additions).

**Group G SIGNED OFF 2026-10-13. Census (§16.7) complete.**
