# Handoff 58 — B-5 HARDEN **MERGED** · next stop is the ALPHA GATE

**Date:** 2026-07-25
**Merged to main:** `53fea25` (branch `session/ui-b5-harden` squashed and deleted)
**Status:** **B-5 COMPLETE. The UI build track B-1 → B-5 is DONE.** Next is the Alpha gate — Christopher running his real league on it, not another build session.

---

## ⚠️ The one thing that is NOT verified — read this first

**The outage test (visual gate section D) was deferred for time and has NEVER BEEN RUN.**

That is the section that exercises the degradation path *itself* — the whole headline feature of B-5. Sections A/B/C all passed, but every one of them renders the **live** case. What has never been seen on screen:

- the amber `CACHED` bar actually appearing,
- the age string (`last updated 4m ago`) being correct and readable,
- the rule that **the rows below stay fully legible** (not dimmed/faded) actually holding,
- the bar *disappearing* cleanly when the network returns.

**What IS proven:** the store layer is covered by Go tests, including a regression test for the context bug described below. So the data path is tested; the *rendering* of the degraded state is not.

**Risk if wrong:** bounded but real. The change is additive — the worst realistic case is that M2 behaves as it did before B-5 (blank during an outage), not that it corrupts anything. But do not *rely* on the cached board during a real MFL outage until section D has been run.

**How to run it** (needs ~5 min at a real machine): load M2 successfully once, disconnect the network, trigger an M2 reload, confirm the board still shows 32 franchises under an amber `CACHED` strip with the numbers fully readable, then reconnect and confirm the strip clears. The full script is in the git history of `/root/paste.md` for this session.

---

## What shipped

**The degradation contract (items 1 + the standings seam, done as ONE piece).** This was the shape-setting decision and the reason it went first.

- `freshness.go` — a `Freshness` DTO (`live` / `stale` / `fail` + `fetchedAt` + note) now shared by M1 and M2. Before this, each module invented its own answer to "the fetch failed": M2 treated it as fatal, M1 used a free-text warning, Home refused to show anything network-backed.
- `internal/store/state/standings_cache.go` — last-known-good MFL standings. **It is the only MUTABLE table in that package**; every sibling is an append-only ledger with `RAISE(ABORT)` triggers. The file comment explains at length why a cache is categorically different from a ledger — *do not "fix" it into a log*.
- `m2_standings_source.go` — the seam that decides live-vs-cache. One function, so there is exactly one place to audit the question "can this board lie about its age?"
- **Offseason is deliberately NOT a fourth freshness state.** An offseason board is fresh data about a finished season. It rides a separate `Phase` field and a neutral grey `PhaseBar`. Conflating them would label a correct board as degraded for months and train the user to ignore the signal.

**§1 delta-in-weight (item 2).** `+Δ` = font-weight 600, `−Δ` = 400, greyscale-honest. The delta source needed no new storage: `season_scores` is keyed by `(season, scoring_config_id, mfl_id)`, so a *previous config's rows ARE the previous board*. No prior board renders as an em dash — **absent, not zero**, because "held position" and "we don't know yet" are different claims.

**Dead `RankRow` fields stripped (item 3).** `agePull` / `l4Combined` / `capTier` were serializing on ~1200 rows per board read with nothing reading them.

**Board keyboard nav (item 4).** J/K travel, Enter commits. The **cursor is deliberately distinct from the selection** — travelling the board fires no IPC, Enter is the commit. `isTypingTarget` was extracted so App's guard and the board's are one definition.

**Item 5 (in-app feedback capture) was CUT** by Christopher — Alpha's audience is one person who already has better channels.

---

## Two traps this session found the hard way

**1. The grep in handoff 57 was wrong.** It told a future session to run `grep -n "r\.agePull"` to find dead `RankRow` uses. That pattern **matches `player.agePull`** — the `r` matches the "r" in "playe**r**". The correct check is a word boundary (`\br\.`), or simply: `RankingsBoard.tsx` having zero hits is the confirmation, and all three `InspectorContent.tsx` hits are `PlayerScoreDTO` and must stay. Both DTOs share field names.

**2. The GLM review caught a hole that would have silently defeated the entire cache.** `standingsOrCache` passed the *caller's* context to the fallback read. The most likely way the fetch fails is a **timeout** — at which point that context is already dead, so the local SQLite fallback failed instantly and the board reported "no data" while a perfectly good cached copy sat in the database. **The cache was broken in exactly the scenario it existed for.** The same flaw applied to the cache *write* (a fetch succeeding on its last second would never populate the cache — invisible until the next outage) and to the phase label. Fixed via `fallbackParent()`, which derives from the app-lifetime context. The two `//nolint:contextcheck` suppressions ARE the fix, not a workaround — the linter is objecting to the deliberate part.

This is the argument for keeping the review gate: it was not a style nit, it was the feature not working.

---

## Review method that worked

Split across two nodes in parallel (bird = backend, Hermes = frontend), both returned `finish_reason=stop` in a couple of minutes. **Three-way splitting was chosen because one 36KB part alone exceeded the payload that reset the connection last time.** Each part got a prompt naming its own failure modes and front-loading the invariants the reviewer could not see from a diff — including an explicit list of *deliberate* decisions not to re-litigate, which stopped it reporting intentional design as defects.

Asking the reviewer to **say when it was unsure** paid off directly: it flagged the grid-track count as "tentatively consistent, I would not call it verified" and named exactly what it could not see, rather than guessing. Two leads were triaged out as false positives against source (`Date.parse("")` already guarded; `isFinalPhase` correct — `OFFSEASON` is the only terminal phase in `domain.Phase`).

---

## State

- **main `53fea25`**, clean, pushed. Branch deleted locally and on origin.
- Gates: `make lint` 0 issues · `go test -race ./...` pass · `tsc --noEmit` + `vite build` clean.
- Grid tracks hand-counted and balanced in all four states: **8 / 9 / 6 / 6**.

## Next

**The Alpha gate is not a build session.** It is Christopher running his league on the binary. The open items, smallest first:

1. **Run the outage test** (section D above) — the only unverified thing in B-5.
2. Alpha itself: use it, and let real friction generate the next backlog rather than guessing at one now.

Deferred, not lost: M2/M4 refactors, FILM Thread C, and the B-3 calendar board UI (the calendar backend is merged but has no board yet).
