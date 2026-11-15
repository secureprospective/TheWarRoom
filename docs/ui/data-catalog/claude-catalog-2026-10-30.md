# Data-stub catalog — Claude pass (2026-10-30)
Format: ID · Screen › Region › Element — shape — R/W — refresh — variants. Source column filled in pass 2.

## SHELL (shared rail, all screens)
SH-001 Rail head › league name ("Legacy NFL") — league.name — R — per-load
SH-002 Rail head › phase line "Wk 9 · 3 live" — {season_phase, week, live_game_count} — R — live
SH-003 Rail › MY SHORTCUTS list — per-user list<{label, target_route}> ordered; pin/unpin, drag-reorder — R/W — per-user pref
SH-004 Rail › node unread mark (League Pulse dot) — per-user unread flag per area — R — on-event
SH-005 Rail › NOW leaf (context deadline: "Locks Sun 12:00" / "Tag window shuts in 12d" / "2 moves need you") — next-deadline per workspace — R — live/clock
SH-006 Rail › ☆/★ pin state per leaf — per-user pins — W
SH-007 Rail › Control Room node visible only to commish/co/DOT (role) — viewer.roles — R
SH-008 Rail foot › "/ command" bar — command registry (names, args, help) — R; executes commands — W
SH-009 Customize mode toggle — UI-only, persisted shortcuts — W

## HQ — Franchise HQ v10
### Header / plate
HQ-001 Tabs › "MY MOVES" queue chip counts: need-you / waiting / landed — counts from move envelopes by state — R — on-event
HQ-002 Tabs › command hint "lineup.swap out=… in=…" — command registry entry — static
HQ-003 Plate › team name — franchise.name — R
HQ-004 Plate › record "6–2", standing "2nd NFC West" — standings{W,L,T,division_rank, division_name} — R — per-week
HQ-005 Plate › active count "41 / 44" — roster active count, active limit (rule) — R
HQ-006 Plate › CAP year label, room, cap max — cap{season, committed, dead, room, max} — R
HQ-007 Plate › cap bar segments + key (committed / dead / room) — same as HQ-006 — R
HQ-008 Plate › LINEUP CHECK status (n empty · n bye · n questionable / "Lineup is full") — derived from lineup + player status/bye — R — live
HQ-009 Plate › "Locks in 2d 05h" — lineup lock time (rule: lineup lock, PLANNED) vs now — R — clock
### Lineup workspace
HQ-010 STARTING 19 total proj — sum of slot projections — derived — live
HQ-011 OFFENCE · 8 / DEFENCE · 11 counts + sums — lineup slot config (rule: starters by position) + projections
HQ-012 Slot row: slot code (QB/RB/WR/FLEX/OL/K/DT/DE/LB/CB/S/D·FLX) — league lineup slot definitions incl. flex eligibility map — R — rule
HQ-013 Slot row: position tag (draggable) — player.position — R
HQ-014 Slot row: player name + NFL team — player{name, nfl_team} — R
HQ-015 Slot row: form sparkline last 5 weeks, per-week pts, bye weeks hatched, projection line, avg — player weekly fantasy points (league scoring) ×5 + bye map + projection — R — per-week
HQ-016 Slot row: opponent ("@NYG", "BYE") — NFL schedule per week — R
HQ-017 Slot row: projection — player weekly projection (league scoring) — R — daily/live
HQ-018 Slot row: status LIVE/Q/BYE/OUT — game state + injury designation + bye — R — live
HQ-019 Empty slot "OPEN" + "drop any defender" eligibility hint — slot eligibility — R
HQ-020 Lineup edits: drag/pick-then-slot, cascade, auto-fill; change list in tray — lineup draft (unsaved) — W(local) then save — "CHECK AND SAVE PLAN" writes a lineup move envelope
HQ-021 Eligibility map (pos→slots) — rule data — R
HQ-022 Matchup header "VS OWNER 14 · IRON CURTAIN" — schedule matchup {opp franchise id, name} — R
HQ-023 Matchup: us proj / them proj — both lineups' projections (opp lineup visible) — R — live
HQ-024 Win probability % + bar — derived from projections/variance — NEW model — live
HQ-025 "3 of yours live now" — count of players in live games — R — live
HQ-026 Matchup rows slot-by-slot (ours vs theirs, diff, closest first) — both lineups per slot + proj — R — live
HQ-027 BENCH count — roster minus starters — R
HQ-028 Bench filter chips by position (ALL + 11 positions) — position list (league positions) — R
HQ-029 Bench row: pos, name, team, form, opp, proj — same as slot row — R
HQ-030 Inline taxi/IR mini-lists under bench (fit-to-space) — taxi + IR lists — R
### Contracts and cap workspace
HQ-031 CONTRACTS · 41 count — contract count — R
HQ-032 Legend RFA/UFA — contract end type — R
HQ-033 Contract table: POS, player, NFL team — R
HQ-034 Contract table: salary per year 2026–2029 (4 cols), final year marked with RFA/UFA end marker — contract{salary_by_season[], end_season, end_type rfa|ufa} — R
HQ-035 Contract table: DEAD IF CUT — dead-cap calc (rule: cut penalty %) per contract — derived — R
HQ-036 Contract table: VALUE — player dynasty value (valuation engine) — R — daily
HQ-037 Contract table: $/VALUE efficiency with ok/warn colour — derived — R
HQ-038 Contract table sort (2026 ▼) — UI
HQ-039 CAP RUNWAY: per future season committed %, dead %, room $ — cap projection by season ×4 — R
HQ-040 DECISIONS DUE · n: player, reason text (ends year, value, positional comparison "cheaper than DE going rate", "4th-best LB"), clock ("Tag window shuts in 12d"/"After season"), allowed ops EXTEND/TAG/LET WALK/RELEASE NOW — expiring contracts + window calendar + market rate by position + ops allowed by rules — R; ops W — per-load
HQ-041 DEAD CAP total + rows: player, how (Released Wk 7/Traded 2025), amount, through-season — dead-cap ledger entries — R
### Release drawer
HQ-042 Release plan title — player
HQ-043 FA end type "ends 2027 as RFA" — contract — R
HQ-044 Salary this year / years left / dead now / dead next yr / room after / roster after — cut calc (rule: dead-cap schedule across years) — derived
HQ-045 Cap room before/after bars 2026, 2027 — derived
HQ-046 YOUR LINEUP impact: slot, best replacement, proj delta — lineup + bench projections — derived
HQ-047 BEFORE YOU CUT alternatives: trade (n teams list pos as a need, which owners, value→"trades for a late pick"), ride it out (cost remaining, end type), taxi eligibility (NFL seasons) — team needs model (NEW), pick-value equivalence (valuation), taxi eligibility rule — R
HQ-048 PREPARE RELEASE → hold-to-confirm → hands off to MFL; creates move envelope — W (envelope) — on-event
HQ-049 TRY IN WHAT-IF → opens what-if with this move — W(local)
### My moves workspace
HQ-050 Lane counts PLANNED / HANDED OFF / LANDED / NEEDS YOU — envelope counts by state — R
HQ-051 Move card: kind (SIGN/LINEUP/WAIVER/TRADE/IR/TAXI/BID/RELEASE), description, check text ("Cap ok · roster ok · window open", "Owner 14 reviewing · DOT next", "Seen in MFL rosters 09:12", "Cap changed since you planned it"), state (Ready/Blocked/Not yet done/Awaiting DOT/Landed/Not verified/Stale), action (OPEN MFL/FIX/RE-OPEN/VIEW/CHECK/RE-PLAN), "prepared today" — move envelope{kind, payload, validation results, state, state_reason, created_at, verified_at, source evidence} — R/W — on-event
HQ-052 Trade envelope DOT review stage — trade review state — R
HQ-053 SEASON LEDGER · 38 MOVES: kind, when (Wk · day time), move text, cap effect (incl. dead), result (Landed/Outbid with winning price/Missed lock with minutes late/Declined/DOT passed) — franchise transaction history merged with envelope outcomes — R
### Tray + drawers
HQ-054 Tray message (changes list / per-tab hint) — local
HQ-055 DISCARD / CHECK AND SAVE PLAN / OPEN MFL ↗ — save lineup plan (W), MFL deep link URL per league/franchise
HQ-056 TAXI pull "7/8" — taxi count / taxi max (rule) — R
HQ-057 IR pull "3/12" — IR count / IR max (rule) — R
HQ-058 Taxi drawer table: pos, player, NFL team, NFL YR, taxi years left (pips, 3-yr rule), salary, value, NOTE (advice: "Promote now: beats McMillan (7.9) at WR", "Last taxi year after 2026"), PROMOTE button — player nfl_experience + taxi rules + advice engine (NEW) — R/W
HQ-059 Taxi summary: slots used/max, active/max, taxi pay, open slots — note "taxi pay is off the active cap" (rule) — R
HQ-060 COULD STASH list: active players still taxi-eligible, yr — derived from experience+rule — R; TO TAXI W
HQ-061 IR drawer table: pos, player, team, injury body part, expected back (Wk/Season), weeks until, salary, note ("Activate needs 1 active spot", "Done for 2026 · counts on cap"), ACTIVATE — injury feed {designation, body_part, expected_return} (NEW: return week) — R/W
HQ-062 IR returns timeline weeks 9–17 with names — derived — R
HQ-063 IR summary: slots, active, IR pay on cap, next return — R
HQ-064 IR CANDIDATES: Q players, injury, advice ("Can move to IR if ruled OUT", "Watch Friday report"), TO IR — injury status + IR eligibility rule — R/W
HQ-065 What-if list of candidate moves with cap deltas (2026, 2027), active delta, proj delta: CUT/TAXI/PROMO/SIGN (best FA at need pos)/IR/EXT (extension cost)/TRADE — suggestion engine (NEW) + FA pool + extension pricing rule — R
HQ-066 What-if summary: room 2026, room 2027, active/limit with over-limit warn, week proj — derived
HQ-067 What-if SEND TO MY MOVES / CLEAR — creates envelopes — W
### Inspector (player)
HQ-068 Position, live game state ("On field · Q2") — live game feed — live
HQ-069 Name, NFL team, salary, contract end, end type — R
HQ-070 pts/game, dynasty value — season stats + valuation — R
HQ-071 THIS WEEK: opponent, projection, target share last 4, matchup vs position rank ("Soft · 24th") — advanced stats (target share) + defense-vs-position ranks (NEW) — R
HQ-072 CONTRACT: salary by year, dead if cut today — R
HQ-073 AGE ARC (MINE): age, position age curve bars, peak window years — age-curve model by position — R
HQ-074 MY NOTES (MINE): private per-user per-player notes, count — NEW private notes store — R/W
HQ-075 ACT: BENCH / TO IR / ADD TO TRADE / WATCH — W (lineup, IR, trade builder, watchlist NEW)
HQ-076 Footer data freshness "MFL · 12 min ago" — last sync time per source — R
HQ-077 + ADD DRAWER — user-customizable inspector drawers (per-user pref) — W

