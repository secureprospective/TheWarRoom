# Handoff 57 — B-5 HARDEN · NEXT SESSION BRIEF (nothing built yet)

**Date written:** 2026-07-25 (end of the B-4b session, pre-compaction)
**Status:** **BRIEF ONLY — no code written, no branch created.** This is the entry point for the next session.
**Starting point:** main `d08e988` (code `7f35a6d`), clean, pushed. Working tree clean.
**Branch to create:** `session/ui-b5-harden`

> **Read this file first.** It is written to be self-contained for a cold session — it does not assume the previous session's context survived. Where a fact matters, the file/line is cited so it can be re-verified rather than trusted.

---

## Where the project stands

**B-4 is COMPLETE.** The UI build track is B-1 → B-2 → B-3 → B-4 → **B-5 (this one)** → **ALPHA GATE**. B-5 is the last build session before Christopher runs the league on it.

Merged and live on main: the 4-column shell (`ce710bc`), all modules on the `.twr-board*` grammar (`433d35b` + `13ca56d`), the calendar agenda board (`f370e5c`), the Contextual Inspector (`f028aaf`), and the M1 sweep + Home 2×2 (`7f35a6d`).

**Alpha's audience is Christopher alone** — the league was polled and nobody else wants to run the binary, so no binary leaves the machine. Versioning/releases is DONE and gates nothing.

---

## The five B-5 items

### 1. The degradation contract — §8 cache-only + offseason states  ⬅ **do this first, and as ONE piece**

`frontend/src/components/board/primitives.tsx` ships `EngraveState` (no data) and `SkeletonState` (fetch in flight). Its header comment names the two that were deliberately deferred to B-5: **cache-only** and **offseason**, because they need backend signals that did not exist.

**Design note carried forward from B-4b:** items 1 and 5 below are *the same problem* — surfaces that must **degrade honestly rather than fail or lie**. They currently read as separate items across two layers. Doing them as one pass yields a single freshness/degradation contract instead of three ad-hoc ones. **This is the last session before Alpha freezes that shape.** Recommend deciding the contract first, then wiring all consumers to it.

Session-C already locked the visual tokens: `--fresh-live` (blue, 1px), `--fresh-stale` (amber, 1px), `--fresh-fail` (red, 2px + timestamp + suffix), with the rule that **the data stays legible** — freshness is an edge treatment, never a blur or a hide. See `frontend/src/style.css` `:root` and `docs/ui/wireframes/session-c/`.

### 2. `§1 delta-in-weight`

Session-B locked it: **+Δ = font-weight 600, −Δ = 400**, greyscale-honest (hue may reinforce but must not carry the meaning alone). Deferred from B-2 because no delta signal was plumbed. Source: `docs/ui/wireframes/session-b/session-b-wireframe.html` §1.

### 3. Strip the dead `RankRow` fields  ⬅ **the easy win; Go/IPC change**

After B-4b, **no frontend surface reads** `agePull`, `l4Combined`, or `capTier` — they moved into the Inspector, which uses its own DTO (`PlayerScoreDTO`, `m1_player_score_app.go`). They still serialize on **every row of every board read** (~1200 rows).

- Defined in `m1_app.go` (`RankRow` struct, populated ~line 200).
- Requires `wails generate module` to regenerate bindings, then tsc.

**⚠️ A plain grep is AMBIGUOUS here — verified 2026-07-25.** Both DTOs use *identical field names*. `grep -rn "agePull\|l4Combined\|capTier" frontend/src/` returns three hits, and **all three are the Inspector's `PlayerScoreDTO`, which MUST stay**:

```
InspectorContent.tsx:39   player.agePull      <- PlayerScoreDTO — KEEP
InspectorContent.tsx:47   player.l4Combined   <- PlayerScoreDTO — KEEP
InspectorContent.tsx:190  player.capTier      <- PlayerScoreDTO — KEEP
```

The distinguishing factor is **the receiver, not the field name**: `r.agePull` = `RankRow` (delete), `player.agePull` = `PlayerScoreDTO` (keep). `RankingsBoard.tsx` currently has **zero** hits — that is the confirmation that the `RankRow` copies are dead. Re-verify with `grep -n "r\.agePull\|r\.l4Combined\|r\.capTier" frontend/src/components/RankingsBoard.tsx` (expect no output) before touching the Go struct.

### 4. Board keyboard navigation

