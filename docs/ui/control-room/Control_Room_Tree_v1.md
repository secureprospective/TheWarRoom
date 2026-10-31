# Control Room: the complete tree (v1 + Sol triage, 2026-10-30)

Draft for Christopher, then Sol (organization review), then Claude triage. Built by Claude from the code on
`origin/main` 364379ba, not from memory. Every leaf cites where it comes from.

## 0. Sources (what was actually looked at)
| Source | What it gave |
|---|---|
| `internal/leaguerules` structs + `frontend/src/app/commissioner/rules.legacy.wire.testdata.json` | the **130 rule leaves in 19 sections**, labels, choices, Legacy values, "Needs a ruling" state |
| `frontend/src/app/commissioner/commands.wire.testdata.json` (Go `CommandSafeguards`) | the **92 commissioner commands** and their confirm level: c = confirm, h = hold-to-fire, r = typed reason |
| `docs/commissioner/Commissioner_Control_Census.md` | CM-001..056, plain questions, roles, defaults, signed off 10-13 |
| `docs/commissioner/README.md` | the MFL cut, deputy priority, Christopher's 10-13 answers |
| `internal/commissioner/jobs*.go` | job kinds that exist today: setup, rollover, contract season, DOT review, appeals, stat queue, seeding, rookie draft, playoffs, comms, waivers, bids |
| `internal/serverhttp`, `internal/accounts`, `deploy/server` | sign-in, TOTP, recovery codes, agent tokens, revoke, backups, restore, upgrade, status |
| `frontend/src/app/commissioner/*` | commissioner UI already built: LeagueLanding, JobsStrip, ControlSearch, RuleTray, HoldToFire, PreviewReview, UndoForm, LeaguePeople, LeagueOps, LeagueLedger, BackupNow, StartLeagueForm |
| MFL map `~/fleet/runs/bee-mfl-commish-map-2026-10-13/reports/C1,C4,C5` | MFL's option ranges (the floor, per the settings-breadth ruling) |
| Owner mocks: Home v5, War Room v6, Franchise HQ v10, League Pulse v14 | owner-side overlap (section 4) |

Legend per leaf: **B** = built in Go today · **R** = ruled, not built · **G** = gap found by looking (not built, not ruled) ·
confirm: c / h / r · role: C = commissioner only, CC = co-commissioner too (near-parity), DOT.

## 1. Motivations and purpose (why the Control Room is shaped this way)

**M1. No digging.** Christopher, 10-30: "They don't need to be searching through the depths of menus to
find something." MFL's commissioner office is 16 groups and 328 pages. Rule: **every leaf is reachable in
two clicks or one typed line.** Three doors to every leaf: the rail tree, the search box (finds any label,
value or plain question), the command line (`cap.amount 130M`).

**M2. Deepest sandbox.** Christopher, 10-30: leave "the sandboxing nature to the imagination of the
commish." The commissioner can set every value, override every automated outcome, and handle situations
nobody planned for (Damar Hamlin game cancelled, a feed outage, an owner who vanishes). Rule: **nothing
automated is final**; each automated outcome has a manual override that is itself a command, with a
reason and an undo (full-control ruling 10-13, `rule.exception`, `correction.revert`).

**M3. Jobs first.** The landing is what needs the commissioner now (K3 ruling). Settings are visited;
jobs arrive. A commissioner who opens the room should see "3 things need you" before any setting.

**M4. Organize by what the commissioner is touching, not by MFL's menus.** Rules (how the league works)
are separate from fixes (this one thing went wrong) and from people (who holds what). A commissioner
thinks "a stat is wrong" or "change the cap", never "Scores/Standings Adjustments group".

**M5. Situations, not just settings.** Real league issues arrive as stories ("a game was cancelled").
A Situations index maps each story to the exact tools, in order. This is where the sandbox stops being
scary.

**M6. Safety you can see.** Every change shows its preview (who it affects, does a result flip), its
confirm level (c/h/r), when it takes effect (now or next rollover) and lands on the ledger visible to
owners. Rules changed mid-season default to next rollover.

