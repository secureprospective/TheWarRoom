# Legacy NFL — MFL API Complete Reference
League: Legacy NFL
MFL League ID: 14432
Host Server: www47
Season Year: 2026
API Type: Public read-only (no authentication required)
Confirmed live: June 2026

## Base URL Patterns

**League-specific calls (use host server)**
```
https://www47.myfantasyleague.com/{year}/export?TYPE={endpoint}&L=14432&JSON=1
```

**Non-league calls (use load-balanced host)**
```
https://api.myfantasyleague.com/{year}/export?TYPE={endpoint}&JSON=1
```

**Developer portal**
```
https://api.myfantasyleague.com/2026/api_info
```

## Quick Reference — All Endpoints

| Endpoint | Host | League ID Required | Cache | Update Frequency |
|---|---|---|---|---|
| league | www47 | Yes | Session | On config change |
| rosters | www47 | Yes | Daily | On transaction |
| players | api (load-balanced) | No | Daily | Daily |
| contracts | www47 | Yes | Daily | On transaction |
| transactions | www47 | Yes | Event | On transaction |
| standings | www47 | Yes | Weekly | After game results |
| playerScores | www47 | Yes | Weekly | After game results |
| liveScoring | www47 | Yes | Real-time | Active game hours only |
| salaries | www47 | Yes | Daily | On transaction |
| draftResults | www47 | Yes | Post-draft | Post-draft |
| tradeBait | www47 | Yes | Event | On update |
| nflSchedule | api (load-balanced) | No | Seasonal | Real-time |
| rules | www47 | Yes | Session | On config change |

## Endpoint Detail

### league — League Configuration
```
https://www47.myfantasyleague.com/2026/export?TYPE=league&L=14432&JSON=1
```
Cache: Load on startup, re-verify weekly and on every season transition.

**Confirmed fields from live response:**
```
salaryCapAmount         "125"           League cap in millions (string — parse to float)
rosterSize              "80"            Max roster size per team
taxiSquad               "8"             Taxi squad slots
injuredReserve          "12"            IR slots
usesSalaries            "1"             Salary cap active
usesContractYear        "1"             Contract years active
keeperType              "dynasty"       Confirmed dynasty format
draftPlayerPool         "Rookie"        Rookie-only draft pool
startWeek               "1"
endWeek                 "17"            17-week season
lastRegularSeasonWeek   "13"            Regular season ends Week 13
h2h                     "YES"           Head-to-head scoring
precision               "2"             Score decimal places
includeTaxiWithSalary   "100"           Taxi counts 100% toward cap
includeIRWithSalary     "100"           IR counts 100% toward cap
includeTaxiWithContractYear "100"       Taxi counts for contract years
baseURL                 "https://www47.myfantasyleague.com"
id                      "14432"

starters.count          "21"            Total weekly starters
starters.idp_starters   "12"            IDP (defensive) starters
starters.iop_starters   "8"             Offensive starters

starters.position[]     Position flex rules:
  QB  limit 1
  RB  limit 1-3
  WR  limit 2-5
  TE  limit 1-3
  PK  limit 1
  DT  limit 2-4
  DE  limit 2-4
  LB  limit 2-4
  CB  limit 2-4
  S   limit 2-4

franchises[]            All 32 franchise records (see franchises section)
divisions[]             8 divisions (4 AFC, 4 NFC)
conferences[]           2 conferences (AFC id:00, NFC id:01)
history[]               League history back to 2013
```

**Franchise record fields:**
```
id                      "0001" through "0032"   STRING — always 4 chars
name                    Franchise display name
abbrev                  3-4 char abbreviation
owner_name              GM name
email                   GM email
division                Division ID string
waiverSortOrder         Current waiver priority (32 = worst, 1 = best)
salaryCapAmount         Individual cap override (usually empty — use league default)
future_draft_picks      Comma-separated list of owned future picks
lastVisit               Unix timestamp of GM's last MFL login
logo                    Logo URL
icon                    Icon URL
```

