# Store Hardening Plan — 2026-10

**Status:** revised 2026-10-19 with Christopher's answers; waiting for his go. Nothing built yet.
**Goal:** make the SQLite store layer correct first, efficient second and robust third, for the
MFL harness side and the native league side.
**Picture:** `http://192.168.1.105:8765/store.html` (today → target, while the CT105 page server runs).
**Evidence:** Claude's map of all four databases and Sol 6.1's adversarial review
(`~/fleet/runs/sol61-warroom-store-review/stdout.log` on the Beelink). Every finding below was
checked in the code or reproduced in memory before it was listed.

## Findings (verified)

| # | Finding | Where | Side | Priority |
|---|---|---|---|---|
| F1 | The mirror publishes the new rosters before the disk write; after a failed write, a retry is skipped because the hash already moved | `state/mirror.go:111` | MFL | correctness |
| F2 | Refresh saves new rules before rosters are fetched; a failed fetch leaves new rules over old rosters | `refresh_app.go:94–105` | MFL | correctness |
| F3 | Foreign keys are never switched on; one broken link already exists in dev data (`model_runs` → missing `param_sets`) | `db/pools.go` | both | correctness |
| F4 | `INSERT OR REPLACE` deletes ledger rows without firing the no-delete trigger (reproduced); no code does this today | `history/events.go` | native | correctness |
| F5 | The ledger's JSON check accepts `{}` (reproduced); other JSON columns have no check | `history/events.go:28` | both | correctness |
| F6 | An approved stat line is silently ignored when no held versions exist | `commissioner/stat_ledger.go:94` | native | correctness |
| F7 | A rostered player with no PAID salary cell counts $0 cap and the total is still used | `state/state.go:213` | MFL | correctness |
| F8 | Money parsing can overflow; percent settings accept NaN | `domain/money.go:55`, `rulebook/helpers.go:80` | both | correctness |
| F9 | Native scores depend on which archived copy of the stats file was read; losing or changing copies can change past results | `history/playerstats.go`, `fantasyscore/source` | native | correctness |
| F10 | Only `whatif.db` has migrations; `history.db` (holding the native ledger) and `thewarroom.db` have no schema version or downgrade guard. Even `whatif.db` runs DDL before its downgrade check | `state/migrations.go`, `state/schema.go` | both | robustness |
| F11 | No routine backup or integrity check of any file; the irreplaceable ledger (1.3 MB) sits in a 375–490 MB file | `app.go:289` | native | robustness |
| F12 | Every native read and command re-parses the whole ledger from revision 0; one event is 772 KB | `history/native_assets.go:11` | native | efficiency |
| F13 | Scoring a starter re-reads and re-parses every held copy of the season stats CSV (2026 has 24 fetches, 9 distinct, growing) | `history/playerstats.go:31` | native | efficiency |
| F14 | Archive lookup is a substring scan over URLs | `history/archive.go:111` | both | efficiency |
| F15 | Every what-if preview writes a full league copy to `whatif.db` and never cleans it up | `whatif_app.go:226`, `trade_projection.go:54` | MFL | efficiency |

**Keep:** SQLite + WAL + one writer per file · ledger revisions and retry keys · corrections that
never edit history · integer cents · the gzipped content-addressed archive · what-if
publish-after-success · full frozen seeds · the modernc driver.

## Christopher's answers (2026-10-19)

1. **Real people want to run native leagues.** That's why this week is a cleanup before any new
   direction. Native correctness is now real, not future-proofing.
2. **This runs alongside the ring work,** as small branches merged between ring phases. Only one
   push or verify runs at a time on the Beelink.
3. **A client app plus a server install is the likely native future; don't dig into it yet.** So:
   no `native.db` file split and no off-machine backup design. Keep every fix storage-agnostic, and
   keep the native ledger behind its store interface so a later move to a server stays clean.

## Phases, in build order

Each phase is its own `session/store-*` branch. Sol builds at medium; Claude reviews every line,
plants one mutant per report, runs `make verify` and the live gate on Claude-OS. Christopher
clicks before any merge. Squash-merge only.

### Phase 0 — Impact map (read-only) · ½ day
Run first; it may change the order and ETAs below.
1. Callers: every IPC method, background worker and test that reads or writes each table touched.
2. Real data, from backup-API copies only (Beelink live, Claude-OS live and dev): FK violations,
   invalid JSON, cap-cell drift, what-if generation count, ledger sizes.
