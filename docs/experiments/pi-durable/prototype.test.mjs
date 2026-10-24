import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { drill, fresh, launch, status } from './drill.mjs';

test('SIGKILL during durable sleep resumes the same bounded goal exactly once', async () => {
  await drill(3000);
});
test('SIGKILL inside a phase replays it without duplicating committed work', async () => {
  await drill(0);
});
test('stop persists across restart; invalid bounds preserve database bytes', async () => {
  const workspace = await fresh();
  const running = launch(workspace, 'run', 60_000);
  await running.event('admitted');
  const competing = await launch(workspace, 'run', 60_000).done;
  assert.equal(competing.code, 1, 'a second writer must not open the storage');
  running.child.kill('SIGKILL');
  await running.done;
  const stopped = await launch(workspace, 'stop').done;
  assert.equal(stopped.code, 0, stopped.errors);
  assert.equal((await status(workspace)).task.state.outcome.status, 'aborted');
  assert.equal((await launch(workspace).done).code, 0);
  const before = await readFile(resolve(workspace, 'session.db'));
  const invalid = await launch(workspace, 'run', 28_800_001).done;
  assert.notEqual(invalid.code, 0);
  assert.match(invalid.errors, /Wake delay must/);
  assert.deepEqual(await readFile(resolve(workspace, 'session.db')), before);
  assert.equal((await status(workspace)).task.state.outcome.status, 'aborted');
});
