import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import type { LineupReading, Snapshot } from '../data/contract';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { FixtureProvider } from '../data/provider';
import { commands, createCommands } from '../commands/registry';
import * as slots from '../look/Slots';
import * as lineups from './lineups';
import { FranchiseHQ } from './FranchiseHQ';
import { clockDom, TestElement } from '../clock/testDom';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
const mounted: ReturnType<typeof createRoot>[] = [];
afterEach(async () => {
  await act(async () => { for (const root of mounted.splice(0)) root.unmount(); });
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
function pending() {
  let resolve!: (value: LineupReading) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<LineupReading>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
function headings(root: TestElement): string[] {
  return root.children.flatMap((node): string[] => {
    if (!(node instanceof TestElement)) return [];
    return node.tagName === 'H5' ? [node.textContent] : headings(node);
  });
}
async function setup(read: () => Promise<LineupReading>, cached = false) {
  const snapshot = heldSnapshot();
  const provider = new FixtureProvider();
  provider.lineup = read;
  let event!: () => void;
  const unsubscribe = vi.fn();
  provider.onSeasonChange = (listener) => { event = listener; return unsubscribe; };
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  if (cached) await Promise.resolve();
  const hqUse = vi.spyOn(commands, 'use').mockImplementation(c.use);
  for (const key of ['refresh', 'peek', 'subscribe', 'onSeasonChange'] as const) {
    vi.spyOn(commands.lineups, key).mockImplementation(c.lineups[key]);
  }
  let outside = 0;
  let rosterRenders = 0;
  let capRenders = 0;
  const originalUse = lineups.useLineup;
  vi.spyOn(lineups, 'useLineup').mockImplementation((...args: Parameters<typeof originalUse>) => {
    rosterRenders += 1;
    return originalUse(...args);
  });
  const originalChip = slots.SignalChip;
  vi.spyOn(slots, 'SignalChip').mockImplementation((props) => {
    if (props.detail?.startsWith('Franchises')) capRenders += 1;
    return createElement(originalChip, props);
  });
  function Outside({ value }: { value: Snapshot }) {
    outside += 1;
    return createElement('div', null,
      createElement('p', null, 'Outside roster'),
      createElement(FranchiseHQ, { snapshot: value }));
  }
  const dom = clockDom([]);
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.stubGlobal('document', dom.document);
  vi.stubGlobal('window', { document: dom.document, HTMLIFrameElement: class {} });
  const root = createRoot(dom.element);
  mounted.push(root);
  await act(async () => { root.render(createElement(Outside, { value: snapshot })); });
  return {
    ...dom, root, snapshot, c, event: () => event(), unsubscribe,
    counts: () => ({ outside, hq: hqUse.mock.calls.length, roster: rosterRenders, cap: capRenders }),
    replace: async (value: Snapshot) => {
      await act(async () => { root.render(createElement(Outside, { value })); });
    },
  };
}
function expectReserves(root: TestElement, snapshot: Snapshot) {
  const groups = headings(root);
  for (const [status, name] of [['IR', 'IR'], ['TAXI_SQUAD', 'Taxi']] as const) {
    const count = snapshot.rosters.value[0].players.filter((p) => p.rosterStatus === status).length;
    expect(groups.find((g) => g.startsWith(`${name} ·`))).toContain(`${name} · ${count}`);
  }
}
describe('HQ saved lineup', () => {
  it('shows full legal starters and bench in reading order, leaving IR and Taxi alone', async () => {
    const r = heldLineup();
    const test = await setup(vi.fn().mockResolvedValue(r), true);
    expect(test.c.lineups.peek('0025').reading).toEqual(r);
    expect(test.counts().roster).toBe(1);
    expect(test.element.textContent).toContain('Week 5 lineup · legal · 21 of 21 starters');
    expect(headings(test.element as unknown as TestElement))
      .toContain('Starters · 21 fresh · 10-07');
    expectReserves(test.element as unknown as TestElement, test.snapshot);
    const text = test.element.textContent!;
    expect(text.indexOf('Starters ·')).toBeLessThan(text.indexOf('Bench ·'));
    const names = r.starters.map((p) => test.snapshot.players.value.find((v) => v.id === p.id)!.name!);
    const firstCards = text.slice(text.indexOf('Starters ·'), text.indexOf('Bench ·'));
    for (let i = 1; i < names.length; i += 1) {
      expect(firstCards.indexOf(names[i - 1])).toBeLessThan(firstCards.indexOf(names[i]));
    }
  });
  it.each(['partial', 'illegal', 'empty', 'rules'] as const)('renders %s Go text verbatim', async (kind) => {
    const r = heldLineup();
    r.check = { full: false, legal: true, problems: [
      { subject: 'WR', kind: 'short', message: 'WR: 1 starting, needs at least 2' },
      { subject: 'Total', kind: 'short', message: 'Total: 20 starting, needs at least 21' },
    ] };
    let expected = 'Week 5 lineup · legal, partial · 21 of 21 starters · ';
    expected += r.check.problems.map((p) => p.message).join(' · ');
    if (kind === 'illegal') {
      r.check.legal = false;
      r.check.problems.unshift({ subject: 'QB', kind: 'over', message: 'QB: 2 starting, at most 1' });
      expected = `Week 5 lineup · not legal · ${r.check.problems.map((p) => p.message).join(' · ')}`;
    }
    if (kind === 'empty') {
      r.starters = [];
      r.provenance.freshness.note = 'no saved lineup for franchise 0025';
      expected = 'No saved lineup · no saved lineup for franchise 0025';
    }
    if (kind === 'rules') {
      r.rulesSource.freshness.state = 'fail';
      expected = "The league's lineup rules could not be read: count is unreadable";
      r.check = { full: false, legal: false, problems: [{
        subject: 'Rules', kind: 'unknown', message: expected,
      }] };
    }
    const test = await setup(vi.fn().mockResolvedValue(r));
    expect(test.element.textContent).toContain(expected);
    expectReserves(test.element as unknown as TestElement, test.snapshot);
  });
  it('shows and inspects a saved starter without a directory or roster row', async () => {
    const r = heldLineup();
    r.starters.unshift({ id: '99999' });
    const test = await setup(vi.fn().mockResolvedValue(r));
    expect(test.element.textContent).toContain('99999');
    expect(test.element.textContent).toContain('not on the active roster');
    expect(test.element.textContent).toContain('Unavailable');
    await act(async () => {
      test.c.dispatch('inspector.open', { subject: { kind: 'player', id: '99999', franchiseId: '0025' } });
    });
    expect(test.c.read().subject?.id).toBe('99999');
    expectReserves(test.element as unknown as TestElement, test.snapshot);
  });
  it('marks an IR starter, retaining the separate IR group', async () => {
    const r = heldLineup();
    const snap = heldSnapshot();
    const ir = snap.rosters.value[0].players.find((p) => p.rosterStatus === 'IR')!;
    r.starters.unshift({ id: ir.id });
    const test = await setup(vi.fn().mockResolvedValue(r));
    expect(test.element.textContent).toContain('not on the active roster');
    expectReserves(test.element as unknown as TestElement, test.snapshot);
  });
  it('keeps Active during pending and rejection, with a reserved legality line', async () => {
    const first = pending();
    const test = await setup(() => first.promise);
    expect(test.element.textContent).toContain('Reading lineup…');
    expect(headings(test.element as unknown as TestElement)[0]).toContain('Active · 38');
    expectReserves(test.element as unknown as TestElement, test.snapshot);
    await act(async () => { first.reject(new Error('target lineup: no snapshot loaded')); });
    expect(test.element.textContent).toContain('Lineup unavailable · target lineup: no snapshot loaded');
    expect(headings(test.element as unknown as TestElement)[0]).toContain('Active · 38');
    expectReserves(test.element as unknown as TestElement, test.snapshot);
  });
  it('does zero renders for equal events, and renders only Roster for changed events', async () => {
    const r = heldLineup();
    const read = vi.fn().mockResolvedValue(r);
    const test = await setup(read, true);
    const before = test.counts();
    await act(async () => { test.event(); });
    expect(test.counts()).toEqual(before);
    read.mockResolvedValue({ ...r, week: 6 });
    await act(async () => { test.event(); });
    expect(test.counts()).toEqual({ ...before, roster: before.roster + 1 });
    expect(test.element.textContent).toContain('Week 6 lineup');
    await act(async () => { test.root.unmount(); });
    mounted.splice(mounted.indexOf(test.root), 1);
    expect(test.unsubscribe).toHaveBeenCalledTimes(1);
  });
  it('ignores a slow initial response after a fast event and refreshes changed snapshots', async () => {
    const first = pending();
    const r = heldLineup();
    const read = vi.fn().mockReturnValueOnce(first.promise).mockReturnValueOnce(first.promise)
      .mockResolvedValueOnce({ ...r, week: 6 }).mockResolvedValue({ ...r, week: 7 });
    const test = await setup(read);
    await act(async () => { test.event(); });
    expect(test.element.textContent).toContain('Week 6 lineup');
    const counts = test.counts();
    await act(async () => { first.resolve(r); });
    expect(test.counts()).toEqual(counts);
    await test.replace(structuredClone(test.snapshot));
    expect(test.element.textContent).toContain('Week 7 lineup');
    expect(read).toHaveBeenCalledTimes(4);
  });
});

it('starts the read at franchise selection before mounting HQ', async () => {
  const provider = new FixtureProvider();
  const read = vi.spyOn(provider, 'lineup').mockResolvedValue(heldLineup());
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(heldSnapshot());
  expect(read).not.toHaveBeenCalled();
  c.dispatch('franchise.set', { franchiseId: '0025' });
  expect(read).toHaveBeenCalledWith('0025');
  await act(async () => { await read.mock.results[0].value; });
  expect(c.lineups.peek('0025').reading).toEqual(heldLineup());
});
it('uses Active after a failed refresh while retaining the last success in the cache', async () => {
  const r = heldLineup();
  const read = vi.fn().mockResolvedValue(r);
  const test = await setup(read, true);
  read.mockRejectedValue(new Error('not ready'));
  await act(async () => { test.event(); });
  expect(test.element.textContent).toContain('Lineup unavailable · not ready');
  expect(headings(test.element as unknown as TestElement)[0]).toContain('Active · 38');
  expect(test.c.lineups.peek('0025').reading).toEqual(r);
  expectReserves(test.element as unknown as TestElement, test.snapshot);
});
