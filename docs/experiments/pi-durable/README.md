# Pi Durable: bounded overnight goal (slice 28)

**Local process-crash drill passes. Claude-OS and eight-hour execution are unproven.**
This is an experiment, not a league feature or an unattended coding agent.
B acceptance is supplied by the dispatch brief, not established by this experiment.
ClaudeBox alone runs the live gate; the builder does not access Claude-OS.

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
`/usr/bin/flock` available. Do not install services or change the live app. Copy
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

Ctrl-C (or SIGTERM to the driver) kills its current worker and prevents the driver
from starting another. Once the driver has exited, optionally make cancellation
permanent with the printed run directory's basename:

```sh
node drill.mjs --stop goal-REPLACE_WITH_PRINTED_NAME
```

Expect terminal `aborted` (or unchanged terminal `completed` if already finished).
If another writer is alive, stop refuses rather than opening competing storage.
Do not remove its lock file to bypass this. There is no background schedule to
uninstall. After all commands have exited and evidence is retained, delete the
**disposable copy** using the enclosing temporary directory recorded above. Never
delete the repository checkout or app state. Removing the copy also removes the
experiment dependencies, caches and all goals. No live app rollback is necessary.

### Record the actual live result

ClaudeBox adds the candidate SHA, Claude-OS OS/Node versions, exact commands,
exit codes, observed signal, same before/after task ID, completed list, cancellation
and writer-denial results, evidence location, reviewer and date to `RESULTS.md`.
Record short and overnight runs separately. Failure stays failure; no live acceptance
until observed on that candidate. No secrets or private host paths in committed evidence.

## Review disposition and remaining scope

Finding: source-only restart confidence cannot establish durable work.
Proposal: the smallest credential-free Pi task plus real crash/reopen assertions.
Preserve: app/runtime/deployment code and human merge authority remain untouched.
Evidence: local exercised results in `RESULTS.md`; Claude-OS unknown.
Risk: experimental upstream API, process crash only, no external tools.
Impact: zero changed runtime lines; all prototype, test, lock and documentation files
are confined to this directory. This is not a production debloat claim.

Debt left: none in app/runtime. Live Claude-OS/overnight evidence is the explicitly
assigned ClaudeBox acceptance gate, not an agent-reported pass. Slice 29 owns the
failing-test → fix → PR demonstration and CLI/fleet assessment; no placeholder
runtime integration or autonomous merge path is left behind.
