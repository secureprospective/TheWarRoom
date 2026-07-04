HANDOFF — Session 30: B7c cont. — Tag (§9) / Extension (§10) / Buyout (§12) / §13 Special
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-04 (B7c restructure session close)

== WHERE WE ARE ==
- B7c §11 Restructure MERGED to main (squash `3041910`, 2026-07-04) and functionally
  verified live on the Beelink: all 5 gate steps PASS — cap drops by EXACTLY the move /
  over-max rejected / 2nd-per-team rejected / 2nd-per-contract rejected / a restructured
  cut charges 50% not 35%.
- New pkg `internal/transactions/contracts` behind the Coordinator depguard (PROVEN):
  pure `MaxMove` = the §11 tier step function (≥$3M→$1M / ≥$6M→$2M / ≥$12M→$3M, <$3M
  ineligible); `Restructure` lowers the adjusted (cap-counting) salary by the owner move,
  cap drops by exactly the move (CapUsed is the SUM), sets is_restructured.
- NEW durable table `transaction_counts` `(league,franchise,season,op_kind)→count` with
  TxWriter `OpCount`/`IncOpCount` — THIS IS THE REUSABLE PER-SEASON OP LIMITER every
  remaining op needs (tag/extension/buyout all cap per-team-per-season counts here).
- Limits made unrepresentable: is_restructured guard (one per contract), transaction_counts
  (one per team per year), move≤tier-max, local move≤effective non-negative guard (GLM M1).
- Sealed `Restructure{MFLID,Move}` request; IPC + React "Restructure" dev control (millions
  → exact cents via `domain.ParseMoneyMillions`, integer string-math, NO float money).

== DEFERRED FROM THIS SESSION ==
- **M3 (carry-forward):** no DB-level ceiling on the per-season counter. A blanket
  `CHECK(count≤1)` is WRONG — tag/buyout allow 2/year, so ceilings are per-op-kind and
  correctly live in the handlers behind the depguard. Keep enforcing in-handler; do NOT add
  a table-level CHECK. Revisit only if a per-op-kind ceiling column is ever justified.
- Drop the frozen legacy REAL money columns one release on (still pending from B7b).

== WHAT'S LEFT IN B7c (Build_Tracker row 27) ==
Remaining ops, each rulebook-grounded. Read `docs/league-rules/Official_Rulebook.md` §9–§14
and triage EVERY number vs source before coding. Suggested order (dependency weight):

1. **Franchise Tag (§9)** — FIRST NEW DEPENDENCY: cross-store position aggregation. Tag
   price = average of the top-5 salaries at that position LEAGUE-WIDE (a read across ALL
   rosters — new read seam, mirror M1's Directory/cross-store pattern, NOT a single-franchise
   read). Floor: if tag price < player's prior-year salary → 120% of prior year. Max two
   CONSECUTIVE years (2nd tag = 120% of the first). Sets is_tagged. Per-team-per-season count
   via `transaction_counts` (op_kind "tag", ceiling 2 in-handler). NO dead-cap charge on tag
   itself. This is the natural next commit.
2. **Extension (§10)** — position FLOOR table (§10) + gate "≥1 year remaining, UFAs
   ineligible" + adds ≤3 yrs, ≤6 total + extension years priced at 150% of the highest-paid
   remaining year + one per GM per season (transaction_counts) + no second extension off a
   prior extension. Multi-op history. NOTE: each extension unlocks one more restructure
   (per §11) — wire that interaction.
3. **Buyout (§12)** — needs an OFFSEASON / season-phase concept (two per team, offseason
   ONLY — this is the blocker: there is no season-phase state yet, gate-check it first).
   Rate table by years remaining (2→60% / 3→75% / 4→90% of average remaining salary).
   Writes a dead-cap charge (reuse the §8 deadcap path).
4. **§13 Special Situations** — mostly commish/admin: Cap Relief Appeal (commissioner
   reduces a cap hit), Gaines Adams Rule (player death → remove, NO cap penalty — a cut with
   ZERO dead cap), Retirement (30% of remaining contract per year left → a dead-cap charge).
   Retirement + death are close cousins of the §8 waiver path already built.

== READ FIRST ==
- `docs/league-rules/Official_Rulebook.md` §9–§14 (source of truth for every number)
- `internal/transactions/contracts/contracts.go` (the §11 pure-formula + handler template
  — clone this for tag/extension; note the transaction_counts limiter pattern)
- `internal/transactions/deadcap/deadcap.go` (the dead-cap charge template for buyout/§13)
- `internal/transactions/request.go` (sealed Request set — mirror to add Tag/Extension/etc.)
- `internal/store/state/writes.go` (ApplyContract, ReleasePlayer, AddDeadCap, OpCount,
  IncOpCount, Player, Season) + `types.go` (ContractChange, DeadCapEntry, TxWriter)
- `internal/rankings/` (the cross-store read pattern the Tag aggregation should mirror)
- Build_Tracker row 27

== GATE-CHECK BEFORE CODE (Tag) ==
- Confirm the top-5 aggregation is over CURRENT rostered salaries at the position (adjusted
  vs base — likely base Contract-Year Salary, mirror §11's base-tier choice; confirm w/
  Christopher).
- Confirm "prior-year salary" source for the floor (the frozen REAL col is unread — read
  the cents col; a first-year-in-league player has no prior year → gate that case).
- op_kind string convention for transaction_counts ("restructure" is live — pick "tag" etc.).

== PROCESS (unchanged) ==
- Branch `session/b7c-tag` (or per-op). No work on main. depguard must stay PROVEN (plant a
  used import from rankings, watch it fire, revert — do NOT commit the plant).
- Money stays int64 cents end-to-end; float only at the 3 documented display edges.
- Review = GLM 5.2 BLIND on the Beelink (opencode `build` agent, `/coding/paas/v4` endpoint,
  key from `~/.config/opencode/zai.env`); Gemini fallback if GLM is down. Leads, not findings
  — triage each vs source.
- Gate: lint 0 / `go test -race` green / tsc+vite clean / depguard proven / formula table
  pinned (every fencepost) / FUNCTIONAL gate run live on the Beelink before merge.
- GLM track record (running): B7c-restructure 0 / B7a 3 / M1 3 / B6 1 / K 0 / S 0 / CB 0 /
  LB 0 / DE 0 / TE 0 / WR 1 / RB 1 / DT 3 / QB 2.
