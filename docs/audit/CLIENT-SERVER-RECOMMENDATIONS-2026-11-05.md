# Client/server recommendation — 2026-11-05

**Recommendation: one always-on authoritative Go server; Wails desktop, PWA and CLI/MCP are clients. Do not embed a second server in the desktop.** Build this by completing the hosted path already present, not by replacing the backend or inventing a sync platform.

Status: research proposal for Christopher/head-brain review, not an approved architecture change or deployment clearance. Fixed constraints in the research brief are inputs, not questions reopened here.

## Evidence and limits

Read the complete [inventory](INVENTORY-2026-11-05.md), including its appendices. Source baseline for this recommendation: `c012ac0b1033484cbb58fd1a892317bb3a5f794b`, branch `build/INTEGRATE-1102`; the inventory describes its preceding source snapshot. Independently inspected hosted HTTP/SSE, native desktop construction, CLI documentation, backup/storage/update runbooks, command/placement map and build doctrine. No code changes, app launches, production database opens, feed calls or performance tests were made. External references below are documentation references, not a claim of freshly verified service terms.

`git pull --ff-only` failed: the configured bundle remote is missing. Existing untracked `.golangci-cache/` and `.task-evidence/` are not this research's changes. Existing `HANDOFF.md` belongs to another work stream and is left untouched. This report is the handoff; push cannot be completed against that remote.

The supplied 31-second refresh and approximately three-minute cold start are operator observations, not measurements from this session. The inventory's 417–848 ms store-ready logs explicitly exclude substantial later work. Server placement removes that work from each client, but does not make the work cheaper or establish the speed law.

## 1. One target shape

```text
Wails React / browser React / CLI and MCP
  → authenticated HTTPS commands and authorized readings
  → existing commandcatalog → leagueservice → history + accounts, one writer
                                     → pure commissioner/scoring/valuation functions
server collectors → validated evidence/archive → prepared readings → same service
server commit/publication → recipient-safe SSE invalidation → authorized reread
consistent recovery set → age encryption → verified private R2 object
```

| Runs where | Responsibility and explicit exclusions |
|---|---|
| Dedicated UPS-backed laptop, Proxmox unprivileged Alpine CT | One Go server process, local-disk SQLite, accounts, permissions, native league authority, due worker, feed acquisition, canonical IDs, archive, scoring/model preparation, read caches, photos and backup manager. OpenRC supervision and existing verified origin TLS behind Cloudflare Tunnel. No public DB or administrative port. |
| Wails desktop, later Flatpak | Same React screens and generated command contract as PWA; native window, OS keychain, file-picker/export and verified desktop update integration only. Existing Go credential proxy may remain a necessary OS adapter, not an independent application service. No league SQLite, collectors, engine startup, due worker or authoritative local fallback. |
| PWA | Same React build and data/query policy; browser login/session adapter, manifest, service worker for versioned static assets. No second business implementation. |
| CLI/chat/MCP | Generated human/AI command names, schemas, authorized reads, preview/execute and scoped credentials against the same server. Automation does not bypass human confirmation or commissioner control. |

Reuse `internal/composition/run.go:runNative`: it already constructs the hosted client without the local App. Make that the sole desktop runtime after parity gates. For a person hosting alone, run the **same separately installed server** on their supported host and connect to it; desktop installation does not silently provision or own a server. CasaOS and Proxmox helpers wrap the same executable/configuration/health protocol, not alternative backend implementations. Public packaging remains later work.

### Why this over two alternatives

1. **Rejected: desktop embeds the server for solo use and switches to remote mode for leagues.** Even shared domain code leaves two lifecycles, credential stores, backup/update paths, endpoint discovery modes and authority-transfer cases. It also preserves cold ingestion on a client's launch. There is no approved offline-write requirement paying for those costs.
2. **Rejected: local-first database on every client with replication.** Conflict resolution for draft picks, bids, cap rules and deadlines is much harder than one serial authority. Replicas expand private-data exposure and make revocation, backups and correctness harder. At 32 franchises, replication solves the wrong bottleneck.

This is a modular monolith, not microservices: no Redis, message broker, Kubernetes, separate scoring service or cloud database. Keep the pure-engine and transport/store dependency laws.

### Browser delivery is a real missing boundary

