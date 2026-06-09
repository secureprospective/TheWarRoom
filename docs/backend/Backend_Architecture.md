# Legacy NFL — Backend Architecture Document
**Version:** 1.0 — June 2026
**Status:** Locked. Load this document at the start of every backend build session.
**Authority:** This document governs all backend, infrastructure, and data layer decisions. Where this document and another conflict on a backend matter, this document wins.
**Prepared by:** Christopher Campbell + Claude (Anthropic)
**Session type:** Backend Architecture Brainstorm

---

## 1. How Claude Code Uses This Document

Read every section before writing a single file. Every decision was deliberated and locked. Nothing here is a guess.

Do not reopen locked decisions. If a locked decision creates a technical constraint that feels wrong, flag it to Christopher — do not route around it silently.

Do not add infrastructure not in this document without Christopher's explicit direction.

**API testing agenda (requires MFL league ID — cannot proceed without it):**
- OQ-003: Long play bonus data format in playerScores endpoint
- OQ-005: Salary adjustment line item — which endpoint, what shape
- OQ-011: True Position split exposure in league endpoint
- MFL injury status field presence in players endpoint

These four questions require live API testing against the Legacy NFL league. All other decisions are resolved.

This document works alongside:
- `North_Star.md` — strategic purpose and four pillars
- `Engine_Specification.md` — scoring engine math and layer architecture
- `UI_Direction_Document.md` — frontend constraints the backend must serve
- `Official_Rulebook.md` — authoritative source for all transaction rules
- `MFL_API_Specification.md` — API structure and known risks

---

## 2. Hosting Model

### Phase 1 — Local Only

Go server runs in a Docker container on Christopher's Proxmox homelab. SQLite is local. Nothing is internet-facing. Christopher is the only user.

### Phase 2 — Cloudflare Tunnel

Homelab server exposed to 32 GMs via Cloudflare Tunnel. No inbound firewall ports opened. Zero Trust Access handles authentication (free tier, covers 50 users). TLS terminated at Cloudflare edge. WebSocket proxying supported through the tunnel. Cost: $0 beyond the domain registration (~$12/year).

### Phase 3 — VPS Migration

Go server binary migrates to a VPS (Hetzner or Fly.io). Cloudflare Tunnel config updated to point at VPS instead of homelab. SQLite migrates to PostgreSQL at this boundary — the database abstraction layer makes this a driver swap, not a schema rewrite. Zero Trust transitions to paid tier or standalone auth provider (Clerk, free to 10k users) as user count grows beyond 50.

### Phase 1 VM Specification (Proxmox)

```
OS:    Debian 12 LTS (minimal)
vCPU:  2
RAM:   4096 MB
Disk:  20 GB SSD-backed
Net:   VirtIO, bridge to existing LAN
```

**Docker Services**
- `legacy-nfl-server` — Go binary (restart: always)
- `cloudflared` — Cloudflare Tunnel daemon (restart: always)

SQLite database file on mounted volume. Automated daily backup via cron to Cloudflare R2.

---

## 3. Technology Stack

| Component | Technology | Notes |
|---|---|---|
| Backend language | Go 1.21+ | Scoring engine, API, WebSocket server, transaction system — all one binary |
| Local database | SQLite (WAL mode) | Offline-first, single file, zero configuration |
| Desktop framework | Wails v2 (stable GA) | Locked in UI Direction Document |
| Tunnel | Cloudflare Tunnel (cloudflared) | Phase 2 exposure |
| Authentication | Cloudflare Zero Trust | Phase 2, free tier |
| Object storage | Cloudflare R2 | End-of-season archive |
| Pick values | FantasyCalc API | api.fantasycalc.com/values/current |

**Architecture pattern:** Clean / Hexagonal. `core/` knows nothing about SQLite, MFL, or the frontend. Dependencies flow inward. Infrastructure adapts to the core — never the reverse.

---

## 4. Project Directory Structure

```
legacy-nfl-app/
├── main.go
├── wails.json
├── app.go
├── core/
│   ├── engine/
│   │   ├── engine.go              orchestrator — runs all 6 layers
│   │   ├── types.go               PlayerRecord, EngineOutput, shared types
│   │   ├── layer1/
│   │   │   └── hygiene.go         salary floor enforcement, imputation
│   │   ├── layer2/
│   │   │   └── scoring.go         rulebook scoring matrix, True Position split
│   │   ├── layer3/
│   │   │   └── decay.go           age decay curve
│   │   ├── layer4/
│   │   │   ├── layer4.go          film × RAS × breakout orchestrator
│   │   │   ├── film/
│   │   │   │   ├── provider.go    FilmSignalProvider interface
│   │   │   │   ├── aggregator.go  Approach A aggregation
│   │   │   │   ├── madden.go      Approach D regulation
│   │   │   │   └── ema.go         EMA blend logic
│   │   │   ├── ras/
│   │   │   │   └── ras.go         RAS normalization + position weight tier
│   │   │   └── breakout/
│   │   │       └── breakout.go    four mechanical inputs, Approach A
│   │   ├── layer5/
│   │   │   └── capscaling.go      percentage-of-cap tier multiplier
│   │   ├── layer6/
│   │   │   └── tiebreaker.go      sort key builder — not a score modifier
│   │   └── math/
│   │       └── scurve.go          Shape B sigmoid, shared across components
│   ├── models/                    immutable data structures
│   ├── rubrics/
│   │   ├── rubric.go              PositionRubric struct definition
│   │   └── loader.go              reads from SQLite, builds rubric structs
│   └── transactions/
│       ├── service.go             TransactionService — all transaction types
│       ├── validators/
│       │   ├── deadline.go
│       │   ├── cap.go
│       │   ├── roster.go
│       │   ├── contract.go
│       │   ├── increment.go
│       │   ├── snipe.go
│       │   ├── picks.go
│       │   ├── rfa.go
│       │   └── deadcap.go
│       ├── calculator/
│       │   ├── deadcap.go
│       │   ├── bidpoints.go
│       │   ├── extension.go
│       │   ├── tag.go
│       │   └── buyout.go
│       └── output/
│           └── formatter.go       Phase 1 formatted MFL execution instructions
├── infrastructure/
│   ├── database/
│   │   ├── db.go                  SQLite driver, WAL mode config
│   │   ├── migrations/            sequential numbered migration files
│   │   └── queries/               per-domain query files
│   ├── mfl/
│   │   ├── client.go              rate-limited HTTP client, 429 handling
│   │   ├── scheduler.go           tiered polling jobs
│   │   ├── endpoints/             one file per endpoint type
│   │   ├── normalizer.go          raw MFL → engine-ready structs
│   │   └── crossref.go            MFL ID → external source ID mapping
│   ├── csv_ingestion/
│   │   ├── parser.go              CSV → FilmSignal struct mapping
│   │   ├── fuzzy_match.go         composite key matching + confidence scoring
│   │   └── providers/
│   │       ├── pff_csv.go         PFFCSVProvider
│   │       └── ras_csv.go         RASCSVProvider
│   ├── fantasycalc/
│   │   └── client.go              pick value API ingestion
│   ├── config/
│   │   └── loader.go              rubric params from SQLite → PositionRubric
│   └── comms/
│       ├── websocket.go           WebSocket server, session registry
│       ├── broadcaster.go         fan-out to all active sessions
│       └── snapshot.go            in-memory league state snapshot per league
├── interface/
│   └── bindings/                  Wails-exposed functions for frontend
└── frontend/
    └── ...                        per UI Direction Document
```

---

## 5. Go Scoring Engine Architecture

### Core Types

