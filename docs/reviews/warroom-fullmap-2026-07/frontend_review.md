# Frontend + Cross-Cutting Standards Review — TheWarRoom

## 1. FRONTEND MAP

### Data-flow spine
```
Wails IPC (wailsjs/go/main/App.ts, wailsjs/go/models.ts)
  │
  ├── harness store (src/store/harness.ts) ── owns M1/M2/M3/admin IPC calls
  │     ├── ScoreRookies ──────────────► rookies ───► RookieTable
  │     ├── RunValidationSuite ────────► validation ─► ValidationBoard
  │     ├── GetParams / SetParam ──────► params ─────► AdminPanel
  │     ├── GetRankings ───────────────► rankings ───► RankingsBoard
  │     ├── ScoreLeague ───────────────► scoreReport ► RankingsBoard
  │     └── GetPowerRankings ──────────► powerRankings ► PowerRankingsBoard
  │
  └── Direct IPC (components bypass the store)
        ├── TransactionWorkspace ──► GetFranchises, GetRoster, GetFreeAgentPool,
        │                            GetCurrentPhase, GetLegalOps,
        │                            PreviewTransaction, ExecuteTransaction
        ├── TradeBuilder ──────────► GetFranchises, GetRoster, GetLegalOps,
        │                            PreviewTransaction, ExecuteTransaction
        ├── LeagueControls ────────► GetCurrentPhase, GetFranchises,
        │                            PreviewTransaction, ExecuteTransaction
        └── TransactionsPanel(dev) ► ExecuteTransaction, GetFranchiseState,
                                     GetCurrentPhase, GetFreeAgents
```

### Module responsibilities (one line each)

| Module | Responsibility |
|---|---|
| `App.tsx` | Tab shell; mounts all boards; calls `loadAll` on mount; `SHOW_DEV_PANEL` flag gates legacy tab |
| `store/harness.ts` | Zustand store; sole IPC gateway for M1/M2/M3/admin; request-sequencing for power rankings |
| `store/ping.ts` | B0 health-check store (Ping IPC); **zero importers** |
| `components/AdminPanel.tsx` | Live calibration form; reads `params`, calls `setParam` per-key |
| `components/RankingsBoard.tsx` | M1: persisted 32-team board; rank/cap-efficiency views; score-league trigger + report |
| `components/PowerRankingsBoard.tsx` | M2: live blended power board; scouting-weight slider; sortable MFL columns |
| `components/RookieTable.tsx` | Sandbox: scored rookie board with every L4 intermediate visible |
| `components/ValidationBoard.tsx` | Module 3: 12 architectural test cases with PASS/FAIL/PENDING states |
| `components/TransactionsPanel.tsx` | B7a dev surface: raw 14-op executor (legacy, behind `SHOW_DEV_PANEL`) |
| `components/transactions/TransactionWorkspace.tsx` | M4 real operator UI: franchise rail → roster → player action panel; 7 player ops |
| `components/transactions/TradeBuilder.tsx` | M4 multi-leg trade surface: browse rosters → cart → atomic N-leg swap |
| `components/transactions/LeagueControls.tsx` | Commissioner surface: calendar ops + destructive powers (6 ops) |
| `components/transactions/ConfirmModal.tsx` | Shared staged-confirm modal (plain/quote/rejected states) |

---

## 2. DEAD CODE

### `src/store/ping.ts:14 — usePingStore has zero importers` — **CONFIRMED DEAD**
- **Confidence: hi**
- The export is flagged by `ts-prune` and no file in the provided source imports `usePingStore` or `./store/ping`. The B0 health-check surface was superseded by the harness store's `loadAll`.
- **Confirm:** `grep -r "usePingStore\|store/ping" frontend/src/` — expect zero hits outside `ping.ts` itself. The entire file (~20 lines) is deletable.

