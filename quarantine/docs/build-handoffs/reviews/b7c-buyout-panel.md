# §12 Buyout + Season-Phase — Expert Panel Responses

**Panel:** GLM 5.2 · Gemini · DeepSeek (parallel, blind). Brief: `docs/transactions/Season_Phase_Design.md` + the paste.md brief 2026-07-10.
**Status:** collecting. Triage happens once all three are in.

---

## Gemini (received 2026-07-10)

### Standout leads (look real vs source — pending cross-triage)
- **Season-tick ambiguity (Q1, "definite defect").** `remaining = expiration − season` behavior
  depends entirely on WHEN the global `season` int increments. If it ticks up at playoffs-end
  (before offseason moves), a genuine 2-years-left contract computes `remaining = 1` → auto-rejected.
  If it ticks at end-of-offseason, math works — but that's an undocumented implicit assumption. **The
  offseason/season-increment lifecycle boundary is unspecified and the buyout math hangs on it.**
- **Phase rollback exploit (Q6, "definite defect").** `ADVANCE_PHASE` allowing any target to support
  rollback means a commissioner rolling REGULAR_SEASON → OFFSEASON re-opens the buyout gate; any team
  under its per-season cap of 2 can now buy out mid-timeline. Rollback-as-append-row can't be a naive
  toggle without handling interleaved mutations.
- **Player↔buyout link scrubbed (Q6, "definite defect").** Releasing via the generic §8 path (delete
  roster rows + generic dead-cap entry) erases the specific-player↔buyout link needed later to enforce
  "no re-bid until next offseason." Fix: ensure the dead-cap entry carries player_id, or add a small
  `bought_out_players` log now. (Note: our DeadCapEntry already has MFLID — verify this covers it.)
- **Final-charge rounding unspecified (Q1).** avgRemaining is round-half-up, but the design is silent
  on rounding `avgRemaining × 75%/90%` (yields sub-cent fractions). Must pin a direction for int64.

### Judgment calls
- Q2 phase gate seam: praised — first-step-inside-WriteTx + single-writer mutex = no race; ordering
  before transaction_counts correct; fail-closed on empty table is safe.
- Q3 minimal enum: flags "hidden structural debt" — trades/waivers governed by in-handler week-checks
  while buyout is governed by the phase gate = split temporal architecture; unifying later may need a
  season_phases log migration.
- Q5 immutability: recommends adding the DB-level UPDATE/DELETE triggers for consistency with the
  dead-cap ledger; app-level append-only is functional but a raw SQL tool could mutate phase history.

### Confirmed correct
- Q4 edge handling: fail-loud reject for <2 or >4 remaining is "the only correct path"; §10 makes 5/6
  guaranteed to occur; route to §13 commissioner. Do NOT fabricate rates.

---

## DeepSeek (received 2026-07-10)

### Must-fix before coding
- **6a — season-number lifecycle UNDEFINED (definite defect by omission).** SAME catch as Gemini,
  independent. When does the `season` int advance relative to phase? Affects (i) which season a
  buyout counts against in `transaction_counts`, (ii) which season the dead-cap charge lands in.
  Two candidate policies: (a) increment on first OFFSEASON after PLAYOFFS → buyouts count next season;
  (b) increment at REGULAR_SEASON start → offseason buyouts count prior season. Must pick before code.
- **1b/6c/6d — §8 reuse boundary (must verify).** Reuse the §8 *dead-cap write* portion ONLY, not any
  waiver/claim/bidding mechanics — else a bought-out player enters waivers and can be re-bid immediately,
  violating "no re-bid until next offseason." Also verify §8 lands dead cap WHOLE in one year (no
  multi-year spread) or the "lands whole in current season" reuse is wrong.

### Judgment calls / recommendations
- **6b — round-half-up trap.** Go's `math.Round` is banker's rounding, not half-up; `math.Floor(x+0.5)`
  for non-negative cents. Confirm our `domain` rounding helper does half-up. (Applies to BOTH avgRemaining
  AND the ×rate step — the final-charge rounding Gemini also flagged.)
