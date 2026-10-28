# Handoff

## Baton
Bee — 2026-10-28, `session/terra-p7-split`. Do not push without direction.

## Where it stands
Removed historical market, draft-lab, counterparties and offer-history frontend screens and removed Sources/Archive routes. Wails bindings regenerated. Typecheck, 1,572 Vitest tests, Go build, uncached composition tests and bindings freshness pass.

## Next move
Review the frontend removal and live-test preserved draft room, asset board, player values and pool. Reconcile the generated command catalogue: it still describes the removed offers read and filter; the frontend refuses the filter. `datasources/DataSources.tsx` remains imported by its tests; shared draft report components remain in use.

## Blocked on
No upstream configured; pull and push unavailable. Wails generation emits existing missing generic-model warnings despite passing freshness.

## Tried and rejected
Wire fixture regeneration produced unrelated volatile changes; discarded them and reran clean.

## Debt left
Full race suite timed out under concurrent workload; targeted race tests passed. UI affordances for selecting
franchise in quotes and exception forms have not been implemented or verified.
Bee — 2026-10-28, `session/terra-side-ui`. Do not push.

## Where it stands
- Commissioner bid choices identify player and side; the tray sends both. Single-pool choices are unchanged.
- Startup draft readings identify the viewed side; live commissioner tray actions send it, except open.
- Build, touched Go race tests and composition retry, frontend suite and typecheck passed. Initial concurrent
  tests timed out under load. Hosted fixture regeneration made volatile changes; those were discarded.

## Next move
Review the branch and test AFC and NFC auction decisions and startup draft actions in the GUI.

## Blocked on
Nothing known; the GUI click-through remains outstanding.

## Tried and rejected
No stored side map: sides are counted from conferences. Fixture timestamp and identity churn was not retained.

## Debt left
Rookie draft per side; per-side O-Line lots; reserve and rule-exception scoping by franchise; per-side
contract quote. Older F3a O-Line transition question belongs to Christopher/ClaudeBox.
Tom (Beelink) — 2026-10-28, `session/migration`. Do not push.

## Where it stands
- "Migrate my MFL league" engine, M1–M5 (brief `~/fleet/briefs/tom-migration.md`):
  `internal/leaguemigration` (read, prepare, review, invites, assist), `mflsource` (read-only MFL
  reader), `commish.league.cutover`, the "Your MFL league" settings entry, the all-or-nothing ledger batch,
  and six `league.migrate.*` registry commands on the assist door, wired in composition.
- Proof: round trip on the held Legacy NFL (14432) and 10 synthetic edge leagues, checked against the raw
  MFL export field by field; 7 injected failure points leave nothing; scripted AI assist run end to end
  through the registry. Click script: `docs/build-handoffs/MFL_Migration_Click_Test.md`.

## Next move
Christopher reviews the report. Then the UI debt below, then the click walk (part B) on Claude-OS.

## Blocked on
Nothing.

## Tried and rejected
Storing MFL owner emails at import: they would sit in the league-visible ledger. Invites re-read MFL's owners
(read only) instead. Adding the move's commands to the Wails/native doors: that changes the frozen door
inventory and generated bindings; they sit on their own assist door until the UI needs them.

## Debt left
- UI: start-form "Import my MFL league", review screen, cutover confirm, invite screen; and
  `scoring.rules` shown on the scoring screen. None is built; part B of the click script waits on them.
- CLI: the move is not on `twr`'s hosted dispatch yet (assist door only).
- Invites are issued in the app; nothing emails the link. Invitee principal is the owner's email.
- MFL's owner emails come only with the commissioner's key: offered for 14432 only; other leagues list
  every owner for a hand invite unless a key is added.
