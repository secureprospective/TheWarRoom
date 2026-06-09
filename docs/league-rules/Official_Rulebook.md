# Legacy NFL — Official Rules Reference
**Version:** 2026
**Status:** Source of Truth. All scoring engine values, contract mechanics, and transaction rules derive from this document.

---

## 1. SALARY CAP

- Each team: $125M cap for 2026
- Teams must stay under cap at all times. No exceptions.
- All salaries rounded to nearest $10,000
- Example: $10,987,000 becomes $10.99M

---

## 2. ROSTERS

**Active Roster:** Minimum 35, maximum 48 (in-season). IR and Practice Squad do not count toward the 48 limit but count toward the cap.

**Practice Squad:** Maximum 8 players. Must have 3 or fewer years of NFL experience (tracked by career games played on NFL.com). Once a player enters his 4th NFL season, he must be promoted or waived.

**IR:** Any player placed on IR or PUP by their NFL team is eligible for Legacy IR. IR players can be traded. Must be declared on Proboards.

---

## 3. STARTING LINEUP (21 Starters)

**Offense (9):**
1 QB, 1 RB, 2 WR, 1 TE, 1 RB/WR Flex, 1 TE/WR Flex, 1 Offensive Flex (RB/WR/TE), 1 Kicker

**Defense (12):**
2 DT, 2 DE, 2 LB, 2 CB, 2 S, 2 Defensive Flex (DT/DE/LB/CB/S)

**Lineup Penalties:**
- Failure to set: removal from league
- Invalid lineup: zero applied to highest-scoring starter that week

---

## 4. SCORING

### Passing
| Stat | Points |
|------|--------|
| Passing TD | 5 |
| Passing Yards | 0.05/yard |
| Interception Thrown | -2 |
| Long Pass (40+ yards) | +1 |
| Passing 2-Point Conversion | 2 |

### Rushing
| Stat | Points |
|------|--------|
| Rushing TD | 6 |
| Rushing Yards | 0.1/yard |
| Rushing Attempt | 0.15 |
| Long Rush (20+ yards) | +1 |
| Long Rush (40+ yards) | +1 additional |
| Rushing 2-Point Conversion | 2 |

### Receiving
| Stat | Points |
|------|--------|
| Receiving TD | 6 |
| Receiving Yards | 0.1/yard |
| Reception | 1 |
| Long Reception (20+ yards) | +1 |
| Long Reception (40+ yards) | +1 additional |
| Receiving 2-Point Conversion | 2 |

### Kicking
| Stat | Points |
|------|--------|
| FG Made 0-39 yards | 3 |
| FG Made 40-49 yards | 4 |
| FG Made 50-59 yards | 5 |
| FG Made 60-69 yards | 6 |
| Missed FG | -3 / -1 |

### Returns
| Stat | Points |
|------|--------|
| Punt Return TD | 6 |
| Punt Return Yards | 0.025/yard |
| Kickoff Return TD | 10 |
| Kickoff Return Yards | 0.025/yard |

### Fumbles
| Stat | Points |
|------|--------|
| Fumble Lost | -2 |
| Own Fumble Recovery | 2 |
| Opponent Fumble Recovery | 3 |
| Opponent Fumble Recovery Yards | 0.025/yard |
| Forced Fumble | 4 |

### Defense
| Stat | Points |
|------|--------|
| Interception Return TD | 6 |
| Interception Caught | 5 |
| Interception Return Yards | 0.025/yard |
| Pass Defensed | 2.5 |
| QB Hit | 1 |
| Blocked Field Goal | 7 |
| Blocked Punt | 7 |
| Blocked Extra Point | 7 |
| Tackle (LB/CB/S/QB) | 1.5 |
| Tackle (DT/DE only) | 2.5 |
| Assist (all except DT/DE) | 1 |
| Assist (DT/DE only) | 1.5 |
| Sack | 4.5 |
| Tackle for Loss | 2.5 |
| Safety | 10 |

**Note:** Tackle for Loss is a direct stat (2.5 pts). Do not approximate via solo tackle proxies.
**Note:** DT/DE receive higher tackle/assist values — this is the True Position split. It is not a bonus; it is the base rate for those positions.

---

## 5. SEASON STRUCTURE

- Regular season ends Week 13
- 14 playoff teams: each division winner + 2 wild cards per conference
- #1 seed each conference earns first-round bye
- Playoff home field advantage: +3 points to home team

---

## 6. FREE AGENCY

### Minimum Salaries
| Experience | Minimum |
|------------|---------|
| 0 years (Rookie) | $330,000 |
| 1 year | $380,000 |
| 2 years | $430,000 |
| 3 years | $480,000 |
| 4-6 years | $530,000 |
| 7-9 years | $580,000 |
| 10+ years | $630,000 |

### Bid Point Formula
`Annual Salary (in millions) × Year Multiplier = Bid Points`

| Years | Multiplier |
|-------|-----------|
| 1 | 1.00x |
| 2 | 1.20x |
| 3 | 1.40x |
| 4 | 1.60x |

**Examples:** 1YR/$5M = 5.000 pts | 2YR/$5M = 6.000 pts | 3YR/$1.3M = 1.820 pts

