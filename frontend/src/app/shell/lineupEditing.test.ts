import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { commands, createCommands } from '../commands/registry';
import { LiveProvider } from '../data/provider';
import type { LineupCheck } from '../data/contract';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { lineupReceipt } from '../data/lineupReceiptTestData';
import { clockDom, TestElement } from '../clock/testDom';
import { FranchiseHQ } from './FranchiseHQ';
import * as draftModule from './draftLineup';
import { EnvelopeRail } from './EnvelopeRail';
import MyMoves from './MyMoves';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const roots: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of roots.splice(0)) root.unmount(); });
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
async function flush() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}
async function setup() {
  const snapshot = heldSnapshot();
  const provider = new LiveProvider();
  const read = vi.spyOn(provider, 'lineup').mockResolvedValue(heldLineup());
  const check = vi.spyOn(provider, 'checkLineup').mockImplementation(async (_id, ids) => ({
    legal: true, full: ids.length === 21,
    problems: ids.length === 21 ? [] : [{ subject: 'WR', kind: 'short', message: 'Go: needs another WR' }],
  }));
  const drafted = vi.spyOn(provider, 'draftLineup').mockResolvedValue(lineupReceipt());
  const moves = vi.spyOn(provider, 'moves').mockResolvedValue([]);
  const listeners = new Set<() => void>();
  vi.spyOn(provider, 'onMovesChange').mockImplementation((listener) => {
    listeners.add(listener);
    return () => { listeners.delete(listener); };
  });
  vi.spyOn(provider, 'onSeasonChange').mockImplementation(() => () => {});
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  await c.lineups.refresh('0025');
  const hooks = ['use', 'useLineupEdit', 'useMoves', 'loadMoves', 'checkLineup', 'onMovesChange'] as const;
  for (const key of hooks) {
    vi.spyOn(commands, key).mockImplementation(c[key]);
  }
  for (const key of ['refresh', 'peek', 'subscribe', 'onSeasonChange', 'onMovesChange'] as const) {
    vi.spyOn(commands.lineups, key).mockImplementation(c.lineups[key]);
  }
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const root = createRoot(dom.element);
  roots.push(root);
  let outside = 0;
  function Outside() {
    outside += 1;
    return createElement(FranchiseHQ, { snapshot });
  }
  const rosterRender = vi.spyOn(draftModule, 'useDraftCheck');
  await act(async () => { root.render(createElement(Outside)); });
  async function dispatch<K extends keyof typeof c.registry>(
    id: K, args: Parameters<(typeof c.registry)[K]['run']>[0],
  ) {
    await act(async () => { c.dispatch(id, args); await flush(); });
  }
  return {
    ...dom, root, c, snapshot, provider, check, drafted, read, moves, dispatch, rosterRender,
    outside: () => outside,
    event: async () => { await act(async () => { for (const listener of listeners) listener(); }); },
  };
}
function findControls(element: TestElement): TestElement[] {
  return element.children.flatMap((child): TestElement[] => {
    if (!(child instanceof TestElement)) return [];
    return child.tagName === 'BUTTON' ? [child] : findControls(child);
  });
}
function findLabel(element: TestElement, label: string): boolean {
  return element.getAttribute('aria-label') === label || element.children.some(
    (child) => child instanceof TestElement && findLabel(child, label));
}