- `mflsource` has no HTTP test (the MFL client pins MFL's hosts); it composes the existing fetchers.
- Settings MFL holds but the app has no field for (h2h, rostersPerPlayer, contract-year use) are not read;
  team-defense or DL/DB positions refuse the import with a plain list.
- A raw `commish.league.create` with the MFL preset accepts caller-supplied values (same power as setting
  rules by hand); the move's own path always builds them from the read.
Full composition race test timed out; clean non-race composition test passed.

## Bee — Terra L3 rookie drafts, 2026-10-28
- Baton: review holds it; branch `session/terra-rookie-side`. No push requested or attempted.
- Implemented in `5c84fae7`, `64ae7233`: rookie keys and replay per counted side; shared league order
  is validated first, then picks follow their current owner's side, retaining original identities and
  receiving side-local slots. Shared setup stays league-wide; live acts and close outcomes name a side.
- Flow: owner/tray command → validated league order → side replay → ledger → side-local roster/pool;
  storage reloads both drafts, and checkpoint restores compensate only the named side's live entries.
- Tests reproduce the original combined draft, differing traded originals across rounds, and missing
  shared-start checkpoints. They cover duplicate refusal without mutation, both copies, traded picks,
  close/issuance, real database reopen and persistent checkpoint restores. Single-pool digest pins pass.
- Final race PASS counts (top-level `^--- PASS`): commissioner 254; rookiedraft 33; history 234;
  composition 588 (isolated 20-minute retry); serverhttp 47. App commissioner: 49 PASS, two FAIL below.
  History's four opt-in fixture/performance tests skipped. Build and frontend typecheck pass.
  Frontend: 1,726 pass with a 20-second timeout; default five seconds timed out and cascaded to four fails.
- Required fixture regeneration ran. Discarded unrelated cutover inventory and volatile hosted identities/
  timestamps. Clean reruns retained the failures below; no wire fixture changes committed.
- Blocked gates: app commissioner `TestNativeCatalogPublication` and `TestCommissionerCommandSafeguardsWire`
  fail on the pre-existing `commish.league.cutover` inventory; both reproduced with original production
  code via a temporary Go overlay. `make verify` stops at inherited bloat: 25 tiny files versus 24 baseline;
  all 25 are unchanged from the starting revision. No gates or baselines weakened.
- Initial combined race run timed out composition at ten minutes; isolated retry passed in 1,156 seconds.
  Commit hooks pass. Production footprint across affected files: 2,254 → 2,392 physical lines;
  tests/docs separate. Source trace found no preset branch or new storage representation.
- Next: reviewer/GUI verification of AFC/NFC picks and checkpoint restores. No live GUI run performed.
- Debt left: inherited cutover catalog/wire failures and bloat baseline drift block full verify;
  GUI verification outstanding. Pull blocked by this branch having no upstream.
Hosted fixture regeneration changed frontend fixtures; reverted under the backend-only constraint.
Composition race suite timed out under load; non-race suite passed.
Baseline was tested from a temporary archive inside this worktree rather than using the other worktree, to honor the worktree-only instruction.
Did not update frontend fixtures or generated frontend command files: outside this slice. Did not commit while required checks are not green.
Did not alter Go command catalogue or endpoint CSV: outside this frontend slice. Removed workspace endpoints are retired in the frontend registry instead.

## Bee — Terra gate 1, 2026-10-28
- Baton: review; `session/terra-gate1`. No upstream configured; no push attempted.
- Removed retired `offers.filter` from the browser catalogue and handler; regenerated browser and doors surfaces. Targeted Go tests (16), frontend command tests (190), Go build and TypeScript pass.
- Bloat blocks lint: tiny files 27 versus 24. Documented `scripts/bloat.sh --update` refuses any increase; baseline unchanged. Nine files newly tiny and six no longer tiny since its baseline commit, including cleanup and platform moves. Boundary check independently flags `internal/app/core/host_lab.go` decision syntax.
- Placement, transport, isolated-cache golangci-lint, Wails version, vulnerability scan and bindings freshness pass; Wails emits existing missing-model warnings. Full verify/race suite not run.
- Next: resolve the ratchet policy conflict and review the lab boundary before rerunning verify. Shared golangci cache produced stale paths from another worktree; isolated cache was clean.

## Bee — Terra gate 2, 2026-10-28
- Baton: review; `session/terra-gate1`. No upstream configured; pull failed. No push requested.
- Removed unused `internal/app/core/host_lab.go`; retained `LabFacts` for the asset board. Folds: `internal/app/offers/bids/pricing.go` → `internal/app/offers/bids/plans.go`; `internal/store/history/replay_cache.go` → `internal/store/history/commissioner.go`.
- Bloat passes: comment 5/5, provenance 0/0, tiny 24/24, duplicates 0/0; update lowered only comment ceiling 13→5. Go build passes; non-race tests: bids 2 pass, history 234 pass/4 skip, core 20 pass, composition 578 pass/2 fail. Boundary checker tests 8 pass.
- Blocked: boundary check exposes unreviewed migration functions/source, changed adapter sources/functions and stale entries after removing the first blocker. Do not refresh fingerprints to hide changes; needs boundary policy review. Composition parity tests fail on inherited `offers.filter` metadata; both reproduced on the starting commit. Frontend untouched.
- Next: review migration and other changed boundary wiring against the architecture; reconcile retired offer filter metadata and frozen door tests. No GUI verification performed.
Bee — 2026-10-28, `session/terra-f9`. Do not push without direction.

## Where it stands
- `oline.auction.close` / "Close O-Line auction" is commissioner-only in the registry and CLI.
- Leftover NFL units alphabetically pair with unit-less franchises by stable ID. One-year contracts
  use the league's lowest known minimum salary; existing auction awards are preserved.
- Full cap/roster leaves a recorded reservation and a named cut prompt. The restricted due worker
  retries after room opens, including restart. O-Line off does nothing. No preset branching.
- Flow and boundaries: `docs/architecture/oline-auction-close.md`.
- Observed evidence: 10 new race tests pass; commissioner, market, serverhttp, history, leagueservice
  and commandcatalog full race package runs passed before the final test-only additions. Clean
  composition: 588 top-level tests / 828 including subtests pass without race. Build, TypeScript,
  generated catalogs and full golangci-lint pass. Fixture writers were run; volatile hosted values
  were not retained.

## Next move
Review the commit, then exercise the close command and jobs cut prompt on a disposable league.
Pass: no cap overrun; after the requested cut the reserved unit appears exactly once and the
prompt clears. A standing bid or tie must be decided before close.

## Blocked on / debt left
- Full composition race run timed out after 15 minutes; clean non-race suite passes.
- `internal/app/commissioner` has one failure: `TestNativeCatalogPublication` finds the pre-existing
  recognized `commish.league.cutover` command missing from the native catalog. Not changed here.
- `make lint` stops on inherited bloat: 25 tiny files against baseline 24. HEAD and candidate both
  have the same 25; no baseline raised or unrelated file removed. Full golangci-lint: 0 issues.
- No GUI click-through or dedicated close button added. No full verify/push claimed; no upstream.

## Tried and rejected
- Do not assign contested lots: outstanding bids/ties require settlement first.
- Do not overrun even a soft cap or use an exception to bypass the assignment cap check.
- Do not retain hosted fixture ID/timestamp churn or alter unrelated cutover publication.