### `src/App.tsx:23 — SHOW_DEV_PANEL = true; the dev tab and TransactionsPanel may be fully superseded` — **CONFIRMED DEAD (stale flag)**
- **Confidence: hi**
- The comment says the dev panel stays "until all 14 ops port into the workspace — then it is deleted." Counting the ported ops:
  - TransactionWorkspace: `ROSTER_STATUS`, `WAIVER`, `SIGN`, `TAG`, `EXTENSION`, `RESTRUCTURE`, `BUYOUT` (7)
  - TradeBuilder: `TRADE` (1)
  - LeagueControls: `ADVANCE_PHASE`, `ROLLOVER_SEASON`, `SET_SIGNING_WINDOW`, `RETIREMENT`, `DEATH`, `CAP_RELIEF` (6)
  - **Total: 14** — every op the dev panel exposes.
- **Confirm:** diff the `Kind` union in `TransactionsPanel.tsx:25-40` against the `Pending['kind']` union in `ConfirmModal.tsx:14-28` + the ops wired in the three M4 components. They should be identical sets.

### `src/components/PowerRankingsBoard.tsx:14 — 'mflPerfZ' in SortKey type is never a clickable sort column` — **CONFIRMED DEAD**
- **Confidence: med**
- `SortKey` includes `'mflPerfZ'`; `getSortVal` (line ~176) handles it; but no `<SortTh k="mflPerfZ" ...>` exists in the rendered `<thead>`. The type member and its `case` are unreachable from the UI.
- **Confirm:** `grep "mflPerfZ" frontend/src/components/PowerRankingsBoard.tsx` — appears in type + `getSortVal` switch only, never in JSX.

### `wailsjs/go/models.ts` entries — **FALSE POSITIVES (generated bindings)**
- **Confidence: hi** — per the brief, these are Wails-generated; consumed at runtime via `main.*` type references across the codebase.

---

## 3. DUPLICATE / NEAR-DUPLICATE

### `src/components/transactions/TradeBuilder.tsx:13-14 + src/components/transactions/TransactionWorkspace.tsx:13-14 — `initials` and `money` helpers duplicated verbatim`
- **Confidence: hi**
- Both files define:
  ```ts
  const money = (m: number) => `$${m.toFixed(1)}M`;
  const initials = (name: string) => name.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase();
  ```
- **Confirm:** side-by-side diff of the two files' top-level `const` block. Extract to a shared `transactions/format.ts`.

### `src/components/transactions/TradeBuilder.tsx:330-336 + src/components/transactions/TransactionWorkspace.tsx:~470-478 — `Empty` component duplicated`
- **Confidence: hi**
- Identical `<div className="m-auto max-w-[260px] p-8 text-center text-[13px] text-[#64748b]">{text}</div>`.
- **Confirm:** `grep "function Empty" frontend/src/components/transactions/` — two definitions, same body.

### `src/components/transactions/TradeBuilder.tsx:315-326 + src/components/transactions/TransactionWorkspace.tsx:~460-471 — `Th` component duplicated`
- **Confidence: hi**
- Identical sticky-header `<th>` with the same className string and `{children, right}` props.
- **Confirm:** `grep "function Th" frontend/src/components/transactions/` — two definitions.

### `src/components/transactions/TransactionWorkspace.tsx + TradeBuilder.tsx + LeagueControls.tsx — the stage→preview→confirm→cancel lifecycle is triplicated`
- **Confidence: hi**
- All three components independently implement:
  - `const stageGen = useRef(0)` + the generation-token discard pattern
  - `stage()` / `stageAdvancePhase()` / etc. → `PreviewTransaction(req)` → fold result into `Pending`
  - `confirm()` → `ExecuteTransaction(pending.request)` → handle `!res.ok` → `setPending(null)` → refresh
  - `cancel()` → `stageGen.current++; setPending(null)`
- The bodies are structurally identical; the only divergence is the `refreshX()` call after commit.
- **Confirm:** diff the three `confirm()` functions side-by-side; the only line that differs is the post-success refresh call. Extract a `useTransactionStaging(refreshFn)` hook.