**M7. One action, one home; the owner screens keep their place.** Where an owner screen already shows
the thing (trades in War Room, standings in Pulse, lineup in HQ), the commissioner acts **in place** via a
commissioner layer on that screen's center card, and the Control Room lists the job and links there. No
second copy of a screen (section 4).

**M8. Co-commissioner near-parity.** Same room, same tree. The few C-only leaves show a lock and the
words "commissioner only".

**M9. Never invent.** Unset values say "Needs a ruling" (ruling). Gaps found below are flagged G, not
filled in.

## 2. Top-level organization (the trunk)

Control Room (rail node 5) has **seven branches**. Left rail = funnel tree (ruling): each branch expands
to its sections; clicking a section fills the work area; clicking a leaf opens it in the inspector or a
center card.

| # | Branch | Holds | Why separate |
|---|---|---|---|
| 1 | **JOBS** | what needs a decision now, upcoming, blocked | M3 landing |
| 2 | **SITUATIONS** | real-league stories → the tools for each | M5 sandbox map |
| 3 | **RULES** | the 130 rule leaves, 8 families | how the league works |
| 4 | **SEASON DESK** | running the year: calendar, waivers/bids runs, draft, playoffs, rollover | time-driven operations |
| 5 | **FIX** | corrections and overrides, by the thing that is wrong | one-off repairs |
| 6 | **PEOPLE** | seats, roles, abilities, succession, votes, agents | who holds what |
| 7 | **LEAGUE OPS** | comms, ledger, data feeds, server, backups, migration | the platform itself |

Always present on every branch: search box (top), command line (bottom tray), alert band (owners'
alerts ruling), preview tray for a pending change.

## 3. The tree, to the tips

### 3.1 JOBS
Source: `jobs.go`, `jobs_league.go`, `jobs_transactions.go`. States: live › upcoming › blocked › not built.

| Leaf | Purpose | Opens | Status |
|---|---|---|---|
| League setup | rules still "Needs a ruling" (count) | RULES filtered to unset | B |
| Trade review (DOT) | trades waiting for votes | trade center card (War Room layer) | B |
| Trade appeals | one appeal per party, commissioner rules | appeal card | B |
| Stat corrections queue | feed changed a stat; approve/reject (1–2 clicks) | stat card with result-flip check | B |
| Waivers run | claims at window close; manual run / order / award override | Season Desk › Waivers | B |
| Bids run | cleared players' bids; process / award override | Season Desk › Bids | B |
| Contract season | which spring window is open/next/undated | Season Desk › Calendar | B |
| Rookie draft | draft due / live / closing | Draft room tray | B |
| Playoff seeding | seeds ready to build | Season Desk › Playoffs | B |
| Playoffs | games to advance / override | bracket card | B |
| Rollover | dated → preview → run | Season Desk › Rollover | B |
| Communications | notices due (newsletter) | League Ops › Comms | B |
| Notifications & refresh schedule | CM-051/054 | — | B (shown as "not built") |
| Missed lineups | MissedFlag raises a team (rules) | FIX › Lineup | R (flag is a rule; job row G) |
| Soft-cap compliance | team over a soft cap, deadline | FIX › Money | R (rule exists; job row G) |
| Must-cut teams | team over roster limits; no auto-cut by design | team card | B (`must_cut.go`) |
| Vacant franchises | orphan team → caretaker seat | People › Seats | B |
| Pending invites | invites not redeemed | People › Seats | B |
| Open votes | vote-out / election running, tie-break | People › Succession | B |
| Inactivity watch | commissioner quiet past threshold | People › Succession | B |
| Feed health | stat/news feeds stale or failing | League Ops › Data | G (status exists; job row not) |
| Backup failed / overdue | last backup state | League Ops › Server | G (BackupNow exists; job row not) |
| Upgrade available | signed bundle ready | League Ops › Server | G |

