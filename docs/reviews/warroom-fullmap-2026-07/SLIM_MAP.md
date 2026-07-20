# TheWarRoom — Full-Codebase Map & Slim Review
**Date:** 2026-07-20 · **Branch reviewed:** `session/review-harvest` (tip `788e7c3`; main `ee891ef` + TWR-1)
**Method:** two-layer, leads-not-findings. (A) Deterministic ground truth on CT105
(`golangci-lint`, `staticcheck`, `dupl`, `ts-prune`). (B) GLM 5.2 semantic review on Hermes —
the **entire Go backend in one 231k-token context** + a frontend/standards pass. Every GLM lead
was then **triaged by Claude against source** (the greps in each row were run; results below).
**Read-only:** no source was modified by this review.

> **How to read this:** each item is a **LEAD with a disposition**, not a verdict. `CONFIRMED` = I
> re-ran the check against source and it holds. `NEEDS A RULING` = real, but the fix touches a
> documented/durable decision — a human decides. `NOT DEAD / keep` = GLM flagged it, triage cleared
> it. Nothing here has been changed; this is the map for tomorrow's ClaudeBox + enterprise-engineer pass.

---

## 0. Executive summary

**The codebase is already static-analysis pristine** — `golangci-lint` (strict repo config) and
`staticcheck` (incl. U1000 unused) both report **0 issues**; error-wrapping, typed IPC boundary,
single-writer, engine-purity, and the "ledger is KING" money model all verified **clean**. The
mechanical wins are gone. What's left is the judgment layer, and it clusters tightly:

1. **A dead-scaffolding tail on the scouting sub-system** — ~2000+ lines of fetchers + a whole
   orphaned `scouting.Profile` type hierarchy built ahead of a wiring that hasn't happened. Real,
   documented deferrals — but a 20-year engineer will ask "why is this in `main`?" Have the answer ready.
2. **Two adapter-layer asymmetries** — `m2_app.go` carries orchestration logic that its M1 twin
   delegates to a service package; the whole M4 frontend bypasses the store-owns-IPC pattern the
   M1–M3 code follows. Neither is a bug; both are consistency questions worth one ruling each.
3. **One latent frontend bug** — `stage()` in the three M4 components awaits `PreviewTransaction`
   with no `try/catch`; a rejected IPC call leaves the modal stuck "previewing…" forever.
4. **Small, safe deletions** — `ping.ts` (dead), `TransactionsPanel.tsx` (superseded), a dead
   `mflPerfZ` sort key, and a handful of triplicated helpers (`finite`, `boolToInt`, format utils).
5. **Two big refactor leads, high-effort** — the 10 L4 rubrics share one structural `Apply` shape
   (~1200 lines collapsible); `TransactionWorkspace.tsx` is 735 lines (over the 400 cap). Flag for
   the human; do not auto-apply.

**Deterministic vs. semantic, honestly:** the static tools found essentially nothing (1 production
dup). Every lead below came from the GLM semantic pass and survived source triage — which is the
whole point: the value now is in what linters can't see. GLM's precision was high and it
**correctly self-limited** on six standards areas where it found no violation (§ standards, backend).

---

## 1. Codebase map

