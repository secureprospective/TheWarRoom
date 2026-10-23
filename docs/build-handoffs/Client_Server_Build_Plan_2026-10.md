# Client/server build plan — October 2026

Plan only. Source baseline: `d75929f6b768b249a718e591917987396a84c0de`, inspected 2026-10-23.
No server behavior, security test or deployment drill was exercised for this document.
This reshapes hardening plan v3 block C. It does not reopen earlier rulings or declare block B complete.

## Binding decisions

| Area | Build to this decision |
|---|---|
| Purpose | One native league, about 12 owners. Owners reach Christopher's homelab before July 2027. No scope cuts. |
| Runtime | Separate `cmd/twr-server`, same engine. Server SQLite, WAL, one writer. Ledger and pinned evidence are authoritative. Replay plus disposable in-memory cache. No `native.db` split. Stored snapshots need a 100k-event benchmark and a further decision. |
| Transport | HTTPS + JSON; recipient-filtered SSE invalidations, never raw ledger broadcasts. Protocol and durable command versions are separate. Server owns admission time and due processing. |
| Accounts | Personally handed, commissioner-issued, one-use 48h enrollment codes. Username and Argon2id password. Commissioner and all co-commissioners require TOTP. Account principal is `acct.<UUID>`. Per-league roles; verified legacy aliases; history never rewritten. |
| Credentials | Hashed opaque device sessions; keychain on desktop; 7-day idle/30-day absolute expiry. Immediate revocation includes queued writes and SSE. Eight printed recovery codes. Audited console recovery needs the offline key. Commissioners reassign seats, never reset accounts. |
| Agents | Separate scoped tokens: league, franchise, command allowlist, read/preview/execute. Read/preview 30 days; execute 24 hours. Current issuer rights intersect the issuance ceiling and token scope on every path and inside append. |
| Exposure | Outbound Cloudflare Tunnel, no port forward. CGNAT and no public IPv6 rule out direct 443. Christopher accepts Cloudflare seeing plaintext passwords, TOTP and hidden bids. Admin is WireGuard/LAN only. |
| Host | One light Proxmox CT, systemd, TLS proxy, isolated VLAN and Proxmox firewall. Deny CT access to LAN/management over IPv4 and IPv6. LUKS backing storage, console unlock remains the approved baseline pending the open ruling below. |
| Safety | Go at least 1.26.9; pinned CI actions; bounded requests and hashing; CSP; MFL destination checks and body caps; 0600 files. Sandboxed service. Hash-chained ledger/security audit, off-host anchors. Encrypted off-host backups. VPN-only dashboard, no outbound alerts. Incident runbook. |
| Delivery | Free TUF-based update integrity; no Apple Developer or Authenticode purchase. First install has unsigned-app OS warnings; guide and clean-machine tests cover them. Passkeys remain later, gated by three-OS enrollment/recovery evidence. |
| Recovery | RPO 15 minutes, RTO 4 hours including host replacement and unlock. Christopher's wife is deputy key holder, with a printed sealed copy. Anchors detect loss; they cannot recover missing events. |
| Dates | Build freeze May 28 2027. Invitation decision June 25; enrollment by June 30. June is for drills, not planned feature construction. |
| Gate 5 | Three-day Pi Durable prototype after B. Overnight goals on Claude-OS first. Explore repair-to-test-to-PR, CLI agents and fleet dispatch. Outside the app runtime; an agent never merges. |

Inputs: the four approved panels' `engineer.out` and `engineer.r2.out`, including their round-two corrections, in fleet runs `panel-warroom-gate1-architecture-latest`, `panel-warroom-gate2-auth-latest`, `panel-warroom-gate3-security-latest`, and `panel-warroom-gate4-rollout-latest`. Christopher's subsequent rulings override recommendations: Tunnel replaces direct forwarding; 15-minute RPO replaces five minutes; free signing replaces paid platform signing. Gate 3 F7 supplies the eight blockers below. Panel source locations describe the old fast-push tree; the evidence below is rechecked against this baseline.

## Where the code is today

Locations are repo-relative, with baseline line numbers. “Missing” means the inspected tracked tree lacks the ruled implementation, not that a similarly named file proves absence.

