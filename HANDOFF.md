# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-6 stopped, partial item 1.
Local commits only; no remote contact or push authorized.

## Where it stands
- League held player-record position now wins over a conflicting NFL-feed position for scoring and
  lineup legality. Explicit commissioner position choices still win. No scoring math/settings changed.
- Regression observed failing first (feed LB over league DE); final targeted race 14 named PASS,
  zero failures. Plain make build PASS. Existing stat-queue tests pass but do not prove the requested
  league-position suggestion behavior.
- Evidence and precise blocker: `docs/build-handoffs/APP-SCHED-6-evidence.md`.
- APP-SCHED-5 copied-profile comparison remains previous-build evidence only: 148 recorded weeks,
  4 exact / 0 within .01 / 33 differ / 111 unavailable. Not rerun on this change.

## Next move / blocked on
- Existing stat queue approves a weekly revision, not a durable player position; newest==counted
  suppression hides feed-vs-league disagreement. Integrate player-level position suggestions with
  existing PlayerPosition ledger, preserving preview, atomic append checks, undo and rejection.
  Deduplicate across weeks/unchanged feed. Stopped per brief's balloon rule at this boundary.
- Then rerun 148 comparison on a fresh scratch-profile copy, isolate Schultz W4 stat/rule residual,
  classify 472 absent starter-stat occurrences with participation/status evidence and wire legitimate
  inactive zeroes without masking real gaps. These items are not delivered.
- Earlier debt: verified custom identity 0835 (seven occurrences); slow initial ledger adoption and
  native reads; live ESPN/Sleeper/advanced-rushing inputs; source-health UI; historical roster refusals.

## Tried and rejected
- Treating a weekly StatApprove as league-wide position approval: conflates unrelated stat choices
  and cannot govern future weeks from the current evidence contract.
- One position suggestion per player-week: repeats unchanged feed changes.
- No source-origin branches, guessed aliases, fabricated inactivity/zeroes or settings tweaks.
