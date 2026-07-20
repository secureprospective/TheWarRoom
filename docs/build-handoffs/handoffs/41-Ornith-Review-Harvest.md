# Handoff 41 — Ornith Review Check / WarRoom Review-Week Harvest

**From:** UI wireframe track A–E close (2026-07-20, main `d137a68`).
**For:** the next session that runs the Ornith review check — triaging the autonomous
WarRoom Review Week output (GLM 5.2 + Ornith) into training data + a couple of real fixes.
**Type:** collaboration/harvest session (GOAL 2 primary, GOAL 1 secondary). NOT a build row.

## The entry point — read this, it IS the full brief
`docs/reviews/warroom-review-week-2026-07/SESSION_PROMPT_review-harvest.md` (rev 2, 2026-07-18)
is the complete, governing session prompt. This handoff only **refreshes its state** — do not
re-derive it; open that prompt and execute it against the freshness notes below.

Supporting artifacts (all on CT105 at `docs/reviews/warroom-review-week-2026-07/`):
- `REVIEW_LOG.md` — GLM 5.2 BLIND review, **13 findings across 8 chunks** (complete).
- `ARCHITECTURE_MAP.md` + `boundary_rules.json` — Ornith arch-map: 1043 import edges →
  **17 true-positive boundary anomalies (22% precision / high recall)**.
- `ornith-phase0-grades.md` — the Ornith training-grade record (structure B+ / completeness F;
  boundary-rules B+). Doctrine: `/root/.claude/plans/local-model-teammate-doctrine.md`.
- `SOURCES.md` — the 4 authoritative URLs that hardened the Phase-3 findings.
- Transcripts on the Beelink: `/home/chris/opencode/review_logs/warroom/transcripts/`.

## ⚠ STATE REFRESH — what changed since the prompt was written (2026-07-18 → 2026-07-20)
- **The review ran on a clone PINNED to main `f624467`. Current main is `d137a68` — 32 commits
  ahead.** Landed since the pin: M2 Power Rankings slice-1 (`686a574`) AND the entire UI
  wireframe track A–E (`e82f01a`, all design-language sessions, merged). **Every lead must be
  re-verified against current main before any fix** — the prompt already says this; it is now
  emphatically true (32 commits of drift).
- **Priority is unchanged and explicit (Christopher, in the prompt): GOAL 2 is PRIMARY** — this
  review is mostly TRAINING DATA for shaping Bender/Ornith as a valued local-model teammate under
  the weight-agnostic doctrine. **GOAL 1 (TheWarRoom code fixes) is SECONDARY** — do it only after
  Goal 2 lands, or skip to just the two HIGH leads.
- **The two HIGH leads to re-check first** (Goal-1 tier 1, may have shifted in 32 commits):
  - **TWR-2** — §9 tag price not snapped to the $10k grid (`pricing.go` / `contracts.go`). GATING
    triage check first: does `state.TxWriter.SetCell`/`ApplyContract` snap internally? YES → no-op,
    close. NO → one `RoundToNearest10k` snap + a plant test (mirror `TestIntegration_CapReliefSnapsToGrid`).
  - **TWR-1** — unbounded MFL fetch on the Wails `OnStartup` UI thread, no timeout (`app.go`
    `initStoreFloor`). THE KNOWN BLACK-SCREEN BUG CLASS + the #1 salary-ledger cutover fast-follow.
    Fix = `context.WithTimeout` and/or move seeding off the UI thread; rides a Beelink functional gate.
- **Ornith's 22%-precision root cause is by-design, not a defect:** it flagged edges against
  boundary rules it INFERRED (no path→chunk table). The repo has a REAL depguard config — check each
  flagged "leak" against the actual depguard before treating it as a violation (root/binding→ingestion
  IS allowed). LEADS, not findings — triage every item against source. [[feedback_glm_code_reviewer]]

## Guardrails
Branch fresh off main (`git checkout main && git pull && git checkout -b session/review-harvest`).
Any Goal-1 fix runs the full gate (make lint 0 / `go test -race` / tsc+vite / GLM-5.2 blind review /
live Beelink gate for anything startup/display-bearing). Beelink clone hygiene per
[[reference_beelink_functional_gate]]. See memory [[project_warroom_review_week]] +
[[project_hermes_pr_trainer]] + [[reference_codeword_grade_the_logs]].
