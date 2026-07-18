SESSION PROMPT — Harvest the WarRoom Review Week (GOAL 2 primary)
Project: Hermes/Ornith/Bender teaming (+ TheWarRoom as the corpus) · Node: ClaudeBox / CT105 (SSHes to Beelink)
Written: 2026-07-18 · rev 2 · Source work: WarRoom Review Week 2026-07-12..15 (GLM 5.2 + Ornith, both COMPLETE)

WHAT THIS IS — AND THE PRIORITY (Christopher, 2026-07-18)
The autonomous review week finished: GLM 5.2 logged 13 findings across 8 chunks; Ornith's arch-map
classified 1043 import edges into 17 true-positive boundary anomalies (22% precision / high recall).
**The point of this session is NOT the code fixes.** Christopher's framing: the review is mostly
TRAINING DATA. The HIGHEST-PRIORITY work is GOAL 2 — making Bender a valued team member and shaping
a local-model collaboration doctrine that will still hold when today's tiny Ornith (well under 1% of
Claude's size, but deliberately well-designed) is swapped for much heavier local weights later.
GOAL 1 (the TheWarRoom fixes) is SECONDARY — do it only after Goal 2 is landed, or skip to just the
two HIGH leads. Someday heavier local weights run these jobs; this doctrine is how we shape Bender
and local models for that future. ClaudeBox needs Bender as a valued team member.

READ FIRST — the shaped doctrine this session executes:
  /root/.claude/plans/local-model-teammate-doctrine.md  (v1.0 — the weight-agnostic collaboration
  contract: small models own high-recall SIGNAL + UNCERTAINTY, frontier owns JUDGMENT + teaches
  specificity back; valued-teammate operationals; invariant-vs-temporary; real-vs-theater loops).
Artifacts on CT105: docs/reviews/warroom-review-week-2026-07/ (REVIEW_LOG.md, ARCHITECTURE_MAP.md,
ornith-phase0-grades.md, boundary_rules.json). Transcripts on the Beelink at
/home/chris/opencode/review_logs/warroom/transcripts/. Ornith driver skill: Beelink
~/ai-workspace/skills/arch-map/. SSH: ssh -i /root/.ssh/beelink chris@192.168.1.190.

GOVERNING DOCTRINE (do not skip)
- LEADS, not findings. GLM ran BLIND; Ornith at 22% precision. This is not a defect to apologize
  for — high recall + frontier triage IS the architecture (see doctrine §1). TRIAGE every item
  against source. [[feedback_glm_code_reviewer]]
- The review ran on a clone PINNED to main f624467. Current main has moved. Re-verify leads.
- Ornith flagged import EDGES against boundary rules it INFERRED (no path→chunk lookup table — the
  exact 22%-precision root cause). The repo has a REAL depguard config; an edge Ornith called a
  "leak" may be explicitly ALLOWED (root/binding→ingestion IS by design). Check each against the
  actual depguard before treating it as a violation.

=====================================================================
GOAL 1 — 3rd-PARTY IMPROVEMENT TO TheWarRoom
=====================================================================
Read docs/reviews/warroom-review-week-2026-07/REVIEW_LOG.md + ARCHITECTURE_MAP.md first.
Branch fresh off main: git checkout main && git pull && git checkout -b session/review-harvest.
Work the leads in priority order; each gets triaged, then either fixed (with a test) or
recorded as a documented accept-tradeoff. Build env + gates per the project CLAUDE.md.

TIER 1 — the two HIGH leads (triage first, they carry the most risk)
  [TWR-2 · §9 tag price not snapped to $10k grid] pricing.go:66,105 + contracts.go:209,218.
    THE GATING TRIAGE CHECK (do this exact check before any fix): does state.TxWriter.SetCell /
    ApplyContract snap to the $10k grid internally? If YES → TWR-2 is a no-op, document and close.
    If NO → an off-grid tag price (e.g. $12k-granularity floor from tagFloorPrice) enters the
    ledger and breaks the universal FLAT-$10k CapUsed invariant that every OTHER money path holds
    (deadcap/extension/buyout/retirement/cap-relief/sign all call domain.RoundToNearest10k). Fix =
    one RoundToNearest10k snap on the resolved tag price, plant a test pinning an off-grid input →
    on-grid ledger (mirror TestIntegration_CapReliefSnapsToGrid).
  [TWR-1 · unbounded MFL network fetch on the Wails OnStartup UI thread, no timeout] app.go
    initStoreFloor → rulebook.Initialize + roster seed, raw OnStartup ctx with no deadline. THIS
    IS THE KNOWN BLACK-SCREEN BUG CLASS — the -probe diagnostic exists because of it, but the root
    cause is unfixed. This is also the #1 carry-forward fast-follow from the salary-ledger cutover
    ("move app.startup/initStoreFloor OFF the UI thread"). Cross-refs arch anomaly A2. Fix =
    wrap initStoreFloor's ctx in context.WithTimeout (the probe validated 20s/step) AND/OR move
    seeding off the UI thread (goroutine + runtime.EventsEmit progress + shell loading state +
    defer recover), surface the deadline via startupErr/Ping. Verify on the Beelink (it's a
    startup-path change) — this one likely rides a functional gate.

