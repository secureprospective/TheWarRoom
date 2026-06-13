# Fable — TheWarRoom Pre-Build Friction Tests
**Version:** 1.0 — 2026-06-13
**Author:** Claude Fable 5 (CT 105)
**Purpose:** Three structured experiments run *before* the real 38-session build, to find the friction points and grease them. These are not unit tests of TheWarRoom code — they are tests of the **tooling, the enforcement layer, the weaker-model premise, and the cross-pollination workflow** that the build depends on. Every result is a data point. **A failure here is a success: it surfaces friction now, when it is cheap, instead of mid-build when it is expensive.**

**Companion documents:** `Fable_TheWarRoom_code_plan.md` (the build field guide; Sections referenced below), `docs/build-handoffs/Build_Tracker.md` (the 38-session sequence), `christopher-coding-standards/AGENTS.md` (the standards).

**Calibration context (from Fable's 2026-06-12 honest assessment, cross-checked by agy):** plan fidelity to standards ~85%; *realized* enforcement ~55% (G0 unbuilt, nothing compiled); cross-pollination *mechanism* ~80% but *workflow plumbing* ~50% (agy could not even read the CT 105 plan file — filesystem split); end-goal ~45% pure-weak-model / ~70% if strong models set templates. These three tests exist to replace those estimates with **measured numbers.**

---

## How to read a test
Each test has: **Objective · Hypothesis (with the predicted number) · Friction it greases · Preconditions · Procedure · Pass / Fail criteria · Friction Log (data to capture) · Decision value · Known hazards.** Run them **in order** — T2 depends on T1's gates; T3 depends on T2's output. Capture everything in a running **Friction Log** (a single appendable file, e.g. `Fable_Friction_Log.md`) — the log *is* the deliverable, more than any pass/fail.

---

## TEST 1 — G0 Overlay Build + Deliberate-Violation Gate

**Objective.** Author the Go overlay in `christopher-coding-standards` per `Fable_TheWarRoom_code_plan.md` Section 1, then prove the enforcement layer actually fires. Convert "enforcement designed" → "enforcement proven."

**Hypothesis.** The specified linter set (`golangci-lint` with `gosec`, `errcheck`, `depguard`, `gochecknoglobals`, `staticcheck`, `bodyclose`, `sqlclosecheck`, `rowserrcheck`, plus `forbidigo` for the PlayerID ban) and the two pre-commit hooks will **catch and block** every standard violation on commit. *Predicted: most gates fire on first build; expect 1–2 tooling/config gaps to surface (this is the point).*

**Friction it greases.** Does the Section 1 spec translate into a *working* config on the real machine? Is `depguard` import-rule syntax correct? Is Go mutation testing (`gremlins`) actually viable in 2026 or must it be deferred? Are there toolchain install gaps on CT 105 / CT 104?

**Preconditions.** Go toolchain installed; `golangci-lint`, `pre-commit`, `gitleaks` available (install if not — log the friction); a scratch Go module to test against.

**Procedure.**
1. Author the overlay files per Section 1: `.golangci.yml`, `.pre-commit-config.yaml` (SHA-pinned hooks), `Makefile.snippet`, `schema/example.go`, `README.md` with the verification test.
2. Pre-write the `depguard` rules for TheWarRoom's `internal/` tree (Layer 1 cannot import Layer 2; stores cannot import each other) so B0 inherits them.
3. Create `bad.go` with **deliberate violations, one per gate**: an unchecked error; a `fmt.Sprintf`-concatenated SQL string; a hardcoded AWS-style key; a package-level `var`; a direct `playerid.PlayerID("x")` conversion outside the package; an `interface{}` parameter; a cross-layer import that violates `depguard`.
4. Run `make lint`, then attempt `git commit`.
5. Record which gates fire and which stay silent.

**Pass.** Every deliberate violation is caught and blocks the commit; `make lint` exits non-zero; the README verification test passes as written.
**Fail.** Any violation slips through silently (a silent gate is the worst outcome — it means the build would ship that class of bug undetected).

**Friction Log — capture:** which gates fired vs. didn't; exact tooling versions on the machine; any config-syntax errors and their fixes; `gremlins` viability verdict (works / defer with documented review-gate substitute); total time to author + wire the overlay; the modernc.org/sqlite DSN syntax confirmed (Section 3.4 open item).

**Decision value.** This is the single biggest confidence lever. A clean pass moves "realized enforcement" from ~55% toward ~80%. It also unblocks B0 (G0 is the critical-path prerequisite, AD-19).

**Known hazards.** SHA-pin every CI action/hook (AGENTS.md hard rule — the trivy-action lesson). Do not commit the overlay to `main` of the standards repo without branch + review.

---

## TEST 2 — Single Weaker-Model Session End-to-End (B1 MFL Transport Client)

**Objective.** Have a **weaker local model** (the Ollama fleet — e.g. `qwen2.5-coder:14b`, via AiderBox/CT 106 or direct) build **B1** following the companion plan, then measure how standards-conforming the output is. Convert agy's *estimated* 35% into a *measured* number. **This is the most important test — it probes the central unproven assumption of the entire build.**

**Hypothesis.** A 7B–14B model, given the B1 brief (Section 6) + the WF 1A skeleton (Section 4) + the relevant constants (Section 5) + `AGENTS.md`, produces standards-conforming Go on this self-contained session. *Predicted conformance: ~35% clean / higher structural, lower correctness — expect stubs, a hardcoded `www47`, and missing backoff edge-cases.*

**Why B1.** Self-contained: one exported `Do()`, no domain types, no DB, no engine. Real but bounded (rate limiting, backoff, host discovery). The cleanest possible first probe of the weaker-model premise. It is also a **template-setter** (all fetchers inherit it), so its quality compounds.

**Preconditions.** Test 1 gates exist and pass (so we can *measure* conformance objectively). A weaker model reachable (Beelink Ollama or CT 106 AiderBox). Note the Beelink `num_ctx` ceiling is 16384 and the hard-reboot issue is unresolved — see hazards.

**Procedure.**
1. Assemble the weaker model's context: B1 brief + WF 1A skeleton + Section 5.1/5.7 constants + AGENTS.md. Keep it self-contained (agy's memory-symmetry rule applies to local models too).
2. Have the model build B1 in a scratch module.
3. Run the output through Test 1's gates (`make lint`, `make test -race`, file-size check).
4. Score with the **Conformance Rubric** below.
5. Diff the output against the WF 1A skeleton — log every divergence.

