# Handoff 50 — Alpha Versioning phase COMPLETE + recovered pending harness cases 3G/3H

**From:** the 2026-07-24 session (Alpha Versioning Tiers 2 + 3).
**For:** the next build session.
**Type:** STATE MARKER + carried-work recovery. No task is in-flight; pick from NEXT below.

## WHAT JUST SHIPPED — the 3-tier Versioning & Releases phase is DONE
All three tiers merged to main, each live-gated on the Beelink + GLM-reviewed:

- **T1** (last session) — build stamping, tagged `v0.5.0`.
- **T2** `facb73c` — `internal/store/state/migrations.go`: forward-only registry over a
  per-`(owner,version)` `schema_migrations` table (D-V6); the two money migrations wrapped
  unchanged; pre-marker DBs reconciled from DATA predicates; `VACUUM INTO` backup before any
  real work (keep 3, never OS copy); downgrade guard via `maxKnownStateVersion()` (derived from
  the registry). Live gate: real 831-contract DB → both `reconciled`, 0 backups, data intact.
- **T3** `0b971ca` — `instance_lock.go` (flock single-instance), `logging.go` (stderr+file,
  keep 10), dev-build guard (`version=="dev"` → separate `thewarroom-dev.db`), frontend rename
  `config v{n}`→`rulebook v{n}`. Live gate: real ledger opened, 2nd instance LOCKED OUT, log
  written, no dev DB.

⚠️ **Behavior change:** `wails dev` now uses `thewarroom-dev.db` and seeds fresh — a dev
session will NOT show the real roster. Intended protection.

Plan doc (all three tiers): `docs/roadmap/Alpha_Versioning_and_Releases_DECIDED.md`.
**Nothing downstream is gated on versioning any more.**

## ⚠ RECOVERED CARRIED WORK — harness cases 3G + 3H (never wired)
Flagged by Christopher 2026-07-12, carried silently through the entire B5b rubric arc, and
almost forgotten (he remembered them as "3h and 3p" — there is no 3P; the harness runs
3A–3M). **Verified still `gatedPending`/`subSignalPending` in `internal/harness/cases.go:68–71`
as of this session.** Spec: `Testing_App_Specification.md` Tests 3G (line 420) + 3H (line 444).

### 3G — DT dynamic PFF alpha (SL-021): α=0.50 Y1 → 0.10 Y2+, DE control fixed 0.15
- The introspection hook **exists and is correct**: `DT.PFFAlpha(nflYear)` → 0.50 (year ≤1),
  0.10 (year 2+), `internal/engine/l4/defense/dt.go:151`.
- **The blend it feeds is DORMANT** — DT film is Data-Parity neutral (IDP film weights UNSET,
  Build_Tracker B5b-DT), so the full spec (live EMA → 0.75 vs 0.63) cannot run yet.
- **Partially wireable now** (assert the α schedule via the hook + a DE fixed-0.15 control).
  **Full 3G belongs with the pending IDP FILM calibration pass** (the IDP arm of Thread C —
  only offense/K film shipped so far; see the FILM planning doc / handoff 48).

### 3H — Confidence floor: an all-Unknown component → effective 1.000
- Old stated blocker: "component-confidence inputs are not on `Layer4Input`." Confidence is
  deliberately engine-INTERNAL (`internal/engine/types.go:16`).
- **But the end-state already exists via Data-Parity**: `HasRAS=false → RASEffective=1.000`,
  both-film-absent → 1.000 (proven in the K rubric), etc.
- **Likely wireable NOW** as a property assertion over the existing `Has*=false` paths: build
  an all-absent DT (or any) spec and assert `Layer4Output.Combined == 1.000` exactly. **The one
  thing to verify first:** that an all-absent breakout composite (neutral 0.50s through the
  breakout S-curve) lands on exactly 1.000 — S-curve centering is the risk. If it does, 3H is a
  small, self-contained wire-up with no engine change.

**Recommendation:** wire **3H now** (small, if the floor check holds); fold **3G** into the IDP
film calibration where its blend goes live. Do NOT ship blind film weights for 3G.

## NEXT (nothing gated on versioning) — pick one
- **B-2 module migration** — UI build track (restyle M1/M2/Transaction/Trade/League into the
  Session-B component language). See `[[project_thewarroom_ui_roadmap]]`.
- **Harness 3H** (small, above) and/or **3G** (with IDP film).
- **M2/M4 refactors** — handoffs 43 (M2 service extraction) / 44 (M4 read-model store).
- **League-calendar backend** — branch `session/league-calendar` WIP.
- **M2 slice-2** — weeklyResults columns.
- **FILM Thread C (IDP arm)** — OffenseFilm/K shipped; IDP film (Coverage/IDPFilm) + 3G still
  pending, decision-gated, expert-panel weight gate. Handoff 48.

## OPEN / CARRIED (unchanged)
- OQ-013 (created→official id ramp), OQ-014 (Money type), OQ-016 (RookieConsensus CSV loader),
  AgeTrajectory field vestigial, K2 NFLProduction seat reserved neutral in both `applyScouting`
  branches.
- ⚠️ ROTATE the free CFBD key + the z.ai `GLM_API_KEY` at beta.
- **Panel at Phase 2:** exposure path (Cloudflare Tunnel + Zero Trust vs hosted vs local-server).
