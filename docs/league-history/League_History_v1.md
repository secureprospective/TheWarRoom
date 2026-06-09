# Legacy NFL — League History & Human Behavior Reference
**Version:** 1.0 — June 2026
**Status:** Living document. Add examples as they occur or are surfaced from archives.
**Source:** Manual curation by Christopher Campbell from direct league access.

---

## Purpose

This document preserves the institutional knowledge of how Legacy NFL GMs actually play this game. It is not a rules document — the rulebook covers rules. This is a behavioral document. It teaches future Claude sessions what good, bad, and edge-case human gameplay looks like in practice.

Legacy NFL is one of the oldest active 32-team dynasty leagues in existence. The human patterns documented here represent years of competitive behavior in one of the most complex fantasy formats ever built.

---

## Section 1: Free Agency Bidding Behavior

### Example 1.1 — UFA Bidding War: Najee Harris
**Date:** May 2026
**Result:** WON BY BROWNS

**Sequence:**
| GM | Bid | Points |
|----|-----|--------|
| Falcons (Andy) | 3yr @ $1.0M | 1.4 |
| Lions | 3yr @ $1.5M | 2.1 |
| Browns (Dan) | 1yr @ $2.5M | 2.5 |
| Falcons (Andy) | 3yr @ $2.0M | 2.8 |
| Browns (Dan) | 1yr @ $2.9M | 2.9 |
| Falcons (Andy) | 4yr @ $2.0M | 3.2 |
| Browns (Dan) | 1yr @ $3.3M | 3.3 |

**What this illustrates:**
Two distinct strategic schools colliding. Falcons is trying to win with term — locking Harris long at a cheap annual rate. The 4-year bid at $2.0M is a bet that term makes the bid points competitive without driving up annual cost. Browns counters with single-year bids, keeping cap flexibility, accepting higher annual salary to control the bid point number precisely without committing years.

Three GMs watched the same thread. Lions jumped in early and was outbid. The 24-hour clock reset on every new bid. Browns won by being willing to pay more per year on a short commitment.

**Application insight:** The bid point formula (salary × year multiplier) is the competitive currency — not salary alone, not years alone. GMs are optimizing the formula, not just the dollar amount.

---

## Section 2: RFA Behavior

### Example 2.1 — RFA Match Decision: Andrei Iosivas
**Date:** May 2026
**Result:** RE-SIGNED BY EAGLES

**Setup:** Philadelphia Eagles hold RFA rights on Iosivas (WR-CIN). 4th round tender.

**Sequence:**
| GM | Action | Details |
|----|--------|---------|
| Cardinals (Gremlin) | Offer sheet | 4yr (45%) @ $1.45M + Pick 4.15 |
| Vikings (Logan) | Snipe bid | 3yr @ $2.40 = 3.36pts + Pick 4.31. Declared snipe explicitly. |
| Commissioner (Greg) | Closes thread | "WON BY VIKINGS. EAGLES HAVE 24 TO MATCH OR TAKE COMPENSATION." |
| Panthers (Jesse) acting for Eagles | Match | "Eagles match" |

**What this illustrates:**
The rights-holding team (Eagles) sat completely silent during the bidding war. This is correct behavior — they are not bidding, they are watching. Vikings executed the snipe correctly: explicitly declared it, applied the full-point increment above 2.36pts (went to 3.36pts).

The draft pick is a real strategic variable. Cardinals offered Pick 4.15. Vikings offered Pick 4.31. Those are different values. The winning team's pick conveys only if the rights-holding team lets the player walk. Eagles matched — pick does not convey. Eagles keep the player. Vikings lose both the player and do not give up the pick.

**Application insight:** RFA resolution has two phases — the bidding war and the match window. The commissioner activates the match clock when a bid stands. The match decision is binary: match and keep the player, or decline and collect the pick. Application needs to handle both phases with separate clock management.

---

## Section 3: Trade Behavior

### Example 3.1 — Clean Approval: Rams/Cowboys
**Date:** May 2026
**Result:** APPROVED 3-0

**Assets:**
Rams send: Legette Xavier (WR, 1.5/1.5/1.5/UFA), 2027 R1 (GB), 2027 R2 (DET), 2027 R2 (TB)
Cowboys send: Thomas Jr. Brian (WR, 2.4/2.4/2.4/UFA), Robertson Amik (CB, 1/1/1/UFA), 2026 Pick 5.30

**GM rationale posted:**
Rams GM: "Thomas is a good NFL talent, despite being down year. Here's hoping he bounces back."
Cowboys GM: "Going for the rebuild. I don't think I'm close enough to the top teams to compete. I'm hoping to get enough picks and draft good enough to make the team better."

**DOT vote:** Falcons Approve → Lions Approve → Washington Approve → 3-0 Approved

**What this illustrates:**
The rationale requirement serves a function. Both GMs declared their team direction publicly. Cowboys explicitly stated rebuild mode — this is relevant context for DOT, who can now evaluate whether the trade makes sense for a rebuilding team. The trade was approved cleanly because both sides got something that matched their stated direction.

**Application insight:** GM rationale should be stored and surfaced historically. A GM who has declared rebuild mode multiple times and consistently trades elite assets for picks is building a pattern that DOT and the application can both track.

---

### Example 3.2 — Human Error Void: GB/WAS
**Date:** October 30, 2025
**Result:** VOID (trade deadline missed)

**Assets:**
Packers send: Kaleb Johnson (RB, $3M/2029), Nick Bosa (DE, $4M/2026), James Pearce (DE, $1.42M/2027 RFA)
Washington sends: Micah Parsons (DE, 2.40/7/7/7), Evan Williams (S, $0.50M/2026 RFA)

