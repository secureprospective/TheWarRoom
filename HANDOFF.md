# Handoff

- Baton: ClaudeBox, 2026-11-18 (PR #359 open against `bundle/b5`).
- State: Hosted Picks tab is implemented on `session/pulse-b5picks`; PR CI and pre-push verify are green.
- Verified: `pnpm test` 921/921; `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; `correct.pick` preview → review → execute test; PR #359 checks green.
- Next: ClaudeBox review and merge PR #359 into `bundle/b5`.
- Blocked: `tidy-check` found unrelated active lanes `b5perm` (PID 3899489), `b0load2` (PID 3922341), `b3home` (PID 3945562); left untouched.
- Slop fixed: hosted pick parsing preserves slot/status metadata; correction choices carry slot to target the exact pick. Pick ownership ops without matching hosted commands remain unavailable.
- Open decisions deliberately untouched: snooze per person/league; deputy job visibility; written-request visibility; division naming.
