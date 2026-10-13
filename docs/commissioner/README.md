# Commissioner map — MFL commissioner office, mapped read-only (2026-10-13)

**What this is:** every control in MFL's commissioner office for Legacy NFL (league 14432), mapped
by Bee (GPT-6.1 Sol) through a read-only browser tool and gated sector by sector by Claude. Nothing in
the league was changed: no form was submitted, no button pressed, no never-open page loaded.
It is step **K1–K2** of the commissioner-surface process below; the companion to the owner map
(2026-10-04/05, 736 nodes).

**TheWarRoom's role (Christopher, 2026-10-13):** TheWarRoom retires MFL and ProBoards. Each MFL
control is judged by one test: *does the league's own platform need this function?*

## Files
| File | What it holds |
|---|---|
| this README | process, the cut (kept/merged/retired/deferred) with Christopher's rulings, league rules summary, open questions |
| `Commissioner_Tree.md` | the full commissioner tree: group › entry › sub-page, URL, outcome (328 rows) |
| `Commissioner_Owner_Overlap.md` | each control × the owner-map nodes that show or act on the same thing |
| `Commissioner_API_and_Build_Paths.md` | MFL API read/write types per control, and which build routes MFL makes possible (facts, not decisions) |
| `Commissioner_Governs_Edges.md` / `.tsv` | kept controls × the owner pages their settings change (193 edges): seeds the registry `governs` column |
| `Commissioner_Never_Open_Register.md` | every entry never opened (may act on load) or refused by the guard, with what its label says it does |