3. Upgrade path: new build on old files, old build on new files.
4. Screens: readings whose shape changes (for example, "cap total invalid").
5. Tests and fixtures that assume current behaviour.
6. Wider project: ring roadmap, `SYSTEM_MAP.md`, the endpoint registry, `cmd/fit`.
**Output:** one page per phase (what it touches, what could break, how each risk is tested),
plus the corrected order and ETA.

### Phase 1 — Faster test suite · 1 day
Comes early because every later phase pays the ~45-minute verify on each push.
- Today: 1,993 Go tests, about 74k lines. The root package has 28k lines and takes **40 minutes**
  under `-race`, about 80% of the wait. Next slowest: `store/history` (6 min) and `labcompile`
  (2 min).
- Profile the root package (`go test -json` timings) and find the slowest tests.
- Speed them up without losing coverage: shared fixtures, no repeated full app/database builds,
  slow live-style checks kept out of the race run.
- Then measure coverage overlap. A test is removed only with evidence it covers nothing unique
  ("deleting tests is not improvement").
**Gate:** same coverage, push verify about 10–15 minutes.

### Phase 2 — Migration standard (all four files) · 1 day
Comes before the fixes, because the JSON and foreign-key fixes change schemas.
- One migration runner per file using `user_version`: refuse newer → back up (`VACUUM INTO`) →
  migrate, with each step and its marker in one transaction → integrity + FK check.
- Fold in `whatif.db`'s existing migrations; check versions before any DDL (F10).
- Per-connection settings asserted on open: `foreign_keys=ON`, `recursive_triggers=ON`,
  `synchronous=FULL` (F3, F4).
**Gate:** an old database opens and upgrades; a database from a newer build is refused untouched.

### Phase 3 — Stop wrong state, both sides · 1–1.5 days
- Mirror: persist, then publish (F1). Refresh: stage rules and rosters, publish together (F2).
- NULL-safe JSON checks on the ledger and other JSON columns, as migrations (F5).
- An approved stat line is never silently ignored (F6).
- $0-cap drift marks the total invalid and blocks dependent what-ifs (F7).
- Checked money arithmetic; reject NaN or out-of-range percents (F8).
- Repair the broken `model_runs` link from provenance; never invent a parent (F3).
**Gate:** each fix has a test that fails without it; an injected mirror-write failure leaves memory,
disk and revision unchanged.

### Phase 4 — Pinned native evidence · 1.5–2 days
Moved up because real native leagues are likely. This is logic, not storage, so it carries over
to a future server.
- Record the exact stat line, identity and source digest each native result used (F9), and make
  the initial baseline durable.
- Native results rebuild from the ledger plus pinned evidence, with no network and no dependence on
  which archive copies remain.
**Gate:** delete the archived stat copies in a test database; past native results don't change.

### Phase 5 — Local backups and checks · ½ day
- Backup-API copies daily and before every upgrade, with a manifest (schema versions, ledger
  revision).
- Weekly `integrity_check` + `foreign_key_check`, shown in Data Sources.
**Gate:** a restore on a clean profile reproduces the league and native results.

### Phase 6 — Fast reads · 1.5 days
- Archive: exact-URL index, metadata first, bodies by hash (F14).
- Parse each stats file once, cached by body hash and parser version (F13).
- Previews in memory; prune unpublished what-if copies (F15).
- Benchmark `observations` indexes and `WITHOUT ROWID` before changing them.
**Gate:** a full native week scores within a stated budget, with results unchanged.

## Deferred to the client/server decision
- A separate native ledger store (`native.db` or a server database).
- A heavy projection cache for native reads (F12); the full-replay cost is acceptable at today's
  league sizes.
- Off-machine backups.
- The expert panel, which reviews the client/server storage design when that door opens.

## ETA
About **7–8 working days** in total:

| Phase | Days |
|---|---|
| 0 Impact map | 0.5 |
| 1 Faster tests | 1 |
| 2 Migration standard | 1 |
| 3 Stop wrong state | 1–1.5 |
| 4 Pinned evidence | 1.5–2 |
| 5 Local backups | 0.5 |
| 6 Fast reads | 1.5 |

- **Calendar time is longer,** because the work interleaves with the ring work.
- **Paced by:** the push verify (shorter after Phase 1), and Christopher's click-throughs.
- **Earliest protected point:** after Phase 5.
