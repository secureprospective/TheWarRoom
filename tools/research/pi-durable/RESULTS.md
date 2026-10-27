# Slices 28–29 evidence

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

## Claude-OS observed result — slice 28, 2026-10-24/25

Host: Claude-OS VM, Debian 13 (trixie), kernel 6.12.107, Node v24.22.1 (private copy
inside the workspace), flock from util-linux. Candidate `a4bb7a9a` (PR #155's branch
head; its experiment files hash identically to the fingerprints above). Disposable copy
`/tmp/twr-durable-*`, credential-free, made with the README runbook. Run by ClaudeBox;
recorded and checked by Tom on 2026-10-25 from the copy's own log and evidence files.

| Command | Observed result |
|---|---|
| `corepack pnpm test` | 3/3 pass; the two drills' evidence: delay 3000 ms and 0 ms, both SIGKILL, task 8 before and after, `['prepared','finished']`, PASS. |
| `node drill.mjs` | PASS at `2026-10-24T23:43:11Z`, delay 3000 ms, SIGKILL, task 8 before and after, `['prepared','finished']`. |
| `node drill.mjs --overnight > overnight.log` | **PASS.** Started `2026-10-24T23:43:22.899Z`, delay 28,800,000 ms (8 h). Worker killed with SIGKILL at the committed sleeping checkpoint; status before restart: task 8, `sleeping` until `2026-10-25T07:43:23.368Z`, checkpoint `finish`, completed `['prepared']`. The restarted worker entered `finish` once and settled `completed` ("checklist ready for human review") at `2026-10-25T07:43:23.477Z`: 109 ms after the persisted wake, 8.00003 h after admission, well inside the 12 h expiry. After: same task 8, `['prepared','finished']`; a further start changed nothing. |

`overnight.log` holds exactly the workspace line and
`{"result":"PASS","evidence":".../.runs/goal-Fkc3KN/evidence.json"}`. The evidence is
committed as [`evidence/claude-os-overnight-2026-10-24.json`](evidence/claude-os-overnight-2026-10-24.json);
the only edit is the absolute workspace path, replaced by `"run": ".runs/goal-Fkc3KN"`.
It holds no tokens, keys or profile paths.

What this proves: a durable sleep committed before a SIGKILL was honoured once, by a
fresh process, eight hours later, on Claude-OS. What it does not: the restart was
immediate (the new process then waited eight hours), so it is not an eight-hour outage,
a whole-host restart or a power-loss test.

## Claude-OS observed result — slice 29 repair demo, 2026-10-25

Same host and Node. Candidate: this PR's first commit `2dc6975c`, copied with
`git archive` into a fresh `/tmp/twr-demo29-*`. Tom ran it over `ssh claudeos`,
command line only, no desktop.

| Command | Observed result |
|---|---|
| `corepack pnpm install --frozen-lockfile --ignore-scripts` | Exit 0. |
| `corepack pnpm test` | Exit 0, 4/4 pass (the three slice 28 tests plus the repair demo). |
| `node demo.mjs` | Exit 0, PASS, `.runs/demo-CArIo6`. Failing test reproduced for the real reason (`actual: 'legacy-nfl war room'`, `expected: 'legacy-nfl-war-room'`, exit 1). Worker SIGKILLed inside `fix`; status before restart: task 8, checkpoint `fix`, completed `['reproduced']`. The restart finished `['reproduced','fixed','verified','proposed']` on the same task; `test-runs.log` reads `fail, pass`, so the failing run was not repeated. `main` stayed at `b097d2f`; branch `durable/fix-slug` is one commit ahead (`a2905bb`); no remote; clean tree; `outbox/fix.patch` written. A third start changed nothing. |
| `node demo.mjs --stop demo-CArIo6` | Exit 0; outcome unchanged `completed` ("branch durable/fix-slug and outbox/fix.patch ready for human review"). |

Evidence, including the patch text: [`evidence/claude-os-repair-demo-2026-10-25.json`](evidence/claude-os-repair-demo-2026-10-25.json).
Its `errors` fields hold only Node's `--allow-child-process` security warning.
Nothing was merged: merging is the human's step. No process was left running; deleting
the workspace (`/tmp/twr-demo29-*`, about 384 MB) waits on a human confirmation.

Reviewer mutant (local, not committed): making `propose` commit the fix on `main`
instead of a new branch fails the demo with `the goal must never write main`.

Unproven: whole-host restart or power loss, model-written fixes (the fix is a fixed
patch), real pull requests and fleet dispatch. See the README's follow-up assessment.

## Plan correction / debt

No binding decision changed. The probe intentionally uses Pi Durable's custom-task
API without a paid model/provider, proving persistence independently of coding
behavior. Slices 28 and 29 now have observed Claude-OS results, above.

Debt left: none in runtime or installer. The README's follow-ups are proposals, not
open work.