Christopher's franchise: ID "0025" (Arizona Cardinals / Gremlin)

### rosters — All Team Rosters
```
https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=14432&JSON=1
```
Cache: Daily. Refresh on any transaction event.

**Confirmed fields from live response:**
```
franchise[].id          "0001" through "0032"   Franchise ID (string)
franchise[].week        Current week number
franchise[].player[]    Array of player records

player record:
  id                    Player ID — STRING. May have leading zeros (e.g., "0835")
  salary                Salary in millions — STRING, parse to float (e.g., "7", "1.30", "17.70")
  contractYear          Final year of contract — STRING, parse to int (e.g., "2026")
  contractStatus        Contract type — STRING, NORMALIZE (see dirty values below)
  contractInfo          Free-text contract notes — DISPLAY ONLY, do not parse
  status                "ROSTER" or "TAXI_SQUAD"
```

**contractStatus dirty values confirmed in live data:**
```
Clean values:    UFA, RFA, FT1, FT2
Dirty values:    "UFA " (trailing space), "UFA  " (two spaces), "EXT (2024)", 
                 "EXT + FT1", "FT1 (2026)", "FT2 (2026)", "YFA" (typo — treat as UFA),
                 "EXT (2024) " (trailing space), "FT1+ EXT2 (2024)", "EXT2"

Normalization rule: strip whitespace, then:
  starts with "UFA" → UFA
  starts with "RFA" → RFA
  starts with "FT1" → FT1
  starts with "FT2" → FT2
  "YFA"            → UFA (confirmed typo in live data)
  anything else    → flag for admin review
```

### players — Full Player Database
```
https://api.myfantasyleague.com/2026/export?TYPE=players&JSON=1
```
Cache: ONCE PER DAY MAXIMUM. MFL enforces this.
**Host correction (B3, 2026-06-18): use the LEAGUE host (www47, L=14432), NOT the global `api` host.** The global feed (2578 live) OMITS commissioner-created players — an owner bids on someone not yet in MFL's DB and the commissioner creates a league-local record (live ids 0816/0820/0835/0838). The league feed (2621) is a superset that includes them, so every rostered id resolves at the B3 join. `internal/ingestion/players` is therefore a league-scoped, DiscoverHost-first call.

**Confirmed fields from live response:**
```
players.player[]        Array of all player records
players.timestamp       Unix timestamp of last database update

player record:
  id                    Player ID — STRING. Leading zeros on IDs under 1000 (e.g., "0531")
  name                  "Last, First" format (e.g., "Mahomes, Patrick")
  position              MFL position string (see position map below)
  team                  3-letter NFL team code, or "FA" for free agent
  status                "R" = 2026 rookie (only present on rookie records)
```

**MFL position codes → engine mapping:**
```
MFL Code    Engine Position    Notes
QB          QB
RB          RB
WR          WR
TE          TE
PK          K                  NORMALIZE: MFL uses PK, engine uses K
DE          DE
DT          DT
LB          LB
CB          CB
S           S
PN          —                  Punter — not scored, filter out
Coach       —                  Filter out
XX          FLAG               Unclassified — manual resolution required
EDGE        DE                 OQ-004 RESOLVED: MFL labels edge rushers DE; no separate EDGE class (0 live)
TMWR        —                  Team WR aggregate — filter out
TMRB        —                  Team RB aggregate — filter out
TMDL        —                  Team DL aggregate — filter out
TMLB        —                  Team LB aggregate — filter out
TMDB        —                  Team DB aggregate — filter out
TMQB        —                  Team QB aggregate — filter out
TMPK        —                  Team PK aggregate — filter out
TMPN        —                  Team PN aggregate — filter out
Def         —                  Team defense aggregate — filter out
ST          —                  Special teams aggregate — filter out
Off         —                  Team offense aggregate — filter out
```