Current `internal/serverhttp/hosted.go` refuses any Origin header; its SSE request validator also refuses Origin and query parameters. Current desktop keeps bearer secrets out of the WebView. A PWA cannot simply reuse that Go proxy or enable wildcard CORS.

Propose same-origin static UI and `/v1` API through the one tunnel. Browser sessions use Secure, HttpOnly, host-only cookies with SameSite protection; unsafe requests additionally require a session-bound CSRF token and exact Origin validation. Validate Fetch Metadata where available, without relying on it alone. Native/CLI bearer auth remains a narrow credential adapter into **the same** session/admission checks. Reject ambiguous cookie-plus-bearer requests. Never store long-lived browser bearers in localStorage, URLs or service-worker caches.

Serve a restrictive CSP; escape untrusted league/news content; no remote executable scripts. Keep version/resource/backup gates and per-request authorization on both adapters. Use authenticated fetch streaming for browser SSE so league/version/proof headers remain explicit; no new WebSocket or query-token protocol. Browser logout/revocation cancels streams and clears private memory. This auth/exposure proposal needs review before implementation, not a permissive switch on today's origin guard.

## 2. Data, updates, one read cache and offline behavior

**Authority:** server `league_event` plus its atomic audit/receipt state determines native league changes; accounts live on that same writer so revocation and admission cannot race a commit. Football evidence is durable provenance used by that authority, not another league master. MFL remains a validated reference/import source: failure or refresh never overwrites native league decisions. Import goes through settings/preview/confirmed ledger changes in the one app, not a source-tagged league mode.

**Updates: keep SSE, commands over HTTPS.** This matches the existing `serverhttp/invalidations.go` and `serverclient/invalidations.go`. WebSocket adds unnecessary duplex lifecycle and replay machinery; periodic client polling duplicates traffic and delays updates. The current server internally checks committed heads every 250 ms, checks sessions every 500 ms and sends a 15-second heartbeat. It emits only a protocol invalidation, not ledger payloads or a replay cursor. Preserve that privacy property.

Generalize the existing stream to invalidate authorized evidence/model/time-dependent readings as well as ledger changes. A league revision alone is insufficient for live stats, new evidence, prepared values, parameter changes and clock boundaries. Return an opaque **recipient-view revision** covering each reading's dependencies and a validity deadline for time-dependent views. Never expose a private global ledger counter or private-change timing to unrelated owners. Keep server-only global generation counters solely as cheap work gates; compute notifications after recipient authorization.

Replace the 250 ms detection timer with coalesced post-commit/publication wakeups **only after measuring it against the speed law**. Preserve subscribe/read race protection, session rechecks, reconnect refresh and bounded slow-consumer handling. A periodic internal reconciliation can catch missed wakes; it is not a second client protocol. On disconnect, mark view freshness unknown; on reconnect, invalidate and reread rather than replaying ledger events. Restart/restore changes the server epoch so coincident revision numbers cannot resurrect old cache entries. HTTP/2 and one shared stream per app session/active league avoid per-screen connections. Test stream budgets with multiple devices: today's 64 total / two per device are not a proved production capacity.

### Turn War Room keep-alive into retained data, not retained screens

Extend `frontend/src/app/data/query.ts`, not a second library/cache alongside it. One bounded read-cache owner per client session; hooks subscribe to it and store only ephemeral UI state. Keys include server identity/epoch, authenticated account and auth generation, league, authorized franchise/view, command/schema version and canonical validated arguments. Results carry view revision, validity deadline and provenance/freshness. Generated catalog dependency metadata drives invalidation; do not hand-maintain a parallel invalidation catalog.

