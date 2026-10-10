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

## Decisions (Christopher, 2026-10-10)
- **D1 The Lab:** TheWarRoom rewrites the Lab's math in Go now, reading a frozen, fingerprinted copy
  of the Lab's compiled data; before the stand-alone app it gets its own pipeline from the MFL
  archive (compiler runs s1–s5). No Lab code is merged; the Lab is only the reference the tests check.
- **D2 Taxi age:** check MFL's rule, under 3 years of experience. The rulebook's "3 or fewer" is a
  rulebook mismatch for Christopher to fix there.
- **D3 Bids:** "We just need the framework, eventually ProBoards and MFL will be eliminated." Build
  the bid and claim framework as TheWarRoom's own system, designed to become the authority in the
  stand-alone app; for now it holds plans and honest status, nothing invented about other teams.
- Defaults taken (Q4, Q5, his to overturn): Claude captures the option/tender/re-sign MFL pages on
  Claude-OS, no submits; a what-if keeps its starting numbers until Reset.

## Expert panel
### Run 1 (10-10): parity method and the wire shape. Seats: Kimi (visionary, rerun at medium after a
22-min stall), GLM (mechanic), Nemotron (devil); Sol's seat skipped (out of tokens).
Runs: `~/fleet/runs/panel-panel-ring2-parity-contract-20261010T155210Z-304800/` (GLM, Nemotron) and
`...-20261010T161439Z-556587/` (Kimi). Quotes: GLM 16/17 ok, Kimi 5/5, Nemotron unparsed (format;
Claude spot-checked its `gauss` quote, real).

**Claude's triage (recommendation; durable, so Christopher may overrule):**
- P1 Parity: adopt JS golden fixtures (all three agree). Generated offline by a pinned Node command
  from the frozen `lab.json`; goldens stamped with BOTH the lab.json hash and the engine.js hash
  (GLM F3, Kimi F4); Node is never a runtime dependency.
- P2 Port the JS algorithms faithfully: same Gauss elimination with partial pivoting, same
  summation order. Rejected: Nemotron's gonum QR (would diverge from the reference by design).
- P3 Tolerance: exact for IDs, counts, enums, nulls, membership, cents; floats within
  1e-9 + 1e-9×|want| (GLM; enum margins measured: nearest non-identity 1.6e-3). Record the max
  observed error per function; Kimi's bit-exact target is the aim, not the gate. Rejected:
  Nemotron's looser 1e-7/1e-6 tiers.
- P4 The generator fails on any non-identity decision margin under 1e-6 and names the row (GLM F2).
- P5 Required mutants: a threshold flip (8→7 delivered records; 80→81 trades) must turn a golden
  cell value↔null (GLM F5); a seeded Go bug (one ridge term dropped) must fail parity (Kimi F3).
- P6 The Lab's "measured 2026-10-06" comments become coded assertions in TheWarRoom's own parity
  tests (Nemotron F5), never edits to the Lab (D1).
- P7 Goldens also carry the compiler's prices so Go re-checks the compiler↔JS anchor (GLM F4,
  Nemotron F2) at the JS suite's own 0.01.
- W1 Wire shape: one generic shell extending the existing `Sourced[T]`/Provenance seam, with a
  typed payload per surface (GLM F6; Kimi and Nemotron converge on shell + typed body). Shell
  carries status (ok / thin sample / failed / not wired), absence reason, count and the threshold
  it was judged against, baseline, cutoff (`archiveThrough`), method, provenance (input hash,
  build stamp), and "How this is counted" text written in Go. React does no math.
- Open for Christopher (Q6): the Lab's spread tiers (solid / rough / thin). Nemotron and Kimi:
  a confidence score, banned (CLAUDE.md:86). GLM: a plain grading of the shown range, allowed.
  Claude's recommendation: show the range itself, keep the tier internal.

- Still owed: facts schema (before compiler runs s1–s5); what-if and plan lifecycle (before p2h).

## Next actions
1. (done: panel run 1 triaged above)
2. (done: D1–D3)
3. p2a (eligibility) brief written: `~/fleet/briefs/warroom-ring2-p2a-eligibility.md`. Answer key
   captured by Claude on Claude-OS 11:27 CDT: `~/fleet/runs/warroom-ring2-2026-10-10/mfl-O18.html`
   (IR page, 55 rows, 38 "Can not be deactivated") and `mfl-O98.html` (taxi, 69 rows, 28 "Cannot be
   demoted"); owner-only pages, never in git. Defaults taken: unknown eligibility leaves the Act
   available with "Not checked; MFL will decide"; ineligible disables it with the reason.
   Dispatch when Sol resets (about 12:20 CDT).
4. Ask Christopher Q6 (spread tiers).

## Notes
- Sol's usage limit: the p0 run hit it at 10:49 CDT; Christopher said it resets in about 1h30.
- The Lab lives only on `session/league-history-pwa` (main checkout); read it, never change it.
