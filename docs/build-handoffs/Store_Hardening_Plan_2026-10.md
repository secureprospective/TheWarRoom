# Store Hardening Plan — 2026-10

**Status:** proposed, waiting for Christopher's go (2026-10-19). Nothing built yet.
**Goal:** make the SQLite store layer correct first, robust second and efficient third, for the
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
6. Wider project: ring roadmap, `SYSTEM_MAP.md`, the endpoint registry, `cmd/fit`, CT105 backups.
**Output:** one page per phase (what it touches, what could break, how each risk is tested),
plus the corrected order and ETA.

### Phase 1 — Migration standard (all four files) · 1 day
Comes before the fixes, because the JSON and foreign-key fixes change schemas.
- One migration runner per file using `user_version`: refuse newer → back up (`VACUUM INTO`) →
  migrate, with each step and its marker in one transaction → integrity + FK check.
- Fold in `whatif.db`'s existing migrations; check versions before any DDL (F10).
- Per-connection settings asserted on open: `foreign_keys=ON`, `recursive_triggers=ON`,
  `synchronous=FULL` (F3, F4).
**Gate:** an old database opens and upgrades; a database from a newer build is refused untouched.

### Phase 2 — Stop wrong state · 1–1.5 days
- Mirror: persist, then publish (F1). Refresh: stage rules and rosters, publish together (F2).
- NULL-safe JSON checks on the ledger and other JSON columns, as migrations (F5).
- An approved stat line is never silently ignored (F6).
- $0-cap drift marks the total invalid and blocks dependent what-ifs (F7).
- Checked money arithmetic; reject NaN or out-of-range percents (F8).
- Repair the broken `model_runs` link from provenance; never invent a parent (F3).
**Gate:** each fix has a test that fails without it; an injected mirror-write failure leaves memory,
disk and revision unchanged.

### Phase 3 — Backups and checks · 1 day
- Daily and pre-upgrade backups via the SQLite backup API, with a manifest (schema versions, ledger
  revision); off-machine copy to CT105 by deliberate sync, never rsync.
- Weekly `integrity_check` + `foreign_key_check`, shown in Data Sources.
- Restore drill on a clean profile.
**Gate:** restore on a clean profile reproduces the league and native results.

### Phase 4 — `native.db` and pinned evidence · 3–4 days · expert panel first
Hard to undo, so the panel reviews the design before any code (½ day).
- Record the exact stat line, identity and source digest each native result used (F9).
- Move `league_event`, the authority policy, seed documents (stored once, by hash), pinned
  evidence and projections into `native.db`.
- Copy, verify (counts, revisions, IDs, payload hashes, replay results), and keep the old copy
  until everything matches. No cross-file move-and-delete.
**Why now:** only test leagues exist today, so the move is cheap. After a real native season it
needs a careful data migration.
**Gate:** native results rebuild offline from `native.db` alone.

### Phase 5 — Fast reads · 2 days
- Archive: exact-URL index, metadata first, bodies by hash (F14).
- Parse each stats file once, cached by body hash and parser version (F13).
- Native projection cache keyed by revision; full replay stays as the check (F12). Reversals can
  reach back before a snapshot, so the cache rebuilds from a safe point, never blindly patches.
- Previews in memory; prune unpublished what-if copies (F15).
- Benchmark `observations` indexes and `WITHOUT ROWID` before changing them.
**Gate:** a full native week scores within a stated budget; projection and replay hashes agree,
including through reversal-of-reversal, rollover and draft restore.

## ETA
About **8–11 working days** in total:

| Phase | Days |
|---|---|
| 0 | 0.5 |
| 1 | 1 |
| 2 | 1–1.5 |
| 3 | 1 |
| 4 | 3.5–4.5 |
| 5 | 2 |

- **Earliest protected point:** after Phase 3, about 4 days in.
- **Paced by:** the ~45-minute verify on every push, and Christopher's click-throughs.

## Open questions for Christopher
1. When does a real native league start? That sets how urgent Phase 4 is.
2. Where does this fit against the ring roadmap: before ring 5, or alongside it?
3. Off-machine backups: CT105 only, or somewhere else too?