### Go backend (231k tokens, reviewed whole)
| Package | Responsibility | Notes |
|---|---|---|
| `main` (root `*.go`) | Wails adapter: wires deps, routes IPC, owns pool lifetime | `m2_app.go` carries more logic than the thin-adapter law wants — §6.1 |
| `internal/domain` | Leaf value types (Money int64-cents, Position, Phase, PlayerRecord) | clean leaf |
| `internal/playerid` | Validated MFL player-ID wrapper (RISK-003) | clean leaf |
| `internal/scouting` | Leaf scouting-input types + `SchoolTier` | `Profile`/`OffenseFilm`/`IDPFilm`/`NGSCoverage`/`SafetyRole` **orphaned** — §2, §4 |
| `internal/schema` | Hand-written boundary validation for external/IPC JSON | `RawPlayerRecord.Validate()` unused — §2 |
| `internal/engine` (+ `l4/*`) | Pure scoring pipeline L1→L6 — imports `domain` only | purity **verified clean**; 10 L4 rubrics share one shape — §3, §5 |
| `internal/composition` | Reads stores+harness, assembles engine inputs | boundary pkg |
| `internal/db` | SQLite split read/write pool (single-writer) | **verified clean** |
| `internal/ingestion` (+ `*/`) | Layer-1 MFL/CSV fetchers | several fetchers built-but-unwired — §4 |
| `internal/mfl` | MFL HTTP transport (rate limit, host discovery, 429 backoff) | clean |
| `internal/normalize` | Raw→domain join (rosters × players DB) | clean |
| `internal/store/{params,rulebook,state}` | B4 calibration / B3b rulebook / B3c mutable state+ledger | ledger model **verified clean** |
| `internal/output` | B6 double-immutable per-season engine output | clean |
| `internal/harness` | 13-case validation suite + rookie flow | consumes `PFFAlpha` seam (case 3G pending) |
| `internal/transactions` (+ handlers) | B7a sole runtime mutator; holds the ONLY state Writer | **single-writer verified clean** |
| `internal/rankings` | M1 orchestrator (scores league → B6) | the thin-adapter model M2 should mirror |
| `internal/powerrankings` | M2 blend math (robust-z / MAD / weighted blend) | holds math only; orchestration leaked to `m2_app.go` |

**Dependency spine:** `engine → domain` only (pure); `transactions → store/state (+normalize,domain)`;
`rankings → composition,engine,normalize,output,store/state`; `composition → engine,domain,scouting,store/params`;
`main → everything`. No boundary leak reaches the engine (depguard **verified clean**).

### Frontend (React/TS)
`store/harness.ts` is the sole IPC gateway for M1/M2/M3/admin → boards. The **M4 transaction layer**
(`TransactionWorkspace`, `TradeBuilder`, `LeagueControls`, + legacy `TransactionsPanel`) calls IPC
**directly, bypassing the store** — §6.2. `store/ping.ts` is dead (§2).

---

## 2. Dead code (triaged)

| # | Location | Lead | Disposition | Conf |
|---|---|---|---|---|
| D1 | `frontend/src/store/ping.ts` (whole file) | `usePingStore` — zero importers | **CONFIRMED DEAD** — grep returns nothing. Safe delete (~20 lines). | hi |
| D2 | `internal/ingestion/extcsv.go:144` | `StreamCSVGz` — no caller | **CONFIRMED DEAD** — only def+comment; gz-stream consumer never arrived. | hi |
| D3 | `internal/scouting` — `Profile`/`OffenseFilm`/`IDPFilm`/`NGSCoverage` | ~70 lines of types, no producer/consumer | **CONFIRMED ORPHAN** — only *comments* in 3 fetchers reference them, no code. See §4/NEEDS RULING. | hi |
| D4 | `internal/scouting/constants.go` — `SafetyRole` + 4 consts | never read/set | **CONFIRMED DEAD** — only a comment in `s.go` (SL-OQ-035/036 kept "schema-only"). | hi |
| D5 | `frontend/.../PowerRankingsBoard.tsx:21,234` | `'mflPerfZ'` sort key in type+switch, never a column | **CONFIRMED DEAD** — not in JSX. 3-line removal. | hi |
| D6 | `internal/schema/schema.go:65` | `RawPlayerRecord.Validate()` never called | **CONFIRMED UNUSED** — `DecodePlayerRecord` (cases_eval.go:368) reads `rec.ID` directly. It's a *public-seam pattern* member; delete or wire, but not load-bearing. | med |
| D7 | `internal/store/state/writes.go:85-95` | standalone `Store.MovePlayer/SetRosterStatus/ApplyContract` | **CONFIRMED test/admin-only** — production mutates via the `TxWriter` interface (`w.MovePlayer`) through the Coordinator; these `*Store` wrappers have no prod caller. GLM caught the subtle method-set distinction correctly. | med |
| D8 | `internal/store/state/ledger.go:58` | `LedgerCells` — no prod caller | **CONFIRMED test/audit-only** (comment says so). Move to a test helper or keep as documented audit seam. | low |
| D9 | `internal/ingestion/cfbd.go:39` | `NewCFBDClient` — no caller | **CONFIRMED unwired** — CFBD fetchers take `*http.Client` but nothing calls the factory yet. | med |
| — | `internal/engine/l4/defense/dt.go:146` — `DT.PFFAlpha()` | GLM flagged as maybe-dead (med) | **NOT DEAD — keep.** Called by `dt_test.go`; it's the documented **case-3G introspection seam**, 3G wiring deliberately deferred (matches project record). GLM correctly hedged. | hi |
| — | `wailsjs/go/models.ts` entries | ts-prune "unused" | **FALSE POSITIVE** — generated Wails bindings. | hi |

