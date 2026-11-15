import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it } from 'vitest';
import { createMigration, parsePreparation, type Preparation } from './migration';
import MoveMFL from './MoveMFL';

const prep: Preparation = {
  league: 'native.mfl-12345-2026', name: 'Our league', season: 2026, franchises: 16,
  players: 400, picks: 64, settings: 200, notes: [], problems: [], ready: true, preset: 'mirror16',
  waiting: [{ field: 'contracts.maximumYears', label: 'Maximum new contract years',
    schema: { kind: 'count', minimum: 1, maximum: 20, choices: [], fields: [] } }],
};
const receipt = { entry: 'import.1', league: prep.league, revision: 1,
  recordedAt: '2026-10-31T12:00:00Z', message: 'Imported' };

it('prepares, chooses a bundle, answers catalogue inputs and imports once into the selected league',
  async () => {
  const calls: { command: string; args: Record<string, unknown> }[] = [];
  let opened = '';
  const flow = createMigration({ run: async (command, body) => {
    const args = JSON.parse(body) as Record<string, unknown>;
    calls.push({ command, args });
    return command.endsWith('prepare') ? { ...prep, preset: args.preset || 'mirror16' } : [receipt];
  }, open: async league => { opened = league; } });
  flow.input('12345');
  flow.key('fixture-key-no-real-secret');
  await flow.prepare();
  expect(flow.peek().preset).toBe('mirror16');
  expect(flow.peek().key).toBe('');
  await flow.importLeague();
  expect(calls).toHaveLength(1);
  await flow.prepare('legacy_nfl');
  flow.answer('contracts.maximumYears', { t: 'text', v: '7' });
  flow.reason('Move our league into the app');
  expect(flow.blocked()).toBe('');
  await flow.importLeague();
  await flow.importLeague();
  expect(calls).toHaveLength(3);
  expect(calls[2]).toEqual({ command: 'league.migrate.import', args: {
    mflLeague: '12345', preset: 'legacy_nfl', reason: 'Move our league into the app',
    answers: [{ field: 'contracts.maximumYears', value: 7 }],
  } });
  expect(opened).toBe(prep.league);
});
it('blocks problems, invalid answers and missing reasons without recording', async () => {
  let imports = 0;
  const flow = createMigration({ run: async command => {
    if (command.endsWith('import')) imports++;
    return { ...prep, ready: false, problems: ['A player has no franchise'] };
  }, open: async () => {} });
  flow.input('12345');
  await flow.prepare();
  flow.reason('Move');
  await flow.importLeague();
  expect(imports).toBe(0);
  flow.input('22222');
  expect(flow.peek().prep).toBeUndefined();
});
it('keeps a failed import retryable but never imports again after recording', async () => {
  let imports = 0;
  const flow = createMigration({ run: async command => {
    if (command.endsWith('prepare')) return { ...prep, waiting: [] };
    if (++imports === 1) throw new Error('MFL is unavailable; nothing imported');
    return [receipt];
  }, open: async () => { throw new Error('League readings unavailable'); } });
  flow.input('12345'); await flow.prepare(); flow.reason('Move');
  await flow.importLeague();
  expect(flow.peek().done).toBe('');
  await flow.importLeague(); await flow.importLeague();
  expect(imports).toBe(2);
  expect(flow.peek().done).toBe(prep.league);
  expect(flow.peek().error).toContain('League readings unavailable');
});
it('renders the desktop import entry and refuses malformed preparation', () => {
  const html = renderToStaticMarkup(createElement(MoveMFL));
  for (const words of ['Move my MFL league', 'MFL league ID', 'type="password"', 'Prepare'])
    expect(html).toContain(words);
  expect(() => parsePreparation({ ...prep, ready: 'yes' })).toThrow();
});
it('clears the credential and typed reason when choosing a different MFL league', () => {
  const flow = createMigration({ run: async () => prep, open: async () => {} });
  flow.input('12345'); flow.key('first-league-fixture-key'); flow.reason('First league');
  flow.input('23456');
  expect(flow.peek().key).toBe('');
  expect(flow.peek().reason).toBe('');
});

it('edits a nested waiting setting through the migration controller', async () => {
  const nested: Preparation = { ...prep, waiting: [{ field: 'lineup.positions', label: 'Positions',
    schema: { kind: 'list', minimum: 0, maximum: 20, choices: [], fields: [],
      element: { kind: 'count', minimum: 1, maximum: 20, choices: [], fields: [] } } }] };
  const flow = createMigration({ run: async () => nested, open: async () => {} });
  flow.input('12345'); await flow.prepare();
  flow.edit({ action: 'add', path: ['lineup.positions'], index: 0 });
  expect(flow.peek().drafts['lineup.positions']).toMatchObject({ t: 'list', items: [{ t: 'text' }] });
  flow.edit({ action: 'remove', path: ['lineup.positions'], index: 0 });
  expect(flow.peek().drafts['lineup.positions']).toMatchObject({ t: 'list', items: [] });
  const before = flow.peek();
  flow.edit({ action: 'add', path: ['unknown'], index: 0 });
  expect(flow.peek()).toBe(before);
});
