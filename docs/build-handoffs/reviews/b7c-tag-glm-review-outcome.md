# B7c §9 Tag — GLM-5.2 Blind Review Outcome (2026-07-04)

Reviewer: GLM-5.2 on the Beelink (opencode `build` agent, run by Claude over SSH). BLIND,
leads-not-findings; every lead triaged against source. Raw review preserved in git history of
`b7c-tag-glm-review-prompt.txt` run + the session log.

## Triage

| Lead | Severity | Disposition |
|---|---|---|
| **M1** price resolved outside the spanning tx (stale-price TOCTOU) | MAJOR | **Accepted v1 tradeoff.** `wmu` serializes *writes*; the price is a Reader read between two ops — no realistic race in a single-user desktop app. The league-wide top-5 read isn't on `TxWriter`; resolving inside the tx needs that surface. **Carry-forward** (proper fix = league read inside the tx), documented, matches B7a's accepted TOCTOU notes. |
| **M2** per-season counter check-then-bump race | MAJOR | **Refuted.** `OpCount` check + `IncOpCount` both run inside one `WriteTx`, which holds `wmu` end-to-end (`state.go:47,108`). A concurrent `ExecuteTag` blocks on `wmu` until the first commits, then sees `spent=1` and rejects. Same proven posture as restructure. |
| **M3** top-5 pool uses base `Salary` not effective | MAJOR | **Confirmed correct by Christopher** — the pool is base contract salary ("salary" = the contract's face figure; restructure is a cap mechanic, not a salary change). No change. |
| **M4a** restructuring a tagged player is representable | MAJOR | **FIXED** — added a guard: `Restructure` rejects `IsTagged` (a tag is a fixed one-year deal). Christopher-confirmed. Test `TestIntegration_RestructureRejectsTaggedPlayer`. |
| **M4b** tag preserves `IsRestructured` → later cut charges 50% on the tag figure | MAJOR | **FIXED** — `Tag` now resets `IsRestructured=false` (a tag is a fresh contract → a later cut charges the standard 35%). Test `TestIntegration_TagResetsRestructureFlag`. |
| **m1** no early pre-checks before the league scan | MINOR | **Declined.** The scan is cheap and the authoritative guards are in-tx; a pre-check would duplicate the limit logic outside the lock (drift risk) for marginal savings on a rare admin op. |
| **m2** base salary overwritten, prior-year history lost | MINOR | **Accepted** — this is the deliberately-deferred consecutive-year/2nd-tag decision (handoff 30). The transaction log holds the history for when those mechanics land. |
| **m3** `Trade.validate` dup check keys on untrimmed `MFLID` | MINOR | **Carry-forward** — pre-existing B7a code, out of scope for the tag build. Real latent gap; note for a B7a touch-up. |
| **n1** self-inclusion of the tagged player in the pool | NOTE | Accepted — the natural reading of "league-wide"; his pre-tag salary is used (price computed pre-tag). |
| **n2** fewer-than-5 at a position averages the available set | NOTE | Accepted default. |
| **n3** float64 money at the display DTO edge | NOTE | Accepted — documented display-edge (no frontend math), consistent with the whole codebase. |
| **n4** rounding audit | NOTE | Confirmed correct (GLM independently verified half-up). |
| **n5** zero-price guard | NOTE | Confirmed (defense-in-depth). |

## Net
2 real defects fixed (M4a, M4b), both planted-tested. 1 refuted (M2), 1 confirmed-as-built
(M3), 1 accepted-tradeoff with carry-forward (M1). GLM track record on tag: **2**.