- **1a — 90% rate reachability.** A standard 4-yr contract bought in year 1 has remaining=3 → 75%, never
  90%; 90% (remaining=4) is only reachable on a §10-extended contract. Probably intended — DOCUMENT it so
  a reader doesn't assume a new 4-yr deal uses 90%.
- **3 — minimal enum: add a `meta` JSON column** to the transitions table now (freeform, e.g.
  `{"week":9,"deadline":"trade"}`) so finer phases later carry granularity without a schema change.
- **2 — rollback openness:** fine for v1 + commissioner confirm, but consider a flag distinguishing
  a correction from forward progress. (Gemini escalated this to a defect via the buyout-gate re-open;
  DeepSeek treats it as a posture to document — triage the delta.)
- **5 — DB triggers:** add UPDATE/DELETE-blocking triggers at table creation (zero cost, consistent
  with dead-cap ledger). Same as Gemini.
- **6e — ADVANCE_PHASE UX guardrails** (confirm dialog, mandatory note, unusual-path warning).
- **6f — test matrix:** phase-reject, rolled-back tx leaves no partial count bump, no-op transition
  rejected, stale-phase read under serialize.

---

# CODE REVIEW (post-build, on the diff) — GLM 5.2, 2026-07-11

**Model confirmed `build · glm-5.2`** (a first pass accidentally ran on a free deepseek default —
discarded; re-run forced onto `zai/glm-5.2`). **Verdict: SHIP. 0 MAJOR / 0 MINOR / 6 NOTE.**
GLM independently recomputed the §12 math, traced the phase-gate atomicity, the counter
read-check-bump, and the append-only log — all correct against the stated rules.

**NOTEs triaged (all test-hardening / benign):**
- N1 rejected buyout burns no slot but untested → **ADOPT** (integration test added).
- N3 no rollback proof on the buyout's final step (IncOpCount) — restructure has it → **ADOPT**
  (fake apply-order + counter-fail-rolls-back tests added, parity with restructure).
- N2 averaging path not exercised e2e with UNEQUAL salaries → covered by the `buyoutCharge`/
  `remainingAfter` unit tests (4/6/8M); the store path is exercised (equal cells). Low risk —
  **carry-forward** (an unequal-cell e2e needs an extension-first fixture).
- N4 TAG/EXTENSION legal in all phases → **intentional** (locked GQ2 no-regression); rulebook
  cross-check for offseason-only §9/§10 windows is a **carry-forward** for when finer phases land.
- N5 genesis `from_phase=""` sentinel → benign, never read back. No action.
- N6 seed probe+insert two-pool / no UNIQUE(league,season) → season_phases is a MULTI-ROW append
  log (a UNIQUE would be WRONG); concurrent multi-process Initialize is not a real risk (single
  process, wmu). **Refuted** — no action.

Full review: scratchpad `glm52-buyout-out.txt`.

---

## GLM 5.2 (received 2026-07-10)

### Must-fix before coding
- **Season-int lifecycle vs phase (definite defect, Q1 + Q6).** THIRD independent hit. On
  PLAYOFFS→OFFSEASON does the `season` int increment or stay? If it stays at N, a buyout charges
  dead cap to a CLOSED season N (useless — a buyout exists to clear cap for the UPCOMING season) and
  consumes N's buyout count instead of N+1's. "The most critical unresolved data-model risk."
- **Fresh-DB seed = split source of truth (definite defect, Q2).** "Seeds OFFSEASON" is ambiguous:
  does it INSERT a synthetic seed row `(NULL→OFFSEASON,'seed')` into the append-only table, or does
  `CurrentPhase()` carry a hardcoded fallback for the empty table? The latter = two sources of truth
  for the initial phase. Must be the inserted row; document it.

### Judgment calls / confirmed
- Q2 seam: first-step-inside-WriteTx = correct (fail-fast before the count read); single-writer mutex
  → the tx-scoped phase read is atomic, no race. Confirmed.
- Q3 minimal enum: sound; inventing fine phases guesses semantics the rulebook doesn't have; append-only
  + static map = new phase is one constant + map row, no migration. Confirmed.
- Q4 edge reject: "the only faithful implementation of a rulebook with gaps." Confirmed.
- Q5 triggers: escalates to a **defect of design-consistency** — the design's OWN precedent (dead-cap
  ledger triggers) makes the omission inconsistent. Add them.
