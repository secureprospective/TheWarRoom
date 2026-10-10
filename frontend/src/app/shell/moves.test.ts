import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import fixture from '../data/fixtures/snapshot.json';
import { parseEnvelopeDemo, parseSnapshot } from '../data/parse';
import { FixtureProvider, LiveProvider } from '../data/provider';
import { createCommands } from '../commands/registry';
import { EnvelopeRail, railStages } from './EnvelopeRail';
import PlayerAct from './PlayerAct';
import MyMoves from './MyMoves';
import { renderState } from './state';
import { clockDom } from '../clock/testDom';

const demoImports = vi.fn();
vi.mock('../data/fixtures/envelope-demo.json', () => {
  demoImports();
  return { default: JSON.parse(readFileSync('src/app/data/fixtures/envelope-demo.json', 'utf8')) };
});
const snapshot = parseSnapshot(fixture);
const receipt = parseEnvelopeDemo(JSON.parse(readFileSync(
  'src/app/data/fixtures/envelope-demo.json', 'utf8',
))).receipt;
const subject = { kind: 'player' as const, id: '11675', franchiseId: '0001' };
const serverExecutor = (executor: ReturnType<typeof createCommands>) => ({
  ...executor,
  use: () => renderState(executor.read()),
  useMoves: <T,>(selector: (state: ReturnType<typeof executor.readMoves>) => T) =>
    selector(executor.readMoves()),
});
const render = (executor: ReturnType<typeof createCommands>) =>
  renderToStaticMarkup(createElement(MyMoves, { snapshot, executor: serverExecutor(executor) }));

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
afterEach(() => {
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});

describe('move rail', () => {
  it('renders the real demo in order with local times, notes and landed current', () => {
    expect(railStages(receipt)).toEqual(['draft', 'ready', 'handed_off', 'not_yet_done', 'landed']);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt }));
    expect(html).toContain(snapshot.players.value.find((p) => p.id === '11675')!.name);
    expect(html).toContain('IR placement');
    expect(html).toContain('G2 · reversible');
    expect(html).toMatch(/aria-current="step"[^>]*><span[^>]*>Landed/);
    let index = -1;
    for (const label of ['Draft', 'Ready', 'Handed off', 'Not yet done', 'Landed']) {
      const next = html.indexOf(`>${label}<`);
      expect(next).toBeGreaterThan(index);
      index = next;
    }
    for (const entry of receipt.audit) {
      expect(html).toContain(entry.note);
      expect(html).toContain(`dateTime="${entry.at}"`);
      expect(html).toContain(new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' })
        .format(new Date(entry.at)));
    }
  });
  it('ends at blocked with the exact Go note and no later stages', () => {
    const blocked = {
      ...receipt,
      state: 'blocked' as const,
      spec: { ...receipt.spec, target: { kind: 'unmapped' as const } },
      audit: [{
        at: receipt.audit[0].at, from: 'draft' as const, event: 'checks_block' as const,
        to: 'blocked' as const, note: 'MFL target not verified',
      }],
    };
    expect(railStages(blocked)).toEqual(['draft', 'blocked']);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: blocked }));
    expect(html).toContain('MFL target not verified');
    expect(html).toMatch(/aria-current="step"[^>]*><span[^>]*signal-red[^>]*>Blocked/);
    expect(html).not.toContain('>Ready<');
    expect(html).not.toContain('>Landed<');
  });
  it('inserts pending stages only when the trail visits them and mutes future stages', () => {
    const pending = {
      ...receipt, state: 'dot_review' as const,
      audit: [receipt.audit[0], {
        at: receipt.audit[1].at, from: 'ready' as const, event: 'await_dot' as const,
        to: 'dot_review' as const, note: 'Awaiting DOT',
      }],
    };
    expect(railStages(pending)).toEqual([
      'draft', 'ready', 'dot_review', 'handed_off', 'not_yet_done', 'landed',
    ]);
    const html = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: pending }));
    expect(html).toContain('Awaiting DOT');
    expect(html.match(/data-reached="false"/g)).toHaveLength(3);
    expect(html).not.toContain('Bid pending');
    expect(html).not.toContain('Waiver pending');
  });
});

