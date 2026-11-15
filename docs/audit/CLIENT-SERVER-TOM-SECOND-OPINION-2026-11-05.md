# Tom second opinion: TheWarRoom client/server plan (2026-11-05)

Read-only review of Sol's RECOMMENDATIONS, Christopher's RULINGS, the INVENTORY (grepped), and twr-architecture.html. Code citations are taken as claims; I did not check them against source.

## 1. Verdict on the target shape: **sound, with changes**

- One Go server that holds the only writer, with thin desktop, PWA and CLI clients, is right for 32 franchises and one maintainer. Most of it already exists: the hosted origin, accounts on the same writer, `runNative`, SSE invalidation, the age→R2 backups, and TUF. Sol is right to finish that path and not build a new one.
- Solo use as "the same app pointed at a local server" is right because it keeps one code path. But three solo details are missing, and each is a real trap (item 4, #2).
- The one structural change I would make is **two SQLite files on the one server, both owned by the one process**:
  - **authority**: ledger, accounts, plans and audit chain. A few tens of MB, hot, snapshotted every 5 minutes.
  - **evidence**: raw_archive, observations, identity and model outputs. Hundreds of MB, append-mostly, backed up incrementally.

  Putting everything in one file is what breaks the backup and speed numbers. Details are in item 2.
- No Postgres, no Litestream, no WebSocket, no sync layer. I agree with all of those.

## 2. Ruling 6: move leagues, accounts, plans and full history at cutover?

**Moving all of it is right. Moving it as one file at the cutover moment is not.**

Risks as currently planned:
- **It breaks backups.** 531 MiB of logical pages is already over the 512 MiB / 128 MiB caps. Ruling 4 lets us raise the caps, but a full ~550 MB snapshot every 5 minutes is still a problem:
  - It holds the writer each time, which breaks the 100 ms click budget.
  - It uploads about 160 GB a day over a home uplink.
  - Raising the cap stops the backup failing outright, but none of those costs go away.
- **It mixes public and private data.** Most of the bytes are public football evidence that the server's own collectors will keep producing after step 2:
  - raw_archive is about 165 MB and observations about 102 MB.
  - The private league authority is small: league_event about 36 MB, audit_chain about 5 MB, roster_restore about 1.3 MB.
  - Copying both as one blob ties the cutover's correctness check to 500 MB of data that has nothing to do with the leagues.
- **It carries dead weight.** The listed table bytes add up to only about 340 MB. The other ~215 MB is indexes and/or free pages, and the split hasn't been measured. Derived outputs (model_scores, run_scores) can be recomputed.

What I would do instead:
1. **Seed the evidence in step 2, not at cutover.**
   - Run `VACUUM INTO` on a backup-API snapshot.
   - Load it into the server's evidence file through the archive's existing content-digest path, so every row is checked and nothing is file-copied.
   - Recompute model outputs on the server and don't migrate them.
   - Keep all provenance. Nothing gets deleted to fit a cap.
2. **Keep the cutover small and fenced.** The step-5 cutover then moves only the authority data, through the existing transfer/import:
   - leagues and league_event
   - the audit chain
   - accounts
   - plans: thewarroom.db personal/rulebook/params and whatif.db trade_plans

   Gate it on ledger and checkpoint parity, and fence the old writer.
3. **Back up each file on its own schedule.**
   - Authority: a full encrypted set every 5 minutes. It's small, so the writer stall is negligible and RPO 15 min holds easily.
   - Evidence: raw_archive is already keyed by content SHA, so upload each object once, with a daily manifest. That gives incremental backup with no new tool.
   - Restore = authority set + evidence manifest. Measure both against RTO 4 h.

## 3. Ruling 5a: the simplest correct "updated data available, refresh" alert for a solo owner

Build no new channel. A solo owner's local server runs the same collectors, so football data updates by itself. The only thing that can't update itself is MFL reference data, because our ledger wins. So:

- **Detect on the server.** The local server's existing MFL reference poll does a cheap check about hourly, inside the 2 req/s limit and never using the once-a-day players endpoint. It compares a digest or last-transaction ID against the last confirmed import.
- **Record it in the existing status reading.** When the values differ, the server sets `import_available{since, summary}` there and fires the existing SSE invalidation.
- **Show a banner in the client.** The banner reads "MFL has changes since your last import — Review" and opens the existing import preview, then a confirmed ledger change. Nothing applies automatically. There is no OS notification and no email.
- **One code path.** Hosted leagues and commissioners get the same banner.

## 4. Top 5 things Sol got wrong, missed or over-engineered (ranked)

1. **One DB file and one backup cadence for 550 MB.**
   - Problem: full 5-minute snapshots stall the writer and flood the uplink. Raising the cap (ruling 4) hides this rather than fixing it.
   - Fix: split into authority and evidence files with separate backup cadences, as in item 2. This is simpler than Sol's fallback ("replace with one incremental pipeline") and fixes the problem at its source.
2. **Solo use is under-designed and has three traps.** Each needs a fix in step 5, because step 5 removes the local backend that solo owners use today.
   - Trap (a): the backup gate locks solo owners out. With ruling 4's fail-closed admission and no R2 configured, a solo server would block every league action.
     - Fix: make the backup target a setting (R2 or a local folder, same age encryption). The gate checks whichever target is configured.
   - Trap (b): TLS on localhost. The hosted origin and native client require verified TLS.
     - Fix: the installer generates a self-signed certificate on a loopback-only listener and pins it. This keeps the same TLS code path.
   - Trap (c): installation. Non-technical owners can't be expected to install a server by hand.
     - Fix: one installer with an explicit "I play solo" option that installs the same server binary as an OS service.
     - This contradicts Sol's "desktop installation does not … provision a server", but it is not silent provisioning, so it still satisfies ruling 1.
3. **Invalidation is over-engineered.** Sol proposes opaque recipient-view revisions, per-reading dependency graphs, validity deadlines and generated dependency metadata. That's too much for one league of 32.
   - Fix: a handful of topics per league:
     - `ledger-public`
     - `franchise:<id>-private`, sent only to that franchise
     - `scores`
     - `values`
     - `clock`
   - The server sends each topic only to subscribers authorized for it. The client refetches its active queries for that topic. Hidden bids stay private because they only change a private topic.
4. **Browser streaming is over-engineered.** "Authenticated fetch streaming" for SSE isn't needed when the UI and API share an origin and auth is a cookie.
   - Fix: use plain `EventSource`, which sends same-origin cookies, and put league and version in the path. That's less code and gives standard reconnects.
   - The same goes for the privileged "Update server" helper. On one CT, a single OpenRC-run update job with the same verify → snapshot → migrate → health → rollback steps is enough. No separate privileged daemon.
5. **Server memory isn't treated as a gate before step 2.**
   - Problem: the measured desktop RSS peak is 2.3 GiB, and held seasons have no cap. Moving collectors into a CT moves that peak onto the box that must answer clicks in 100 ms.
   - Fix: put bounded held-season and decoded generations (inventory debt #1) **inside step 2's gate**, with a CT memory limit. Sol leaves it as "a separate measured change".
   - Smaller gap: nobody has measured the tunnel round-trip yet. Measure it first; it decides whether 100 ms "confirmed" is reachable at all (ruling 3).

## 5. Migration order

Keep 1 → 2 → 6's closing gate. Changes:

- **Before step 2:** split authority and evidence, fix backup capacity, and seed the evidence (item 2). Server memory bounds go into step 2's gate.
- **Move step 5 (retire the local backend) ahead of steps 3 and 4.** Measuring the 200/100 ms gate against a local backend that is about to be deleted proves nothing, and the tree shrinks soonest this way.
- **Then step 3 (one read cache)**, measured against the real server and tunnel.
- **Then step 4 (PWA)**, since it needs the server path finished and a security review.
- **Solo installer, local backup target and loopback TLS move from step 6 into step 5.** Otherwise step 5 breaks solo use.
- **Step 6 unchanged.** No real-owner admission until it passes.

## 6. Inaccuracies in twr-architecture.html vs the docs

1. **"every 15 min, encrypted" to R2 is wrong.** BACKUPS.md does consistent sets every **5 minutes**. 15 minutes is the RPO ceiling, and writes stop at 14 minutes uncovered.
2. **"restore within 4 h" is presented as fact.** It's the RTO target and hasn't been drilled. Sol says a paper runbook doesn't prove it.
3. **"~31 s season refresh, ~3 min cold start" and "Screens wait 31 s – 3 min" are presented as measurements.** The inventory says neither was reproduced; both are operator observations.
4. **"Server caches — bounded … old seasons drop out"** reads as if data is discarded. Sol bounds only the in-memory decoded generations. Durable evidence is kept in full.
5. **"One SQLite database … the single source of truth"** is right about the authority, but the Today box omits that history.db already holds **accounts**. whatif.db is labelled "what-if scratch", but it holds contract ledgers, rosters and **trade_plans**, which are plans Christopher ruled must migrate.
6. **"fetched ONCE for all leagues" includes MFL.** MFL import is per league and credential, and Sol says private responses must never be merged across owners.
7. **"TODAY — every owner's computer runs the whole thing"** and "3 files per computer … on every owner's machine" describe a hypothetical. Today only Christopher runs it; the Pain box itself correctly says "would".
8. **Section 5 still shows the 7 rulings as open questions.** It doesn't reflect the 11-05 answers:
   - backup limits raised
   - TOTP optional for everyone
   - ledger wins
   - solo means a local server
   - speed conditions: home broadband, one league
9. **Rejected alternative (a)** reads as if solo use is gone. Under ruling 1, solo stays, as the same app plus a locally installed server.

Follow-up outside this review: ruling 5a says to update the "MFL wins (R2)" Hard Constraint in the project CLAUDE.md. It still says "MFL wins". I did not edit it.