### 3.2 SITUATIONS (the sandbox map)
Each situation is a card: what happened → steps → the exact commands, pre-filled where the context is
known. Commands all exist (B) unless marked. This is how a non-technical commissioner finds the deep
tools without knowing their names (CLI/chat-first ruling: the same list becomes the chat's playbook).

| Situation | Steps → commands |
|---|---|
| An NFL game is cancelled or postponed | stat queue hold · `commish.score.franchise` / `commish.score.player` deltas · `commish.standings.record.adjust` or tie · `rule.exception` · notice to the league |
| An NFL stat was corrected | Jobs › stat queue · `commish.stat.queue.approve/reject` · result-flip check · notice |
| The feed is wrong or silent | `commish.stat.correct` manual entry with source · feed health (League Ops) |
| Wrong position from the feed | `commish.player.position` / `.clear` |
| An owner vanished mid-season | `commish.seat.caretaker` · `commish.lineup.correct` (caretaker) · `commish.seat.invite` new owner · `commish.seat.replace` |
| An owner wants out | `commish.seat.remove` (team, contracts and picks stay) · caretaker · invite |
| A lineup was wrong because of the app | `commish.lineup.correct` + rescore |
| A trade looks like collusion | DOT review · `trade.override` (C, reason, visible) · appeal path |
| A trade was approved by mistake | `correct.transaction` (compensating entries) · `correction.revert` |
| A player was cut by mistake | `correct.roster` · `commish.contract.edit` · dead-cap `commish.cap.adjust` |
| Career-ending injury / suspension relief | `commish.cap.relief` (reasons from contracts.capRelief) |
| A player retires or dies | contracts.retirement / death rules apply; `commish.cap.adjust` if needed |
| A team is over the cap | soft-cap job · deadline · `commish.cap.adjust` / `rule.exception` |
| A team is over roster size | must-cut job (no auto-cut) · `commish.reserve.exception` |
| Someone needs IR/taxi outside the rules | `commish.reserve.exception` (time-boxed) |
| One action outside its window | `commish.calendar.exception` (one action, one team, time-boxed, reason, visible) |
| Waiver or bid awarded wrong | `waivers.award.override` · `bids.award.override` |
| A draft pick went wrong | Draft tray: `draft.change` · `draft.skip` · `draft.assign` · `draft.insert` · `draft.revert` · `draft.timer` |
| A pick is owned by the wrong team | `correct.pick` |
| Standings must be set by hand | `commish.standings.override` / `.clear` |
| A playoff game must be decided by hand | `bracket.advance` (reason) |
| A rule loophole shows up mid-season | `commish.rules.set` effective next rollover, or now with reason · `rule.exception` meanwhile · house rule text |
| A house rule exists only in words | House rules text · (parked: words → enforced rule, CLI/chat campaign) |
| The commissioner steps down / goes quiet | `commish.succession.stepdown` · takeover · `commish.deputy.order` |
| The league wants a new commissioner | `commish.succession.voteout.open` / `election.open` (CC or DOT) · `commish.vote.cast/close/tiebreak` |
| Undo anything | `commish.correction.revert` (compensating entry) |
| Expand / shrink the league | league.teams/conferences/divisions/alignment (next rollover) — **G: no guided flow** |
| Move from MFL | League Ops › Migration (`league.migrate.*`) |
| Bring in an AI helper | People › AI agents (scoped token) |

### 3.3 RULES (130 leaves, 19 code sections → 8 families)
Each leaf row in the UI: label · value · state (Set / Needs a ruling / Pending next rollover) · citation ·
changed-at · competitive flag · "governs" chips (owner screens it changes) · edit. Edit opens the
RuleTray with preview, effective timing (now / next rollover) and confirm. **All leaves below are B**
(exist in the catalogue) unless marked G.

**Family A · League & teams** (sections: League shape, Player pools)
- Teams · Conferences · Divisions · Competition format · First rosters · Division alignment · Conference of each division · Dynasty league · Roster carry policy · Keepers · Sides · Player pools
- G: Competition format offers only head-to-head; MFL also offers all-play and total-points (C1).
- G: First rosters offers draft / free agency; MFL also offers auction, draft + auction, manual load (C1). Manual load is covered by Migration; auction is not.
- Presets (apply a bundle): 32-Team Default · 16-Team Mirror · Legacy NFL · Your MFL league · Start empty (`presets.go`, B)
- League name (B, on create) · franchise names → People

**Family B · Rosters & lineups** (Roster, Lineup, IR, Taxi squad)
- Roster: Active roster size (min, max) · Reserves excluded from roster size · Roster position limits
- Lineup: Starting position limits (per position min/max, incl. OL) · Starting group limits (offense, defense, total) · Invalid lineup penalty · Flag missed lineups
- IR: IR spots · IR eligible designations · Leave IR when healthy · Automatically return ineligible IR players
- Taxi: Taxi spots · Taxi experience limit · Taxi experience measure · Automatically promote ineligible taxi players
- G: **Lineup lock** (weekly lock vs per-game kickoff lock). MFL has it (C4: weekly lock day/hour, game-time yes/no); our catalogue has no lineup-lock leaf.

**Family C · Scoring** (Scoring, Standings)
- Scoring values (per stat: label, group, used, points) · Round each stat's points to the cent · Extra points by position · Scoring ranges · Points by play distance · Stats counted as a total of others · Scoring rules by range and position · Team offensive line scoring (used, weights, labels, no-sack, sacks, clamp)
- Standings tie breakers (ordered list)
- Playoff home-field points lives in Playoffs (one home)

**Family D · Money** (Salary cap, Contracts)
- Cap: Salary cap · Cap enforcement (hard/soft) · IR salary share · Taxi salary share · Money rounding increment · Soft cap compliance deadline
- Contracts: Maximum new contract years · Flat annual salaries · Minimum bidding salaries (by experience) · Release dead cap rate · Restructured release dead cap rate · Waiver claim clears dead cap · Re-sign a released player this season
- Franchise tag: enabled · annual limit · top players · floor multiplier · consecutive limit · second multiplier · extension multiplier · exclude playoff contracts
- Extension: enabled · annual limit · minimum remaining · added years · total years · multiplier · position floors · repeat before free agency · UFA eligible
- Restructure: enabled · annual limit · per contract · extra per extension · tiers · conserve total · existing years only
- Buyout: enabled · annual limit · offseason only · one year allowed · rates by years · rebid next offseason
- RFA tender: enabled · prices · prior salary multiplier · match window · compensatory allowed
- RFA re-sign: enabled · premiums by year
- Retirement: enabled · rate · Death: enabled · rate · Cap relief appeals: enabled · reasons

**Family E · Player movement** (Waivers, Free-agent bids, Trades)
- Waivers: Waiver trigger (cut only, ruled) · Waiver period · Waiver priority (reverse standings / rolling)
- Bids: UFA bidding (blind/open) · RFA bidding · Bid year multipliers · Maximum active bids · One-year bid limit · Bid salary length limits (tiers) · Minimum bid increase · Time a bid must stand to win · Late bid threshold · Late bid minimum increase · Last multi-year bidding week · Playoff-only bidding after week · Playoff bid maximum years · Lift one-year bid cap in playoffs · Bid point decimal places
- R3 (10-30) is law here: a bid is binding; open bids total ≤ available cap; ends only when legally outbid. **G: no catalogue leaf states "bids count against cap"; it must be a fixed rule, not a setting** (ruling says "period").
- Trades: Trade review (none/DOT/commissioner) · Trade review seats · Votes to approve or veto · Future pick trading years · Tradable pick rounds · Refuse invalid rosters · Trade offer lifetime · Maximum trade parties · Trade compensatory picks · Trade salary cap space · Withdraw accepted trades

**Family F · Season shape** (Schedule, Playoffs, Calendar)
- Schedule: Regular season weeks · Schedule pattern (only "division weighted" today; G vs MFL generator)
- Playoffs: Teams per conference · Division winners per conference · Wild cards · First-round bye seed · Seeding · Reseed rounds · First playoff week · Final playoff week · Home field points · Bracket (graph; built graphically, presets)
- Calendar: League action windows (each: start, end, opens) · League timezone · Shared date boundary · Season rollover instant · Annual rollover date · Trade deadline · Trade deadline week

**Family G · Draft** (Draft)
- Rookie draft: pace (slow/live) · pick clock · expired clock (skip/autopick) · timeouts (unlimited/count) · rounds · maximum picks per team · forfeit extra picks · salaries by slot · contract years by round · RFA after by round · first-round order source · order overrides · later-round pattern · tie breaker
- Startup draft: rounds · first-round order (random/commissioner) · commissioner's order · later-round pattern · contract years · salary · incoming rookies in pool
- G: first-round order source offers only "standings"; MFL sets any order by hand (covered by overrides) but no lottery option exists. Flag, don't invent.

**Family H · League voice** (House rules, Notifications)
- House rules text (per id) · Tell the league about transactions · Remind owners of deadlines

### 3.4 SEASON DESK
| Section | Leaves | Status |
|---|---|---|
| Calendar | year view of every window, deadline, draft start, rollover; drag to move (`calendar.window.set/move/remove`); window exception | B |
| Waivers | current claims · order (`waivers.order.set/auto`) · run now (`waivers.process`) · award override | B |
| Bids | open auctions · process cleared (`bids.process`) · award override · O-Line starting auction close (`oline.auction.close`) | B |
| Draft | setup summary · order (`draft.order.*`) · start (`draft.start`) · startup open (`draft.startup.open`) · **live tray lives in the Draft room** (ruling) · revert · archive | B |
| Schedule | generate · edit a game · publish | B |
| Playoffs | build bracket from preset · edit node · seed slot · save preset · advance (override) · title label (`title.set`) | B |
| Rollover | preview (dry run) · run · closed books summary | B |
| Seed held league | `commish.league.seed_held` | B |

### 3.5 FIX (by the thing that is wrong)
All confirm = r (typed reason) unless noted; every fix shows a preview and is undoable.
| Thing | Leaves (commands) |
|---|---|
| A stat | approve/reject queue (h) · manual stat (h) |
| A score | player score delta · franchise score delta · result-flip warning |
| A lineup | lineup correct for a week |
| A player | position override / clear |
| A roster | add/remove (`correct.roster`) · reserve exception |
| A transaction | correct transaction · provenance |
| A pick | pick owner |
| Money | contract edit · cap adjust · cap relief |
| Standings | record adjust · full order override / clear |
| A rule outcome | rule exception (scope, until) · calendar exception |
| A trade | override DOT · rule appeal |
| Undo | revert any ledger entry (compensating) |

### 3.6 PEOPLE
| Section | Leaves | Role |
|---|---|---|
| Franchises & seats | franchise name/abbr · invite (single-use) · revoke invite · remove owner · replace owner · caretaker seat | CC (remove/replace = r) |
| Roles | grant/revoke commissioner, co-commissioner, DOT · deputy priority order | C only |
| Abilities & visibility | per-team or all abilities (lineups, add/drop, trades, IR, taxi, board, chat…) · who can read which report | CC |
| Succession | step down (named successor) · inactivity thresholds (in/off season) · periods · takeover · accept/decline · vote-out · election · cast/close/tiebreak votes | per census |
| Sign-in help | recovery codes · revoke sessions · **commissioner resets an owner's authenticator (ruled 10-30, R; no code found)** | CC |
| AI agents | "Connect an AI agent" wizard: job, scope, expiry, token, status, revoke (agent-token endpoints B; wizard R) | C |
| Support / operator roles | R9 support, R10 server admin (ruled; G in code) | C |

### 3.7 LEAGUE OPS
| Section | Leaves | Status |
|---|---|---|
| Comms | official notice (pin) · newsletter · league notes · What's New · digest of changes · mail log | B |
| Ledger | every change, who/what/why, filters, "via AI" tag; owners see it filtered | B |
| Data & feeds | refresh now · cache clear · feed health · score truth ladder state (ESPN → nflverse) | B (health view G) |
| Engine | valuation parameters (C only; list empty, nothing invented) | B shell |
| Server | status · backup now · backup history (R2) · restore · upgrade (signed bundle) · version gate | B (scripts) / UI partly G |
| Migration | prepare · import · review · invites · cutover (`league.migrate.*`) | B |
| Leagues | create league (preset) · league picker | B |

## 4. Owner-side overlap (built mocks) and where each action lives
Rule (M7): the owner screen keeps the view; the commissioner gets a **layer** (extra buttons on that
screen's center card, gated by role); the Control Room job links there. No second screen.

| Thing | Owner screen (mock) | Commissioner action lives | Control Room shows |
|---|---|---|---|
| Trades | War Room · trade center card | same card + DOT vote / override / appeal | Jobs row → card |
| Bids & waivers | War Room · Market | same card + award override | Jobs row; Season Desk run controls |
| Lineups | Franchise HQ lineup | HQ lineup + "correct this week" (commish layer) | Fix › Lineup; missed-lineup job |
| Roster / contracts / cap | Franchise HQ | HQ player card + contract edit / cap adjust | Fix › Money |
| Standings / race | League Pulse › Race | Pulse standings + override (commish layer) | Fix › Standings |
| Playoff bracket | League Pulse | bracket card + advance | Season Desk › Playoffs |
| Live scores | League Pulse › Live | game card + score delta | Fix › Score |
| Archive / trophies / titles | League Pulse › Archive | trophy card + title label, history link | Season Desk › Playoffs (title) |
| Transactions feed, WIRE tags | Home › Transactions | transaction card + correct | Fix › Transaction |
| NFL stats, career card | Home › NFL stats | career card + stat correct / position override | Fix › Stat / Player |
| News | Home › News | — (commish notices are League Ops › Comms, show in Home feed) | — |
| Alerts band | all screens | commish alerts route there too (CM-051) | Jobs mirrors |
| Draft | Draft room (not mocked yet) | commissioner tray in Draft room (ruling) | Jobs row + Season Desk › Draft |
| Rulebook | owners read rules (rules are public) | edit in Control Room | Rules |

Duplications found in built code to resolve: `Commissioner_Owner_Overlap.md` maps 328 MFL rows to owner
nodes; this tree's overlap table replaces it at the screen level.

## 5. Gaps found by looking (G) — for triage, not filled
1. Lineup lock leaf missing (MFL weekly/game-time lock).
2. Competition format: only head-to-head (MFL: all-play, total points).
3. First rosters: no auction option (MFL has it).
4. Schedule pattern: one option.
5. Draft first-round order: one source (standings).
6. R3 bid-cap binding: must be a fixed rule, verify the engine enforces "open bids ≤ available cap".
7. Commissioner authenticator reset: ruled 10-30, no code.
8. Support / operator roles R9/R10: ruled, no code.
9. Job rows missing: feed health, backup overdue, upgrade available, soft-cap compliance, missed lineups.
10. No guided flow for expanding/shrinking the league.
11. Flagship rulings with no Control Room home yet: decision windows (labels per team, owner-side), new-owner kit (People › Seats → invite should attach it), custom trophy case (Pulse), progress bars, rebuild coach — owner features; only the new-owner kit touches the commissioner.

## 6. Sol review and Claude triage (2026-10-30)
Sol 6.1 review: Beelink `~/fleet/runs/sol-uiux/stdout.log` (read-only, `origin/main` 364379ba). Each
point was checked by Claude in the code before it was accepted.

### Accepted (verified in code)
| # | Finding | Verified at | Change to the tree |
|---|---|---|---|
| T1 | Owners can withdraw a standing bid: `withdraw()` only checks the bid is theirs and live | `internal/commissioner/market/bid_execute.go:136` | **Conflicts with R3** (a bid ends only when legally outbid). Engine fix, logged as a ruled bug, not a question |
| T2 | Jobs don't emit vacant franchises, pending invites, open votes or inactivity | `jobs_league.go` emits only seed, stat_queue, in_season, communications, rookie_draft, dot_review, playoffs, waivers, bids | Those four rows are relabelled **G** (job row missing) |
| T3 | Built jobs missing from the tree: Record starting assets, In-season operations, startup draft, O-Line must-cut | `jobs_league.go`, `draft_startup.go`, `oline_close.go` | Added to Jobs |
| T4 | Search matches control names, questions and rule labels, not current values | `ControlSearch.tsx:7` | M1 corrected: search finds labels and questions; value search is **G** |
| T5 | "Undo anything" is false: MFL cutover is permanent | `migration.go:49-54` | Undo reads "every change except cutover, which says so before it runs" |
| T6 | Missing leaves: accept invitation, record activity, step aside from review (`trade.recuse`), `contract.action.set` alias, withdraw notice, local sign-in after restore | `commands.wire.testdata.json`, `LeagueOps.tsx`, `LocalSignIn.tsx` | Added to People, Fix and League Ops |
| T7 | Unbuilt per the controls fixture: titles (CM-048), history link (CM-049), engine parameters (CM-055); awards (CM-050) deferred | `controls.wire.testdata.json` | Status split into three columns: **command / screen / workflow**; no blanket B |
| T8 | Two-click budget fails: cap rule = 4+ clicks, five common tasks = 3 | tree §2–3 | **Navigation rule changed:** a branch opens straight onto ready launchers (the top tasks of that branch); rule families are **filter chips** on one Rules list, not a level of the tree. Cap rule: Rules → type/chip "cap" → edit = 2 |
| T9 | Rename families: C Scoring & standings, D Cap & contracts ("Money" reads as dues, which is retired), H Rulebook & notifications | rules fixture + README | Renamed. Cross-links (IR/taxi salary share from Rosters, RFA from Player movement, deadline from Trades) are aliases to one home |
| T10 | Sides is derived, never set | `constitution.go` comment | Shown read-only with "counted from conferences" |
| T11 | Placement: seed held league → League Ops › Leagues; titles + history link → Pulse Archive with a commissioner layer | `migration.go`, census CM-048/049 | Moved |
| T12 | Situations must say what is still manual, and claim no command that doesn't exist ("stat queue hold" has none) | command fixture | Each situation gets three lines: tools, still manual, check afterwards. "Hold" removed; marked **G** |
| T13 | New situations: commissioner is a trade party (recuse → co-commish appeal), tied blind auction, startup draft stalls, invite won't redeem, wrong historical franchise | command fixture | Added |
| T14 | Real gaps: no pick-forfeit (penalty) command; no RFA match-dispute command | command fixture (verified: no such verb) | **G**; commissioner can only fix through `correct.pick` / contract edit today |
| T15 | Deep links from a Job to an owner screen must carry league, season/week, entity, permission and a return path; blind bids stay private even to the commissioner (R2 ruling) | `HostedWorkflows.tsx` | Added to the M7 overlap rule |
| T16 | Server admin and league admin are separate permissions (R10 operator ruling agrees) | ruling R10 | League Ops splits into **League** (comms, ledger, data, migration, leagues) and **Server** (status, backups, restore, upgrade; operator role) |

### Rejected or qualified
| # | Sol said | Why not taken |
|---|---|---|
| X1 | Escalate the authenticator reset: `internal/accounts/README.md:102,149` says no commissioner-reset door | The README predates Christopher's **10-30 ruling** (TOTP opt-in; the commissioner can reset an owner's authenticator). The ruling stands; the README describes what was built before it. Kept as **R, unbuilt**, its own slice |
| X2 | Open bids aren't totalled against cap (G6) | Partly checked: `nativecheck/bids.go:186` says cap is "judged by the caller". The caller wasn't traced. **Lead, not finding**: verify when T1 is fixed |
| X3 | G3 auction "too broad" because free-agency start uses bids | Accepted as qualified: a free-agency start is a bid auction. The remaining gap is MFL's draft + auction mix only |
| X4 | Awards entry in Archive | Kept as deferred with the side-games suite (ruling), shown honestly as "later" |

### Questions only Christopher can answer
Q1. Pick forfeiture as a penalty (T14): should the commissioner get a "take away a pick" penalty command,
    or are penalties out of scope?
Q2. Breadth (standing ruling: open rigid settings on sight). Confirming the order for the build backlog:
    lineup lock first (it's missing entirely), then all-play / total-points formats, then schedule patterns,
    then a rookie draft lottery option. Any of these you do NOT want?