TIER 2 — MEDIUM correctness/robustness leads (triage, fix the real ones)
  - rulebook.go Initialize: two writes (insertVersion + setActive) on the fresh-DB path with no
    wrapping tx → orphaned version row possible. Fix = one BeginTx/Commit. (arch A1 related)
  - frontend bridge-call error handling: harness.ts setParam (no try/catch → unhandled rejection,
    error slice never set) and TransactionsPanel.tsx run()/refreshPhase/loadFranchise/
    refreshFreeAgents (no catch → blank panel, lost rejection). Mirror the loadRankings try/catch +
    set({error}) pattern. NOTE: TransactionsPanel is the OLD dev panel — confirm it's still live
    and not superseded by the M4 transactions/ workspace before investing.
  - probe.go:32 goroutine no recover — MEDIUM only (probe-only blast radius, every path os.Exit).
    Low value; document or add a recover that logs+os.Exit(2).

TIER 3 — LOW / hygiene (batch these; don't over-invest)
  - Unstructured logging → slog: state.go:328 (runtime load path, logs domain data), freeagency.go:99,
    app.go:128. Consistency with the structured-logging mandate.
  - app.go:213 _ = pools.Close() drops the DB-close error — given the DB-corruption history a WAL
    checkpoint error is a corruption signal worth logging before exit.
  - ping.ts stuck-loading on rejection — LOW, the store is DEAD CODE (zero importers); fix only if
    wiring it up.

TIER 4 — dependency hygiene (own small commit, verify each)
  - Go toolchain 1.26.4 → 1.26.5 (fixes GO-2026-5856 crypto/tls ECH, which IS code-affected via
    4 call traces; also GO-2026-4970 not-called). x/sys → v0.44.0+ (GO-2026-5024, not-called).
  - vite ^3.0.7 → >=6.4.3: 14 known + 2 newer HIGH advisories (arbitrary file read via dev-server
    WS, fs.deny bypass). All dev-server-only (not in the shipped Wails binary) but a multi-major
    jump that also needs @vitejs/plugin-react ^2→^6 and typescript ^4→^5. SCOPE THIS CAREFULLY —
    it can break the frontend build; may deserve its own dedicated session. Confirm with Christopher
    before doing the big jump; the toolchain bumps are safe to do now.

ARCHITECTURE ANOMALIES (A1–A5) — adjudicate against the REAL depguard, don't blind-fix
  A1 store/rulebook→ingestion, A2 wails→ingestion (8 edges, = TWR-1's root), A3 ingestion/
  schooltier→scouting, A4 engine(composition/output)→store/db direct, A5 output→sqlite lib.
  For EACH: is this edge already permitted by the repo's depguard rules (many are — the binding
  surface IS the hub; schooltier→scouting was a DOCUMENTED exception when built)? If depguard
  allows it, it is NOT a violation — Ornith inferred boundaries the repo doesn't enforce. The one
  with real teeth is A2's overlap with TWR-1 (fix the timeout, not the import). A4 (engine imports
  store/db directly instead of via interfaces) is the one genuine "should-use-interfaces" smell —
  weigh it as a refactor lead, not a bug; likely a documented carry-forward, not this session.

GOAL-1 CLOSE: lint 0 / go test -race green / tsc+vite clean per fix. Commit each triaged cluster
separately with a message citing the GLM/arch lead id. GLM-review the correctness fixes over SSH
([[reference_glm_review_over_ssh]]) before any merge. Real startup/tag changes ride a Beelink
functional gate. Write a REVIEW_HARVEST_OUTCOME.md recording every lead → {fixed+test | accepted-
tradeoff | already-fixed | false-positive-vs-depguard}, the triage reasoning for each.

=====================================================================
GOAL 2 — SHAPE BENDER + LOCAL MODELS AS VALUED TEAMMATES  *** PRIMARY — DO THIS FIRST ***
=====================================================================
Execute the doctrine at /root/.claude/plans/local-model-teammate-doctrine.md against this corpus.
The point of grading 3rd-party work is that the NEXT autonomous review is better AND that Bender/
Ornith become better teammates — the WarRoom review is the first corpus for a loop that outlives
Ornith's current weights. This is the "grade the logs" self-learning loop
([[reference_codeword_grade_the_logs]], [[project_hermes_pr_trainer]]) applied here.
Artifacts on the Beelink: /home/chris/opencode/review_logs/warroom/ (transcripts/, REVIEW_LOG.md,
ornith-phase0-grades.md, arch-map/lessons.md). Ornith driver skill: Beelink
~/ai-workspace/skills/arch-map/. SSH: ssh -i /root/.ssh/beelink chris@192.168.1.190.