Deferred from B-2, then from B-4a. Session-B locked the map: **J/K** row movement, **Enter** to select (opens the Inspector — the selection store already exists at `frontend/src/store/inspector.ts` with `select()` + `openNonce`), **T**, **1/2/3** density, **Esc**. Global keys 1/2/3, `i`, `Esc` are already wired in `App.tsx`; J/K/Enter on the boards are not.

Note `App.tsx` already ignores keys while focus is in an `INPUT`/`TEXTAREA`/`contentEditable` — reuse that guard, don't reinvent it.

### 5. In-app Alpha feedback capture

"Flag this screen" → **append-only local log**. Named in the roadmap's final-pass gap items as a B-5 deliverable. Smallest honest version: a control that appends `{timestamp, module, note}` to a local file or SQLite table. No network, no telemetry.

### Fold in if scoped: cached / soft-fail standings read  (Go)

`GetPowerRankings` does a **live MFL fetch and treats a standings failure as FATAL** — `m2_app.go:79` says so in a comment, and `m2_app.go:105` is the fetch. Consequences today:
- M2 is unusable during an MFL outage even though scores are persisted in SQLite.
- **Home cannot carry a standings card at all** — B-4b scoped it out for exactly this reason (Christopher's call), which is why the Home seasonal card falls back to an asset pulse instead of the standings summary §11 actually specifies for the offseason slot.

A read that serves last-known values and degrades fixes both. This is the same contract as item 1 — hence the recommendation to do them together.

---

## Hard constraints that bit during B-4b (do not re-learn these)

- **Home must stay local-only.** It calls neither `GetPowerRankings` (fatal MFL) nor `GetRankings` (its *name resolution* is a cached players-directory MFL fetch — `app.go` `directory()` → `players.Fetch`). Home READS the board M1 already loaded and engraves on a cold landing. GLM caught the first cut violating this. If B-5 adds a Home data source, re-check this.
- **Adj/$M on M1 is NOT a diagnostic.** It is the locked "Cap efficiency view" (`UI_Direction_Document` §12.2) — a board-level lens the per-player Inspector cannot express. It is bound to the cap-eff chip (column renders only when the chip is on; `COLS` 7 tracks → `COLS_CAPEFF` 8; never in Matrix). **Do not "finish the sweep" by removing it.**
- **Grid-track/cell lockstep is a real correctness contract.** `--twr-cols` drives BOTH the sub-header and every data row; `--twr-cols-mtx` overrides at matrix density; `.twr-hide-mtx` cells vacate their tracks. Any column change must balance in all `{state} × {density}` combinations. tsc and lint **cannot** see a violation — count them, or have the reviewer count them.
- **Never surface confidence scores in UI** (engine-internal flags only). **No `git --no-verify`.** **No work on main.**

## Build / gate commands

```bash
export PATH=/usr/local/go/bin:$PATH GOMEMLIMIT=1500MiB GOGC=20
go build ./... && make lint && go test -race ./...      # lint OOMs on CT105 without the env vars
cd frontend && npx tsc --noEmit && npm run build
```

**Beelink visual gate:** clone is `/home/chris/opencode/TheWarRoom`; `wails dev` **needs `-tags webkit2_41`** (system has webkit2gtk-4.1, not 4.0). Push the branch to origin FIRST — the Beelink pulls from origin and cannot see an unpushed local branch. Commands for Christopher go in `/root/paste.md`, labeled, one batch.

**GLM 5.2 review gate:** **split the review across bird + Hermes in parallel** — a combined payload past ~6 min of reasoning resets the connection (`curl (56)`, zero-byte body, nothing salvageable). Give each part a prompt naming its *specific* failure modes and front-load the invariants, or the reviewer reports deliberate absences as defects. Recipe: memory `reference_glm_review_over_ssh`; lesson: `lesson_split_the_review_not_retry`. A sentinel file means the job **ended**, not that it **succeeded** — check the exit code and `finish_reason`, not just the sentinel.

## Suggested order

1. Decide the **degradation/freshness contract** (items 1 + standings seam) — it is the shape-setting decision and the last chance before Alpha.
2. Item 3 (dead `RankRow` fields) — mechanical, gets a Go/IPC change through the gates early.
3. Items 2, 4, 5 — independent, any order.
4. Full gate stack → Beelink visual gate → split GLM review → triage → merge → **ALPHA GATE**.

**Ask Christopher before starting** which of the five he wants in scope — B-5 as written is large, and he has consistently preferred scoping down over scope creep ("biggest risk is scope, not missing ideas").