```go
type DataState int

const (
    DataPresent DataState = iota  // value exists and is sourced
    DataAbsent                    // confirmed not applicable
    DataUnknown                   // not yet collected, may exist
)

type PlayerID   string  // 4-5 digit string, leading zeros enforced
type FranchiseID string // 4-digit string, "0001"–"0032", "0000" = commissioner ops

func NewPlayerID(raw string) PlayerID       // "531" → "0531"
func NewFranchiseID(raw string) FranchiseID // "1" → "0001"

type FilmSignal struct {
    Name         string
    Value        float64    // normalized 0.0–1.0
    State        DataState
    IsStatic     bool       // static = set once; dynamic = EMA blended
    IsSubjective bool       // true = Madden regulation applies
    Source       string     // "RSP" | "PFF" | "TDN" — audit trail
    ProviderSlot string     // which FilmSignalProvider interface slot
}

type PlayerRecord struct {
    PlayerID        PlayerID
    Position        Position
    Age             float64
    IsVeteran       bool
    YearsExp        int
    Season          int
    ScoringConfigID string

    Stats           SeasonStats
    FilmSignals     []FilmSignal
    PreviousEMA     map[string]float64  // sub-signal name → last EMA value

    RAS             float64
    RASState        DataState

    BreakoutAge         float64;  BreakoutAgeState   DataState
    SchoolTier          SchoolTier; SchoolTierState   DataState
    CollegeUsageRate    float64;  CollegeUsageState  DataState

    Salary          float64   // annual, millions
    ContractYear    int
}
```

### Six-Layer Pipeline

Each layer is a pure function. No I/O. No database calls. Layers chain sequentially.

```
PlayerRecord + ScoringConfig + PositionRubric
    ↓
Layer 1: hygiene.Apply()
    → salary floor enforcement, missing RAS imputation (position-group mean, fallback 5.00)
    → Unknown flags set for missing data
    → returns cleaned PlayerRecord

Layer 2: scoring.Apply()
    → rulebook scoring matrix applied to SeasonStats
    → True Position split applied from ScoringConfig (MFL-sourced)
    → TFL scored as direct stat at 2.5 pts — no proxy
    → returns BasePoints (float64)

Layer 3: decay.Apply()
    → (1 - decay_rate) ^ max(0, age - peak_limit)
    → both decay_rate and peak_limit admin-tunable per position
    → returns AgePull (float64)

Layer 4: layer4.Apply()
    → film.Apply()    → film_effective
    → ras.Apply()     → RAS_effective
    → breakout.Apply() → breakout_effective
    → Layer4Output = film_effective × RAS_effective × breakout_effective
    → no overall Layer 4 cap — component caps are the natural bounds
    → returns Layer4Output struct

    ScoutingAdjusted = BasePoints × AgePull × Layer4Output.Combined

Layer 5: capscaling.Apply()
    → salary as % of league cap → Cold / Neutral / Hot
    → Cold < 1.2%, Neutral 1.2%–4.8%, Hot > 4.8% (admin-tunable percentages)
    → returns CapMultiplier (float64)

    AdjustedScore = ScoutingAdjusted × CapMultiplier

Layer 6: tiebreaker.Build()
    → NOT a score modifier
    → builds TiebreakerKey: {IsVeteran, RAS, ScarcityRank}
    → used only in ranking sort when AdjustedScores are equal
```

### Layer 4 Internals

**S-curve (Shape B sigmoid):**
```
math/scurve.go
    input float64       normalized composite input
    cap   float64       asymptote — admin-tunable per component per position
    k     float64       steepness — admin-tunable
    x0    float64       inflection point — admin-tunable
    → returns float64 in [1.0 - cap, 1.0 + cap]
```
One function. All three components call it with their own rubric parameters.

**EMA — stateless in the engine:**
```
new_value = (1 - α) × PreviousEMA[signalName] + α × current_observation
First observation: new_value = current_observation (initial value rule).
EMA previous values live in ema_state SQLite table, carried in PlayerRecord.PreviousEMA.
Season transition default: CONTINUATION. Prior season's final EMA value is the starting point.
RESET override: per sub-signal flag in position rubric.
```

**Madden regulation (Approach D) — subjective signals only:**
```
film/madden.go
    expertValue   float64    normalized expert claim (0.0–1.0)
    maddenNorm    float64    Madden attribute normalized within position group
    threshold     float64    admin-tunable
    blendScale    float64    admin-tunable
    weight        float64    sub-signal rubric weight
    → returns regulated weight
```
Analytical sub-signals (`IsSubjective = false`) bypass this function entirely.

**Confidence scoring (internal only — never surfaced in UI):**
```
film_confidence     = Σ(Present signal weights) / Σ(all expected signal weights)
RAS_confidence      = 1.0 if Present, 0.0 if Absent
breakout_confidence = Σ(Present field weights) / Σ(expected field weights)
```
Confidence scales each component's deviation from 1.00 per SL-013. Unknown signals contribute 0. Engine degrades gracefully — it does not fail.

**Film signal provider interface:**
```go
type FilmSignalProvider interface {
    GetSignal(playerID PlayerID, slotName string) (float64, DataState, error)
}
```
Implementations: `RSPProvider` (manual entry), `PFFCSVProvider` (CSV upload), `ManualEntryProvider` (fallback). Engine never references a specific source. Swapping the source behind the interface is a configuration change — no engine code changes.

### Batch Processing

```go
func (e *Engine) ScoreAll(players []PlayerRecord, config ScoringConfig) []EngineOutput
```
Worker pool: `runtime.NumCPU()` goroutines. Each player is independent — no shared state.

Full 32-team rescore (1,500+ players) completes in seconds on the homelab VM.

### Scoring Run Trigger — Hybrid

- **Scheduled baseline:** daily rescore of all players
- **Event-triggered partial rescore:** fires immediately on high-impact events (transaction, roster change, scoring config change) for affected players only
- **Affected players scope:** players directly involved in the event (changing teams, being released). Layer 5 is percentage-of-league-cap — a trade does not change the league cap and does not require roster-wide rescoring for either team.
- **Concurrency:** if a new trigger fires while a rescore is in progress, it queues. Executes after current run completes. No cancellation.

### Error Handling in Batch

Failed player record: skip and flag. Player excluded from that run's output. Flagged in admin console for review. Batch continues. One bad record never blocks 1,499 others.

### Rubric Loading

Rubrics loaded from SQLite at engine startup and passed as parameters. Admin console writes updated values to SQLite. Next scoring run reads fresh rubrics — no app restart required. All admin-tunable parameters per SL-017 are exposed through this mechanism.

---

## 6. MFL API Integration Architecture

### Design Principle — League-Agnostic Ingestion

No Legacy NFL-specific logic in the ingestion code. All league rules, scoring values, position structures, and contract mechanics come from the league endpoint config or admin console. The ingestion layer is a data pipeline, not a rules engine. Legacy NFL is the test case — its edge-case ruleset forces contact with endpoints simpler leagues never touch.

### Rate-Limited Client

All MFL API calls go through one HTTP client wrapper.
- On 429: exponential backoff (1s → 2s → 4s → ... → 60s max)
- Never retry immediately
- Log backoff events; surface persistent backoff in admin console
- All calls server-side — browser cross-domain calls blocked by MFL

### PlayerID and FranchiseID Enforcement

All ID padding logic lives in `NewPlayerID()` and `NewFranchiseID()` constructors. Enforced at the ingestion boundary. Nothing downstream handles padding. All SQLite columns are TEXT type for both ID types.

### Tiered Polling Scheduler

| Tier | Trigger | Endpoints |
|---|---|---|
| Startup | App launch | league, players, rosters, contracts, salaries, nflSchedule |
| Daily | 24h TTL | players, rosters, contracts, salaries |
| Event-triggered | On transaction event | rosters, contracts, transactions, tradeBait |
| Weekly | After final game results | standings, playerScores |
| Real-time window | Active game hours only (from nflSchedule) | liveScoring |
| Season transition | Commissioner trigger | league, draftResults |

Startup is sequential through dependency tiers. Tier 0 (players) and Tier 1 (league) must complete before engine runs. Failure in either blocks startup with a clear error.

### Endpoint Dependency Order

```
Tier 0: players          → master player ID list
Tier 1: league           → scoring config, all league rules
Tier 2: rosters, contracts, salaries    → depends on Tier 0 + 1
Tier 3: standings, transactions, tradeBait, nflSchedule
Tier 4: playerScores, liveScoring       → in-season only
```

### EDGE Position Mapping — Option C

Unresolved EDGE players flagged in admin console with Unknown position. Layer 2 score does not calculate until resolved. Admin console surfaces "Unresolved EDGE positions" queue. Christopher resolves per player — small set, changes rarely.

### Cross-Reference Table

```
player_id_crossref
├── mfl_id        TEXT  (primary, PlayerID)
├── nfl_id        TEXT
├── pff_id        TEXT
├── ras_id        TEXT  (fuzzy-matched — see RAS section)
├── madden_id     TEXT
└── last_verified DATETIME
```