### `src/components/TransactionsPanel.tsx:~75-78 + src/components/transactions/LeagueControls.tsx:~37-40 — `refreshPhase` duplicated`
- **Confidence: hi**
- Both define the exact same 3-line function:
  ```ts
  async function refreshPhase() {
    const r = await GetCurrentPhase();
    setPhase(r.ok ? r.phase : `? (${r.detail})`);
  }
  ```
- **Confirm:** `grep "refreshPhase" frontend/src/components/` — two files, same body.

### `src/components/transactions/TradeBuilder.tsx:~89-91 + src/components/transactions/LeagueControls.tsx:~54-56 — franchise-name lookup (`frName`/`frLabel`) duplicated`
- **Confidence: med**
- Both look up `franchises.find(x => x.franchiseID === id)` and format slightly differently (`frLabel` includes the ID; `frName` does not). Same data, different formatting — a shared `useFranchiseLookup()` hook would unify.
- **Confirm:** diff the two functions; note the `frLabel` includes the ID prefix.

### `src/store/harness.ts:63-71, 80-88, 90-104, 106-120 — four IPC methods repeat the same try/catch/set-error skeleton`
- **Confidence: med**
- `loadAll`, `loadRankings`, `loadPowerRankings`, `scoreLeague` all follow: `set({loading, error: ''})` → `try { const x = await Ipc(); set({x, error: x.ok ? '' : x.error}) } catch (e) { set({error: String(e)}) }`.
- **Confirm:** align the four method bodies; the skeleton is identical with only the IPC call and state key changing.

---

## 4. STUBS WITHOUT A DESTINATION

### `src/components/TransactionsPanel.tsx:1-280 (entire file) — legacy B7a dev panel, all 14 ops now ported to M4 surfaces`
- **Confidence: hi**
- The file's own header comment says it is "the B7a dev surface" and that "a full transaction UI is B7b." B7b/M4 has landed: `TransactionWorkspace` (7 ops) + `TradeBuilder` (1 op) + `LeagueControls` (6 ops) = all 14. The `SHOW_DEV_PANEL` flag at `App.tsx:23` is still `true`, shipping dead scaffolding.
- **Confirm:** verify the `Kind` union in `TransactionsPanel.tsx:25-40` is a subset of the ops wired across the three M4 components.

### `src/store/ping.ts:1-20 (entire file) — B0 health-check store with zero consumers`
- **Confidence: hi**
- Never imported. `App.tsx` calls `loadAll` from the harness store for its startup data load. `Ping()` / `usePingStore` have no UI surface.
- **Confirm:** `grep -r "ping" frontend/src/ --include="*.ts" --include="*.tsx" | grep -v node_modules` — the only hit should be `ping.ts` itself.

### `src/App.tsx:23 + App.tsx:60-62 — SHOW_DEV_PANEL conditional + the 'transactions' Tab branch` — dead once the flag flips
- **Confidence: hi**
- The `SHOW_DEV_PANEL` constant, the `{SHOW_DEV_PANEL && (...)}` conditional, and the `tab === 'transactions'` branch in the render switch are all gated on a flag whose exit condition (all 14 ops ported) is met.
- **Confirm:** verify the op coverage as above; if complete, the flag, the conditional, and the branch are all deletable.

---

## 5. SLIMMING (ranked by lines saved vs. risk)

### 1. Delete `TransactionsPanel.tsx` + flip `SHOW_DEV_PANEL = false` — **~280 lines saved, risk: near-zero**
- **Confidence: hi**
- The component is behind a dev flag, all ops are ported, and it calls IPC directly (violating the store-owns-IPC rule). Removing it also eliminates the standards violation for free.
- **Confirm:** verify no other component imports `TransactionsPanel`.

### 2. Extract `useTransactionStaging()` hook — **~120 lines saved, risk: med**
- **Confidence: hi**
- The stage→preview→confirm→cancel lifecycle + `stageGen` token is triplicated. A hook that takes `(refreshFn: () => Promise<void>)` and returns `{pending, busy, stage, confirm, cancel}` would collapse ~40 lines per consumer.
- **Confirm:** align the three `confirm()` functions; the divergence is a single `refreshX()` line, easily parameterized.