**Not in this public repo:** the per-sector page reports, screenshots, captured field values, and
league-private data (every player's salary/contract, 291 salary adjustments, owner details). They are
in the Beelink run dir `~/fleet/runs/bee-mfl-commish-map-2026-10-13/` (reports C1–C5, C7, captures).

## How it was mapped
- Public API/help research first (Bee): `~/fleet/runs/bee-mfl-commish-research-2026-10-13/`.
- Read-only tool `~/fleet/bin/mflmap.py`: exact allow list of 60 reviewed commissioner pages;
  a never-open list (Clear Rosters, Recalculate Standings, Revert/Skip/Archive Draft, Delete
  Transactions, Process Waivers, Randomize Schedule, Auto-seed, Delete/Replace Franchise, …); `import`
  and `live_draft` blocked; no clicks inside forms; action words refused on links and script buttons.
- Sectors C1 General + Franchise · C2 Players and Rosters · C3 Scoring and Standings · C4 Salary,
  Add/Drop, Waivers, Trades/IR/Taxi · C5 Schedule, Draft, Playoffs · **C6 Appearance/Messages/History/
  Caution skipped by Christopher** ("none of that matters") · C7 synthesis (offline).

## Counts

| Measure | Count |
| --- | --- |
| Accepted C1–C5 index rows read | 297 |
| Historical C3 menu placeholders superseded | 4 |
| Current C1–C5 tree rows | 293 |
| C6 skipped menu rows | 35 |
| Full tree / each of tables 2–4 | 328 |
| Menu controls (C1–C6, aliases retained) | 120 |
| Accepted Settings-now blocks | 69 |
| Kept menu controls | 46 |
| Governs edges | 193 |
| C1 current tree rows | 62 |
| C2 current tree rows | 33 |
| C3 current tree rows | 102 |
| C4 current tree rows | 29 |
| C5 current tree rows | 67 |
| C6 current tree rows | 35 |
| never-open (rule) | 106 |
| not visited (guard: NOT LOGGED IN; no franchise proof) | 1 |
| not visited (guard: NOT LOGGED IN; no owner franchise on public view) | 1 |
| not visited (guard: matches deny word csetup) | 6 |
| not visited (guard: parameterized csetup denied) | 4 |
| not visited (guard: privacy) | 2 |
| not visited (guard: prohibited form submission) | 1 |
| not visited (guard: same FMOBILE function already denied) | 2 |
| not visited (guard: sector stopped after NOT LOGGED IN) | 11 |
| opened | 69 |
| skipped (Christopher) | 35 |
| variant | 90 |


## Commissioner surface: research → plan → build process (2026-10-13)

Christopher, 2026-10-13: do the same complete research and planning session for the commissioner
control surface that the owner UI got, for conformity. Because the commissioner surface controls the
league surface and the owner experience, double the effort on front-to-back wiring. The command line
(`/command`, the command bar) is the AI's tool for finding and closing wiring gaps.

### How the owner side was done (the template)
| Step | Owner side (what happened) | Evidence |
|---|---|---|
| 1 Map | Bee mapped MFL's owner menus + home page, 11 gated sectors, 736 nodes, PDF | `~/fleet/runs/bee-mfl-ui-map-2026-10-04/`, RESUME-2026-10-05-mfl-ui-map.md |
| 2 Research | Two independent IA passes (Sol, GLM) on the same brief | `~/fleet/briefs/warroom-endpoint-ia-bee-2026-10-06.md`, `…-pass2-…` |
| 3 Target spec | Passes merged; Christopher's rulings binding; revised once per ring | `docs/ui/Target_UI_Spec_2026-10.md` (rev 3) |
| 4 Registry | Every endpoint a row: gm_question, gravity G0–G3, ring, disposition kept/merged/retired, verb, placement, status spec→built→landed | `docs/ui/endpoint-registry.csv` (270 rows) |
| 5 Rings | Ring 0 walking skeleton → rings 1–3; Claude head brain, Sol codes chunks, Claude reviews + planted mutant, live click-through on Claude-OS, Christopher "pass" → merge | RESUME-…-ring0..ring3 |
| 6 Landing | Walk every built row live, fix, mark `landed` | RESUME-2026-10-12-ring3-land.md |

## The commissioner side: same steps, same artifacts
| Step | Commissioner side | Done when |
|---|---|---|
| K1 Map | Bee (Sol 6.1) maps the commissioner menu, sectors C1–C6 + C7 synthesis, gated by Claude | every entry opened or recorded with reason; Markdown in repo |
| K2 Cut | kept / merged / retired / deferred per control, Christopher corrects | his corrections applied (first pass: `FIRST-PASS-dispositions.md`) |
| K3 Research | Two independent passes (Sol + GLM, same brief): "maximize the commissioner control surface" for a 32-team dynasty, given the kept rows, the owner registry and the rulings | both answers on disk |
| K4 Target spec | Merge into a commissioner section of the Target UI Spec (same doc, same revision rules), Christopher's rulings binding | his sign-off |
| K5 Registry | Commissioner rows in the SAME registry CSV (own id prefix), same columns, plus the wiring columns below | every kept row present |
| K6 Rings | Same ring loop as the owner side | per ring: Christopher "pass" |
| K7 Landing | Walk every row live AND run the wiring proof | every row `landed` and every governs-edge green |

## Doubling up on wiring (commissioner-only additions)
1. **Governs edges.** Each commissioner row lists the owner rows it changes (e.g. Roster Limits →
   roster view, IR/taxi Acts, bid pre-flight). New registry column `governs` = owner row ids.
2. **Wiring proof per edge, driven by the command bar.** For every commissioner verb
   (`commish.<verb> key=value`), a test sets the value through the command, then reads each governed
   owner endpoint and asserts the change. This runs headless in CI (the command registry is the
   grammar) and once live in the landing walk.
3. **Round trip.** Change → owner surface shows it → revert → owner surface shows the old value.
   Nothing is wired until both directions pass.
4. **Orphan checks, both ways.** A commissioner row with no governs edge, and an owner rule value
   with no commissioner row that sets it, are both build failures (registry test).
5. **The commissioner keeps full control** (memory project_warroom_commish_full_control): every
   automated rule keeps a manual commissioner override, and the override is itself a command.

## Christopher's rulings (2026-10-13)
- **Roles:** the humans decide who holds which role; TheWarRoom provides the permission hierarchy
  (commissioner, co-commissioner, DOT, owner, and what each may do). K3 must design that hierarchy;
  the app assigns nobody on its own.
- **K3 research:** at least two independent passes on different models with different reasoning
  (Sol + GLM as the owner side, same brief, neither sees the other), so the bases are covered. Claude
  merges and flags every disagreement for Christopher.

## Same principles as the owner side (Christopher, 2026-10-13)
MFL's commissioner menu is as much of a nightmare as its owner menus. The commissioner surface is
NOT a copy of MFL's 16 groups. It obeys the owner-side rulings in Target_UI_Spec §1–§6:
- two-click budget to a ready action; typing a command costs 0 clicks;
- the inspector is the only entity page; four channels (colour = status, chrome = consequence
  G0–G3, countdown = urgency, glyph = classification);
- every control is a command (`domain.verb key=value`) in the one command registry;
- every Act is an envelope (prepared, pre-flight checked, executed, verified);
- snappy UI is law; slop is banned; excellence is the standard.

## K3 scope addition: CLI / chat readiness (not the CLI/chat build)
Christopher, 2026-10-13: the target is BOTH a typed command line (natural language mapped to
commands) and a league-aware assistant that proposes actions for the human to confirm, as one
surface. The real CLI/chat build is its own independent campaign. K3 does not design it. K3 only
makes sure the commissioner interface and system wire to it:
- every commissioner control is a registry command (`domain.verb key=value`) with typed arguments;
- every command declares its permission level and its gravity (G0-G3), so a later chat surface can
  confirm stakes / irreversible commands and refuse what the role may not do;
- no commissioner capability exists only as a screen interaction.
- **Defaults (2026-10-13):** every kept control ships with a default, and defaults are logical (what
  a sensible league does, e.g. draft order: 1st pick is the worst team). K3 states the logical default
  for every kept control; the commissioner can change any of it.

## The cut: kept / merged / retired / deferred (first pass by Claude, corrected by Christopher, 2026-10-13)
Test: does the Legacy NFL's own platform (TheWarRoom, replacing MFL + ProBoards) need this function?
kept = the platform must do it · merged → folds into a kept control or something TheWarRoom already has ·
retired = MFL machinery for other league types. C6 (appearance, messages, history, caution) was skipped by Christopher 10-13: its retired rows stand without a page map; League Awards is kept.

| Group | Entry | Disposition | Reason |
|---|---|---|---|
| General | General League Setup | kept | league shape: 8 div / 2 conf / 32 teams / 80 roster / 12 IR / 8 taxi |
| General | League Setup Next Steps | retired | MFL new-league checklist |
| General | Division and Conference Setup | kept | divisions/conferences drive schedule, standings, playoffs |
| General | Accounting Setup | retired | no real-money accounting until the app is fully mature (Christopher 10-13) |
| General | Edit Accounting Records | retired | same |
| General | LeagueSafe | retired | third-party fee collection |
| General | Additional League Rules Setup (O=60) | kept | league rule settings |
| General | League Calendar Setup (O=110) | merged → calendar | league dates already built (#19, rulebook dates, commissioner-editable) |
| General | Pool Setup | deferred → side-games suite | pick'em + survivor; full side-games suite planned (Christopher 10-13) |
| Commish/Franchise | Franchise Setup | kept | franchise identity and owner seats |
| Commish/Franchise | Invite Franchise Owners | kept | onboarding owners to the platform |
| Commish/Franchise | Abilities Setup | kept | owner permissions |
| Commish/Franchise | Remove Owner from League | kept | owner turnover |
| Commish/Franchise | Commissioner Email Setup | merged → comms | notification settings |
| Commish/Franchise | Grant Commissioner Access | kept | commissioner / DOT roles |
| Commish/Franchise | Custom Franchise Order | retired | display ordering |
| Rosters | Starting Lineup Setup | kept | 21 starters, position min/max, IDP 12 |
| Rosters | Roster Position Limits Setup | kept | roster min 18 / max 80 |
| Rosters | Load/Unload Rosters | merged → correction ledger | commissioner roster fix (Session 2 CORRECT backend) |
| Rosters | Clear Rosters | retired | new-league reset |
| Rosters | Transfer 2025 Rosters To 2026 | kept (as season rollover) | dynasty year turnover is a platform job |
| Rosters | Copy Rosters To Previous Weeks | retired | MFL data repair |
| Rosters | Edit Custom Players | retired | MFL player-db workaround |
| Rosters | Change Player Position | kept | commissioner has full control; err toward the human deciding (Christopher 10-13) |
| Rosters | Dynasty and Keepers Setup | kept | dynasty rules |
| Rosters | Load Keepers | merged → season rollover | |
| Rosters | Player Acquired Info Setup / Edit | merged → acquisitions | provenance already built (Session 1, acquisitions) |
| Rosters | Can't Cut / Can't Add Lists | retired | not wired into TheWarRoom (Christopher 10-13) |
| Scores/Standings adj | Adjust Franchise Scores (+ Import) | kept | commissioner score correction |
| Scores/Standings adj | Adjust Player Scores (+ Import) | kept | stat correction at player level |
| Scores/Standings adj | Recalculate Standings | merged → standings | the platform computes; no manual button |
| Scores/Standings adj | Custom Standings Order / Break Ties | merged → standings rules | tiebreakers are rules |
| Scores/Standings adj | Adjust Standings (+ div/conf/VP adjusters) | kept (one control) | commissioner standings override |
| Scoring setup | General Scoring Setup / Custom Scoring System | kept | Layer 2 scoring matrix source |
| Scoring setup | Statistic Changes | kept | stat corrections |
| Scoring setup | Select A Packaged Scoring System | retired | MFL presets |
| Scoring setup | Standings Setup | kept | how standings rank |
| Scoring setup | Order Standings Columns | retired | display |
| Scoring setup | Victory Points Setup | retired | configured (win 2, top-third 2) but not in standings criteria or rulebook; spec rev3 retired M-075 |
| Salary cap | Salary Cap Setup | kept | cap rules |
| Salary cap | Edit Player Salaries and Contracts (+ Import) | merged → contracts | contracts already built |
| Salary cap | Edit Salary Adjustments | kept | dead cap / adjustments |
| Salary cap | Select Packaged Player Salaries | retired | MFL presets |
| Appearance | HPM/Tabs, Skin, League Links, Icons, Images, Scrolling Text, YouTube | retired | MFL page dressing |
| Appearance | Reports and Security Settings | merged → abilities | who sees what |
| Add/Drop & Waivers | General Add/Drop Setup | kept | add/drop rules |
| Add/Drop & Waivers | Waivers/Free Agents Setup | kept | UFA bid rules (bids built) |
| Add/Drop & Waivers | Custom Waiver Order | merged → waivers | |
| Add/Drop & Waivers | Transaction Calendar Setup | merged → calendar | |
| Add/Drop & Waivers | Manually Process Waivers | kept | bid resolution is a platform job |
| Add/Drop & Waivers | Delete Transactions | merged → correction ledger | |
| Other transactions | Review Trades | kept | DOT review |
| Other transactions | Trades Setup | kept | deadline, trade rules |
| Other transactions | Injured Reserve Setup / Taxi Squad Setup | kept | IR/taxi rules |
| Schedule | Weekly Schedule Setup | kept | schedule (league schedule built) |
| Schedule | Rearrange / Switch Weeks / Validate / Generator | merged → schedule | |
| Schedule | Select Packaged / Randomize / Import | retired | MFL presets |
| Draft | General Draft and Timer Setup | kept | rookie draft room |
| Draft | Draft Order (All Rounds / One Round) | kept (one control) | |
| Draft | Change / Skip Draft Pick | kept | Christopher 10-13: the commissioner gets the complete draft control surface |
| Draft | Revert Draft | kept | Christopher 10-13: the commissioner gets the complete draft control surface |
| Draft | Draft Start Date/Time | kept (shown on calendar) | Christopher 10-13: the commissioner gets the complete draft control surface |
| Draft | Import Future Draft Picks | merged → assets | future picks are tradable assets |
| Draft | Archive Draft | kept | Christopher 10-13: the commissioner gets the complete draft control surface |
| Playoffs | General Playoffs Setup | kept → graphical bracket builder | Christopher 10-13: fluid brackets, built graphically from presets, so any league format fits |
| Playoffs | Playoffs Schedule / Auto Seed / Manual Seed | merged → bracket builder | Christopher 10-13: fluid brackets, built graphically from presets, so any league format fits |
| Playoffs | Select Packaged Playoff Bracket | kept (as bracket presets) | Christopher 10-13: fluid brackets, built graphically from presets, so any league format fits |
| Messages | Weekly Newsletter | merged → comms | |
| Messages | HP message #2 Custom Rules | merged → rulebook | |
| Messages | HP message #6 History | merged → league history | |
| Messages | HP message #18 Commissioner Abilities | merged → abilities | |
| Messages | other HP messages (#1,3–5,7–17,19,20) | retired | page dressing |
| History | League History Setup / Link Historical Franchises | merged → league archive | built |
| History | League Awards | kept → with side-games suite | Christopher 10-13: kept; some mini/pool games will be built around awards. C6 not mapped (skipped) |
| Caution | Duplicate This League | retired | |
| Caution | Delete A Franchise | retired | fixed 32 |
| Caution | Replace/Reset Franchises | merged → owner turnover | |

## League rules as configured on MFL (2026-10-13, read-only)
Topic-level values from accepted C1–C5 pages. Per-player salaries, salary adjustments and owner
details are league-private and stay in the run dir.

| Topic | As configured |
|---|---|
| League shape | 32 franchises · 8 divisions · 2 conferences · head-to-head schedule · salaries tracked · initial rosters by draft · dynasty mode |
| Roster | min 18 · max 80 (IR and taxi excluded) · IR 12 · taxi 8 · no position roster minimums |
| Lineup | exactly 21 starters: QB 1 · RB 1–3 · WR 2–5 · TE 1–3 (QB+RB+WR+TE ≤ 8) · PK 1 · DT/DE/LB/CB/S 2–4 each · 12 IDP |
| Standings | through week 17 · order: overall win % › head-to-head › division win % › conference win % › all-play win % › total points › power rank · victory points configured but unused |
| Cap / contracts | cap $125, soft · IR and taxi salary count 100% · dropped player leaves 35% (dead money) |
| Add / drop | no season or weekly add limits · waiver requests for locked players, first-come for the rest, commissioner processes |
| Trades | every accepted trade goes to commissioner approval · no poll, no auto-reject · future picks 2 years, rounds 1–5 · invalid-roster trades refused · offers expire after 7 days · no trade-deadline field on MFL |
| Draft | email draft · 6-hour pick clock · expired clock = skip (no auto-pick) · unlimited timeouts |
| Playoffs | AFC and NFC brackets of 7, from week 14, 3 first-week games · Legacy NFL Super Bowl week 17 · winner title still reads "2025 Super Bowl Champion" (stale label, not changed) |
| Owner abilities | all 32 identical: 13 on (lineups, add/drop, drop-only, trades, pick trades, IR, taxi, articles, board, chat, polls, survivor, NFL pool) · off: customize franchise, customize home page |

## Open questions for Christopher (from C7; nothing changed)

| FIRST-PASS entry | Unchanged disposition | Unchanged reason | Accepted evidence / question |
| --- | --- | --- | --- |
| Additional League Rules Setup (O=60) | kept | league rule settings | C1 shows a blank free-text Additional rules textarea, not structured rule settings. O=26 is its owner display. Does kept mean the text/rulebook capability, or a separate rule editor? |
| Custom Standings Order / Break Ties | merged → standings rules | tiebreakers are rules | C3 Custom Standings Order is an arbitrary 32-franchise manual ordering override (none active), not the ranking-criteria setup. Does merged → standings rules also need to preserve this override behavior? Break Ties is never-open; do not infer its exact workflow. |
| Waivers/Free Agents Setup | kept | UFA bid rules (bids built) | C4 FREEAG exposes add limits, rookie inclusion and minimum roster stay, not bidding amount/budget/scoring fields. C4 ADDDROP is waiver requests for locked players/FCFS for the rest, with manual processing. Is kept intended to cover those acquisition constraints as well as the planned UFA bid rules? |
| Manually Process Waivers | kept | bid resolution is a platform job | Never-open O=47 is labelled generic waiver processing; no bid-specific resolution controls or engine were inspected. C4 current acquisition mode is not demonstrated as a blind-bid process. Confirm whether the retained platform job should resolve all claims, not only bids. |
| Trades Setup | kept | deadline, trade rules | C4 TRADES exposes approval, future-pick scope, validity, comments and proposal expiry but no trade-deadline field. The calendar mechanism can govern transaction windows; C1’s only saved calendar event is Draft. A configured trade deadline cannot be supplied from these sources. |
| Draft Start Date/Time | kept (shown on calendar) | Christopher 10-13: the commissioner gets the complete draft control surface | C5 is an alias of O=110, not a separate draft-settings page. Its kept (shown on calendar) disposition and C1 calendar’s merged → calendar disposition coexist on one URL; the registry must preserve label/context, not deduplicate disposition solely by URL. |
| Adjust Standings (+ div/conf/VP adjusters) | kept (one control) | commissioner standings override | The kept (one control) row includes the Victory Points Adjuster, while Victory Points Setup is retired. No changed disposition is proposed; confirm whether the kept unified override retains a dormant VP field or excludes it. |


### Answers (Christopher, 2026-10-13)
- **Custom Standings Order:** yes, the app keeps the commissioner's manual override of the full standings order.
- **Process Waivers:** the app resolves ALL claims. Bids only matter after waivers have cleared.
- **Trade deadline:** set on ProBoards today; in the app it is set on the league calendar.
- **Draft Start / League Calendar same page:** registry note only, no decision needed.
- **Additional League Rules (free-text rules):** parked with the first-class CLI/chat campaign. The idea
  worth keeping: a commissioner writes a rule in words and the platform turns it into an enforced rule.
- **Waivers (how the league runs them):** a player hits waivers only when he is cut; he stays on waivers
  for 24 hours; the waiver order is set by league rules. Bids apply only after waivers clear. These are
  the waiver settings the app needs (each with that logical default); MFL's add limits, rookie-waiver
  switch and minimum-stay days are not part of the model.
- **Victory points:** dropped, including the victory-points adjuster inside Adjust Standings.
