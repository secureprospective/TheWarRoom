# TheWarRoom

TheWarRoom is a dynasty fantasy football platform that houses a league's rules, rosters, contracts, salary cap, and permanent history, either imported from MFL or governed natively on a hosted server.

## Language

### League & Governance

**League**:
A dynasty fantasy football competition with its own franchises, rules, calendar, and permanent history.
_Avoid_: Instance, workspace

**Franchise**:
One of the persistent teams in a league, holding a roster, contracts, salary cap, and draft picks across seasons independently of who owns it.
_Avoid_: Club, squad, organization

**Seat**:
The single ownership slot attached to a franchise that grants one person authority to run that team.
_Avoid_: Slot, membership

**Owner**:
The person who currently holds a franchise's seat and makes its lineup, contract, draft, and market decisions.
_Avoid_: GM, manager, user, customer

**Caretaker**:
A commissioner temporarily running a vacant (orphan) franchise to set its lineups until a new owner claims the seat.
_Avoid_: Interim owner, proxy owner, bot

**Stand-in**:
An AI that holds an empty seat and plays the franchise as an owner would (lineups, bids, trades) until a real person claims it; test leagues only for now.
_Avoid_: Bot, AI owner, autopilot, ghost owner

**Commissioner**:
The league's head administrator with full authority over rules, seats, roles, calendar windows, and manual corrections.
_Avoid_: Admin, superuser, host

**Co-commissioner**:
An administrator with full commissioner power and a place in the succession order (co-commissioner order) who steps in automatically during commissioner inactivity, recusal, or step-down.
_Avoid_: Vice-commissioner, assistant commissioner, deputy priority, deputy order

**Deputy**:
A league member granted specific delegated permissions only (such as approving stat fixes, ruling on trades, editing lineups, moving picks, changing rules, or running rollover).
_Avoid_: Moderator, helper

**DOT (Department of Trades)**:
The committee of league members seated to vote on and review pending trades for collusion or rule violations.
_Avoid_: Trade committee, review board

**Admission**:
A person's verified membership and role in a league—obtained by redeeming a private invitation or claiming an open team via a join link—that lets them enter the league.
_Avoid_: Enrollment, onboarding, registration

**Twin-pool league**:
A league format with two conferences where each conference has its own independent copy of every NFL player.
_Avoid_: Mirror league, 16-Team Mirror, per-side league

### Migration & League Lifecycle

**Cutover**:
The one-time, irreversible act that moves an imported MFL league onto TheWarRoom as its authoritative home so the league no longer follows MFL automatically.
_Avoid_: Migration finalize, switch-over, activation

**Mirror**:
The read-only copy of an MFL league's rosters, contracts, salary adjustments, picks, and schedule that a load or refresh reads from.
_Avoid_: Cache, replica, shadow league

**Native league**:
A league governed by TheWarRoom after cutover (or created natively) rather than by an external host.
_Avoid_: App league, local league

**Held league**:
A league or asset snapshot whose starting rosters, contracts, picks, and history were imported from an external host (MFL) rather than started from scratch.
_Avoid_: Imported league, legacy mode

**Ledger**:
The league-wide, append-only history of every rule change, roster move, trade, draft pick, and commissioner correction, recording who did it and why.
_Avoid_: Audit log, event stream, history table

**Vacate seat**:
Free an owner's seat on a franchise so the team becomes open for a caretaker or a new owner.
_Avoid_: Release a seat, seat release

### Rosters, Contracts & Assets

**Roster**:
The group of players currently held by a franchise across its active lineup/bench, injured reserve, and taxi squad.
_Avoid_: Team list, player pool

**Taxi (Taxi Squad)**:
A reserve group on a franchise's roster for developing eligible young players who cannot start until promoted to the active roster.
_Avoid_: Practice squad, minors, farm

**IR (Injured Reserve)**:
A reserve group on a franchise's roster for players with qualifying real-world NFL injury or suspension designations, freeing an active roster spot.
_Avoid_: Disabled list, medical reserve

**Contract**:
The multi-year agreement binding a rostered player to a franchise at a fixed annual salary through an expiration season and ending status (such as RFA or UFA).
_Avoid_: Deal, agreement

**Salary adjustment**:
A commissioner-recorded or imported charge or credit applied directly to a franchise's salary cap for a season outside of active player contracts.
_Avoid_: Cap tweak, manual fee, fine

