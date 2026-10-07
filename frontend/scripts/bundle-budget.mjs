import { readFileSync, readdirSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dist = resolve(frontend, 'dist');
const html = readFileSync(resolve(dist, 'index.html'), 'utf8');
const entries = [...html.matchAll(/<script\b[^>]*\bsrc="([^"]+\.js)"/g)]
  .map((match) => resolve(dist, match[1].replace(/^\//, '')));
const fixture = JSON.parse(readFileSync(resolve(frontend, 'src/app/data/fixtures/snapshot.json')));
const envelope = JSON.parse(readFileSync(resolve(frontend, 'src/app/data/fixtures/envelope-demo.json')));
const names = fixture.players.value.map((player) => player.name).filter(Boolean);

function check(entry, budget = 325_000, source = readFileSync(entry, 'utf8')) {
  const size = Buffer.byteLength(source);
  if (size > budget) throw new Error(`Entry ${size} bytes exceeds ${budget} byte budget`);
  if (names.slice(0, 12).every((name) => source.includes(name))) {
    throw new Error('Fixture player run in entry');
  }
  if (source.includes(envelope.receipt.correlationId)) {
    throw new Error('Envelope fixture in entry');
  }
  if (source.includes('Move stages') || source.includes('landing simulated from a real roster snapshot')) {
    throw new Error('MyMoves or rail implementation in entry');
  }
  return size;
}

if (!entries.length) throw new Error('No entry script in dist/index.html');
for (const entry of entries) {
  console.log(`bundle-budget: ${entry}: ${check(entry)} bytes (limit 325000)`);
  if (process.argv.includes('--self-test')) {
    assert.throws(() => check(entry, 1000), /exceeds 1000/);
    assert.throws(() => check(entry, 325_000, names.slice(0, 12).join('|')), /Fixture player run/);
    assert.throws(() => check(entry, 325_000, envelope.receipt.correlationId), /Envelope fixture in entry/);
    assert.throws(() => check(entry, 325_000, 'Move stages'), /rail implementation in entry/);
    console.log('bundle-budget: rejected 1 KB budget, snapshot, envelope and rail leaks');
  }
}

const moveChunk = readdirSync(resolve(dist, 'assets')).find((name) => /^MyMoves\..*\.js$/.test(name));
if (!moveChunk) throw new Error('No lazy MyMoves chunk');
const moveSource = readFileSync(resolve(dist, 'assets', moveChunk), 'utf8');
assert.ok(moveSource.includes('Move stages'), 'Rail must live in the MyMoves chunk');
assert.ok(moveSource.includes('landing simulated from a real roster snapshot'), 'MyMoves caption missing');
console.log(`bundle-budget: lazy MyMoves + rail verified in ${moveChunk}`);
