# HQ My moves — native history read (2026-11-17)

The approved hosted source is the server's native append-only command ledger. Desktop envelope
audits remain the local source; they are not imported or relabelled as native history.

## Sources and access

- Desktop: `frontend/src/app/shell/MyMoves.tsx:26` → `sessionMoves.ts:19` →
  `frontend/src/app/data/provider.ts:256` → `TargetMoves`; storage is
  `internal/store/mflref/moves/store.go:78` (`move_audit`) / `:110` (envelope receipts),
  with append-only triggers in `internal/db/main_schema.go:70-96`.
- Hosted: `internal/commissioner/ledger.go:158` (`Reading`) already owns safe descriptions,
  actor/AI stamps and audience filtering; the existing store read composes it at
  `internal/store/history/commissioner.go:472-492`. The new read uses the same native records,
  never desktop envelopes or raw command payloads.
- `docs/ui/data-catalog/DATA_FLOW_MAP.md:17`: **LEDGER** is “every member (filtered by Audience)”.
  “Append-only. Actor + ‘via AI’ stamped. Commish corrections are new rows, never edits.”
  Thus another admitted owner can see public franchise history, but not another owner's sealed
  bids/private rows. An outsider receives the standard league-membership refusal. `moves.read`
  remains owner-scoped and unchanged.

## Implemented flow

| Responsibility | Contract / effects / failure boundary |
| --- | --- |
| Catalog `franchise.moves.history.read` | Read-only `{league, franchise, season}`; hand-written Raw validation; existing hosted member admission and scoped AI credential admission. |
| `leagueservice.TargetFranchiseMovesHistory` | One captured native record slice from the existing host; derives the recipient revision using the existing request context; errors retain context. No writes, tables, migrations or acquisition. |
| `commissioner.FranchiseMovesHistoryReading` | First uses `Reading`'s audience policy; retains only franchise-relevant entries in one league-clock season, newest first. Governance authorizer derived from those records when the host supplies none, matching the existing store ledger convention. |
| Franchise association | Reuses preview target selection; follows transaction/reversal/bid/claim links, trade parties, outcomes and draft recipients. Pick ownership corrections include both previous and new owners. Unscoped league changes affect every franchise. |
| Season page | Uses `firstSeason` and `RolloverInstants`, not calendar year or a traded pick's future year. Originals and corrections remain separate entries; nothing is folded away. |
| `HQMoveHistory` | Mounted only by hosted HQ's My moves tab; calls `hostedWorkflow`, validates with `parseLedger`, previous/next season and explicit retry. Existing read hook fences superseded scope replies; hosted league/team topics refresh the page. Local HQ keeps its existing renderer. |

Boundary wiring review: `commissionerOwnerBindings` groups the existing seven owner-market handlers
unchanged plus the new typed `Bind1`. No policy, branch, argument rename or authority change in the
composition map. Updated only its source fingerprint after reviewing this additive grouping.

## Desktop fields not available in the native history reading

- Envelope correlation ID and planned/ready/handed-off/MFL-observed lifecycle states.
- Per-transition from/event/to/note timestamps and MFL handoff target/link.
- Expected off-app change, check verdicts, bid/contract plan terms and handoff deadline.
- Structured subject/player/pick and retained trade-asset labels (native history supplies safe prose).
- Per-move cap delta/result columns (the former HQ season table never populated them).

## Slop touched and preservation

Replaced hosted HQ's always-empty season-ledger promise with actual safe rows and honest empty/error
states. Reused existing parsing, authorization, clocks and read-lifetime machinery. No parallel history
store or lifecycle policy. The near-limit composition map is grouped by owner-market responsibility,
not split into extra files. Tiny adapters are kept in existing owner API/service files.

Four lint file-length reports concern untouched files: `leaguerules/catalogue.go`,
`store/history/commissioner.go`, `serverhttp/hosted.go`, `commissioner/rules_season.go`.
Their responsibility boundaries are unchanged; no size-only split was made.

## Evidence and limits

- Initial frozen frontend install/build passed before implementation; final build passed.
- Race tests passed for commissioner (+ market), leagueservice, commandcatalog,
  app/commissioner (+ native), composition and serverhttp; commissioner race batch rerun after
  the final pick-correction association fix.
- TLS hosted test proves owner history, another member's public reading, sealed-bid exclusion,
  outsider refusal, empty season page and unchanged owner-only `moves.read` refusal.
- Domain tests prove rollover paging, original/correction coexistence, actor/AI stamp,
  unknown/invalid scope, and pick correction association with both owners.
- Frontend: 115 files / 885 tests passed, including hosted HQ history, season navigation,
  refusal/retry, delayed-season/franchise fencing and unchanged legacy renderer. `make lint`:
  0 issues; Impeccable detector: no findings. Build retains existing chunk/dynamic-import warnings.
- No browser/device visual inspection, live server/database inspection or deployment. Manual check:
  open hosted HQ → My moves; expect dated native rows, actor/reason, separate corrections and
  working season navigation; verify another member cannot see sealed bids.
- Tidy check reports pre-existing Chromium symlinks and an unrelated active fleet launcher outside
  the permitted write scope. Left untouched. No session servers/watchers were started.

Footprint against the branch's `origin/main` merge base (`a480ceecd`), counting the complete
changed handwritten production file set: 1,706 → 1,986 physical lines; 88,739 → 100,085 Unicode
code points including LF (+280 / +11,346). This is a new read/projection/UI, not a debloat claim.
Generated output separately: 30,826 → 30,954 lines; 861,917 → 865,283 code points. Tests/docs
are excluded from production totals. No guarantees or tests were removed to manufacture savings.
