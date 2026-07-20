HANDOFF — Session 48: Scouting FILM Calibration (Thread C) + the deferred refactors
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: the scouting wire-up arc. Handoff 47's Thread A (S-Phase 4b IDP BreakoutAge) and
Thread B (crosswalk consolidation) are BOTH now MERGED (main HEAD `971ae50`). The last unwired
scouting group is FILM — and it is the one that is NOT a plumbing clone.

== WHERE WE ARE (main HEAD 971ae50) ==
- WIRED end-to-end (fetch → crosswalk join → Profile → engine Layer 4): RAS (S-Phase 0),
  SchoolTier (1), offense CollegeShare (2), IDP CollegeDefense (3), OFFENSE BreakoutAge (4),
  IDP BreakoutAge (4b, threshold 0.12 single line).
- CROSSWALK + BIRTHDATES now fetched ONCE (Thread B) in `m1_scouting.buildScoutingDirectory`
  and threaded into every assembler. Any new signal reuses those maps — do NOT re-fetch.
- The assembly template: `internal/scouting/assembly/{ras,schooltier,collegeshare,
  collegedefense,breakoutage,breakoutage_idp}.go` + `m1_scouting.go` (per-signal merge helpers).
- STILL-DEFERRED-NEUTRAL: AgeTrajectory (Profile field vestigial — age derives from spec.Age;
  confirm before wiring); the FILM components (OffenseFilm + IDPFilm + NGSCoverage) + K film.

== THREAD C — FILM CALIBRATION (the real work; decision-gated, PLANNING-FIRST) ==
This is a WEIGHTS/calibration program, NOT a fetch→Profile clone. Do NOT ship blind film weights.
- The original film sources (OffenseFilm/IDPFilm/NGSCoverage) were ELIMINATED — see the
  SOURCE-DRIFT notes on each type in `internal/scouting/types.go`. The redesign is built around
  Madden sub-attributes + NFLProduction + pfrcoverage, with the blend WEIGHTS currently UNSET.
- Frame it as its own planning-first program (like Versioning & Releases): before any wiring,
  (1) confirm which live sources actually exist + are reachable + zero-leak-clean, (2) sample
  their live distributions (same discipline as the 4b threshold sample — 25k-season live pull →
  candidate reads → Christopher decides, per [[lesson_inference_vs_calibration]]), (3) bring
  Christopher the weight/normalization decisions as an AskUserQuestion gate. Only then wire.
- Because it sets durable convention (a scouting sub-weight scheme), the weight scheme is an
  expert-panel decision-gate candidate ([[feedback_expert_panel_decision_gate]]), not a solo call.
- Reuse the merged shape: caller-supplied crosswalk Map + birthdates, assembly leaf owns any
  multi-season loop, engine stays pure, Profile a leaf, 400-line cap / funlen 40 / zero-leak.

== INDEPENDENT ALTERNATIVES (any can go first — Christopher picks) ==
- M2 service extraction (handoff 43) + M4 read-model store (handoff 44) — the two SLIM_MAP
  refactors staged but not done.
- M2 Power Rankings slice-2 ([[project_thewarroom_m2_power_rankings]]) — the 5 weeklyResults
  optimal-lineup columns + movement history + early-season confidence badge.
- League calendar backend branch ([[project_thewarroom_league_calendar]]).
- Versioning & Releases ([[project_thewarroom_versioning_releases]]) — Christopher-led, the
  named next-phase before Phase-2 alpha; planning-first.
- Harness 3G/3H (Module 3 pending cases, [[lesson_thewarroom_module3_pending_3g_3h]]).

== EXECUTION MODEL ==
INVERTED gate: GLM 5.2 writes heavy wiring, Claude head-brain reviews vs source + owns method
decisions + gates. GLM has been DOWN through S-Phase 1–4b → if still down, Claude writes directly
+ self-reviews (waive GLM with a note, same posture). Any calibration threshold/weight is
Christopher's, planning-first.

== CLOSE GATE (unchanged) ==
- Build green: `make lint` 0 (golangci + ifaceguard + filelen 400-cap) + `go test -race ./...` green.
- Review (GLM 5.2 blind, leads-not-findings, triage vs source) if GLM is back; else waive w/ note.
- Functional check (Beelink live gate): reset clone to clean main before/after; steps to /root/paste.md.
- FILM specifically: live-sample the sources BEFORE picking weights; a clean Score League with the
  new sub-signal moving the board sensibly (no blind weights, no NaN).

== CARRIED ==
- Open items: OQ-013 (created→official id ramp), OQ-014 (Money type), harness 3G/3H pending,
  FILM weights UNSET (this handoff), AgeTrajectory field vestigial (confirm before any wiring).
- ⚠️ ROTATE the free CFBD key at full beta. ⚠️ ROTATE the z.ai GLM_API_KEY.
- Memory: [[project_warroom_review_week]], [[project_thewarroom]], [[lesson_inference_vs_calibration]],
  [[feedback_expert_panel_decision_gate]].