## WR — War Room v6
### Header (shared across tabs)
WR-001 Rail NOW leaf "Deadline in 12d" — trade deadline from league calendar — R — clock
WR-002 Command hint "trade.build with=14 send=… get=…" — command registry
WR-003 Team name, record, standing, contention window "window 2026–27" — franchise + standings + window model (NEW: derived contention window) — R
WR-004 DIRECTION CONTEND/RETOOL/REBUILD toggle — per-franchise posture (owner-set, NEW store; drives "your lens") — R/W
WR-005 BEST 21 score + league rank "3rd of 32" — valuation Best-21 per franchise, all franchises ranked — R — daily
WR-006 VALUE BY AGE histogram (age buckets 22–32+) with peak bars — roster value aggregated by age bucket — derived
WR-007 NEEDS · NEXT SEASON per position (ok/thin/hole) — team-needs model (NEW) — R
WR-008 CAP RUNWAY 2026–2028 bars + room — cap projection by season (live-updated by deal in desk) — R
### Trade desk
WR-009 MY SHELF assets (players + picks) w/ filter ALL/OFF/DEF/PICKS — my roster + my picks — R
WR-010 Asset block: pos/PK, name, salary or pick note ("proj #27", "unprojected"), age, value (my lens) — player{pos,name,age,salary,contract_years,end_type} + pick{season,round,projected_slot} + lens value — R
WR-011 Used marker (already in deal) — local
WR-012 Version tabs A/B/C, + VERSION, "unsent · autosaved · only you see this" — private trade drafts per user per partner (NEW store, autosave) — R/W
WR-013 YOU SEND / YOU GET trays: object count, rows w/ age, YOU value, THEM value, PLAYERS/PICKS subheads, totals in both lenses — deal draft + both lens values — derived
WR-014 YOUR LENS / THEIR LENS bars + numbers + lens caption ("contend: now beats later", "rebuild: youth and picks") — lens valuation model keyed on each franchise's direction (NEW: posture-adjusted value) — derived
WR-015 Verdict EMPTY/BOTH WIN/THEY'LL BALK/BAD FOR YOU + sentence — derived
WR-016 Effect: CAP ROOM AFTER (+was), ROSTER n/44 legal/over, BEST 21 after + rank change, PROJ / WEEK delta ROS, LINEUP CHANGES (slot, out, in, delta) — cap, roster limit rule, Best-21 ranks, optimal lineup recompute — derived
WR-017 TUNE IT suggestions TRIM/ASK/SWEETEN with resulting lens numbers, APPLY — suggestion search over both rosters — derived (NEW engine)
WR-018 Ghost tick on lens = last round's position (negotiation mode) — thread round history — R
WR-019 Partner selector (list of other owners "Owner 14 · Iron Curtain") + their direction badge — franchises + posture — R
WR-020 Partner read: record · cap room, needs, spare positions, lean text ("gain from players 25 and under… vets over 29 worth ~60%") — partner standings, cap, needs model, surplus model, lens params — R (NEW for needs/spare/lean)
WR-021 THEIR SHELF assets — partner roster + picks — R
WR-022 Send gating: cap <0, roster >44, empty sides disable SEND — rules — derived
WR-023 SEND TO OWNER 14 — creates trade offer (W) — on-event to partner
### Tray / pocket / negotiations
WR-024 Tray msg ("Version A unsent. Cap after the deal $0.40M: tight but legal") — derived
WR-025 POCKET count; pocket cards: title, partner, send legs, get legs, status STALE/STILL GOOD/CHANGED + reason ("Bayou traded that pick on Tue", "Hamilton went Q (ankle) Wed", lens numbers), SAVED — saved private trades + re-validation against injuries/cap/other deals/asset ownership (NEW recheck job) — R/W
WR-026 Load pocket card as new version (drag/click), stale refusal — W(local)
WR-027 NEGOTIATIONS count "2 YOUR TURN" — threads where it's my turn — R — on-event
WR-028 Negotiation card: partner, ROUND n, title, legs, last note quote, YOUR TURN/THEIR TURN, expires in — trade thread{rounds[{who, at, send, get, note}], turn, expires_at} (NEW: counters/rounds/expiry/notes) — R
WR-029 Thread on desk: round strip R1..Rn (YOU OFFERED/THEY COUNTERED + time), turn, expiry, ◂ MY VERSIONS, "OWNER 14 SAYS" quote, my note input — R/W
WR-030 Thread diff marks THEIR ADD / YOUR CHANGE on tray rows — derived from rounds
WR-031 DECLINE / COUNTER / SEND COUNTER / HOLD TO ACCEPT — trade.respond writes (decline, counter w/ note, accept) — W
WR-032 Accept → "ACCEPTED · PROCESSING… rosters and cap update when league processes" (+ DOT review path from HQ) — W
### Calendar strip (bottom)
WR-033 Events: week/date label, name, countdown, hot/now state, due-count badge, pinned plan/version chip — league calendar events (deadline, playoffs, contract decisions, RFA/UFA tags, FA open, rookie draft w/ my pick count, cutdown) + my due items + my plan pins — R
### Plans
WR-034 PLANS list (private): name, description, ACTIVE, steps count, checked count — private plans store (NEW) — R/W
WR-035 + New plan (empty or copy active) — W
WR-036 MY NEEDS · RANKED (drag reorder) — owner-ranked needs (NEW, seeded from needs model) — R/W
WR-037 Plan columns per event (DEADLINE Wk 11 · 12d, CONTRACTS Feb 15, TAGS Mar 1, FA Mar 15, DRAFT Apr 30, CUTDOWN Aug 28) — calendar — R
WR-038 Step: checkbox, kind (TRADE/RE-SIGN/RELEASE/TAG/BID/PICK/CUT), text, link to desk/market/draft object ("↪ DESK: Version A"), fallback note, window-warning "usually not possible at this event" — plan step{kind, text, event, link_ref, note, done} + allowed-window map per kind (rule) — R/W
WR-039 DUE · not planned items per event (re-sign UFA ask ~$6.5M, release saves/dead, RFA tag cost, my pick #123, roster 47→44) — expiring contracts + market ask estimate (NEW) + tag cost rule + cutdown math — R; click adopts into plan (W)
WR-040 WHERE THINGS STAND: desk threads w/ turn, pocket count/stale, market bids w/ state, cap free to bid, draft picks + queue head — cross-tab live summary — R
WR-041 CAP 2027 now vs "if every step happens" + assumption note — derived from plan steps cap effects — derived
WR-042 CONFLICTS: WINDOW / ASSET (desk version sends player plan re-signs) / PICK (version spends pick plan assumes) — cross-check plans vs pocket/versions — derived (NEW)
### Market
WR-043 MY TARGETS lanes per position w/ need tag (THIN/HOLE/DEPTH), ranked targets, price (my bid or est. win), state (HIGH BID/OUTBID/NOT PLACED/est. win) — private targets per user (NEW) — R/W
WR-044 FA POOL filters ALL / MY NEEDS / ★ TARGETS / AFFORDABLE with counts — derived
WR-045 FA row: star, pos, name, age, MIN BID, VALUE, RIVALS (0–5 pips: how many teams likely to bid), EST. WIN price, TREND ▲▼ vs previous estimate, MY BID state — FA pool + min bid rule + valuation + rival model (NEW) + winning-price estimate (NEW) + estimate history — R
WR-046 Header "in-season · drag to lane or BIDS" — season phase
WR-047 Budget: BIDS AGAINST CAP ROOM, HARD LIMIT, placed vs ready segments, $ used of cap room, free / AT CAP — sum of open bids vs cap room (R3 rule: open bids ≤ cap) — derived; engine gap: aggregation lead
WR-048 READY TO PLACE stage: bid rows w/ stepper (min bid floor, cap ceiling), est. win, rivals, odds % bar, REMOVE — bid drafts (NEW private store) + win-odds model (NEW) — R/W
WR-049 PLACE n BIDS → confirm "FINAL · CAN'T BE PULLED" w/ rows, total vs room — bid.place W (bids binding: R3)
WR-050 PLACED · FINAL: LOCK, player, amount, HIGH BID placed at / OUTBID by Owner N at $X · money freed, BID AGAIN (min = top+0.1) — bids{status high|outbid, placed_at, top_bid, top_bidder?} — R; NOTE: shows outbidder name + amount — confirm visibility rule (R2: no early peeking at bids)
WR-051 RECENT WINNING BIDS last 2 weeks: pos, name, min → won — completed auctions history — R
WR-052 Tray: CLEAR READY / PLACE n BIDS, msg "$x of $y cap room in bids" — derived
WR-053 FA inspector: age, min bid, rivals, value, est winning bid, affordability sentence — R
### Draft room
WR-054 Clock bar pre-draft: ROOKIE DRAFT date/time CT, my picks list (slot, own/via, proj) , ORDER note — draft schedule + pick ownership — R
WR-055 Clock bar live: ON THE CLOCK slot + team, timer bar, YOU PICK IN n, LAST PICK (slot, team, player, pos, ADP), DRAFT <queue head>/BEST AVAILABLE, pause/resume mock — live draft state{current_pick, deadline, picks made} — live; mock draft runner (local/NEW)
WR-056 DRAFT ORDER: round tabs R1–R4/ALL/MINE, round headers, slot no, owner, "↪ via original", ON CLOCK, picked player — pick ownership incl. traded-via chain + projected order from standings — R
WR-057 Order status "projected from standings" / "n of N made" — R
WR-058 Legend mine/traded/PROJ — static
WR-059 MY BOARD tiers 1–5 with counts left, prospects (pos, name, ADP, need highlight, in-queue, N = has my notes), "cleared: T1…", "LIKELY THERE AT #27" line, MY NOTES ONLY toggle — prospect pool + consensus ADP (source + date "league mock drafts, Oct 28") + my tiers (NEW private) — R/W
WR-060 MY QUEUE ordered, drag reorder, auto-pick follows it; removal log "X went 1.05 to Y. Removed from your queue." — private queue (NEW/PARTIAL) — R/W
WR-061 Scouting report modal (PRIVATE): pos, name, tier, consensus ADP + source/date, verdict line, grade 1–5, confidence L/M/H, my rank, quick-note game context (set/clear game), trait chips per position (+/−), note text, observations list (date, game/clip, trait, sign, text), strengths, weaknesses, fantasy role, comp, COMPARE…, ADD TO QUEUE, save state — private scouting store (NEW) + position trait taxonomy + college schedule for game context (NEW) — R/W
WR-062 Tray: CLEAR QUEUE / SAVE QUEUE — W
### Inspector
WR-063 pos, shelf side / FREE AGENT, name, team, age, salary thru year — R
WR-064 to you / to them lens values (or est winning bid for FA) — derived
WR-065 Why sentence (edge gap / bait / priced out) — derived
WR-066 CONTRACT years, end type, dead if cut — R
WR-067 VALUE ARC age 24→34 bars w/ now — age-curve model per player — R
WR-068 IN YOUR LINEUP: slot, +pts/wk, replaces whom (age), L5 avg both, fills need years — lineup sim + last-5 averages — derived
WR-069 NOTES (MINE) count — private notes — R/W
WR-070 ADD TO TRADE / WATCH — W
WR-071 footer "/ inspect DE garrett" command echo, + DRAWER — command registry / prefs

## LP — League Pulse v14 (read-only screen: "nothing here changes the league")
### Scoreline strip
LP-001 Rail NOW leaf "Leading 132.1–118.4" / "Week 9 live now" — my live score vs opp — live
LP-002 Command hint per tab (pulse.live game=…, pulse.matchup week= team=, pulse.race side= view=, pulse.archive franchise= era=) — command registry
LP-003 Week selector ◂ WEEK n ▸ (no future weeks) — current week + completed weeks — R
LP-004 Slate: local time CT, n NFL live, n final, n to play / past week "FINAL · all 16 results locked" — NFL slate status — live
LP-005 Freshness: "scores: ESPN live · 4s ago / official after nflverse · Tue"; past week "results: checked facts" — per-source last-update + truth-ladder stage per week — R — live
LP-006 My box: YOU score, win-prob bar, opp score, opp abbrev — R — live
### Live tab — left column
LP-007 Toggle MATCHUPS / NFL GAMES; count "16 games"/"week n" — R
LP-008 Matchups list filters ALL/LIVE/CLOSE (+MINE in MU view) with counts; close = margin <10 — derived
LP-009 Group headers MY MATCHUP / NFC SIDE · n / AFC SIDE · n — conference membership — R
LP-010 Matchup row: both franchise names, scores, win-state per side, win % both sides colored, wp bar, status LIVE/WAITING/ALL PLAYED/FINAL — all fantasy matchups for week w/ live scores + win prob model (NEW) — live
LP-011 NFL games list: grouped LIVE/TO PLAY/FINAL; teams, scores, possession marker, clock or kickoff time, exposure chips "n YOU / n THEM" (my starters / opp starters in game) — NFL scoreboard live + possession + roster↔NFL-game map — live
LP-012 NFL filter ALL / MY MATCHUP / LIVE with counts — derived
### Live tab — center
LP-013 FOLLOWING chips (followed NFL games: score, quarter/clock, SNF/FINAL) — per-user followed games (auto = games with my exposure? NEW pref) — R/W
LP-014 Game header: away/home score, abbrevs, live clock w/ blink, "X at Y" — live
LP-015 Field graphic: endzones, yard lines, drive start, line of scrimmage, first-down marker, ball, drive trail animation — live drive state{possession, down&distance, yardline, drive start, drive summary "7 plays · 41 yds · 3:58", last play text} (NEW: drive feed) — live
LP-016 Pre-game "Kickoff <time> · field appears…" / final summary — R
LP-017 MY MATCHUP · STARTERS spine: header both team names + totals; OFFENSE/DEFENSE groups; per slot: player name, NFL team, compact stat line (QB C/A yds TD; RB car-yds, rec; WR rec-yds, tgt; OL ypc, sacks allowed; IDP tkl, sack), game-state chip (clock/FINAL/kick time), on-field ball icon (player's unit currently on field), points (up/dn coloring), slot label; proj when not started — live box stats per player mapped to slot + on-field derivation from possession — live
LP-018 "MY MATCHUP ‹" back button when viewing another matchup — local
LP-019 Spine header legend + proj totals both sides — derived
### Live tab — right
LP-020 IN THIS GAME: "n rostered"; you/them this-game totals; every rostered player in that NFL game with owner abbrev / "free agent", NFL team, points — league-wide roster index × NFL game — live
LP-021 PLAY-BY-PLAY: n plays newest first / "starts <time>"; per play: clock, text, ribbon me/them, "your starter · +x pts" — NFL play-by-play feed (NEW) + per-play fantasy delta attribution (NEW) — live
LP-022 Tray toast "Puka Nacua · 18-yd catch · +2.8 YOU" on new scoring play — live event
### Matchups tab (ws-mu, reachable via my box / filters)
LP-023 Score panel each side: YOU/THEM label, franchise name, record, conf+division, live proj, points, PLAYING/TO PLAY/DONE counts — live
LP-024 Center: status, big win-prob bar, % both + "WIN PROB · model" — live
LP-025 View toggle ACROSS THE LINE / CLASSIC; legend playing/done/to play; source "live · ESPN · stat lines in BOX SCORE" / "checked facts" — R
LP-026 Across rows per slot: state dot, player, NFL team, game state, points, proj, win/lose shading — live
LP-027 Past weeks: generated full-week lineups for any team (needs historical lineups per franchise per week) — R
### Race tab
LP-028 STANDINGS view SEEDS/DIVISIONS + side AFC/NFC — R
LP-029 Seeds table: seed, franchise, division + ★ leader, W-L, PF (or GB in div view), PWR rank, playoff ODDS %; cut line after seed 7 — standings + seeding rules (div winners first, 7 per side) + power + sim odds — R — weekly/live
LP-030 Division view: division header w/ leader + record; GB column — derived
LP-031 MY ROAD WKS 9–14: week, opponent, NOW tag, ★ if div leader, opp record, my win % — remaining schedule + win model — R
LP-032 Mode POWER RANKINGS / PLAYOFF PICTURE; horizon THIS SEASON / DYNASTY · 3 YR / BLEND; caption "side · wk · in season" — R
LP-033 Power weights chips (FORM 30, LINEUP 30, HEALTH 15, ALL-PLAY 15, RUN-IN 10 / VALUE 35, AGE 20, PICKS 20, CAP 15, DEPTH 10 / 60-40 blend) with tooltip definitions — league setting (power formula weights, labelled) — R
LP-034 Power rows: rank, move ▲▼ vs last week, franchise, WINDOW label (BUILT TO LAST/WIN NOW/ON THE RISE/REBUILDING/MIDDLE), score + bar, per-component ranks; blend shows season rk, dyn rk, both scores, SHIFT; PEAK YEAR (dyn); VS RECORD (seed − power rank) — power model per franchise per component (NEW) + weekly history — R
LP-035 WHY IT MOVED: THIS WEEK (move + note text per team e.g. "TE1 out for the year · 4 straight vs leaders next"), THIS SEASON (rank trajectory segmented chart wk1→now), LAST 12 MONTHS (blend rank then→now + what they did "traded 2027 1st for WR1") — power history + generated move notes from transactions/injuries (NEW) — R
LP-036 Playoff picture bigs: my seed today → projected, MAKE PLAYOFFS % sim, FIRST-ROUND BYE % sim, MAGIC NUMBER — playoff sim (NEW, 10k runs) — R
LP-037 THE RUN-IN rows (top 11): now seed, proj seed ▲▼, franchise, W-L, proj wins, wk10–14 opponent + win % cells (★ outline leaders), ODDS, BUBBLE tag, flag ("3 IN A ROW vs ★", "n of 5 vs ★", SOFT FINISH) — schedule + win model + sim — R
LP-038 PROJECTED BRACKET: wild card wk15 pairs, divisional wk16 w/ bye, final wk17 vs other side's #1 — playoff format setting (7 per side, 1 bye, side champs meet) — R
LP-039 THIS WEEK MOVES IT: scenario → my odds ("I beat X 96%", "bye 52%", "+3%") — sim conditional outcomes (NEW) — R
### Race drawers (tray pulls)
LP-040 TIEBREAKERS: #, tied teams, step used, detail, resulting order + note on order (league setting link Control Room › League) — tiebreak engine w/ explanation (NEW/PARTIAL) — R
LP-041 BRACKET drawer: projected both sides + final — R
LP-042 PRODUCTION tabs: ALL-PLAY (W-L, pct, actual, luck), STARTERS BY POSITION (pts per pos group per franchise), STARTERS BY PLAYER (rank, player, owner, pos, pts, games started), FRANCHISE VIEW (seasons, titles, playoffs, all-time W-L, dynasty pwr) — season aggregates — R
### Live/Matchup drawers
LP-043 BOX SCORE / MATCHUP BOX: side toggle; slot, player, line 1, line 2, TD/TO, state, pts — fantasy box per lineup — live
LP-044 WEEK STORY: SUMMARY (time-stamped headline + generated paragraph, awards HIGH SCORE / TOP QB / TOP RB / TOP IDP), PREVIEW · THIS GAME (written Thu, proj, edge text), RECAPS (post-MNF) — generated text (NEW content engine) + awards calc — R
LP-045 SCHEDULE: LEAGUE grid (franchise × weeks, result W/L or opponent), MY TEAM list (wk, opp, result, score), COMPACT — full schedule + results — R
LP-046 STAT CHANGES (count badge "2"): wk, player, stat was→now, pts was/now, Δ owner, result impact ("no change · W stands") + disclaimer — stored first-held results vs latest official (truth ladder diff) — R
LP-047 NFL BOX SCORE tabs PASSING/RUSHING/RECEIVING/DEFENCE: player, team, stats, ROSTERED by (abbrev/YOU/—) — NFL box stats + roster index — live
### Archive tab
LP-048 Mode FRANCHISE / LEAGUE; franchise selector (all 32, mine first) — R
LP-049 ERAS (one per owner): All eras + each owner name, years, seasons count, record in era, colour swatch, "You · Christopher" — franchise ownership history (owner tenures) (NEW/PARTIAL) — R
LP-050 TOTALS for era: record, win %, playoffs n/seasons, div titles, conf titles, titles of finals; last title, last final, playoff streak — season history per franchise — R
LP-051 League mode MOST TITLES list — R
LP-052 TROPHY CASE shelf plaques (★ year, NFC year, DIV year) / league "record book" — R
LP-053 Every-season chart: bar = win %, W-L label, year + regular-season rank of 32, stamps (DIV TITLE/CHAMPION/NFC CHAMP/PLAYOFF WIN/PLAYOFFS), era bar, dim outside era; hover tip (record, rank, playoffs stage, division, PF, owner) — franchise season history{W,L,rank,div_title,playoff_stage,PF} — R
LP-054 Season story card: year · stage · record · rank; owner; data-check badge "CHECKED 9/9" / "7/9 CHECKS · 2 OPEN" (history verification status per season); headline + paragraph (commissioner/generated season notes); stamps — season notes store (NEW) + checks status (league-history verification) — R
LP-055 PLAYOFF RUN: round, opponent, score, W/L — playoff results history — R
LP-056 SEASON AWARDS: top scorer, best week (score, wk), points for (rank) — derived history — R
LP-057 WHO WAS HERE · year: pos, player, how acquired ("Draft 1.03", "Trade", "Free agent"), year, by which owner (era colour), note ("for McCaffrey + 2022 1st") — roster lineage from transaction history (NEW derivation) — R; empty state "appears once transactions are imported"
LP-058 HOW WE GOT HERE: key moves per year per owner — curated/derived key moves (NEW) — R
LP-059 League center: champions table season, champion, runner-up, reg season, MY FINISH (rank · stage), CHECKS n/9 — R
LP-060 Archive drawers: TROPHY CASE (league/NFC/division titles, records held, "won by you/owner", placeholder art), WEEK BY WEEK (wk, opp, result, for, against, standing after; "not held" gaps), RECORD BOOK (record, holder, value, season) — R
### Inspector
LP-061 Team: FRANCHISE / YOU, name, side + division, record, seed; POWER rank + move; PLAYOFF ODDS; THIS WEEK (opp, score, win prob); SEASON (PF, PA, all-play %, streak, points-by-week bars); HEAD TO HEAD vs you (all-time, last meeting); LAST 5 W/L by week; HISTORY (titles, finals, open in ARCHIVE); GO TO ROSTER ▸ / TRADE DESK ▸ — R (+ deep links)
LP-062 Player: PLAYER, YOURS/THEIRS, name, NFL team, rostered by / free agent, this week pts + proj, season PPG, TODAY (game, state, last scoring plays w/ delta), LAST 6 WEEKS bars, CONTRACT (link HQ), VALUE (link War Room), GO TO roster/trade desk, NFL BOX — R
LP-063 Drag any name to inspector to pin; MY TEAM button — local
LP-064 Tray footer "reads only · nothing here changes the league" — static

## HM — Home v5 (reads and links only)
### Header / chrome
HM-001 Rail head phase "Wk 9 · 6 live" / "Offseason" — season phase — R
HM-002 NOW leaf "Lineups lock Sun" / "Tag window · Mar 4" — next deadline by phase — R
HM-003 Layout label "DEFAULT · IN-SEASON" / "MINE (custom)" — per-owner per-league layout{mode, tiles[{id,span,order}], overflow[], pins} (NEW store) — R/W
HM-004 (mock-only phase/alert toggles — not product) — n/a
HM-005 NOW strip: tag LIVE/OFFSEASON, sentence (score vs opp + lock time / key offseason dates), 4 stats (in: record, seed, power, cap room; off: dynasty power, first pick slot, next-year room, roster n/limit) — R — live
HM-006 Alert band: top alert kind, text, countdown, action link, "+n ALL ALERTS"; crit styling; "ALERTS UNAVAILABLE · last check failed hh:mm · retrying every 60s · not an all-clear" — alerts engine{id, kind, urgency, title, detail, deadline, action route} + alert-check health status — R — 60s
HM-007 Top-5 ticker per position (QB/RB/WR/TE/DL/LB/DB/K/OL): rank, player, NFL team, stat line, fantasy pts (league scoring); "TOP 5 · WK 9 LIVE / WK 18 FINAL" — weekly NFL leaders scored with league rules — live
### Tile registry (each tile: title, source label, provider, link route, refresh policy, allowed sizes, states loading/empty/stale/unavailable/error, "as of" time, "+n more in <screen>")
HM-008 Tile registry metadata itself — registry entries {id,title,src,provider,link,refresh,spans} — static/config
HM-009 DECISIONS DUE tile: kind (lineup/trades/bye/ir/clock), text, date, countdown (cold if >3d); phase-specific list incl. full league calendar (trade deadline, taxi promotions close, playoff seeding locks, franchise tag window, rookie options due, RFA tenders due, FA opens, rookie draft, cut-down, week 1 lock) — leagueclock.ActionCalendar + alerts — R — 60s
HM-010 THIS WEEK tile: both team names, records, proj, scores, win % both, wp bar, PLAYING/YET TO PLAY/DONE, slot duel rows (player, pos, my pts, opp pts, opp player) — pulse.Matchup — live
HM-011 RACE tile: seed now, projected seed + BUBBLE, playoff odds, power; run-in wk/opp/win% with leader marker; warning "3 OF NEXT 4 vs DIVISION LEADERS ★"; mini standings (seed, team, record, me, CUT line) — standings + sim — R
HM-012 LEAGUE HEADLINES tile: kind (HI/UPS/TRD/STK/REC/CLN/LOW/INJ/DIV), generated fact text, week — headline generator from scores, trades, streaks, records (NEW) — on-event
HM-013 NFL NEWS tile: filter ALL/MY PLAYERS/INJURIES/ON THE WIRE + count; time, kind (INJ/DEPTH/TRADE/SIGN/IR/SUSP), headline, source attribution, ownership relevance (mine/opp/wire), league tag ("Your FLEX", "Opponent QB", "On the wire", owner name, "Free agent") — news feed free blend (NEW) + player-id match + league ownership + my lineup slot — live 60s
HM-014 TRANSACTIONS tile: time, kind (TRADE/BID/WIRE/IR/TAXI/FA/CUT), who + what, detail ("$1.2M · 3 bids", "clears Wed 22:00", "dead cap $1.4M", "taxi → active") — ledger.Transactions(league) — on-event
HM-015 MY SHORTCUTS tile: grouped buttons with hotkeys (Set lineup L, Roster O, Cap C, Taxi·IR I, Trade Desk T, Market M, Draft Room D, Plans P, Live G, Race R, Archive A, Rules U) — owner pins + hotkey map — R/W
HM-016 POWER tile: horizon caption with weights; rank, team, score, bar — pulse.Power — weekly
HM-017 PLANS tile: active plan name, checked n of m; step checkbox, text, date — plans.Checklist(active) — R (W check?)
HM-018 MY PICKS tile: pick label (2027 1.07 / 2028 1st), origin (own · proj 7th / from Rams / to Mudcats traded), rookie slot salary ("$2.1M slot") — draft.OwnedPicks + rookie scale by slot (rule) — R
HM-019 CAP RUNWAY tile: room this year of cap, bar committed/dead/room, dead total; per year room + note ("3 expiring", "RFA: Baird", "UFA: Kelce") — cap.Runway(4y) + expiring notes — R
HM-020 MY BIDS tile: player+pos, amount, state PLACED·HIGH/OUTBID/DRAFT·NOT PLACED — market.MyBids — on-bid
HM-021 FORM · LAST 5 tile: abbr, player, avg, 5 weekly bars — stats.Form(starters,5) — after final
HM-022 Default layouts per phase (in-season / offseason) — config — R
### Customize / alerts / overflow
HM-023 Tile size buttons 1×1/2×1/1×2 (disabled when not allowed), ☆ pin tile's screen to MY SHORTCUTS, ✕ to overflow, drag swap, grid-full bump flow — layout W
HM-024 ADD TILE drawer: all registry tiles w/ sizes, source, on grid / in overflow — R
HM-025 OVERFLOW drawer: bumped tiles, RESTORE — layout R/W
HM-026 USE THIS PHASE'S DEFAULT / RESET TO DEFAULT / DONE — W
HM-027 ALERTS drawer: n active · n snoozed; per alert kind, countdown, title, detail, primary action, SNOOZE 1H / UNTIL 8 AM, snoozed-until + UNSNOOZE; failure card + RETRY NOW; note "alerts ignore layout; snooze can't turn a kind off" — alert snooze store per user (NEW) — R/W
HM-028 Inspector: MY FRANCHISE, division, name, owner since year, window label (WIN NOW), record·seed, power; selected tile metadata (id, source, provider, refresh, as of, opens, sizes, states); layout mode, tile counts — R
HM-029 Tray ALERTS n / ? ; ADD TILE; OVERFLOW n — R
### Cards
HM-030 News card: kind, time · source, headline, blurb paragraphs (summary only), player panel (name, pos·team·age, in this league owner, status, "For you" relevance e.g. "Starting FLEX wk 9" / "On the wire · bids close Wed 22:00"), PLAYER › , OPEN SOURCE › (url) — news item{summary, source_url} + player link — R
HM-031 Transaction card (trade): both sides: assets (name, pos, NFL team, age, contract $×yrs / pick proj), asset count, cap out; "processed · commissioner approved · no veto window"; BOTH ROSTERS ›, OPEN IN LEDGER › — trade record w/ approval path — R
HM-032 Transaction card (other): what, detail, claimable-by + dead cap to dropper (WIRE), runner-up "sealed · shown after the window" (BID), when; OPEN IN MARKET ›/LEDGER › — R
### Depth charts tab
HM-033 NFL team picker by division (32 teams) — static NFL ref
HM-034 Header: team name, "updated Oct 27 17:27 · source TBD"; legend yours/rostered/on the wire, Q injury, R rookie, experience codes (23/1 drafted yr/round, U = UDFA, SF/CF signed) — NFL depth chart feed (NEW, source TBD) — R — daily
HM-035 Sections OFFENSE·DEFENSE·ST / SPECIAL TEAMS / PRACTICE SQUAD / RESERVES (IR/NFI/DFR) + COACHING STAFF line; personnel note "11 personnel · one RB, one TE (41%)", "base 3-4" — depth feed + personnel usage stats (NEW) — R
HM-036 Depth row: pos slot (LWR/RWR/SWR/LT…/NB, PT/PK/LS/H/KO/PR/KR), jersey no, player name colored by league ownership (tooltip owner), exp/acq code, Q/R flags; click → player card — depth feed × league ownership — R
### NFL stats tab
HM-037 SAVED VIEWS list (name, pos) + SAVE CURRENT AS VIEW — per-user saved filters (NEW) — R/W
HM-038 Filters: POSITION (QB RB WR TE DL LB DB K), WEEKS range (season), IN YOUR LEAGUE ALL/MINE/ROSTERED/ON THE WIRE, NFL TEAM, MINIMUM (GP≥3, FFB≥10/G, SNAPS≥50%) — needs snap share data (NEW) — R
HM-039 Header: pos · weeks, n players, sorted by, as of; COLUMNS, EXPORT CSV — W(export)
HM-040 Stats table per position columns (QB GP C-A PASS YD TD INT RUSH YD R TD; RB ATT RUSH YD YPC R TD TGT REC REC YD REC TD; WR/TE TGT REC YD Y/R TD; DL/LB/DB TKL AST SACK TFL QH FF INT PD; K FGM-A LONG XP) + IN LEAGUE owner + FFB PTS + FFB/G scored with league rules — nflverse official + live blend, aggregated by week range, league scoring — R
### Player career card (opened from any name on Home)
HM-041 Header: pos, NFL team, jersey no, full name; bio Ht/Wt, Born (age), College, Draft (year #pick team, experience) — player bio (NEW/PARTIAL) — R
HM-042 Fantasy team (you / owner / on the wire), salary · contract year, contract notes (EXT history, restructure), acquired how/when/from — contract + acquisition history — R
HM-043 Tabs PLAYER NEWS (dynasty blurb, practice report, outlook w/ ages), 2026 STATS (game log: wk, opp, result, stats, FFB, BYE row, total), PROJ. (wk, opp, rank vs pos, proj FFB, range), CAREER STATS (year, NFL team, fantasy team that season, stats, FFB, FFB/G, career total) — weekly game logs + projections w/ ranges (NEW) + historical fantasy ownership by season — R
HM-044 Actions FULL PROFILE ›, PROPOSE TRADE › (War Room, not if mine), + WATCHLIST, TRANS. HISTORY › — links + watchlist W

## CR — Control Room v1 (commish / co-commish; DOT sees subset)
### Shell
CR-001 Rail head "Wk 9 · in season" — phase — R
CR-002 Rail tree leaves (9 branches) + tab badge = count of NEEDS YOU jobs — jobs count — live
CR-003 Signed-in role label + role (C / CC / DOT) — viewer roles — R; every button gated by ONE permission check mayRun(cmd) = role + delegated permission sets + LOCKED/CONLY/DOTONLY lists — NEW permission-set engine
CR-004 Find box: indexes rules (name, family, value), jobs, situations (+synonyms), fixes, teams, pick actions; ranked word match — search index over all CR entities (NEW) — R
CR-005 Command bar: any command w/ args; "Every screen action is also a command"; output line — command registry (92 commissioner commands, confirm levels c/h/r) — R/W
CR-006 Editor tray (bottom) per action: fields, typed reason (required for G3/hold), preview line, hold-to-confirm 600 ms / click confirm — command safeguards + backend preview (NEW/PARTIAL: RuleChangeForm exists) — W
CR-007 Toast "MOCK · would record" / refusals (not delegated, not built, reason required) — UI
CR-008 Inspector default: n things need you · n blocked; how-to; LATEST ON THE LEDGER 7 entries (actor, text, time); YOUR ROLE text — R
### Jobs
CR-009 QUICK ACTIONS 8 launchers — static routes
CR-010 THIS WEEK kv: week n of N regular weeks, lineups lock, waivers run, trade deadline (+"instant unset"), next window, last backup — calendar + backup status — R
CR-011 Job rows grouped NEEDS YOU / UPCOMING / BLOCKED / NOT BUILT: dot, name, words (live description), count, when (due/countdown), NO PRODUCER YET tag — job{kind, state, title, detail, count, due_at, opens, census_id} produced by job producers (stat queue, trade review, appeal, waiver run, bid processing, must-cut, vacant franchise*, missed lineups*, soft-cap*, weekly notice, trade deadline, playoff seeding, rollover, setup needs-ruling, feed health*, backups*, invites*, votes*, notifications) — *=no producer yet — R — live
CR-012 Job inspector: state, when, census id, OPEN — R
CR-013 Stat corrections card: per change player·team·pos, stat + week, was, now, effect (team ± pts, result flip named), approve/reject; flip needs reason + hold — stat queue{player, stat, week, old, new, fantasy delta per team, flips[]} — R/W
CR-014 Trade review card: proposed date, rule "DOT decides at 3 approvals or 3 vetoes", closes in; each side's assets (pos, name, $·yrs, picks), cap room after + roster n/max after; DOT approve/veto pips + who voted + recused seat; validity note; override reason + HOLD OVERRIDE APPROVE/VETO (C only) — trade review{votes[], threshold, deadline, recusals} — R/W
CR-015 Appeal card: veto tally + date, appellant quote, trade legs, ruling text required, UPHOLD/OVERTURN — appeal{party, statement} — R/W
CR-016 Must-cut card: team n/max, since, owner notified count, lineup validity; REMIND OWNER, ALLOW UNTIL, CUT FOR THEM — R/W
CR-017 Missed lineups card: team, problem (n starters on bye / QB empty), owner state (vacant·caretaker / last seen nd), SET LINEUP (CARETAKER), REMIND, CORRECT AFTER LOCK — lineup validity per team + last-seen per owner (NEW) — R/W
### Situations
CR-018 Category chips w/ counts (GAMES, PLAYERS, OWNERS, TRADES, ROSTERS, MONEY, MARKET, CALENDAR, DRAFT, PICKS, STANDINGS, RULES, LEAGUE) — static catalog (35 situations)
CR-019 Situation card: ordered steps (text, command, OPEN › deep link; tags RULED·NOT BUILT / GAP), "Still up to you", "Check afterwards" — static playbook data + command registry (built/unbuilt status per command) — R
### Rules
CR-020 Family chips (8 families + ALL, NEEDS A RULING, PENDING, PLANNED) with counts — rules catalog — R
CR-021 Rule table: section, rule name, current value, state (SET/NEEDS A RULING/NEXT ROLLOVER/PLANNED/FIXED), ⚖ competitive flag; pager — 130 rule rows from leaguerules (state, value, source, choices, parts, competitive) — R
CR-022 Rule inspector: family › section, name, value, state, source text, command, competitive, CHOICES, PARTS, CHANGES THESE OWNER SCREENS (governed screens per family), HISTORY (rule change history), CHANGE ›, ONE-OFF EXCEPTION › — rule change history per rule (NEW/PARTIAL) — R
CR-023 Rule editor: value (select from choices or input), TAKES EFFECT next rollover/now, reason (required if now), preview (affects n teams, governs screens, competitive warning), SAVE → pending list — RuleChangeForm + backend preview + pending-rollover queue — W
CR-024 Presets: APPLY A PRESET (32-Team Default, 16-Team Mirror, Legacy NFL, Your MFL league, Start empty), COMPARE TO A PRESET — preset bundles (settings values) — R/W
### Season desk
CR-025 League-year calendar: months Mar→Feb, windows (contract window, RFA tenders, UFA, re-sign, open FA, rookie draft, cut day, regular season, trade deadline, playoffs, rollover) w/ dates, today line, timezone + "owners see their own time"; drag to move / click edit (window name, opens, closes, what it opens, preview) — calendar.windows — R/W
CR-026 WAIVERS desk: claims count/players; window, priority method, next run; claims list (player, claiming teams, winning priority #); RUN NOW, SET ORDER BY HAND, AUTO ORDER, FIX AN AWARD — waiver claims + priority order — R/W
CR-027 BIDS desk: open auctions count; mode (UFA/RFA open), stand-to-win hours, blind note; auctions (player pos, bid count — never amounts, stands Nh / cleared); PROCESS CLEARED, FIX AN AWARD, CLOSE O-LINE AUCTION (C only) — auctions{player, bid_count, standing_since} — R/W
CR-028 DRAFT desk: date, speed, clock; order method, rounds, startup used; projected R1 order; SET ORDER, START, OPEN DRAFT ROOM TRAY, REVERT, ARCHIVE — R/W
CR-029 SCHEDULE desk: weeks, pattern; published date, edits count; week matchups list; EDIT A GAME, REGENERATE, PUBLISH — R/W
CR-030 PLAYOFFS desk: format, weeks, bracket preset, seed timing, home field bonus; seeds-if-ended-today + bubble; BUILD, EDIT, SEED A SLOT, ADVANCE (OVERRIDE), TITLE LABEL — R/W
CR-031 ROLLOVER desk: date + instant ruling state; contracts age, expiring handling, rule changes waiting n; checklist (instant, pending changes, backup first); SET INSTANT, PREVIEW (DRY RUN), RUN ROLLOVER (C only) — rollover dry-run report (NEW/PARTIAL) — R/W
CR-032 C ONLY lock badges on buttons per role — mayRun
### Fix
CR-033 12 fix tiles (stat, score, lineup, player position, roster, transaction, money, standings, rule outcome, window, reserve, undo) w/ command — static
CR-034 Fix editor: fields per fix (prefilled), reason required, preview (teams touched, result flip), HOLD APPLY — W
CR-035 EXCEPTIONS IN FORCE: what, who, ends (date/"until cleared"/"owner seated"), END NOW / CLEAR — active exceptions list w/ expiry (NEW unified view over reserve/calendar/rule/position/cap/caretaker/timer/standings exceptions) — R/W
CR-036 RECENT FIXES: when, fix text, by (role), UNDO — corrections ledger filtered — R/W
### Picks
CR-037 Pick actions: grant (from nothing), move, swap, forfeit, condition, void/restore — pick commands (only move built) — W
CR-038 Season chips 2026/2027/2028, team filter, "n non-standard picks" — R
CR-039 Pick table: round, original team, owned by (bold if traded), condition, how (Original / Trade date / Penalty: reason / Granted: reason), status ACTIVE/FORFEITED; pager — pick{season, round, original, owner, condition, status, provenance} (league-granted picks w/ orig "League") — R
CR-040 Pick editor: season, round, team(s), condition text, reason (visible), preview (slot placement for granted picks, max-picks check "16 rookie picks over max 15") — W
CR-041 Pick inspector: history chain (issued at draft close, trades), MOVE / ADD CONDITION / FORFEIT — pick history — R
CR-042 Limits note (max rookie picks per team, extras forfeited) — rule — R
### People
CR-043 Mode SEATS & ROLES / PERMISSIONS (permissions C only)
CR-044 Franchise table (32): team, seat state (owner seated/CARETAKER/quiet nd), role (Commissioner/Co-commish #1/DOT), last seen, 2FA flag — seats + roles + last-seen (NEW) + TOTP enabled flag — R
CR-045 ROLES desk: counts per role, deputy order, grants private; GRANT ROLE, REVOKE ROLE, DEPUTY ORDER (locked from sets) — W
CR-046 SUCCESSION desk: vote state, inactivity thresholds (season/off), last commish action; STEP DOWN, INACTIVITY RULES, OPEN VOTE-OUT, OPEN ELECTION — R/W
CR-047 ABILITIES & AI desk: owner abilities n on/off, AI agents connected; SET ABILITIES, WHO SEES WHAT, CONNECT AN AI AGENT (wizard: job, scope+expiry, tool + token, status/last used/revoke — NOT BUILT), PERMISSION SETS — R/W
CR-048 Team inspector: seat, role, authenticator on/off, permission sets held, picks count; RENAME/ABBREVIATION, INVITE OWNER, SEAT ME AS CARETAKER, RESET AUTHENTICATOR (not built), REPLACE OWNER, REMOVE OWNER, SEE PICKS — R/W
CR-049 Permission sets list: name, n actions, n people, until — permission_set{name, purpose, commands[], holders[{team, scope, expiry}], version} (NEW) — R/W
CR-050 Set editor grid: all commands grouped (Trades, Stats & scores, Draft, Waivers & bids, Picks, Seats…), plain-word name, command on hover, tags 🔒 never / C-ONLY / DOT; checkbox; staged add/remove highlights + stage bar (save/discard) — CMDLIST + PLAIN names + LOCKED/CONLY/DOTONLY — R/W
CR-051 WHO HOLDS IT: person, built-in role, scope, until, REVOKE; grant row: person, scope (whole league / these teams), expiry (none/end of season/30 days/date), reason required if set has C-only, HOLD GRANT — W
CR-052 BUILT-IN ROLES summary (Commissioner / Co-commish / DOT powers) — R
CR-053 + New permission set: name, purpose (private), SAVE — W
CR-054 Who can do what: person picker → every power + source; action search → holders list with source, exclusions note, delegability note — derived from roles + sets — R
### League
CR-055 TELL THE LEAGUE: type NOTICE/NEWSLETTER/LEAGUE NOTE, text, pin to top, LET AI DRAFT IT, PUBLISH — notices store + AI draft — W
CR-056 DATA & ENGINE desk: last refresh, engine parameters; REFRESH NOW, CLEAR A CACHE, ENGINE (C only), FEED HEALTH › — R/W
CR-057 LEAGUES desk: league name, source (native/migrated from MFL), created; NEW LEAGUE (PRESET), MIGRATE AN MFL LEAGUE, SEED A HELD LEAGUE — R/W
CR-058 RULEBOOK desk: official rulebook version, house rules count, words→rule later; ADD/REMOVE HOUSE RULE, OPEN RULEBOOK — rulebook text store — R/W
CR-059 LEDGER: time, actor (Commissioner / Co-commish / DOT · team / System / "AI for Commissioner" + VIA AI tag), text; filter chips ALL/RULES/FIXES/TRADES/PEOPLE/VIA AI/SYSTEM; note "owners see this minus role grants and invites" — ledger{at, actor, via_ai, kind, text, visibility} — R
### Server
CR-060 STATUS: version + uptime, tunnel state, signed-in devices, DB size; 24h requests, p95, errors, disk %, memory; DOWNLOAD STATUS REPORT — server telemetry (NEW/PARTIAL) — R
CR-061 BACKUPS: destination (R2, encrypted), last + result, retention, next; recent list date/result/size; BACK UP NOW, RESTORE… — backup history — R/W
CR-062 UPGRADES: running, available + signature check, release notes; UPGRADE — update channel — R/W
CR-063 SIGN-IN HELP: authenticator users count, locked-out; recent sign-in trouble (team, event, date); RESET AUTHENTICATOR, SIGN OUT ALL DEVICES, RECOVERY CODES — auth events (NEW) — R/W
CR-064 FEEDS: live chain ESPN→Sleeper, final nflverse, staleness; per-feed checks (state, freshness, role), week compare diffs → stat queue; RETRY, SWITCH LIVE PROVIDER — provider health — R/W
CR-065 ROLES ON THIS SERVER: server admin holder, support holder; GRANT SUPPORT / SERVER ADMIN (not built) — R/W

## SOURCE MAP (pass 2, origin/main 364379ba) — data domains → status → rows
Status: EXISTS = a package produces it today · PARTIAL = close but missing a field/shape · NEW = nothing serves it.
Evidence = exported funcs seen via git show (package listed).

| # | Domain | Status | Evidence on main | Rows |
|---|---|---|---|---|
| D01 | League/franchise identity, phase, week | EXISTS | leagueclock (Calendar, Clock), app/now (LiveWeek, DeriveWeekPhase in app/control) | SH-001/002, HM-001, CR-001 |
| D02 | Standings, seeding, divisions, GB, H2H, bubble | EXISTS | engine/race (Seeding, Division, HalfGamesBack, HeadToHead, Bubble), app/season (Seeding) | HQ-004, LP-028–031, HM-011, CR-030 |
| D03 | Tiebreaker explanation (step used, detail) | PARTIAL | engine/race HeadToHead/Order exist; no "step used" narrative | LP-040 |
| D04 | All-play, production by position/player, schedule grid, brackets | EXISTS | app/season (AllPlay, PositionPoints, ScheduleGrid, Brackets, ProjectedBracketSide, PlayoffReport) | LP-038/041/042/045 |
| D05 | Power rankings + win probability + expected wins | PARTIAL | powerrankings (WinProbability, ExpectedWins, Blend, Lineup); mock adds components FORM/LINEUP/HEALTH/ALL-PLAY/RUN-IN, dynasty horizon, window labels, weekly history, "why it moved" notes | LP-033–035, HQ-024, LP-010/024, HM-016 |
| D06 | Playoff odds sim, magic number, bye %, scenario "this week moves it" | NEW | no sim on main (Simulat only in mflref/trades) | LP-036/037/039, HM-011 |
| D07 | Live NFL scoreboard, possession, scoring plays | EXISTS | ingestion/livegame (ParseScoreboard, ScoringPlays, GameLive, Followed) + app/now (PublishLive, RefreshLiveGame, FillLiveSpines) | LP-004/011/014, HQ-018/025 |
| D08 | Full play-by-play + drive chart (down/distance, yardline, drive summary) | PARTIAL | livegame has Possession/Drive refs + ScoringPlays only; mock needs every play + per-play fantasy delta | LP-015/021/022 |
| D09 | Live fantasy box per slot + compact stat lines + on-field flag | PARTIAL | app/now (MatchSpine, InspectLiveBox, HeldMatchupStarters); on-field derivation + compact lines new | LP-017/023/026/043, HM-010 |
| D10 | Followed games pref | PARTIAL | livegame.Followed exists; per-user follow store not seen | LP-013 |
| D11 | Projections (weekly, ranges, ROS) | PARTIAL | Projection refs in app/mflref, app/now, app/market (MFL-sourced); native ranges/rank-vs-pos NEW | HQ-017/023, HM-043 |
| D12 | Player weekly fantasy points + form (last N) | EXISTS | app/season PlayerPoints, fantasyscore | HQ-015, HM-021, LP-062 |
| D13 | Advanced usage (target share, snap %), defense-vs-position rank | NEW | no TargetShare/snap-share | HQ-071, HM-038 |
| D14 | Injuries (designation, body part) | EXISTS (data) | ingestion/injuries, app/control HeldInjuries, IRAlerts | HQ-018/061/064 |
| D15 | Injury expected return week | NEW | not seen | HQ-061/062 |
| D16 | NFL schedule, byes | EXISTS | ingestion/nflschedule, app/now PublishByes, app/control ByeAlerts | HQ-016, HM-009 |
| D17 | Rosters, lineup slots, eligibility, taxi/IR | EXISTS | leaguerules, nativecheck/taxi.go, engine/eligibility | HQ-005/011–021/056–064 |
| D18 | Contracts, salary by year, end type RFA/UFA | EXISTS | store/state, leaguerules (contracts) | HQ-033/034/069, WR-010 |
| D19 | Dead cap ledger + cut calc | EXISTS | transactions/deadcap, transactions/breakdown | HQ-035/041/044, HM-032 |
| D20 | Cap runway (multi-year committed/dead/room) | PARTIAL | inputs exist (contracts+deadcap); no runway aggregator (Runway 0 hits) | HQ-039, WR-008, HM-019 |
| D21 | Move envelopes (planned→handed off→landed→needs you) | EXISTS (MFL path) | mflref/envelope + app/mflref; native-mode lifecycle = ? | HQ-001/048/050–053/067 |
| D22 | Franchise transaction history / season ledger | EXISTS | transactions, store/history Ledger | HQ-053, HM-014, LP-057 |
| D23 | League ledger w/ actor, via-AI, visibility, kind filters | PARTIAL | store/history Ledger + governance Audience/Private; via-AI flag + kind chips = ? | CR-008/036/059 |
| D24 | Player valuation (dynasty value), pick value | EXISTS | app/market (PickValue, DynastyCells, PoolAssetValues), pickvalue | HQ-036, WR-010, HM-018 |
| D25 | Best 21 team score + league rank | NEW (ruled 10-29) | Best21/BestN 0 hits | WR-005/016, LP-034 |
| D26 | Age curves (value by age, peak window, value arc) | NEW | AgeCurve/AgeArc 0 hits (playerrecord has career, not curve) | HQ-073, WR-006/067 |
| D27 | Team needs / surplus / "trades for a late pick" equivalence | NEW | Needs hits are unrelated | HQ-047, WR-007/020/036/039, WR-043 |
| D28 | Franchise posture (contend/retool/rebuild) + lens valuation | NEW | Posture only in lab (R&D) | WR-004/014/019/064 |
| D29 | Trade builder versions, pocket, re-check staleness | NEW (private) | store/personal has Offers + ObserveOffers (closest) | WR-012/025/026 |
| D30 | Trade negotiation threads (rounds, counters, notes, expiry, turn) | NEW/PARTIAL | app/offers RecordOffers; counter rounds not seen | WR-027–032, HM-031 |
| D31 | Trade review DOT votes, recusal, override, appeal | EXISTS | governance (Reviewers, TradeReviewReady, Vote, AppealJudge) | CR-014/015, HQ-052 |
| D32 | Tune-it suggestion engine (TRIM/ASK/SWEETEN) | NEW | — | WR-017 |
| D33 | Lineup impact sim (replaces whom, pts/wk) | PARTIAL | powerrankings.Lineup (best lineup); per-trade delta new | WR-016/068, HQ-046 |
| D34 | FA pool, min bid, bid drafts/plans/posted | EXISTS (MFL path) | app/offers (TargetPool, TargetBidDraft/Plans/Posted/Withdraw), commissioner/market | WR-044/045/048, HM-020 |
| D35 | Rivals count, est. winning price, trend, win odds, recent winning bids | NEW (recent wins PARTIAL from history) | — | WR-045/048/051/053 |
| D36 | Open bids ≤ cap room aggregation; bids binding (R3) | PARTIAL + BUG | nativecheck/bids.go "caller judges cap"; market/bid_execute.go:136 withdraw() | WR-047/049/050 |
| D37 | Outbid shows rival name + amount | RULING CHECK | conflicts with R2 "no early peeking"? (placed bid outbid notice) | WR-050, HM-032 "runner-up sealed" |
| D38 | Targets lanes / watchlist / stars | NEW | Watchlist 0 hits | WR-043, HQ-075, HM-044 |
| D39 | Plans (private, steps on calendar events, links, conflicts) | NEW | Plan( hits are MFL move plans | WR-034–042, HM-017 |
| D40 | Calendar events (deadline, tags, FA, draft, cutdown) + window moves | EXISTS | leagueclock Calendar/Decide, app/control CaptureWindows/MergeMFLCalendar | WR-033/037, HM-009, CR-025 |
| D41 | Draft: order, traded-via chain, live clock, picks | EXISTS | app/draft (OpenRoom, Pick, NextDraft), app/market NativePickHistory | WR-054–057, CR-028 |
| D42 | Prospect pool, consensus ADP, tiers | PARTIAL | ADP in ingestion/marketfeeds; tiers + "likely there at #27" new | WR-059 |
| D43 | My draft queue + auto-pick | PARTIAL | leaguerules AutoPick menu; per-user queue store = ? | WR-060/062 |
| D44 | Private scouting reports, traits taxonomy, game context | NEW | "Scouting" 1 hit in app/control (label only) | WR-061 |
| D45 | Private notes per player | NEW | — | HQ-074, WR-069 |
| D46 | Mock draft runner | NEW (local) | — | WR-055 |
| D47 | Archive: champions, seasons, finishes, checks n/9 | EXISTS | app/archive Facts, labcompile/seasons, leaguerecovery | LP-053/054/059 |
| D48 | Owner eras (tenures), roster lineage, key moves, season stories | NEW | OwnerTenure/Lineage 0 hits | LP-049/054/057/058 |
| D49 | Trophy case, record book, season awards | NEW/PARTIAL | Trophy 0; records derivable from archive facts | LP-052/056/060 |
| D50 | Week story / previews / recaps / headlines / awards | NEW | Headline only in research (split out) | LP-044, HM-012 |
| D51 | Stat changes (first-held vs latest) + stat queue | EXISTS | app/control CompareAlerts, commissioner stat queue (CM-031) | LP-046, CR-013 |
| D52 | NFL news feed w/ ownership relevance | PARTIAL | ingestion/sitenews, researchfeeds (research split out 10-28: data kept) | HM-013/030 |
| D53 | NFL depth charts + personnel usage | NEW | DepthChart 0 hits | HM-033–036 |
| D54 | NFL stat tables by position/week range, saved views, CSV | PARTIAL | fantasyscore + nflverse ingest; saved views NEW | HM-037–040 |
| D55 | Player bio (ht/wt/college/draft), career by season incl. fantasy owner then | PARTIAL | playerrecord (AddCareer, CareerMeasures) | HM-041–043 |
| D56 | Alerts (lineup, trade offer, bye, IR) + health/failed-check state | EXISTS | app/control (GatherAlerts, LineupAlert, ByeAlerts, IRAlerts, AlertUrgency) | HM-006/027 |
| D57 | Alert snooze per user | NEW | Snooze 0 hits | HM-027 |
| D58 | Home layout per owner per league, tile registry, shortcuts, pins | NEW | Shortcut 0; Layout hits unrelated | SH-003/006/009, HM-003/008/015/022–026 |
| D59 | Top-5 weekly leaders ticker | PARTIAL | derivable from fantasyscore | HM-007 |
| D60 | Jobs (commissioner job feed) + producers | PARTIAL | app/control alerts + governance; 7 jobs have no producer (vacant, missed, feeds, backup, invites, votes, softcap) | CR-002/011 |
| D61 | Rules catalog (130 rows), change form, preview, pending-rollover | EXISTS | leaguerules + RuleChangeForm (frontend) + commissioner rules wire | CR-020–024 |
| D62 | Rule change history per rule, presets apply/compare | PARTIAL | presets = setting bundles (ruled); history = ledger filter | CR-022/024 |
| D63 | Command registry, confirm levels, safeguards | EXISTS | commandcatalog, CommandSafeguards | SH-008, CR-005/006 |
| D64 | Roles, succession, elections, invites, last activity | EXISTS | governance (HasRole, Invite, ElectionStatus, InactivityEligible, LastActivity, SuccessionPeriods) | CR-044–046 |
| D65 | Permission sets (create/grant/revoke/scope/expiry/staged) | NEW | PermissionSet 0 hits; Allow() rigid tiers | CR-003/049–054 |
| D66 | Exceptions in force (unified, with expiry) | PARTIAL | Exception hits in commissioner/market, history; unified list = ? | CR-035 |
| D67 | Picks: grant/swap/forfeit/condition/void + provenance | PARTIAL | move exists (correct.pick); others ruled not built | CR-037–041 |
| D68 | TOTP status + reset | PARTIAL | store/accounts TOTP; commish reset not built | CR-044/048/063 |
| D69 | Owner last-seen | PARTIAL | store/personal LastSeen, governance LastActivity | CR-017/044 |
| D70 | Notices / newsletter / AI draft | PARTIAL | Notice in app/commissioner, leagueservice; AI draft NEW | CR-055 |
| D71 | Rulebook text + house rules | EXISTS | Rulebook in app/core, leaguerules | CR-058 |
| D72 | Server status, backups, upgrades, feeds health | EXISTS/PARTIAL | serverstatus (Checks), app/platform Backup, serverversion, releasetrust; telemetry (p95, requests) NEW | CR-060–064 |
| D73 | Support / server-admin roles | NEW (ruled R9/R10) | — | CR-065 |
| D74 | AI agent connections (scope, token, last used) | NEW (ruled 10-24) | — | CR-047 |
| D75 | Data freshness per source ("MFL · 12 min ago", "ESPN 4s ago", truth-ladder stage) | EXISTS | app/report (FeedTime, LiveFreshness, StaleFreshness), app/now FeedFresh | HQ-076, LP-005, HM tile "as of" |
| D76 | Search index over CR entities | NEW (UI) | — | CR-004 |
| D77 | Rookie slot salary scale | ? | not checked | HM-018 |
| D78 | "How acquired" per player (draft slot, trade, FA $) | PARTIAL | transactions history | HM-042, LP-057 |

Counts: EXISTS 27 · PARTIAL 31 · NEW 27 · ruling-check 1 (D37) · unchecked 1 (D77)  (some rows dual-tagged)
