# Handoff 40 — Versioning & Releases (next phase)

**From:** M2 Power Rankings slice-1 session; **state-refreshed 2026-07-20 at the UI wireframe
track A–E close.**
**For:** THE session Christopher runs right after the wireframe-track clear — opening the
versioning/releases phase.
**Type:** PLANNING-FIRST (this is a phase shift, not a Build_Tracker code row). **ENTER PLAN
MODE** (plan mode overrides act-fast — [[feedback_plan_mode_override]]); frame and decide the
conventions before writing any release plumbing.

## ⚠ STATE REFRESH — current main is `d137a68` (2026-07-20)
- Since this handoff was first written (`686a574`), the entire **UI wireframe track A–E is
  COMPLETE + merged** (`e82f01a` + CLAUDE build-state `d137a68`). The design language grid→mobile
  is now locked as a spec — which SHARPENS the "why now": the app has both a used feature set AND a
  committed UI direction heading toward the Phase-2 alpha build.
- **Clean slate STILL HOLDS (verified 2026-07-20):** no git tags, no `VERSION` file, no
  `wails.json` productVersion, no `-ldflags -X` version injection. This phase starts from zero.
- **This phase is the gate in front of the build track.** The UI roadmap's build track
  (B-1…B-5 → ALPHA) hard-depends on this phase finishing first (stamped builds + the D-V6 schema
  migration story before any binary leaves the machine). Versioning is not optional polish — it is
  the unblock for Alpha. See [[project_thewarroom_ui_roadmap]] + [[project_thewarroom_versioning_releases]].
- When D-V2 wiring lands and you tag current main as the first version, pick the number once D-V1
  is locked (the scaffold's `v0.4.0` placeholder now sits behind M2 + the A–E merge — re-derive it
  from whatever cadence rule D-V1/D-V4 settle, don't inherit the placeholder blindly).

## Read first
1. `docs/roadmap/Versioning_and_Releases_Planning.md` — the framing scaffold: current
   clean-slate state + 7 framed decisions (D-V1…D-V7) + a recommended first-session
   sequence. **Start here.**
2. Project `CLAUDE.md` build-state header (M2 slice-1 is the latest merge).
3. This is a Business+Technical convention-setting phase — the versioning SCHEME
   (D-V1) is a durable decision; run the [[feedback_expert_panel_decision_gate]] before
   locking it.

## The clean slate you inherit
- No `wails.json` productVersion, no `VERSION` file, no git tags, no ldflags version
  injection. Release process today is just branch → gates → live gate → squash-merge.
- Build_Tracker rows + `handoffs/NN-*.md` are a rich changelog SOURCE to distill from.
- `rulebook.ActiveVersion` (in-app league-config versioning) is SEPARATE from app-release
  versioning — don't conflate them.

## Suggested opening moves (from the scaffold, react/adjust)
1. Lock D-V1 (scheme — SemVer `0.y.z` recommended) + D-V2 (source of truth — git tag →
   `-ldflags -X main.version`, synced to `wails.json`, surfaced in a UI "About").
2. Wire the minimum end-to-end and tag current main as the first version; prove the
   binary reports its own version on the Beelink (live gate the version surface).
3. Stand up changelog distillation from Build_Tracker + tags.
4. **Open D-V6 (schema/data migration across releases) as its own design thread** — the
   real risk once a binary ships to a league member; see
   [[lesson_thewarroom_db_corruption_probe]]. Scope it; don't half-answer it.
5. Defer cross-compile/signing (D-V3), cadence (D-V4), auto-update to when Phase-2
   distribution timing is real — capture the questions, don't force answers.

## Also still open (independent of versioning)
- M2 slice-2: `weeklyResults` optimal-lineup columns (Bench/Max/Min PF/Coulda-Won/
  Woulda-Lost) + movement history.
- The league-calendar branch (`session/league-calendar`, backend WIP — separate).
- Other Build_Tracker items.

## Guardrails (unchanged)
No work on main; branch `session/<desc>`. Every code change: make lint 0 / `go test
-race` / tsc+vite / GLM-5.2 blind review on non-trivial work / live Beelink functional
gate for anything display-bearing. Beelink clone hygiene per
[[reference_beelink_functional_gate]] before/after each gate.
