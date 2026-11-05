# Handoff

## Baton
Bee → reviewer, 2026-11-05, `session/trade-floor-cut` (local only).

## Where it stands
Trade Floor slices remain intact. Follow-up A: `32761874` removes Home's all-Pulse-before-any-tile dependency; existing progressive Pulse publications release week-dependent reads, other sources start immediately. Reproduced the empty Legacy Home on pre-branch `4e871c35`; MFL reference Home already loaded. No bisect needed.

Follow-up B: app and mirror mount owners retain one hidden War Room after first visit, preserving existing `useContractRead` state and change subscriptions. League/franchise changes dispose it. Stable lazy component identity avoids the extra lazy-to-loaded remount. No second cache; shared query audited, Pool remains G1 and ineligible for that held-read cache. HQ/Home/Pulse still remount and reread.

TypeScript and plain `make build` pass. Targeted: 7 files, 89 tests pass. Final full Vitest: 1,826 passed; exactly four inherited failures (assets, controls readability, personal, appLeaguePulse). Live copied-profile Home sources visible; three Legacy War Room revisit samples: before 966/819/830 ms, final 194/112/114 ms, including click/capture overhead. External screenshots/logs retained. Own GUI stopped and disposable profile deleted before close; protected PID untouched.

## Next move
Review the retained-view lifetime (including open-dialog navigation and hidden-view refresh load). Verify complete standings/odds when the slow season scorer finishes; this run proves unrelated Home tiles no longer wait for it, not that scoring itself is fast.

## Blocked on
Pull unavailable: configured bundle remote missing. No merge/push authorized. No MFL actions performed. Long soak, hosted-account navigation and open-dialog navigation not exercised.

## Tried and rejected
No parallel cache or registry-policy bypass for G1 Pool reads. No guessed standings, timeout-based success, MFL write, or hook bypass.
