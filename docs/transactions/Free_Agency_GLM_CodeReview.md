# Free Agency v1 — GLM-5.2 Blind Code Review Outcome

**Reviewer:** GLM-5.2 (Beelink, over SSH, `opencode run -m zai/glm-5.2`), BLIND on the diff.
**Date:** 2026-07-11. Leads triaged vs source below. Branch `session/free-agency-pool`.

GLM produced 9 leads (L1–L9). **No live defect in shipped logic** — the two S/A-tier "must verify"
items were a misread (L1) and a real-but-guarded gap (L2); three genuine hardening fixes applied
(L2, L4, L7); the rest triaged as consistent-with-existing-idiom or documented-benign.

## Applied

| Lead | Severity | Action |
|------|----------|--------|
| **L2** | A | **APPLIED** — SIGN never validated the target franchise. Added `requireKnownFranchise` in `SignContract` (rejects a franchise with no roster this season, the "known franchise" proxy — state has no separate registry). Test `TestIntegration_SignRejectsUnknownFranchise`. Tradeoff documented (a fully-gutted franchise can't sign until it holds a player again — vanishingly rare). |
| **L4** | B | **APPLIED** — a sub-$5k salary is `> 0` at `validate()` but rounds to $0, and the §6 floor that would catch it is skipped in v1. Added a post-round `salary <= 0` guard in `freeagency.Sign`. Test `TestIntegration_SignRejectsSubGridSalary`. |
| **L7** | C (test teeth) | **APPLIED** — the buyout-lockout test only advanced phases without a sign attempt, so a regression lifting the lockout at REGULAR_SEASON would pass. Added an in-window (REGULAR_SEASON, season N) sign attempt asserting the lockout still holds. |

## Triaged OUT (with reason)

- **L1 (S, "contracts DELETE binds 3 placeholders with 1 arg")** — **REFUTED.** GLM misread the
  diff; `ReleasePlayer`'s contracts delete uses `w.s.execPlayer(ctx, w.tx, query, mflID)`, which
  appends `(leagueID, season, mflID)` — all three placeholders bound. Confirmed at `writes.go:234`.
  The passing waive/buyout/retire tests corroborate.
- **L3 (A, change-log PK collision on re-sign into an overlapping year)** — the shared
  `logCellChange` id is `cyc:<league>:<mfl>:<year>:<UnixNano>`; two sequential same-year calls
  (clear old → lay new) get distinct nanos on Linux. This is the SAME id scheme
  `writeCell`/`voidCell`/`insertExtensionCell` already ship. The overlap path is exercised by
  `TestIntegration_BuyoutLockoutBlocksSign` (re-sign in 2027 over a VOID 2027 cell) and
  `TestIntegration_RolloverPromotesExpiredContractToPool` (re-sign over a UFA 2027 cell) — both
  green under `-race`. Consistent-with-idiom + covered.
- **L5 (B, `contract_status = UFA` on a signed active contract)** — the pool derives from
  `player_status_events`, and cap/eligibility from cells; nothing reads `contract_status == UFA` as
  "signable/expiring." Matches the seed's own convention. Vestigial.
- **L6 (B, promoted players leave orphan `contract_years` cells)** — `CapUsed` is roster-joined
  (`loadCellCap` JOINs `rosters` on the current season), so orphan cells of an off-roster player
  contribute $0; `LedgerCells` is per-id only. No global sum-without-join consumer exists. Same
  posture as a §8 cut leaving VOID cells. Benign; a re-sign clears them.
- **L8 (C, trigger test matches error substring)** — identical to the shipped
  `TestCapReliefAppendOnlyTriggers` / dead-cap trigger tests; matches the house convention.
- **L9 confirms** — all 5 `ReleasePlayer` callers updated (compiler-enforced, build green);
  `MinSalaryFloor` figures transcribed from Official_Rulebook §6 and pinned by `TestMinSalaryFloor`;
  `execPlayer` binds `w.s.leagueID/season` (confirmed) — GLM independently judged the RolloverSeason
  promotion ordering **sound** ("found no wrongly-promoted or wrongly-kept case").

## Gate after fixes
lint 0 / `go test -race ./...` green / tsc+vite clean / depguard proven (planted `freeagency`
import fired the deny). Functional gate on the Beelink pending.
