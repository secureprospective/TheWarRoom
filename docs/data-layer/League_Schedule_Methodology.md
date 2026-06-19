# Legacy NFL — League Schedule Methodology (OQ-012)
Version: 1.0 — June 2026
Status: Design spec. Resolves OQ-012 (league fantasy schedule source & schema). Generator build is a separate session.

## Overview

The Legacy NFL fantasy league mirrors the real NFL's structure and scheduling logic. This document is the **deterministic methodology** for generating the 13-week regular-season schedule and the 4-week postseason bracket.

The schedule is a pure function of the league's prior-year standings: feed in the 32 franchises with their conference, division, and prior-year division rank, and the full schedule falls out with no further input. Only the cross-conference games key off placement — everything else is a fixed rotation.

This is the **generation** spec. At runtime the live league matchups are commissioner-defined in MFL; ingestion of the live league schedule is a separate concern (not the `nflSchedule` fetcher from B2, which is real NFL games).

---

## League Structure

- **2 conferences:** NFC, AFC
- **4 divisions per conference:** West, East, North, South
- **4 teams per division** → 32 franchises total
- Every franchise carries a **rank 1–4 = its order of finish in its division last season.**

Rank is the only thing prior-year standings feed into. It seeds the cross-conference matchups (and, downstream, the playoff seeding).

### Seeding input (per franchise)

| Field | Values | Source |
|---|---|---|
| `conference` | NFC, AFC | League config (fixed) |
| `division` | West, East, North, South | League config (fixed) |
| `rank` | 1, 2, 3, 4 | Prior-year division finish; **preseason power ranking in the inaugural season** |

Conference and division are permanent franchise attributes. Only `rank` changes year to year.

**Rank source:** in any established year, `rank` is the franchise's order of finish in its division last season (1 = won the division, 4 = last). In the **inaugural season** there is no prior year, so rank comes from a **preseason power ranking** — the top team in each division is #1, the bottom is #4.

---

## Regular Season — 13 Weeks

Each franchise plays 13 games across three buckets. 32 teams = 16 matchups every week; no byes.

| Bucket | Games | Weeks | Determined by |
|---|---|---|---|
| **A — Division** (double round-robin) | 6 | 2, 4, 6, 8, 10, 12 | Fixed rotation |
| **B — Intra-conference, inter-divisional** | 4 | 3, 5, 9, 11 | Fixed division pairing + fixed rank rotation |
| **C — Cross-conference** | 3 | 1, 7, 13 | Prior-year placement (weeks 1 & 7) + fixed filler (week 13) |

### Bucket A — Division games (6)

Within each division, teams ranked 1–4 play a full round-robin **twice**. Same pairing pattern both rounds:

| Round 1 week | Round 2 week | Matchups |
|---|---|---|
| 2 | 8 | 1 vs 4, 2 vs 3 |
| 4 | 10 | 1 vs 3, 2 vs 4 |
| 6 | 12 | 1 vs 2, 3 vs 4 |

Each franchise plays its three division rivals twice = 6 games.

### Bucket B — Intra-conference, inter-divisional games (4)

