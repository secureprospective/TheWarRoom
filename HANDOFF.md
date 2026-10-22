# R2 caller-team handoff

## Baton
ClaudeBox/reviewer — 2026-10-22. Bee completed the startup follow-up on `session/r2-caller-team`. Not pushed.

## Where it stands
- Existing R2 identity, own-franchise enforcement and picker behavior remain unchanged; 133 bound methods, no Go changes in this follow-up.
- Live command-registry creation now loads MFL key status once. Control Room consumes the same state without issuing a mount request. Fixture creation makes no key-status request.
- Regression: `loads identity at live startup: own-team Act enabled without Control Room; others refused` mounts Acts without Control Room, observes pending → own-team enabled / other-team Go refusal, then mounts the connection surface and verifies no second read. Removing startup loading fails the request-count assertion; restored.
- Final gate results are in the builder report. Scratch logs: `r2-startup-verify.log` and `r2-startup-mutant.log` (operator log directory, outside this repo).

## Next move
Review the single follow-up commit and live-test a fresh throwaway profile: own-team planning enables without visiting Control Room; another franchise refuses; opening Control Room does not cause another status read. Nothing may be submitted to MFL.

## Blocked on
Independent review and live GUI verification. No implementation blocker.

## Tried and rejected
- Control Room-only loading left own-team actions stuck on Checking MFL after fresh launch.
- First full verify caught two overlong test names; shortened without changing assertions.