**What happened:**
Both GMs accepted. Lions GM caught it: "I think you missed the deadline by 7 minutes." Browns confirmed: "Yeah, posted 1 min after the deadline and accepted 6 and a bit minutes after." Washington GM (moderator) self-voided: "Trade was posted after the deadline has expired. Trade is voided."

**What this illustrates:**
A completely legitimate trade — no format errors, no cap violations, no value concerns — was voided because of a 7-minute timestamp miss. This required a human to catch it. The application catches it instantly and blocks submission before the thread is ever posted.

**Application insight:** Trade deadline enforcement is the clearest case for hard automation. Timestamp check is binary. No human should ever need to void a trade for this reason again. The application will not allow a trade to be submitted after the deadline has expired.

---

### Example 3.3 — Value Veto: NYG/WAS
**Date:** September 13, 2025
**Result:** VETOED 1-3

**Assets:**
Washington sends: Drake London (WR-ATL, 3/3/11/11)
Giants send: Eric Ayomanor (WR-TEN, $0.83M/2027 RFA), Dru Tranquill (LB-KC, $3.25M/2025), 2026 2nd (LAR), 2026 2nd (DAL)

**GM rationale posted:**
Washington: "London is a true WR1 and will be a great pickup for my team."
Giants: "Get a young WR to pair with Ward. Also get a starter at LB and some picks."

**DOT deliberation:**
- Falcons: Approve
- Browns: Veto 1-1 (linked comparable trade for reference)
- Colts: Veto 1-2 — "Exchanging high value picks for mid-late picks is definitely detrimental to the Giants. Seems to be in a state of perpetual rebuild."
- Panthers: Veto 1-3 — "Just not good value for Drake London."

**What this illustrates:**
No rule was violated. Three experienced GMs independently judged that a team described as perpetually rebuilding was giving up too much. Drake London at 3/3/11/11 is an elite asset. What came back did not match that value in the DOT's collective assessment.

The phrase "perpetual rebuild" is the key. This is a league health signal. A team that consistently trades elite assets for insufficient return eventually becomes uncompetitive. An uncompetitive team is difficult to hand off to a new owner. Damaged ownership handoffs damage the league.

**Application insight:** The trade analyzer surfaces value data to support DOT. It cannot and should not make this call. Human judgment about league competitive health is irreplaceable. The application's job is to make sure DOT has the best possible information — comparable trade history, asset valuations, each team's roster trajectory — so their judgment is fully informed.

---

## Section 4: Waiver Behavior

### Example 4.1 — Single-Team Claim: Evan Hull
**Date:** October 29, 2024
**Result:** CLAIMED BY FALCONS

**Waiver post (Panthers GM Jesse):**
"24-25: $0.67M 26: RFA. Cap hit if unclaimed: 24-25: $0.23M"

**Claim (Falcons GM Andy):** "Claim"

**What this illustrates:**
The waiver format is minimal by design. Releasing team posts contract and dead cap math — that is the required format. A claim is a single word. The dead cap calculation was done manually by the releasing GM: $0.67M × 35% = $0.2345M → $0.23M.

This math is done manually by every GM every time they waive a player.

**Application insight:** Dead cap calculation is automatic in the application. Releasing GM initiates the waiver, the application calculates and displays the dead cap hit instantly. No manual math. No rounding errors.

---

## Section 5: Trade Block Behavior

### Example 5.1 — Vikings Trade Block
**Date:** April 2026 (updated May 2026)
**GM:** Vikings (Logan)

**Posted availability:**
- QB Kirk Cousins ($2.65, restructured to $4.65/2027, UFA)
- DT Dexter Lawrence ($4/2028)
- DE Sam Williams ($1.63/2027)
- DE Myles Garrett (franchised twice at $9.58, extension eligible — "will listen")

**Stated needs:** WR, TE, LB, CB, S. Always interested in picks.

**GM philosophy stated publicly:** "I will always listen to any offer for any player/pick. However, some are definitely more attainable than others."

**What this illustrates:**
The trade block is a market-making tool. GMs use it to attract offers without committing to a specific deal. Logan is a 2025 NFC Champion in what appears to be a retool cycle. Myles Garrett franchised twice at $9.58 and listed as available is a significant signal — that is a massive cap number being shopped openly.

The public GM philosophy statement is part of league culture. GMs who state they will always listen are signaling openness. This creates market activity.

**Application insight:** Trade block module should surface listed players with current contract data auto-populated, stated needs, and allow GMs to browse opposing trade blocks filtered by position need. This turns a passive forum thread into an active matchmaking interface.

---

## Section 6: League Health Observations

### The Perpetual Rebuild Problem
When a GM consistently trades elite assets for insufficient return, the team eventually becomes uncompetitive. Finding a new engaged owner for an uncompetitive franchise is harder than finding one for a team with assets. League health depends on competitive parity across all 32 teams.

The NYG/WAS veto was at least partially motivated by this concern. DOT has an implicit responsibility to protect the league from trades that damage long-term franchise viability — even when both GMs agree to the trade.

### The 32-Owner Commitment Problem
Finding 32 like-minded people who will run individual teams with clear effort and sustained goals is genuinely difficult. When an owner quits, the franchise they leave behind reflects the decisions they made. An engaged league with tools that make ownership easier and more rewarding is more likely to retain owners and attract quality replacements.

This is part of the application's value proposition that goes beyond analytics. A better interface, automated rule enforcement, and intelligence tools reduce the friction of ownership. Lower friction means higher retention.

### Commissioner and DOT Load
The screenshots throughout this document show commissioners and DOT members manually catching timestamp errors, calculating dead cap, enforcing bid increments, and tracking vote counts. Every one of those tasks is automatable. The application does not replace commissioner judgment — it removes the mechanical tasks that consume their time so they can focus on league governance.