- Cache accepted immutable reading payloads once, deduplicate in-flight requests, fence old responses, and expose small selectors to Home/HQ/Pulse/War Room. Do not clone an entire ledger into each screen store.
- Start with a conservative shared bound of 32 inactive entries plus a byte budget; choose the byte budget from measured real bundles. Active entries are pinned only while observed; cap active query shapes/pagination too. Cancel work when no observers need it. No uncapped outer franchise maps.
- A matching, still-valid revision serves immediately. Invalidated entries may stay visibly labelled while revalidating but cannot enable an authoritative action. Mutation receipts update the initiating view/revision and invalidate related reads immediately, without waiting for SSE.
- Authorization changes/session end clear private entries and fence pending publications. On reactivation after sleep/disconnect, recheck membership/session before rendering private cached content. Never allow cross-account or cross-server reuse.
- Separate G0 reads from G1 refresh/preview effects. Pool's read → refresh → reread sequence must become an explicit refresh/job command followed by a pure reading **before** sharing its completed results. Never cache a side effect as a read.
- Retain only drafts, scroll/selection and required dialog state outside the read cache. Once parity and navigation tests pass, delete both hidden War Room mount owners and redundant result maps. No hidden-screen subscriptions just to make revisits fast.

**Offline:** installed/static shell and harmless preferences remain available; already-visible in-memory readings can remain labelled disconnected/read-only. No persistent private football cache initially, no queued league writes, no offline scoring and no automatic replay on reconnect. A cold offline launch shows the shell and connection explanation, not fabricated league data. A lost response means outcome unknown: query the durable receipt by request identity before retry; ensure idempotency rather than blindly resubmitting. Rule/draft/lineup execution always validates current authority and expected revision server-side. The server clock, not a device clock, decides deadlines.

## 3. Engines, collectors and near-instant startup

Move existing collectors and preparation ownership to the server; do not instantiate the whole desktop App inside `twr-server`. Hosted currently uses pinned football evidence, while directory/keys/live stats/model work is wired in desktop composition. Connect the **existing producers** to server lifecycle and validated publication; remove their desktop startup wiring in the same slice. Pinned evidence remains reproducibility/test/restore input, not a permanently separate live supply implementation.

| Work | Server schedule / policy |
|---|---|
| Native due actions | Existing `leagueservice.DueWorker`, same admission/writer path; durable due state, catch-up after restart, idempotent execution. Never depend on a window being open. |
| ESPN live / Sleeper gap fill | Preserve current busy 1 min / idle 5 min and final reread after 10 min initially. One provider fetch per game/stat scope shared across leagues; do not multiply by clients. Consolidate duplicated ESPN acquisition only where validated payload requirements genuinely match. |
| nflverse held finals and supporting evidence | Preserve source due policy initially (inventory: 20 h open / 30 d closed file gate); run independently of screen reads. Polling/revision policy must support later corrections. Parse once per selected content digest and bound decoded generations/seasons; durable evidence remains intact. |
| MFL reference/import | Shared runtime host discovery, existing 2 req/s burst 1 ceiling, and endpoint-specific restrictions such as players once/day. One server-wide limiter, coalesced work by endpoint/season/league/credential scope; never merge private responses across owners. Disabled unless reference/import is configured. No MFL mutations. |
| Values/models | Existing dirty/input-manifest checks and `appvalue.Preparation` producer; publish versioned completed readings. Reuse saved prepared JSON only after digest/from-until validation. Share canonical model math between entry paths; do not assume modelrun and native preparation are duplicates. |

Use bounded worker concurrency in the one process, provider timeouts, jittered backoff, Retry-After handling and no overlapping refresh for the same scope. Preserve per-stat priority **nflverse > ESPN > Sleeper**; MFL is not a native scoring fallback. Missing is not zero; last-good reads retain source/as-of/reduced provenance. A provider outage must not invent facts or silently rebalance the model.

Give league writes/reads priority over backfills and value preparation. Persist immutable input manifests and completed publications; expensive preparation never holds SQLite's writer during computation. Publish only if dependencies are still current. Reserve memory/CPU for serving, measure the inventory's multi-gigabyte historical decode spikes, and address compact facts/bounded generations as a separate measured change. Moving those spikes to the laptop is not a fix by itself.

Client launch becomes static shell → session/version check → small authorized bootstrap reading → shared queries + stream. No season-wide fetch, database migration, historic decode or engine warming on the critical client path. Server restart restores authoritative state first, warms bounded hot projections, then backfills; unavailable derived views say Preparing while independent screens work. Correctness-sensitive operations must wait/refuse if their required state is not ready.

