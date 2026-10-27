import { it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { parsePersonal, type ListChange, type ListKind } from './contract';
import { createPersonalState, personalState } from './state';
import PersonalLists from './PersonalLists';
import { commands } from '../commands/registry';
import { renderState } from '../shell/state';
import { FixtureProvider } from '../data/provider';
import { parseShellStatus } from '../shell/StatusStrip';

const wire: unknown = JSON.parse(readFileSync('src/app/personal/wire.testdata.json', 'utf8'));
const reading = parsePersonal(wire);

it('validates fixtures and rejects malformed IPC rows, unknown lists and noncanonical ids', () => {
  expect(reading.entries[0].id).toBe('0816');
  expect(reading.bait.rows[0].cells).toContain('Future picks');
  for (const patch of [{ kind: 'waiver' }, { id: '816' }, { order: -1 }, { at: 'yesterday' }]) {
    expect(() => parsePersonal({ ...reading, entries: [{ ...reading.entries[0], ...patch }] })).toThrow();
  }
  expect(() => parsePersonal({ ...reading, submitted: true })).toThrow();
  expect(parseShellStatus({ health: 'Reduced sources', state: 'fail', gameday: 'Week 5' }).state)
    .toBe('fail');
  expect(() => parseShellStatus({ health: 'Okay', state: 'green', gameday: '' })).toThrow();
});

it('writes list mutations only through the provider, reloads and retains data on failure', async () => {
  const calls: { mine: string; kind: ListKind; change: ListChange }[] = [];
  let fail = false;
  const provider = {
    read: vi.fn(async () => { if (fail) throw new Error('offline'); return reading; }),
    change: vi.fn(async (mine: string, kind: ListKind, change: ListChange) => {
      calls.push({ mine, kind, change });
    }),
    handoff: vi.fn(async (_kind: ListKind | 'ir') => {}),
  };
  const state = createPersonalState(provider);
  await state.load('0025');
  for (const [kind, action] of [['watch', 'add'], ['bait', 'remove'], ['note', 'add'],
    ['draft', 'up'], ['draft', 'down']] as const)
    await state.change('0025', kind, { id: '0816', entity: 'player', action, text: 'Plain note' });
  expect(calls.map(c => [c.mine, c.kind, c.change.action])).toEqual([
    ['0025', 'watch', 'add'], ['0025', 'bait', 'remove'], ['0025', 'note', 'add'],
    ['0025', 'draft', 'up'], ['0025', 'draft', 'down'],
  ]);
  expect(provider.read).toHaveBeenCalledTimes(6);
  await state.handoff('draft');
  expect(provider.handoff).toHaveBeenCalledWith('draft');
  fail = true;
  await state.load('0025');
  expect(state.peek('0025').value).toEqual(reading);
  expect(state.peek('0025').error).toContain('offline');
  expect(state.peek('0001').value).toBeUndefined();
});

it('renders status, market, dated escaped notes, MFL handoffs and ordered-list Acts', async () => {
  const snapshot = await new FixtureProvider().snapshot();
  const shell = vi.spyOn(commands, 'use').mockReturnValue({
    ...renderState(commands.read()), franchiseId: '0025' });
  const lists = vi.spyOn(personalState, 'useReading').mockReturnValue({
    value: reading, pending: false, error: '',
  });
  try {
    const html = renderToStaticMarkup(createElement(PersonalLists, { snapshot }));
    expect(html).toContain('Questionable');
    expect(html).toContain('Proj 12.4');
    expect(html).toContain('Plain &lt;text&gt;');
    expect(html).not.toContain('Plain <text>');
    expect(html).toContain('Open scratchpad on MFL');
    expect(html).toContain('Plain-text note');
    expect(html).toContain('Keep dated note');
    expect(html).toContain('for your franchise when no player is chosen');
    expect(html).toContain('data-endpoint="M-117"');
    const draft = renderToStaticMarkup(createElement(PersonalLists, { snapshot, draft: true }));
    expect(draft).toContain('Move up');
    expect(draft).toContain('Move down');
    expect(draft).toContain('Open my draft list on MFL');
  } finally { shell.mockRestore(); lists.mockRestore(); }
});

it('keeps franchise reads independent and excludes older same-franchise publication', async () => {
  const waiting: { mine: string; finish: (value: typeof reading) => void }[] = [];
  const provider = {
    read: (mine: string): Promise<typeof reading> => new Promise(finish => waiting.push({ mine, finish })),
    change: vi.fn(async () => {}), handoff: async () => {},
  };
  const state = createPersonalState(provider);
  const old = state.load('0001');
  const other = state.load('0002');
  const latest = state.load('0001');
  await state.change('0001', 'note', { id: '0816', entity: 'player', action: 'add', text: 'Do not send' });
  expect(provider.change).not.toHaveBeenCalled();
  const current = { ...reading, entries: [] };
  waiting[2].finish(current);
  await latest;
  waiting[0].finish(reading);
  await old;
  expect(state.peek('0001').value).toBe(current);
  expect(state.peek('0002').pending).toBe(true);
  waiting[1].finish(reading);
  await other;
  expect(state.peek('0002').value).toBe(reading);
  expect(state.peek('0001').pending).toBe(false);
});
