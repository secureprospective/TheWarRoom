import { execFileSync, spawnSync } from 'node:child_process';
import { appendFile, lstat, mkdir, realpath, writeFile } from 'node:fs/promises';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { BACKGROUND_CONTEXT as context } from '@earendil-works/chord/context';
import { createModels } from '@earendil-works/pi-ai/models';
import { Harness, createRegistry, defineDoc, defineTask } from '@earendil-works/pi-durable';
import { openNodeSqliteStorage } from '@earendil-works/pi-durable/storage/sqlite/node';

// Slice 29: a durable failing-test -> fix -> review-branch goal in a throwaway repo.
// The repair is a fixed patch, not model output: this proves the pipeline survives
// a crash and stops at human review, not that an agent can find the fix.
process.umask(0o077);
const base = dirname(fileURLToPath(import.meta.url));
const [directory, action = 'run'] = process.argv.slice(2);
const workspace = resolve(directory ?? '.');
if (dirname(workspace) !== resolve(base, '.runs') ||
    !(await lstat(workspace)).isDirectory() || await realpath(workspace) !== workspace) {
  throw new Error('Use a fresh directory directly inside .runs, without symlinks');
}
if (!['run', 'status', 'stop'].includes(action)) throw new Error('Unknown action');
const repo = resolve(workspace, 'repo');
const BRANCH = 'durable/fix-slug';
const FIXED = "export const slug = (text) => text.trim().toLowerCase().split(/\\s+/).join('-');\n";

const git = (...args) => execFileSync('git', ['-c', 'user.name=durable-demo',
  '-c', 'user.email=durable-demo@invalid', ...args], { cwd: repo, encoding: 'utf8' }).trim();
// The test child may read only the scratch repo; it gets no write or spawn permission.
async function runTests() {
  const run = spawnSync(process.execPath, ['--permission', `--allow-fs-read=${repo}`,
    '--test-isolation=none', '--test', 'slug.test.mjs'], { cwd: repo, encoding: 'utf8', timeout: 30_000 });
  await appendFile(resolve(workspace, 'test-runs.log'), `${run.status === 0 ? 'pass' : 'fail'}\n`);
  return { exit: run.status, tail: (run.stdout + run.stderr).trim().split('\n').slice(-6) };
}

const Goal = defineDoc({
  kind: 'experiment.repair', version: 1, scope: 'conversation',
  history: 'latest', fork: 'initial', checkpointWhen: () => true,
  initial: () => ({ taskId: null, completed: [], failing: null }),
});
const step = (name, next, extra = {}) => async (runtime, ctx) => runtime.commit(async (tx) => {
  const doc = await tx.doc(Goal, runtime.conversationId);
  doc.completed.push(name);
  Object.assign(doc, extra);
  return next;
}, ctx);
const RepairGoal = defineTask({
  name: 'experiment.repair', version: 1,
  initial: () => ({ phase: 'reproduce' }),
  phases: {
    reproduce: async (_task, runtime, ctx) => {
      const failing = await runTests();
      if (failing.exit === 0) {
        return runtime.commit(() => ({ status: 'terminal',
          outcome: { status: 'failed', error: { message: 'Nothing to repair: tests already pass' } } }), ctx);
      }
      await step('reproduced', { status: 'running', checkpoint: { phase: 'fix' } }, { failing })(runtime, ctx);
    },
    fix: async (task, runtime, ctx) => {
      console.log(JSON.stringify({ event: 'fix-entered', taskId: task.id }));
      // The demo kills this phase here. Rewriting the same file on replay is idempotent.
      if (task.input.pause) await new Promise((done) => setTimeout(done, 1000));
      await writeFile(resolve(repo, 'slug.mjs'), FIXED);
      await step('fixed', { status: 'running', checkpoint: { phase: 'verify' } })(runtime, ctx);
    },
    verify: async (_task, runtime, ctx) => {
      const passing = await runTests();
      if (passing.exit !== 0) {
        return runtime.commit(() => ({ status: 'terminal',
          outcome: { status: 'failed', error: { message: 'Fix did not make the test pass' } } }), ctx);
      }
      await step('verified', { status: 'running', checkpoint: { phase: 'propose' } })(runtime, ctx);
    },
    propose: async (_task, runtime, ctx) => {
      // A local branch plus a patch file is the whole "PR". No remote exists, nothing is
      // pushed, and main is never written: a human reviews and merges.
      if (spawnSync('git', ['rev-parse', '--verify', '--quiet', `refs/heads/${BRANCH}`], { cwd: repo }).status !== 0) {
        git('switch', '--quiet', '-c', BRANCH);
        git('commit', '--quiet', '-am', 'fix: slug joins every word, not just the first two');
      }
      git('switch', '--quiet', 'main');
      await mkdir(resolve(workspace, 'outbox'), { recursive: true });
      await writeFile(resolve(workspace, 'outbox/fix.patch'), git('format-patch', '-1', '--stdout', BRANCH) + '\n');
      await runtime.commit(async (tx) => {
        (await tx.doc(Goal, runtime.conversationId)).completed.push('proposed');
        return { status: 'terminal', outcome: { status: 'completed',
          result: `branch ${BRANCH} and outbox/fix.patch ready for human review` } };
      }, ctx);
    },
  },
  abort: (_task, runtime, ctx) => runtime.commit(() => ({
    status: 'terminal', outcome: { status: 'aborted' },
  }), ctx),
});
const registry = createRegistry();
registry.install({ name: 'repair-experiment', tasks: [RepairGoal] });
// No providers, credentials, remote or execution environment; git and node are the only children.
const harness = await Harness.open(await openNodeSqliteStorage(resolve(workspace, 'session.db')),
  { models: createModels(), registry, readOnly: action === 'status' }, context);
try {
  const root = await harness.root(context);
  let goal = await harness.snapshot(Goal, root.id, context);
  if (action === 'status') {
    console.log(JSON.stringify({ event: 'status', goal,
      task: goal?.taskId ? await harness.getTask(goal.taskId, context) : null }));
  } else {
    if (!goal?.taskId && action === 'run') {
      await root.commit(async (tx) => {
        const doc = await tx.doc(Goal, root.id);
        if (!doc.taskId) doc.taskId = await tx.createTask(RepairGoal, { pause: true },
          { ownership: { kind: 'conversation', conversationId: root.id } });
      }, context);
      goal = await harness.snapshot(Goal, root.id, context);
    }
    if (!goal?.taskId) throw new Error('No goal to stop');
    console.log(JSON.stringify({ event: 'admitted', taskId: goal.taskId }));
    if (action === 'stop') await harness.abortTask(goal.taskId, context);
    harness.resume();
    const receipt = await harness.waitForTask(goal.taskId, context);
    console.log(JSON.stringify({ event: 'settled', receipt,
      goal: await harness.snapshot(Goal, root.id, context) }));
  }
} finally {
  await harness.close(context);
}
