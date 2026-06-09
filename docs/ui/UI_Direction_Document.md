# Legacy NFL — UI Direction Document
**Version:** 1.0 — June 2026
**Status:** Locked. Load this document at the start of every UI build session.
**Authority:** This document governs all UI and frontend decisions. It does not replace the Engine Specification, Official Rulebook, or Module Specifications — it references them. Where this document and another conflict on a UI matter, this document wins.

---

## 1. How Claude Code Uses This Document

Read every section before writing a single file. This document is the output of a dedicated UI architecture brainstorming session. Every decision was deliberated and locked. Nothing here is a guess.

**Do not reopen locked decisions.** They are marked as such. If a locked decision creates a technical constraint that feels wrong, flag it to Christopher — do not route around it silently.

**Do not add features not in this document** without Christopher's explicit direction. The scope is defined. Additions create scope creep that breaks the architecture.

**Deferred items are listed in Section 20.** Do not design or build them in Phase 1. They are roadmapped, not forgotten.

**This document works alongside:**
- `North_Star.md` — strategic purpose and the four pillars
- `Engine_Specification.md` — scoring engine math and layer architecture
- `Module_Specifications.md` — detailed module input/output specs
- `Official_Rulebook.md` — authoritative source for all transaction rules
- `MFL_API_Specification.md` — API structure and data pipeline

---

## 2. Project Overview

The Legacy NFL application is a full-stack fantasy football intelligence and transaction platform built as a harness on MyFantasyLeague (MFL). It is not a generic fantasy tool. It is purpose-built for a 32-team dynasty league with a custom IDP-heavy scoring system.

**What it is:**
- A valuation engine surfacing intelligence MFL's stock interface cannot provide
- A transaction system that enforces rulebook constraints and removes human error
- A communication platform that replaces the Proboards forum entirely
- A desktop-first, cross-platform application

**What it is not:**
- A replacement for MFL in Phase 1 or Phase 2
- A generic fantasy platform
- A mobile application (Phase 1 and Phase 2)
- An AI agent that makes decisions for GMs

**Build phases relevant to UI:**
- **Phase 1:** Christopher's personal tool. One user. One league. Read-only against MFL. Desktop only.
- **Phase 2:** 32 Legacy NFL GMs. Multi-user. Authenticated MFL access. Desktop only.
- **Phase 3:** Public multi-tenant. Other leagues. Mobile harness. League-branded themes.

The UI must be built for Phase 1 but architected to reach Phase 3 without a rebuild.

---

## 3. Technology Stack

Every item in this section is locked. Do not substitute without Christopher's explicit approval.

| Component | Technology | Version | Rationale |
|---|---|---|---|
| Desktop framework | Wails | **v2 (stable GA)** | v3 is alpha as of June 2026. v2 is production-locked, fully documented, zero breaking risk. |
| Backend language | Go | 1.21+ | Scoring engine implemented natively in Go. No Python sidecar. No multi-runtime complexity. |
| Frontend framework | React | Latest stable | Via Wails React template. |
| Styling | Tailwind CSS | Latest stable | Standard component layout. Custom work for admin and commissioner consoles. |
| Client state | Zustand | Latest stable | Lightweight. Prevents unnecessary re-renders during high-volume array sorting. |
| Local database | SQLite | — | Offline-first. Single file. Zero configuration. |
| Theme state | CSS Custom Properties | — | **Never React state or Zustand for theme.** CSS variables only. |
| List rendering | Virtualized | — | React-window or equivalent. Non-negotiable. |

**Critical note on Go and Python:** The engine specification may reference Python in earlier drafts. The implementation language is Go. The 6-layer scoring engine is written entirely in Go within the `core/engine/` directory. There is no Python interpreter, no Python sidecar, no subprocess call to Python at any point.

**Project directory structure (locked):**

```
legacy-nfl-app/
├── main.go
├── wails.json
├── app.go
├── core/
│   ├── engine/             # Layers 1-6 scoring engine in Go
│   ├── models/             # Immutable data structures
│   └── rubrics/            # Position-specific S-curve parameter matrices
├── infrastructure/
│   ├── database/           # SQLite driver, migrations, local cache
│   ├── mfl/                # MFL API ingestion pipeline
│   └── config/             # Admin control parameters
├── interface/
│   └── bindings/           # Wails-exposed functions for the frontend
└── frontend/
    ├── src/
    │   ├── components/     # Standard Tailwind-based UI components
    │   ├── admin/          # Commissioner and Admin console views
    │   ├── store/          # Zustand state slices
    │   └── App.jsx
    └── package.json
```

---

## 4. Core Architectural Principles

These are not preferences. They are hard constraints that every file must respect.

**Principle 1 — Modular workspace**
The application is a personal desktop environment, not a fixed-layout web app. Every panel is a widget. Every widget has controls. The default state ships clean and organized. Power user configuration is one unlock away. Nothing is hard-coded to a fixed position.

**Principle 2 — Edit Mode (Option A) for layout customization**
Layout customization requires an explicit "Customize Layout" toggle. In Edit Mode, panels get drag handles, resize grips, and remove buttons. Outside Edit Mode, the layout is locked and stable. A GM mid-bid during a game week cannot accidentally rearrange their workspace.

**Principle 3 — CSS Custom Properties as the sole source of theme truth**
All density tokens, color values, spacing scales, and typography live as CSS custom properties at the `:root` level. Switching themes or density is a single `data-*` attribute change on the document root. Zero JavaScript re-renders. Zero React updates. One frame. Components read CSS variables. They never subscribe to theme state.

```css
/* Correct pattern */
[data-density="narrative"] { --row-height: 72px; --font-size-data: 14px; }
[data-density="tactical"]  { --row-height: 52px; --font-size-data: 12px; }
[data-density="matrix"]    { --row-height: 32px; --font-size-data: 11px; }

/* Never do this */
// const [density, setDensity] = useState('tactical');
// components subscribing to density state = unacceptable
```

