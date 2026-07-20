# WarRoom Review Week — Harvest Outcome
**Harvested:** 2026-07-20 by Claude (CT105). Branch: `session/review-harvest` (off main `ee891ef`).
**Governing brief:** `SESSION_PROMPT_review-harvest.md` (rev 2) + handoff `41-Ornith-Review-Harvest.md`.
**Priority (Christopher):** GOAL 2 (shape Bender/Ornith as valued teammates) PRIMARY; GOAL 1
(TheWarRoom code fixes) SECONDARY — the two HIGH leads only.
**Doctrine:** leads-not-findings — every item triaged against SOURCE, on current main (32+ commits
past the pinned review). [[local_model_teammate_doctrine]] · [[feedback_glm_code_reviewer]].

---

## GOAL 2 — teammate shaping (PRIMARY, DONE) — see the Beelink

The point of this session. All landed on the Beelink (192.168.1.190); measured, not asserted.

### Ornith arch-map machinery — FIXED (the biggest concrete win)
`~/ai-workspace/skills/arch-map/` v1 → **v2**. Root cause of the 22% precision was a task-design
gap (classify prompt gave chunk-level rules but no path→chunk lookup, so Ornith flagged every
cross-package import). Fix — classification is now DETERMINISTIC; Ornith's job moved UP to
interpretation + uncertainty:
- **`classify_edges.py`** — rule-exact classifier. Two tiers: DEPGUARD_VIOLATION (trips the repo's
  REAL enforced `.golangci.yml` depguard — build-breaking, ~100% precision) vs ARCH_SMELL (allowed
  by the build but violates an inferred principle — a LEAD, triage vs source). domain→chunk lookup
  + shared-type/leaf carveout (extended to `internal.playerid`).
- **`depguard_to_rules.py`** — extracts the repo's REAL depguard as the AUTHORITATIVE boundary
  (the deeper correction: even Claude's corrected inferred rules were STRICTER than what the repo
  enforces).