**Speed acceptance:** measure click-to-visible-screen and click-to-confirmed-outcome through the real tunnel, warm and cold, not just Go handler time. Cache/prefetch small authorized screen bundles and progressively publish independent Home tiles. Show local interaction feedback within 100 ms, but do not count that as a committed change. Require 200 ms screen / 100 ms click budgets as given; report full distributions and every observed breach separately. Ordinary WAN RTT or a cold offline browser cannot be guaranteed below those bounds by architecture alone. Christopher must identify the supported measurement conditions; do not silently redefine success as a spinner or raise the limits.

## 4. SQLite, backups and migrations

**Keep server-local SQLite WAL, FULL synchronous, foreign keys, one immediate writer and bounded read pool.** Existing `internal/db/pools.go` already does this. One server history-schema file plus accounts is the target; import required settings/plans from desktop files through validated migrations, rather than copying all three desktop DBs into production. Temporary what-if computation is a request/session-scoped projection; any deliberately saved plan is server-owned private data through the same command boundary.

At this scale a network Postgres adds another supervised service, credentials, backup consistency and update surface without a demonstrated write-concurrency need. A remotely shared SQLite file or multi-writer SQLite replication is inappropriate. Reconsider storage only on measured lock/throughput limits, not franchise count alone. WAL permits concurrent readers, not concurrent writers; bound read transactions and watch WAL/checkpoint growth.

### Prefer the existing encrypted recovery sets over adding Litestream now

`deploy/server/BACKUPS.md` already describes five-minute consistent full recovery sets, age encryption **before** R2, verified ciphertext read-back, configuration/checkpoint manifests, retention and writer admission stopping at 14 minutes of uncovered time. Reuse it. Preserve the post-writer-wait coverage check and restart-paused behavior. Do not copy a live main DB/WAL/SHM set or count an upload acknowledgement as verified recovery.

**Immediate blocker: moving the measured desktop history wholesale exceeds today's recovery envelope.** Inventory: history logical bytes 556,761,088 (about 531 MiB), before other members; current uncompressed recovery cap is 512 MiB, encrypted-object cap 128 MiB. Hosted size is unknown. Compression cannot fix an uncompressed-member limit. Full snapshots every five minutes also reserve the writer, potentially breaking the 100 ms click budget. Prove a realistic hosted evidence load and backup/write latency before admitting owners.

First measure: recovery-set sizes, snapshot duration/writer stall, upload plus read-back time, daily R2 bytes and four-hour restore. Retain evidence required for replay/provenance/corrections; do not delete it to fit a cap. If the measured workload requires larger bounds, make one explicit capacity/retention change with storage and RPO tests. Do not silently lift limits. If full-snapshot cost is still unacceptable, replace this backup implementation with **one** encrypted incremental recovery pipeline; do not add a parallel authority/backup manager.

Litestream-style WAL shipping is a possible later replacement, not the initial recommendation: prove the exact pinned release supports local pre-upload encryption, R2 compatibility, config/secret recovery, manifests, retention, authenticated restore and the same coverage admission contract. R2 server-side encryption alone does not meet the current encrypt-before-upload design. SQLite replication is backup here, not automatic failover. Published Litestream features are not evidence that these project guarantees already exist.

Migrations: reuse supported-version refusal, read-only preflight, consistent pre-migration snapshot/checkpoint, transactional migration and monotonic release stamp. Test oldest supported and current files, interruption/disk-full and preserved state on refusal. Never migrate from a desktop client or run old binaries against upgraded state. Rollback restores matched binary/data/configuration under service exclusion; do not promise lossless rollback after new writes.

RPO requires checked snapshots no more than 15 minutes behind acknowledged state; RTO includes spare hardware, Proxmox/CT provisioning, restore key availability, R2 download, chain/manifest checks, session invalidation, tunnel readiness and owner access. Automate recovery/install/health steps after explicitly supplied credentials and recovery authorization. A paper runbook and a local snapshot test do not prove RTO. Keep restore keys off-server with deputy custody. One laptop is still one failure domain; UPS is not high availability.

## 5. Engineering and one-person operation

### Keep the repo and contract, retire duplicate doors

