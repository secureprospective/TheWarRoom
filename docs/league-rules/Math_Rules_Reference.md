# Contract & Money Math — Rules Reference

**Status:** Canonical extraction of every mathematical rule in `Official_Rulebook.md` (v2026). This is the single place the engine's money math is specified. When code and this doc disagree, fix the code; when this doc and the rulebook disagree, the rulebook wins and this doc is corrected.
**Written:** 2026-07-04 (session `session/salary-ledger`). Pair with `docs/transactions/Salary_Ledger_Design.md`.
**Purpose:** stop re-deriving (and re-arguing) the same formulas every session. Every number below is cited to its rulebook section.

---

## 0. MONEY & ROUNDING — the rules that touch EVERYTHING (read first)

| Rule | Value | Source | Note |
|------|-------|--------|------|
| **Salary granularity** | **All salaries round to nearest $10,000** | §1 | `$10,987,000 → $10.99M`. Applies to EVERY money figure the engine computes, not just bids. |
| Internal money type | `int64` **cents** (`$1 = 100`; `$10k = 1,000,000` cents) | impl (OQ-014) | $10k granularity = round to nearest `1_000_000` cents. |
| Display | millions, 2 decimals | §1 | `$10.99M`. Display only — never the stored precision. |
| Cap (2026) | **$125M** per team | §1 | Per-SEASON value ("for 2026") — sourced from MFL/config (B3b), not hardcoded. Teams under cap **at all times**. |

**MISSED-STEP GUARD — the $10k snap.** All percentage math (35% dead cap, 30% retirement, buyout %, 120% tag, 150% extension, RFA %) computes to exact cents **round-half-up FIRST**, then the $10k snap is applied via ONE shared `domain.RoundToNearest10k(Money)` — `result = pct(base).RoundToNearest10k()`, never the reverse (rounding-first vs -last diverges by up to $10k/op). The $10k snap is applied NOWHERE in current code — add it in the refit.

**RESOLVED — snap SCOPE (Christopher, 2026-07-04): ALWAYS nearest $10k, EVERYWHERE. One flat rule, not individualized per op.** Every money figure the system produces — salary cells, owner-directed moves, AND every derived charge (dead cap, retirement, buyout, tag price, extension terms) — snaps to the nearest $10,000 (`RoundToNearest10k`, half-up) AFTER its math. No "salaries-only vs charges-exact" split; the earlier GLM/Gemini divergence is closed in favor of the simplest rule. Charges are computed exactly, then snapped like everything else — the cell IS the snapped figure, no separate display-rounding path.

---

## 1. CONTRACT FUNDAMENTALS

| Rule | Value | Source |
|------|-------|--------|
| Flat annual salaries | **No backloading/frontloading — every contract year is the same salary** | §6 | 
| Max length at signing | **4 years** | §6 |
| Max length with extension | **6 total years** | §10 |
| Rookie R1 length | **5 years** (exceeds the FA max — draft-specific) | §15 |
| Rookie R2–R5 length | **3 years** + RFA | §15 |

**Ledger consequence:** a bid/seed/rookie contract is N identical cells (flat, by §6). Per-year variation is created ONLY by restructure (moves money between cells) and extension (appends cells at a different price). This is why the ledger's flat-fill seed is rule-correct, not an approximation.

---

## 2. FREE AGENCY — bidding (§6)

**Bid points** (auction currency — NOT the cap hit): `Annual Salary (millions) × Year Multiplier`.

| Years | Multiplier | | Constraint | Value |
|-------|-----------|-|------------|-------|
| 1 | 1.00× | | Max 1-year bid | $12M |
| 2 | 1.20× | | $12M–$24M bid | needs ≥2 years |
| 3 | 1.40× | | Over $24M bid | needs ≥3 years |
| 4 | 1.60× | | Max active bids | 30 |