**Principle 4 — Density never hides data. It hides display.**
Every data field flows through every pipeline stage at all times. CSS controls visibility. JavaScript never has conditional logic tied to what density mode is active. Hidden columns in Narrative mode still exist in the data. The engine does not skip fields based on the current density.

**Principle 5 — Density drives API payload scope**
The backend sends tiered payloads based on active density:
- Narrative: name, position, team, adjusted score, contract tier, injury status
- Tactical: above + snap share, target share, xFP, trend direction, breakout flag
- Matrix: above + YPRR, TPRR, EPA, RAS, all Layer 4 sub-signals, full contract detail

Switching density from Narrative to Matrix triggers a scoped data request. The CSS variable switch is instant (one frame). Data enrichment follows asynchronously and populates progressively. No spinner. No blank screen. Progressive reveal.

**Principle 6 — List virtualization is non-negotiable**
A 32-team roster matrix is potentially 1,500+ player records. The player list renders only visible rows — approximately 15-20 at any time. All others exist in data, not in the DOM. This constraint goes in on day one. It cannot be added cleanly after the fact.

**Principle 7 — Clean Architecture separation**
The `core/` module knows nothing about SQLite, MFL, or the frontend. It only knows the math in `Engine_Specification.md`. If MFL changes its API structure, only `infrastructure/mfl/` changes. The scoring engine is untouched.

**Principle 8 — Phase 1 is read-only against MFL**
All transactions are generated by the application and executed manually on MFL by the GM. No authenticated write calls in Phase 1. Transaction outputs are correctly formatted instruction sets, not API submissions.

---

## 5. Application Shell

The shell is the persistent container that wraps all modules. It never reloads. Modules render inside it.

**Four-column layout:**

```
+-------------+--------------------------------+-------------------+------------------+
|             |                                |                   |                  |
|  NAV RAIL   |    FLUID WORKSPACE             |   CONTEXTUAL      |  COMMS PANEL     |
|  (fixed)    |    (main content area)         |   INSPECTOR       |  (collapsible)   |
|             |                                |                   |                  |
|  ~158-200px |    flexible                    |   ~196-320px      |   ~280px open    |
|             |                                |                   |   ~48px collapsed|
+-------------+--------------------------------+-------------------+------------------+
```

**Panel behaviors:**
- Every panel has two persistent controls on hover: a collapse toggle and a settings gear
- Collapse toggle reduces the panel to an icon strip, returning real estate to the workspace
- Settings gear exposes what that specific panel shows, in what order, with what defaults
- All panel states (width, collapsed/expanded, visible/hidden) persist per user profile
- On large monitors: all four columns visible simultaneously
- Communication panel collapses to a notification icon strip by default when not actively used

**The Contextual Inspector** populates when any player row, asset card, or entity is clicked anywhere in the application — without navigating away from the current list view. One click. Inspector updates. No page transition. This is the flat hierarchy principle.

**Named layout presets** (saved workspace configurations):

| Preset | Default configuration |
|---|---|
| Default | Three-column, all panels visible, Tactical density |
| Draft Mode | Full-screen rookie board, comms pinned right, inspector collapsed |
| Gameday | Live scoring front, matchup center expanded, trade floor hidden |
| Trade Season | Inspector expanded, analyzer pinned, League Chat persistent |
| Admin | Commissioner panel dominant, calibration layer accessible |

Switching presets is one click. GMs with multiple leagues or contexts will use this daily.

---

## 6. Navigation Architecture

**Six primary nodes + persistent communication section:**

```
NAV RAIL
─────────────────────────────
[App: Legacy NFL ▾]   ← League switcher (see Section 10)
─────────────────────────────
[ti-layout-dashboard]  Home            ← League landing (default active)
[ti-crosshair]         War Room
[ti-building]          Franchise HQ
[ti-arrows-exchange]   Trade Floor
[ti-activity]          League Pulse
[ti-settings]          Control Room    ← Content role-gated (see Section 16)
─────────────────────────────
[ti-message-2]  League Chat   🔴       ← Comms toggle, not a nav node
[ti-mail]       Messages      🔴       ← Comms toggle, not a nav node
─────────────────────────────
[CC avatar]  Christopher  [role badge]  ← User identity + role indicator
```

**Module-to-nav mapping:**

| Nav Node | Module Spec Reference | Core purpose |
|---|---|---|
| Home | New (landing) | League pulse — what the league is doing right now |
| War Room | Module 1 (Asset Rankings) + Module 5 (Free Agency) | Engine output, valuations, free agent pool |
| Franchise HQ | Module 4 (Transactions) | Your team, your contracts, your direction |
| Trade Floor | Module 7 (Trade Analyzer) + Module 6 (Rookie Draft) | Market interaction, trades, draft |
| League Pulse | Module 2 (Power Rankings) + Module 3 (Matchups) | Competitive landscape |
| Control Room | Module 8 (Commissioner) + Pillar 4 (Admin) | System configuration, role-gated |

**Design rules:**
- GMs never sink more than two clicks to execute an action
- Filter state persists when moving between War Room sub-sections (e.g., position filter set in Asset Rankings stays active when moving to Free Agency)
- The active nav node is visually distinct (background highlight + icon color change)
- League Chat and Messages are comms toggles, not navigation. They open/close the communication panel. They carry unread count badges.

---

## 7. Theming System

### 7.1 Density Tiers

Three density tiers. Each maps directly to an ICP user tier.

