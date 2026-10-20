# Handoff 56 — B-4b: M1 diagnostic sweep-out + Home 2×2 landing

**Date:** 2026-07-25
**Branch:** `session/ui-b4b-home-diagnostics` → **squash-merged to main `7f35a6d`**, branch deleted
**Status:** Frontend + Go build ✅ · `make lint` 0 ✅ · `go test -race` 49 pkgs ✅ · GLM 5.2 review ✅ · **Visual gate PASSED** + post-fix Home spot-check PASSED (Beelink)

**B-4 is now COMPLETE (4a Inspector + 4b sweep/Home). Next is B-5 harden → ALPHA GATE.**

---

## Half 1 — M1 diagnostic sweep-out

The board now carries the locked facet map **exactly**: `Rank·Player·Pos·Franchise·Base·Adjusted·Salary`. The B-2 recessed diagnostic group is gone — **AgePull / L4 / CapTier** are engine internals the B-4a Inspector already carries per player, so keeping them on the board duplicated the Inspector at the cost of the 7-col lock.

**Adj/$M is the deliberate exception, and this is the load-bearing decision:** it is *not* a diagnostic. It is the locked **"Cap efficiency view"** (`UI_Direction_Document` §12.2) — a **board-level lens** (rank the league by score-per-$M) that a per-player Inspector structurally cannot express. Sweeping it would have deleted a locked capability. So it is *bound to that lens*:

- the column renders **only while the cap-eff chip is on** (`COLS` 7 tracks → `COLS_CAPEFF` 8)
- never in Matrix (`.twr-hide-mtx`), which stays the pure `Rank·Player·Pos·Adj·Sal` scan
- turning the chip **off** now also drops a `capEff` sort back to `adjusted` — otherwise the board sat ordered by a column it no longer rendered, an invisible sort state with no header to undo it

## Half 2 — Home 2×2 landing

`HomePlaceholder` replaced by `components/home/` — the quadrant grid (`UI_Direction` §10.4, Session-A §6). Forces narrative density by **re-declaring `data-density` on the container**, so the shell's global tier is untouched when the operator leaves. Stacks to one column under 1100px.

**Home reads LOCAL state only.** Two things it deliberately does *not* do:
- **`GetPowerRankings`** — live MFL fetch, **fatal on failure** (`m2_app.go:79`). A hard-failure network path on the calmest screen in the product is not acceptable.
- **`GetRankings`** — it *reads* the persisted board but never triggers its load, because name resolution inside it is a cached players-directory MFL fetch. M1 is the default module and loads it on mount; a cold landing engraves instead. (This was GLM lead H1 — the first cut did call it.)

### Seasonal cards — why 4 and not the spec's 9

`UI_Direction` §11 locks a **nine-card trigger map keyed off calendar dates** (contract options, RFA tender, UFA bidding, re-signing, rookie draft, UDFA, cut day, in-season). **The Phase-1 backend does not carry those dates.** It knows three phases (`domain.Phase`) plus the append-only commissioner calendar. Hardcoding nine date windows against data the engine was never told would be inventing state.

So the alpha set is driven by what the backend actually knows (`components/home/seasonal.ts`, **pure**):

| Card | Trigger | Content |
|---|---|---|
| `deadline` | a PLANNED calendar op inside 7 days | countdown hero + queue |
| `offseason` | phase OFFSEASON | scheduled commissioner ops |
| `inseason` | phase REGULAR_SEASON | top-5 asset pulse off the persisted board |
| `playoffs` | phase PLAYOFFS | rollover readiness |

The **deadline card is the honest generalisation** of the whole date-triggered family: the commissioner schedules the real deadline as a calendar blob and the card counts down to it, rather than the app asserting a date nobody gave it. 7 days = the tightest window in the locked map (RFA tender, May 1→3) rounded up.

**Three slots have no backend source and engrave:** league activity (no transaction-feed reader exists — the only `ledger` in `internal/store/state` is the per-player contract ledger, not an event log), trade block (no such concept), league chat (comms layer doesn't exist).

---

## GLM 5.2 review — split across two nodes

**The combined 33.5KB payload reset the connection twice** (`curl (56) Recv failure`, ~6 min in, from the Beelink). Christopher's call: split it. **M1 → bird, Home → Hermes, in parallel** — both returned `finish_reason=stop` in ~2 min. The reasoning window, not the payload size, was the problem. Each half also got a prompt targeted at its real failure modes, which produced a *better* review than one generic pass.

**Fixed (4):**
- **H1 HIGH** — Home called `loadRankings` on mount → cached MFL touch on the landing. Now reads without loading.
- **H2 HIGH** — `PlannedQueue` de-duplicated by `eventID`; a blank id would drop **every** row sharing it. Now skipped by **object identity** (`imminent` is always an element of `planned`), React key carries an index.
- **H3 MED** — only one error banner rendered; a corrupt/locked local DB showed as one failing card while hiding that the whole state engine was down. Both render now.
- **M2 LOW** — documented the unreachable `'—'` arm as a guard, not a second contract.

**Triaged to FALSE POSITIVE against source (2):** the cap-eff row filter *does* exist (`RankingsBoard.tsx:91`, `capEffOnly` in deps — GLM correctly flagged it as unverifiable-from-diff and predicted it would evaporate); Franchise *does* carry `twr-hide-mtx` (`:238`), so the matrix 5-track count balances.

**GLM verified independently:** track/cell lockstep balanced across all four `{chip} × {density}` states (it counted them), and the new chip handler **removes a real StrictMode hazard** the old code had — side effects inside a `setState` updater, which React double-invokes.

---

## Build

tsc + vite clean. JS 234.5 → **247.2KB**, CSS 16.95 → **18.55KB**. **No new deps. Go untouched** (build ✅, `make lint` 0, `go test -race` 49 pkgs 0 FAIL).

## Next — B-5 harden → ALPHA GATE

- wireframe §8 states still unwired: **cache-only** and **offseason** (need backend freshness / season-phase signals)
- §1 **delta-in-weight**
- **strip the dead `RankRow` fields** — `agePull`, `l4Combined`, `capTier` are read by **no frontend surface** after this session but still serialize on every row (~1200 rows/board read). Go/IPC change.
- board **keyboard nav** (deferred from B-2)
- in-app Alpha feedback capture

**Open backend fork, scoped OUT of B-4b:** a **cached/soft-fail standings read** would unblock a real Home standings card *and* fix M2's fatal-on-MFL-outage posture. Not started — worth folding into B-5, since cache-only/offseason are the same "degrade honestly instead of failing" problem.

## Operational lessons (both cost real time)

1. **`pgrep -f <pattern>` over SSH matches its own command line** — reported a dead job as alive twice. Use the bracket trick: `pgrep -af "[h]ermes-run"`.
2. **GLM 5.2 resets the connection when reasoning runs past ~6 min.** Split the review rather than retrying the whole payload; retrying the same shape failed identically.