### MFL Unavailability on Startup

Use last cached config. Surface staleness warning in admin console ("Scoring config last verified X days ago"). Do not block scoring runs. Retry on next connection attempt.

### MFL Host Server Discovery

The `wwwXX` host server for league-specific calls is fetched from the league endpoint on startup and cached. Re-verified on each season transition and periodically (weekly). If the cached host fails, fall back to `api.myfantasyleague.com` (load-balanced) for the re-discovery call.

### MFL Credential Storage

User credentials for private league authentication stored via OS keychain using Go `keyring` package. Never stored in SQLite.

### API Testing Agenda (First Build Session)

Once MFL league ID is supplied, one dedicated session tests:
1. OQ-003: playerScores long play bonus format
2. OQ-005: Salary adjustment line item endpoint and shape
3. OQ-011: league endpoint True Position split exposure
4. Injury status field presence in players endpoint

Normalizer and scoring parser for affected components finalized after this session.

---

## 7. External Data Source Integration

### PFF — CSV Upload (Primary Analytical Signal)

PFF has no consumer API. Data enters via manual CSV export from PFF+ Premium Stats.

Admin console data upload interface: File picker per data type (grades, YPRR, TPRR). CSV is parsed, players mapped via cross-reference table, written to `film_signals` table.

```
infrastructure/csv_ingestion/
├── parser.go          CSV → FilmSignal struct mapping
├── fuzzy_match.go     composite key matching + confidence scoring (for RAS)
└── providers/
    └── pff_csv.go     PFFCSVProvider implementing FilmSignalProvider
```

Upload frequency: weekly during season (grades update weekly), annually for breakout/usage metrics. Staleness indicator in admin console per data type ("PFF grades last updated [date]").

**Film signal upload tracking:**
```
film_signal_uploads
├── id            TEXT PRIMARY KEY
├── league_id     TEXT
├── source        TEXT        ← 'PFF' | 'RSP_manual' | 'RAS_csv' | etc.
├── data_type     TEXT        ← 'grades' | 'yprr' | 'tprr' | 'ras' | etc.
├── season        INTEGER
├── week          INTEGER     ← NULL for annual uploads
├── record_count  INTEGER
├── uploaded_at   DATETIME
└── uploaded_by   TEXT
```
`film_signals` table includes `upload_id` field. Engine reads most recent upload per player per signal per source.

### YPRR and TPRR

Sourced from PFF CSV exports (same upload path as grades, different data type selector). Fantasy Points Data Suite approved as Tier 3 supplement — same CSV upload mechanism. When both sources present, engine uses most recently uploaded. Admin console shows last upload date per source.

### RAS — Kaggle/nflverse CSV, Fuzzy Matching

ras.football has no API. Community CSV snapshots via nflverse ecosystem or Kaggle are the ingestion path. Christopher downloads and uploads via admin console CSV interface. Updates once per combine/pro day season (February–April active window).

Fuzzy matching required: RAS data has no standard player IDs. Cross-reference uses composite key:
```
player_name + draft_year + college + drafted_team
```
Matching is high-risk on name alone (suffixes, spelling variances). Fuzzy matching component scores confidence (0.0–1.0). Low-confidence matches surfaced in admin console "Unresolved RAS matches" queue for manual confirmation.

```
player_id_crossref
└── ras_match_confidence  REAL    ← flags below threshold for admin review
```

### Madden — Kaggle CSV

EA ratings API is in a TOS gray zone. Kaggle dataset ("Madden 26 Week 15 Player Ratings") provides full attribute-level granularity — the engine attribute fields required for Approach D regulation are present. One upload per Madden release cycle via admin console CSV interface. Same upload path as PFF.

Phase 2+ revisits EA API or commercial data path once application serves multiple users.

Attribute coverage confirmed for Layer 4: physical/athletic, mental/cognitive, offensive skill, defensive skill — all present at attribute level, not just OVR.

### FantasyCalc — Automated API

```
GET https://api.fantasycalc.com/values/current?format=1  (1QB format for Legacy NFL)
```
Daily poll during offseason, more frequent during draft season. Returns pick values in JSON. No scraping, no TOS exposure. Community CSV dumps from KTC available as fallback via admin console upload if FantasyCalc is unavailable.

### Injury Status — MFL First

RotoWire commercial feed is enterprise-only, not viable for Phase 1. Verification order:
1. MFL players endpoint — verify injury status field presence (API testing agenda)
2. If absent: nflverse injury data (free, structured)
3. Commercial feed (RotoWire B2B, SportsDataIO) — Phase 2/3 only

### Snap Counts

FantasyPros snap counts (approved Tier 5). Fetched weekly during season.

### Depth Charts

Ourlads (approved Tier 5). Fetched multiple times weekly.

### Age and Experience

NFL.com player profiles (approved Tier 1). Seasonal update. NFL.com career games played is the authoritative source for experience — not MFL's experience field.

---

## 8. SQLite Schema

WAL mode enabled on initialization. All player and franchise ID columns are TEXT type — never INTEGER.

### Schema Migration Strategy

Sequential numbered migration files in `infrastructure/database/migrations/`. Applied automatically on app startup via embedded migration runner. New columns, new tables, and index additions are non-destructive migrations. Phase 2 additions (new columns, foreign keys to new tables) do not require dropping existing tables.

### Domain 1 — League and Configuration

```sql
CREATE TABLE leagues (
    id                TEXT PRIMARY KEY,
    mfl_league_id     TEXT NOT NULL,
    name              TEXT NOT NULL,
    season            INTEGER NOT NULL,
    connected_at      DATETIME,
    last_synced_at    DATETIME
);

CREATE TABLE scoring_configs (
    id                TEXT PRIMARY KEY,
    league_id         TEXT REFERENCES leagues,
    season            INTEGER,
    source            TEXT,           -- 'mfl' | 'admin_override'
    config_json       TEXT,           -- full MFL league endpoint response
    created_at        DATETIME,
    is_active         BOOLEAN
);

CREATE TABLE transaction_configs (
    id                        TEXT PRIMARY KEY,
    league_id                 TEXT REFERENCES leagues,
    dot_expiry_mode           TEXT,   -- 'none' | 'void' | 'commissioner_override' | 'auto_approve' | 'auto_veto'
    dot_review_hours          INTEGER, -- NULL = no limit
    trade_deadline_week       INTEGER,
    snipe_window_hours        REAL,
    snipe_increment           REAL,
    min_bid_increment         REAL,
    year_multipliers_json     TEXT,   -- {1: 1.00, 2: 1.20, 3: 1.40, 4: 1.60}
    max_contract_years        INTEGER,
    buyout_rates_json         TEXT,   -- {2: 0.60, 3: 0.75, 4: 0.90}
    dead_cap_rate             REAL,
    restructure_dead_cap_rate REAL,
    transaction_permissions_json TEXT, -- per phase/flag: which types are legal
    created_at                DATETIME
);
```

`scoring_configs` is DECISION-010 in practice. Every engine output references a `scoring_config_id`. When config changes, a new version is written. Historical outputs retain their original config ID.

`transaction_configs` holds all transaction mechanics not present in the MFL league endpoint. Bid point year multipliers, snipe rules, dead cap rates — all league-configurable, not hardcoded.

### Domain 2 — Users and Franchises

```sql
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    display_name  TEXT,
    role          TEXT,       -- 'gm' | 'dot' | 'commissioner' | 'admin'
    created_at    DATETIME
);
-- Phase 1: one row. Christopher, role = 'admin'.
-- Phase 2: 32 rows. Commissioner promotes roles.

CREATE TABLE franchises (
    id               TEXT PRIMARY KEY,
    league_id        TEXT REFERENCES leagues,
    mfl_franchise_id TEXT NOT NULL,   -- "0001"–"0032", FranchiseID type
    name             TEXT,
    owner_user_id    TEXT             -- NULL until Phase 2 GM connects
);
-- Phase 1: all 32 rows exist, all owner_user_id → Christopher's user ID.
```

### Domain 3 — Players