### 3. Delete `store/ping.ts` — **~20 lines saved, risk: zero**
- **Confidence: hi**
- Zero importers; B0 reference superseded.
- **Confirm:** `grep -r "usePingStore" frontend/src/` returns nothing.

### 4. Extract shared `transactions/format.ts` (`initials`, `money`, `Empty`, `Th`) — **~25 lines saved, risk: trivial**
- **Confidence: hi**
- Four utilities duplicated across `TradeBuilder.tsx` and `TransactionWorkspace.tsx`.
- **Confirm:** diff the four functions across the two files.

### 5. Remove dead `'mflPerfZ'` from `SortKey` + its `getSortVal` case — **~3 lines saved, risk: zero**
- **Confidence: med**
- The type member exists and is handled but is never wired to a clickable column header.
- **Confirm:** `grep "mflPerfZ" PowerRankingsBoard.tsx` — type + switch only, no JSX.

---

## 6. STANDARDS ADHERENCE

### Rule: "The IPC call lives in the store, never the component" (SYSTEM_MAP.md → `frontend/src/store/`)

**`src/components/transactions/TransactionWorkspace.tsx:4-12 — imports 7 IPC functions directly; bypasses the store entirely`**
- **Confidence: hi**
- ```ts
  import { GetFranchises, GetRoster, GetFreeAgentPool, GetCurrentPhase,
    GetLegalOps, PreviewTransaction, ExecuteTransaction } from '../../../wailsjs/go/main/App';
  ```
- Every IPC call happens inside the component, not a Zustand store.
- **Confirm:** check whether the SYSTEM_MAP rule is intended to apply to the M4 transaction components or just the harness/M1-M3 store.

**`src/components/transactions/TradeBuilder.tsx:4-9 — imports 5 IPC functions directly`**
- **Confidence: hi** — same violation pattern as TransactionWorkspace.

**`src/components/transactions/LeagueControls.tsx:3-7 — imports 4 IPC functions directly`**
- **Confidence: hi** — same violation pattern.

**`src/components/TransactionsPanel.tsx:3-7 — imports 4 IPC functions directly`**
- **Confidence: hi** — same violation, though the component is slated for deletion.

> **Synthesis:** The harness store correctly centralizes M1/M2/M3 IPC. The entire M4 transaction layer (4 components) was built outside the store pattern. This is either an intentional architectural divergence (the transaction components manage local-only ephemeral state — cart, form inputs, modal staging) or an oversight. If intentional, a `transactions.ts` store for the shared read-model (franchises, phase, legalOps) would still reduce duplication without forcing form state into the store.

### Rule: "Target file size under 250 lines. Hard cap 400" (AGENTS.md)

| File | Approx. lines | Over target? | Over cap? |
|---|---|---|---|
| `TransactionsPanel.tsx` | ~280 | yes | no |
| `TransactionWorkspace.tsx` | ~500 | yes | **yes** |
| `TradeBuilder.tsx` | ~370 | yes | no |
| `LeagueControls.tsx` | ~300 | yes | no |
| `PowerRankingsBoard.tsx` | ~260 | yes | no |

- **Confidence: med** (line counts estimated from source; `make filelen` may only check `.go` files)
- **Confirm:** `wc -l frontend/src/components/transactions/TransactionWorkspace.tsx` — if >400 and the cap applies to `.tsx`, it violates.

### Rule: "No `interface{}`/`any` escapes" (AGENTS.md, .golangci.yml ifaceguard)

- **No `any` found at the IPC boundary.** All Wails model types are concrete (`main.TransactionRequest`, `main.PowerRow`, etc.). The `catch (e)` blocks use implicit `unknown` (TS strict) or `any` (loose) — no explicit `any` annotation.
- **Confidence: hi** — clean.

### Rule: "Errors wrapped with context, never silently dropped" (AGENTS.md, wrapcheck/errcheck — Go side)

