# Legacy NFL — MFL Scoring Rules Decode
**Source:** rules endpoint — live response June 2026
**League:** 14432 / www47
**Status:** Verified against Official_Rulebook.md

---

## Open Questions Resolved by This Response

| OQ | Question | Resolution |
|---|---|---|
| OQ-002 | Missed FG point value | MG 0-29 yards = -3 pts / MG 30-99 yards = -1 pt ✓ |
| OQ-003 | Long play bonus format | Separate events (P40, R20, R40, C20, C40). Not embedded in total points. ✓ |
| OL-040 | PAT/XP scoring | EP made = +1 pt, EM missed = -1 pt ✓ |
| OQ-041 | FG 70+ threshold | No separate 70+ tier. FG 60-99 = 6 pts covers all 60+ attempts. ✓ |
| OQ-011 | True Position split in rules | YES — position-specific tackle/assist rules confirmed in endpoint ✓ |

---

## Complete Scoring Table (All Positions)

### Passing

| Event Code | Stat | Points |
|---|---|---|
| #PP | Passing TD | 5 |
| PY | Passing Yards | 0.05 / yard |
| IN | Interception Thrown | -2 |
| P40 | Long Pass (40+ yards) | +1 per qualifying throw |
| P2 | 2-Point Conversion (pass) | 2 |

### Rushing

| Event Code | Stat | Points |
|---|---|---|
| #RR | Rushing TD | 6 |
| RY | Rushing Yards | 0.1 / yard |
| RA | Rushing Attempt | 0.15 |
| R20 | Long Rush (20+ yards) | +1 per qualifying run |
| R40 | Long Rush (40+ yards) | +1 additional per qualifying run |
| R2 | 2-Point Conversion (rush) | 2 |

### Receiving

| Event Code | Stat | Points |
|---|---|---|
| #CC | Receiving TD | 6 |
| CY | Receiving Yards | 0.1 / yard |
| CC | Reception (PPR) | 1 |
| C20 | Long Reception (20+ yards) | +1 per qualifying catch |
| C40 | Long Reception (40+ yards) | +1 additional per qualifying catch |
| C2 | 2-Point Conversion (catch) | 2 |

### Long Play Bonus Notes (OQ-003 Confirmed)

Long play bonuses are discrete events that fire per qualifying play.

- A 43-yard rush triggers BOTH R20 (+1) AND R40 (+1) = +2 bonus points.
- A 43-yard catch triggers BOTH C20 (+1) AND C40 (+1) = +2 bonus points.
- These appear as separate stat events in the playerScores endpoint.

### Kicking

| Event Code | Stat | Points |
|---|---|---|
| FG (0-39) | Field Goal Made, 0–39 yards | 3 |
| FG (40-49) | Field Goal Made, 40–49 yards | 4 |
| FG (50-59) | Field Goal Made, 50–59 yards | 5 |
| FG (60-99) | Field Goal Made, 60+ yards | 6 |
| MG (0-29) | Missed FG, under 30 yards | -3 |
| MG (30-99) | Missed FG, 30+ yards | -1 |
| EP | Extra Point Made (PAT) | +1 |
| EM | Extra Point Missed (PAT) | -1 |

### Returns

| Event Code | Stat | Points |
|---|---|---|
| #UT | Punt Return TD | 6 |
| UY | Punt Return Yards | 0.025 / yard |
| #KT | Kickoff Return TD | 10 |
| KY | Kickoff Return Yards | 0.025 / yard |

### Fumbles

| Event Code | Stat | Points |
|---|---|---|
| FU | Fumble Lost | -2 |
| OFC | Own Fumble Recovery for TD | 2 |
| FC | Opponent Fumble Recovery | 3 |
| FCY | Fumble Recovery Return Yards | 0.1 / yard |
| FF | Forced Fumble | 4 |

### IDP Defense — Universal (All Positions)