- `cmd/twr-server`: composition/lifecycle for the one service; reuse `internal/serverhttp`, `internal/leagueservice`, collectors, stores and offsite manager. `internal/engine` stays I/O-free; ingestion does not import upward; stores do not import each other.
- Root Wails entry/`internal/composition`: native client composition only at the end. `internal/serverclient`: native credential/update adapter and CLI transport as appropriate, not another league service.
- `cmd/twr`: human/AI client and MCP projections. Offline restore/transfer commands stay operator-only exceptions, not owner-side DB access.
- `frontend/src/app`: one UI, one read-cache owner, small transport/session adapters for browser/native. `deploy/server`: one server payload and automated service lifecycle, later packaging adapters. Update placement/system map when responsibilities change; no new top-level framework or package hierarchy by default.
- Extend `internal/commandcatalog` and `frontend/scripts/nativegen` as the sole contract source: typed args/results, Raw validation, version, permission/exposure, effect class and reading dependencies. Generate TS types/catalogs, CLI/MCP descriptors and a machine-readable HTTP schema from it. Keep hand-written boundary validation; generated TS is not runtime validation. No independently maintained OpenAPI business catalog.
- Today's 141 desktop bindings are not 141 CLI-executable commands: the inventory distinguishes 101 named mutations, special reads and special pick/seed doors. Close actual UI/CLI/AI capability gaps deliberately; schema generation does not grant exposure or delegation. Share service semantics, not necessarily identical credential powers.

### Verification before each cutover

Reuse existing scorer goldens, Raw validation/fuzz coverage, SQLite transaction tests, real TLS hosted workflow tests, CLI/MCP release mutation matrix, restore/version/trust tests and locked-mock UI tests. Add only missing behavior checks:

1. Same command/read via Wails, browser and CLI/AI where authorized; identical outcome, rejection and unchanged-state assertions. Concurrent revisions, duplicate request IDs, timeout-after-commit and restart catch-up.
2. Browser cookie/CSRF/Origin/CSP tests, login/logout/TOTP and revoked stream tests; two leagues/accounts, hidden bids/private trades, recipient revisions, delayed responses after auth switch. Do not mock away admission or persistence.
3. Shared-cache race/fence/dedup/eviction tests and real browser navigation: Home/HQ/Pulse/War Room, changed franchise/league, open dialogs, background/sleep/reconnect, static update and offline cold launch.
4. Feed replay/recorded fixtures for per-stat ladder, leading-zero IDs, missing-versus-zero, late corrections, no future affiliation, reduced-source labels and digest-valid preparation. No real MFL calls needed.
5. Full-size cold server restart + 32-owner/multi-device load, live scoring/backfill + backup overlap, measured memory/WAL growth; client near-instant revisit must not depend on hidden mounts.
6. Disposable-host install/update and destructive off-host restore drill, with actual R2 and fresh tunnel, latest acknowledged ledger/checkpoints, all restored sessions revoked, timed RPO/RTO.

Run changed-package `go test -race` and affected frontend tests in batches, then the normal hook/CI gates. No repeated full-suite runs on an unchanged tree and no hook bypass. Tests, reviewer trace and live locked-mock screen verification must refer to the same candidate.

### Deploy/update without recurring console recipes

Reuse existing pinned TUF release verification; no paid platform signing and no unsigned download-and-execute fallback. Build one reproducible server artifact plus UI assets and version metadata. Existing version/security-floor gates cover desktop/CLI; extend them to browser assets/service-worker upgrades. Cache hashed static assets only, never authenticated `/v1` data at Cloudflare or in the service worker. Keep API responses no-store. Retire old static caches at a safe reload boundary, not mid-command.

Replace the current manual `UPGRADE.md` console sequence with one authorized **Update server** command/job: verify staged artifact → pause admissions → consistent verified off-site recovery set → stop/exclude writers → migrate/check → activate → readiness → fresh backup → resume. A minimal privileged installed helper performs only approved lifecycle/filesystem operations; candidate code stays unprivileged. Failure safely retains/restores a matched state and presents one actionable fault, never automatically downgrades live data. Desktop uses its existing verified update path; later Flatpak uses its packaging trust without a second app updater fighting it.

One-time domain/tunnel/R2 credentials and offline recovery-key custody cannot be magically automated away. Put unavoidable inputs in a guided setup/restore flow; no recurring cron edits, hand-copied DBs, manually pinned football files, owner-by-owner client repair or routine CT console steps. Restoration and destructive authority changes remain explicitly authorized operations.

### Small operational surface

