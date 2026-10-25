# Pi Durable: bounded overnight goal (slices 28–29)

**Observed on Claude-OS: the eight-hour goal survived a SIGKILL and finished once
(slice 28), and a repair goal killed mid-fix resumed to a review branch (slice 29).**
Results and evidence: [`RESULTS.md`](RESULTS.md). This is an experiment, not a league
feature or an unattended coding agent. B acceptance is supplied by the dispatch brief,
not established by this experiment.

## Goal and boundaries

Use the real `@earendil-works/pi-durable` 1.4.0 harness to durably prepare a synthetic
checklist, sleep, then finish it for human review. The default drill takes seconds;
`--overnight` persists an eight-hour wake. The goal has two commits, one task,
a twelve-hour expiry checked before finishing, and no model spend. Reopening a
finished or aborted goal never creates a replacement. An expired goal fails rather
than pretending to finish. This fixed goal isolates restart behavior before the
repair/test/PR demonstration in slice 29; it does not claim to demonstrate that work.

| Responsibility | Contract / effects / failure |
|---|---|
| `drill.mjs` | Creates a private disposable `.runs/goal-*` directory; launches, kills, reopens and checks the worker. A bounded watchdog kills a stuck worker. Captures evidence only after assertions pass. |
| `worker.mjs` | Checks workspace and action/delay before opening storage. Atomic admission stores the task ID in `experiment.goal`; subsequent starts reuse it. Only the fixed custom task is scheduled. |
| `prepare` phase | Commits `prepared` and the sleeping `finish` checkpoint together. No external side effect. |
| `finish` phase | Checks expiry, commits `finished` and terminal result together. A crash before commit replays the phase, not previously committed preparation. |
| Pi SQLite storage | Owns transcript, task and document state in `session.db`/WAL/SHM. Real process exit and reopen, not an in-memory mock. |
| Read-only status / stop | Status never schedules work. Stop takes the writer lock, persists abort and waits for terminal state. |

The driver launches workers with an empty inherited environment except a synthetic
HOME and Node-only PATH. Node permissions restrict reads to this experiment and
writes to the selected run directory; no child-process permission is granted.
`flock --nonblock --no-fork` holds one writer per workspace and releases on SIGKILL;
status is read-only. All created files use umask 077; the drill checks group/other
permission bits. Use the driver, not a direct worker invocation (which bypasses the
writer lock). No shell/file tools, provider, execution environment, Git remote,
GitHub token, app credentials, runtime imports, merge or deploy action is supplied.
No API/MFL calls are made. There is no cron job, daemon or restart supervisor installed.

This is not a hostile-code sandbox: Node permissions do not substitute for OS/network
isolation or protect against malicious dependencies. Run only in a disposable,
credential-free workspace. No claim of power-loss durability: upstream SQLite uses
WAL with synchronous NORMAL. Exactly-once here means **atomic local document effects**,
not exactly-once arbitrary tools or network side effects.

## Dependency identity

The npm package metadata identifies `https://github.com/earendil-works/pi`, directory
`packages/durable`; 1.4.0's installed README, declarations and task/storage APIs were
inspected. `pi-ai` and `chord` are direct imports required by its harness, pinned at
the matching 1.4.0 release. The separate `pnpm-lock.yaml` pins transitive resolutions
and integrity hashes. No root or frontend dependency changes. Install scripts are
disabled. No model provider is registered despite the upstream SDK dependencies.

Reference: <https://github.com/earendil-works/pi/tree/main/packages/durable>.
The installed 1.4.0 README/declarations, not mutable main, govern this candidate.
The package explicitly describes its API as experimental. This prototype does not
assert compatibility with a fleet Pi CLI or authorize adding it to app runtime.

## ClaudeBox live runbook

Run on Claude-OS as a normal user, with Node **24.22.1 or newer**, Corepack and
`/usr/bin/flock` available (and `git` for the slice 29 demo). Do not install services or change the live app. Copy
**only this experiment's tracked files** from the reviewed candidate into a new
private disposable directory. Do not copy `.git`, app databases, `.env`, SSH keys,
GitHub credentials, Pi auth files, `node_modules` or `.runs`. Do not use the live app
workspace or a personal Pi HOME. The commands below are for ClaudeBox, not league
operators; no commissioner setup steps are added.

From the reviewed repository checkout (record `git rev-parse HEAD` first):

```sh
umask 077
candidate=$(git rev-parse HEAD)
workspace=$(mktemp -d -t twr-durable-XXXXXXXX)
chmod 700 "$workspace"
git archive "$candidate" docs/experiments/pi-durable | tar -x -C "$workspace"
cd "$workspace/docs/experiments/pi-durable"
printf '%s\n' "$candidate" > candidate.txt
# Keep even package-manager caches and its HOME inside the disposable copy.
mkdir -m 700 .tool-home
export HOME="$PWD/.tool-home" COREPACK_HOME="$PWD/.tool-home/corepack"
corepack pnpm install --frozen-lockfile --ignore-scripts
corepack pnpm test
node drill.mjs
```

The tests kill a real worker (SIGKILL), reopen the same SQLite storage and check:

1. Sleeping checkpoint survives; resumed task ID is identical; `prepared, finished`
   appear exactly once; another restart changes nothing.
2. SIGKILL after entering `finish` but before its commit replays that phase safely.
3. A second writer is refused; abort survives restart; delay above eight hours is
   refused before opening storage and the existing DB bytes are unchanged.

`node drill.mjs` performs the sleeping-checkpoint drill and prints PASS plus the
path to `evidence.json`. It prints the run directory at startup, preserves exit
signal, before/after task records, output, timestamps and Node version. Any failed
assertion exits nonzero and **does not** write PASS evidence. Retain the failing
output and workspace for diagnosis; do not label a source review or compilation a pass.