- Q6 rollback: allowing backward phase jumps does NOT auto-reverse `transaction_counts` — a valid v1
  manual-correction posture, but DOCUMENT it. (Aligns with DeepSeek; tempers Gemini's "exploit.")
- Q6 voided cells: clarify "voided" = zero-cap-contribution-preserving-history (not row delete) —
  ALREADY TRUE in source (`VoidCells` preserves history, drops cap contribution to 0). Closed.

---

## FINAL CONSOLIDATED TRIAGE (all 3 in, triaged vs source)

**ADOPT — must-fix before code:**
1. **Season-int ↔ phase lifecycle.** UNANIMOUS (all three, blind). Resolution below.
2. **Fresh-DB seed inserts a real transition row** `(from=NULL, to=OFFSEASON, note='seed')`; `CurrentPhase()`
   has NO fallback — the table is the sole source of truth. (GLM)

**ADOPT — fold into the build (cheap, right):**
3. Final buyout charge takes ONE `domain.RoundToNearest10k` snap (flat-$10k doctrine), same as §8 — state it. (all)
4. DB-level UPDATE/DELETE triggers on `season_phases` (parity with dead-cap ledger). (all three)
5. Document the 90%-rate reachability (only via §10-extended contracts; a normal 4-yr deal tops out at 75%). (DeepSeek/GLM)
6. Document that phase rollback does NOT auto-reverse `transaction_counts` — commissioner-trusted posture. (GLM/DeepSeek)
7. Phase-gate test matrix: phase-reject, no partial count bump on rollback, no-op transition rejected, default-deny. (DeepSeek)

**ADOPT — lightweight future-proof:**
8. Add a `meta` JSON/TEXT column to `season_phases` for later finer-phase granularity without a schema change. (DeepSeek)

**REFUTED / CLOSED vs source (do NOT act):**
- §8 "spreads dead cap over years" → §8 `Charge` lands WHOLE, no spread. (verified deadcap.go)
- §8 reuse inherits waiver/bidding → `Waive` has no claim/bidding path in v1; reuse is the write path only. (verified)
- Banker's-rounding trap → house `RoundToNearest10k` is half-up-away-from-zero, not `math.Round`. (verified money.go)
- Player↔buyout link scrubbed → `DeadCapEntry` carries `MFLID`+`Reason`; link preserved in `dead_cap_ledger`. (verified)
- "voided = delete rows" → `VoidCells` zeroes cap contribution AND preserves history. (verified)

**DISAGREEMENT resolved:** phase-rollback — NOT a hard exploit (per-season ceiling of 2 bounds it; commissioner
trusted; counts don't auto-reverse). Downgraded to item 6 (document the posture).

---

## The season-lifecycle resolution (recommended, pending Christopher)
**Invariant: the loaded `season` int is the season the OFFSEASON phase BELONGS TO — offseason sits at the
START of its season's lifecycle, not the end.** Cycle: `OFFSEASON(N) → REGULAR_SEASON(N) → PLAYOFFS(N) →
[rollover to N+1] → OFFSEASON(N+1) …`. Consequences (all correct): a buyout in OFFSEASON(N) counts against
N's `transaction_counts`, charges dead cap to N (the upcoming managed season it clears cap for), and computes
`remaining = expiration − N`. **No season-rollover machinery is built in this session** — the PLAYOFFS→OFFSEASON
increment is the existing season-rollover carry-forward (already noted for §11's in-season restructure unlock).
v1 correctness holds as long as we are in OFFSEASON(N) with the int already at N — which the fresh-DB seed
(OFFSEASON, loaded season) satisfies.

---

## Cross-panel convergence (2 of 3 in)
- **UNANIMOUS must-fix:** season-int ↔ phase lifecycle (Gemini Q1 + DeepSeek 6a), arrived at
  independently. Highest-priority hole.
- **BOTH flag:** final-charge rounding direction; DB immutability triggers for parity; player↔buyout
  link / §8-reuse boundary for the deferred re-bid ban.
- **DISAGREEMENT to resolve:** phase-rollback — Gemini = exploitable defect; DeepSeek = document-as-posture.
