# Handoff

- Baton: Bee, 2026-11-17 (Picks lane; based on latest `bundle/b5`).
- State: Hosted Picks tab reads all `native.assets.read` picks, filters year/round/traded/team, and states condition/action gaps honestly. Owner corrections use existing `correct.pick`; hosted pick choices include slot for exact addressing. No rookie-draft command is repurposed for future ownership.
- Verified: `pnpm test` 921/921; `pnpm exec tsc --noEmit`; `make lint` (0 golangci issues); `make bloat`; targeted preview/review/execute test.
- Next: push and open PR against `bundle/b5`; ClaudeBox reviews after CI is green.
- Blocked: `tidy-check` reports unrelated active lanes `b5server` (PID 3560779) and `b2docket` (PID 3627613); left untouched.
- Slop fixed: hosted pick parsing preserves slot/status metadata; hosted correction choices carry slot so `correct.pick` can identify the exact pick.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