- Examples: `1YR/$5M = 5.000`, `2YR/$5M = 6.000`, `3YR/$1.3M = 1.820`. Bid totals to **3 decimals**; each new bid **> previous by ≥ 0.1**.
- **Snipe:** any bid after the 20-hour mark must exceed previous by a **full point** or is invalid.
- **Phase rules:** multi-year bids close end of Week 13; after Week 12 playoff teams may place 1-year bids only, non-playoff teams may not bid; during playoffs the $12M 1-year cap is **lifted**.
- **Cap hit = the annual salary** (flat across the contract's cells). Bid points decide who wins the auction only.

### Minimum salaries by NFL experience (§6)
| Exp (yrs) | Min | | Exp | Min |
|-----------|-----|-|-----|-----|
| 0 (rookie) | $330,000 | | 4–6 | $530,000 |
| 1 | $380,000 | | 7–9 | $580,000 |
| 2 | $430,000 | | 10+ | $630,000 |
| 3 | $480,000 | | | |

**OPEN RULING (§14):** does the experience minimum floor apply only at bid time, or to every computed salary (extension/restructure result)? Not specified.

---

## 3. RESTRICTED FREE AGENCY (§7)

**Tender price → pick compensation:**
| Tender | Comp | | Tender | Comp |
|--------|------|-|--------|------|
| $600K | 5th | | $2.5M | 2nd |
| $1.0M | 4th | | $5.0M | 1st |
| $1.5M | 3rd | | $6.5M | 1st + 2nd |

- Tender rises to **110% of previous year's salary** if that exceeds the standard price.
- **Baseline re-sign** (no competing offer) = tender price + : 1yr **+0%**, 2yr **+15%**, 3yr **+30%**, 4yr **+45%**.
- Winning offer stands 24h → rights team may match / exceed / let walk for pick comp.

---

## 4. DEAD CAP — waiver/cut (§8) — RESOLVED

- **Formula: 35% × EACH remaining year's cell, summed** (flat per-cell — Christopher-locked 2026-07-04). `35%/35%/35%` across the remaining PAID cells; nothing on the UFA year.
- **50%** (not 35%) per remaining year if the contract **was restructured** (§11).
- **0** dead cap if the player is **claimed** on waivers, or in his final/expiring year (0 remaining).
- Whole charge lands in the **cut year** (flat, no spread) — already shipped (B7b, `dead_cap_ledger`).
- Rounding: compute the charge exactly per remaining cell, then snap to nearest $10k (§0 RESOLVED — universal snap, flat math). Cannot re-sign a waived player for the rest of that season.

---

## 5. FRANCHISE TAG (§9)

| Rule | Formula | Status |
|------|---------|--------|
| Tag price | **average salary of top-5 players at that position, league-wide** (BASE salary) | SHIPPED |
| Floor | if tag price < player's **prior-year salary** → **120% of prior year** | SHIPPED |
| Consecutive limit | max **2 consecutive years**; **2nd tag = 120% of first tag** | DEFERRED (needs cross-season per-player history) |
| Tagged extension | extension years cost **120% of tag price/year** (not 150%) | cross-rule → §10 follow-up |
| Playoff exclusion | FA contracts signed during playoffs don't count toward tag calc | not yet modeled |

---

## 6. CONTRACT EXTENSION (§10)

- **One per GM per season.** Player must have **≥1 year remaining**; **UFAs ineligible**.
- Adds **≤3 years**; **≤6 total** contract years (= count of PAID cells).
- **Extension years priced at 150% of the player's highest-paid remaining year**, subject to **position floor (whichever is greater)**:

| Pos | Floor | | Pos | Floor |
|-----|-------|-|-----|-------|
| QB | $15M | | DE | $7M |
| WR | $10M | | S | $5M |
| RB, TE, LB | $8M | | DT | $4M |
| | | | CB, K | $3M |

- Per extension year: `max(1.5 × highest remaining PAID cell, position_floor)`, then §0 $10k snap.
- **No second extension off a prior extension** — must reach free agency first. Requires a **contract-generation / `is_extended` marker** (Panel-1 finding — cells alone can't express it).
- Each extension **unlocks one additional restructure** (§11).

---

## 7. CONTRACT RESTRUCTURE (§11) — RESOLVED

- **Owner-directed money movement between the player's existing cells** — total **conserved** (changes only *when* salary hits the cap).
- **Tier max, per SOURCE year** (Christopher-locked 2026-07-04): each source year's reduction ≤ its own tier:

| That year's cell salary | Max move out of it |
|-------------------------|--------------------|
| ≥ $3M | $1M |
| ≥ $6M | $2M |
| ≥ $12M | $3M |
| < $3M | ineligible |

- **One restructure per team per year**; **once per contract** (+1 per extension, §6 above). No cell may go negative; destinations must be existing contract years of the same player.
- A restructured contract carries the **50% dead-cap** rate (§4).

---

## 8. CONTRACT BUYOUT (§12) — needs season-phase concept

- **Two per team per season, OFFSEASON only.** Cannot bid on a bought-out player until the following offseason.
- **Charge = rate × average remaining salary:**

| Years remaining | Rate |
|-----------------|------|
| 2 | 60% |
| 3 | 75% |
| 4 | 90% |

- "Average remaining salary" = mean of the remaining PAID cells (per-cell ledger), then rate, then §0 $10k snap.
- **OPEN (§14):** no rate given for **1 year remaining** — is a 1-year-left buyout disallowed, or does it fall back to §8 waiver? Not specified.

---

## 9. SPECIAL SITUATIONS (§13)

| Situation | Math | Source |
|-----------|------|--------|
| **Retirement** | Team owes **30% of remaining contract per year left** → treat as `30% × each remaining cell, summed` (a dead-cap cousin) | §13 |
| **Death (Gaines Adams Rule)** | Remove from roster, **0 cap penalty** (a release with zero dead cap) | §13 |
| **Cap Relief Appeal** | Commissioner **reduces** a cap hit (career-ending/recurring injury, suspension) — a commissioner-override case | §13 |

---

## 10. ROOKIE DRAFT — fixed slot pricing (§15)

**Round 1 (5-year contracts):** picks 1–10 $3.0M · 11–15 $2.4M · 16–20 $2.0M · 21–32 $1.5M
**Round 2 (3yr + RFA):** 1–12 $1.42M · 13–24 $1.25M · 25–32 $1.08M
**Round 3 (3yr + RFA):** 1–16 $0.83M · 17–32 $0.67M
**Round 4** (3yr + RFA) $0.60M · **Round 5** (3yr + RFA) $0.50M
- Max **15 rookie selections** per team per draft; picks beyond 15 before the clock are forfeited without comp.
- Rookie signing writes flat cells for the contract length (R1 = 5, R2–5 = 3).

---

## 11. TRADES (§14 rulebook)

- Deadline **Week 9**; needs **3 approves or 3 vetoes** from DOT. Salary-cap trading **abolished (2017)** — no cap space exchanged. Picks tradeable **≤2 years** in advance; comp picks not tradeable and not usable as RFA comp.
- **Ledger:** a trade reassigns a player's future cells to the new franchise (cells are franchise-agnostic; cap follows the current roster). No money math.

---

## 12. SEASON STRUCTURE / PHASE (§5) — timing that gates money ops

- Regular season ends **Week 13**. **14 playoff teams** (division winners + 2 wild cards/conf). #1 seed/conf earns a bye. Playoff **home field = +3 points** to home team.
- Phase-gated money rules that need a season-phase concept: buyout (**offseason only**, §8 here), bidding phases (post-Week-12 playoff-only 1-year bids; playoff $12M-cap lift, §2).

---

## 13. SCORING (§4) — engine L2 (currently DEFERRED, BasePoints is a proxy)

Full stat→points is in the rulebook §4 (passing/rushing/receiving/kicking/returns/fumbles/defense). Captured there; NOT re-transcribed here because L2 base scoring is deferred (M1 uses a labeled MFL proxy). Two structural notes that affect the engine:
- **True-Position tackle split:** DT/DE tackle = **2.5**, assist = **1.5**; all other defenders tackle = **1.5**, assist = **1**. This is the base rate, not a bonus (§4).
- **Tackle-for-Loss is a direct stat (2.5)** — never approximated (§4). Long-play bonuses stack (20+ and 40+ both add).

Starting lineup = **21 starters** (9 off / 12 def, §3); invalid lineup zeroes the highest-scoring starter that week.

---

## 14. RULINGS — all RESOLVED with Christopher 2026-07-04

1. **Rounding order + direction + scope** — RESOLVED: %-math round-half-up to exact cents FIRST, **then** `RoundToNearest10k` (never reversed). **SCOPE RESOLVED (Christopher 2026-07-04): universal — snap EVERY money figure (salaries, owner moves, ALL derived charges) to nearest $10k. Flat rule, not per-op.**
2. **Experience minimum-salary floor** — RESOLVED: applies **only to UFA free-agency bidding** (§6). It does NOT floor extension or restructure results (extension has its own §10 position floor).
3. **Buyout at 1 year remaining** — RESOLVED: **disallowed** — nobody buys out a 1-year deal. A 1-year-left exit is the normal §8 waiver (35% × the one remaining cell).
4. **Retirement per-cell** — RESOLVED: **`30% × each remaining cell, summed`** — same flat per-cell shape as §4 dead cap, at 30%.
5. **Tag "prior-year salary" base** — RESOLVED: the prior season's **BASE** cell for that player, consistent with the shipped top-5-BASE tag.

---

*Cross-references: `Salary_Ledger_Design.md` (the table + write-paths), `Official_Rulebook.md` (source of truth), `Build_Tracker.md` (what's shipped).*