describe('My moves and inspector', () => {
  it('loads the live empty list without importing demo and surfaces failures', async () => {
    const provider = new LiveProvider();
    const moves = vi.spyOn(provider, 'moves').mockResolvedValue([]);
    const executor = createCommands(() => undefined, provider);
    executor.loadSnapshot(snapshot);
    executor.dispatch('franchise.set', { franchiseId: '0001' });
    const { document, element } = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
    const root = createRoot(element);
    await act(async () => {
      root.render(createElement(MyMoves, { snapshot, executor }));
      await executor.loadMoves('0001');
    });
    expect(moves).toHaveBeenCalledWith('0001');
    expect(element.textContent).toContain('No moves drafted this session');
    expect(demoImports).not.toHaveBeenCalled();
    expect(element.textContent).not.toContain('Fixture');
    moves.mockRejectedValue(new Error('binding unavailable'));
    await act(async () => { await executor.loadMoves('0001'); });
    expect(element.textContent).toContain('binding unavailable');
    expect(element.textContent).not.toContain('No moves drafted this session');
    await act(async () => { root.unmount(); });
  });
  it('shows only the newest fifteen rails with an older-moves count, without paging', async () => {
    const provider = new LiveProvider();
    vi.spyOn(provider, 'moves').mockResolvedValue(Array.from({ length: 31 }, (_, index) => ({
      ...receipt, correlationId: `move-${index}`,
      audit: receipt.audit.map((entry) => ({
        ...entry, at: `2026-10-06T00:00:${String(index).padStart(2, '0')}Z`,
      })),
    })));
    const executor = createCommands(() => undefined, provider);
    executor.loadSnapshot(snapshot);
    executor.dispatch('franchise.set', { franchiseId: '0001' });
    await executor.loadMoves('0001');
    expect(executor.readMoves().moves[0].correlationId).toBe('move-30');
    const html = render(executor);
    expect(html.match(/aria-label="Move stages"/g)).toHaveLength(15);
    expect(html).toContain('16 older moves this session not shown');
    expect(html).not.toContain('Next moves');
    expect(executor.registry).not.toHaveProperty('moves.page');
    expect(executor.read()).not.toHaveProperty('movesPage');
  });
  it('shows Drafting immediately, then shows a rejected binding in the inspector', async () => {
    const provider = new LiveProvider();
    let reject!: (cause: Error) => void;
    let started!: () => void;
    const fired = new Promise<void>((resolve) => { started = resolve; });
    vi.spyOn(provider, 'draftIR').mockImplementation(() =>
      new Promise<typeof receipt>((_resolve, fail) => {
        reject = fail;
        started();
      }));
    const executor = createCommands(() => undefined, provider);
    executor.loadSnapshot(snapshot);
    executor.dispatch('franchise.set', { franchiseId: '0001' });
    const { document, element } = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
    const root = createRoot(element);
    await act(async () => { root.render(createElement(PlayerAct, { subject, executor })); });
    act(() => { executor.dispatch('roster.ir', { subject }); });
    expect(element.textContent).toContain('Drafting…');
    expect(element.querySelector('[disabled]')).toBeDefined();
    await act(async () => { await fired; });
    await act(async () => { reject(new Error('draft rejected')); });
    expect(element.textContent).toContain('draft rejected');
    expect(element.textContent).not.toContain('Drafting…');
    await act(async () => { root.unmount(); });
  });
  it('labels the fixture landing as simulated and never offers an enabled draft', async () => {
    const executor = createCommands(() => undefined, new FixtureProvider());
    executor.loadSnapshot(snapshot);
    await executor.loadMoves('0001');
    const html = render(executor);
    expect(demoImports).toHaveBeenCalledTimes(1);
    expect(html).toContain(
      'Fixture · demo envelope · landing simulated from a real roster snapshot',
    );
    expect(html).toContain('>Landed<');
    const inspector = renderToStaticMarkup(createElement(PlayerAct, {
      subject, executor: serverExecutor(executor),
    }));
    expect(inspector).toContain('disabled=""');
    expect(inspector).toContain('Drafting needs the desktop app');
    expect(inspector).toContain('My moves');
  });
  it('renders not-my-franchise disabled with the reason beside the control', () => {
    const executor = createCommands(() => undefined, new LiveProvider());
    executor.loadSnapshot(snapshot);
    executor.dispatch('franchise.set', { franchiseId: '0002' });
    const html = renderToStaticMarkup(createElement(PlayerAct, {
      subject, executor: serverExecutor(executor),
    }));
    expect(html).toContain('disabled=""');
    expect(html).toContain('<p>Only players on my franchise</p>');
  });
});