| Ruled piece | Source evidence today | Remaining work |
|---|---|---|
| Shared engine and headless entry | `main.go:21–24` enters composition; `internal/composition/run.go:47` starts Wails. `internal/commissioner/ledger.go:70–77` replays records. `internal/commissioner/command.go:53–59` defines transport-neutral Session and Authorizer. `internal/nativecheck` and `internal/engine` are indexed in `SYSTEM_MAP.md`. | `git ls-files cmd` lists only fit and fixtures. No `cmd/twr-server`. Reuse the domain engine and history store, not Wails composition or a second engine. |
| SQLite and ledger | `internal/db/pools.go:24,40–67` has FULL/FK/recursive-trigger pragmas, pre-open version refusal, WAL and one writer. `internal/store/history/events.go:18–40` has revision/request constraints. `internal/store/history/commissioner.go:104–147` owns the append transaction and commit. | Host the existing store behind a narrow service. Do not split the database. Server credential admission is missing. Existing decoded-record caching is described below; it is not a hosted projection cache. |
| Replay/cache/evidence | `internal/store/history/native_assets.go:11–24` reads transaction records. `internal/store/history/commissioner.go:44–83` already caches decoded records by revision and fetches only the tail. `internal/commissioner/ledger.go:70–74` already uses closed books when available. `internal/store/history/playerstats.go:150–181` resolves held evidence and caches decoded weeks. `internal/composition/commissioner_host.go:57–69` gets player facts from desktop state. | Server-held, digest-pinned facts, scoring evidence and provenance. Complete pinned-result work from B/C; compare after source archives disappear. Reuse the decoded-record cache; assess bounded memory projections and measure 100k events. Preserve existing frozen rollover books as domain history, not permission to add disposable persisted snapshots. |
| Catalog | `internal/app/catalog/command.go:4–10` explicitly says Unset is not publishable. `internal/app/commissioner/api.go:159–160` builds entries with unset IDs/versions. `internal/commissioner/command.go:62–90` contains gravity definitions. `internal/commissioner/ledger.go:24` embeds RawCommand. | Catalog argument schemas, versions, permissions, modes, undo/facts metadata, durable upcasters, generated clients and one dispatch path. Current metadata is not permission to expose methods. |
| Authorization | `internal/store/history/commissioner.go:248–265` still bypasses the injected Authorizer for trade/owner-waiver/owner-bid families and nil auth. `internal/commissioner/command.go:59` lacks argument-aware scope. `internal/app/commissioner/commissioner_api.go:20–24` redirects owner mutations to special doors. | Unconditional credential AND domain checks, including retry, read, preview and append; no hosted nil-auth fallback. Collapse special doors without losing legality checks. |
| People and identity | `internal/governance/people.go:52–68` permits one co-commissioner. `internal/composition/commissioner_host.go:23–35` still hashes installation and UID. | Immutable accounts; league aliases applied to grants, ownership, privacy and retries; multiple deputies and ordered succession/tie handling. Host admin gains no league role. |
| Clock | `internal/store/history/commissioner.go:106–123` samples the store clock after BeginTx, truncating to microseconds. `internal/commissioner/bid_execute.go:333–338` already rejects at/after close. `internal/composition/native_bid_clock_test.go:284` contains a deadline regression. `internal/composition/native_schedule_app.go:99–115` still runs a desktop one-second ticker. | Preserve the fixed local deadline behavior. Move scheduling to server, use one admission instant, test queued writes, clock health, restart catch-up and lost wakes. Test nanosecond boundaries explicitly; current truncation needs checking. |
| Bid privacy | `internal/commissioner/bid_reading.go:123` exposes total record count as Revision. `internal/commissioner/ledger.go:331–339` filters ledger visibility. | Per-view concurrency tokens and recipient-specific invalidations. Filtering bodies alone does not hide activity counts, errors or cursors. |
| Accounts, sessions, tokens, HTTP/SSE | Session is only the principal port above. Tracked production Go search for Argon2id, TOTP, `text/event-stream`, `ListenAndServe` and `HandleFunc` found no implementation. Tracked path search found no hosted account/session/token service. | Enrollment, credential storage, TOTP, recovery, revocable sessions, token policy, HTTPS API and SSE remain to build. Do not mistake league seat invites for account enrollment. |
| Local backups and migrations | `internal/db/backups.go:23–32,55–92` builds a fixed three-file local backup set. `internal/db/backup_restore.go:41–96` implements restoration. `internal/db/pools.go:40–51` already refuses newer files before opening the writer. | Adapt matched-set capture to the actual server-owned files plus auth/config/evidence. Encrypted off-physical-host coverage, fresh auth epoch, key custody and timed spare-host restore are missing. Do not repeat obsolete panel findings as open defects. |
| Desktop/platform | `internal/composition/instance_lock_windows.go:15–28` already uses LockFileEx; Unix twin exists. `.github/workflows/verify.yml:23–32` pins actions by SHA. `frontend/package.json:29–30` is already Vite 8/Vitest 5. `go.mod:3` still declares Go 1.26.4. | Three-OS packaged install/update/enroll proof, TUF updater and signed release configuration. Existing locking is not proof of packaged readiness. Patch the build toolchain and module floor. |
| Input/host defenses | `internal/mfl/client.go:303–338` checks destinations/redirects and caps bodies. `internal/ingestion/leaguearchive/disk.go:25–72,83–100` has Lstat, handle checks and byte/file/entry budgets, not race-safe opens. No tracked `.service`, cloudflared or TUF deployment files were found. | Preserve existing input fixes; exercise hostile paths. Before server accepts other people's archive uploads/folders, add race-safe opening to this same guard. CT/VLAN/Tunnel, sandbox, encryption, audit anchors and operations remain unproved. |

