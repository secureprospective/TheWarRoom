# TheWarRoom Hardening Plan — 2026-10

**Status (2026-10-19):** agreed with Christopher, not started. This is the one plan for the cleanup
week and the work after it. It replaces the separate store plan; the command findings stay as evidence
in `Command_Protocol_Audit_2026-10.md`.
**Why now:** real people are interested in running native leagues. This week is the cleanup before
any new direction.
**Order of priorities:** correctness, then efficiency, then robustness, for the MFL harness side and
the native league side.
**Runs alongside the ring work,** as small branches merged between ring phases. Only one push or
verify runs at a time on the Beelink.

## How every phase runs
- One `session/<phase>` branch per phase, cut from the latest `main`.
- Sol (`gpt-6.1-sol`, medium) builds from a brief file. Claude reviews every line and plants one
  mutant per report. Claude also runs `make verify` and the live gate on Claude-OS.
- Christopher clicks before any merge. Squash-merge only.
- Every fix ships with a test that fails without it.
- A decision that is hard to undo goes to the expert panel first (non-Claude seats, `panel-dispatch.sh`).

## Rulings so far (Christopher, 2026-10-19)
1. Real native leagues are likely, so native correctness is real work, not future-proofing.
2. This plan runs alongside the rings.
3. A client app plus a server install is the likely native future, but don't dig into it yet:
   - no `native.db` split
   - no off-machine backup design
   - every store and command change stays storage-agnostic and transport-neutral
   - the native ledger stays behind its store interface
4. Make tests faster rather than trimming them by size. A test is removed only with evidence that
   it covers nothing unique.
5. The CLI's local socket (an exception to "no inbound HTTP server") goes to the expert panel.
6. The CLI may come before the MFL-side commands. The MFL-side commands must exist before an owner
   other than Christopher uses the app, which will be near a real Alpha release.
7. Keep the command panel small: protocol shape and the socket question only. No client/server design
   yet ("no need to launch the space shuttle now").

## Where things stand
- **Clean sweep:** done on `session/code-sweep`.
  - ~1.6k dead code lines and ~8.8k stale doc lines removed.
  - `SYSTEM_MAP.md` brought up to date.
  - A flaky test fixed, and the Go test limit raised to 60 minutes.
  - Verify passed on `84ff9e24`, which is on GitHub.
  - Later commits are docs only: this plan and the command audit.
  - **Not merged; waiting for Christopher's go.**
- **Store layer:** mapped and reviewed. Findings F1–F15 below were checked in code or reproduced.
  Sol 6.1's review: `~/fleet/runs/sol61-warroom-store-review/stdout.log` (Beelink).
- **Command protocol:** audited. The native command engine is sound. Gaps G1–G9 are in
  `Command_Protocol_Audit_2026-10.md`.
- **Codebase picture:** `http://192.168.1.105:8765/` (CT105 page server; stop it when no longer needed).

## Store findings (verified)

The databases are three files in `~/.config/TheWarRoom/`:

| File | Size | Holds |
|---|---|---|
| `thewarroom.db` | 0.7 MB | MFL mirror, moves, params, rulebook, personal lists |
| `whatif.db` | 1.8 MB | what-if league |
| `history.db` | 375–490 MB | archive 224 MB, facts 222 MB, runs, and the native ledger (1.3 MB) |

