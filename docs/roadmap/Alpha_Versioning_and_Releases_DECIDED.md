# Alpha Versioning & Releases — DECIDED PLAN

**Status:** DECIDED 2026-07-21. Supersedes the open questions in
`Versioning_and_Releases_Planning.md` (that doc stays as the framing scaffold + the
D-V1…D-V7 question set; this doc carries the answers).
**Decided by:** Christopher, after a three-model expert panel (GLM 5.2, DeepSeek, Gemini)
triaged against source by Claude head-brain.
**Pillars:** Business, Technical.

---

## 0. THE FINDING THAT RESHAPED THIS PHASE

**The league was polled 2026-07-21. Nobody else wants to run the binary.** ("No one wants
to go through the trouble of updating and bug reporting.") **Alpha's audience is one
person: Christopher, on Linux.**

Every heavyweight item in the original scope — code signing, cross-platform builds,
artifact hosting, recovery UI, first-run polish, diagnostics export, the WebKitGTK distro
matrix, changelog ceremony — was justified by an external audience that does not exist.
**They are all deferred to Beta.**

**Consequence — the Alpha gate is UNBLOCKED.** `CLAUDE.md` and handoff `40` both stated
the UI build track (B-2…B-5 → ALPHA) *hard-depends* on this phase completing first. That
dependency was justified by "stamped builds + the D-V6 migration story before any binary
leaves the machine." **No binary is leaving the machine.** The dependency is downgraded
from a hard gate to a light one (see §3 sequencing).

**The one real risk that survives has nothing to do with distribution:** the league ledger
is append-only, carries years of dynasty history, and is the only artifact in this project
that cannot be rebuilt from source. There are at least two live DBs (CT105 + the Beelink
gate clone). Migration safety is not about league members. It is about not destroying your
own ledger during a schema change.

---

## 1. DECISIONS LOCKED

| ID | Decision | Status |
|---|---|---|
| D-V1 | **SemVer `0.y.z`**, `1.0.0` reserved for public beta. Schema version is a **separate axis** — never encode data compatibility in the SemVer string. | LOCKED |
| D-V2 | **Git tag is the source of truth** → `git describe --tags --always --dirty` → `-ldflags -X` into `main.version` + short SHA + build date → **Wails binding** → UI. Source default `"dev"`, never `""`. No `version.json`, no duplicated TS constant. | LOCKED |
| D-V3 | **Linux only.** Windows/macOS/signing/hosting → Beta docket. | LOCKED |
| D-V4 | **No separate release gate.** The existing merge gate (lint 0 / `go test -race` / tsc+vite / live Beelink gate) IS the release gate. Tag off already-gated main. | LOCKED |
| D-V5 | **No CHANGELOG.md for Alpha.** `Build_Tracker.md` already IS the changelog and has a single reader. Revisit at Beta. | LOCKED |
| D-V6 | **Per-`(owner, version)` `schema_migrations` table.** No central coordinator. No `PRAGMA user_version`. Forward-only. `VACUUM INTO` pre-migration backup. | LOCKED |
| D-V7 | Phase mapping unchanged; Alpha ceremony collapses to §2 below. | LOCKED |

### D-V6 rationale (the contested one — decided on VERIFIED facts, not opinion)

The panel split. **GLM chose per-owner; Gemini and DeepSeek chose a single global
coordinator.** Source triage settled it:

- **The three schema owners are cleanly disjoint.** `state` touches only rosters /
  contracts / contract_years / ledgers / transaction_counts / season_phases /
  cap_relief / player_status; `rulebook` only `rulebook_*`; `params` only `param_*`.
  **Zero cross-package SQL. Zero cross-package foreign keys** (the only FK is
  `rulebook → rulebook_versions`, within one owner).
- **Only `state` has migrations at all.** `rulebook` and `params` have none.
- Therefore DeepSeek's "untested version combinations" hazard does not apply — disjoint
  owners cannot interact, so `state=v5, rulebook=v3` has no failure mode.
- Gemini's premise ("non-deterministic startup order") is **false**: `app.go
  initStoreFloor` is a single fixed explicit sequence.

**GLM's premise was the only one that survived contact with the code.** Adopt option (ii).

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    owner       TEXT    NOT NULL,
    version     INTEGER NOT NULL,
    applied_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    method      TEXT    NOT NULL DEFAULT 'migrated',  -- 'migrated' | 'reconciled'
    PRIMARY KEY (owner, version)
);
```

Each package's `Initialize()` creates the table (idempotent), reads its own rows, runs its
own pending migrations. Adding a fourth store package costs one new owner string and zero
coupling.

---

## 2. THE WORK, IN TIERS

### TIER 1 — Version stamping (~half a session). DO THIS FIRST.

- Build script computes `git describe --tags --always --dirty`.
- `-ldflags -X main.version=… -X main.commit=… -X main.buildDate=…`.
- Source defaults `"dev"` / `""` / `""` — a dev build must be visibly distinct.
- `internal/build` (or `main`) var → Wails-bound `AppInfo()` struct → Zustand → UI surface.
- Sync `wails.json productVersion` from the same `git describe` output (packaging metadata
  only; inert on Linux, matters for Windows/macOS later).
- `make release` target refuses to tag from a dirty tree.
- **Tag current main `v0.5.0`.** Do NOT inherit the scaffold's `v0.4.0` placeholder — the
  scouting engine, FILM, and B-1 shell all landed after it was written.

**Why first:** B-2 is restyle-heavy with many Beelink live-gate round trips, and "which
build is actually on that machine" is currently unanswerable. Immediate payoff.

**Verify:** `wails build` on the Beelink; the About surface reports the tag + SHA; a dirty
build visibly reports `-dirty`.

### TIER 2 — Ledger safety. BEFORE the next schema-touching work, not before B-2.

B-2 is a **frontend restyle and does not touch the schema**, so this does not gate it. Do
it before the calendar backend, M4 read-model store, or M2 slice-2 columns.

- `schema_migrations` table per §1.
- **One-time reconciliation** for pre-marker DBs (there are ≥2 in the wild, both
  Christopher's): if the table is absent and tables exist, run **precise**
  `is_already_applied` predicates per migration — a **data** check, not just column
  presence — stamp rows with `method='reconciled'`, never re-run a completed migration.
  Reconciliation is the ONE place expensive data checks are acceptable; after it, the
  marker is the check.
- **`VACUUM INTO` pre-migration backup**: `PRAGMA wal_checkpoint(TRUNCATE)` then
  `VACUUM INTO 'thewarroom.db.premigration-<ts>'`. Keep last 3, prune older. Skip entirely
  when no migrations are pending. **Never an OS file copy** — WAL state spans
  `.db` + `.db-wal` + `.db-shm`.
- **Downgrade check (~5 lines/owner):** if any owner's max DB version exceeds the binary's
  max known version, **refuse to open** with a plain-language message. Cheap, and the two
  machines will drift.
- **Forward-only.** No down-migrations. The backup IS the rollback.
- **Migration regression test:** build a DB at the pre-marker state, insert data, run
  `Initialize()`, assert `schema_migrations` populated, cents backfilled, legacy columns
  dropped, data intact.

### TIER 3 — Cheap self-protection. Whenever convenient.

- **Single-instance lock** (`flock` on a lockfile / PID file) — **verified absent today.**
- **Disk logging** to `os.UserConfigDir()/TheWarRoom/logs/` with rotation — **verified
  absent today**; only `log.Printf` to stderr, which goes nowhere from a `.desktop` launcher.
- **`dev`-build guard:** a binary reporting `dev` refuses to open the real DB (or uses a
  `-dev` copy). Prevents nuking the live ledger during `wails dev`.
- Rename `rulebook.ActiveVersion` in all user-facing text to **"Rulebook Version"** —
  never bare "version". Prevents ambiguous reports.

---

## 3. SEQUENCING

```
Tier 1 (½ session)  →  B-2 module migration  →  Tier 2 (before next schema change)
                                              →  Tier 3 (opportunistic)
```

Tier 1 first purely because it pays for itself inside B-2's gate loop. **Do not let the
phase name give this gravity** — it was scoped when Alpha meant "ships to ten people."

---

## 4. VERIFIED FACTS (established 2026-07-21 — do not re-derive)

These were established empirically this session and are permanent regardless of audience:

1. **`migrateMoneyCents` is ALREADY ATOMIC.** `ALTER TABLE ADD COLUMN` + backfill + verify
   run in ONE explicit transaction with `defer tx.Rollback()` and a loud-failing verify
   (`internal/store/state/schema.go`, labeled "ATOMIC (z.ai Ship-4 M1)"). A prior GLM
   review already fixed this. **There is no live half-migrated bug.** The code comment
   documents the hazard AND its closure — reading only the hazard clause is what misled
   the 2026-07-21 panel brief.
2. **`dropLegacyMoneyColumns` is already idempotent** — each column dropped only if present.
3. **DDL IS TRANSACTIONAL on `modernc.org/sqlite`** — probed directly: `ALTER TABLE` inside
   a transaction, rolled back, column gone. DeepSeek's "DDL forces an implicit commit"
   claim is **FALSE** (that's MySQL/Oracle semantics). Migrations can be fully atomic.
4. **`VACUUM INTO` works on `modernc.org/sqlite`**, and `PRAGMA user_version` survives it.
   **This closes the build-time check docketed in `Vision_2026.md` D2** — the fork
   primitive behind the shadow ledger / trade analyzer / January planner is confirmed viable.
5. **The three schema owners are disjoint** (see §1 rationale).
6. **No single-instance lock and no disk logging exist** (greps returned nothing).
7. **Clean slate confirmed:** no git tags, no `wails.json productVersion`, no `VERSION`
   file, no `-ldflags` injection.

---

## 5. WHY WAILS — the answer, recovered from source (asked 2026-07-21)

`docs/ui/UI_Direction_Document.md`, locked technology table:

> **Desktop framework | Wails | v2 (stable GA) | v3 is alpha as of June 2026. v2 is
> production-locked, fully documented, zero breaking risk.**

> **Backend language | Go | Scoring engine implemented natively in Go. No Python sidecar.
> No multi-runtime complexity.**

**The Go engine picks the framework.** Tauri is Rust, Electron is Node — both would demote
the engine to a sidecar process. Wails compiles Go + frontend into one binary with
in-process bindings. `Salary_Ledger_Design.md` actively depends on this:
*"Wails is in-process so per-tick preview is a local call — latency is irrelevant."*

**v2 was chosen BECAUSE v3 was alpha — the same reason not to migrate to v3 today.**
DeepSeek's "budget for a v2→v3 migration" re-raised a tradeoff already evaluated and
deliberately taken. v3 is already docketed: `UI_Direction_Document.md` line 768 lists
multi-window/multi-monitor as Phase 3, *"requires Wails v3 when it reaches stable GA."*

**Decision: do not migrate. Revisit only if v3 reaches GA or WebKitGTK actually bites.**
Fetched release data (2026-07-21) showed v3 still in extended alpha (`alpha2.117`) and
v2 as stable — but the timestamps returned were stale (2024), so treat specifics as soft.

**If a move is ever forced, the ranked alternatives:**
1. **Wails v3** once GA — same model, engine + React survive, cheapest.
2. **Go serves React over localhost + system browser** — kills the WebKitGTK dependency,
   identical rendering everywhere, Windows/macOS free, **and it is already client-server so
   going hosted later becomes a deployment change, not an architecture change.** Loses
   native window chrome / file dialogs / tray.
3. **Tauri + Go sidecar** — adds a language and process lifecycle.
4. **Electron** — consistent Chromium, 150MB+ binaries, engine still a sidecar.

---

## 6. BETA DOCKET (deferred, with reasoning intact)

- **Distribution: apt repo + `.deb` with declared `Depends: libwebkit2gtk-4.1-0`.** The
  correct Linux answer — solves the WebKit dependency declaratively AND eliminates the
  auto-update problem (apt IS the updater). Rejected for Alpha only because Christopher
  builds from source on both machines; an apt repo would replace `git pull && make build`
  with GPG-signed repo hosting for one user. **NOTE: apt manages binaries and does nothing
  for the schema. Never run migrations from a `postinst` script** — wrong user, wrong time.
- **Phase-2 exposure — RECOVERED FROM `Backend_Architecture.md`:** the original documented
  plan for letting league members in was **Cloudflare Tunnel (`cloudflared`) + Cloudflare
  Zero Trust (free tier)** — run the desktop app locally, expose it through the tunnel.
  **No rewrite, no hosting migration, no per-platform binaries**, and Christopher already
  runs Cloudflare infra for three properties. **This is cheaper than every option the panel
  or Claude proposed** and should be the starting position whenever Phase 2 arrives.
- **PANEL THIS WHEN PHASE 2 IS REAL:** exposure path — Cloudflare Tunnel vs hosted web app
  vs local-server+browser. Durable, contested, never reviewed.
- Deferred user-facing work: code signing, cross-platform builds, recovery UI, first-run
  experience, diagnostics export/support bundle, WebKitGTK distro matrix (Ubuntu 20.04
  ships only 4.0 — a `webkit2_41` binary will not start there), `CHANGELOG.md`.
- **Two-product fork** (`Vision_2026.md`): League Instance vs Owner's Cockpit. The Cockpit
  is inherently local-first (one owner, own data, own machine) — desktop + local DB is the
  CORRECT architecture for it, not a compromise. D2's fork primitive assumes it.

---

## 7. PANEL PROVENANCE + A CORRECTION TO CARRY

Three models answered a self-contained brief (archived; prompt was in `/root/paste.md`).
**The brief contained a Claude error:** it quoted the half-migrated hazard comment while
omitting "The transaction closes that window," presenting a CLOSED hazard as live. All
three built Q3f answers on that false premise.

**Quality signal worth remembering:** GLM explicitly hedged — *"Medium confidence on the
specific fix, because I haven't seen the actual migration code. I'm inferring the
non-atomicity from the brief's description"* — and correctly identified the brief as the
weak link. DeepSeek asserted the same wrong conclusion with high confidence. Gemini
asserted a false premise (non-deterministic startup) it could not have checked.

**Two of three models reached the same architectural recommendation via premises that were
both false when checked against source.** Convergence is not corroboration. Triage against
source, always — this is the case study.

Panel answers are **leads, not findings**, and their user-facing half is now scoped to a
Beta that may look nothing like what they assumed. Archive, do not treat as decided.