```sql
CREATE TABLE players (
    mfl_id            TEXT PRIMARY KEY,   -- PlayerID type, leading zeros enforced
    name              TEXT NOT NULL,
    position          TEXT,               -- resolved; NULL if EDGE unresolved
    position_raw      TEXT,               -- raw MFL tag (may be 'EDGE')
    position_flagged  BOOLEAN DEFAULT 0,  -- Option C EDGE flag
    nfl_team          TEXT,
    age               REAL,
    is_veteran        BOOLEAN,
    years_exp         INTEGER,
    draft_year        INTEGER,
    last_updated      DATETIME
);

CREATE TABLE player_id_crossref (
    mfl_id               TEXT REFERENCES players,
    nfl_id               TEXT,
    pff_id               TEXT,
    ras_id               TEXT,
    madden_id            TEXT,
    ras_match_confidence REAL,    -- fuzzy match score; below threshold → admin review
    last_verified        DATETIME
);

CREATE INDEX idx_players_position ON players(position);
CREATE INDEX idx_players_nfl_team ON players(nfl_team);
```

### Domain 4 — Rosters and Contracts

```sql
CREATE TABLE rosters (
    id             TEXT PRIMARY KEY,
    league_id      TEXT REFERENCES leagues,
    mfl_id         TEXT REFERENCES players,
    franchise_id   TEXT REFERENCES franchises,
    roster_status  TEXT,    -- 'active' | 'ir' | 'practice_squad'
    season         INTEGER,
    as_of          DATETIME
);

CREATE TABLE contracts (
    id                  TEXT PRIMARY KEY,
    league_id           TEXT REFERENCES leagues,
    mfl_id              TEXT REFERENCES players,
    franchise_id        TEXT REFERENCES franchises,
    annual_salary       REAL,
    adjusted_salary     REAL,    -- after adjustment items (OQ-005 dependent)
    contract_years      INTEGER, -- years remaining
    expiration_year     INTEGER,
    contract_status     TEXT,    -- 'ufa' | 'rfa' | 'rookie_slot'
    is_restructured     BOOLEAN DEFAULT 0,
    is_tagged           BOOLEAN DEFAULT 0,
    season              INTEGER,
    last_updated        DATETIME
);

CREATE TABLE waiver_order (
    league_id      TEXT REFERENCES leagues,
    season         INTEGER,
    franchise_id   TEXT REFERENCES franchises,
    position       INTEGER,    -- 1 = first priority
    updated_at     DATETIME,
    PRIMARY KEY (league_id, season, franchise_id)
);
-- Season start: position = inverse final standings rank.
-- After each successful claim: claiming team → last position, others move up one.

CREATE INDEX idx_contracts_league_season ON contracts(league_id, season);
CREATE INDEX idx_rosters_league_franchise ON rosters(league_id, franchise_id);
```

### Domain 5 — Engine Outputs

```sql
CREATE TABLE engine_outputs (
    id                    TEXT PRIMARY KEY,
    league_id             TEXT REFERENCES leagues,
    mfl_id                TEXT REFERENCES players,
    season                INTEGER,
    scoring_config_id     TEXT REFERENCES scoring_configs,  -- DECISION-010
    run_at                DATETIME,

    base_points           REAL,
    age_pull              REAL,
    film_multiplier       REAL,
    ras_multiplier        REAL,
    breakout_multiplier   REAL,
    layer4_output         REAL,
    scouting_adjusted     REAL,
    cap_tier              TEXT,   -- 'cold' | 'neutral' | 'hot'
    cap_tier_basis        REAL,   -- % of cap at calculation time
    adjusted_score        REAL,

    film_confidence       REAL,   -- internal, not UI-surfaced
    ras_confidence        REAL,
    breakout_confidence   REAL,

    tiebreaker_json       TEXT    -- {is_veteran, ras, scarcity_rank}
);

CREATE TABLE ema_state (
    league_id      TEXT REFERENCES leagues,
    mfl_id         TEXT REFERENCES players,
    signal_name    TEXT,
    current_value  REAL,
    last_updated   DATETIME,
    PRIMARY KEY (league_id, mfl_id, signal_name)
);

CREATE TABLE matchup_projections (
    id                TEXT PRIMARY KEY,
    league_id         TEXT REFERENCES leagues,
    mfl_id            TEXT REFERENCES players,
    week              INTEGER,
    season            INTEGER,
    projected_points  REAL,
    floor             REAL,
    ceiling           REAL,
    xfp               REAL,    -- Expected Fantasy Points — Tactical tier source
    opponent_rank     INTEGER,
    calculated_at     DATETIME,
    scoring_config_id TEXT REFERENCES scoring_configs
);

CREATE INDEX idx_engine_outputs_league_season_pos
    ON engine_outputs(league_id, season);
CREATE INDEX idx_engine_outputs_adjusted_score
    ON engine_outputs(adjusted_score DESC);
CREATE INDEX idx_matchup_projections_league_week
    ON matchup_projections(league_id, season, week);
```

xFP is an engine-derived calculation produced by Module 3 (Matchup Score Predictions), not an external ingestion. Tactical tier reads from `matchup_projections.xfp`. If Module 3 has not yet run for the current week, xFP shows Unknown in the Tactical payload.

### Domain 6 — Film Signal Entry

```sql
CREATE TABLE film_signal_uploads (
    id            TEXT PRIMARY KEY,
    league_id     TEXT REFERENCES leagues,
    source        TEXT,        -- 'PFF' | 'RSP_manual' | 'RAS_csv' | 'Madden_csv'
    data_type     TEXT,        -- 'grades' | 'yprr' | 'tprr' | 'ras' | 'attributes'
    season        INTEGER,
    week          INTEGER,     -- NULL for annual uploads
    record_count  INTEGER,
    uploaded_at   DATETIME,
    uploaded_by   TEXT
);

CREATE TABLE film_signals (
    id             TEXT PRIMARY KEY,
    league_id      TEXT REFERENCES leagues,
    mfl_id         TEXT REFERENCES players,
    signal_name    TEXT,        -- matches rubric slot name
    value          REAL,        -- normalized 0.0–1.0
    data_state     TEXT,        -- 'present' | 'absent' | 'unknown'
    source         TEXT,        -- 'RSP' | 'PFF' | 'TDN' | etc.
    provider_slot  TEXT,        -- FilmSignalProvider interface slot
    upload_id      TEXT REFERENCES film_signal_uploads,
    entered_by     TEXT,
    season         INTEGER,
    created_at     DATETIME
);

CREATE INDEX idx_film_signals_player_season
    ON film_signals(league_id, mfl_id, season);
```

Engine reads most recent upload per player per signal per source.

### Domain 7 — Transactions

```sql
CREATE TABLE transactions (
    id               TEXT PRIMARY KEY,
    league_id        TEXT REFERENCES leagues,
    type             TEXT,   -- 'ufa_bid' | 'rfa_offer' | 'waiver' | 'trade' |
                             --  'extension' | 'tag' | 'buyout' | 'restructure'
    status           TEXT,   -- 'active' | 'won' | 'expired' | 'voided' | 'pending_dot'
                             --  | 'approved' | 'vetoed'
    submitted_at     DATETIME,
    resolved_at      DATETIME,
    deadline_at      DATETIME,        -- 24-hour bid clock expiry
    snipe_window_at  DATETIME,        -- 20-hour mark (deadline_at - 4h)
    match_decision_at DATETIME,       -- RFA match window expiry (rights-holder 24h)
    payload_json     TEXT,            -- full transaction detail per type
    created_by       TEXT             -- franchise_id
);

CREATE TABLE transaction_players (
    transaction_id   TEXT REFERENCES transactions,
    mfl_id           TEXT REFERENCES players,
    franchise_id     TEXT REFERENCES franchises,
    side             TEXT   -- 'offering' | 'receiving' | 'releasing'
);

CREATE TABLE dot_votes (
    id              TEXT PRIMARY KEY,
    transaction_id  TEXT REFERENCES transactions,
    voter_id        TEXT,
    vote            TEXT,   -- 'approve' | 'veto'
    voted_at        DATETIME,
    UNIQUE(transaction_id, voter_id)
);

CREATE TABLE dead_cap_ledger (
    id              TEXT PRIMARY KEY,
    league_id       TEXT REFERENCES leagues,
    franchise_id    TEXT REFERENCES franchises,
    mfl_id          TEXT REFERENCES players,
    transaction_id  TEXT REFERENCES transactions,
    amount          REAL,
    applied_at      DATETIME,
    expires_at      DATETIME
);

CREATE INDEX idx_transactions_status_deadline
    ON transactions(league_id, status, deadline_at);
CREATE INDEX idx_transactions_type_status
    ON transactions(league_id, type, status);
```

