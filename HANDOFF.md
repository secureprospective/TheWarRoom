# Handoff

## Baton
Bee → ClaudeBox/operator, 2026-11-05, `session/keepalive-hq-home-pulse`.

## Where it stands
`frontend/src/app/shell/AppLeagueMount.tsx` owns one visited-screen set for Home, HQ, Pulse, War Room and Control. League/franchise scope resets it; at most five instances survive. Existing screen-owned refresh subscriptions stay live while hidden; inactive shared rail/tab portals are detached. Redundant loaded-component identities removed. Production physical lines 126 → 110 (−16).

## Evidence
Plain `make build` PASS; tsc PASS; 44 targeted Vitest tests across five files PASS. `warroomLifetime.test.tsx` exercises the four actual screens in one table: retained DOM/no revisit reads, hidden season refresh, franchise reset, five-screen bound, league disposal and subscription cleanup. Legacy NFL screenshots inspected at 1920×1080. Disposable process stopped and copied profile deleted.
Visual first/revisit milliseconds after a league reset: Home 20696/2000, HQ 6956/415, Pulse 8742/350, War 6618/579. Timed click → matching held-data pixels, not application performance marks; Home first includes league selection. Initial cold launch was much slower. No claim these are backend durations.

## Next / blocked
Review commit `2f20cf64`. Pull/push blocked by missing bundle remote. Application-log/performance-mark timing, hosted browser behavior and live GUI refresh mutation not verified (hidden refresh covered by tests).

## Tried / rejected / debt
No second cache or per-screen keep-alive implementation. No unrelated backend cold-read rewrite. Screens still retain their existing background subscriptions/retries, so cold acquisition and hidden-screen refresh costs remain. Wrapper keeps full workspace height; screenshots caught and corrected the initial auto-height regression.
