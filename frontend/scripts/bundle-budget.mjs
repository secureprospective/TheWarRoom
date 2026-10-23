import { readFileSync, readdirSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dist = resolve(frontend, 'dist');
const html = readFileSync(resolve(dist, 'index.html'), 'utf8');
const entries = [...html.matchAll(/<script\b[^>]*\bsrc="([^"]+\.js)"/g)]
  .map((match) => resolve(dist, match[1].replace(/^\//, '')));
const manifest = JSON.parse(readFileSync(resolve(dist, '.vite/manifest.json'), 'utf8'));

// Rolldown can extract eagerly loaded shared chunks; charge the whole static graph.
function eagerFiles(entry) {
  const files = new Set();
  function visit(chunk) {
    if (files.has(chunk.file)) return;
    files.add(chunk.file);
    for (const key of chunk.imports ?? []) visit(manifest[key]);
  }
  visit(Object.values(manifest).find((chunk) => resolve(dist, chunk.file) === entry));
  return [...files].map((file) => resolve(dist, file));
}
const eager = new Set(entries.flatMap(eagerFiles));
const fixture = JSON.parse(readFileSync(resolve(frontend, 'src/app/data/fixtures/snapshot.json')));
const envelope = JSON.parse(readFileSync(resolve(frontend, 'src/app/data/fixtures/envelope-demo.json')));
const names = fixture.players.value.map((player) => player.name).filter(Boolean);

// Entry ceiling: ring 1 left 320,449 bytes; ring 2 allowed eligibility (p2a, +369) and Market
// (p2c, +1,000). 321,818 is their sum, not a loosening: new surfaces must load lazily.
const BUDGET = 321_818;
// Words only the commissioner forms and screens carry: they must live in lazy chunks.
const commissionerForms = ['Start an app league', 'Give or take a role', 'Change a rule',
  'Press and hold to confirm', 'Undo this change'];

function check(entry, budget = BUDGET, source = readFileSync(entry, 'utf8')) {
  const size = Buffer.byteLength(source);
  if (size > budget) throw new Error(`Entry ${size} bytes exceeds ${budget} byte budget`);
  if (names.slice(0, 12).every((name) => source.includes(name))) {
    throw new Error('Fixture player run in entry');
  }
  if (/CM-\d{3}|commish\.[a-z]/.test(source)) {
    throw new Error('Commissioner registry in owner entry');
  }
  if (source.includes('League receipt')) {
    throw new Error('Commissioner proxy in owner entry');
  }
  if (commissionerForms.some((words) => source.includes(words))) {
    throw new Error('Commissioner forms in owner entry');
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
  const source = eagerFiles(entry).map((file) => readFileSync(file, 'utf8')).join('');
  console.log(`bundle-budget: ${entry}: ${check(entry, BUDGET, source)} eager bytes (limit ${BUDGET})`);
  if (process.argv.includes('--self-test')) {
    assert.throws(() => check(entry, 1000), /exceeds 1000/);
    assert.throws(() => check(entry, BUDGET, names.slice(0, 12).join('|')), /Fixture player run/);
    assert.throws(() => check(entry, BUDGET, envelope.receipt.correlationId), /Envelope fixture in entry/);
    assert.throws(() => check(entry, BUDGET, 'Move stages'), /rail implementation in entry/);
    assert.throws(() => check(entry, BUDGET, 'League receipt'), /Commissioner proxy in owner entry/);
    assert.throws(() => check(entry, BUDGET, 'CM-001'), /Commissioner registry in owner entry/);
    assert.throws(() => check(entry, BUDGET, 'Give or take a role'), /Commissioner forms in owner entry/);
    assert.throws(() => check(entry, BUDGET, 'commish.league.teams'), /Commissioner registry in owner entry/);
    console.log('bundle-budget: rejected 1 KB budget, snapshot, envelope, rail and commissioner leaks');
  }
}

const moveChunk = readdirSync(resolve(dist, 'assets'))
  .find((name) => /^MyMoves\..*\.js$/.test(name));
if (!moveChunk) throw new Error('No lazy MyMoves chunk');
const moveSource = readFileSync(resolve(dist, 'assets', moveChunk), 'utf8');
// The rail is shared with the bid plans, so it may sit in its own lazy chunk beside MyMoves.
const railSource = readdirSync(resolve(dist, 'assets'))
  .filter((name) => /^EnvelopeRail\..*\.js$/.test(name))
  .map((name) => readFileSync(resolve(dist, 'assets', name), 'utf8'))
  .join(moveSource);
assert.ok(railSource.includes('Move stages'), 'Rail must live in a lazy chunk, not the entry');
assert.ok(moveSource.includes('landing simulated from a real roster snapshot'),
  'MyMoves caption missing');
console.log(`bundle-budget: lazy MyMoves + rail verified in ${moveChunk}`);

const commissionerChunks = readdirSync(resolve(dist, 'assets'))
  .filter((name) => /^proxy\..*\.js$/.test(name));
assert.ok(commissionerChunks.some((name) =>
  readFileSync(resolve(dist, 'assets', name), 'utf8').includes('League receipt')),
  'Commissioner IPC proxy must live in a lazy chunk');
console.log('bundle-budget: lazy commissioner IPC proxy verified');

const lazySources = readdirSync(resolve(dist, 'assets')).filter((name) => /\.js$/.test(name))
  .filter((name) => !eager.has(resolve(dist, 'assets', name)))
  .map((name) => readFileSync(resolve(dist, 'assets', name), 'utf8'));
for (const words of commissionerForms) {
  assert.ok(lazySources.some((source) => source.includes(words)),
    `Commissioner form text “${words}” must live in a lazy chunk`);
}
console.log('bundle-budget: lazy commissioner forms and screens verified');
