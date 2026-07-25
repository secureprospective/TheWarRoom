# Handoff 55 — B-4a Contextual Inspector (per-player score breakdown)

**Date:** 2026-07-25
**Branch:** `session/ui-b4a-inspector` (HEAD `4347e56`, pushed)
**Status:** Frontend + Go build ✅ · `go test -race` ✅ · GLM 5.2 review ✅ · **Visual gate PENDING** (merge held)

## What shipped
The Contextual Inspector's per-player anatomy (B-1 shipped only the empty shell).

**Backend seam (read-only):** `GetPlayerScore(mflID) → PlayerScoreResult` (`m1_player_score_app.go`) surfaces the full persisted breakdown from `season_scores` — `BasePoints`, `AgePull`, Layer-4 sub-signals (`FilmEffective`/`RASEffective`/`BreakoutEffective`) → `Combined`, `ScoutingAdjusted`, `CapMultiplier`/`CapTier`, `AdjustedScore`, veteran flag. Pure read projection off the existing `Score()` getter — **no recompute, no schema change**. Mirrors `GetRankings`' resolution (`ActiveVersion` → output reader) and degrade-not-hide posture (names-offline warns, breakdown stays complete). `FilmRaw` deliberately excluded (DEBUG-only, never-UI). Bindings regenerated via `wails generate module`.

**Frontend:**
- `store/inspector.ts` — shared selection + IPC (WF5), monotonic `selectSeq` stale-fetch guard, `openNonce` so re-selecting the same row re-opens the panel.
- `components/inspector/InspectorContent.tsx` — score-dominant hero, 6 layer bars (multipliers centered at 1.0: boost>1 green-right / penalty<1 amber-left, exactly 1.0 neutral), composite section, contract/cap block, loading/not-found/error states.
- `RankingsBoard.tsx` — rows are click/Enter/Space selectable, neutral `.is-selected` axis.
- `App.tsx` — feeds `inspector` node, opens on `openNonce`.

## Honesty rulings baked in
- **No hero hue-banding:** raw `AdjustedScore` is not the normalized 0–100 the Session-C ≥90/80/70/55 bands assume, so the hero is achromatic; quality reads through the per-layer bars + cap tier (defined scales), not an invented threshold. *(Flag for the gate: if a normalized score lands later, revisit hero banding.)*
- **capEff** computed identically to `GetRankings` (consistency).
- Empty franchise → `—`, never an unconfirmed "Free agent".

## Decision honored
M1 diagnostic **re-home is additive here** (Inspector now carries AgePull/L4/CapTier/CapEff); the **board sweep-out is B-4b** (Christopher, 2026-07-25) — so the M1 board is untouched this session and needs no re-gate.

## GLM 5.2 review (triaged vs source)
Fixed: reopen-on-reselect (openNonce), NaN bar guard + 1.0-neutral color alignment, false "Free agent", stale label clear. Non-issues verified: capEff mirrors GetRankings; `React.ReactNode` matches house pattern (tsc clean); no negative multipliers; value-DTO is house style + `res.ok`-guarded.

## Next
Visual gate (`/root/paste.md`, Beelink `/home/chris/opencode/TheWarRoom`). On **PASS** → squash-merge to main (linear history), delete branch, update CLAUDE.md header. Then **B-4b** = sweep the now-redundant diagnostics out of the M1 board + the Home 2×2 + 3–4 Alpha seasonal cards.
