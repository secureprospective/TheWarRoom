# Handoff
- Baton: Bee, 2026-11-15; F27 ready for review, no deployment authorized.
- State: hosted ESPN cadence 10 seconds during live games; Sleeper cached for 60 seconds; semantic scores-topic deduplication; Pulse follow delay 1 second. Local cadence and checkpoint/replay reduction unchanged.
- Next: review PR, then operator-controlled live verification and deployment.
- Blocked on: nothing in implementation; live upstream-to-screen timing not exercised.
- Tried/rejected: shared poller cadence change would also change legacy local mode; kept cadence/throttling in the hosted collector.