| Tier | Density name | ICP user | Core behavior |
|---|---|---|---|
| 1 | Narrative | Social Casual | Clean player cards, color does the work, no raw tables, high-dopamine visual cues |
| 2 | Tactical | Alpha Competitor | Cards open to show volume metrics, Buy/Sell indicator tags, balanced color + numbers |
| 3 | Matrix | Dynasty Portfolio Whale | Full density, monospace, zero padding, keyboard navigation active |

**Density implementation:**
```css
/* Root attribute drives all density CSS variables */
:root[data-density="narrative"] { /* token set A */ }
:root[data-density="tactical"]  { /* token set B — default */ }
:root[data-density="matrix"]    { /* token set C */ }
```

**Density is set via the Quick Access Panel (see Section 9).** It is not a persistent UI element in the workspace header.

**Per-module density override:** Each module can override the global density default. A gear icon on the module header exposes a local density setting. The gear icon appears on hover of the module header area. The override persists to the user's profile. The Quick Access Panel also exposes module-specific overrides.

### 7.2 Color Themes

- **Phase 1:** Dark only. This is an enterprise power tool for expert users.
- **Phase 2:** Dark and Light.
- **Phase 3:** League-branded. Configurable primary and accent color per league instance. Build the token system from day one so Phase 3 is a configuration change, not a rebuild.

### 7.3 Color System Rules

**One color system. One meaning per color. Applied universally.**

Color coding must function as a standalone signal — not just a supplement to numbers. Tier 1 users read color, not numbers. If green means elite, that must be true in every panel, every card, every density mode, consistently.

| Signal | Color | Usage |
|---|---|---|
| Elite / positive / success | Green | Engine scores 88+, positive trends, cap-efficient contracts |
| Good / informational | Blue | Engine scores 75-87, neutral information, position badges (offense) |
| Warning / watch | Amber | Engine scores 60-74, hot contract tier, expiring contracts, active bid clocks |
| Danger / negative | Red | Engine scores below 60, bid timers under 1 hour, active DOT flags, injury alerts |
| Muted / inactive | Secondary gray | Historical data, inactive states, disabled elements |

**Position badge color convention:**
- Offense (QB, WR, RB, TE): Blue (info)
- Defense front seven (DE, DT, LB): Red (danger) / Cyan (teal) for LB
- Defense secondary (CB, S): Amber (warning)
- Special teams (K): Gray (secondary)

### 7.4 Density-Specific Behavioral Rules

**Narrative mode only:**
- Score update animations (animate in on data refresh)
- Win probability bar live updates
- Injury alert pulse animation
- High-dopamine visual cues

**Narrative mode restrictions:**
- No raw efficiency tables visible at any level
- No multi-column data grids in primary view
- Social layer (League Chat, League Feed) is prominent

**Matrix mode additions:**
- Keyboard navigation activates: J/K to move rows, Enter to open inspector, T to tag for trade block
- All animations disabled
- Social layer collapses to notification dots only — no chat previews
- Monospace font throughout

### 7.5 The Curiosity Trigger (Education Loop)

When a Tier 1 user in Narrative mode encounters a Volume Discrepancy alert on a player they own:
1. Tapping the alert expands that one player card to Tactical density
2. The rest of the list stays in Narrative
3. Tapping deeper (Layer 4 breakdown) switches the inspector to Matrix density for that player's detail
4. No mode announcement. No settings menu.
5. After three organic expansions to a higher density, the next session defaults that module to the higher density
6. The user was guided there. They did not choose to upgrade. The data was interesting.

---

## 8. Modular Workspace System

### 8.1 Edit Mode

Layout customization requires entering Edit Mode explicitly.

**Entering Edit Mode:** A persistent "Customize Layout" toggle accessible from the Quick Access Panel or a keyboard shortcut (to be confirmed during build).

**In Edit Mode:**
- Panels display drag handles in their header
- Panels display resize grips at their edges
- Panels display a remove button (sends them to a hidden panel library)
- A panel library drawer opens showing available panels not currently on screen

**Exiting Edit Mode:** Layout locks. Changes persist to the user's active preset. Accidental moves during active use are impossible.

**Widget library:** A collection of optional panels a GM can add to their workspace beyond defaults. Examples: bid clock tracker, cap space calculator, draft pick value chart, injury feed. Available in the panel library drawer during Edit Mode.

### 8.2 Named Presets

See Section 5 for preset definitions. Presets are created, named, and saved from within Edit Mode. The preset dropdown is accessible from the Quick Access Panel.

### 8.3 Panel-Level Settings

Each panel's settings gear (visible on hover) exposes:
- What sub-sections are visible within that panel and in what order
- Default sort column and direction
- Default filter state
- Local density override (for that module)
- Panel-specific options relevant to that module's content

---

## 9. Quick Access Panel

**What it is:** A lightweight settings popover accessible from anywhere in the application without entering the full Control Room. It is the home for commonly-changed preferences and layout controls.

**Trigger:** A settings or control icon in the workspace header bar (precise placement to be confirmed during build). Keyboard shortcut support recommended.

**Contents:**
- Global density control (Narrative / Tactical / Matrix three-state selector)
- Per-module density overrides (list of active modules with local density setting)
- Active layout preset selector
- Notification preferences
- "Enter Edit Mode" shortcut

**Design rules:**
- The Quick Access Panel is a popover — it overlays the current view, does not navigate away from it
- It closes on click-outside or Escape key
- It does not contain engine parameters, scoring weights, or any commissioner/admin functions
- Those live in Control Room

---

## 10. Landing Experience (Home Module)

### 10.1 Two-Level Structure

The application has two levels:

**App-level shell:** Persistent across all modules. Contains the nav rail with the league switcher at the top. The league switcher is a dropdown in the nav rail header showing all connected leagues. Selecting a league loads its landing view.

**League-level landing (Home):** The 10,000-foot view for the selected league. This is what the league is doing right now — not what the GM's personal team is doing (that's Franchise HQ).

