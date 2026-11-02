# APP-STALL-2 — handoff (2026-11-02)

Branch `build/STALL-2` off `build/INTEGRATE-1102` @ 0c08e727. Commits: c9655954, 4c253a95, dcf0e177, cec664b8.

## Causes (evidence)
- **Per-tick replay** (confirmed): the 1 s value pass (`Preparation → Source.Scopes → ReadNativeBase →
  GovernanceState → governance.Replay`) and `DueWorker.ProcessDueReserves` re-folded 16.7k events per call.
  `governance.Replay` clones the whole State on every event. A later dump showed that the value pass reads
  records through `leagueservice.Records`. That path decoded fresh records from events on every call, so it
  could never hit a memo.
- **Stat-approval flood** (confirmed; not re-approval): all 16,682 `commish.stat.queue.approve` items are
  distinct player-weeks (weeks 1–8 × ~2,100 players; 0 duplicate items, 0 repeated digests). They arrived in two
  bursts: 11-01 20Z (7,059) and 11-02 17Z (9,623). The "344 per run" is week 8's live lines (774 live players).
  `FeedStatMoves` already skips seen items, so the feed was idempotent. The flood came from one entry per line.
- **"Still reading…" forever** (new finding, the real hang): a history read-pool deadlock. `nativeRecords`,
  `leagueRecords` and the preview opened a read tx and then called `linkedIdentities → accounts.Aliases`. That
  read uses the same 10-connection pool (`accountstore.New(a.core.HistPools)`). SIGQUIT at 4 min (build before
  the fix): 9 `Tx.awaitDone` goroutines, each created by a goroutine in `Aliases` waiting in `DB.conn`. 34
  goroutines in total were waiting for a connection, and nobody was using one.

## Fix
1. `commissioner.GovernanceState` memoizes per held revision (`governanceCache` on records[0], set by
   `WithReplayCache`). It is keyed by verified-identity content (`Principals.Key`) and closed books, and
   revalidated on every use by replay keys and canonical requests. It survives `WithoutClosedBooks`, so aliased
   reads hit it. `leagueservice.Records` now uses the store's revision-keyed cache (`HeldCommissionerRecords`),
   which validates the same way, adds stricter receipt checks and carries the memos. No incremental replay:
   `governance.Replay` was not proven incremental-safe (closed books, reverts and draft restores rewrite the
   active set), so the fix memoizes whole results keyed on the revision.
2. One ledger entry per feed batch: `Args.Stat.Items` (≤500, distinct, approve only), grouped by feed reason.
   `RawStatChange.Lines()` expands a batch, so every fold (effects, decisions, seen set) reads it item by item. Old
   per-line entries still read; no history rewritten. Batches are feed-only: `CheckStatAppend`, previews and
   `statWeekFor` refuse them, and the feed checks each item against its own evidence before one append.
   Known difference: decision lists order a pass by reason group rather than by line. State is identical.
3. Read paths resolve linked identities before `BeginTx`. "No aliases" is remembered on the ctx.
4. `engine: not due, last run … (no new inputs since; next by age at …)`, logged once per last run.

## Verification (fresh copy of ~/f2click config/TheWarRoom + data, build cec664b8, DISPLAY=:5, 1920x1080)
- Idle CPU before any clicks: 474 → 502 → 307 → 410 → 180 → **60% at 2:35**. The previous build idled at
  184–311% at the same point.
- After visiting HQ/Pulse, CPU 300–730% and RSS peaked at **6.2 GB** (5.0–6.2 GB at 3:30–4:00), then
  2.3–2.7 GB. **RSS and post-click CPU criteria FAIL.** Cause: see debt 1.
- RSS samples every 30 s (MB): 2204 1798 2425 1889 1659 1675 1999 5017 6187 4788 3827 3650 3859 2857 2368 2297
  2503 2454 2506 2727.
- Ledger: 16,774 → 16,775 in 10 min. One entry carries the 344 week-8 live lines (`feed: newest live`); the old
  code wrote 344 entries. A restart on an already-batched copy added 0. "deadline exceeded": 0.
- Screens: Home full (live matchup, race, news, transactions). War Room Trade Desk full. Control Room → Jobs
  full (7 jobs; ledger shows "Counted … for 344 player weeks"). HQ Lineup: cap/roster counts load, but the
  lineup, bench and this week stay "Not available yet" at 10 min. Pulse Now: week 8 header, matchups "Reading
  the week…". Pulse Race: "My road" loads, standings/playoffs stay "Reading…" at 10 min.
- Tests (`-race -p 2`, one package at a time, `-run` filtered): commissioner 49/49 (adds batch equivalence,
  feed-only batch, memo hit/invalidation); store/history 102/102 (adds 40-reader alias pool test, which failed
  before the fix; two feed tests updated to the batch contract); governance 1; leagueservice 1 (new);
  app/commissioner 4; appvalue 6; composition 39; app/scoring 4. `make build` passes.
- Screenshots: `~/scratch/stall2-shots/` (4x = final run, 5x = 10 min).

## Debt
1. **Week results re-read the whole held season per starter.** `weekResults → franchiseWeekScore → addStarters →
   PlayerScore → nativeEvidence → PlayerWeekStats → heldSeason` reads and hashes the weekly, plays, snap and
   injury archives, then `heldAbsenceProofs → heldNonParticipation → depthAt` for every player. This is what
   pins Pulse standings/Now and the HQ lineup and drives the 5–6 GB spikes. It was hidden behind the pool
   deadlock until now. The fix is a `heldSeason` result cache keyed on the archive revision plus crosswalk and
   identity inputs. That touches scoring inputs, so it needs a design call; I did not attempt it.
2. `wholeStream` holds `eventsMu` across DB reads, so one slow reader serializes the rest.
3. HQ footer shows "Could not refresh your league…": MFL keyring unavailable in the test desktop (expected here).
4. Carried from STALL-1: Home right tile column clipped; stat-approval compaction/snapshot policy.
5. Write paths (`BeginWrite` + `transactionRecords`) still read aliases inside the write tx. They don't
   deadlock, because aliases use the read pool, but they hold the writer while waiting.
