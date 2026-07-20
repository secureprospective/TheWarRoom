# Deterministic Ground Truth — TheWarRoom full-codebase slim review (2026-07-20)

Run on branch `session/review-harvest` (tip `788e7c3` = main `ee891ef` + TWR-1 fix). These are
tool facts, not opinions — they are AUTHORITATIVE for what each tool covers, and every GLM 5.2 lead
is triaged against them. Raw outputs are the sibling `*.txt` files.

## Headline: the codebase is already static-analysis clean
| Tool | Scope | Result |
|---|---|---|
| `golangci-lint run ./...` (repo `.golangci.yml`, strict) | all Go | **0 issues** |
| `staticcheck ./...` | all Go | **0 issues** (incl. U1000 unused — authoritative "no unexported dead code") |
| `dupl -threshold 48` | all Go | 227 dup-block lines — **1 production dup**, rest is test boilerplate |
| `ts-prune` | frontend | 1 real dead export (`ping.ts`), rest generated-binding false positives |
| `deadcode ./...` | all Go | **not used** — OOM on the 2GB host AND unreliable for a Wails app (every exported `App` IPC method is called from JS, so Go whole-program reachability flags them as false-dead). `staticcheck U1000` is the authoritative unused signal here. |

The implication for the review: the easy/mechanical wins are already gone. The value is in what
static tools cannot see — semantic dead code, near-duplicate logic, orphan stubs, over-abstraction,
and standards judgment. That is exactly what the GLM 5.2 semantic pass targets.

## The one production duplicate (verified against source)
`internal/store/state/cap_relief.go:109-133` `loadCapRelief` ≡
`internal/store/state/state.go:372-396` `loadDeadCap` — structural twins: identical
`SELECT franchise_id, COALESCE(SUM(<col>),0) ... GROUP BY franchise_id` → `map[string]domain.Money`
shape, differing only in table (`cap_relief_ledger` / `dead_cap_ledger`), column
(`relief_cents` / `dead_cap_cents`), and error strings.
- **Lead (slimming, med value / low risk):** extract `sumLedgerByFranchise(ctx, table, col, label)`;
  table/column are compile-time constants so the interpolation is safe. Saves ~24 lines and removes
  a divergence risk between two ledger readers that must stay in lock-step.
- **Confirm:** the two functions have no behavioural difference beyond the three tokens above.

## Duplicate blocks that are ACCEPTABLE (not leads)
The remaining `dupl` hits are all `_test.go` table-test boilerplate — expected and idiomatic:
- `internal/engine/l4/{defense,offense}/*_test.go` — per-position scorer test scaffolds.
- `internal/harness/realrubric_test.go` — repeated rubric-case blocks.
- `internal/transactions/*_integration_test.go` — parallel integration harness setup.
- `internal/ingestion/*/fetcher_test.go` — parallel fetcher-test setup.
Flagging these would be noise; a shared test helper is optional style, not debt.

## Frontend dead code (verified)
`ts-prune`: `src/store/ping.ts:14 usePingStore` — dead (zero importers), consistent with the
GOAL-1 harvest note that `ping.ts` is dead code. The `wailsjs/go/models.ts` entries
(`engine`/`harness`/`params`/`rankings` "used in module") are generated Wails bindings — false
positives, do not touch.
