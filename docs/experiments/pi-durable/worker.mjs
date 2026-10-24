import { lstat, realpath } from 'node:fs/promises';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { BACKGROUND_CONTEXT as context } from '@earendil-works/chord/context';
import { createModels } from '@earendil-works/pi-ai/models';
import { Harness, createRegistry, defineDoc, defineTask } from '@earendil-works/pi-durable';
import { openNodeSqliteStorage } from '@earendil-works/pi-durable/storage/sqlite/node';

process.umask(0o077);
const base = dirname(fileURLToPath(import.meta.url));
const [directory, action = 'run', delayText = '1000'] = process.argv.slice(2);
const workspace = resolve(directory ?? '.');
if (dirname(workspace) !== resolve(base, '.runs') ||
    !(await lstat(workspace)).isDirectory() || await realpath(workspace) !== workspace) {
  throw new Error('Use a fresh directory directly inside .runs, without symlinks');
}
if (!['run', 'status', 'stop'].includes(action)) throw new Error('Unknown action');
const delay = Number(delayText);
if (!Number.isSafeInteger(delay) || delay < 0 || delay > 28_800_000) {
  throw new Error('Wake delay must be 0 through 28800000 milliseconds');
}

const Goal = defineDoc({
  kind: 'experiment.goal', version: 1, scope: 'conversation',
  history: 'latest', fork: 'initial', checkpointWhen: () => true,
  initial: () => ({ taskId: null, completed: [] }),
});
const BoundedGoal = defineTask({
  name: 'experiment.checklist', version: 1,
  initial: () => ({ phase: 'prepare' }),
  phases: {
    prepare: async (task, runtime, ctx) => {
      await runtime.commit(async (tx) => {
        (await tx.doc(Goal, runtime.conversationId)).completed.push('prepared');
        return { status: 'sleeping', until: task.input.wakeAt, checkpoint: { phase: 'finish' } };
      }, ctx);
    },
    finish: async (task, runtime, ctx) => {
      console.log(JSON.stringify({ event: 'finish-entered', taskId: task.id }));
      // The drill kills this phase before its commit. It must rerun safely.
      if (task.input.pause) await new Promise((done) => setTimeout(done, 1000));
      await runtime.commit(async (tx) => {
        if (Date.now() >= task.input.expiresAt) {
          return { status: 'terminal', outcome: { status: 'failed', error: { message: 'Goal expired' } } };
        }
        (await tx.doc(Goal, runtime.conversationId)).completed.push('finished');
        return { status: 'terminal', outcome: { status: 'completed', result: 'checklist ready for human review' } };
      }, ctx);
    },
  },
  abort: (_task, runtime, ctx) => runtime.commit(() => ({
    status: 'terminal', outcome: { status: 'aborted' },
  }), ctx),
});
const registry = createRegistry();
registry.install({ name: 'bounded-experiment', tasks: [BoundedGoal] });
// No providers, shell, file tools, credentials, Git client or execution environment.
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
        if (!doc.taskId) doc.taskId = await tx.createTask(BoundedGoal, {
          wakeAt: Date.now() + delay, expiresAt: Date.now() + 43_200_000,
          pause: delay === 0,
        }, { ownership: { kind: 'conversation', conversationId: root.id } });
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
