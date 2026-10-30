# UI direction v2: command center (2026-10-30)

Rulings from Christopher's UI/UX groundwork session. Amends `UI_Direction_Document.md` and `Target_UI_Spec_2026-10.md`; the four-channel visual system (colour = meaning, frame = consequence, countdown = urgency, glyph = classification) still governs.

## Look
- Contrast is a textured rubber grip / safety edge on heavy equipment: knurl texture, hard borders, monochrome hazard stripes. Never glow, bubbles, soft gradients or blur.
- Hover depth is subtle: 1–2px transform plus a hard offset shadow, under 80ms, transform/opacity only. Hover never lifts past an item's G-level frame.
- Alert colour is an accent (bar, label, dot), not a wash. Text stays high-contrast.

## Space
- A cinematic terminal: the page never scrolls. A fixed-height table may scroll inside its own frame.
- Extra detail opens in drawers: side drawer for a task, bottom drawer for "right now" data, nested drawer for confirms only (max stack 2), peek strip for previews. Player and team detail always goes to the inspector.
- Rows stay compact (30px). Spare screen space gets real function (matchup, form, inline taxi/IR, ledgers), never stretched rows. Primary target 1920x1080; checked at 2550x1352.

## Rail
- Opens on click. Six fixed nodes; leaves never move.
- One pinned NOW line per node, set by season phase.
- "My shortcuts" group at the top, filled by the owner in Customize mode (pin ☆, drag grip).

## Interaction
- Position tags are drag handles. Bench → lineup, lineup → bench, lineup ↔ lineup; the lineup auto-adjusts (bumped starters cascade into weaker eligible slots, holes auto-fill from the best eligible bench player not on bye).
- The app never second-guesses a deliberate move. If the user commands badly, they lose.
- Commit controls live only in action trays and drawers, never in read tables. G3 is a ~600ms hold.
- Owners can add their own inspector drawers (marked MINE).

## Approved additions
Function with form is welcome; slop is not. Approved on HQ: this-week matchup panel with win %, last-5 form bars, inline taxi/IR on tall screens, decisions-due list, cap runway, dead-cap ledger, season move ledger. Win % is new engine math; nothing computes it yet.

## Mocks (docs/ui/mocks/)
- `rail-mock-v1.html`: rail options (option 3 ruled).
- `franchise-hq-mock-v10.html`: Franchise HQ, all three workspaces, RFA/UFA contract caps, content-sized release drawer, release drawer with G3 hold, Customize mode.
- `glm-rail-funnel-2026-10-30.md`: GLM 5.3 rail tree. Leads only; triaged in session.

## Contract end (10-30)
- Final-year cap shows RFA (lime) or UFA (signal yellow) as a textured rubber block with a legend. These hues are classification only and sit outside the five meaning colours; always paired with a letter or legend.
- Drawers are sized to their content; a confirm drops in below the actions instead of covering them.
- Bottom drawers sit above the tray so the pull tabs stay visible, size to their rows, and pair a data table with a summary side panel (Taxi: could-stash list; IR: candidates + return timeline; What-if: stackable moves with live cap/roster/projection totals).
- A closed drawer is fully gone: no peeking header. Every add or change must have a functional or aesthetic reason.
- Row data packs beside the name it describes (form bars next to the player, not at the far edge). Form bars: last 5 weeks, bright = beat this week's projection, dashed line = projection, hatched = bye, avg beside, values on hover; on bench rows too.
- My Moves is a queue, not a workspace tab: its own control in the tab bar with counts (needs you / waiting / landed).

## War Room (10-30)
- Trade Floor folds into War Room: the back room for consequential planning, transactions and the offseason. Five areas: Table (direction), Trade Desk, Market, Draft Room, Plans.
- Trade Desk: two-lens calculator with "their read"; send/get trays organized with colour definition; A/B/C versions. Pocket is a drawer of saved unsent trades with stale flags.
- Market: a draft bid can be removed; a placed bid is final, and only being outbid invalidates it.
- Draft Room: the center scouting card is approved (an exception to inspector-only, still to be written into the rule sheet). Draft order shows rounds, traded picks and live fill.
- Plans is a private thinking desk. It holds checklists per calendar event, ranked needs and live summaries of Trade Desk, Market and Draft Room. Nothing executes from Plans.
- The bottom calendar strip is reserved for future franchise-sim play controls. Reserve the space only; don't build it.
- Players, picks and names are physical draggable objects everywhere.
- Never invent system states for owner judgments ("PRICED OUT", "NEXT UP"). Show only states the machinery really produces.
- Filter buttons look like real machine buttons. Screens are built for long hours: easy on the eyes.
- Palette: smoky.