**`src/components/transactions/TransactionWorkspace.tsx ~stage() function — PreviewTransaction has no try/catch; a throw leaves the modal stuck in "previewing" forever`**
- **Confidence: hi**
- The `stage()` function awaits `PreviewTransaction(req)` with no error handler. If the IPC call rejects (runtime/binding error), `pending.previewing` stays `true` and the modal shows "Checking the move with the engine…" indefinitely. The `confirm()` function correctly uses `try/finally`, but `stage()` does not.
- **Confirm:** `grep -A5 "async function stage" frontend/src/components/transactions/TransactionWorkspace.tsx` — no try/catch around the `await PreviewTransaction` call.

**`src/components/transactions/TradeBuilder.tsx ~stage() function — same missing try/catch on PreviewTransaction`**
- **Confidence: hi** — identical pattern.

**`src/components/transactions/LeagueControls.tsx ~stage() function — same missing try/catch on PreviewTransaction`**
- **Confidence: hi** — identical pattern.

**`src/components/TransactionsPanel.tsx ~run() function — try/finally with no catch; rejection from ExecuteTransaction is silently swallowed`**
- **Confidence: med**
- `void run()` is called from `onClick`; a rejected promise inside `run()`'s try block hits `finally` (resetting `busy`) but the error is never surfaced to the user — it becomes an unhandled promise rejection in the console.
- **Confirm:** check if `run()` has a `catch` block (it does not appear to).

### Rule: "No package-level mutable state / globals" (AGENTS.md, gochecknoglobals — Go side)

**`src/store/harness.ts:10 — `let powerReqSeq = 0;` is module-level mutable state`**
- **Confidence: med**
- The Go side bans package-level `var`/`let` via `gochecknoglobals`; the TS side has no equivalent linter. This counter is functionally correct (request sequencing) but is hidden mutable state outside the Zustand store. It could be a closure variable inside `create()` or a `useRef`-equivalent on the store.
- **Confirm:** `grep "^let " frontend/src/store/` — this is the only module-level `let` in the store.

### Rule: "Consistent error/loading handling"

**`src/store/harness.ts vs. transaction components — two completely different error strategies`**
- **Confidence: hi**
- The harness store: every IPC call is wrapped in try/catch, errors go to `set({error: String(e)})`, and loading flags are managed.
- The transaction components: errors are encoded as `{ok: false, detail}` in the response (never thrown), so no try/catch is needed for the happy path — but a runtime throw is unhandled. Loading is managed via local `busy` state with `try/finally` in `confirm()` but not in `stage()`.
- **Confirm:** compare error handling between `harness.ts:loadRankings` (try/catch) and `TransactionWorkspace.tsx:stage` (no try/catch).

---

## Cross-Cutting Synthesis: Top 5 Highest-Value Slimming Moves (Whole Codebase)

| # | Move | Lines saved | Risk | Rationale |
|---|---|---|---|---|
| **1** | **Delete `TransactionsPanel.tsx` + flip `SHOW_DEV_PANEL`** | ~280 | near-zero | All 14 ops are ported to the M4 surfaces. Eliminates the biggest file, a standards violation (IPC-in-component), and the most complex single component. |
| **2** | **Extract `useTransactionStaging()` hook from the 3 M4 components** | ~120 | med | The stage→preview→confirm→cancel lifecycle + `stageGen` token is triplicated with near-identical bodies. Unifying eliminates the race-handling divergence risk (today all three are correct, but a fourth consumer would likely copy one and miss the L3 guard). |
| **3** | **Delete `store/ping.ts`** | ~20 | zero | Zero importers; B0 reference superseded by the harness store. Pure deletion. |
| **4** | **Extract shared `transactions/format.ts` + shared `Empty`/`Th` components** | ~25 | trivial | Four utilities copy-pasted across `TradeBuilder` and `TransactionWorkspace`. Mechanical extraction. |
| **5** | **Add try/catch to the three `stage()` functions (or fold into the hook from #2)** | ~15 (net positive after bug fix) | low | Today a runtime throw from `PreviewTransaction` leaves the modal stuck in "previewing" forever. The hook extraction (#2) solves this naturally; if done independently, each `stage()` needs a `catch` that sets `pending.previewOK = false`. |