Filter rule: Any player record where id starts with "0" and is in range 0151–0782 is a team aggregate entry. Filter these entirely. Also filter any record with position = Coach, PN, TMWR, TMRB, TMDL, TMLB, TMDB, TMQB, TMPK, TMPN, Def, ST, Off.

2026 Rookie identification: status == "R" on the player record.

**MFL NFL team codes (confirmed from live data):**
```
BUF, IND, MIA, NEP, NYJ, CIN, CLE, TEN, JAC, PIT, DEN, KCC, LVR, LAC,
SEA, DAL, NYG, PHI, ARI, WAS, CHI, DET, GBP, MIN, TBB, ATL, CAR, LAR,
NOS, SFO, BAL, HOU
FA = Free Agent
```

### contracts — Salary and Contract Data
```
https://www47.myfantasyleague.com/2026/export?TYPE=contracts&L=14432&JSON=1
```
Cache: Daily. Refresh on transaction events.
Overlaps significantly with rosters — use rosters as primary source; contracts may provide additional fields.

### transactions — Transaction Log
```
https://www47.myfantasyleague.com/2026/export?TYPE=transactions&L=14432&JSON=1
```
Cache: No cache — event-triggered fetch only.

### standings — League Standings
```
https://www47.myfantasyleague.com/2026/export?TYPE=standings&L=14432&JSON=1
```
Cache: Weekly, after final game results post.

### playerScores — Player Scoring by Week
```
https://www47.myfantasyleague.com/2026/export?TYPE=playerScores&L=14432&W={week}&JSON=1
```
Cache: Weekly, after final results. Replace {week} with week number (1–17).
Note: Long play bonuses are discrete separate stat events (P40, R20, R40, C20, C40). A 43-yard run triggers both R20 and R40. OQ-003 resolved — see docs/data-layer/MFL_Scoring_Rules_Decode.md.

### liveScoring — Real-Time In-Game Scoring
```
https://www47.myfantasyleague.com/2026/export?TYPE=liveScoring&L=14432&JSON=1
```
Cache: NO CACHE. Poll only during active game windows (derived from nflSchedule).

### salaries — League-Wide Salary Data
```
https://www47.myfantasyleague.com/2026/export?TYPE=salaries&L=14432&JSON=1
```
Cache: Daily.
Note: Cardinals export showed a $5.49 adjustment line item in prior testing. Watch for salary adjustment fields — raw salary sum may not equal cap usage (OQ-005 open).

### draftResults — Rookie Draft Results
```
https://www47.myfantasyleague.com/2026/export?TYPE=draftResults&L=14432&JSON=1
```
Cache: Post-draft, static for the season.

### tradeBait — Active Trade Block Listings
```
https://www47.myfantasyleague.com/2026/export?TYPE=tradeBait&L=14432&JSON=1
```
Cache: Event-triggered.

### nflSchedule — NFL Schedule Data
```
https://api.myfantasyleague.com/2026/export?TYPE=nflSchedule&JSON=1
```
Cache: Seasonal. Use to determine active game windows for liveScoring polling.

### rules — Scoring Rules Configuration
```
https://www47.myfantasyleague.com/2026/export?TYPE=rules&L=14432&JSON=1
```
Cache: Load on startup with league endpoint.
**Resolved via this endpoint:** OQ-002 (missed FG values), OQ-003 (long play bonus format), OQ-011 (True Position split). Live-tested June 2026. Full decode at docs/data-layer/MFL_Scoring_Rules_Decode.md.

## Hard Rules — Non-Negotiable