Trade proposals: no independent expiration clock. Active until explicitly declined, countered, or accepted — or the Week 9 hard deadline passes. The "Expired" card state triggers when the trade deadline passes on an unresolved proposal. The UI countdown timer shows time remaining to the trade deadline, not offer age.

Bid clock: server-authoritative. `deadline_at = NOW + 24h` on each valid bid. `snipe_window_at = deadline_at - 4h`. Clock does not tick in DB — server calculates `deadline_at - NOW` per request.

Simultaneous bids: SQLite serializes writes. Second bid validates against the state committed by the first. If second bid fails (e.g., no longer meets snipe increment against the new leader), it is rejected with a clear error requiring the GM to resubmit manually. No auto-adjustment on GM's behalf.

### Domain 8 — Communication

```sql
CREATE TABLE channels (
    id            TEXT PRIMARY KEY,
    league_id     TEXT REFERENCES leagues,
    type          TEXT,   -- 'league_feed' | 'league_chat' | 'commissioner_desk'
                          -- | 'dot_chamber' | 'team_channel' | 'direct_message'
    name          TEXT,
    franchise_id  TEXT,   -- for team_channel and direct_message types
    access_level  TEXT    -- 'all' | 'dot' | 'commissioner' | 'pair'
);

CREATE TABLE messages (
    id             TEXT PRIMARY KEY,
    channel_id     TEXT REFERENCES channels,
    thread_id      TEXT,              -- NULL = top-level; references messages.id
    sender_id      TEXT,              -- user_id or 'system'
    message_type   TEXT,   -- 'text' | 'transaction_event' | 'trade_card' |
                           --  'bid_clock' | 'dot_vote' | 'ruling'
    payload_json   TEXT,
    created_at     DATETIME,
    is_archived    BOOLEAN DEFAULT 0
);

CREATE TABLE notification_preferences (
    user_id        TEXT,
    channel_id     TEXT REFERENCES channels,
    event_type     TEXT,
    delivery_mode  TEXT,   -- 'push' | 'silent' | 'queue'
    PRIMARY KEY (user_id, channel_id, event_type)
);

CREATE TABLE notification_queue (
    id          TEXT PRIMARY KEY,
    user_id     TEXT,
    message_id  TEXT REFERENCES messages,
    delivered   BOOLEAN DEFAULT 0,
    created_at  DATETIME
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,   -- session token
    user_id     TEXT,
    connected   BOOLEAN DEFAULT 0,
    last_seen   DATETIME
);

CREATE INDEX idx_messages_channel_created
    ON messages(channel_id, created_at DESC);
CREATE INDEX idx_notification_queue_user_delivered
    ON notification_queue(user_id, delivered);
```

### Domain 9 — Seasonal State and Calendar

```sql
CREATE TABLE league_state (
    league_id    TEXT REFERENCES leagues,
    season       INTEGER,
    phase        TEXT,         -- primary phase (drives seasonal card)
    flags_json   TEXT,         -- active flags array
    phase_since  DATETIME,
    updated_at   DATETIME,
    PRIMARY KEY (league_id, season)
);

CREATE TABLE league_calendar (
    id              TEXT PRIMARY KEY,
    league_id       TEXT REFERENCES leagues,
    season          INTEGER,
    event_type      TEXT,   -- 'phase_transition' | 'flag_set' | 'flag_clear'
                            -- | 'contract_year_rollover' | 'season_open'
    target_phase    TEXT,
    target_flag     TEXT,
    trigger_type    TEXT,   -- 'datetime' | 'event'
    trigger_at      DATETIME,
    event_trigger   TEXT,   -- 'last_draft_pick' | 'commissioner_close'
                            -- | 'nfl_kickoff' | 'contract_year_rollover'
    fired_at        DATETIME,
    is_active       BOOLEAN DEFAULT 1
);
```

**Phase enum:**
```
OFFSEASON_GAP, CONTRACT_OPTIONS, RFA_TENDER, UFA_BIDDING,
TEAM_RESIGNING, UNRESTRICTED_FA, ROOKIE_DRAFT,
UNDRAFTED_ROOKIE_FA, CUT_DAY, IN_SEASON
```

**Flag enum:**
```
UFA_WINDOW_ACTIVE, TRADE_WINDOW_OPEN, TRADE_WINDOW_CLOSED,
PLAYOFFS_ACTIVE, DRAFT_ACTIVE, UNDRAFTED_FA_ACTIVE
```

**Contract year rollover:** Commissioner-triggered event. Commissioner fires from Commissioner Panel. When fired: all contracts in the league decrement `contract_years` by 1. Players reaching 0 years transition to UFA or RFA status per contract type.

**Season year boundary:** Commissioner triggers "Open New Season" from Commissioner Panel (confirmed against MFL season year in Phase 1 and 2; commissioner-only in Phase 3). `season` integer increments in `leagues` table. EMA CONTINUATION applies. New scoring config fetched from MFL.

### Domain 10 — Draft

```sql
CREATE TABLE draft_picks (
    id              TEXT PRIMARY KEY,
    league_id       TEXT REFERENCES leagues,
    season          INTEGER,
    round           INTEGER,
    pick_number     INTEGER,
    original_owner  TEXT REFERENCES franchises,
    slot_price      REAL,           -- fixed rookie slot price from rulebook
    mfl_id          TEXT,           -- NULL until pick is made
    picked_at       DATETIME,
    is_compensatory BOOLEAN DEFAULT 0  -- compensatory picks cannot be traded
);

CREATE TABLE draft_pick_ownership (
    pick_id        TEXT REFERENCES draft_picks,
    franchise_id   TEXT REFERENCES franchises,
    acquired_via   TEXT,   -- 'original' | 'trade'
    acquired_at    DATETIME
);

CREATE TABLE pick_value_historical (
    league_id       TEXT REFERENCES leagues,
    season          INTEGER,
    round           INTEGER,
    pick_number     INTEGER,
    avg_adj_score   REAL,
    sample_size     INTEGER,
    calculated_at   DATETIME,
    PRIMARY KEY (league_id, season, round, pick_number)
);
-- Phase 3 wireframe. Populates passively as seasons accumulate.
-- Activates in trade analyzer when sample_size >= 3 for sufficient pick slots.
-- Phase 1/2: FantasyCalc API is the pick value source.
```

### Domain 11 — Rubrics and Admin Config

```sql
CREATE TABLE rubric_params (
    id            TEXT PRIMARY KEY,
    league_id     TEXT REFERENCES leagues,
    position      TEXT,
    component     TEXT,        -- 'film' | 'ras' | 'breakout'
    param_key     TEXT,        -- 'scurve_cap' | 'ema_alpha' | 'weight' | etc.
    signal_name   TEXT,        -- NULL for component-level params
    value         REAL,
    updated_at    DATETIME,
    updated_by    TEXT
);

CREATE TABLE user_preferences (
    user_id             TEXT PRIMARY KEY,
    density_default     TEXT,
    layout_json         TEXT,
    module_density_json TEXT,
    updated_at          DATETIME
);
```

### Domain 12 — Supplemental Tables

```sql
CREATE TABLE standings (
    id             TEXT PRIMARY KEY,
    league_id      TEXT REFERENCES leagues,
    franchise_id   TEXT REFERENCES franchises,
    season         INTEGER,
    week           INTEGER,
    wins           INTEGER,
    losses         INTEGER,
    ties           INTEGER,
    points_for     REAL,
    points_against REAL,
    all_play_wins  INTEGER,
    all_play_losses INTEGER,
    last_updated   DATETIME
);

CREATE TABLE nfl_schedule (
    id             TEXT PRIMARY KEY,
    season         INTEGER,
    week           INTEGER,
    game_id        TEXT,
    home_team      TEXT,
    away_team      TEXT,
    kickoff_at     DATETIME,
    is_complete    BOOLEAN DEFAULT 0
);

CREATE TABLE watch_list (
    id             TEXT PRIMARY KEY,
    user_id        TEXT,
    league_id      TEXT REFERENCES leagues,
    mfl_id         TEXT REFERENCES players,
    tag            TEXT,   -- 'watch' | 'target'
    note           TEXT,
    created_at     DATETIME
);

CREATE INDEX idx_standings_league_season ON standings(league_id, season);
CREATE INDEX idx_nfl_schedule_week ON nfl_schedule(season, week);
```