These are source-traced observations, not passing feature claims. New tests named below are acceptance targets, not existing tests.

## Intended flow and PR rules

Desktop Go holds secrets and performs HTTPS/SSE. The webview never receives a bearer.
Validated Raw request + authenticated credential → catalog dispatch → current league/franchise policy → transaction admission → shared engine → ledger/evidence commit → authorized reply and view invalidation.
Inside the transaction, recheck credential revocation, scope, current grants, expected revision and server deadline. Denial or persistence failure appends nothing. Publish cache changes and invalidations only after commit. Retried canonical requests need current credential admission, then return their original receipt. Conflicts ask the owner to reread; never silently rebase.

Server evidence ingestion is keyless reference ingestion or commissioner import. No owner MFL keys cross to the server. The desktop MFL mirror remains reference/local planning, never a native authority overwrite. Hosted offline mode allows stale marked reads, never local execution.

Each row below is one PR and one bounded Sol run, estimated at no more than two engineer-days. If its stated acceptance will not fit, split before dispatch; do not silently omit failures. Use targeted race tests for touched packages. Each PR receives normal hooks and full race CI. Keep v3 block-level integrated live gates and Christopher clicks; do not add a full manual verify per item. Review corrections use the same final candidate. New packages require placement/map entries in that PR. Paths marked **new** are proposed homes, not existing code.

### Slices in build order

Lanes: **S** shared/server integration; **C** catalog/client; **A** accounts; **O** operations/delivery; **X** isolated prototype. IDs in “Needs” are merge dependencies. Estimates include tests and review fixes, not queue time.

