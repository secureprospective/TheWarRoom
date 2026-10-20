HANDOFF — Session 31: B7c cont. — Extension (§10) / Buyout (§12) / §13 Special
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-04 (B7c §9 tag session close)

== WHERE WE ARE ==
- B7c §9 Franchise Tag MERGED to main (squash `08e4250`, 2026-07-04). Core tag flow
  functionally verified live on the Beelink (all 4 gate steps); GLM-5.2 blind review closed
  (2 real fixes applied + planted-tested, rest refuted/confirmed — see
  `docs/build-handoffs/reviews/b7c-tag-glm-review-outcome.md`).
- Tag mechanics: `internal/transactions/pricing.go` = a `Directory` port (players-DB position
  join — state stays position-blind, the M1 seam) + pure `tagPrice` (top-5 BASE salaries at
  the position league-wide, averaged, half-up) + `tagFloorPrice` (max(avg, 120%×prior base
  salary)). `Coordinator.ExecuteTag(ctx, mflID, dir)` resolves the price authoritatively from
  its own Reader + a per-call Directory (the Lookup is lazy), delegates to Execute. Handler
  `contracts.Tag`: sets salary=price + IsTagged, RESETS IsRestructured (a tag is a fresh
  contract), one-per-team-per-season via `transaction_counts` (op_kind "TAG").
- Composition rulings locked with Christopher: tag pool = BASE salary (not effective);
  Restructure now REJECTS a tagged player (a tag is a fixed one-year deal). Tagging a
  restructured player is allowed (resets the flag).

== LOCKED DECISIONS THAT CARRY (reuse, don't re-litigate) ==
- Money = int64 cents, FLAT math, CapUsed = SUM of contract salaries. No float money.
- Per-season op limits via `transaction_counts` + TxWriter `OpCount`/`IncOpCount`, one row
  per (franchise, season, op_kind). op_kinds live: "RESTRUCTURE", "TAG". Ceilings are
  per-op-kind, enforced IN-HANDLER behind the depguard (M3 — never a table CHECK).
- Cross-store position/facts reads go through the `Directory` port (normalize.Lookup),
  resolved in the Coordinator, NOT in the position-blind state store.
- Authoritative resolution: no money crosses the IPC boundary; the frontend sends ids, the
  Coordinator computes figures from committed state.
- ACCEPTED v1 TRADEOFF / carry-forward: ExecuteTag resolves the price OUTSIDE the spanning tx
  (the league-wide read isn't on TxWriter). Fine for the single-user desktop app (wmu
  serializes writes); the proper fix (a league read inside the tx) waits until TxWriter grows
  a read surface OR concurrency becomes real.

== WHAT'S LEFT IN B7c (Build_Tracker row 27) ==
Read `docs/league-rules/Official_Rulebook.md` §10, §12, §13 and triage EVERY number vs source.

1. **Extension (§10)** — the natural next commit (no new store dependency; clone the
   restructure/tag handler shape). Rules: a position FLOOR table (§10 — salary can't extend
   below the position floor); "≥1 year remaining, UFAs ineligible"; adds ≤3 years, ≤6 total;
   extension years priced at 150% of the highest-paid remaining year; one extension per GM per
   season (transaction_counts, op_kind "EXTENSION"); no second extension off a prior extension.
   NOTE the interaction: each extension unlocks one more restructure (§11) — the per-contract
   restructure guard (is_restructured) needs to reflect that (an extension should re-allow a
   restructure). GATE-CHECK: how "extension years" and "≤6 total" map onto the current contract
   fields (ContractYears / ExpirationYear) — the store tracks expiration + years, confirm the
   shape before coding. Tagged-player extension (§9 line: "tagged player may be extended,
   extension years cost 120% of tag price/yr") is a real cross-rule — gate-check whether it's
   in scope here or its own follow-up.
2. **Buyout (§12)** — BLOCKER: needs an OFFSEASON / season-phase concept (two per team,
   offseason ONLY) that does not exist yet. Gate-check the season-phase design FIRST. Rate
   table by years remaining (2→60% / 3→75% / 4→90% of average remaining salary); writes a
   dead-cap charge (reuse the §8 deadcap path).
3. **§13 Special Situations** — mostly commish/admin: Cap Relief Appeal (commissioner reduces
   a cap hit), Gaines Adams Rule (player death → remove, NO cap penalty = a cut with ZERO dead
   cap — trivial reuse of the release path), Retirement (30% of remaining contract per year
   left → a dead-cap charge, a close cousin of §8). Retirement + death reuse the waiver/deadcap
   machinery.

== READ FIRST ==
- `docs/league-rules/Official_Rulebook.md` §10 / §12 / §13
- `internal/transactions/contracts/contracts.go` (Restructure + Tag — the handler templates)
- `internal/transactions/pricing.go` (the Directory + cross-store read pattern, if §10 needs it)
- `internal/transactions/request.go` (sealed Request set — mirror to add Extension)
- `internal/store/state/writes.go` + `types.go` (ApplyContract, ContractChange, OpCount)
- Build_Tracker row 27

== PROCESS (unchanged) ==
- Branch `session/b7c-extension`. No work on main. depguard stays PROVEN (plant a rankings
  import, watch it fire, revert — don't commit the plant).
- Review = GLM 5.2 BLIND on the Beelink (opencode `build` agent; Claude can drive it over SSH
  with `/root/.ssh/beelink`, source `~/.config/opencode/zai.env`, pipe a self-contained
  inlined prompt into `opencode run --agent build`). Gemini fallback. Leads, not findings.
- Gate: lint 0 / go test -race green / tsc+vite clean / depguard proven / formula table pinned
  (every fencepost) / functional gate run live on the Beelink before merge.
- GLM track record (running): tag 2 / restructure 0 / B7a 3 / M1 3 / B6 1 / K 0 / S 0 / CB 0 /
  LB 0 / DE 0 / TE 0 / WR 1 / RB 1 / DT 3 / QB 2.