## 3. Duplicate / near-duplicate (triaged)

| # | Location | Lead | Disposition | Conf |
|---|---|---|---|---|
| DUP1 | `state/cap_relief.go:109` ≡ `state/state.go:372` | `loadCapRelief`/`loadDeadCap` identical query/scan/sum | **CONFIRMED** (only production dup `dupl` found). Extract `sumLedgerByFranchise(ctx, table, col)`. ~24 lines, lo risk (table/col are constants). | hi |
| DUP2 | `engine/finite.go` · `composition/playerspec.go:181` · `output/helpers.go:158` | `finite()` triplicated | **CONFIRMED** — 3 identical bodies. Move to a leaf pkg (M17 "extract on 2nd use"). | hi |
| DUP3 | `output/helpers.go:185` · `params/helpers.go:22` · `state/helpers.go:181` | `boolToInt()` triplicated | **CONFIRMED** — 3 identical bodies. Same fix. | hi |
| DUP4 | `TradeBuilder.tsx` + `TransactionWorkspace.tsx` | `money`/`initials`/`Th`/`Empty` copied verbatim | **CONFIRMED** (lines 20-23/404/416 vs 22-23/721/733). Extract `transactions/format.ts` + shared components. | hi |
| DUP5 | 3× M4 components | stage→preview→confirm→cancel lifecycle + `stageGen` token triplicated | **CONFIRMED structural** — the biggest FE dup; extract `useTransactionStaging(refreshFn)`. Note: ties to bug B1 below. | hi |
| DUP6 | `TransactionsPanel.tsx:67` + `LeagueControls.tsx:43` | `refreshPhase` identical | **CONFIRMED**. Resolves for free if TransactionsPanel is deleted (§5). | hi |
| DUP7 | `m2_app.go` `parseStanding` vs `leaguestandings.Validate()` | same numeric parse in two layers | **CONFIRMED but architecturally consistent** (fetcher "transforms nothing"). Redundant, not wrong — fold into the M2 service if §6.1 is actioned. | med |
| — | `*_test.go` table blocks (`dupl`) | 200+ dup lines | **ACCEPTABLE** — idiomatic per-position/per-case test scaffolds. Not a lead. | hi |

## 4. Stubs without a destination (NEEDS A RULING)

These are **real, documented deferrals** — not accidental orphans — but they ship unwired code in
`main`. The enterprise reviewer *will* ask about them; the ruling is "next sprint" vs "carve out."

- **`scouting.Profile` type hierarchy + ~11 unwired scouting fetchers** (`agetrajectory`,
  `collegeshare`, `collegedefense`, `crosswalk`, `kicking`, `madden`, `nflproduction`, `pfrcoverage`,
  `ras`, `touchshare`, `veteranfilm`) — complete `Fetch()` + tests, **no production caller**; the M1
  orchestrator states scouting sub-signals are "Data-Parity ABSENT … no fetcher wired yet." ~2000+ lines.
  **Ruling needed:** are these the next data-integration sprint, or should the orphaned `Profile`
  types (D3) be deleted now and the fetchers moved behind a build tag / `experimental/` until wired?
- **`internal/ingestion/salaryadjustments/`, `internal/ingestion/schedule/`** — clone the rosters
  template, no consumer (cap adjustments flow via `dead_cap_ledger`, not MFL ingest). Dead templates or future sync layer?
- **`NewCFBDClient`** (D9) — prematurely extracted; wire it when the CFBD orchestration lands, or drop it.

## 5. Slimming opportunities (ranked; effort × risk)