| ID / lane / days | Goal and files/packages | Test that proves the slice | Needs |
|---|---|---|---|
| 01 S / 2 | Thinnest vertical: new `cmd/twr-server`, `internal/leagueservice`, `internal/serverhttp`, `internal/serverclient`. Serve one fixture-backed native read computed by the shared commissioner engine over loopback HTTPS JSON, protocol v1. Include a local Go client invocation. No writes or public bind. | Start real binary without Wails/display, use trusted test CA client, compare JSON with direct engine read; bad cert/unsupported major rejected. | — |
| 02 C / 2 | Finish native catalog schema/version/permission/mode metadata in `internal/app/catalog` and feature declarations. Reject unset publication. | Enumerate native reads and mutation families; every publishable entry complete; deliberately unset entry refused. | — |
| 03 C / 2 | Durable command version envelope and in-memory upcasters in `internal/commissioner`, history decoding. | Old recorded fixtures replay identically; unknown newer command/event versions refuse without changing file bytes. | 02 |
| 04 C / 2 | Generate typed client commands, capability rendering and plain-word grammar from catalog; frontend commands and generator. | Go/client schema parity; no hand-coded privilege upgrade; existing confirmation/reason behavior retained. | 02 |
| 05 S / 2 | Move native reads/previews behind the shared service; adapt Wails commissioner APIs. Register only completed catalog entries. | Wails/service return identical authorized readings and previews; invalid Raw payload rejected; preview writes zero events. | 01–03 |
| 06 S / 2 | One mutation path for generic, trade, bid and waiver doors; preserve fact/gate/legality checks. Service, commissioner APIs, history append. | Each family through old adapter and new dispatch has identical receipt/refusal and state; duplicate request appends once. | 05 |
| 07 S / 2 | Unconditional argument-aware credential admission inside transaction, separate from domain Authorizer; no hosted nil fallback. | Queue write then revoke/change role; every family, preview/read/retry rejects; zero new events; cross-franchise references denied. | 06 |
| 08 S / 2 | Add one HTTPS mutation through that path, expected revision and server-owned admission instant. `serverhttp`, history append. Loopback fixture authority only. | Real client write/read/reopen; concurrent expected revisions have one winner; deadline−1ns/at/+1ns for place/raise/withdraw, including queued writes and stopped worker. | 04,07 |
| 09 S / 2 | Server due worker: nearest deadline, commit wake, startup catch-up and limited system principal; replace hosted desktop ticker. New `internal/leagueservice` worker. | Lost wake, restart and award/bid race process once; late claim refused; clock-untrusted state suspends timed mutations. | 08 |
| 10 S / 2 | Pin player facts, identities and schedule evidence on server via existing history/nativecheck ports. | Same command rejects changed/missing digest; desktop key never required; invalid import preserves prior state. | 05 |
| 11 S / 2 | Complete pinned stat evidence/native scoring, including distance bonuses and kicker fallback. History/fantasyscore. | Results unchanged after archived duplicate bodies disappear; explicit distance/kicker cases; frozen-result digest parity. | 10 |
| 12 S / 2 | Bounded revision-keyed memory projection cache; no stored snapshots. Service and replay benchmarks. | Cold replay equals warm cache after corrections/rollover/restart; 100k-event benchmark records memory, p95 reads and restart time. | 09,11 |
| 13 A / 2 | League-scoped canonical account principals and verified alias resolution in governance/commissioner. No history edits. | Alias grants, private reads, owner comparisons and retries agree; conflicting/cross-league aliases denied; original bytes unchanged. | 07 |
| 14 A / 2 | Multiple co-commissioners and explicit ordered succession/appeal/tie handling; governance state/folds. | Two deputies coexist; remove one without removing another; deterministic tie/succession and commissioner-only restrictions replay. | 13; ruling Q2 |
| 15 A / 2 | New `internal/store/accounts`, `internal/accounts`: immutable account IDs, username/password storage and local bootstrap. Argon2id 64MiB/t3/p1, salt16/hash32. | Unique usernames, malformed hashes, correct/incorrect passwords; no password in ledger/log; host bootstrap has no implicit league role. | 13 |
| 16 A / 2 | Personally handed enrollment codes, hash only, one use, 48h expiry; separate account enrollment from seat grant. Accounts/store and auth routes. | Concurrent redemption succeeds once; exact expiry denied; wrong identity/occupied seat does not silently replace owner. | 14,15 |
| 17 A / 2 | Privileged TOTP enrollment, encrypted seeds, replay protection and recent-auth checks. Accounts/store. | Commissioner/deputy cannot act before setup; reused code fails; five-minute step-up enforced; seed absent from logs. | 16 |
| 18 A / 2 | Eight printed one-time recovery codes; audited offline-key console recovery. Accounts/store, server console command. | Concurrent recovery code use succeeds once; reset revokes delegations; commissioner seat reassignment cannot reset another account. | 17 |
| 19 A / 2 | Opaque hashed per-device sessions, idle/absolute expiry and fresh auth epoch. Accounts/store and credential admission. | 7d/30d boundaries; active usage never extends absolute expiry; revoke queued write before append; no plaintext bearer at rest. | 18 |
| 20 S / 2 | Wire account/session admission to every hosted read, preview, execute and retry. Remove fixture authority from deployable mode. | Real HTTPS enrollment/login/write; spoofed principal ignored; missing/nil credential fails closed on every route. | 19,08 |
| 21 C / 2 | Desktop native-only startup, join/login/privileged TOTP/recovery guide and screens. Feature API/transport, frontend control/governed. | No MFL setup needed; owner reaches assigned franchise; expired invite and lost factors have usable failure paths. | 04,20 |
| 22 C / 2 | Desktop device session keychain, device/all-device revoke and memory-only fallback. New `internal/serverclient` credential adapter. | Packaged keychain roundtrip on three OSes; unavailable keychain writes no plaintext; revoked device loses native access. | 21 |
| 23 S / 2 | Recipient-safe per-view concurrency tokens. Commissioner readings, history read adapters and service. Remove bid ledger-count leak. | Insert another owner's hidden bid: unauthorized bodies/tokens/errors remain identical; visible changes still conflict correctly. | 12,20 |
| 24 S / 2 | SSE post-commit invalidations with bounded queues, reconnect/reread and immediate revoke. Serverhttp/client. | No raw ledger or hidden-activity cursor; dropped connection rereads safely; revoke closes SSE ≤1s; rollback emits nothing. | 22,23 |
| 25 A / 2 | Scoped token issuance/storage, allowlists, modes, expiry, issuance ceiling and current issuer rights. Accounts/store. | Read/preview versus execute lifetimes; wrong league/franchise/command denied; role loss shrinks scope; tokens cannot administer credentials/roles. | 19,20 |
| 26 S / 2 | `twr` CLI/AI transport through same dispatch, token audit ID, human confirmations and preview default. New `cmd/twr`. | Each command family on CLI/HTTPS/legacy adapter enforces token scope at append; revoked token retry denied; no bearer logging. | 24,25 |
| 27 O / 1 | Close B toolchain gap and record remaining B acceptance, not a second rewrite. `go.mod`, CI/build scripts, hardening evidence. | Patched Go ≥1.26.9; govulncheck; existing migration/wrong-state/shutdown/backup/input tests and B live/restore gate on same build. | — |
| 28 X / 2 | Pi Durable overnight-goal prototype on Claude-OS, outside runtime. `docs/experiments` prototype/runbook; any later execution uses isolated disposable workspace. | Crash/restart resumes a bounded goal; no app credentials, runtime import or merge permission. Record observed result, not source-only confidence. | 27; B accepted |
| 29 X / 1 | Finish three-day prototype: failing-test → fix → PR demonstration; assess CLI/fleet follow-ups. `docs/experiments`. | Human must merge; no autonomous deploy. Report restart evidence and stop/delete instructions. | 28 |
| 30 O / 2 | HTTP resource limits, KDF semaphore/throttles, proxy trust and secret redaction. Serverhttp/accounts. | 100 logins: ≤2 active hashes/4 queued; bounded memory, command p95<1s, hash p95≤750ms measured on target CT; spoofed forwarding headers fail. | 20 |
| 31 O / 2 | Complete input regression coverage: MFL host/body caps, CSP/Origin rejection, server archive race-safe open on same DiskFS. | Redirect/host tricks, decompressed oversize, injected scripts and symlink-swap fail; sentinel secrets absent; prior import unchanged on failure. | 10,30 |
| 32 O / 2 | Transactional ledger/security hash chain with trusted existing-history baseline. History append and account audit. | Edit/delete/truncate detected; privileged mutation rolls back if audit fails; old receipts/history remain intact. | 18,23 |
| 33 O / 2 | Off-host append-only anchors each minute; audit verifier and operator status. New `scripts/server` plus service audit export. | Change anchored head/tail detected independently off-host; outage shown; documented unanchored tail limitation. | 32 |
| 34 O / 2 | CT deployment templates: systemd sandbox, TLS proxy, cloudflared, VLAN/firewall and admin separation. New `deploy/server`, docs. | Disposable deployment boots; state-only writes; forbidden filesystem/network actions fail; no public admin; IPv4/IPv6 lateral access blocked. | 27,30 |
| 35 O / 2 | Prove ruled disk encryption/unlock on chosen CT/VM host, permissions and systemd credential delivery. Deploy/runbook. | Cold reboot and deputy unlock; locked storage inaccessible; DB/WAL/SHM/log 0600; swap covered; no claim against root/running compromise. | 34; ruling Q1 |
| 36 O / 2 | Encrypted matched-set off-host backups and retention, adapt existing db backup API to hosted state/evidence/auth/config. | Loss of Proxmox still leaves verified recoverable state ≤15m old; fail closed before coverage breaches 15m; private restore key never on CT. | 33,35 |
| 37 O / 2 | Spare-host restore, auth epoch reset and incident runbook. Backup/console tooling and docs. | Restore after simulated host loss within 4h including unlock; consumed recovery codes/stale sessions cannot revive; compare evidence/replay digests. | 36 |
| 38 O / 2 | VPN-only dashboard, backup/anchor/clock/disk/Tunnel/HTTPS health and daily/away checklist. Server status/deploy/docs. | Fault injection appears on VPN dashboard; public status leaks no sensitive state; no outbound notification; deputy executes checklist. | 37 |
| 39 S / 2 | Versioned encrypted migration export with events, evidence/provenance/identity mappings, source epoch and projection digests. New `internal/leaguetransfer`, CLI. | Missing blob/bad digest rejected; frozen-time roster/cap/lineup/bid/result/standings parity; no copied schema marker becomes target authority. | 11,13,26 |
| 40 S / 2 | Inactive idempotent import, preserving IDs/revisions/receipts without re-executing commands. Transfer/history. Race-safe archive guard required. | Repeat same league/epoch/content returns original receipt; conflict rejected; crash injection leaves no active partial league. | 31,39 |
| 41 S / 2 | Explicit source seal, target activation and fenced reverse transfer. Transfer/service/desktop authority labels. | Crash at each boundary never yields two writable authorities; pre-activation abort unseals source; post-activation reverse transfer includes every server event. | 40 |
| 42 O / 2 | Server-first upgrade, read-only preflight, matched backup/migrate/readiness, protocol major + current/previous minor and security floor. Serverhttp/deploy. | Unsupported old/new protocol/command/schema refused before native access, even without SSE; newer file untouched; binary-only downgrade refused. | 37,41 |
| 43 O / 2 | Free TUF trust bootstrap, release artifacts/config signing and offline key ceremony. New `scripts/release` and delivery docs. | Tamper/expiry/rollback metadata rejected; threshold root rotation works; clients get public hostname/trust only, no tunnel secret or brittle leaf pin. | 27 |
| 44 C / 2 | Desktop staged TUF updater and clear update-required UX. Serverclient/platform API and frontend. | Valid staged update launches; corruption/expired metadata/downgrade writes no native data; interrupted update recovers. | 42,43 |
| 45 O / 2 | Windows/macOS/Linux package CI on inventoried owner OS/CPU targets; reuse existing platform locks. Workflow/build scripts. | Clean-machine install/launch; lock contention and crash release; WebView dependencies and unsigned-app warning guide exercised on all three OSes. | 43 |
| 46 C / 2 | Hosted lineup, roster/IR/taxi and draft workflows through generated transport; local stale mode. Commissioner/draftroom/governed frontend. | Owner sets lineup, manages eligible reserves, drafts and reconnects; no local authoritative mutation offline. | 24,41 |
| 47 C / 2 | Hosted bids, waivers, trades, results/standings and rollover workflows. Commissioner/governed frontend. | Fake league walks each workflow; private bids stay private; correction/rollover and replay agree; no duplicate retry action. | 46 |
| 48 C / 2 | MFL-side command catalog/transport for solo-owner pools and bid plans. Offers/control and frontend commands. | Same catalog permissions/confirmations as bar/CLI; operations remain local plans; zero MFL submissions or server-held MFL keys. | 26,47 |
| 49 C / 2 | Finish Block E user workflow and explicit MFL/reference versus hosted-native labels. Offers/control/startup frontend. | Refresh changes only MFL reference; cannot replace native state; disconnected native reads marked stale; Christopher E live walk passes. | 48 |
| 50 S / 2 | Automate release blocker authorization/privacy/deadline matrix (RB2–4,8). Existing service/HTTP integration tests. | Every catalog family, two leagues, roles/token modes and all adapters; denied state unchanged; privacy transcripts identical; exact boundary/catch-up/revoke cases. | 29,38,42,47 |
| 51 O / 2 | Release blocker resource/input/deployment/update matrix (RB1,5–7). Staging harness and recorded evidence. | External Tunnel/isolation checks, flood/slow-body, secret/CSP, tampered updates, sandbox and restored epoch/anchor checks on candidate. | 44,45,50 |
| 52 C / 2 | Three-OS novice enrollment/device recovery/update and full 12-owner synthetic season acceptance; final fixes and freeze record. Tests/docs only unless a failing case needs a bounded fix PR. | Each OS install warning→enroll→franchise ≤10m; 24 sessions reconnect; full draft/season; all RBs green on exact final build; Christopher freeze approval. | 49,51 |

