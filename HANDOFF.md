# Handoff

- Baton: Bee, lane `pulse-m0frame`, 2026-11-19.
- State: Fast-forwarded to `bundle/b9` `5b4edaea7`. Built `PhoneAppFrame` for the browser under `PHONE_QUERY`, moved Ask answers to `phoneAsk.ts` (held Home readings only), and retired `PhoneShell*`/`PhoneAsk` wrappers. Account controls sit at the foot of the league sheet; Authenticator and Sign out reuse NativeAccount's existing flows. The commissioner-only Control Room entry is in that sheet on the League tab and opens the existing route outside the placeholder body. Responsive screenshots/reports are in `docs/ui/mobile-v1/m0frame-proof/` (390/430, no sideways scroll, clipped text, or targets below 44px); the proof uses empty-list fallbacks, not mock league/week data. Focused Vitest (10 tests), frontend `tsc --noEmit`, and `make lint` pass; handwritten production LOC is `2047 -> 2433` (`+386`).
- Next: commit the Control Room follow-up, push, open one PR into `bundle/b9`, and inspect CI.
- Blocked on: nothing locally. CI is pending the PR.
- Tried and rejected: no new reads or backend paths, no mock league/sample scores/badges/live dot, and no visible reuse of existing tab content; each phone tab remains a named placeholder.