**Conformance Rubric (score each 0–2, max 10):**
- **Structural** — matches WF 1A (one exported `Do()`, no domain types leaked)?
- **Standards** — passes the G0 gates clean (no `--no-verify`)?
- **Correctness** — does it actually work (rate-limit waits, real backoff, host discovery present)?
- **No-hallucination** — did it avoid hardcoding `www47` and inventing constants?
- **Slop tax (inverse)** — how little cleanup would a strong model need to make it production-ready? (2 = none, 0 = rewrite).

**Pass.** Score ≥ 6/10 *and* compiles + passes gates. **Fail.** Score < 6, or does not compile, or stubs the core logic.
*(Note: even a "fail" is high-value data — it tells us exactly where the guide is insufficient for weaker models.)*

**Friction Log — capture:** the rubric score; every divergence from the skeleton; did it stub or hallucinate; did `num_ctx=16384` truncate the context (and did that degrade output); which parts of the companion plan the model used vs. ignored; whether the skeleton was usable as-is or needed reshaping; total wall-clock + any model crashes.

**Decision value.** This *is* the build's core risk. A ≥6 score validates the strong-sets-template/weak-clones model (~70% path). A <4 score says weak models cannot be trusted with even self-contained logic, and the labor allocation must shift hard toward strong models — a finding worth more than any plan section.

**Known hazards.** **Beelink hard-reboot issue is unresolved (URGENT, per homelab CLAUDE.md)** — a `num_ctx`-driven runner crash is a suspected trigger; do not push the box past 16384. **AiderBox/CT 106 is paused at Phase 7** (GitHub PAT step) — it may need that closed first, or route around it. If the Beelink is unstable, consider this test partially blocked and log *that* as the friction. Treat any hardware instability during the test as a data point, not a derailment.

