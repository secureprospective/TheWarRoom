# Handoff

- Baton: Bee, lane `pulse-m0frame`, 2026-11-19.
- State: Fast-forwarded to `bundle/b9` `5b4edaea7`. Built `PhoneAppFrame` for the browser under `PHONE_QUERY`, moved Ask answers to `phoneAsk.ts` (held Home readings only), and retired `PhoneShell*`/`PhoneAsk` wrappers. Account access now sits at the foot of the league sheet: in-memory signed-in username when known, Authenticator routes to NativeAccount's existing screen, Sign out dispatches its existing `forget` action. Phone authenticator view has a return-to-league action. Focused Vitest (8 tests), frontend `tsc --noEmit`, and `make lint` pass; affected handwritten production LOC is `2047 -> 2421` (`+374`). `PhoneAppFrame` desktop fallback test passes at 1280. The responsive proof pair is in `docs/ui/mobile-v1/m0frame-proof/`: at 390/430 the app has no sideways scroll, clipped text, or targets below 44px. The screenshot uses empty-list fallbacks (no fabricated league/week data); mock-only screen content is intentionally absent from placeholder tabs.
- Next: commit, push, and open one PR into `bundle/b9`; inspect CI.
- Blocked on: nothing locally. CI is pending the PR.
- Tried and rejected: no new reads or backend paths, no mock league/sample scores/badges/live dot, and no visible reuse of existing tab content; each phone tab remains a named placeholder.