01 is deliberately a real HTTPS client/server path, not a health endpoint. It is fixture-only and loopback-only until catalog and policy gates pass. 08 is also a closed development path, not permission to expose writes. No network command goes live before 02–07 pass. No real owner connects before all eight release blockers and the invitation gate pass.

## Safe parallel dispatch

Lane letters describe responsibility, not blanket permission to run all rows in a lane concurrently.
Only the following bounded pairs are pre-approved for two worktrees. Merge prerequisites first.

| Pair | Non-overlapping file ownership during the pair |
|---|---|
| 01 S + 02 C | New server/service/client packages versus catalog and existing feature declarations. 01 uses a fixed test read, not catalog declarations. |
| 04 C + 06 S | Frontend generator/commands versus Go mutation adapters/history. Freeze the catalog contract after 02; no regenerated bindings in either until integration. |
| 09 S + 15 A | Worker/service files versus new accounts/store files. 15 consumes the merged admission/principal contract; does not edit it. |
| 21 C + 23 S | Desktop enrollment/frontend files versus commissioner read-token files. No shared service wiring edits. |
| 24 S + 25 A | SSE HTTP/client files versus token accounts/store files. Shared admission wiring belongs to 26 after both merge. |
| 28 X + 30 O | Experiment docs versus HTTP/accounts hardening. No fleet or live Claude-OS changes in the planning PR. |
| 34 O + 39 S | Deployment templates/docs versus transfer exporter package/CLI subcommand. |
| 43 O + 46 C | Release trust/scripts versus owner workflow UI, after their separate dependencies merge. |

