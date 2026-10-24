# Handoff

## Baton
Bee — 2026-10-24. Slice 26a on `session/cs26`; review pending. Do not merge.

## Where it stands
- Scoped token HTTPS admission uses the shared service and transaction-time credential checks; preview is the default dispatch mode.
- Events retain token audit ID and named issuer attribution. G2/G3 need five-minute, command-bound confirmation minted by a recently authenticated human device session.
- Public caretaker seat events apply only to vacant franchises. Owner seating clears the caretaker. Lineup/pick seat checks accept the active commissioner caretaker, never proxy authority over an owner.
- Dedicated lineup/draft service paths retain calendar checks, recheck them under the writer, and keep lineup legality/locks and draft clock/roster/cap checks. Generic mutation refusals remain.
- People displays caretaker status; native command catalog and Wails models regenerated.
- Final `make verify VERSION=dev COMMIT=cs26a BUILD_DATE=cs26a` passed: race-enabled Go suite, lint, vulnerability scan, binding/catalog freshness, TypeScript, 130 frontend files / 1,417 tests, build, bundle and CSP checks.

## Next move
Claude reviews the diff and tests; run the throwaway-profile live gate before merge. CLI/MCP remains slice 26b.

## Blocked on
Review/live gate only; no deployment or merge performed.

## Tried and rejected
- No generic lineup/draft bypass or commissioner proxy over owned teams.
- No agent-controlled confirmation boolean: consent is signed and bound to the exact canonical request and token ID.
- Initial full verification exposed missing caretaker catalog/golden rows and long frontend lines; fixed and regenerated. One attempt timed out under load; later attempts waited for the shared linter lock. Final full gate passed.