---

## 9. Transaction System Backend

### Validation Pipeline

Every transaction type passes through a composable validation chain. Chain stops at first failure. Clear typed error returned to submitting GM.

```
1. SeasonalStateValidator   is this transaction type permitted in current league phase?
2. DeadlineValidator        week 9 hard block for trades (checked first, before all else)
3. CapSpaceValidator        does the team have cap space?
4. RosterLimitValidator     does the team have roster space?
5. ContractRuleValidator    do terms meet rulebook minimums and position floors?
6. IncrementValidator       is bid increment ≥ 0.1 points?
7. SnipeValidator           if past 20h mark, is snipe increment ≥ 1.0 points?
8. PickEligibilityValidator picks within 2-year trading window? Not compensatory?
9. RFATagValidator          is 'RFA' explicitly included in bid?
10. DeadCapValidator        does releasing team have cap room for dead cap hit?
```

### Bid Clock Architecture

All bid processing is atomic: read current state → validate → write → commit. Single SQLite transaction.

```go
func (ts *TransactionService) SubmitBid(bid BidRequest) (BidResult, error) {
    return ts.db.WithTransaction(func(tx *sql.Tx) error {
        current := tx.GetActiveBid(bid.PlayerID)

        if bid.Points < current.Points+0.001 {
            return ErrIncrementTooSmall
        }
        if time.Now().After(current.SnipeWindowAt) {
            if bid.Points < current.Points+1.0 {
                return ErrSnipeIncrementRequired
            }
        }
        // cap, roster, contract validators...

        newDeadline    := time.Now().Add(24 * time.Hour)
        newSnipeWindow := newDeadline.Add(-4 * time.Hour)
        tx.WriteBid(bid, newDeadline, newSnipeWindow)
        return nil
    })
    // broadcast outside the transaction — commit first, then notify
}
```

SQLite's serialized write model handles simultaneous bids correctly. Second bid always validates against state committed by first. No race condition. If the second bid fails, it is rejected with a clear error requiring manual resubmission. No auto-adjustment.

### Clock Monitor Goroutine

```
goroutine: clock_monitor
    adaptive tick:
        30 seconds — default
        5 seconds  — when UFA_WINDOW_ACTIVE flag is set

    on each tick:
        SELECT from transactions WHERE status = 'active' AND deadline_at <= NOW
        for each expired:
            mark status = 'won'
            trigger resolution
            broadcast result to League Feed
            generate Phase 1 formatted output
```

### DOT Voting State Machine

```
submitted → pending_dot (both GMs accepted + both rationale submitted)
              ↓ votes accumulate
approved  (3 approves)  → post to League Feed, generate MFL instruction
vetoed    (3 vetoes)    → post to League Feed
expired   (review window closed, < 3 in either direction) → per dot_expiry_mode in transaction_configs
```

DOT Chamber auto-thread created when trade enters `pending_dot`. Thread contains: all assets exchanged, both rationale statements, engine value differential, cap impact for both teams.

**Rationale gate:** `payload_json` tracks `{gm1_rationale_submitted: bool, gm2_rationale_submitted: bool}`. Both must be true before trade routes to DOT. If one GM submits and the other does not within 24 hours, the trade remains in a "rationale pending" sub-state with a notification reminder.

### Dead Cap Calculator

```go
func CalculateDeadCap(contract Contract) float64 {
    rate := deadCapRate  // from transaction_configs (default 0.35)
    if contract.IsRestructured {
        rate = restructureDeadCapRate  // default 0.50
    }
    return rate * contract.AnnualSalary * float64(contract.YearsRemaining)
}
```

Calculated at waiver initiation. Written to `dead_cap_ledger`. Locked at release — not recalculated after the fact.

### Franchise Tag Calculator

```go
func CalculateTagPrice(leagueID, position string, priorSalary float64) float64 {
    // Top 5 salaries at position league-wide from contracts table
    // Excludes playoff FA contract designations
    top5Avg := db.AvgTopNSalariesAtPosition(leagueID, position, 5)
    tag := top5Avg
    if tag < priorSalary {
        tag = priorSalary * 1.20
    }
    return tag
}
```

Calculated at moment of application from current salary data.

### Phase 1 Output Format

`output/formatter.go` generates human-readable MFL execution instruction sets per transaction type. Phase 2 replaces with authenticated MFL API write calls. The formatter output becomes the source of truth for the write call's content.

---

## 10. Communication Backend Architecture

### Design Principles

- First-class platform, not a chat feature on a transaction app
- Replaces Proboards entirely
- All transaction events auto-post as structured typed message objects
- Commissioner tasks automated within rule frameworks
- User notification preferences: fully configurable per channel, per event type
- Multi-league: one WebSocket connection aggregates all league notifications

### WebSocket Server

Embedded in the main Go server binary. No separate service. Phase 3 extraction if load justifies it (Phase 2 load does not justify separation).

### Message Types

| type | Generator | Renders as |
|---|---|---|
| text | Any GM | Standard message |
| transaction_event | System only | Structured event card |
| trade_card | System on GM initiation | Interactive Accept/Counter/Decline |
| bid_clock | System | Live countdown card |
| dot_vote | System | Vote card (buttons for DOT, count-only for GMs) |
| ruling | Commissioner only | Official ruling card |
| system_announcement | System or Commissioner | League-wide notice |

### Event Flow

```
Transaction engine fires event
    → in-process event bus
    → message service writes typed message to correct channel
    → broadcasts to all active sessions with channel access
    → notification system evaluates each user's preferences
    → pushes to relevant sessions or writes to notification_queue
```

### Channel Access Enforcement

Role checked at the service layer — not just UI. DOT Chamber query from a standard GM returns `ErrUnauthorized` before any message data is touched.

```go
func (s *ChannelService) GetMessages(userID, channelID string) ([]Message, error) {
    user    := db.GetUser(userID)
    channel := db.GetChannel(channelID)
    if !user.Role.CanAccess(channel.AccessLevel) {
        return nil, ErrUnauthorized
    }
    return db.GetMessages(channelID)
}
```

### Multi-League Session Model

One WebSocket connection per user aggregates all league notifications. League switcher changes active display context — does not change or close the connection. Channel IDs are scoped by `league_id`. Clean separation between leagues in data; single connection in transport.

### Multiple Simultaneous Sessions

Session registry: `user_id → [session_id, ...]`. Broadcasts fan out to all active sessions for eligible users. Read state tracks against `user_id` — switching devices does not reset read position.

### Reconnect Delivery

On WebSocket reconnect, server sends one current state snapshot: active bid leaders, clock times, active DOT vote counts. Not a replay of missed events. The server maintains an in-memory `league_state_snapshot` struct per league (rebuilt from SQLite on startup, updated on each state change). Missed bid history is in channel message records — loads when GM opens the relevant channel.

### Notification Preferences

Three delivery modes: `push` (WebSocket event → UI toast + badge), `silent` (written to channel, no alert), `queue` (held for offline delivery). Configurable per user, per channel, per event type. Admin console does not surface in standard GM preferences — this is a user-level setting.

### Message History

Full history retained in SQLite. End-of-season archival to Cloudflare R2 triggered by the Commissioner "Close Season" action. Current season in active SQLite. Prior seasons in R2 cold storage. Nothing is lost.

### DOT Chamber Threading

Auto-thread per trade. Each trade submission creates a new thread in the DOT Chamber. Each trade is its own isolated deliberation. Easy to archive and reference.

---

## 11. Tiered Payload System

### Three Explicit Query Paths

```
GET /api/players/narrative    minimal fields, fast query, few joins
GET /api/players/tactical     adds snap_share, target_share, xFP, trend_direction, breakout_flag
GET /api/players/matrix       full engine output, all Layer 4 sub-signals, full contract detail, RAS
```

Three separate query paths. Not one parameterized endpoint. Separate query complexity, separate cache TTLs, separate invalidation triggers.

### Pre-Fetch Strategy

On player list load:
1. Narrative tier fetched immediately → list renders
2. Tactical tier fetched async → merges into local state silently
3. Matrix tier fetched for viewport + 50-row scroll buffer → merges into local state