| Event Code | Stat | Points |
|---|---|---|
| IC | Interception | 5 |
| ICY | Interception Return Yards | 0.1 / yard |
| #IR | Interception Return TD | 6 |
| PD | Pass Defensed | 2.5 |
| SK | Sack | 4.5 |
| TKL | Tackle for Loss | 2.5 |
| QH | QB Hit | 1 |
| SF | Safety | 10 |
| BLF | Blocked Field Goal | 7 |
| BLP | Blocked Punt | 7 |
| BLE | Blocked Extra Point | 7 |
| #BF | Blocked FG Returned TD | 7 |
| #BP | Blocked Punt Returned TD | 7 |
| #DR | Defensive Return TD | 6 |

---

## True Position Tackle / Assist Split

### How MFL Encodes It

MFL uses **ADDITIVE position-specific rules** stacked on a universal base rule.

**Universal base (all positions):**
- TK = 1.5 pts per tackle
- AS = 1.0 pts per assist

**DT-specific additional rule:**
- TK = +1.0 (stacks with base → **2.5 total**)
- AS = +0.5 (stacks with base → **1.5 total**)

**DE-specific additional rule:**
- TK = +1.0 (stacks with base → **2.5 total**)
- AS = +0.5 (stacks with base → **1.5 total**)

**LB (no override — base only):**
- TK = 1.5
- AS = 1.0

Confirmed match against rulebook for DT, DE, LB. ✓

### CB and S — Confirmed (MFL Config is Correct)

MFL has separate position-specific rules for CB and S (both apply the same additive modifiers):

**CB-specific additional rule:**
- TK = +0.5 (stacks with base → **2.0 total**)
- IC = +1 (stacks with base → **6.0 total**)
- PD = +0.5 (stacks with base → **3.0 total**)

**S-specific additional rule:**
- TK = +0.5 (stacks with base → **2.0 total**)
- IC = +1 (stacks with base → **6.0 total**)
- PD = +0.5 (stacks with base → **3.0 total**)

MFL config is correct. Rulebook document understated CB and S scoring. Engine uses MFL values. ✓

### Final Tackle / Assist Reference Table

| Position | Tackle | Assist | Source |
|---|---|---|---|
| DT | 2.5 | 1.5 | Base + DT rule (additive) — confirmed ✓ |
| DE | 2.5 | 1.5 | Base + DE rule (additive) — confirmed ✓ |
| LB | 1.5 | 1.0 | Base only — confirmed ✓ |
| CB | 2.0 | 1.0 | Base + CB rule (additive) — confirmed ✓ |
| S | 2.0 | 1.0 | Base + S rule (additive) — confirmed ✓ |
| QB | 1.5 | 1.0 | Base only (rare event — QB makes a tackle) |

---

## Rulebook Corrections (MFL Config is Authoritative)

The following rulebook values were incorrect. MFL config is confirmed correct per DECISION-009.

| Stat | Rulebook (incorrect) | MFL Config (confirmed) |
|---|---|---|
| Interception Return Yards | 0.025 / yard | **0.1 / yard** ✓ |
| Fumble Recovery Return Yards | 0.025 / yard | **0.1 / yard** ✓ |
| CB Tackle | 1.5 | **2.0** (base + CB additive) ✓ |
| S Tackle | 1.5 | **2.0** (base + S additive) ✓ |
| CB Interception | 5 | **6** (base + CB additive) ✓ |
| S Interception | 5 | **6** (base + S additive) ✓ |
| CB Pass Defensed | 2.5 | **3.0** (base + CB additive) ✓ |
| S Pass Defensed | 2.5 | **3.0** (base + S additive) ✓ |

All other scoring events: exact match between rulebook and MFL config. ✓

---

## Claude Code Implementation Notes

The rules endpoint is the source of truth per DECISION-009 (MFL-sourced config). All values confirmed.

**Implementation:**

1. Build scoring with ADDITIVE rule interpretation (consistent across DT/DE/CB/S)
2. CB and S are separate position rules in MFL — implement as separate rule sets in the engine
3. ICY and FCY are 0.1/yard (not 0.025 — rulebook was wrong)

**Event codes to implement as separate events (not embedded in yardage totals):**

P40, R20, R40, C20, C40 — these are discrete per-play bonuses. Read them as individual stat events from playerScores, not calculated from raw yards.

---

*Decoded from live rules endpoint — Legacy NFL 14432 — June 2026*