| # | Finding | Where | Side |
|---|---|---|---|
| F1 | The mirror publishes new rosters before the disk write; after a failed write, a retry is skipped because the hash already moved | `state/mirror.go:111` | MFL |
| F2 | Refresh saves new rules before rosters are fetched; a failed fetch leaves new rules over old rosters | `refresh_app.go:94–105` | MFL |
| F3 | Foreign keys are never switched on; one broken link already exists in dev data (`model_runs` → missing `param_sets`) | `db/pools.go` | both |
| F4 | `INSERT OR REPLACE` deletes ledger rows without firing the no-delete trigger (reproduced); no code does this today | `history/events.go` | native |
| F5 | The ledger's JSON check accepts `{}` (reproduced); other JSON columns have no check | `history/events.go:28` | both |
| F6 | An approved stat line is silently ignored when no held versions exist | `commissioner/stat_ledger.go:94` | native |
| F7 | A rostered player with no PAID salary cell counts $0 cap, and the total is still used | `state/state.go:213` | MFL |
| F8 | Money parsing can overflow; percent settings accept NaN | `domain/money.go:55`, `rulebook/helpers.go:80` | both |
| F9 | Native scores depend on which archived copy of the stats file was read; losing or changing copies can change past results | `history/playerstats.go`, `fantasyscore/source` | native |
| F10 | Only `whatif.db` has migrations, and it runs DDL before its downgrade check; `history.db` and `thewarroom.db` have no schema version | `state/migrations.go`, `state/schema.go` | both |
| F11 | No routine backup or integrity check of any file | `app.go:289` | both |
| F12 | Every native read and command re-parses the whole ledger; one event is 772 KB | `history/native_assets.go:11` | native |
| F13 | Scoring a starter re-parses every held copy of the season stats CSV (2026: 24 fetches, 9 distinct, growing) | `history/playerstats.go:31` | native |
| F14 | Archive lookup is a substring scan over URLs | `history/archive.go:111` | both |
| F15 | Every what-if preview writes a full league copy to `whatif.db` and never cleans it up | `whatif_app.go:226`, `trade_projection.go:54` | MFL |

**Keep:**
- SQLite + WAL + one writer per file
- ledger revisions and retry keys
- corrections that never edit history
- integer cents
- the gzipped content-addressed archive
- what-if publish-after-success
- full frozen seeds
- the modernc driver

## The phases, in order

| # | Phase | Covers | Days | Gate |
|---|---|---|---|---|
| 0 | Land the sweep | merge `session/code-sweep` | — | Christopher's go |
| 1 | Impact map (read-only) | every phase below | 0.5 | one page per phase: touches, risks, tests; order and ETAs corrected |
| 2 | Faster tests | test time | 1 | same coverage; push verify about 10–15 min (today ~45) |
| 3 | Migration standard | F10, F3, F4 | 1 | old files upgrade; newer files refused untouched |
| 4 | Stop wrong state | F1, F2, F5–F8, F3 repair | 1–1.5 | each fix has a failing-first test; injected mirror-write failure changes nothing |
| 5 | Pinned native evidence | F9 | 1.5–2 | past native results unchanged after archive copies are deleted |
| 6 | Local backups and checks | F11 | 0.5 | restore on a clean profile reproduces league and native results |
| 7 | Fast reads | F13, F14, F15 | 1.5 | a native week scores within a stated budget; results unchanged |
| 8 | Command panel + catalog as data | G1, G6 | 3–3.5 | panel recommendation recorded; every command self-describes, with a version |
| 9 | One command path + reads as commands | G3, G5 | 2–4 | the six special IPC doors collapse into one; native readings are catalog reads |
| 10 | Frontend from the catalog + typed grammar | G2, G7 | 2 | the bar shows Go's real roles and gravity; `domain.verb key=value` parses |
| 11 | `twr` CLI + AI tools | — | 2–3 | CLI runs the bar's commands with the same confirmations; AI previews by default |
| 12 | MFL-side commands (solo owner first: free-agent pools and bids) | G4 | ring-sized | before any owner other than Christopher; near Alpha |

**Phase details**
- **1 Impact map.** Covers:
  - callers of every touched table and command
  - real data from backup-API copies only: FK violations, invalid JSON, cap-cell drift, what-if
    generation count
  - the upgrade path, both ways
  - screens whose readings change
  - tests that assume current behaviour
  - wider project: `SYSTEM_MAP.md`, the endpoint registry, `cmd/fit`, the ring roadmap
- **2 Faster tests.**
  - Today: 1,993 Go tests, ~74k lines.
  - The root package has 28k lines and takes 40 min under `-race`, about 80% of the wait.
  - Profile it, then share fixtures and stop rebuilding the app or database per test.
  - Measure coverage overlap last.