```
1. Player IDs are ALWAYS strings. Never integers.
   Leading zeros on IDs under 1000 MUST be preserved.
   Enforce at ingestion boundary. Never allow integer conversion downstream.
   
2. players endpoint: once per day MAXIMUM.
   Cache the full response. All lookups read from cache.
   
3. liveScoring: only during active game windows.
   Derive windows from nflSchedule. Never poll on off-days.
   
4. 429 response: exponential backoff.
   1s → 2s → 4s → ... → 60s max.
   Never retry immediately.
   
5. All calls are server-side.
   MFL blocks cross-origin browser requests.
   No direct browser → MFL API calls.
   
6. Host server www47 can change.
   Fetch from league endpoint on startup, cache it.
   Re-verify on each season transition and weekly.
   If www47 fails: fall back to api.myfantasyleague.com for re-discovery call.
   
7. salaryCapAmount from league endpoint is a string.
   Parse to float. Confirmed: "125" = $125M.
   
8. salary from rosters/contracts is a string.
   Parse to float. Examples: "7" = $7M, "1.30" = $1.30M, "17.70" = $17.70M.
```

## Startup Call Order (dependency sequence)

```
Tier 0 — Must complete before engine runs:
  players    → master player ID/name/position lookup table
  
Tier 1 — Must complete before scoring:
  league     → cap amount, roster rules, scoring config, franchise list
  rules      → scoring point values (OQ-002/003/011 resolved — see MFL_Scoring_Rules_Decode.md)

Tier 2 — Roster state (depends on Tier 0 + 1):
  rosters
  contracts
  salaries

Tier 3 — Supporting data:
  standings
  transactions
  tradeBait
  nflSchedule
  draftResults

Tier 4 — In-season only:
  playerScores    (weekly, after results)
  liveScoring     (real-time, active game windows only)
```

## Open Questions — API-Related

```
OQ-002  Missed FG scoring value
        RESOLVED — MG 0-29 yards = -3 pts, MG 30-99 yards = -1 pt.
        Confirmed via live rules endpoint June 2026.
        See docs/data-layer/MFL_Scoring_Rules_Decode.md.

OQ-003  Long play bonus format in playerScores
        RESOLVED — Discrete separate stat events (P40, R20, R40, C20, C40).
        Not embedded in yardage totals. A 43-yard run triggers both R20 and R40.
        Confirmed via MFL rules endpoint June 2026.
        See docs/data-layer/MFL_Scoring_Rules_Decode.md.

OQ-005  Salary adjustment line items
        RESOLVED (B3, 2026-06-18). Source = the salaryAdjustments export
        type (league-scoped), the per-franchise DEAD-CAP ledger of unclaimed
        drops: fields franchise_id/amount/description/id/timestamp. NOT the
        salaries endpoint (that is per-player, league-level, no franchise key,
        no adjustment field). Live: franchise 0025 = 20 entries = $5.495.
        Cap usage = Sum(roster salaries) + Sum(salaryAdjustments.amount) per
        franchise. Negative amounts are VALID (commissioner cap credits).
        Built: internal/ingestion/salaryadjustments. Aggregation deferred to
        the cap-math engine.

OQ-011  True Position split in league endpoint
        RESOLVED — Additive mechanic confirmed. Universal base: TK 1.5 / AS 1.0.
        DT/DE +1.0/+0.5 additive → 2.5/1.5. CB/S +0.5/+0.0 additive → 2.0/1.0.
        LB base only. Confirmed via live rules endpoint June 2026.
        See docs/data-layer/MFL_Scoring_Rules_Decode.md.
```

## Cross-Reference Join Pattern

```
rosters response    →  player.id (string)
                              ↓
players cache       →  lookup by id
                              ↓
                       name, position (normalize), team

Complete player record for engine input:
mfl_id          from rosters player.id
name            from players lookup
position        from players lookup (normalize PK→K, flag XX/EDGE)
nfl_team        from players lookup
salary          from rosters player.salary (parse to float)
contract_year   from rosters player.contractYear (parse to int)
contract_status from rosters player.contractStatus (normalize)
contract_info   from rosters player.contractInfo (display string only)
roster_status   from rosters player.status (ROSTER or TAXI_SQUAD)
is_rookie       from players player.status == "R"
franchise_id    from rosters franchise.id (parent)
```

---
Legacy NFL — MFL API Reference
Compiled June 2026
Live-tested against League ID 14432, host www47
