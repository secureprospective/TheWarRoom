# Versioning & Releases — Planning Scaffold

**Status:** PLANNING (framing, not decided). Seeded 2026-07-19 at M2-slice-1 session close.
**Owner call:** Christopher drives the decisions; this doc frames the picture so the
next session starts warm, not cold. Nothing below is locked — the "recommended" notes
are starting positions to react to, per the [[feedback_expert_panel_decision_gate]] and
Christopher's framing-pace preference.

---

## Why now

M-series UI is landing (M1 Rankings, M2 Power Rankings, M4 Transactions all merged).
The app is becoming a thing a person USES, not just a thing that builds. The delivery
lifecycle is **Phase 1 personal tool → Phase 2 league-wide alpha → Phase 3 public/
self-hosted beta**. Phase 2 is the first time the binary leaves this machine — which
is exactly when versioning, reproducible builds, changelogs, and a schema-migration
story stop being optional. **This phase sets those conventions before the first
external hand-off, not after.**

## Current state (the clean slate tomorrow inherits)

- **No version anywhere:** `wails.json` has no `info.productVersion`; no `VERSION`
  file; no git tags; no `-ldflags -X` build-time version injection.
- **Release process today:** branch → quality gates → live Beelink gate → squash-merge
  to main. No tag, no artifact, no release notes step.
- **Rich changelog SOURCE already exists:** `docs/build-handoffs/Build_Tracker.md`
  (per-row completion state + squash SHAs) and the `handoffs/NN-*.md` series. A release
  changelog can be DISTILLED from these rather than hand-kept in parallel.
- **In-app config versioning already exists and is SEPARATE:** `rulebook.ActiveVersion`
  (league-config versions in SQLite). Do not conflate app-release versioning with
  league-config versioning — they are different axes.

---

## The decisions to make (framed as open questions + options)

### D-V1. Versioning scheme — **durable, convention-setting → expert-panel candidate**
- **Options:** (a) SemVer `MAJOR.MINOR.PATCH`, `0.x` through Phase 1–2, `1.0.0` at
  public beta; (b) CalVer `YYYY.MM.PATCH`; (c) phase-anchored (`0.<phase>.<module>`).
- **Recommended starting position:** SemVer, currently `0.y.z`. Bump MINOR per shipped
  module/slice, PATCH for fixes, reserve `1.0.0` for the Phase-3 public beta. It maps
  cleanly onto the Build_Tracker cadence and is what any future contributor expects.
- **Open:** does a "module complete" (e.g. M2 slice-1) = a MINOR bump, or do we batch
  modules into a release? (Cadence question, D-V4.)

### D-V2. Where the version lives (single source of truth)
- **Options:** `wails.json info.productVersion` (Wails stamps the binary/installer),
  a root `VERSION` file, a git tag, or a Go `-ldflags -X main.version=` injection.
- **Tension:** the binary should REPORT its version (UI "About" + logs), and the build
  should stamp it reproducibly. A git tag alone doesn't reach runtime.
- **Recommended:** git tag is the source of truth; the build injects it via
  `-ldflags -X` into a `main.version` var AND syncs `wails.json` productVersion; the UI
  shows it. One tag → one stamped binary. Decide the exact plumbing tomorrow.

### D-V3. Build & distribution (Wails desktop specifics)
- `wails build -tags webkit2_41` produces the binary; **which target OSes for Phase 2
  league members?** (Christopher builds on Linux; league members' platforms UNKNOWN →
  needs a real answer before Phase 2.) Cross-compile matrix + code-signing per-OS.
- **Artifact hosting:** GitHub Releases (private repo — access for league members?), or
  a download surface on the existing Cloudflare/secureprospective infra.
- **Auto-update:** Wails has no built-in updater. Decide: manual re-download for Phase
  2, or invest in an update mechanism before Phase 3. (Probably defer to Phase 3.)

### D-V4. Release cadence & gating
- **What is a releasable state?** The existing gates (make lint 0 / `go test -race` /
  tsc+vite / live Beelink functional gate) are the floor. Does a release require a
  live gate every time, or only a tag off an already-gated main?
- **Cadence:** per-module, per-Build_Tracker-tier, or time-boxed?
- **Recommended:** tag releases off main after a module's live gate passes (we already
  gate every merge); no separate release gate needed — the merge gate IS the release
  gate at this stage.

### D-V5. Changelog / release notes
- **Recommended:** generate from Build_Tracker completion rows + handoff titles +
  squash-commit subjects since the last tag. A `CHANGELOG.md` distilled at tag time,
  not hand-maintained in parallel. Keep-a-Changelog format is a safe convention.

### D-V6. **Schema / data migration across releases — the real risk, flag HARD**
- The app has a SQLite schema. Once a version ships to a league member, the NEXT
  version that changes the schema must MIGRATE their existing DB, not assume a fresh
  one. There is already a DB-corruption/init lesson on record
  ([[lesson_thewarroom_db_corruption_probe]]) — released versions raise the stakes.
- **Open questions:** is there a schema-version marker in the DB today? A migration
  runner on startup? An idempotent/forward-only migration convention? This likely
  deserves its own design sub-session BEFORE the first schema-changing release ships
  externally. **Do not let a released binary's first schema change be unplanned.**

### D-V7. Distribution ↔ phase mapping
- Phase 1 (now): local `wails build`, tags for personal history — low ceremony.
- Phase 2 (league alpha): signed cross-platform binaries + a controlled download +
  the migration story from D-V6 — this is where the ceremony has to exist.
- Phase 3 (public/self-hosted beta): auto-update + broader signing + support surface.

---

## Recommended first-session sequence (a starting plan to react to, tomorrow)

1. **Frame + lock D-V1 (scheme) and D-V2 (source of truth)** first — everything else
   hangs off them. These two are the convention-setters; consider the expert-panel gate
   for D-V1.
2. **Wire the minimum:** git tag → `-ldflags -X main.version` → UI "About"/version
   surface → tag the current main as the first version (e.g. `v0.4.0` — pick the number
   once D-V1 lands). Prove the binary reports its own version on the Beelink.
3. **Stand up the changelog distillation (D-V5)** from Build_Tracker + tags.
4. **Open the D-V6 schema-migration design as its own thread** — scope it, don't rush a
   half-answer; it's the one that bites after a release, not before.
5. Defer D-V3 cross-compile/signing + D-V4 cadence + auto-update to when Phase 2 dating
   is real; capture the questions, don't answer them prematurely.

## Cross-project reuse note

Christopher runs several shipping web properties (SecureProspective, TFM, CCwork) with
their own deploy stories, and a Coding-Standards fleet repo. A versioning/release
convention settled cleanly HERE (SemVer + tag-injected version + distilled changelog +
migration discipline) is a candidate to promote into
`christopher-coding-standards` as a reusable release-overlay later — flag it, don't
build it now.