- **3 Migration standard.**
  - One runner per file using `user_version`: refuse newer → back up (`VACUUM INTO`) → migrate,
    step and marker in one transaction → integrity and FK check.
  - Every connection asserts `foreign_keys=ON`, `recursive_triggers=ON`, `synchronous=FULL`.
- **4 Stop wrong state.**
  - Persist the mirror, then publish it.
  - Stage refresh, then publish rules and rosters together.
  - NULL-safe JSON checks, added as migrations.
  - Approved stats are never silently ignored.
  - $0-cap drift marks the total invalid and blocks dependent what-ifs. The screens show it.
  - Checked money arithmetic; NaN and out-of-range percents rejected.
  - Repair the broken link from provenance; never invent a parent.
- **5 Pinned native evidence.**
  - Record the exact stat line, identity and source digest each native result used.
  - Make the initial baseline durable.
  - This is logic, not storage, so it carries over to a server.
- **6 Local backups.**
  - Backup-API copies daily and before upgrades, with a manifest.
  - Weekly `integrity_check` + `foreign_key_check`, shown in Data Sources.
- **7 Fast reads.**
  - Exact-URL archive index; bodies fetched by hash.
  - Parse each stats file once, cached by body hash.
  - Previews in memory; prune unpublished what-if copies.
  - Benchmark `observations` indexes before changing them.
- **8 Command panel + catalog.**
  - Small panel: protocol shape, versioning, and the CLI socket question.
  - Then each command declares in Go:
    - its argument schema and version
    - its permission and gravity
    - its undo class
    - the facts it needs
    - whether it reads, previews or executes
  - Stored commands get upcasters.

**Total:** about 7–8 days for phases 1–7 and 9–13 days for phases 8–11, plus phase 12.
- Calendar time is longer, because the work interleaves with the rings and waits on the push verify
  and Christopher's clicks.
- **Both sides are protected after phase 6.**

## Backlog (known, not scheduled)
- **Built but not wired** (`SYSTEM_MAP.md`):
  - ~20 native owner readings have no screen
  - the IR/taxi entry checks have no caller
  - three IPC methods are used only by tests
  - the empty `transaction_corrections` table
- **To verify:** whether play-by-play distance bonuses reach native scores. Only a test uses
  `ReadPlays`/`AttachPlays`. Check this with phase 5.
- **Ruling owed:** the bloat ratchet's small-file count blocked a cleanup, which conflicts with the
  standards ("blind size caps are not acceptance laws").
- **Polish:** about 200 TypeScript exports that nothing outside their file uses.
- **Not reviewed:** the `docs/ui` wireframes.

## Deferred to the client/server decision (panel when it opens)
- Accounts and sign-in, per-league roles, AI tokens (G8)
- A server-owned league clock for bids and waivers (G9)
- Native storage on a server; a separate native ledger store
- A heavy projection cache for native reads (F12); the full-replay cost is fine at today's sizes
- Off-machine backups
- Transport, version negotiation between clients and servers, live push to clients

## Plan v2: the deep audit folded in (2026-10-19)

Inputs: `Deep_Audit_2026-10-19.md`, `Audit_Recommendations_2026-10-19.md` (A1–A16), Sol 6.1 medium
review (`~/fleet/runs/sol61-warroom-audit-triage/stdout.log`, Beelink). Claude triaged Sol's points:

**Accepted from Sol:**
- Identity migration keeps the old principals through an alias, and never rewrites history.
- Recovery needs a proof of authority, not just a high gravity level.
- The deadline check is "at", not "after", and it covers withdrawals too.
- A downgrade must be refused before `db.Open` changes the file.
- Backups are restored as a matched set.
- A clock scheduler must cover catch-up and lost wake-ups.
- Owner screens are built as whole workflows, after phases 9–10.
- Platform work means packaged builds, not just compiling.

**Rejected:**
- A9's 8k-line target. The package is split only where profiling shows the tests cost time.

**Corrections to the audit:**
- Locks are nested in two places (`whatif_app.go:87`, `target_app.go:26/36`). No deadlock shown.
- Kicker code has three cases, not one. The missing baseline fallback still stands.