> *(Christopher's original label was "Non-Division Inter Conference"; these matchups are within the **same** conference across divisions, so the doc calls them intra-conference, inter-divisional.)*

Each division is permanently paired with one other division **in its own conference**:

- **West ↔ North**
- **East ↔ South**

(Applies identically in both NFC and AFC.)

A franchise plays **all four** teams of its paired division, one per week. With paired divisions A (teams a1–a4) and B (teams b1–b4):

| Week | Rank pairing | Matchups |
|---|---|---|
| 3 | rank-equal | a1–b1, a2–b2, a3–b3, a4–b4 |
| 5 | swap within {1,2} and {3,4} | a1–b2, a2–b1, a3–b4, a4–b3 |
| 9 | cross {1,3} and {2,4} | a1–b3, a3–b1, a2–b4, a4–b2 |
| 11 | cross {1,4} and {2,3} | a1–b4, a4–b1, a2–b3, a3–b2 |

Over the four weeks each team meets every team of the paired division exactly once.

### Bucket C — Cross-conference games (3)

Cross-conference division pairings rotate on the cycle **W → N → E → S → W** (applied to the AFC side; NFC division is the anchor). The two **placement** games (weeks 1, 7) pair rank-equals; the week-13 **filler** crosses ranks.

| Week | Cycle shift | Division pairings (NFC ↔ AFC) | Rank pairing | Type |
|---|---|---|---|---|
| 1 | identity | W↔W, E↔E, N↔N, S↔S | 1v1, 2v2, 3v3, 4v4 | **Placement** |
| 7 | +1 | W↔N, E↔S, N↔E, S↔W | 1v1, 2v2, 3v3, 4v4 | **Placement** |
| 13 | +2 | W↔E, E↔W, N↔S, S↔N | 1v4, 2v3, 3v2, 4v1 | Fixed filler |

**Only weeks 1 & 7 are true prior-placement games** (rank-vs-rank across conferences). Week 13 is a fixed rank-crossed filler — not placement-driven. Each division ends up facing 3 of the 4 cross-conference divisions (one it never plays that year).

### Home / Away assignment

13 is odd, so a perfect split is impossible; the rule below gives every franchise the best-possible **7 home / 6 away** (or 6/7) and balances the league exactly (16 teams at 7, 16 at 7-away). Home/away has no scoring effect in fantasy — this is purely to populate MFL's field deterministically.

| Bucket | Rule | Result per team |
|---|---|---|
| **A — Division** | Higher-ranked (lower number) team hosts the round-1 meeting; the rematch flips | 3 home / 3 away |
| **B — Intra-conf** | West/East division hosts weeks 3 & 9; North/South hosts weeks 5 & 11 | 2 home / 2 away |
| **C — Cross-conf** | NFC hosts weeks 1 & 13; AFC hosts week 7 | NFC 2 / AFC 1 home |

Net: **every NFC franchise = 7 home / 6 away; every AFC franchise = 6 home / 7 away.** (Optionally flip the Bucket-C conference each season to even it out over time — not important.)

### Worked example — NFC West, rank 1

| Wk | Opp | Bucket |
|---|---|---|
| 1 | AFC West 1 | C (placement) |
| 2 | NFC West 4 | A |
| 3 | NFC North 1 | B |
| 4 | NFC West 3 | A |
| 5 | NFC North 2 | B |
| 6 | NFC West 2 | A |
| 7 | AFC North 1 | C (placement) |
| 8 | NFC West 4 | A |
| 9 | NFC North 3 | B |
| 10 | NFC West 3 | A |
| 11 | NFC North 4 | B |
| 12 | NFC West 2 | A |
| 13 | AFC East 4 | C (filler) |

13 games: 3 division rivals ×2, all 4 of NFC North ×1, three distinct cross-conference opponents. No accidental repeats.

---

## Postseason — Weeks 14–17

**7 teams per conference** (14 total) make the playoffs.

### Seeding (per conference)

- **Seeds 1–4:** the four **division winners**, ordered by record.
- **Seeds 5–7:** three **wildcards** — the best non-division-winners, ordered by **standard NFL tiebreakers**.

Tiebreaker ladder spelled out at bracket-build time (head-to-head → division record → common games → points). Ties to **OQ-010** (playoff bid rules).

### Bracket

Reseed every round so the highest remaining seed always draws the lowest.

| Week | Round | Games/conf | Format |
|---|---|---|---|
| 14 | Wild Card | 3 | 2v7, 3v6, 4v5 — **#1 seed byes** |
| 15 | Divisional | 2 | #1 vs lowest remaining seed; other two reseed |
| 16 | Conference Championship | 1 | highest vs lowest remaining |
| 17 | Championship | 1 (total) | NFC champ vs AFC champ |

---

## Schedule Schema (data representation)

A generated schedule is a list of matchups:

| Field | Type | Notes |
|---|---|---|
| `week` | int (1–17) | 1–13 regular season, 14–17 postseason |
| `home_franchise` | string | MFL franchise ID (e.g. `"0007"`) |
| `away_franchise` | string | MFL franchise ID |
| `game_type` | enum | `DIVISION`, `INTRA_CONF`, `CROSS_CONF_PLACEMENT`, `CROSS_CONF_FILLER`, `PLAYOFF_WILDCARD`, `PLAYOFF_DIVISIONAL`, `PLAYOFF_CONFERENCE`, `PLAYOFF_CHAMPIONSHIP` |

The generator's input is the per-franchise seeding table (conference, division, rank). The regular-season output (weeks 1–13) is fully deterministic from that input. The postseason (weeks 14–17) is generated after week 13 from final standings.

---

## Open Items

| Item | Status |
|---|---|
| **Home/away designation** | RESOLVED 2026-06-19 — deterministic rule yielding 7/6 per team (see *Home / Away assignment*). Cosmetic; no scoring effect. |
| **Inaugural-season rank seeding** | RESOLVED 2026-06-19 — preseason power ranking (top of division = #1, bottom = #4); established years use prior-year division finish. |
| **Playoff tiebreaker ladder (OQ-010)** | "Standard NFL" locked at design level; exact ordering spelled out at bracket build. |
| **Live league-schedule ingestion** | This is the generation spec. Reading the commissioner-entered schedule back from MFL is a separate ingestion concern (distinct from B2's `nflSchedule`). |

---

*Built by: Christopher Campbell + Claude (Anthropic)*