**Cap (Salary Cap)**:
The maximum total money a franchise may commit to player salaries and dead cap in a season under the league's rules.
_Avoid_: Budget, spending limit

**Dead cap**:
Salary cap money charged to a franchise's books for players no longer on its roster (for example, 35% of salary left behind when a contracted player is cut).
_Avoid_: Dead money, buyout penalty, sunk cost

**Cap schedule / Dead-cap schedule**:
A franchise's multi-year table of committed player salaries and dead-cap charges across upcoming seasons.
_Avoid_: Salary ledger, dead-cap ledger

**Cut**:
Drop a player from a franchise's roster.
_Avoid_: Release a player, drop

**Pick (Draft Pick)**:
A tradable right belonging to a franchise to select a player in a specific season, round, and slot of the rookie or startup draft.
_Avoid_: Selection token, draft asset

### Schedule, Rules & Operations

**Window**:
A named date range on the league calendar that opens or closes specific league actions (such as free-agent bidding, trading, re-signing, or cuts).
_Avoid_: Phase, period, stage

**Calendar**:
The season schedule of action windows and deadlines that governs when owners and commissioners may perform specific moves.
_Avoid_: Timeline, schedule of events

**Rules**:
The league's enforceable settings governing roster limits, starting lineups, scoring, standings tiebreakers, salary cap, contracts, waivers, bids, trades, and playoffs.
_Avoid_: Config, preferences, parameters

**Situation**:
A step-by-step playbook card in the Control Room that guides the commissioner through handling a real-world edge case (such as a cancelled NFL game, vanished owner, or mistaken cut) and links directly to the tools needed.
_Avoid_: Playbook, runbook, wizard, guide

**Docket**:
The cross-league queue of active items grouped into three lanes—**Needs you**, **Waiting on**, and **Landed**—showing who holds the ball on every pending action, offer, bid, claim, or commissioner job.
_Avoid_: Inbox, notification center, feed, task list

**Needs you**:
The highest-priority lane of the Docket listing time-sensitive actions where the signed-in owner or commissioner must act next (such as an unanswered trade offer, missed lineup, closing auction bid, or blocked commissioner job).
_Avoid_: Alerts, to-do, action required

### Product Surfaces & Environments

**Pulse (League Pulse)**:
The league-wide live scoreboard and season race screen showing weekly matchups, followed NFL game play-by-play, standings, playoff bracket, and championship odds.
_Avoid_: Scoreboard, match center

**HQ (Franchise HQ)**:
An owner's team management screen for setting starting lineups, managing active/IR/taxi spots, tracking multi-year contracts and cap runway, testing what-if cap moves, and reviewing team moves.
_Avoid_: My Team, clubhouse, dashboard

**War Room**:
The market and acquisition screen where owners evaluate the player pool, place blind or open free-agent bids, submit waiver claims, negotiate and evaluate trades on the trade desk, and make draft picks.
_Avoid_: Market, trade center, draft hub

**Trade desk**:
The War Room workspace for building and reviewing trade offers with **You send** and **You get** columns.
_Avoid_: Trade ledgers, trade slates

**Control Room**:
The commissioner-only workspace for running the league across ten tools: Jobs, Situations, Rules, Season, Fix, Picks, People, Feedback, League notices/ledger, and Server backups.
_Avoid_: Admin panel, commish office, settings

**Thin client**:
The desktop or browser app mode that connects to a hosted server without keeping a local league database on the device.
_Avoid_: Native mode, hosted mode, cloud mode, native client

**Hosted server**:
The always-on league server accessible over HTTPS where a native league runs so all owners and the commissioner share one live, authoritative league.
_Avoid_: Remote, cloud, SaaS

**Lab**:
The test bench (Proxmox CT 260) where every new build lands first. Disposable: it may be wiped or
reseeded at any time, and holds a copy of the Legacy NFL league for rehearsals, stress runs and UI walks.
It is never the live league host.
_Avoid_: Staging, sandbox, dev server, prod, test server, laptop

**Season server**:
The always-on machine (Alpine, 192.168.1.162) that runs the two rehearsal leagues as a real league's
host. Frozen from one Saturday swap to the next; changed only on Saturday night with Christopher's OK;
its league data is never reset.
_Avoid_: prod, test server, laptop

**Release**:
A signed software build of the server or desktop application verified through the offline TUF trust ceremony before installation or upgrade (distinct from cutting a player or vacating a seat).
_Avoid_: Build, drop, deployment