Matrix data expands as GM scrolls. Inspector population reads from local state — no network request if data already fetched. Inspector never blocks.

### Payload Structs

```go
type NarrativePlayer struct {
    PlayerID      PlayerID
    Name          string
    Position      string
    NFLTeam       string
    AdjustedScore float64
    ContractTier  string    // 'cold' | 'neutral' | 'hot'
    InjuryStatus  string
}

type TacticalPlayer struct {
    NarrativePlayer
    SnapShare        float64
    TargetShare      float64
    ExpectedPoints   float64   // xFP — from matchup_projections
    TrendDirection   string    // 'up' | 'flat' | 'down'
    BreakoutFlag     bool
}

type MatrixPlayer struct {
    TacticalPlayer
    YPRR               float64
    TPRR               float64
    EPA                float64
    RAS                float64
    FilmMultiplier     float64
    RASMultiplier      float64
    BreakoutMultiplier float64
    Layer4Output       float64
    ContractDetail     ContractDetail
    // all Layer 4 sub-signals as named fields
}
```

Embedding ensures Narrative is always a valid subset of Tactical, which is always a valid subset of Matrix. Data enriches progressively — never overwrites.

### Derived Field Definitions

| Field | Calculation |
|---|---|
| TrendDirection | Compare current EMA value to 4-week-prior EMA; ≥ +5% = 'up', ≤ -5% = 'down', otherwise 'flat' |
| BreakoutFlag | breakout_raw at upper zone boundary per position rubric |
| xFP | From `matchup_projections.xfp` — Module 3 output; Unknown if Module 3 not yet run for current week |

### Cache TTLs and Invalidation

| Tier | TTL | Invalidation trigger |
|---|---|---|
| Narrative | 5 minutes | Injury status update, active bid clock change |
| Tactical | 1 hour | Weekly snap count update, scoring run |
| Matrix | On scoring run | Engine rescore completes |

### Performance Requirements (from UI Direction Document)

- Inspector population on row click: < 50ms (local state lookup)
- Player list scroll (1,500+ records): zero stutter (virtualized rendering)
- Density switch: CSS instant + progressive data reveal, no blocking spinner
- Trade card action: < 100ms UI response, async transaction processing

---

## 12. Seasonal State Machine

### State Model — Composite

Primary phase drives the seasonal card. Active flags drive transaction validation. Both can change independently.

**Primary phases:**
```
OFFSEASON_GAP, CONTRACT_OPTIONS, RFA_TENDER, UFA_BIDDING,
TEAM_RESIGNING, UNRESTRICTED_FA, ROOKIE_DRAFT,
UNDRAFTED_ROOKIE_FA, CUT_DAY, IN_SEASON
```

**Active flags:**
```
UFA_WINDOW_ACTIVE     → triggers 5s clock monitor polling
TRADE_WINDOW_OPEN     → trades legal
TRADE_WINDOW_CLOSED   → trades blocked (post-Week 9)
PLAYOFFS_ACTIVE       → lifts $12M 1-year bid cap
DRAFT_ACTIVE          → draft board live, pick timer running
UNDRAFTED_FA_ACTIVE   → undrafted FA pool open
```

### State Machine Goroutine

- Default tick: 60 seconds
- Active window tick: 30 seconds (when any active window flag is set)
- Datetime triggers: goroutine handles
- Event triggers (`last_draft_pick`, `commissioner_close`, `nfl_kickoff`, `contract_year_rollover`): fired by the action that causes them

**On State Transition:**
1. Write new phase/flags to `league_state`
2. Broadcast `seasonal_state_change` via WebSocket → frontend updates seasonal card
3. Signal clock monitor (5s if `UFA_WINDOW_ACTIVE`, else 30s)
4. If crossing season boundary: fetch new scoring config from MFL, write new `scoring_config` version, trigger EMA season transition (CONTINUATION default)

### Calendar Seeding

Hybrid: Legacy NFL dates ship as defaults from the rulebook. Commissioner confirms or adjusts each season via admin console. Other leagues configure their own dates. All calendar events are rows in `league_calendar` — no hardcoded dates in code.

### Transaction Permission Matrix

Stored in `transaction_configs.transaction_permissions_json`. Per phase + flag combination: which transaction types are legal. League-configurable.

### Draft Pick Value Model — Phase 1/2

FantasyCalc API (`api.fantasycalc.com/values/current?format=1` for 1QB). Daily poll. JSON response mapped to pick slot values. `pick_value_historical` table accumulates passively as Legacy NFL seasons complete. Phase 3 migration: when `sample_size >= 3` for sufficient pick slots, admin console offers option to switch trade analyzer from FantasyCalc to internal historical model.

---

## 13. Offline Capability

**Works offline (reads from SQLite cache):**
- All player valuations and engine outputs
- Roster and contract views (last cached state)
- Transaction history
- Communication history
- Engine rescore against cached data
- Admin console — all rubric params and config
- Seasonal state — last known phase

**Does not work offline:**
- Live scoring during games
- Fresh roster/contract sync from MFL
- Transaction confirmation against MFL
- Weekly player scores not yet cached

**Staleness handling:**
Single banner in admin console: "MFL sync unavailable — data as of [timestamp]." Per-endpoint last-sync times visible in admin console. No scattered per-panel warnings.

**Offline transaction generation (Phase 1):**
Allowed with staleness warning displayed. Not blocked. Christopher is aware of cache age and makes the judgment call.

**Sync on reconnect:**
Tiered startup sequence runs exactly as on app launch. Cache refreshes in dependency order. Engine rescores if new data changes outputs. Banner clears when sync completes.

---

## 14. Role-Based Data Access

### Four Roles — Additive

| Role | Who | Access |
|---|---|---|
| GM | All league members | All modules, all channels except DOT Chamber |
| DOT | Trade review body | + DOT Chamber access |
| Commissioner | League operator | + Commissioner Panel, Commissioner Desk posts |
| Admin | Christopher | + Admin Calibration, role simulation toggle |

### Backend Enforcement

Role checked at service layer before any data is touched. UI hiding is redundant — backend is the enforcement.

Every service method pattern:
```go
if !user.Role.CanAccess(requiredLevel) {
    return nil, ErrUnauthorized
}
```

DOT Chamber query from standard GM: `ErrUnauthorized` at the query level, before message data is touched.

### Phase 1

Christopher is GM + DOT + Commissioner + Admin. One user, one row in `users` table, `role = 'admin'`. All permissions active by default.

### Phase 2

Commissioner assigns roles via Commissioner Panel. Newly connected GMs default to GM role. Role takes effect on promoted GM's next session load. Role assignment UI is a Phase 2 build item — schema and enforcement logic are Phase 1.

---

## 15. Historical Score Preservation (DECISION-010)

Every engine output record carries a `scoring_config_id` referencing the exact config version that produced it. When scoring rules change (league votes, MFL config update), a new `scoring_config` record is written. Old output records retain their original config ID.

Prior season scores are never retroactively changed. Year-over-year comparisons reflect actual rules in effect for each season.

Re-calculation of historical seasons requires an explicit admin override flag (`admin_recalculate = true`). For data corrections, not for scoring rule changes.

When admin console parameters change (rubric weights, S-curve params): marks current season outputs as stale, queues rescore for current season only. Historical seasons unaffected.

---

## 16. Phase 1 → Phase 2 Migration

Phase 2 is additive. Nothing locked in Phase 1 is reopened or rebuilt.

| Item | Phase 1 | Phase 2 | Migration |
|---|---|---|---|
| Users | 1 row (Christopher, admin) | 32 rows | Commissioner uses role assignment UI to onboard GMs. Schema unchanged. |
| Franchise ownership | All 32 → Christopher | One owner per franchise | `owner_user_id` populated per franchise. Schema unchanged. |
| Communication | Local WebSocket (loopback) | Cloudflare Tunnel exposed | Config change. Zero code change. |
| MFL access | Read-only, formatted output | Authenticated write calls | `output/formatter.go` logic moved into MFL write client. Interface unchanged. |
| Network | Local only | Cloudflare Tunnel + Zero Trust | Infrastructure config only. |
| Role assignment | Admin has all roles | Commissioner assigns per franchise | Phase 2 UI build. Schema and enforcement already in Phase 1. |
| Hardware | Current homelab | Upgraded RAM (4GB target) | Hardware only. |
| Pick values | FantasyCalc API | FantasyCalc API + internal model building | No migration needed — both paths exist from Phase 1. |

