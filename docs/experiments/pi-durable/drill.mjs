import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir, mkdtemp, readdir, stat, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

process.umask(0o077);
export const base = dirname(fileURLToPath(import.meta.url));
export async function fresh() {
  await mkdir(resolve(base, '.runs'), { recursive: true, mode: 0o700 });
  return mkdtemp(resolve(base, '.runs/goal-'));
}
export function launch(workspace, action = 'run', delay = 1000) {
  const nodeArgs = [
    '--permission', `--allow-fs-read=${base}`, `--allow-fs-write=${workspace}`,
    resolve(base, 'worker.mjs'), workspace, action, String(delay),
  ];
  const command = action === 'status' ? process.execPath : '/usr/bin/flock';
  const args = action === 'status' ? nodeArgs :
    ['--nonblock', '--no-fork', resolve(workspace, 'owner.lock'), process.execPath, ...nodeArgs];
  const child = spawn(command, args, { cwd: workspace, env: { HOME: workspace, PATH: dirname(process.execPath) },
    stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '', errors = '';
  child.stdout.on('data', (data) => { output += data; });
  child.stderr.on('data', (data) => { errors += data; });
  const done = new Promise((resolveDone, reject) => {
    child.on('error', reject);
    child.on('close', (code, signal) => resolveDone({ code, signal, output, errors }));
  });
  const watchdog = setTimeout(() => child.kill('SIGKILL'),
    action === 'run' ? Math.min(Math.max(delay, 0), 28_800_000) + 15_000 : 15_000);
  done.finally(() => clearTimeout(watchdog));
  return { child, done, async event(name) {
    const deadline = Date.now() + 10_000;
    while (Date.now() < deadline) {
      const found = output.trim().split('\n').filter(Boolean).map(JSON.parse)
        .find((entry) => entry.event === name);
      if (found) return found;
      if (child.exitCode !== null || child.signalCode) throw new Error(output + errors);
      await new Promise((done) => setTimeout(done, 10));
    }
    throw new Error(`Timed out waiting for ${name}: ${output} ${errors}`);
  } };
}
export async function status(workspace) {
  const run = launch(workspace, 'status');
  const result = await run.done;
  assert.equal(result.code, 0, result.errors);
  return JSON.parse(result.output);
}
export async function drill(delay = 1000) {
  const workspace = await fresh();
  console.error(JSON.stringify({ event: 'workspace', workspace }));
  let active;
  const evidence = { workspace, node: process.version, started: new Date().toISOString(), delay };
  let interrupted = false;
  const interrupt = () => { interrupted = true; active?.child.kill('SIGKILL'); };
  const start = () => {
    if (interrupted) throw new Error('Drill stopped; no worker will be restarted');
    return launch(workspace, 'run', delay);
  };
  process.once('SIGINT', interrupt);
  process.once('SIGTERM', interrupt);
  try {
    active = start();
    const admitted = await active.event('admitted');
    if (delay === 0) await active.event('finish-entered');
    else {
      // Wait for the atomic prepare + sleeping checkpoint, not just task admission.
      // A second writer is forbidden; read-only status never starts scheduling.
      let snapshot;
      for (let i = 0; i < 30; i++) {
        snapshot = await status(workspace);
        if (snapshot.goal.completed.includes('prepared')) break;
        await new Promise((done) => setTimeout(done, 10));
      }
      assert.deepEqual(snapshot.goal.completed, ['prepared']);
      assert.equal(snapshot.task.state.status, 'sleeping');
    }
    active.child.kill('SIGKILL');
    evidence.crashed = await active.done;
    assert.equal(evidence.crashed.signal, 'SIGKILL');
    evidence.before = await status(workspace);
    assert.deepEqual(evidence.before.goal.completed, ['prepared']);
    active = start();
    evidence.resumed = await active.done;
    assert.equal(evidence.resumed.code, 0, evidence.resumed.errors);
    evidence.after = await status(workspace);
    assert.equal(evidence.after.goal.taskId, admitted.taskId);
    assert.deepEqual(evidence.after.goal.completed, ['prepared', 'finished']);
    assert.equal(evidence.after.task.state.outcome.status, 'completed');
    active = start();
    assert.equal((await active.done).code, 0);
    assert.deepEqual((await status(workspace)).goal, evidence.after.goal);
    for (const file of await readdir(workspace)) {
      assert.equal((await stat(resolve(workspace, file))).mode & 0o077, 0, file);
    }
    if (interrupted) throw new Error('Drill stopped');
    evidence.result = 'PASS';
    await writeFile(resolve(workspace, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n', { mode: 0o600 });
    return evidence;
  } finally {
    process.removeListener('SIGINT', interrupt);
    process.removeListener('SIGTERM', interrupt);
    if (active && active.child.exitCode === null && !active.child.signalCode) {
      active.child.kill('SIGKILL');
      await active.done;
    }
  }
}
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  if (process.argv[2] === '--stop' && /^goal-[A-Za-z0-9]+$/.test(process.argv[3] ?? '')) {
    const result = await launch(resolve(base, '.runs', process.argv[3]), 'stop').done;
    process.stdout.write(result.output);
    process.stderr.write(result.errors);
    process.exitCode = result.code ?? 1;
  } else {
    if (process.argv.length > 3 || (process.argv[2] && process.argv[2] !== '--overnight')) {
      throw new Error('Usage: node drill.mjs [--overnight | --stop goal-NAME]');
    }
    const evidence = await drill(process.argv[2] === '--overnight' ? 28_800_000 : 3000);
    console.log(JSON.stringify({ result: evidence.result, evidence: resolve(evidence.workspace, 'evidence.json') }));
  }
}