## League Pulse (10-30)
- Three tabs: Live · Race · Archive. No new rail node. Matchups is not a tab; it lives inside Live.
- Palette: jet black (neutral black, soft warm-white text at about 84%), with a truer-green field. It is the most-used and TV screen, so it must never glare. The War Room stays smoky and HQ stays cool grey.
- Raised cards keep the page colour inside. Lift comes only from a light border, a hard black shadow and a hazard top edge. A lighter card fill is rejected.
- Live is the game-day / TV screen:
  - Left: MATCHUPS | NFL GAMES toggle, opening on Matchups. My game is pinned, then the NFC and AFC sides. Each row has a win % per team, coloured text only (green ≥65, amber ≤35); no LIVE tags.
  - Clicking a matchup fills the center card; MY MATCHUP ‹ returns to mine.
  - Center: field and follow bar on top, with a light ball/LOS slide on each new play.
  - Center card: 16 v 16 starters in OFFENSE/DEFENSE groups, shorthand stat lines ("16-71 rush | 0-23 rec") with solid TD tags first, a clock chip, and a football icon when the player is on the field (offense when their team has the ball, defense when the opponent has it).
  - Right: IN THIS GAME at natural height (never cut off); the selected game's play-by-play fills the rest.
  - Bottom pulls: MATCHUP BOX, WEEK STORY, SCHEDULE, STAT CHANGES and NFL BOX SCORE. Each follows the selected matchup.
- Race:
  - Standings: SEEDS | DIVISIONS × AFC | NFC, with power rank and odds columns. My Road shows my win % per remaining game.
  - Power Rankings is first-class, because most of the year is offseason. Three horizons:
    - THIS SEASON: form 30, lineup 30, health 15, all-play 15, run-in 10.
    - DYNASTY · 3 YR: Best 21 value 35, age 20, picks 20, cap 15, depth 10, plus peak year.
    - BLEND: 60/40.
  - Every part shows a rank of 1–16, with weights visible. The score is a number followed by a thin bar; text never sits on the bar. Each team carries a window label, and VS RECORD shows seed minus power rank.
  - Rank movement is not drawn in the table. WHY IT MOVED has THIS WEEK (notes) / THIS SEASON (segmented rank charts: square per week, green climb, amber fall, zoomed to the team's range) / LAST 12 MONTHS (blend rank plus trades, drafts and FA).
  - Playoff Picture: seed now → projected. The run-in grid covers wks 10–14 with opponent and win % cells; division leaders are outlined. Flags read "N IN A ROW vs ★" / SOFT FINISH, and a team that projects out of the playoffs is tagged BUBBLE. Odds follow the projection. Below sit the projected bracket and THIS WEEK MOVES IT.
  - Engine needs: per-game win probability from power, run-in projection, and dynasty components from valuation. None of these exist yet.
- Archive: full franchise history, defaulting to my team. Eras by owner explain "how did we get here". The chart has stamps (DIV TITLE / PLAYOFFS / PLAYOFF WIN / conference champ / CHAMPION). A trophy shelf opens a full-screen TROPHY CASE. FRANCHISE | LEAGUE toggle.
- Open: Race default view by phase (power in the offseason?); window labels shown league-wide (yes/no).

## Mocks added (10-30)
- `warroom-mock-v6.html`: War Room, current.
- `league-pulse-mock-v14.html`: League Pulse, current. v1–v13 are history.
- Research: `research/league-pulse-reorg-opinion-2026-10-30.md`, `plans-tab-opinion-2026-10-30.md`, `scouting-notes-research-2026-10-30.md`.

## Home (10-30)
- Three subtabs: BRIEF · DEPTH CHARTS · NFL STATS (rail leaves under Home). Palette: SLATE (ruled 10-30): dark blue-grey, distinct from HQ cool grey, War Room smoky, Pulse jet black.
- Brief: customizable 3x2 tile grid (default subject to season phase; custom never moves on its own), overflow drawer instead of displacement, one-row alert band outside the layout (all screens), NFL news front and center, broadcast-style top-5-per-position ticker, transactions with typed blocks (WIRE = solid green).
- Full cards sized to content: news (blurb + player panel), trade (side by side), other transactions (narrow). Player career card (MFL-style: bio, league contract, NEWS / season STATS with FFB pts / PROJ / CAREER) opens from any player name on Home. Ruled exception to inspector-only, like the Draft Room scouting card.
- Depth charts: team picker by division, OFF / DEF / ST five-deep grid, names coloured by league ownership. Source TBD (Sleeper depth order / ESPN); Ourlads is reference only.
- NFL stats: saved views, position / weeks / ownership / team / minimum filters, sortable, FFB PTS and FFB/G scored with league rules.
- `home-mock-v5.html` is current; v1–v4 are history.

## Rulings (10-30, late)
- Race defaults to Power Rankings in the offseason.
- Window labels ("win now", "rebuilding") show league-wide.
- Center cards (scouting, player career, news, trade) are an approved exception to inspector-only; more areas will use them.
- Control Room covers both day-to-day commissioner jobs and league setup/settings; co-commissioner gets near-parity.
- Open: "one-of bids" (to be re-explained).