Reserve `SYSTEM_MAP.md`, placement, shared depguard rules and root composition wiring to one integrator at a time. Serialize those metadata edits before dispatching a pair, or merge one PR before rebasing the other. Do not call work parallel if both need the same file. No two verifies/push gates run together on Beelink. Two runs shorten calendar time, not engineer-day effort.

## Release blockers: exact acceptance before any owner connects

This is gate 3 engineer F7, adjusted only for the binding Tunnel ruling. Record candidate SHA, commands, measured results and reviewer for each. These are requirements, not reported passes.

| Blocker | Proof and slices |
|---|---|
| RB1 External exposure | Public hostname reaches owner HTTPS routes only. No inbound CT port-forward; prove origin/admin not reachable publicly. Test CT→LAN/management denial over IPv4/IPv6, valid TLS and renewal through the chosen proxy/Tunnel chain. Scanning a shared Cloudflare edge alone proves nothing about origin isolation. 34–35,51. |
| RB2 Enrollment/recovery/revocation | Enrollment expires at 48h; concurrent redemption once. Codes cannot replay; required TOTP replay-resistant; multiple deputies. 7d idle/30d absolute session limits, queued-command revocation and SSE close ≤1s. 14–24,50. |
| RB3 Authorization | Every catalog family × two leagues × roles/token modes × every applicable path, including legacy doors and references. Denial discloses nothing and appends zero events. 07,20,25–26,50. |
| RB4 Hidden bids | Hidden insert changes no unauthorized response, token/cursor, error, cache or SSE before disclosure. 23–24,50. |
| RB5 Exhaustion | 100 login flood: ≤2 active hashes, four queued, no OOM, ordinary command p95<1s. Oversize/slow requests end; forged proxy headers cannot evade throttles. Initial JSON 256KiB, seed 2MiB, headers 16KiB; header/body/command 5/10/5s. SSE two/device, 64 global, bounded queues, 10s write deadline. Measure on target resources. 30,51. |
| RB6 Hostile input/secrets | Bad MFL hosts/redirects/oversize fail; CSP blocks injection; browser Origins cannot mutate; sentinel secrets absent from app/proxy logs. Preserve validated Raw boundaries. Race-safe archive opening before accepting others' files. 31,51. |
| RB7 Delivery/storage integrity | Tampered/expired/downgraded updates and unsupported versions fail without writes. Valid update launches on three OSes using free integrity signatures. Verify sandbox/permissions, restored fresh auth epoch and detection of anchored audit edits/deletions. 32–45,51–52. |
| RB8 Time | Deadline−1ns/at/+1ns and queued writes: late place/raise/withdraw denied. Claims close without worker progress; restart catch-up processes once. 08–09,50. |