describe('HQ lineup editing', () => {
  it('checks every toggle, marks changes, retains the prior verdict and drops a stale reply', async () => {
    const t = await setup();
    await t.dispatch('lineup.edit', { franchiseId: '0025' });
    expect(t.element.textContent).toContain('Click a starter to bench him');
    const unchanged = findControls(t.element as unknown as TestElement)
      .find((control) => control.textContent === 'Check and save plan')!;
    expect(unchanged.getAttribute('title')).toBe('No changes from saved lineup');
    const before = { outside: t.outside(), hq: vi.mocked(commands.use).mock.calls.length };
    const renders = t.rosterRender.mock.calls.length;
    let resolve!: (check: LineupCheck) => void;
    t.check.mockReturnValueOnce(new Promise((done) => { resolve = done; }));
    await t.dispatch('lineup.toggle', { playerId: '16195' });
    expect(t.element.textContent).toContain('1 change from saved');
    expect(t.element.textContent).toContain('Changed from saved');
    expect(t.element.textContent).not.toContain('Reading lineup…');
    expect(t.rosterRender.mock.calls.length).toBeGreaterThan(renders);
    expect({ outside: t.outside(), hq: vi.mocked(commands.use).mock.calls.length }).toEqual(before);
    t.check.mockResolvedValueOnce({ legal: false, full: false,
      problems: [{ subject: 'WR', kind: 'over', message: 'Go: WR limit exceeded' }] });
    await t.dispatch('lineup.toggle', { playerId: '16428' });
    expect(t.element.textContent).toContain('Draft · not legal · Go: WR limit exceeded · 2 changes');
    await act(async () => { resolve({ legal: true, full: true, problems: [] }); });
    expect(t.element.textContent).toContain('Go: WR limit exceeded');
    expect(t.check).toHaveBeenCalledTimes(3);
    await t.dispatch('lineup.reset', {});
    expect(t.element.textContent).toContain('Draft · legal · 21 of 21 starters · 0 changes from saved');
    await t.dispatch('lineup.toggle', { playerId: '16195' });
    expect(t.element.textContent).toContain(
      'Draft · legal, partial · 20 of 21 starters · Go: needs another WR',
    );
  });
  it('shows the plan area only while editing or holding a plan', async () => {
    const t = await setup();
    const hint = 'Press Check and save plan when the lineup is right';
    expect(t.element.textContent).not.toContain(hint);
    expect(findLabel(t.element as unknown as TestElement, 'Lineup plan')).toBe(false);
    await t.dispatch('lineup.edit', { franchiseId: '0025' });
    expect(t.element.textContent).toContain(hint);
    await t.dispatch('lineup.cancel', {});
    expect(findLabel(t.element as unknown as TestElement, 'Lineup plan')).toBe(false);
  });
  it('reloads moves after saving, so a superseded plan stops offering hand-off', async () => {
    const t = await setup();
    const old = { ...lineupReceipt(), correlationId: 'lineup-old' };
    t.moves.mockResolvedValue([old]);
    await act(async () => { await t.c.loadMoves('0025'); });
    await t.dispatch('lineup.edit', { franchiseId: '0025' });
    await t.dispatch('lineup.toggle', { playerId: '16195' });
    t.moves.mockResolvedValue([{ ...old, state: 'stale', audit: [...old.audit, {
      at: '2026-10-07T14:01:00Z', from: 'ready', event: 'invalidate', to: 'stale',
      note: 'superseded by lineup-test',
    }] }]);
    await t.dispatch('lineup.draft', {});
    expect(t.c.readMoves().moves.find((r) => r.correlationId === 'lineup-old')?.state).toBe('stale');
  });
  it('shows Go blocks, the real WR swap, hand-off instructions and Landed on the event', async () => {
    const t = await setup();
    await t.dispatch('lineup.edit', { franchiseId: '0025' });
    await t.dispatch('lineup.toggle', { playerId: '16195' });
    await t.dispatch('lineup.toggle', { playerId: '16428' });
    t.drafted.mockResolvedValueOnce(lineupReceipt('blocked'));
    await t.dispatch('lineup.draft', {});
    expect(t.element.textContent).toContain('Not ready · week 5 lineups are locked');
    await t.dispatch('lineup.draft', {});
    expect(t.element.textContent).toContain('Start Smith, Xavier (WR)');
    expect(t.element.textContent).toContain('Bench Wicks, Dontayvion (WR)');
    expect(t.element.textContent).toContain('Open MFL lineup page');
    const ready = lineupReceipt();
    const handed = { ...ready, state: 'handed_off' as const, audit: [...ready.audit, {
      at: '2026-10-07T14:01:00Z', from: 'ready' as const, event: 'hand_off' as const,
      to: 'handed_off' as const, note: 'Opened MFL lineup page',
    }] };
    vi.spyOn(t.provider, 'handOff').mockResolvedValue(handed);
    await t.dispatch('move.handoff', { correlationId: ready.correlationId });
    expect(t.element.textContent).toContain("Opened MFL's week 5 lineup page.");
    expect(t.element.textContent).toContain('press Submit Lineup');
    expect(t.element.textContent).toContain('Handed off · Opened MFL lineup page');
    const landed = { ...handed, state: 'landed' as const, audit: [...handed.audit, {
      at: '2026-10-07T14:02:00Z', from: 'handed_off' as const, event: 'match' as const,
      to: 'landed' as const, note: 'MFL saved the drafted starters',
    }] };
    t.moves.mockResolvedValue([landed]);
    const readCount = t.read.mock.calls.length;
    await t.event();
    expect(t.read.mock.calls.length).toBe(readCount + 1);
    expect(t.element.textContent).toContain('Landed · MFL saved the drafted starters');
    expect(t.c.readMoves().moves[0]).toEqual(landed);
  });
  it('keeps handed-off instructions after cancel and refreshes My moves on target:moves', async () => {
    const t = await setup();
    await t.dispatch('lineup.edit', { franchiseId: '0025' });
    await t.dispatch('lineup.toggle', { playerId: '16195' });
    await t.dispatch('lineup.draft', {});
    const ready = lineupReceipt();
    const handed = { ...ready, state: 'handed_off' as const, audit: [...ready.audit, {
      at: '2026-10-07T14:01:00Z', from: 'ready' as const, event: 'hand_off' as const,
      to: 'handed_off' as const, note: 'Opened MFL',
    }] };
    vi.spyOn(t.provider, 'handOff').mockResolvedValue(handed);
    await t.dispatch('move.handoff', { correlationId: ready.correlationId });
    await t.dispatch('lineup.cancel', {});
    expect(t.element.textContent).toContain('TheWarRoom will show Landed when MFL has them');
    await act(async () => {
      t.root.render(createElement(MyMoves, { snapshot: t.snapshot, executor: t.c }));
    });
    t.moves.mockResolvedValue([{ ...handed, state: 'landed' }]);
    await t.event();
    expect(t.element.textContent).toContain('Landed');
  });
  it('labels lineup rails with the week, last-name summary, and a Ready hand-off Act', async () => {
    const t = await setup();
    await act(async () => { t.root.render(createElement(EnvelopeRail, {
      snapshot: t.snapshot, receipt: lineupReceipt(),
    })); });
    expect(t.element.textContent).toContain('Week 5 lineup');
    expect(t.element.textContent).toContain('+Smith −Wicks');
    expect(t.element.textContent).toContain('Lineup change');
    expect(t.element.textContent).toContain('Open MFL lineup page');
  });
});