### Contract Rules
- Maximum contract length: 4 years
- Flat annual salaries — no backloading or frontloading
- Maximum 30 active bids at any time
- Maximum 1-year bid: $12M
- $12M–$24M bids require minimum 2 years
- Over $24M bids require minimum 3 years
- Bid totals posted to three decimal places
- Each new bid must exceed previous by at least 0.1 points

### Sniping Rule
Any bid after the 20-hour mark must be declared a "Snipe" and must exceed previous bid by a full point. Bids not meeting this are invalid.

### Season Bidding Rules
- Multi-year bids close end of Week 13
- After Week 12: playoff teams may place 1-year bids only. Non-playoff teams may not bid.
- During playoffs: $12M cap on 1-year bids is lifted

---

## 7. RESTRICTED FREE AGENCY (RFA)

### Tender Prices
| Tender | Compensation |
|--------|-------------|
| $600K | 5th round pick |
| $1.0M | 4th round pick |
| $1.5M | 3rd round pick |
| $2.5M | 2nd round pick |
| $5.0M | 1st round pick |
| $6.5M | 1st and 2nd round pick |

Tender rises to 110% of previous year's salary if that exceeds the standard price.

### Baseline Re-Sign Prices (No Competing Offer)
| Years | Price |
|-------|-------|
| 1 Year | Tender price + 0% |
| 2 Years | Tender price + 15% |
| 3 Years | Tender price + 30% |
| 4 Years | Tender price + 45% |

When a winning offer stands 24 hours, the RFA rights team may: match, exceed, or let walk and collect pick compensation.

---

## 8. RELEASING PLAYERS (WAIVERS)

- Post player name, contract, cap hit on Proboards
- If unclaimed: releasing team owes 35% of salary per remaining contract year (dead cap)
- If claimed: dead cap obligation ends
- A team that waives a player cannot re-sign that player for the rest of that season
- Dead cap formula: 35% × annual salary × number of remaining years

---

## 9. FRANCHISE TAG

- One tag per team per year (optional)
- Tag price = average salary of top 5 players at that position league-wide
- If tag price < player's previous year salary: tag set at 120% of previous year
- Same player may be tagged maximum two consecutive years. Second tag = 120% of first tag
- Tagged player may be extended. Additional extension years cost 120% of tag price per year
- Free agent contracts signed during playoffs do not count toward franchise tag calculations

---

## 10. CONTRACT EXTENSION

- One extension per GM per season
- Player must have at least one year remaining. UFAs not eligible
- Adds up to 3 years. No player may exceed 6 total contract years
- Extension years priced at 150% of player's highest-paid remaining year
- Subject to position floors (whichever is greater):

| Position | Floor |
|----------|-------|
| QB | $15M/year |
| WR | $10M/year |
| RB, TE, LB | $8M/year |
| DE | $7M/year |
| S | $5M/year |
| DT | $4M/year |
| CB, K | $3M/year |

- A player cannot be extended a second time based on a prior extension. Must reach free agency first.

---

## 11. CONTRACT RESTRUCTURE

- One restructure per team per year
- A contract may only be restructured once per contract
- Exception: each extension unlocks one additional restructure
- Restructured contracts carry 50% dead cap penalty if player later waived

### Restructure Limits
| Contract Year Salary | Max Move |
|---------------------|----------|
| $3M or more | $1M |
| $6M or more | $2M |
| $12M or more | $3M |

---

## 12. CONTRACT BUYOUT

- Two buyouts per team per season, offseason only
- Cannot bid on bought-out player until following offseason

### Buyout Rates
| Years Remaining | Rate |
|----------------|------|
| 2 years | 60% of average remaining salary |
| 3 years | 75% of average remaining salary |
| 4 years | 90% of average remaining salary |

---

## 13. SPECIAL SITUATIONS

- **Cap Relief Appeal:** Commissioner may reduce cap hit for career-ending injury, recurring injury, or behavioral suspension
- **Gaines Adams Rule:** Player death — removed from roster with no cap penalty
- **Retirement:** Team responsible for 30% of remaining contract per year left

---

## 14. TRADES

- Trade deadline: Week 9
- Requires 3 approves or 3 vetoes from DOT to finalize
- Salary cap trading abolished in 2017
- Picks may be traded up to two years in advance only
- Compensatory picks cannot be traded and cannot be used as RFA compensation
- Once posted and accepted on Proboards, trade cannot be withdrawn

### Required Trade Format
- All players with full contracts
- Year of any draft picks included
- Cap space being exchanged by each team
- At least one sentence from each GM explaining rationale

---

## 15. ROOKIE DRAFT

### Fixed Slot Pricing

**Round 1 — 5-year contracts**
| Picks | Salary |
|-------|--------|
| 1-10 | $3.0M |
| 11-15 | $2.4M |
| 16-20 | $2.0M |
| 21-32 | $1.5M |

**Round 2 — 3-year contracts + RFA**
| Picks | Salary |
|-------|--------|
| 1-12 | $1.42M |
| 13-24 | $1.25M |
| 25-32 | $1.08M |

**Round 3 — 3-year contracts + RFA**
| Picks | Salary |
|-------|--------|
| 1-16 | $0.83M |
| 17-32 | $0.67M |

**Round 4** — 3-year + RFA — $0.60M
**Round 5** — 3-year + RFA — $0.50M

- Maximum 15 rookie selections per team per draft
- Picks beyond 15 not traded before clock runs are forfeited without compensation
