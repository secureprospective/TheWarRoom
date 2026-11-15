# Handoff
- Baton: Bee, 2026-11-15; F27b ready for review, no deployment authorized.
- State: hosted ESPN board 5 seconds while live (response max-age honored through 10 seconds); changed live summaries fetched immediately with cap 4, unchanged reused for 8 seconds. Shared HTTP client/transport; Sleeper >=60 seconds; semantic scores-topic deduplication; Pulse follow delay 250 ms. Changed cycles log tick duration. Local cadence and checkpoint/replay reduction unchanged.
- Flow: validated board → concurrent packet acquisition/admission → existing serial reducer/checkpoint → readable overlay → scores topic → client coalesced reread. Replay bypasses summary warming.
- Evidence: requested Go race selection PASS: 12 top-level tests + 6 subtests (now selection matches none); hostedPulse frontend 15 PASS; TypeScript check PASS. Commit/push hooks enforce remaining gates.
- Next: review PR #269, then operator-controlled live verification: watch beside ESPN/TV and correlate changed tick logs with screen updates.
- Blocked on: live upstream-to-screen timing not exercised; no deployment authorized.
- Tried/rejected: changing shared poller cadence would alter legacy local mode. A concurrency fixture initially shared its map with packet admission; detached the fixture map and reran race tests cleanly.