### 10.2 League Switcher

Located at the top of the nav rail, above the navigation nodes. Shows the currently active league name with a chevron indicator. Clicking it opens a dropdown listing:
- All connected leagues with status dots (green = active/in-season, gray = offseason)
- Currently active league highlighted
- "Connect league" button at the bottom of the dropdown (triggers connection wizard)

For users with many connected leagues: dropdown is scrollable. No hard cap on connected leagues documented.

### 10.3 Connection Wizard

Triggered from the league switcher dropdown "Connect league" button.

**Wizard flow:**
1. Entry screen: text field for MFL League ID, brief explanation of what happens next
2. App attempts public ID lookup against MFL API
3. **If public:** Preview screen shows league name, team count, commissioner name, scoring format. "Connect this league" confirmation button.
4. **If private or auth required:** Auth prompt screen. GM enters MFL credentials. App fetches league data on behalf of authenticated user. Preview screen as above.
5. Confirmed: league appears in switcher dropdown. App loads that league's landing view.

**Error states:**
- Invalid league ID: clear inline error message, field remains editable
- Private league without credentials: friendly explanation, presents auth path
- Auth failure: clear error, retry option

**Phase 1 note:** Christopher connects the Legacy NFL once. The wizard is built and functional in Phase 1 even though he is the only user. Phase 2 brings 32 GMs through the same wizard.

### 10.4 League Landing Layout

The landing workspace is a **2x2 card grid**. Four cards, each independently refreshing on their data cycle.

**Always-visible cards (all four present at all times in default state):**

| Card | Content | Data source |
|---|---|---|
| League activity | Active bids with live clocks, DOT votes in progress, recent waiver awards, recent completed transactions | Transaction system, live |
| Seasonal (context-dependent) | Changes based on league calendar — see Section 11 | Calendar triggers |
| Trade block | Players listed for trade across all 32 teams. Position badge, player name, owning team, engine score. "View all" button. | Transaction system + engine |
| League chat | Last 3 messages with GM avatar, timestamp. Unread badge. "Open League Chat" button. | Comms layer |

**League activity card format per transaction type:**

```
[BID]  Player name · POS NFL team  |  $X.XM · clock: HH:MM
[DOT]  Team A ↔ Team B trade  |  X of 5 votes cast · Xhr remaining
[WVR]  Player name awarded · GM name  |  Dead cap: $X.XM applied
[EXT]  Player name extended · Team  |  New terms: $X.XM / X yr
[TAG]  Player name franchise tagged · Team  |  $X.XM
[CUT]  Player name released · Team  |  Dead cap: $X.XM
```

**Design rules for the landing:**
- Clean and to the point. This is a pulse view, not an analytics view.
- Card density defaults to Narrative regardless of global density setting. The landing is always approachable.
- GMs who want deep analytics navigate to War Room. The landing does not try to be both.
- Card grid layout adapts if screen width is narrow — can stack to 1x4 single column.

---

## 11. Seasonal Card System

The seasonal card occupies the second slot in the 2x2 landing grid. It is context-driven — it shows what is most relevant to the league right now. Only one seasonal card is active at a time. During true offseason gaps, the second slot shows league standings summary.

**Legacy NFL calendar and trigger map:**

| Seasonal card | Trigger on | Trigger off | Card content |
|---|---|---|---|
| Contract options window | March 1 | April 30 11:59PM EST | Countdown to deadline, players with pending options, extension activity |
| RFA tender deadline | May 1 | May 3 8AM EST | Players tendered, RFA rights summary, deadline countdown |
| UFA bidding — active | May 3 8AM EST | May 10 11:59PM EST | All active bids with live clocks, sorted by time remaining |
| Team re-signing window | May 10 | May 15 11:59PM EST | Players in re-signing negotiation, window countdown |
| Unrestricted free agency | May 19 8AM EST | Pool cleared or season locks | Remaining unsigned players, ranked by engine score |
| Rookie draft — live | July 14 8AM EST | Last pick confirmed | On the clock (team + pick number + countdown timer), last 5 picks with expandable full draft board |
| Undrafted rookie FA | Immediately after last draft pick | Commissioner closes window | Available undrafted rookies, engine projections |
| Cut day and buyouts | August 25 (countdown starts) | September 1 11:59PM EST | Players cut or bought out, dead cap impact, available talent |
| In-season (default) | NFL kickoff week | End of playoffs | Weekly matchup preview, record summary, playoff picture if weeks 12+ |

**Rookie draft card specifics** (most feature-rich seasonal card):
- On the clock: team name, pick number, live countdown timer
- Last 5 picks: pick number, team, player name, position badge — compact single-line rows
- "View full draft board" button: expands card in-place to show all completed picks and remaining order
- Collapse button when expanded

**Draft card timer behavior:** If the on-clock team has not picked within 2 hours of their clock starting, the card shows a visual warning state. Commissioner can advance the pick from the Commissioner Panel.

---

## 12. Module Reference

### 12.1 Home (League Landing)
See Sections 10 and 11.

### 12.2 War Room
**Purpose:** The engine's output. Where GMs go to understand value.

**Sub-sections:**
- Asset Board (Module 1): Global ranked list of all rostered players by Adjusted Score. Position-filtered views. Cap efficiency view. This is the default War Room view.
- Free Agency Intelligence (Module 5): Unowned player pool ranked by engine score. Per-position free agency rankings. Team need overlay. Active bid tracker with clock status.

**Default state:** Asset Board, Tactical density, all positions, sorted by Adjusted Score descending.

**Contextual Inspector content in War Room:** Selected player's engine score dominant display, Layer breakdown bars (RAS, Layer 4 scouting, Layer 3 age curve), contract details (salary, tier, efficiency multiplier), GM note with star rating, "Message [Team] GM" button.