## Schedule and block gates

The 52 slices total **102 engineer-days**: fifty two-day rows plus two one-day rows.
This is approximately the approved 100-day build budget, not a measured completion forecast.
Add **25 contingency days** through May. Do not add the old v3 26–35 days or panel estimates again: overlapping catalog, native, account, security, packaging and MFL work is already here.
Row 27 is a closeout allocation, not an assertion that all B work takes one day. If baseline B is still incomplete, measure its remaining work and draw from contingency explicitly.

At four engineer-days/week, the approximately 31 weeks from Oct 26 through May 28 provide about **124 days**, versus **127 planned days including slack**. The original panel budget already strained capacity; this plan does not hide it. The bounded parallel pairs require at least three additional engineer-days of capacity to retain all 25 contingency days. Christopher must confirm that capacity or accept three fewer contingency days; no feature cut or borrowing June is proposed. Reforecast monthly using actual accepted PRs.

| Window | Build / contingency days | Slices and milestone |
|---|---:|---|
| Oct 26–30 2026 | 5 / 2 | 01,02,27; collect owner OS/CPU inventory and open rulings within closeout. First loopback HTTPS read. B live/restore acceptance before 28. |
| November | 15 / 3 | 03–07,10; 28–29 after B (15 total). Catalog and one-path foundation. Click B if not already accepted. |
| December | 14 / 3 | 08–09,11–15 (14). Shared-engine hosted read/write, clock, pinned evidence/cache. Click C engine/catalog foundation; no owner exposure. |
| January 2027 | 14 / 3 | 16–22. Invite/TOTP/recovery/session and desktop enrollment. Gate 2 click on integrated candidate. |
| February | 14 / 3 | 23–26,30–32. SSE/privacy/tokens/CLI and input/audit. Alias/evidence parity must be demonstrated before migration activation work. |
| March | 14 / 4 | 33–39. Deployment/encryption/backups/restore/dashboard/export. Approve actual deployment boundary and deputy cold-spare recovery. |
| April | 14 / 4 | 40–46. Cutover, versions, TUF, packages, owner workflows. Click D once three-OS update/install proof is complete. |
| May 1–28 | 12 / 3 | 47–52. Click E, integrated C native-season gate, all RBs; freeze May 28. |
| June 1–24 | 12 / 2 | Separate staging rehearsal below; no planned unfinished features. Two spare engineer-days remain within roughly four weeks at four days/week. |
| June 25–30 | 2 / — | Christopher cutover/invitation go/no-go June 25; personally hand codes and finish enrollment by June 30. |
| July | 0 planned build | Operate before draft. |