THE FRAME (doctrine §1 — apply it, don't relitigate it): Ornith's 22% precision was a TASK-DESIGN
failure, not a size failure — it was handed a judgment prompt for a filtering job. Cheap high-recall
signal + frontier precision-triage is the CORRECT architecture; design around it. The sharper move
to build toward: teach the small model to OWN UNCERTAINTY QUANTIFICATION ("here are the patterns;
I'm unsure about these 3 boundary cases — which is the rule?") rather than reach confident wrong
answers. That scales to any weight and is high-status work, not assistance.

TRAIN ORNITH (the arch-mapper — the biggest, most concrete win)
  Root cause of Ornith's 22% precision (from ARCHITECTURE_MAP.md "Ornith Precision Analysis"): the
  classification prompt gave chunk-level may/must-not-import rules but NO mapping from Go import
  paths to chunk names, so Ornith flagged every cross-package import as a violation (recall was
  high, precision terrible). CONCRETE FIX to bake into the arch-map skill:
    1. Pre-compute a domain→chunk lookup table (path → chunk) and inject it into the classification
       prompt, OR classify programmatically (deterministic graph math already exists in the kit)
       and use Ornith ONLY for the architectural INTERPRETATION of already-flagged edges. The
       ARCHITECTURE_MAP.md itself prescribes this — implement it in the skill.
    2. Fold the reasoning-model pipeline lessons already proven (ornith-phase0-grades.md "Pipeline
       lessons"): dir-level digests not file lists, "keep hidden reasoning brief" line, max_tokens
       ≥2× expected answer, read reasoning_content for diagnostics, ~3 tok/s budget. Confirm these
       are IN the skill, not just in a grades doc.
    3. Add the mechanical coverage check Ornith skipped (Phase-0 error #1): after grouping, diff the
       union of chunk paths against the inventory and assign the remainder — completeness is a
       mechanical post-check, do it mechanically.
  Write these as lessons into Beelink ~/ai-workspace/skills/arch-map/lessons.md AND an
  ornith-lessons.md (parallel to the PR-trainer's ornith-lessons.md), so the next arch-map run
  loads them. This is real training: the skill the model runs under is the model's behavior.

GRADE GLM (per-rule precision → reviewer calibration)
  Per the post-trip plan: triage REVIEW_LOG.md leads-not-findings (Goal 1 does the triage), then
  grade GLM per-rule precision and write glm-review-lessons.md — which rules GLM over/under-fired,
  where "invisible snapping" hedges appeared (TWR-2's own triage-check hedge), where it was
  precise. This tunes the review prompt/rubric for the next chunk review. Keep it leads-not-findings
  doctrine: GLM's value is high recall + cheap; Claude's triage is the precision layer.

TRAIN BENDER / HERMES (the orchestrator + collaborator)
  Bender is the Hermes Telegram persona (VM 502) that ORCHESTRATED this review (systemd timers,
  warroom plugins, watchdog, notify). Capture the collaboration doctrine it should carry so it runs
  the NEXT review loop better and presents results correctly:
    - "Leads, not findings" is the reviewer contract — Hermes must surface GLM/Ornith output AS
      leads for Claude to triage, never as settled findings. Encode this in the review skill's
      output framing.
    - The self-learning loop shape (poller → reviewer → "grade the logs" → Claude grades vs source
      → lessons → next run improves) is PROVEN for the PR-trainer; the WarRoom review is the same
      loop on a different corpus. Wire the WarRoom review lessons (glm-review-lessons.md +
      ornith-lessons.md) into the same lessons-feed the review skill reads, so grading actually
      changes the next run.
    - Record any orchestration friction found in the review-week transcripts (timer/plugin/watchdog
      behavior) as Hermes-side lessons.
  Coach Hermes, don't do its job by hand — Hermes runs the task, we write the lessons it learns
  from ([[project_hermes_node]]). Drop the lesson files where Hermes/the skills read them; verify
  by inspecting the skill, not just the doc.

GOAL-2 CLOSE: arch-map skill updated on the Beelink (path→chunk table / interpretation-only mode +
coverage post-check + reasoning-model lessons confirmed in-skill); glm-review-lessons.md +
ornith-lessons.md written where the review skill reads them; Hermes review skill reframed to
leads-not-findings with the lessons feed wired. Reset any Beelink review clone to clean per
[[reference_beelink_functional_gate]] hygiene if touched.

=====================================================================
SESSION CLOSE (both goals)
=====================================================================
- Update project CLAUDE.md build state (what got fixed from the review) + a handoff.
- Update memory: [[project_warroom_review_week]] (mark harvested), [[project_thewarroom]],
  and the Hermes/Ornith training memories.
- Append to /root/.claude/backbone/context.md; commit the backbone.
- Standing git authorization applies after gates pass ([[feedback_claude_runs_git_ops]]).
PRIORITY NOTE (Christopher, 2026-07-18): Goal 2 is the point of this session — land it FIRST and in
full (Ornith machinery fix + reasoning-model lessons in-skill + coverage post-check + glm-review-
lessons + Bender leads-not-findings framing + the doctrine committed). Goal 1's code fixes are
secondary training-data harvest: if time is short, do ONLY the two HIGH leads (TWR-2 tag-snap triage,
TWR-1 startup timeout) and record the rest as leads for later. The measure of success is a better
NEXT autonomous run and a stronger Bender — not lines of TheWarRoom changed. A/B check: re-run the
arch-map on the same input later; if false-positive rate drops, the loop (and the teammate) improved.