**Discrepancy Hub (sub-feature of Asset Board):** Surfaces engine valuation vs market value (KTC) gaps. Top undervalued players, top overvalued assets. Accessible as a filter/view toggle within the Asset Board, not a separate nav destination.

### 12.3 Franchise HQ
**Purpose:** Your team. Your contracts. Your direction.

**Sub-sections:**
- Roster Command: Active roster with contract table, directional bias toggle, player heat map, watch list
- Transaction Center: Bid submission, waiver claim submission, contract extension submission, all formatted for manual MFL execution in Phase 1. In Phase 1, outputs are correctly formatted instruction sets.

**Directional bias toggle (Rebuild / Retool / Contender):**
- Lives in the Roster Command header
- Changes a `team_operational_bias` record in local SQLite
- Triggers re-calculation of `bias_adjusted_score` for all rostered players
- Adjusts which free agents are flagged as Watch or Target
- Does not affect the base Adjusted Score — that is immutable. Bias is a lens, not a modifier.

**Bias effect on rankings:**

| Bias | Age curve impact | Contract impact | FA targeting |
|---|---|---|---|
| Rebuild | Rewards younger players, penalizes age-decay assets | Rewards cold-tier contracts | Flag breakout candidates under 24 |
| Retool | Default Layer 3 decay | Default Layer 5 scaling | Flag high-value neutral tier targets |
| Contender | Flattens decay past peak for veterans | Tolerates hot-tier if base points elite | Flag veterans with declining contract length |

**Watch list:** Aggregated view of all player-tagged Watch/Target items from the notes system. Location within Franchise HQ to be confirmed during build — roadmapped.

### 12.4 Trade Floor
**Purpose:** Market interaction. Trades and rookie draft.

**Sub-sections:**
- Trade Analyzer (Module 7): Two-team asset comparison, cap impact for both teams, roster need alignment, historical comparable trades from league history, DOT-facing summary report.
- Draft Room (Module 6): Pre-draft prospect rankings with scouting layer applied, per-pick value at each slot, team need analysis, real-time draft board during live drafts, post-draft valuation.

**Critical rule:** The trade analyzer surfaces information. It does not make DOT decisions or veto recommendations. The NYG/WAS veto (Drake London for picks) was a human judgment about league competitive health. The analyzer supports DOT. It does not replace them.

### 12.5 League Pulse
**Purpose:** Competitive landscape.

**Sub-sections:**
- Power Rankings (Module 2): Weekly ladder with movement indicators, rationale line per team, trend visualization.
- Matchup Center (Module 3): Projected scores per active matchup, confidence range, key variance players, start/sit indicators.

### 12.6 Control Room
**Purpose:** System configuration. Role-gated content (see Section 16).

**Content by role:**

| Role | What they see in Control Room |
|---|---|
| GM | Personal settings: density defaults, layout presets, notification preferences |
| DOT member | Above + DOT vote history and personal veto record |
| Commissioner | Above + Commissioner Panel: transaction queue, DOT vote management, cap and roster alert flags, trade deadline status, clock override capability |
| Admin | Above + Admin Calibration: all engine parameters from `Engine_Specification.md`, S-curve values, Madden thresholds, scoring weight overrides, sub-signal weights, decay rates, Layer 5 cap tier percentages |

**Admin Calibration note:** This is Pillar 4 (Admin Console) from `North_Star.md`. It exposes every tunable parameter listed there. It does not expose code-locked structural mechanics. Refer to `North_Star.md` Section "The Four Pillars — Pillar 4" for the complete list of exposed vs code-locked parameters.

**Role simulation toggle (Admin only):** Christopher can switch his view to simulate what a standard GM sees. Used for testing and UI verification. Lives in Admin Calibration. Not a general feature.

---

## 13. Communication Layer

The communication layer is a **persistent shell element, not a nav node.** It is toggled via the comms icons at the bottom of the nav rail. It does not navigate you away from your current workspace.

### 13.1 Channel Structure

| Channel | Access | Purpose |
|---|---|---|
| League Feed | All GMs | Auto-generated activity stream. Every transaction event posts here automatically. Read-only (GMs react and comment, but events are system-generated). |
| League Chat | All GMs | Open discussion. All 32 GMs. Player card drops, trade discussions, commentary. |
| Commissioner Desk | Commissioner posts, all GMs read | Official rulings, schedule changes, veto decisions, announcements. Commissioner only posts here. |
| DOT Chamber | DOT members only | Private. Auto-threads when a trade is submitted for review. DOT votes, discusses, and resolves here. |
| Team Channels | Public to all GMs | Each of the 32 franchises has a public channel. GMs post trade block intent, team direction signals, recruiting messages. Navigation: dropdown in comms panel for browsing all 32 + click-through from any GM name, team name, or entity anywhere in the app. |
| Direct Messages | Sender and recipient only | GM to GM private. Trade negotiation lives here. |

### 13.2 Automatic League Feed Events

Every transaction in the system auto-posts a structured card to the League Feed. No manual action required from any GM.

| Trigger | Auto-posts to League Feed |
|---|---|
| UFA bid placed | Player name, bid amount ($X.XM), bidding team, clock status |
| Bid topped (snipe window) | Snipe alert, new leader, new amount, clock reset confirmation |
| Waiver claim awarded | Player name, awarding team, dead cap applied ($X.XM), releasing team |
| Trade submitted for DOT | Summary of assets exchanged, link to DOT Chamber thread |
| Trade approved by DOT | Full trade summary, engine value differential |
| Trade vetoed by DOT | Veto notice with commissioner ruling link |
| Player placed on trade block | Player name, owning team, engine score, link to Team Channel post |
| Rookie draft pick made | Pick number, team, player name, position, engine projection |
| Contract extension signed | Player name, team, new terms |
| Franchise tag applied | Player name, team, tag value |
| Player cut or bought out | Player name, releasing team, dead cap impact |
| DOT vote cast | Anonymous count update (X of 5 votes cast — individual votes not public) |