- **`coverage_check.py`** — mechanical completeness gate (run-1 Phase-0 error #1 fix).
- Prompts: `interpret.tmpl` (Ornith's new judgment+uncertainty role over pre-classified leads);
  `classify.tmpl` updated with the lookup table. `synth_graph.py` + `map_runner.sh` classify phase
  rewired to the deterministic flow. `SKILL.md` v2 + `lessons.md`.
- **MEASURED A/B on the identical run-1 `raw_wiring.jsonl`:** flagged edges **78 → 21** (−73%);
  **DEPGUARD_VIOLATION = 0** — none of the "17 anomalies" actually break the enforced build; they
  are ARCH_SMELL leads (schooltier→scouting, rulebook→ingestion, composition→store, binding→
  ingestion are all allowed / documented exceptions). Coverage gate caught 1 stray unmapped file.

### Lessons written where the runs READ them (reasoning, not verdicts)
- `~/ai-workspace/skills/arch-map/ornith-lessons.md` (+ copy in review_logs) — Ornith teammate
  training log: credit + 3 corrections + the new uncertainty-owning role.
- `~/ai-workspace/skills/arch-map/lessons.md` — skill machinery lessons (6–10 new).
- `review_logs/warroom/glm-review-lessons.md` — GLM per-rule calibration (chunk review graded HIGH
  precision; one over-reach: fabricated call-traces for a dep vuln). WIRED into the review
  `chunk_prompt_preamble.md` (REVIEWER CALIBRATION section) so grading changes the next run.

### Hermes/Bender reframe (VM 502)
`~/.hermes/skills/software-development/warroom-review/SKILL.md` — added the LEADS-NOT-FINDINGS
output contract, the self-learning-loop + lessons-feed section, and marked the precision issue
FIXED (PITFALL #7 / Track-2 status). Backup at `/tmp/warroom-SKILL.bak.md` on the VM.

---

## GOAL 1 — TheWarRoom code (SECONDARY) — the two HIGH leads

### TWR-2 — §9 tag price not snapped to $10k grid → **FALSE POSITIVE vs documented design**
- **Gating check (as the brief prescribed):** does `SetCell`/`ApplyContract` snap internally?
  `SetCell` (ledger_writes.go:35) does NOT snap — the off-grid tag price DOES enter the cell.
- **BUT** that is an INTENTIONAL, TESTED, DOCUMENTED decision: `TestIntegration_TagOffGridPriceSnapsInCap`
  (tag_integration_test.go:265-314) explicitly asserts "the ledger CELL holds the exact cent-precise
  figure (the KING — no rounding at rest), while the franchise's derived CapUsed SNAPS each cell to
  $10k at aggregation" (`helpers.go:44`). The franchise-CapUsed on-grid guarantee the finding worried
  about IS held; every downstream consumer (§8 dead cap, §10 extension) snaps its OWN output.
- **Disposition:** GLM ran blind and did not see the test. Snapping the tag at resolution would
  CONTRADICT and break that documented test = reopening a durable money-model decision → per the
  project's hard constraints, FLAG, don't route around. Fix drafted then **REVERTED**; tree clean.
- **Flag for Christopher (design consistency, NOT a bug):** the tag is inconsistent with the other
  write paths (deadcap/extension/buyout/retirement/relief/sign snap the cell at write; tag doesn't).
  Both are harmless because aggregation re-snaps, but "cell is KING, no rounding at rest" vs
  "snap at write" is a money-model consistency question worth one deliberate ruling. Not this session.

### TWR-1 — unbounded MFL fetch on the Wails OnStartup thread → **FIXED (+ test gate)**
- **Confirmed real on current main:** `startup` (app.go:118) saves the raw OnStartup ctx and calls
  `initStoreFloor(ctx)` (app.go:171) which fires up to 4 MFL fetches on a fresh DB (rulebook
  league+rules, state players+rosters) with NO deadline → a slow/unreachable MFL hangs OnStartup
  indefinitely (the black-screen class the `-probe` diagnostic exists for; = arch anomaly A2's teeth).
- **Fix:** wrap the init ctx in `context.WithTimeout(ctx, storeFloorTimeout=120s)` (app.go). A hung
  fetch now aborts and surfaces via `startupErr`/`Ping` (+ the existing stderr log) within the bound,
  instead of hanging forever. `a.ctx` keeps the unbounded app-lifetime ctx for later IPC. Minimal,
  backend-only; normal launch behaves identically (init completes well under 120s).
- **Gate:** `go build` clean / `make lint` 0 issues / `go test -race ./...` green (41 pkgs).
- **REMAINING (pre-merge):** the live Beelink FAILURE-PATH functional gate — launch with MFL
  unreachable and confirm a bounded, surfaced error instead of a black window. Not run tonight (needs
  the Beelink GUI + a simulated-unreachable MFL). **TWR-1 stays on `session/review-harvest`, unmerged,
  until that gate passes.** Normal-launch path is covered by existing gates.

---

## The rest of the 13 GLM leads (recorded as leads for later — NOT worked this session)
Per the brief, GOAL 1 = the two HIGH leads only. The MEDIUM/LOW leads below are real and triaged as
worth doing, batched, in a dedicated hygiene/deps session — not here.

| Lead | Chunk | Sev | Triage disposition |
|---|---|---|---|
| Go toolchain 1.26.4→1.26.5 (GO-2026-5856 crypto/tls ECH) | config | MED | REAL; toolchain bump is safe. **Note:** GLM's specific call-traces (incl. money.go:110→tls.Conn.Write) are implausible — money.go does no TLS; cite govulncheck, not the invented trace. Do in a deps session. |
| x/sys → v0.44.0+ (GO-2026-5024) | config | LOW | REAL, not-called (Windows-only). Bump with the toolchain. |
| vite ^3→>=6.4.3 (14+ CVEs, dev-server-only) | config | HIGH | REAL but dev-server-only (not in shipped binary); multi-major jump (needs plugin-react ^2→^6, ts ^4→^5). SCOPE its own session — confirm with Christopher first. |
| rulebook.Initialize double-write no tx wrap | db-store | MED | REAL (orphan version row possible on partial failure). Fix = one BeginTx/Commit. Batch. |
| log.Printf→slog ×4 (state.go:328, freeagency.go:99, app.go:128) | multi | MED/LOW | REAL hygiene. Consolidate as one logging sweep. Batch. |
| app.go:213 `_ = pools.Close()` drops DB-close error | wails | LOW | REAL (loses a corruption signal given DB-corruption history). Log before exit. Batch. |
| probe.go:32 goroutine no recover | wails | MED | REAL but probe-only blast radius (every path os.Exit). GLM correctly downgraded. Low value. |
| harness.ts setParam no try/catch | frontend | MED | REAL (unhandled rejection, error slice never set). Mirror the loadRankings pattern. Batch. |
| TransactionsPanel.tsx bridge calls no catch | frontend | MED | REAL **but** GLM flagged "confirm not superseded by the M4 workspace" — TransactionsPanel is the OLD dev panel; confirm live before investing. |
| ping.ts stuck-loading on rejection | frontend | LOW | REAL but the store is DEAD CODE (zero importers). Fix only if wired up. |

## Arch anomalies A1–A5 (Ornith, triaged vs the REAL depguard — all ALLOWED)
A1 rulebook→ingestion, A2 wails→ingestion (=TWR-1's root — fix the timeout, not the import), A3
schooltier→scouting (documented exception), A4 composition/output→store/db (composition reads stores
via port interfaces by design; `engine-is-pure` is scoped to `internal/engine/**` only), A5
output→sqlite (output IS a store, allowed). **0 are enforced-depguard violations.** A4's
engine→store-via-interfaces is the one genuine refactor-lead (not a bug) — a documented carry-forward.

## Gate summary
GOAL-2 machinery: measured A/B (78→21, 0 violations) on the Beelink. GOAL-1: `go build` clean /
`make lint` 0 / `go test -race ./...` green (41 pkgs). TWR-1 live failure-path gate = pre-merge TODO.