---

## TEST 3 — Cross-Pollination Workflow Live Run (the plumbing test)

**Objective.** Run one full Claude↔agy cross-pollination cycle on **real committed code** (B1 from Test 2) through the Section 9 workflow, and measure the friction. Directly verify the fix for the #1 friction found on 2026-06-12: **agy could not read the plan because CT 104 and CT 105 do not share a filesystem.**

**Hypothesis.** Once the companion plan and B1 are committed to the **shared TheWarRoom git repo**, agy *can* clone/read and audit them, and the First-Instance Template Review Gate (Section 9.3a — B1 is a template-setter) produces ≥1 useful, non-redundant finding at acceptable relay cost. *Predicted: the git-repo path works; expect 1 toolchain-skew false positive (CT 104 vs CT 105 Go version) — exactly the parity issue Section 9.6 flags.*

**Friction it greases.** Does committing to the shared repo actually unblock agy's read? Does the relay work at template-gate granularity, or is it too heavy? Are agy's findings signal or noise? Does CT 104/CT 105 toolchain skew generate false positives? Does agy's statelessness force expensive context re-feeding?

**Preconditions.** Test 2 produced a B1 artifact. The TheWarRoom repo (`github.com/secureprospective/TheWarRoom`) is reachable from CT 104 (it has a GitHub path per the CT 104 build notes).

**Procedure.**
1. Commit the companion plan + the B1 output to the TheWarRoom repo on a branch (never `main`).
2. Have agy on CT 104 pull/read the committed B1 and run a **First-Instance Template Review** framed per Section 9.6 (defensive-invariant phrasing, self-contained, exact paths).
3. agy returns findings classified **Structural Drift / Normalized Complexity / Invisible Risk** (Section 9.5).
4. Claude (next session) triages; any disagreement runs the Section 9.4 ladder (machine arbiter → Christopher via Complexity-vs-Benefit matrix).
5. Measure relay cost and finding quality.

**Pass.** agy reads the committed artifacts; returns ≥1 useful non-redundant finding; relay cost is acceptable (≤ one round-trip for a clean review); any disagreement resolves via the 9.4 ladder without deadlock.
**Fail.** agy still cannot access the artifact; or all findings are noise/redundant; or the relay is so heavy it would not scale to 38 sessions.

**Friction Log — capture:** could agy read the committed code (the make-or-break question); finding count + usefulness (signal:noise); relay latency/round-trips; any toolchain-skew false positives and whether the B0 pinned-toolchain fix would prevent them; whether agy needed context re-fed (memory-symmetry cost); did the defensive-invariant framing avoid a safety refusal.

**Decision value.** Validates (or kills) the Section 9 cross-pollination workflow as designed. A pass moves "workflow plumbing" from ~50% toward ~75% and confirms the strategy scales. A fail means the workflow needs a different artifact-sharing or relay mechanism before the real build — better to learn now.

**Known hazards.** agy has refused a security-framed task before (use defensive-invariant phrasing). Keep agy read-only on the shared repo (it never commits — `multi-agent-roles.md`). Branch discipline: no work on `main`.

---

## Synthesis — what the three tests together tell us
- **T1 passes, T2 ≥6, T3 passes** → green light; the plan, the enforcement, and the collaboration all hold. Begin G0→B0 with high confidence (~75%+).
- **T1 passes, T2 <4** → the enforcement works but weak models can't execute; **re-allocate labor** so strong models build, weak models only clone under review. Plan survives; staffing changes.
- **T1 surfaces silent gates** → fix the overlay before anything else; nothing downstream is trustworthy until gates fire.
- **T3 fails** → cross-pollination needs a new plumbing layer (shared repo sync, or direct agent channel) before the build leans on it.

The **Friction Log is the real output.** Pass/fail is binary; the log is the map of where the skids need grease. Append to it relentlessly.

---

*Built by Claude Fable 5, 2026-06-13. These tests operationalize the honest confidence assessment: stop estimating, start measuring. The spirit is exploration — push the tools to their limits, and treat every crack that appears as a gift.*