### 13.3 Communication Panel Layout (when expanded)

The communication panel, when expanded, shows:
- Channel selector at top (dropdown for Team Channels, direct list for fixed channels)
- Active channel content (message thread)
- Message input at bottom
- Unread indicators on channel names

The panel collapses to a 48px icon strip showing only the nav rail comms icons and their unread badge counts.

### 13.4 Notification Architecture

- Unread count badges on League Chat and Messages icons in the nav rail
- System toast notifications for high-priority events (bid topped during snipe window, trade offer received, DOT vote needed)
- Toast notifications are non-blocking and auto-dismiss
- Notification preferences are configurable in the Quick Access Panel

---

## 14. Notes System

Notes are a personal scouting tool. They live in the Contextual Inspector, not in the communication layer.

### 14.1 Private GM Notes

- Attached to every player record
- Visible only to the GM who wrote them
- Persist across sessions in local SQLite
- Components: star rating (1-5), free text field, Watch/Target tag
- Displayed in the Contextual Inspector below engine data and contract details

### 14.2 Published Notes

- Any private note can be pushed to the GM's Team Channel or to League Chat
- One-click publish button in the inspector
- Publishing converts the note from private journal to a public market signal
- Publishing does not remove the private note — it sends a copy to the chosen channel as a structured card
- Published note card shows: player name, position, engine score, the note text, and the posting GM's name

### 14.3 Watch List

An aggregated view pulling all players tagged Watch or Target across the GM's notes. Location within the application to be confirmed during build (roadmapped). Engine score displayed alongside each tagged player so the GM can see when valuation moves on watched players.

---

## 15. Trade-From-Chat Flow

Trades can be initiated from within the communication layer without leaving the current workspace. This is the primary trade negotiation surface. The Trade Analyzer (Trade Floor) handles formal pre-submission analysis. The chat flow handles real-time negotiation.

### 15.1 Initiation

Two entry points:
1. In any DM thread: `/offer` command or drag a player card from your roster into the message input
2. In League Chat: post a player card with a "Looking to move" tag — any GM can click it to open a trade builder with that player pre-selected

### 15.2 Multi-Asset Ledger

When the trade builder opens:
- Split panel: GM 1's assets on the left, GM 2's assets on the right
- Both full rosters visible and selectable
- Draft picks selectable by year (all future years available)
- Multi-player, multi-pick packages supported (the Legacy NFL explicitly allows complex multi-asset trades)
- No FAAB — the Legacy NFL uses bid points formula, not FAAB

### 15.3 Proposal and Negotiation Flow

1. GM 1 assembles package and submits proposal
2. Trade card appears in the DM thread between the two GMs with inline buttons: Accept / Counter / Decline
3. If Counter: trade builder opens pre-populated with the original offer. GM modifies and submits. New trade card appended to the DM thread below the original — maintains chronological negotiation history
4. Counter-offer loop repeats until accepted or declined
5. Expiration timer: embedded countdown in each trade card. When expired, the card grays out and action buttons deactivate. Offers do not auto-reject — they expire to an inactive state.

### 15.4 After Agreement (DOT Review)

When both GMs accept the trade:

1. **Rationale submission step:** A confirmation screen requires both GMs to submit their stated rationale for the trade. This is a rulebook requirement (Official_Rulebook.md). It is not optional and cannot be skipped. Both sides must submit before the trade advances.

2. **Auto-route to DOT:** Trade auto-posts to the DOT Chamber with a full structured thread containing: all assets exchanged, both GMs' rationale statements, engine value differential, cap impact for both teams.

3. **DOT voting:** DOT members vote within the DOT Chamber thread. 3 approves = trade executed. 3 vetoes = trade rejected. Result posts to League Feed.

4. **Trade deadline enforcement:** Any trade proposal submitted after the Week 9 deadline timestamp is rejected at submission with a clear error message. The deadline is a hard block, not a soft warning. No exceptions.

5. **DOT vote expiry:** If DOT does not reach 3 votes before the review period closes, the card enters a "vote expired" state. The specific resolution rule is to be confirmed in the transaction specification. UI renders a "vote expired" state card — rule enforcement is the transaction spec's job.

### 15.5 Trade Card States

Each trade card in the DM thread has distinct visual states:
- **Active:** Full color, action buttons enabled, countdown timer showing
- **Countered:** Dimmed, labeled "Countered — see below"
- **Accepted:** Green indicator, "Pending DOT review" label, link to DOT Chamber thread
- **Declined:** Red indicator, "Offer declined" label, buttons removed
- **Expired:** Gray, "Offer expired" label, buttons deactivated

---

## 16. Role Architecture

### 16.1 Four Roles — Additive Model

Roles are additive. Each role inherits everything below it.

| Role | Who | Added capabilities |
|---|---|---|
| GM | All league members | Home, War Room, Franchise HQ, Trade Floor, League Pulse. Personal settings in Control Room. All comms channels except DOT Chamber. |
| DOT member | Trade review body (subset of GMs) | DOT Chamber access. DOT vote history in Control Room. |
| Commissioner | League operator | Commissioner Panel in Control Room (transaction queue, clock override, cap flags, deadline management, roster compliance). Can post to Commissioner Desk. |
| Admin | Christopher | Admin Calibration in Control Room (engine parameters). Role simulation toggle. |

### 16.2 UI Role Indicators

**Nav rail user avatar:** Carries a small role badge. No badge = standard GM. Distinct visual indicator per role above GM. Hovering the avatar shows the role label clearly.

