import assert from 'node:assert/strict';
import { execFileSync, spawn } from 'node:child_process';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { basename, dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Slice 29 driver: builds a throwaway repo with one failing test, runs the durable
// repair goal, kills it mid-fix, restarts it, and checks it stopped at a review branch.
process.umask(0o077);
const base = dirname(fileURLToPath(import.meta.url));
const BRANCH = 'durable/fix-slug';
const env = (workspace) => ({ HOME: workspace, PATH: `${dirname(process.execPath)}:/usr/bin:/bin`,
  GIT_CONFIG_NOSYSTEM: '1', GIT_CONFIG_GLOBAL: '/dev/null', GIT_TERMINAL_PROMPT: '0' });

function launch(workspace, action = 'run') {
  const nodeArgs = ['--permission', `--allow-fs-read=${base}`, `--allow-fs-write=${workspace}`,
    '--allow-child-process', resolve(base, 'demo-worker.mjs'), workspace, action];
  const command = action === 'status' ? process.execPath : '/usr/bin/flock';
  const args = action === 'status' ? nodeArgs :
    ['--nonblock', '--no-fork', resolve(workspace, 'owner.lock'), process.execPath, ...nodeArgs];
  const child = spawn(command, args, { cwd: workspace, env: env(workspace), stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '', errors = '';
  child.stdout.on('data', (data) => { output += data; });
  child.stderr.on('data', (data) => { errors += data; });
  const done = new Promise((resolveDone, reject) => {
    child.on('error', reject);
    child.on('close', (code, signal) => resolveDone({ code, signal, output, errors }));
  });
  const watchdog = setTimeout(() => child.kill('SIGKILL'), 120_000);
  done.finally(() => clearTimeout(watchdog));
  return { child, done, async event(name) {
    const deadline = Date.now() + 30_000;
    while (Date.now() < deadline) {
      if (output.split('\n').some((line) => line.includes(`"event":"${name}"`))) return;
      if (child.exitCode !== null || child.signalCode) throw new Error(output + errors);
      await new Promise((wake) => setTimeout(wake, 10));
    }
    throw new Error(`Timed out waiting for ${name}: ${output} ${errors}`);
  } };
}
async function status(workspace) {
  const result = await launch(workspace, 'status').done;
  assert.equal(result.code, 0, result.errors);
  return JSON.parse(result.output);
}
async function scratchRepo(workspace) {
  const repo = resolve(workspace, 'repo');
  await mkdir(repo);
  // Bug: replace() swaps only the first space.
  await writeFile(resolve(repo, 'slug.mjs'), "export const slug = (text) => text.toLowerCase().replace(' ', '-');\n");
  await writeFile(resolve(repo, 'slug.test.mjs'), [
    "import assert from 'node:assert/strict';", "import { test } from 'node:test';",
    "import { slug } from './slug.mjs';",
    "test('slug joins every word', () => assert.equal(slug('Legacy NFL War Room'), 'legacy-nfl-war-room'));", '',
  ].join('\n'));
  const git = (...args) => execFileSync('git', ['-c', 'user.name=durable-demo', '-c', 'user.email=durable-demo@invalid',
    ...args], { cwd: repo, env: env(workspace), encoding: 'utf8' }).trim();
  git('init', '--quiet', '-b', 'main');
  git('add', '.');
  git('commit', '--quiet', '-m', 'slug helper with a failing test');
  return git;
}

export async function demo() {
  await mkdir(resolve(base, '.runs'), { recursive: true, mode: 0o700 });
  const workspace = await mkdtemp(resolve(base, '.runs/demo-'));
  console.error(JSON.stringify({ event: 'workspace', workspace }));
  const git = await scratchRepo(workspace);
  const mainBefore = git('rev-parse', 'main');
  const evidence = { run: `.runs/${basename(workspace)}`, node: process.version, started: new Date().toISOString(), mainBefore };
  let active = launch(workspace);
  let interrupted = false;
  const interrupt = () => { interrupted = true; active?.child.kill('SIGKILL'); };
  process.once('SIGINT', interrupt);
  process.once('SIGTERM', interrupt);
  try {
    await active.event('fix-entered');
    active.child.kill('SIGKILL');
    evidence.crashed = await active.done;
    assert.equal(evidence.crashed.signal, 'SIGKILL');
    evidence.before = await status(workspace);
    assert.deepEqual(evidence.before.goal.completed, ['reproduced']);
    assert.equal(evidence.before.task.state.checkpoint.phase, 'fix');
    if (interrupted) throw new Error('Demo stopped; no worker will be restarted');
    active = launch(workspace);
    evidence.resumed = await active.done;
    assert.equal(evidence.resumed.code, 0, evidence.resumed.errors);
    evidence.after = await status(workspace);
    assert.equal(evidence.after.goal.taskId, evidence.before.goal.taskId);
    assert.deepEqual(evidence.after.goal.completed, ['reproduced', 'fixed', 'verified', 'proposed']);
    assert.equal(evidence.after.task.state.outcome.status, 'completed');
    if (interrupted) throw new Error('Demo stopped; no worker will be restarted');
    active = launch(workspace);
    assert.equal((await active.done).code, 0);
    assert.deepEqual((await status(workspace)).goal, evidence.after.goal);
    // One failing run before the crash, one passing run after: reproduce was not replayed.
    evidence.testRuns = (await readFile(resolve(workspace, 'test-runs.log'), 'utf8')).trim().split('\n');
    assert.deepEqual(evidence.testRuns, ['fail', 'pass']);
    evidence.mainAfter = git('rev-parse', 'main');
    assert.equal(evidence.mainAfter, mainBefore, 'the goal must never write main');
    assert.equal(git('remote'), '', 'the scratch repo has no remote to push to');
    assert.equal(git('rev-list', '--count', `main..${BRANCH}`), '1');
    assert.equal(git('status', '--porcelain'), '');
    evidence.branch = BRANCH;
    evidence.patch = await readFile(resolve(workspace, 'outbox/fix.patch'), 'utf8');
    assert.match(evidence.patch, /^\+export const slug = .*split/m);
    evidence.result = 'PASS';
    await writeFile(resolve(workspace, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n', { mode: 0o600 });
    return { ...evidence, workspace };
  } finally {
    process.removeListener('SIGINT', interrupt);
    process.removeListener('SIGTERM', interrupt);
    if (active.child.exitCode === null && !active.child.signalCode) {
      active.child.kill('SIGKILL');
      await active.done;
    }
  }
}
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  if (process.argv[2] === '--stop' && /^demo-[A-Za-z0-9]+$/.test(process.argv[3] ?? '')) {
    const result = await launch(resolve(base, '.runs', process.argv[3]), 'stop').done;
    process.stdout.write(result.output);
    process.stderr.write(result.errors);
    process.exitCode = result.code ?? 1;
  } else {
    if (process.argv.length > 2) throw new Error('Usage: node demo.mjs [--stop demo-NAME]');
    const evidence = await demo();
    console.log(JSON.stringify({ result: evidence.result, evidence: resolve(evidence.workspace, 'evidence.json') }));
  }
}
