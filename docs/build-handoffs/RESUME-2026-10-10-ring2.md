# RESUME: TheWarRoom ring 2, the decision layer (started 2026-10-10)

## What we are doing
Ring 2 of `UI_Target_Roadmap_2026-10.md`: the Lab's analytics ported to Go (War Room › Market,
Trade Floor › Draft, Counterparties, Offers), Franchise HQ › Contracts and cap (what-if, dead cap,
window Acts), the free-agent pool and bid tracking, plus IR/taxi eligibility gating (spec rev 1
§8 Q5). Gate: numbers match the Lab's tests; "How this is counted" on every analytic surface;
spec revision 2.

- Branch `session/ring-2` off main fd6871d, worktree `~/work/TheWarRoom/.worktrees/ring-0` (Beelink).
- Sol (gpt-6.1-sol, medium) builds on Bee; Claude (head brain) briefs, reviews, commits, runs the
  Claude-OS live gates; the expert panel (`~/fleet/bin/panel-dispatch.sh`) rules on durable decisions.
- Run folder `~/fleet/runs/warroom-ring2-2026-10-10/` (`dispatch.sh <phase> <brief>`; transcripts in
  `<phase>/sessions/`, which `/fleet` now finds).

## Pass 1 (p0 survey), 2026-10-10
- Sol's survey: `~/fleet/runs/warroom-ring2-2026-10-10/p0/REPORT.md` (18 KB; full evidence in
  `report-draft.md`). Sol hit its usage limit just before writing it; Claude promoted the staged
  final (`report-final-staged.md`, complete, all 7 sections) to REPORT.md.
- Lab tests 14/14 pass (`p0/lab-tests.log`); `make verify` exit 0 on the branch baseline (`p0/verify.log`).
- What exists in Go today: roster terms, contracts, dead-cap ledger, salary adjustments, a cut and
  buyout preview, IR/taxi Acts (built, not Landed). No Lab port, no bid ledger, no eligibility data.
- What the Lab is: a Python compiler turns the archived facts database into `lab.json`; the JS
  engine (`engine.js`, `draftlab.js`, `teams.js`, about 40 functions) computes from it.
- Findings worth keeping:
  - MFL `injuries.json` gives status (IR, IR-R, IR-PUP, Out…), not an eligibility verdict;
    `players_DETAILS` has no canIR/canDemote. Eligibility must come from verified rules, not guesses.
  - Taxi age rule conflicts (verified by Claude): the captured MFL setting says under 3 years
    (`Ring1_Gap_Closure.md:600–605`); the written rulebook says 3 or fewer (`Official_Rulebook.md:18–22`).
  - DEFECT (not a question; the rule is Christopher-locked 2026-07-04, `Math_Rules_Reference.md:81–87`:
    35% of EACH remaining paid year's own cell, summed, $10k snap): `deadcap.Charge`
    (`internal/transactions/deadcap/deadcap.go:25–36`) charges the CURRENT cap salary × years, so
    any contract whose cells differ by year is wrong. Fix in p2h; verified 10-10 by Claude.
  - Waivers run on ProBoards per the spec; no MFL bid state was found in the archive.
  - `salaryAdjustments` are MFL adjustments, not proven itemised dead cap.

## Proposed build order (Sol's p2a–p2l, condensed; each one Sol run)
| Run | What | Depends on |
|---|---|---|
| p2a | IR/taxi eligibility: eligible / ineligible / unknown with reason, source and time; unknown blocks Ready | Q3 |
| p2b–p2c | Market: going rate, buy/sell lists, asset profile, pick clock | Q1, panel |
| p2d | Draft: timing, plays-like-pick, hit rate, draft-or-buy | p2b |
| p2e | Counterparties | p2b |
| p2f | Offers (owner-scoped history) | p2b |
| p2g | Contracts and cap in HQ, in cents | none |
| p2h | Cut / buyout what-if and plans; fix the dead-cap defect | p2g, Q5 |
| p2i | Free-agent pool and bid plans | p2g, Q2 |
| p2j–p2k | Option / tender / re-sign Acts and their lifecycle in My moves | p2g–p2i, Q4 |
| p2l | Full parity and UI audit, spec revision 2 | all |
If Christopher wants TheWarRoom to own its data (Q1 b), five compiler runs (s1–s5) come before p2b.

## Questions for Christopher (pass 1)
1. The Lab. On 10-06 (T399) you said the Lab is your own research tool and its work never mixes
   into TheWarRoom branches; the roadmap says ring 2 ports it. Which? (a) TheWarRoom rewrites the
   Lab's math in Go and reads a frozen, fingerprinted copy of the Lab's compiled data;
   (b) as (a), then TheWarRoom also gets its own Go pipeline from the MFL archive (5 more runs,
   fully stand-alone). Either way the Lab's code is only a reference for tests and nothing of it is
   merged. Recommendation: (a) now, (b) before the stand-alone app.
2. Bids: track only your own bid and claim plans locally until we have verified how waivers run
   (ProBoards); never show other teams' bids. Recommendation: yes.
3. Taxi age: under 3 years (MFL setting) or 3 or fewer (rulebook)? MFL is what enforces it.
4. Option, tender, re-sign: I capture the MFL owner pages on Claude-OS (no submits) instead of
   asking you to describe them. Recommendation: yes.
5. What-if: keep the numbers you started from until you press Reset, even if MFL data refreshes.
   Recommendation: yes.

## Expert panel
- Run 1 (10-10, seats visionary/mechanic/devil; Sol's seat skipped, out of tokens):
  `~/fleet/runs/panel-panel-ring2-parity-contract-latest/`. Decisions: (1) parity method, JS goldens
  vs Go; (2) the analytic reading's wire shape. Result: pending.
- Still owed: facts schema (after Q1); what-if and plan lifecycle (after Q5).

## Next actions
1. Read panel run 1; triage against each other; record the recommendation here.
2. Put Q1–Q5 to Christopher; record answers as D-numbers.
3. When Sol resets (about 12:20 CDT 10-10): brief p2a (eligibility).

## Notes
- Sol's usage limit: the p0 run hit it at 10:49 CDT; Christopher said it resets in about 1h30.
- The Lab lives only on `session/league-history-pwa` (main checkout); read it, never change it.