**What does not change at Phase 2:**
- Schema
- Scoring engine
- Transaction validators (read league config, not user count)
- Seasonal state machine
- Communication layer interface
- FantasyCalc, PFF CSV, Madden CSV, RAS CSV ingestion paths
- Cloudflare DNS and proxy configuration (extended, not replaced)

The connection wizard Christopher uses once in Phase 1 is the same wizard 32 GMs use in Phase 2.

---

## 17. Approved Data Sources (Backend Scope)

Sources added or confirmed during this session. Full list remains in `Approved_Sources.md`.

| Source | Tier | Access method | Data types | Notes |
|---|---|---|---|---|
| FantasyCalc | Tier 4 | Public API — automated | Draft pick values | Replaces KTC as primary. 1QB format param for Legacy NFL. |
| Fantasy Points Data Suite | Tier 3 | CSV export — manual upload | YPRR, TPRR | Supplement to PFF for route efficiency metrics. |
| PFF | Tier 2 | CSV export — manual upload | Grades, YPRR, TPRR | No consumer API. PFF+ Premium Stats CSV download. |
| RAS (ras.football) | Tier 2 | Kaggle/nflverse CSV — manual upload | RAS scores | No API. Fuzzy name matching required for cross-reference. |
| Madden | Tier 2 | Kaggle CSV — manual upload | Attribute-level ratings | Full attribute coverage confirmed. One upload per release cycle. |
| MFL | Tier 1 | API — automated | All roster/contract/scoring | Primary data source. All configurable values sourced per DECISION-009. |

---

## 18. Open Questions — API Testing Required

These cannot be resolved without the MFL league ID and a live testing session.

| ID | Question | Blocks |
|---|---|---|
| OQ-003 | Does playerScores return long play bonus events as discrete stat fields or embedded in totals? | Layer 2 bonus parsing in normalizer |
| OQ-005 | Which endpoint exposes salary adjustment line items and what is the data shape? | Cap calculation accuracy |
| OQ-011 | Does the league endpoint expose DL vs back-seven tackle/assist values as separate config fields? | Whether True Position split is MFL-sourced or requires admin override |
| OQ-INJ | Does the MFL players endpoint include injury status designations? | Narrative tier injury status source |

All four answered in one API testing session. Normalizer and scoring parser for affected components finalized after this session.

---

## 19. Locked Decisions Registry

Complete record of every backend decision made in this session.

| Decision | What was decided |
|---|---|
| Hosting Phase 1 | Proxmox homelab Docker container. Local only. |
| Hosting Phase 2 | Cloudflare Tunnel + Zero Trust. Homelab server. Free tier. |
| Hosting Phase 3 | VPS migration. Cloudflare proxy retained. SQLite → PostgreSQL. |
| Backend language | Go natively. One binary. No Python. No sidecars. |
| WebSocket server | Embedded in main Go server. Single binary. Extract at Phase 3 if load justifies. |
| Message history | Full history in SQLite. End-of-season archival to Cloudflare R2 on Commissioner "Close Season" action. |
| DOT Chamber threading | Auto-thread per trade submission. |
| Message model | Typed structured objects, not free text. Each message_type has a defined payload schema. |
| Film sub-signal source abstraction | Engine operates on slot values. Sources are interchangeable at ingestion layer. Engine never references a specific source. |
| RSP discontinuation handling | Three-State Unknown + confidence scaling degrades to neutral. Admin zeroes slot weight if permanent. No engine changes required. |
| League-agnostic ingestion | No Legacy NFL-specific logic in ingestion code. All rules come from league endpoint config or admin console. |
| EDGE position mapping | Option C — Unknown + admin console flag queue. Never silently defaults. |
| PlayerID and FranchiseID types | Go type aliases. `NewPlayerID()` and `NewFranchiseID()` constructors enforce leading zeros. All DB columns TEXT. |
| Engine architecture | Pure function pipeline. No I/O in engine. Six layers chain sequentially. |
| Batch processing | Worker pool, `runtime.NumCPU()` goroutines. Players are independent. |
| Scoring run trigger | Hybrid — daily scheduled + event-triggered partial rescore for affected players. Concurrent runs queue. |
| Partial rescore scope | Traded players only. Layer 5 is percentage-of-league-cap — trades do not require roster-wide rescore. |
| Batch error handling | Skip and flag. Failed player excluded from run. Admin console surfaced. Batch continues. |
| EMA state location | SQLite `ema_state` table. Engine is stateless. `PreviousEMA` carried in `PlayerRecord`. |
| Season transition EMA | CONTINUATION default. Prior season final values are starting point for next season. RESET per sub-signal flag in rubric. |
| Rubric loading | From SQLite at engine startup. Admin changes write to SQLite. Next run reads fresh. No restart required. |
| xFP source | Engine-derived from Module 3 (Matchup Score Predictions). Not external ingestion. `matchup_projections.xfp`. |
| PFF data access | Manual CSV upload via admin console. PFF+ Premium Stats CSV export. `infrastructure/csv_ingestion/`. |
| RAS data access | Kaggle/nflverse CSV. Manual upload. Fuzzy composite key matching (name + draft_year + college + drafted_team). |
| Madden data access | Kaggle CSV. Manual upload. Full attribute-level coverage confirmed. One upload per release cycle. |
| FantasyCalc approved | Tier 4 source. Public API. Replaces KTC as primary automated pick value source. 1QB format for Legacy NFL. |
| Fantasy Points Data Suite approved | Tier 3 supplement for YPRR/TPRR. CSV upload. |
| RotoWire | MFL players endpoint tested first. Commercial feed deferred to Phase 2/3. |
| Pick value model | FantasyCalc API Phase 1/2. Internal historical model (`pick_value_historical`) wireframed and accumulates passively. Phase 3 activates when `sample_size ≥ 3`. |
| SQLite WAL mode | Enabled on initialization. |
| Schema migrations | Sequential numbered migration files. Applied on startup via embedded migration runner. |
| Seasonal archival | Current season in active SQLite. Prior seasons archived to R2 on season close. |
| Franchise identity | `mfl_franchise_id TEXT`, "0001"–"0032". FranchiseID type. "0000" = commissioner ops. |
| Users table | One row Phase 1. Schema supports 32 rows Phase 2. No migration required. |
| Season boundary | Commissioner-triggered "Open New Season" action. Confirmed against MFL season year in Phase 1/2. Commissioner-only in Phase 3. |
| Contract year rollover | Commissioner event trigger. Commissioner fires from Commissioner Panel. Decrements `contract_years` by 1 for all league contracts. |
| Waiver order | Reverse standings at season start. Rotation (claiming team → last position) after each claim. |
| Trade expiration | No independent proposal clock. Deadline-relative only. Expired state triggers when Week 9 deadline passes on unresolved proposal. UI countdown shows time to deadline. |
| Simultaneous bid rejection | Reject and require resubmit. Clear error with current leader and minimum valid increment. No auto-adjustment on GM's behalf. |
| Clock monitor frequency | Adaptive. 30s default. 5s when `UFA_WINDOW_ACTIVE` flag is set. |
| DOT vote expiry | League-configurable via `transaction_configs.dot_expiry_mode`. Some leagues run no expiry. |
| Tiered payload — query paths | Three explicit endpoint paths. Not one parameterized endpoint. |
| Tiered payload — pre-fetch | Narrative immediate. Tactical async. Matrix viewport + 50-row scroll buffer. |
| Reconnect delivery | Current state snapshot only. Not a replay of missed events. |
| Role enforcement | Service layer. Data layer rejection — not just UI hiding. |
| DECISION-009 | All league-configurable values MFL-sourced from league endpoint. |
| DECISION-010 | Historical scores preserved under rules that produced them. `scoring_config_id` on every engine output. |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
*Backend Architecture Session: June 2026*
*Next session: Claude Code build — application shell, directory scaffold, SQLite initialization*

| Version | Date | Changes |
|---|---|---|
| 1.0 | June 2026 | Initial release. Complete backend architecture session output. Hosting model, Go engine structure, MFL API integration, SQLite schema, transaction system, communication architecture, tiered payload system, seasonal state machine, offline capability, role enforcement, historical preservation, Phase 1→2 migration, and locked decisions registry. |
