# HQ My moves — source mismatch (2026-11-17)

Status: blocked source trace, not an implemented hosted history feature. No runtime server inspection.

## What exists

| Responsibility | Source-traced path / contract |
| --- | --- |
| Legacy My moves | `frontend/src/app/shell/MyMoves.tsx:26` loads receipts through `executor.loadMoves`; `frontend/src/app/commands/sessionMoves.ts:19` calls `provider.moves(franchiseId)`; `frontend/src/app/data/provider.ts:256` reaches the desktop `TargetMoves` binding. These are envelopes, including private plans and lifecycle audits, not a public season ledger. |
| Desktop binding | `internal/app/mflref/planwatch/move_actions.go:229` (`TargetMoves`) → `internal/composition/platform_host.go:64` → `internal/composition/snapshot_app.go:102` (`listMoves`) → local moves store. |
| Append-only source actually used | `internal/store/mflref/moves/store.go:20` places the store in `thewarroom.db`; `:78` reads `move_audit`, `:110` lists envelope receipts. `internal/db/main_schema.go:70-96` defines `move_envelopes` / `move_audit` and their append-only triggers. |
| HQ season ledger | `frontend/src/app/shell/HQMoves.tsx:47-52` renders an empty body and explicitly says the reads do not provide a complete season-scoped ledger. There is no existing populated desktop season-ledger implementation here to expose unchanged. |
| Hosted lifecycle | `internal/leagueservice/native_owner_api.go:172` (`TargetNativeMoves`) reads native commissioner records and calls `market.NativeMovesReading`. It is not a reader over the desktop envelope audit tables. `internal/commissioner/market/moves_landed.go:32` projects selected active landed operations, not the append-only full history. |
| Server load | `cmd/twr-server/league_load.go:31` accepts a desktop **history.db** mirror. `:76-79` supplies `leagueload.Mirror` / `ReplaySource`; `internal/leagueload/load.go:49` reconstructs native state through held MFL responses. This path does not load the separate desktop **thewarroom.db** envelope/audit store. |

## Important distinction

The server **does hold a native append-only command ledger**. `internal/commissioner/ledger.go:158`
(`Reading`) already produces audience-filtered safe descriptions; hosted ledger service methods
exist in `internal/leagueservice/commissioner_api.go:23-57`. It would be incorrect to report that
all hosted history is missing. `FranchiseHistoryReading` in
`internal/commissioner/history_link.go:127` instead returns history-link records/titles, not moves.

What is missing from the source-traced hosted loading path is the desktop My moves envelope audit
history. Also missing is the assumed existing complete season-ledger reader used by desktop HQ.
No live database was accessed, so this establishes the supported loading contract, not a count of
rows on a particular deployment. Neither local/private audits nor native pending move payloads may
be relabelled as a public completed-moves ledger.

## Access rule and next move

`docs/ui/data-catalog/DATA_FLOW_MAP.md:17`:

> **LEDGER** public record | every member (filtered by Audience) | commands only, via ledger | Append-only. Actor + "via AI" stamped. Commish corrections are new rows, never edits.

D22 at `:95` names `store/history` as the franchise-history owner. Therefore a future native
history read should compose the existing audience-filtered ledger, preserve corrections and actor
stamps, and season-page franchise-relevant rows. Another admitted owner is **not automatically an
outsider** under this rule; tests must follow Audience, not invent owner-only LEDGER privacy.
`moves.read` remains owner-scoped and unchanged.

First resolve the brief's source mismatch: confirm that “full history” means native committed
franchise ledger rows rather than preservation of desktop envelope lifecycle audits. If native
history is intended, use the existing server ledger; specify franchise association and season
semantics (including transfers, corrections and rollovers), then implement one catalogued read and
hosted HQ rendering. If desktop audit parity is required, its validated acquisition/identity mapping
is missing from the server loading contract and needs a separate approved scope. No tables,
migrations, import changes, commands or UI changes are introduced by this draft.

## Slop and verification

Found, not changed: HQ's season-ledger heading promises every move while its body is always empty;
the inventory's `moves.read` row is not proof of a complete desktop season-history implementation.
Production footprint delta: 0 lines / 0 characters. Only documentation changes.

Initial `pnpm install --frozen-lockfile && pnpm build` passed. Vite reported existing ineffective
dynamic-import and >500 kB chunk warnings. No changed Go packages or functional changes require
new race/authorization/paging/render tests; those outcomes remain unimplemented, not verified.
`pnpm test` passed (114 files, 881 tests); `make lint VERSION=dev COMMIT=pulse-hqmoves
BUILD_DATE=pulse-hqmoves` passed (0 issues). The four existing >600-line Go file reports concern
untouched files and are not reasons to split responsibilities in this documentation-only change.