Reuse `internal/serverstatus` and existing private logs, adding missing measurements rather than a public admin API or a new telemetry stack: API/click/screen latency, writer wait and backup stall, feed age/failures/rate-limit state, job lag, prepared revision age, stream counts/reconnects, memory, disk/WAL and checked-backup age. Structured bounded logs carry request/job IDs, not secrets or private payloads. Local-only sampled profiling is incident tooling, not an exposed endpoint.

Health separates liveness, usable authoritative reads, write readiness and derivative/feed freshness. Present these in the existing server/control surface with one next action; automate checks/alerts without sending email. Alert on backup coverage before the write cutoff, disk pressure, repeated job/provider failures and update failure. A failed optional provider should degrade that reading, not falsely mark the entire server dead. Keep the component count small enough that one person can patch, restore and understand it.

## 6. Migration: vertical replacements with deletion budgets

Research changes only this document. The following are ordered implementation slices, each reviewed separately. No big-bang rewrite and no permanent old/new mode flag. For each slice, measure whole affected **handwritten production** LOC/characters before/after, generated output, tests/docs and total tracked-tree footprint separately; relocation is not shrinkage. A new manifest/test may add a file, so offset it with proven obsolete files in that same slice if the literal tree must hold. If a necessary safe slice cannot satisfy the reset budget, stop for a ruling rather than hide growth in generated code or drop tests.

| Order / coherent cut | Reuse and replacement | Delete in the same accepted slice; preservation gate |
|---|---|---|
| 1. Fix the target entry and contract boundary | Keep hosted native construction, commandcatalog, service and existing transport tests. Inventory required locked-screen operations/exposure using existing generator metadata; integrate client timing into existing diagnostics. | Remove traced duplicate native wrapper/adapter doors as each matching workflow replaces them, including obsolete generated bindings. No deletion merely because grep found no caller. Gate: required native locked screens and CLI/MCP still use the same service and authorization. Hold or shrink production; no new server framework. |
| 2. Replace desktop acquisition ownership | Relocate existing collector/preparation lifecycle into server composition, one provider/scope at a time. Publish validated evidence to the current hosted read path; use temporary computation only where required. | Remove corresponding desktop startup workers, launch-refresh dependencies and parallel manual pinned-evidence provisioning. Keep reproducible pinned-input tests. Gate: real fixture ladder/import parity, restart/correction behavior, no upstream calls from the native client. Complete server recovery-capacity test here before expanding live evidence. |
| 3. Consolidate frontend reads | Extend query.ts for session-scoped retained readings, dependency revisions and one stream subscription, replacing Home/HQ/Pulse/War Room owners incrementally. Make Pool refresh an explicit effect. | Remove per-hook result retention, duplicate lineup/personal maps and replaced invalidation subscriptions; finally remove both hidden War Room mounts. Retain only ephemeral edit/scroll/dialog state. Gate: 200/100 ms measured navigation, no hidden refresh load, no private/stale actionable data after reconnect/revoke. |
| 4. Complete direct browser delivery | Reuse React assets/contract and hosted sessions/admission; replace native-only request authentication assumptions with explicit browser/native adapters, add static manifest/SW delivery. | Remove replaced duplicate frontend transport declarations/DTO lists and native-only screen forks; do not delete the necessary OS credential adapter. Offset genuinely new PWA files against proven redundant delivery code. Gate: one direct-browser locked screen, CSRF/revocation, stream reconnection, offline/static-version behavior, then remaining screen parity. |
| 5. Retire the local backend runtime | Use existing league transfer/import, one native settings/plan path and the now-complete hosted service. Validate migration on consistent copies; user authorizes authority cutover and target account/league mappings. | Remove local-App construction, three-DB client opens/migrations, local server/due/feed/engine lifecycle and obsolete desktop-only bindings. Remove MFL mirror navigation/duplicate native-like authority UI, retaining the thin reference/import adapter and required rule math. Gate: ledger/checkpoint/settings/private-plan parity and fenced old writer; no source-based league branches. Old encrypted recovery artifacts are retained per policy, not live code paths. |
| 6. Replace manual operations, close public packaging seam | Extend existing install, backup, restore, release trust, version and status mechanisms into guided unattended jobs. One server payload, packaging-only CasaOS/Proxmox adapters and later Flatpak client. | Remove superseded manual orchestration and duplicated status/update scripts after disposable-host equivalence tests. Gate: restart/update fault injection and measured off-host RPO/RTO. Then update operator docs to the actual automated path. No real-owner admission until all prior gates pass. |