describe('Inspector IR and taxi Acts', () => {
  function inspector(executor: ReturnType<typeof createCommands>) {
    return renderToStaticMarkup(createElement(PlayerAct, {
      subject, executor: serverExecutor(executor),
    }));
  }
  function held(status: 'ROSTER' | 'IR' | 'TAXI_SQUAD') {
    const value = structuredClone(snapshot);
    const player = value.rosters.value.find((r) => r.franchiseId === subject.franchiseId)!
      .players.find((p) => p.id === subject.id)!;
    player.rosterStatus = status;
    const provider = new LiveProvider();
    const executor = createCommands(() => undefined, provider);
    executor.loadSnapshot(value);
    executor.dispatch('franchise.set', { franchiseId: subject.franchiseId });
    return { executor, provider };
  }
  it.each([
    ['ROSTER', 'Draft move to taxi squad', true],
    ['TAXI_SQUAD', 'Draft promotion from taxi', false],
  ] as const)('names the direction from held %s', (status, label, irAllowed) => {
    const { executor } = held(status);
    const html = inspector(executor);
    expect(html).toContain(`>${label}</button>`);
    expect(html.includes('>Draft IR placement</button>')).toBe(irAllowed);
  });
  it('does not offer taxi for an IR player', () => {
    const html = inspector(held('IR').executor);
    expect(html).not.toContain('Draft move to taxi squad');
    expect(html).not.toContain('Draft promotion from taxi');
  });
  it.each([
    ['roster.ir', 'IR', 'Open MFL IR page', 'IR '],
    ['roster.taxi', 'TAXI_SQUAD', 'Open MFL taxi page', '+Taxi '],
    ['roster.taxi', 'ROSTER', 'Open MFL taxi page', '−Taxi '],
  ] as const)('shows %s %s hand-off and summary as visible text', async (intent, status, label, prefix) => {
    const { executor, provider } = held('ROSTER');
    const ready = {
      ...receipt, state: 'ready' as const, audit: receipt.audit.slice(0, 1),
      spec: { ...receipt.spec, intent, expected: { player: subject.id, rosterStatus: status } },
    };
    vi.spyOn(provider, 'moves').mockResolvedValue([ready]);
    await executor.loadMoves(subject.franchiseId);
    expect(inspector(executor)).toContain(`>${label}</button>`);
    const rail = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: ready }));
    expect(rail).toContain(`>${label}</button>`);
    const name = snapshot.players.value.find((p) => p.id === subject.id)!.name;
    expect(rail).toContain(`<p>${prefix}${name}</p>`);
    expect(rail).not.toContain('roster.taxi');
  });
  it.each(['roster.ir', 'roster.taxi'])('shows the %s block note, never a hand-off', async (intent) => {
    const { executor, provider } = held('ROSTER');
    const blocked = {
      ...receipt, state: 'blocked' as const, spec: { ...receipt.spec, intent },
      audit: [{ ...receipt.audit[0], to: 'blocked' as const, note: 'MFL eligibility not held' }],
    };
    vi.spyOn(provider, 'moves').mockResolvedValue([blocked]);
    await executor.loadMoves(subject.franchiseId);
    const html = inspector(executor);
    expect(html).toContain('MFL eligibility not held');
    expect(html).not.toContain('Open MFL');
    const rail = renderToStaticMarkup(createElement(EnvelopeRail, { snapshot, receipt: blocked }));
    expect(rail).not.toContain('Open MFL');
  });
  it('shows opening and a hand-off failure inline, and excludes unrelated lineup receipts', async () => {
    const { executor, provider } = held('ROSTER');
    const ready = { ...receipt, state: 'ready' as const, audit: receipt.audit.slice(0, 1) };
    const unrelated = { ...ready, correlationId: 'lineup', spec: { ...ready.spec, intent: 'lineup.set' } };
    vi.spyOn(provider, 'moves').mockResolvedValue([unrelated, ready]);
    let reject!: (cause: Error) => void;
    const handoff = vi.spyOn(provider, 'handOff').mockImplementation(() =>
      new Promise<typeof receipt>((_done, fail) => {
        reject = fail;
      }));
    await executor.loadMoves(subject.franchiseId);
    expect(inspector(executor)).toContain('>Open MFL IR page</button>');
    executor.dispatch('move.handoff', { correlationId: ready.correlationId });
    for (let attempt = 0; attempt < 100 && !handoff.mock.calls.length; attempt += 1) {
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
    expect(handoff).toHaveBeenCalledWith(ready.correlationId);
    expect(inspector(executor)).toContain('>Opening MFL…</button>');
    expect(inspector(executor)).toMatch(/disabled=""[^>]*>Opening MFL…/);
    reject(new Error('MFL page could not open'));
    for (let attempt = 0; attempt < 100 && executor.readMoves().drafting[
      `handoff:${ready.correlationId}`
    ]; attempt += 1) {
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
    expect(inspector(executor)).toContain('<p role="alert">Error: MFL page could not open</p>');
    expect(inspector(executor)).not.toContain('Opening MFL…');
  });
});
