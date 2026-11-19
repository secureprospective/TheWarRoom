import { spawnSync } from 'node:child_process';

function run(command, args, env = process.env) {
  const result = spawnSync(command, args, { stdio: 'inherit', env, shell: false });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}

run('node', ['scripts/check-path-casing.mjs']);
run('pnpm', ['exec', 'tsc']);
run('pnpm', ['exec', 'vite', 'build'], { ...process.env, TWR_WEB_BUILD: '1' });
