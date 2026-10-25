# Slice 28 evidence

## Local observed result — 2026-10-24

Runner: Bee, Beelink / LMDE 7, Node v24.22.1, pnpm 10.34.3 through Corepack.
Repository baseline: `746e3b74`; the PR commit identifies the candidate.
No Claude-OS connection, runtime changes, MFL calls or credentials used.

| Command / experiment | Observed result |
|---|---|
| `corepack pnpm install --frozen-lockfile --ignore-scripts` (experiment directory, private COREPACK_HOME) | Exit 0, pinned dependencies installed without lifecycle scripts. |
| `corepack pnpm test` | Exit 0, 3/3 tests pass, no skips. Real worker SIGKILL in sleeping and in-flight phases; same task ID; one committed prepare/finish; terminal restart unchanged. Concurrent writer exit 1; persisted abort survives restart; invalid delay refuses with existing DB bytes unchanged. |
| `node drill.mjs` | Exit 0, PASS; evidence `.runs/goal-VIsaQa/evidence.json`. Started `2026-10-24T23:31:57.411Z`, delay 3000 ms, killed by SIGKILL, resumed exit 0. Before: task 8, `sleeping`, `['prepared']`; after: task 8, terminal `completed`, `['prepared','finished']`. |
| Mutation: omit the final `finished` document write, run sleeping SIGKILL test | Exit 1 at document comparison: actual `['prepared']`, expected `['prepared','finished']`. Restored original, reran all 3 tests and standalone drill successfully. Mutant not committed. |
| `cd frontend && pnpm install --frozen-lockfile && pnpm build` | Exit 0; prerequisite embedded assets built; no tracked frontend files changed. Existing large-chunk advisory only. |
| `go test ./...` with Go 1.26.10 | Exit 0, all packages pass. Live opt-in source tests not enabled. Log `.runs/go-test.log`. No repeated full Go run. |

Executable/dependency SHA-256 fingerprints at the passing local run (paths relative
to this directory):

```text
29bcf6f45630d6d2beaa1e6f47ecd56cfa057229af796504457b3dac9e43cc2a  worker.mjs
0e293d05a979fbc4b6672d9e1a7287ecbd61c614655c587cba2f1a6ce4ea2e70  drill.mjs
8f5776d67b99916623d7eb3d6c6d010aa3e8b7bc280854cd2f4a6716b7d33de4  prototype.test.mjs
1ed3d4b4a3e1d37b2624e2c20e3f3ea0cddb58810f55643aaeb0def5bc7100ce  package.json
dc5b44d321cc79f927cd515b170112d78ae31c7ef8e8e8f9098ad2b6abae534a  pnpm-lock.yaml
```

Local `.runs` files are ignored; this is the retained observation, not a claim that
ignored paths travel with git. ClaudeBox retains its own raw live evidence and
records the tested candidate SHA. No runtime/app production files changed.

## Source-traced, not separately exercised

Twelve-hour expiry branch; Node permission read/write scope; no provider or agent
tools registered; no runtime imports or Git/merge/deploy path. Process-crash
recovery is exercised, not merely source-traced. This fixed goal makes no claim
about idempotency of external tools or crash-safe network effects.

## Claude-OS acceptance — UNRUN

The builder is explicitly not authorized to run it. ClaudeBox executes the README
runbook in a disposable, credential-free copy and records its actual candidate,
commands, exit codes, task IDs, observed result and reviewer here. The eight-hour
mode is also unrun; seconds-long local success is not an overnight pass.

Unproven: Claude-OS execution, eight-hour wake, host reboot/power loss, model calls,
repair-to-PR and fleet dispatch. The latter two belong to slice 29; host/power loss
are not claimed by this process-crash experiment. No app integration is proposed.

## Plan correction / debt

No binding decision changed. The probe intentionally uses Pi Durable's custom-task
API without a paid model/provider, proving persistence independently of coding
behavior. The plan's live observed-result requirement remains open for ClaudeBox,
as assigned by the dispatch brief; this is not accepted live evidence.

Debt left: none in runtime or installer. `README.md`/this file retain the explicit
live acceptance boundary; slice 29 owns the already-planned repair/test/PR follow-up.