| Phase | Action | Days |
|---|---|---|
| 0 | Merge the sweep (Christopher's go) | 0 |
| 1 | Impact map + CPU profiles + a baseline UX walk + record the native-league boundary (ruling 8) | 1 |
| 1b | Go 1.26.9 + govulncheck in verify; CSP, MFL host pin, body limits, 0600 files; frontend tool upgrade; hooks lint (separate branches) | 1.5–2 |
| 2 | Faster tests: profile, share immutable fixtures, isolate mutable state, shuffled + race runs | 1.5–2 |
| 3 | Migration standard (F3, F4, F10) + refuse newer files before open + stored identity with alias + recovery design | 3–4 |
| 4 | Stop wrong state (F1, F2, F5–F8) + bid deadline (place/raise/withdraw) + screens read a published snapshot + clean shutdown | 3–4 |
| 5 | Pinned native evidence (F9) + distance bonuses + kicker fallback check | 2–3 |
| 6 | Backups as matched sets + integrity checks + a restore onto a different home folder keeps commissioner rights (F11) | 1–1.5 |
| 7 | Fast reads (F13–F15) + clock wakes when something is due + hourly work skipped when its inputs are unchanged + command latency benchmark | 3–4 |
| 8 | Panel + catalog as data (G1, G6) | 3–3.5 |
| 9 | One command path, reads as commands (G3, G5) | 3–4 |
| 10 | Frontend from the catalog + typed grammar (G2, G7) | 2–3 |
| 7b | Native owner workflows: lineup, bids, trades, waivers, results | 4–6 |
| 7c | Windows/macOS lock + packaged builds + CI (serialised) + install/update decision | 2–3 |
| 11 | `twr` CLI + AI tools | 2–3 |
| 12 | MFL-side commands, solo owner first (G4) | 3–5 |
| UX gate | Full click-through + failure states, then fixes | 1–1.5 |

**Gates:**
- **Safe for the MFL side:** after phase 6.
- **A real native league:** after phases 3–7, 9–10 and 7b.
- **Other owners install the app:** after 7c.

**Total:** about 37–50 working days of effort. Calendar time is longer.

## Plan v3: test in blocks (Christopher, 2026-10-19)

Replaces the phase-by-phase gating in v2. The work items stay the same; what changes is how they are tested and merged.

**Rules**
- **Work moves in blocks, not phases.** A block is one branch holding several items.
- **Items inside a block** get only their own targeted test (the one that fails without the fix), run locally in seconds.
- **No full gate per item.**
- **Each block gets, once:**
  - the full CI suite
  - one live gate on Claude-OS
  - one click from Christopher, or none if the block has no visible change
- **New tests extend existing table tests where one fits.** A new test file needs a new behaviour behind it.
- **Overlap is measured once** (per-test coverage) at the start of block B. Duplicates are removed with that evidence.
- **Sol builds the next block's items while Claude reviews the current block.**

| Block | Items | Gate (once) | Click | Days |
|---|---|---|---|---|
| **A: Fast gate** (now) | changed-only pre-push, parallel root tests, GitHub CI | push under 5 min, CI green, 3 shuffled race runs clean | none | 0.5 |
| **B: Safe today** (MFL side) | security quick wins (1b); test overlap cut and test speed (2); migrations, newer files refused, stored identity (3); wrong-state fixes, bid deadline, published snapshot, shutdown (4); backups and restore (6); idle CPU, hourly skip, fast reads (7) | one CI run + one live gate + restore drill | one | 6–8 |
| **C: Native ready** | pinned native evidence and kickers (5); catalog, one command path, frontend from the catalog (8–10); owner workflows (7b) | one CI run + a native league walked end to end | one | 12–15 |
| **D: Others install** | Windows/macOS builds and an install path (7c); UX walk and fixes; `twr` CLI (11) | packaged builds launch on 3 OSes + a UX walk | one | 4–6 |
| **E: MFL commands** | solo-owner pools and bids (12) | one CI run + a live walk | one | 3–5 |

**Total:** about 26–35 days, with 4 clicks in all (was 17 gated phases).