Monthly budgets are effort allocations, not promises that dependencies permit running numeric IDs in calendar order. Within a month follow “Needs”; use only the safe pairs above. If throughput forecasts beyond May 28, flag capacity immediately. Do not quietly reduce scope or defer safety work.

### June rehearsal and invitation gate

Use disposable staging DNS, keys, accounts and a fake 12-franchise league. Never rehearse on live files.

| Days | Drill and passing evidence |
|---:|---|
| 2 | Three-OS novice install/enroll/recover/update, including unsigned-app warnings and replacement device/keychain failure. Passkeys are not a launch dependency. |
| 2 | Full native draft/season, 24 session reconnect/retry load, hidden bids, clock boundary and scheduler stoppage. |
| 2 | Repeat and crash migration at every seal/import/activate boundary; parity and exactly-once receipts; old source remains retired. |
| 2 | Host loss and spare-host restore, ≤15m recoverable loss and ≤4h outage including deputy unlock; fresh auth epoch and anchored replay verified. |
| 2 | WAN/Tunnel loss, stale clients, update tamper, audit failure and incident key rotation/rebuild. |
| 2 | Re-run all eight RBs against frozen candidate and close drill findings. Any candidate change repeats affected checks and final integration. |

June 25 go requires B–E and gates 1–4 accepted; every RB green on the same release; clean three-OS workflows; explicit source retirement; alias approval; measured recovery; deputy key access and daily/away dashboard coverage. Christopher approves cutover and invitations. Before activation, abort target and unseal source if necessary. After accepting server events, forward-repair or fenced reverse transfer only: never reopen an old backup and discard new moves. Old disconnected desktop copies cannot be remotely fenced reliably; retiring their writable use is an explicit operator step.

## Questions requiring Christopher's ruling

| ID | Evidenced question | Needed before |
|---|---|---|
| Q1 | Keep console unlock, or approve unattended unlock and on which host form? Gate 3 approved console unlock; the Oct 23 brief explicitly reopens reboot usability. Recommended: backups always encrypted; VM data LUKS with proven TPM/Tang auto-unlock plus passphrase/deputy fallback; CT/LXC encryption at host level. Alpine clevis/OpenRC support is unverified and must be proved in a VM before any promise. No SQLCipher. No protection claim against host root or a running compromised server. This is one unlock decision, not a silent Alpine/runtime switch. | 35 |
| Q2 | What succession, appeals and tie-break ordering should multiple co-commissioners follow? `internal/governance/people.go:59–67` and the existing single-deputy folds cannot specify ordering for several people. Recommend an explicit league-maintained priority list; preserve commissioner-only powers. | 14 |
| Q3 | Confirm at least three extra engineer-days via the bounded second-run pairs, preserving the 25-day slack budget. The dated arithmetic above exceeds four-day/week capacity; it is not evidence that faster model execution removes engineering effort. | Staffing/reforecast now |

Operational inputs to collect, not new architecture questions: actual owner OS/CPU inventory; public hostname; reachable off-physical-host backup/anchor destination; spare-host access; and who checks the VPN dashboard when Christopher is away. Deputy key custody is already ruled. Record these in deployment/enrollment runbooks without secrets.