For the overnight goal, from that same disposable copy:

```sh
node drill.mjs --overnight
```

This kills at the committed sleeping checkpoint, immediately restarts the worker,
and lets the persisted eight-hour wake complete (watchdog: wake delay + 15 seconds
per worker). It is not an eight-hour outage test. Keep the terminal/session alive;
no supervisor is installed. The fixed expiry is twelve hours after first admission.
A full host restart, power loss, model/tool resumption and network side-effect
idempotency are not established by either drill.

### Stop and delete

See [Stop and delete](#stop-and-delete-both-drills-and-the-demo) below; it covers both
drills and the slice 29 demo.

### Record the actual live result

ClaudeBox adds the candidate SHA, Claude-OS OS/Node versions, exact commands,
exit codes, observed signal, same before/after task ID, completed list, cancellation
and writer-denial results, evidence location, reviewer and date to `RESULTS.md`.
Record short and overnight runs separately. Failure stays failure; no live acceptance
until observed on that candidate. No secrets or private host paths in committed evidence.

## Slice 29: failing test → fix → review branch

`demo.mjs` builds a throwaway git repo under `.runs/demo-*/repo` with one helper and
one failing test, then runs `demo-worker.mjs`, a durable goal with four phases:

| Phase | What it does |
|---|---|
| `reproduce` | Runs the test; records the failure. If the test already passes, the goal fails rather than inventing work. |
| `fix` | Writes a fixed patch (idempotent on replay). The fix is canned, not model output: the demo tests the pipeline's durability and its stop at review, not repair skill. |
| `verify` | Reruns the test; a still-failing test fails the goal. |
| `propose` | Commits on a new local branch `durable/fix-slug`, writes `outbox/fix.patch`, switches back to `main`. That branch plus patch is the "PR". |

The driver kills the worker inside `fix`, restarts it, and checks: same task, each step
once, the failing test ran once (not replayed), `main` unchanged, one commit on the
branch, no remote, clean tree. A third start changes nothing.

Boundaries. The scratch repo has no remote, so nothing can be pushed; the goal never
writes `main` and has no merge step; there is no deploy. The worker gets
`--allow-child-process` so it can run `git` and `node --test`. Node permissions do not
reach those children, so their limits come from the empty environment (synthetic HOME,
`GIT_CONFIG_GLOBAL=/dev/null`, no system git config, no credential prompt) and the
disposable workspace. The test child itself runs with `--permission` and read access to
the scratch repo only.

Run it from the same disposable copy as above:

```sh
node demo.mjs
```

It prints PASS and the `evidence.json` path. To review and merge as the human would:

```sh
cd .runs/demo-NAME/repo
git log --oneline main..durable/fix-slug && git diff main durable/fix-slug
git merge --ff-only durable/fix-slug   # the human's decision; the goal never does this
```

## Stop and delete (both drills and the demo)

1. Stop: Ctrl-C (or SIGTERM to `drill.mjs`/`demo.mjs`) kills the current worker and
   prevents another start. Nothing restarts it: there is no cron job, service or supervisor.
2. Make cancellation permanent (optional): `node drill.mjs --stop goal-NAME` or
   `node demo.mjs --stop demo-NAME`. Expect `aborted`, or an unchanged `completed`.
   If a writer is still alive, stop is refused; do not delete `owner.lock` to force it.
3. Check nothing is left running: `pgrep -af 'drill.mjs|demo.mjs|worker.mjs'` prints nothing.
4. Copy out any evidence you want to keep, then delete the whole disposable copy:
   `rm -rf -- "$workspace"` (the `mktemp` directory from the runbook). That removes the
   dependencies, caches, goals and scratch repos. Never delete a repository checkout or
   app state; no app rollback is needed.

## Follow-up assessment (CLI and fleet)

Worth doing, in this order, each as its own decision:

1. **A goal runner for the fleet, outside the app.** The pattern holds: durable phases,
   a sleep that survives a process kill, and a stop at a branch. A `~/fleet` script could
   wrap a builder brief as a goal with this shape. It belongs in the fleet, not in
   TheWarRoom's runtime or its `twr` CLI; the app needs none of it.
2. **A real model in the `fix` phase.** The step that is still unproven. It needs a
   provider credential, a sandbox stronger than Node permissions (a container or VM user
   with no network beyond the model API), and a budget cap per goal. That is a panel
   question before it is a build.
3. **Pull requests stay human.** If the runner ever pushes, it pushes a branch to a fork
   or a throwaway remote with a token that cannot merge; merging stays a person's click.
4. **Host restart.** Only a process kill is proven. Surviving a VM restart needs a
   supervisor (a systemd user unit) and a test that restarts the VM; add it only if
   goals are meant to run unattended across restarts.

Not worth doing now: wiring Pi Durable into the app or the `twr` CLI, or replacing the
existing dispatch scripts. They already work for the current pace of slices.

## Review disposition and remaining scope

Finding: source-only restart confidence cannot establish durable work.
Proposal: the smallest credential-free Pi task plus real crash/reopen assertions.
Preserve: app/runtime/deployment code and human merge authority remain untouched.
Evidence: local and Claude-OS observed results in `RESULTS.md`, raw JSON in `evidence/`.
Risk: experimental upstream API, process crash only; the demo's only tools are local
`git` and `node --test` in a scratch repo.
Impact: zero changed runtime lines; all prototype, test, lock and documentation files
are confined to this directory. This is not a production debloat claim.

Debt left: none in app/runtime. The live gates for slices 28 and 29 are observed and
recorded. The follow-up assessment above lists proposals only; no runtime integration,
push path or autonomous merge path is left behind.
