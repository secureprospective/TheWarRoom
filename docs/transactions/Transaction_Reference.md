# Legacy NFL — Transaction System Reference
**Version:** 1.0 — June 2026
**Status:** Rules + human behavior documentation. Application build is a separate session.

---

## Overview

This document covers every transaction type in the Legacy NFL. For each type it documents:
- The rulebook mechanics
- How humans currently execute it on Proboards
- What the application validates, automates, or improves

The application does not replace human judgment. It removes mechanical friction so human judgment can focus on things only humans can evaluate.

---

## 1. UFA FREE AGENT BIDDING

### Rulebook Mechanics
- GM posts player name, position, NFL team, and bid (salary × year multiplier = bid points)
- Any GM may place a competing bid at any time
- A bid must stand unchallenged for 24 hours to win
- Bids cannot be rescinded once placed
- Each new bid must exceed previous by at least 0.1 points
- Bid totals posted to three decimal places
- Sniping rule: any bid after the 20-hour mark must be declared a "Snipe" and exceed previous bid by a full point

### Bid Point Formula
```
Bid Points = Annual Salary (millions) × Year Multiplier
Year multipliers: 1yr=1.00x | 2yr=1.20x | 3yr=1.40x | 4yr=1.60x
```

### Contract Constraints
- Max contract: 4 years
- Max 1-year bid: $12M (lifted during playoffs)
- $12M–$24M requires minimum 2 years
- Over $24M requires minimum 3 years
- Salary must meet or exceed position minimums

### Invalid Bid Conditions (Application Auto-Rejects)
- Wrong year count for salary range
- Bid point total calculated incorrectly
- Over cap for bidding team
- Over roster limit for bidding team
- Increment less than 0.1 points
- Snipe bid increment less than 1.0 point
- RFA tag not explicitly included when required

### Human Behavior Observed
Falcons GM opens at 3yr/$1M = 1.4pts on Najee Harris. Lions jumps in at 3yr/$1.5M = 2.1pts. Browns counters with 1yr/$2.5 = 2.5pts. Strategic pattern: Falcons tries to win with term (cheap multi-year lock). Browns counters with single-year bids, keeping cap flexibility, driving up the bid point cost. Three GMs watching the same thread simultaneously. Clock resets on every new bid. Sniping rule at 20-hour mark forces decisions.

The bid point formula — not just salary or years alone — is the competitive currency.

### Application Role
- Validates bid math automatically before posting
- Tracks 24-hour clock per bid
- Detects snipe window (20-hour mark) and enforces full-point increment
- Validates cap space for bidding team in real time
- Validates roster space for bidding team
- Flags invalid bids before they are ever submitted
- Displays all active bid threads league-wide in one interface

---

## 2. RFA BIDDING

### Rulebook Mechanics
- RFA player is tendered by rights-holding team at one of six price levels
- Other teams have a 7-day window to submit an offer sheet
- Bidding follows standard UFA rules plus the submitting team lists the draft pick they plan to provide
- Minimum initial bid must exceed the re-signing team's baseline price for the same number of years
- When a winning bid stands 24 hours, the RFA rights team may: match, exceed, or let walk and collect pick compensation
- Player must be in 3rd year or earlier on last year of contract
- "RFA" must be explicitly included in the contract offer or the bid is invalid

### Baseline Re-Sign Prices (No Competing Offer)
```
1 Year: Tender price + 0%
2 Years: Tender price + 15%
3 Years: Tender price + 30%
4 Years: Tender price + 45%
```

### Human Behavior Observed
Andrei Iosivas (WR) — Eagles hold RFA rights. Cardinals submit 4yr (45%) @ $1.45M + Pick 4.15. Vikings snipe at the 20-hour mark: 3yr @ $2.40 = 3.36pts + Pick 4.31, explicitly declaring "Snipe bid, so full point above previous bid." Commissioner closes thread: Eagles have 24 hours to match or take compensation. Eagles match. Player stays. Pick does not convey.

Key insight: The draft pick is part of the bid itself, not an afterthought. Pick slot (4.15 vs 4.31) is a real strategic variable. The rights-holding team sits silent during the bidding war, then activates only at the match decision point.

### Application Role
- Validates RFA tag is present in bid
- Validates pick offered matches available picks for bidding team
- Calculates and displays baseline re-sign price automatically
- Starts match window clock when winning bid closes
- Notifies rights-holding team of match decision deadline
- Tracks pick conveyance — only triggers if player walks

---

## 3. WAIVER CLAIMS

### Rulebook Mechanics
- Releasing team posts: player name, contract details, and dead cap calculation
- Dead cap formula: 35% × annual salary × remaining contract years
- Any team replies "claim" to claim the player
- If multiple teams claim, winner determined by waiver order after 24 hours
- Releasing team cannot re-sign waived player for the rest of that season
- Dead cap obligation ends if player is claimed

### Human Behavior Observed
Panthers GM posts Evan Hull RB waiver: "24-25: $0.67M 26: RFA. Cap hit if unclaimed: 24-25: $0.23M"
Falcons GM replies: "Claim."
Single word. Clean. The dead cap math is already calculated and posted by the releasing GM.

GMs are manually calculating dead cap every time: $0.67M × 35% = $0.2345M → $0.23M.

### Application Role
- Auto-calculates dead cap and displays it when a waiver is initiated
- Validates releasing team has cap room to absorb dead cap
- Tracks 24-hour claim window
- Handles multi-claim waiver order resolution automatically
- Prevents releasing team from re-signing waived player same season

---

## 4. TRADES

