# APP-SCHED-6 — stopped at the position-decision boundary

## Delivered slice (item 1 partial)

Held league player-record position → scoringPosition → NativePlayerScore.Position →
FranchiseWeekScore.judge → existing lineup judgement. An existing explicit commissioner position
choice still wins. With no held league position, the existing NFL-feed fallback remains. An ambiguous
held league position requires a ruling rather than silently using a differing specific feed position.
No scoring arithmetic, settings, source-origin branches, feed evidence or player records changed.

Regression first failed with feed LB replacing league DE. Final targeted race batch:
`go test -race -json ./internal/commissioner -run 'Test(NativePlayerScore|NativeScore|LeaguePosition|StatQueue)' -count=1`:
**14 named PASS, 0 FAIL**. Tests observe DE scoring, DE lineup legality, preserved raw evidence,
ambiguous-position refusal and explicit commissioner override. Existing revision-queue tests pass;
they do **not** establish the new league-position suggestion/approval contract.
**Plain `make build` PASS.** Logs are local under `.tmp/sched6/`.

## Why stopped

The existing position-change queue test is a *weekly feed revision* comparison, not a durable
league-player-position decision. `queueBuilder.candidate` deliberately suppresses items when newest
and counted digests match or there is no explicit weekly baseline. Automatic feed adoption makes
that the normal state. `StatApprove` persists only the weekly revision ID; it neither records a
player position nor invokes `PlayerPosition`. `scoreChangesFrom` reads durable league-wide position
choices only from `PlayerPosition` / `PlayerPositionClear` records. Simply preferring the approved
week's feed position would not update future weeks and would silently promote unrelated stat
approvals into position approvals. Generating one suggestion per player-week would also violate the
requested no-repeat behavior. The two existing decision models therefore need a coherent integration,
including approval preview, append checks, undo, rejection, and cross-week deduplication. This exceeds
the tight precedence fix; stopped under the brief's balloon rule rather than inventing an unreviewed
second position authority or silently weakening the tests.

## Not delivered

- One durable position suggestion, approval updating the league position, and unchanged-feed deduplication.
- Fresh copied-profile 148-row comparison, top five causes and Schultz stat/rule trace.
- Classification of 472 absent-stat occurrences and evidence-backed inactivity ingestion.

The APP-SCHED-5 comparison is **previous-build evidence only**, not a result for this change. No new
comparison or absence categories are claimed. No scratch profile or live profile was modified, no
absent statistics were zero-filled, no git remote contacted. No independent review or GUI test.

## Next bounded step

Integrate the existing `PlayerPosition` decision with the stat queue as a player-level suggestion,
keeping weekly stat approval independent. Establish atomic ledger/preview/undo and deduplication
contracts before rerunning the copied-profile comparison. Then trace Schultz and classify missing
rows using independently held participation/status evidence; absence alone remains unavailable.
