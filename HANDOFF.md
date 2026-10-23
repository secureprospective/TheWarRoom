# Fresh-install crash handoff

## Baton
ClaudeBox / reviewer — 2026-10-23. Bee completed `session/t467-fresh-install`; do not merge without review/live gate.

## Where it stands
- Initial mirror seeding uses the locked refresh without post-refresh readers. Draft feeds load only after history, season and all stores are assigned.
- Regression: `TestFreshInstallDraftFailureKeepsLeagueUsable` reproduced the nil-history panic before the fix; now checks both unavailable draft feeds, correct season, preserved roster and completed store floor. Uses only fake HTTP.
- Siblings fixed by the same ordering: futureDraftPicks fallback; season-zero draft requests; move revalidation waiting on startup's own ready channel.
- Frozen frontend install/build passed. Targeted startup tests and full composition `-race` batch passed (470s).
- Final `go test -race ./... -timeout=15m` timed out in composition (900s); all other packages passed. Full-suite success remains unproven.

## Next move
Review the PR and run a throwaway-profile desktop fresh install. Passing: shell opens, roster remains usable, unavailable draft feeds are labelled, no panic or startup hang.

## Blocked on
Independent review, CI and desktop live gate. No real MFL calls or roster moves were made.

## Tried and rejected
- Publishing history alone would leave season-zero draft reads and startup waiting on its own readiness through plan revalidation. Defer those post-refresh consumers instead.