Exact dead-file/LOC savings are **not established** by this research; the inventory's static consumer list is a starting point, not deletion authority. Any slice that only moves files holds code size, not shrinks it. Existing caller/route traces and feature tests decide which wrappers can actually disappear. Do not delete `modelrun`, transaction rules, archive evidence or privacy-isolation copies just because another representation exists.

## 7. Risks and rulings for Christopher (seven)

1. **Standalone meaning:** approve thin desktop/PWA/CLI only, with solo use requiring the same separately installed server; no embedded/offline authority. This is the main architecture ruling.
2. **Browser trust boundary:** approve same-origin UI + cookie/CSRF sessions alongside existing native/CLI credentials, then require head-brain/security review. Current API is deliberately not browser-ready; no real-owner exposure until proved.
3. **Speed-law test conditions and scale:** which owner geography/network/device and how many simultaneously hosted leagues must meet 200 ms screens / 100 ms confirmed clicks? Keep limits unchanged; explicit conditions distinguish a real breach from an impossible universal WAN promise. Current 250 ms stream polling cannot prove sub-100 ms cross-client updates.
4. **Recovery capacity and availability:** authorize a measured backup capacity/retention revision if full evidence exceeds current 512 MiB/128 MiB limits; retain fail-closed coverage admission even when it pauses league actions during an R2 outage? Recommendation: retain it to honor the fixed RPO. Need actual laptop memory/disk/uplink and an available spare/restore-key-holder for the four-hour drill.
5. **Authority/document conflict at cutover:** newer brief makes native ledger authoritative and MFL thin reference; older `CLAUDE.md` Hard Constraints/Core Plan still say MFL wins. Also confirm how existing privileged TOTP requirements fit optional TOTP: optional for owners only, or for commissioners too? Do not silently remove either protection or overwrite rule in implementation; reconcile governing documents before cutover.
6. **Migration scope:** which existing leagues/accounts/private plans must transfer, and what evidence retention is mandatory for replay/corrections? Approve a copied-profile acceptance dataset and an authorized, fenced cutover. No invented mappings, automatic grant of commissioner power or discarded provenance.
7. **Zero-manual-operations boundary:** approve guided one-time credential/key custody and explicit destructive restore authorization as unavoidable human inputs; routine startup/feeds/backup/update/health require no console work. Resolve any literal total-tree budget conflict with necessary PWA/security tests openly rather than disguising it as debloat.

## References

Repository evidence: [inventory §§1–6 and appendices](INVENTORY-2026-11-05.md); `internal/composition/run.go:runNative`; `internal/serverhttp/hosted.go`; `internal/serverhttp/invalidations.go`; `internal/serverclient/invalidations.go`; `cmd/twr/README.md`; `deploy/server/BACKUPS.md`, `STORAGE.md`, `UPGRADE.md`, `RESTORE.md`; `SYSTEM_MAP.md`; `docs/standards/quality-contract.md`, `architecture-review.md`.

External technical references (verify pinned versions/terms during implementation):

- SQLite WAL concurrency/checkpoint model: https://sqlite.org/wal.html
- SQLite appropriate server-side uses and writer limits: https://sqlite.org/whentouse.html
- SQLite consistent online backup: https://sqlite.org/backup.html and https://sqlite.org/lang_vacuum.html#vacuuminto
- Litestream recovery model/configuration, for evaluation rather than assumed encryption parity: https://litestream.io/how-it-works/ and https://litestream.io/reference/config/
- R2 S3 compatibility and billing: https://developers.cloudflare.com/r2/api/s3/api/ and https://developers.cloudflare.com/r2/pricing/
- age encryption format/implementation: https://github.com/FiloSottile/age
- SSE reconnection, credentials and connection limits: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
- Browser CSRF controls: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
- Session-cookie security: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- Service-worker lifecycle/update behavior: https://web.dev/articles/service-worker-lifecycle
- TUF release trust and rollback protection: https://theupdateframework.io/overview/