### Rulebook Mechanics
- Trade deadline: Week 9
- Both GMs post and accept on Proboards
- DOT reviews: requires 3 approves or 3 vetoes to finalize
- Once posted and accepted, cannot be withdrawn — binding commitment
- Picks tradeable up to two years in advance only
- Compensatory picks cannot be traded
- Salary cap trading (one team covering another's cap hit) abolished in 2017

### Required Format
- All players with full contracts
- Year of any draft picks included
- Cap space being exchanged by each team
- At least one sentence from each GM explaining rationale

### Invalid Trade Conditions (Application Auto-Flags)
- Submitted after Week 9 trade deadline (timestamp check — hard block)
- Missing cap space figures
- Missing GM rationale
- Picks beyond two-year trading window
- Compensatory picks included
- Transaction puts either team over cap
- Transaction puts either team over/under roster limit

### Human Behavior Observed — Clean Approval
Rams/Cowboys trade: Multiple players and 2027 picks from three different teams. Both GMs post rationale. Falcons approves. Lions approves. Washington approves. Commissioner closes 3-0. Clean.

### Human Behavior Observed — Human Error Void (GB/WAS)
Both GMs accept a trade. Lions GM catches it: "I think you missed the deadline by 7 minutes." Browns confirms: "Posted 1 min after the deadline, accepted 6 and a bit minutes after." Washington GM (moderator) self-polices: "Trade was posted after the deadline has expired. Trade is voided."

This is a timestamp error. Binary. The application catches it before the thread is ever posted. No human needs to catch a 7-minute miss.

### Human Behavior Observed — Value Veto (NYG/WAS)
Washington sends Drake London (WR, 3/3/11/11). Giants send Ayomanor (WR, $0.83M RFA), Tranquill (LB, $3.25M), 2026 2nd (LAR), 2026 2nd (DAL). Both GMs accept. Falcons approves. Browns vetoes (1-1). Colts vetoes (1-2): "Exchanging high value picks for mid-late picks is detrimental to the Giants. Seems to be in a state of perpetual rebuild." Panthers vetoes (1-3): "Just not good value for Drake London." Trade voided.

This veto is not catchable by any algorithm. No cap rule was violated. No format rule was violated. Three experienced GMs independently judged that the Giants — described as perpetually rebuilding — gave up too much. League integrity is a human responsibility. The application supports DOT by surfacing value metrics, but the judgment call belongs to humans.

### Application Role
- Hard block on trades submitted after Week 9 deadline (timestamp enforcement)
- Validates required trade format before submission
- Flags missing cap figures, missing rationale, invalid picks
- Validates cap compliance for both teams post-trade
- Validates roster limits for both teams post-trade
- Surfaces valuation data to help DOT assess value — not to replace their judgment
- Tracks DOT vote count and closes thread at 3-0 in either direction

---

## 5. TRADE BLOCK

### Human Behavior Observed
Vikings GM (Logan) posts a public trade block: lists players he is most open to moving with current contracts, notes what positions he wants to upgrade, states his GM philosophy ("I will always listen to any offer for any player/pick"). Myles Garrett — franchised twice at $9.58, extension eligible — listed as available. This is a market signal, not a commitment.

### Application Role
- Display active trade blocks league-wide in a dedicated view
- Allow GMs to update their trade block with current contract data auto-populated from MFL
- Surface opposing GMs' listed needs vs. available assets for matchmaking suggestions

---

## 6. FRANCHISE TAG

### Rulebook Mechanics
- One tag per team per year
- Tag price = average salary of top 5 players at that position league-wide
- If tag price < player's previous year salary: tag = 120% of previous year
- Same player max two consecutive tags. Second tag = 120% of first tag value
- Tagged player may be extended at tag price for tag year + 120% per additional extension year
- Playoff free agent contracts do not count toward franchise tag calculations

### Application Role
- Auto-calculates tag price from MFL salary data across all 32 teams at time of tagging
- Validates one-tag-per-team limit
- Tracks consecutive tag years per player
- Flags if second tag is being applied and calculates 120% premium

---

## 7. CONTRACT EXTENSION

### Rulebook Mechanics
- One extension per GM per season
- Player must have at least one year remaining
- Adds up to 3 years. No player may exceed 6 total contract years
- Extension years at 150% of player's highest-paid remaining year
- Subject to position floors (see rulebook)
- Cannot be extended a second time based on prior extension — must reach FA first

### Application Role
- Validates one-extension-per-GM limit
- Calculates extension pricing automatically (150% of highest remaining year vs. position floor)
- Validates total contract years do not exceed 6
- Tracks extension history per player to enforce re-extension restriction

---

## 8. CONTRACT RESTRUCTURE

### Rulebook Mechanics
- One restructure per team per year
- Once per contract (unless extensions add unlock slots)
- Max money moved depends on year salary ($1M/$2M/$3M thresholds)
- Restructured contracts carry 50% dead cap penalty if player later waived

### Application Role
- Validates one-restructure-per-team limit
- Calculates maximum allowable movement per year
- Flags 50% dead cap penalty on any subsequent waiver of restructured player
- Tracks extension-based restructure unlocks

---

## 9. CONTRACT BUYOUT

### Rulebook Mechanics
- Two buyouts per team per season, offseason only
- Buyout cost based on average remaining salary × rate (60%/75%/90% for 2/3/4 years remaining)
- Cannot bid on bought-out player until following offseason

### Application Role
- Validates offseason-only restriction
- Calculates buyout cost automatically
- Tracks two-buyout-per-season limit
- Flags re-signing restriction for bought-out players current offseason