| Rank | Move | ~Lines | Risk | Note |
|---|---|---|---|---|
| 1 | Delete `TransactionsPanel.tsx` + flip `SHOW_DEV_PANEL=false` | ~619 | near-zero | **Verify first:** all 14 ops confirmed ported to M4 (7+1+6). Kills the biggest FE file *and* a standards violation. It IS still imported by App.tsx behind the flag — this is the ruling. |
| 2 | **L4 rubric data-driven consolidation** | ~1200 | lo-mech/**hi-arch** | 10 rubrics share one `Apply` shape (data + 2 modulator hooks). Biggest single slim, but it's an architectural change to the scoring core — **flag for the human, don't auto-apply.** |
| 3 | Extract `useTransactionStaging()` hook | ~120 | med | Collapses DUP5 *and* fixes bug B1 in one move. |
| 4 | Delete `store/ping.ts` (D1) | ~20 | zero | pure deletion |
| 5 | `finite`/`boolToInt`/format-util consolidation (DUP2/3/4) | ~50 | lo | mechanical M17 fixes |
| 6 | Remove dead `mflPerfZ` (D5) | ~3 | zero | pure deletion |

## 6. Standards adherence (triaged)

**Verified CLEAN (GLM checked and correctly found no violation):** error-wrapping (`%w` everywhere),
typed IPC boundary (no `any`/`interface{}` across the bridge), single-writer law, engine-is-pure
depguard, ledger-is-KING money model. These are the load-bearing invariants and they hold.

**6.1 — `m2_app.go` holds orchestration logic (NEEDS A RULING, hi conf).** 12 functions incl.
`parseStanding`, `buildBlendInputs`, `aggregateScouting`, `buildPowerRows`, `resolveAggMode`,
`clampWeight`. Its M1 twin `m1_app.go` delegates to `internal/rankings`; `internal/powerrankings`
holds only blend math. So M2's orchestration leaked into the adapter while M1's didn't. **Not a bug**
(it passed a 3-stage review at merge) — a consistency question: extract an M2 service to mirror M1, or
accept the asymmetry. One ruling.

**6.2 — whole M4 frontend bypasses the store-owns-IPC pattern (NEEDS A RULING, hi conf).** The
harness store centralizes M1–M3 IPC; all four M4 transaction components import Wails IPC functions
directly. Plausibly intentional (they hold ephemeral form/cart/modal state) — but a shared
`transactions.ts` store for the read-model (franchises, phase, legalOps) would remove duplication.
Rule whether the store pattern applies to M4.

**6.3 — `stage()` missing try/catch → latent bug (CONFIRMED, hi conf).** ⚠️ In
`TransactionWorkspace.tsx` (and `TradeBuilder`, `LeagueControls`), `stage()` awaits
`PreviewTransaction(req)` with **no error handler** (the `try` first appears at line 251 in
`confirm()`, never in `stage()`). A rejected IPC call leaves the modal stuck on
"Checking the move with the engine…" forever. **This is the one genuine bug in the review.** Fixed
naturally by the #3 hook extraction, or add a `catch` that clears `previewing`.

**6.4 — file-size cap (CONFIRMED, med conf).** `TransactionWorkspace.tsx` = **735 lines** (over the
AGENTS.md 400 hard cap; GLM under-estimated at ~500). `TransactionsPanel.tsx` 619 (slated for deletion),
`TradeBuilder.tsx` 418. Note the cap may only be *enforced* on `.go` via `make filelen`; if it applies
to `.tsx`, these violate. The #3 hook + #1 deletion bring the survivors toward target.

**6.5 — `powerReqSeq` module-level mutable state (low conf).** `harness.ts:10 let powerReqSeq = 0` —
Go bans package globals (`gochecknoglobals`); TS has no equivalent linter. Functionally correct
request-sequencing; could be a closure/store field. Minor.

---

## 7. What was NOT changed, and what's next
Nothing in the source was touched — this is a read-only map. **Recommended order for the ClaudeBox
pass tomorrow:** (1) the safe deletions D1/D5 + DUP2/3/4 (mechanical, zero-to-lo risk); (2) fix bug
6.3; (3) get Christopher's ruling on the four NEEDS-A-RULING items (scouting scaffolding §4, m2_app
§6.1, M4 store pattern §6.2, TransactionsPanel deletion §5.1) *before* the enterprise review, so the
answers are ready; (4) treat the L4 consolidation (§5.2) as an enterprise-review discussion item, not
a pre-emptive change. Raw GLM outputs: `backend_review.md`, `frontend_review.md`. Tool ground truth:
`deterministic/SUMMARY.md`.