**Control Room:** On entering, a header line states the user's active role and what they have access to. No hidden surprises. A GM who opens Control Room sees exactly what a GM sees, clearly labeled.

### 16.3 Role Visibility Rules

**Control Room nav node:** Visible to all roles. Content inside it is role-gated.

**DOT Chamber:** Visible in the comms channel list for DOT members and above. Not visible to standard GMs — they cannot see the channel exists from their comms panel. League Feed posts reference DOT review in summary form accessible to all GMs.

**Commissioner Desk:** Visible to all GMs for reading. Only the Commissioner can post.

### 16.4 Multi-Role Handling

Roles stack. Christopher is GM + DOT + Commissioner + Admin. The UI applies the highest role's capabilities. No mode switch required. The role simulation toggle (Admin only) lets Christopher view the application as a standard GM for testing purposes.

### 16.5 Phase 2 Role Assignment

The Commissioner assigns roles via the Commissioner Panel in Control Room. Newly connected GMs default to standard GM role. Commissioner promotes to DOT member from the roster/member management section. Role changes take effect on the promoted GM's next session load. Role assignment UI is a Phase 2 build item.

---

## 17. ICP Alignment

The application serves three user tiers as defined in the ICP. Every UI decision must serve at least one tier without actively harming another.

| ICP Tier | Persona | Density default | Primary need | Anti-pattern to avoid |
|---|---|---|---|---|
| Tier 1 | Social Casual | Narrative | Feel informed without feeling overwhelmed | Exposing raw efficiency tables on primary view |
| Tier 2 | Alpha Competitor | Tactical | Competitive edge, volume metrics, buy/sell signals | Forcing a settings deep-dive to access intermediate data |
| Tier 3 | Dynasty Portfolio Whale | Matrix | Maximum data density, zero latency, multi-league scope | Any animation or decoration that consumes screen space |

**Phase 1 and Phase 2 serve Tier 3 primarily.** The Legacy NFL GMs are dynasty experts. The default density is Tactical (not Narrative). The architecture scales down to Tier 1/2 for Phase 3, not up from Tier 1 to Tier 3.

**Conversion mechanics are built into the UI, not bolted on later:**
- Tier 1 → Tier 2: Volume Discrepancy alert on a player card expands that one card to Tactical density. Curiosity is the trigger. Not a settings prompt.
- Tier 2 → Tier 3: Season-end automatically surfaces Dynasty Engine modules (draft simulator, long-term trade calculator). Year-round engagement replaces seasonal drop-off.

**Color coding serves Tier 1.** Tier 3 reads numbers. Both must work simultaneously. Color is never the only signal — it accompanies a number. But the color must be consistent and meaningful enough for a Tier 1 user to act on it alone.

---

## 18. Performance Requirements

These are acceptance criteria, not aspirational targets. A feature that does not meet them does not ship.

| Action | Maximum acceptable time |
|---|---|
| Density switch (CSS variable update) | < 16ms (one frame) |
| Color theme switch | < 16ms (one frame) |
| Panel collapse / expand | 120ms CSS transition, zero JS thread blocking |
| Panel drag in Edit Mode | 60fps continuous, no data re-fetch triggered |
| Player list scroll (1,500+ records) | Zero stutter — virtualized rendering only |
| Inspector population on row click | < 50ms |
| Trade card action (accept/counter/decline) | < 100ms UI response, async transaction processing |
| League Feed auto-update (new transaction event) | Appended without scroll position disruption |
| Density switch from Narrative to Matrix | CSS instant + progressive data reveal, no blocking spinner |

**Virtualization is mandatory for all lists.** Player lists, trade block lists, draft board, DOT vote lists, comms message threads — every scrollable list uses virtualized rendering. Only visible rows exist in the DOM at any time (~15-20 rows maximum).

**Theme changes must never trigger React re-renders.** If a profiling tool shows a component re-rendering due to a density or color theme change, that is a defect, not a performance issue to optimize. The architecture must prevent it by design.

---

## 19. Phase Boundaries

What is built in each phase affects architectural decisions made today. Every Phase 1 build must be architected to support Phase 2 and Phase 3 without structural rebuild.

**Phase 1 — Christopher's personal tool:**
- Single user, single league (Legacy NFL)
- Read-only against MFL (transactions output as formatted instruction sets)
- Desktop only (Windows, Mac, Linux)
- Dark theme only
- Admin has all roles by default — no role assignment system needed
- No onboarding wizard needed (league is pre-configured)
- No authenticated MFL write access

**Phase 2 — Legacy NFL league-wide:**
- 32 GMs with individual accounts
- Authenticated MFL access for each GM
- Role assignment system (Commissioner assigns via Control Room)
- Connection wizard (32 GMs onboard themselves)
- Multi-user comms live (all channels active)
- Desktop only still

**Phase 3 — Public multi-tenant:**
- Other leagues connect their own instances
- League-branded color themes (primary + accent configurable per league)
- Mobile harness / PWA (designed after desktop Beta is well-tested)
- Multi-window / multi-monitor (requires Wails v3 when it reaches stable GA)
- Light theme available
- Tier 1 and Tier 2 user onboarding flows active

**Architecture decisions made today that serve Phase 3:**
- CSS variable token system supports league-branded themes without code changes
- Connection wizard built in Phase 1 (Christopher uses it once) scales to 1,000+ leagues
- Role architecture designed for multi-league use from the start
- Density system serves Tier 1/2/3 — the Tier 1/2 flows are dormant in Phase 1, active in Phase 3

---

## 20. Deferred Items (Roadmap)

These were explicitly deferred during the architecture session. Do not design or build them in Phase 1 unless Christopher directs otherwise.

