# APP-MERGE-6 — partial; remaining gates blocked

Bee, 2026-11-02. Branch `build/INTEGRATE-1102`, starting at `33fa4a5d0`.
Local work only: no git remote contacted, no push, no full Go repository suite.
`APP-MERGE-5.md` was absent from this checkout; the brief and current source supplied context.

## Item 1 — extraction done; boundarycheck BLOCKED

- `68d39d77f`: moved schedule selection and current/followed/played lineup orchestration to
  `internal/app/mflref/migration/schedule_sync.go`. This is an explicitly fenced reference
  package, indexed in placement and the system map; no engine/model behavior changed.
- Composition supplies stores, authorization, held-player lookup, date acquisition, serialized
  scoring acquisition and worker lifetime. `syncLeagueSchedules` is one-line wiring.
- Current week remains inline. The first played-week worker waits five minutes, then receives
  a ten-minute timeout. Transaction import remains inside that worker, before played lineups.
  The existing app-wide atomic guards and nil-worker inline path remain. Cutover is unchanged
  and its schedule-follow clearing regression passes.
- Reviewed CONTROL-1 source changes: `command_bindings.go` adds the real sign-in-facts binding;
  `commissioner_host.go` wires local-seat/facts callbacks. No new policy in either change.
  Updated their source fingerprints; removed the extracted schedule function allowance.
- New tests observe worker exclusion, first-launch cancellation, guard release and timing.
- `a9ad04e29`: test-only clock repair. The schedule/scoring fixture pinned loads to Nov 1,
  while freshness uses wall time. Crossing its 20-hour window at 08:00 made the final targeted
  batch fail with four reads rather than two. Feed-load timestamps now use wall time while
  league chronology stays pinned. The same batch then passed; production caching is unchanged.

`go run ./tools/boundarycheck` exits **1**, after composition/source checks, at the unchanged path:

```text
internal/leagueservice -> internal/appvalue -> internal/modelrun -> internal/store/history
 -> internal/ingestion/contracts -> internal/ingestion -> internal/mfl
```

This is existing native-to-reference transitive coupling, not the extracted service. Removing
native roots, marking shared history as a fence, or adding an exemption would conceal it.
No boundary guard was weakened. Broader acquisition/parsing separation needs a follow-up.

## Item 2 — scoped frontend repairs done; src/app green BLOCKED

`1f00dd7db`:
- Replaced Control Room raw controls with shared `Act` controls and named
  `commissioner.surface` handlers. Tables expose real keyboard-operable buttons rather than
  clickable rows. Search and command inputs submit native forms through the shared command.
- Added an unstyled `surface` Act variant to retain the locked room CSS, and preserved rail
  `aria-current="page"`. Added tests for shared-control semantics and search/command dispatch.
- Wrapped/formatted the requested room and commissioner source/test files below 111 characters.
- Confirmed `TargetSignInFacts` is used by `control/room/roomReads.ts` for People/Server facts.
  The transport test now names it and verifies all **53** methods.

`pnpm vitest run src/app`: **179 files passed, 1 failed; 1829 tests passed, 2 failed**.
All room and commissioner API cases pass. The only remaining failures are the unchanged guards:
1. Raw controls in `shell/HomeBrief.tsx` and `shell/homeTiles.tsx`.
2. **86** long lines across **10** untouched shell files: HomeBrief, homeTiles, homeBrief.test,
   HQPanels, PulseNow, homeReads, leaguePicker.interaction.test, localWorkflow.test,
   pulseLive.test and pulseReads.test.

Those failures were already present in the initial three-failure reproduction, beyond the
room/commissioner files named by the brief. The tests and keyboard allowlist remain strict;
no exceptions were added. Shell behavior was not rewritten as incidental scope expansion.

## Item 3 — checks executed; acceptance still BLOCKED by the two gates above

Environment for commits/build/Go checks: `GOTOOLCHAIN=go1.26.10`, worktree-local lint cache,
and the specified Go/pnpm PATH entries. Heavy jobs were serialized; capacity was checked.
Every commit used `git commit -F`, with hooks enabled.

| Check | Evidence |
|---|---|
| Go discovery | `go test -list .` on migration/composition before batches; 605 discovered test declarations initially |
| Initial extracted composition candidate | `go test -race -timeout 30m -json ./internal/composition`: **846 PASS** tests/subtests, no failures/skips, 805.963s |
| Final migration service | `go test -race -json ./internal/app/mflref/migration`: **2 PASS** |
| Final affected composition cases | Discovery then `go test -race -timeout 30m -run 'Test(LaunchAutomatically\|ScheduleFollowingBackfillAndCutover\|ScheduleRefreshFillsOnlyMissingNFLDates)' -json ./internal/composition`: **4 PASS**, 40.500s |
| Frontend | `pnpm vitest run src/app`: **1829 PASS / 2 FAIL**, 12.07s |
| Typecheck | `cd frontend && pnpm tsc --noEmit`: exit **0** |
| Desktop build | Plain `make build`: exit **0**, built `build/bin/thewarroom`, 22.945s |
| Boundary | Exit **1**, native transitive path above |
| Commit gates | Go commits: golangci-lint, gitleaks and ifaceguard pass; frontend commit: gitleaks pass, Go-only hooks correctly skipped |
| UI detector | Mechanical detector on room/Act returned `[]`; not a browser acceptance test |

The 846-PASS composition run preceded the contained-context lint repair (worker context became
an injected getter) and fixture-clock repair. The final four affected cases plus two service
cases cover those changes; the whole composition package was not repeated. Do not treat the
846 as a full final-tree rerun. No tests of unrelated packages were repeated.

Wails emitted missing-model-alias diagnostics while generating bindings, but exited successfully;
tracked generated bindings were unchanged. Build success is not live UI proof.

## Impact

Physical lines and Unicode code points including LF, compared with `33fa4a5d0`, over the entire
changed handwritten code set; generated outputs/docs excluded, tests separate. No reduction claim.

| Set | Files | Lines before → after | Code points before → after |
|---|---:|---:|---:|
| Go production | 2 | 272 → 323 | 11123 → 13109 |
| TypeScript production | 23 | 2094 → 4636 | 122820 → 156818 |
| Tests | 15 | 1028 → 1366 | 48374 → 53957 |

Frontend growth is chiefly explicit shared-command wiring and formatting formerly dense lines.

## Debt left / next move

- Resolve native history/acquisition transitive MFL coupling without relaxing the fence.
- Give the HomeBrief/homeTiles shared-control conversion and remaining shell line wrapping their
  own slice; then rerun the two strict controls guards and src/app.
- Independent review and disposable-profile UI acceptance remain unverified: navigate every room
  branch by keyboard, submit search/commands, open/cancel rule and action forms, verify rail current
  state and unchanged appearance. Observe current-week publication before the delayed history job;
  cancel the app during its initial wait and verify clean worker shutdown.
- Carry forward APP-MERGE-4's unresolved operator rulings and transaction-follow activation debt.