| Item | Deferred to | Notes |
|---|---|---|
| Watch list specific location in shell | Roadmap — Phase 2 design | Lives somewhere in Franchise HQ or inspector. Location TBD. |
| Empty state / first-run wizard | Phase 2 design | The screen new GMs see when no leagues are connected. Not needed in Phase 1 (Christopher has one pre-configured league). |
| DOT vote expiry resolution rule | Transaction specification | What happens if 5 DOT members do not reach 3 votes before the review window closes. Rule is in the Official_Rulebook.md. UI just renders a "vote expired" state. |
| Player table column design | Separate UI design session | Which columns show in which density, column widths, sortability, multi-column filter. Needs dedicated design pass. |
| Inspector panel feature specification | Separate UI design session | Christopher confirmed it needs to be "very feature rich when the time comes." Current implementation is a starting point only. |
| Mobile harness / PWA | Phase 3 (after Beta) | After the desktop app is well-tested in Beta. Wails v2 is single-window desktop only. Mobile is a separate build. |
| Multi-monitor / detached panels | Phase 3 | Requires Wails v3 multi-window architecture. Wails v3 is alpha as of June 2026. Not buildable safely until v3 reaches stable GA. |
| League-branded color themes | Phase 3 | CSS token system supports it. Configurable per league. Build the token architecture now, activate the configuration UI in Phase 3. |
| Phase 2 role assignment UX | Phase 2 design | Commissioner assigns GM roles via Commissioner Panel. The panel shell is built in Phase 1. The role assignment interaction within it is Phase 2. |
| In-season weekly scoring integration | Separate build session | Live scoring cards, matchup update frequency, player score refresh during games. Needs dedicated data pipeline session. |
| Multi-team trade builder | Phase 2 | The Legacy NFL rulebook may allow multi-team trades. The two-team builder is Phase 1. Multi-party allocation panel (as described by Sleeper research) is Phase 2. |

---

## 21. Locked Decisions Registry

A complete record of every decision made in the UI architecture session. These are not open for debate in build sessions. If a locked decision creates a technical problem, flag it to Christopher.

| Decision | What was decided |
|---|---|
| Framework | Wails v2 (stable GA) — v3 is alpha, not suitable |
| Backend language | Go natively — no Python, no Python sidecar, no multi-runtime |
| Frontend | React + Tailwind CSS |
| Client state | Zustand (not Redux, not Context for theme state) |
| Theme state | CSS Custom Properties only — never React state |
| List rendering | Virtualized — all scrollable lists, no exceptions |
| Architecture pattern | Clean / Hexagonal — core engine isolated from infrastructure |
| Layout customization | Edit Mode (Option A) — explicit toggle, locked layout during active use |
| Desktop-first | Phase 1 and Phase 2 are desktop only. Mobile is Phase 3. |
| Multi-monitor | Out of scope Phase 1 and Phase 2. Requires Wails v3. Phase 3. |
| Phase 1 MFL access | Read-only. Transaction outputs are formatted instruction sets for manual execution. |
| Density tiers | Three tiers: Narrative / Tactical / Matrix. Map to ICP Tier 1 / 2 / 3. |
| Density implementation | data-density attribute on root, CSS variables cascade. Never JS state. |
| Density default | Tactical — Legacy NFL GMs are Tier 3 operators. Narrative is for Phase 3 public users. |
| Density and data | Density drives API payload scope. Narrative = minimal. Matrix = full. |
| Density and display | Density never hides data from the pipeline. CSS controls visibility only. |
| Color system | One system, one meaning per color, consistent across all panels and density modes. |
| Phase 1 color theme | Dark only. |
| Quick Access Panel | Lightweight settings popover for density, presets, notifications. Not a nav destination. |
| Per-module density | Module gear icon (Option A) and Quick Access Panel. Both available. |
| Navigation structure | Six nodes: Home, War Room, Franchise HQ, Trade Floor, League Pulse, Control Room. |
| Communication layer | Persistent shell element — not a nav node. Toggled via comms icons in nav rail. |
| League Feed | All transaction events auto-post. System-generated. No manual action needed. |
| Team channel navigation | Option C — dropdown for browsing all 32 + click-through from any GM/team reference in app. |
| Trade from chat | /offer command in DM or player card drop. Multi-asset ledger. Accept/Counter/Decline inline. |
| Trade DOT routing | After both GMs agree + rationale submitted, auto-routes to DOT Chamber. Never auto-executes. |
| Trade deadline | Week 9 hard block. Proposals rejected at submission after deadline timestamp. No exceptions. |
| Role model | Four roles: GM, DOT, Commissioner, Admin. Additive. Higher role inherits lower. |
| Control Room | Visible to all roles. Content inside is role-gated. |
| DOT Chamber | Comms-layer privilege, not a nav node. Invisible to standard GMs in their comms panel. |
| Role indicators | Badge on user avatar in nav rail. Control Room header states active role on entry. |
| Role simulation | Admin-only toggle in Admin Calibration. Tests GM view without logging out. |
| Seasonal cards | Trigger map locked per Legacy NFL calendar. Single seasonal card slot in 2x2 landing grid. |
| Landing layout | 2x2 card grid. League-level view, not personal team view. Defaults to Narrative density. |
| League switcher | In nav rail header. Dropdown. Shows all connected leagues. "Connect league" at bottom. |
| Connection wizard | Handles both public and private MFL leagues. Attempt public lookup first, auth prompt if needed. |
| Scoring engine language | Go only. Engine_Specification.md math implemented in Go within core/engine/. |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
*UI Architecture Session: June 2026*
*Next session: Claude Code build — application shell and navigation scaffold*

| Version | Date | Changes |
|---|---|---|
| 1.0 | June 2026 | Initial release. Full UI architecture session output. Stack, shell, navigation, theming, modular workspace, landing, communication layer, notes, trade-from-chat, role architecture, ICP alignment, performance requirements, phase boundaries, and locked decisions registry. |
