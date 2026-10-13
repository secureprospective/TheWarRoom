import { createCommands } from '../commands/registry';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { clockDom } from '../clock/testDom';
import { railStages, type RailReceipt } from '../shell/EnvelopeRail';
import { BidPlanCard } from './BidPlans';
import { BidForm } from './BidForm';
import { OpenPlans, PoolFrame, PoolScreen, PoolInspectorView } from './Pool';
import type { BidFormState, Plan } from './contract';
import { parseBoard, parsePool, parsePreview, type PoolProvider } from './provider';
import { createPoolState, poolState } from './state';
import wire from './wire.testdata.json';

const pool = () => parsePool(wire.pool);
const board = () => parseBoard(wire.board);
const plan = (code: string) => board().value.plans.find((p) => p.stateCode === code)!;
const html = (node: Parameters<typeof renderToStaticMarkup>[0]) => renderToStaticMarkup(node);
const tick = () => new Promise((resolve) => setTimeout(resolve, 5));
const form: BidFormState = {
  planId: '', playerId: '13594', playerName: 'Jones, Amy', salary: '1', years: 1, snipe: false, beat: '',
  evidence: '', closesOpen: false, closesDate: '', closesTime: '',
};

function fakeProvider(overrides: Partial<PoolProvider> = {}) {
  const calls: string[] = [];
  const log = <T>(name: string, value: T) => async (...args: unknown[]): Promise<T> => {
    calls.push(`${name}:${JSON.stringify(args)}`);
    return value;
  };
  const provider: PoolProvider = {
    refresh: log('refresh', undefined) as PoolProvider['refresh'],
    pool: log('pool', pool()) as PoolProvider['pool'],
    plans: log('plans', board()) as PoolProvider['plans'],
    preview: log('preview', parsePreview(wire.preview)) as PoolProvider['preview'],
    draft: log('draft', plan('ready')) as PoolProvider['draft'],
    revise: log('revise', plan('ready')) as PoolProvider['revise'],
    handOff: log('handOff', plan('ready')) as PoolProvider['handOff'],
    posted: log('posted', plan('ready')) as PoolProvider['posted'],
    withdraw: log('withdraw', plan('withdrawn')) as PoolProvider['withdraw'],
    ...overrides,
  };
  return { provider, calls };
}

// A bid plan walked along the given states, for the lifecycle strip.
function walk(path: string[], intent = 'bid.submit'): RailReceipt {
  const states = path as RailReceipt['state'][];
  return {
    state: states.at(-1)!, spec: { intent },
    audit: states.slice(1).map((to, i) => ({
      at: `2026-10-10T12:0${i}:00Z`, from: states[i], to, note: `${to} note.`,
    })),
  };
}

describe('Fix 1: open plans sit above the table', () => {
  const screen = (patch: Partial<ReturnType<typeof poolState.peek>> = {}) => html(createElement(PoolScreen, {
    state: { ...poolState.peek(), reading: pool(), board: board(), ...patch }, franchiseId: '0001',
  }));
  it('lists each open plan on one line with player, contract, points and state', () => {
    const text = screen();
    expect(text.indexOf('Your open bid plans')).toBeLessThan(text.indexOf('<table'));
    const lines = text.split('<button').filter((part) => part.includes('aria-label="Open bid plan: '));
    expect(lines).toHaveLength(2);
    const smith = lines.find((line) => line.includes('Smith, John'))!;
    for (const piece of ['Smith, John (WR, DAL)', '3 years at $5,000,000 a year', '7.000 points', 'Ready'])
      expect(smith).toContain(piece);
    expect(lines.join('')).not.toContain('Withdrawn');
    expect(text).not.toContain('Finished bid plans');
  });
  it('opens the chosen card above the table, once', () => {
    const closed = screen();
    expect(closed).not.toContain('aria-label="Bid plan Smith, John');
    const text = screen({ openPlan: plan('ready').id });
    const card = text.indexOf('aria-label="Bid plan Smith, John');
    expect(card).toBeGreaterThan(-1);
    expect(card).toBeLessThan(text.indexOf('<table'));
    expect(text.split('aria-label="Bid plan Smith, John').length).toBe(2);
    expect(text).toContain('aria-expanded="true"');
  });
  it('says so when there is no open plan', () => {
    const none = { ...board(), value: { plans: [], summary: '0 open of 0 bid plans' } };
    const focus = { id: '', seq: 0 };
    const text = html(createElement(OpenPlans, { board: none, openId: '', focus, busy: '' }));
    expect(text).toContain('No open bid plans. Choose a player, then Plan a bid in the Inspector.');
  });
  it('G: saves inline without a reveal jump; explicit card opens still reveal', async () => {
    const { provider } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    await state.change({ field: 'pool-plan', value: '13594' });
    await state.change({ field: 'pool-save', value: '' });
    expect(state.peek().form).toBeUndefined();
    expect(state.peek().openPlan).toBe(plan('ready').id);
    expect(state.peek().focus).toEqual({ id: '', seq: 0 });
    await state.change({ field: 'pool-open', value: plan('blocked').id });
    expect(state.peek()).toMatchObject({ openPlan: plan('blocked').id, focus: { seq: 1 } });
    await state.change({ field: 'pool-toggle', value: plan('blocked').id });
    expect(state.peek().openPlan).toBe('');
    await state.change({ field: 'pool-toggle', value: plan('blocked').id });
    expect(state.peek()).toMatchObject({ openPlan: plan('blocked').id, focus: { seq: 2 } });
  });
});

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));

describe('Fix 1: the screen scrolls the card into view', () => {
  afterEach(() => {
    for (const { key, descriptor } of globals) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor);
      else Reflect.deleteProperty(globalThis, key);
    }
  });
  it('scrolls when asked, not on every render', async () => {
    const { document, element } = clockDom([]);
    vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
    vi.stubGlobal('document', document);
    vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
    const scrolled = vi.fn();
    const proto = Object.getPrototypeOf(element);
    const scroll = { scrollTop: 0, clientTop: 0, clientHeight: 700,
      getBoundingClientRect: () => ({ top: 0 }) };
    proto.closest = () => scroll;
    proto.getBoundingClientRect = scrolled.mockReturnValue({ top: 400 });
    const root = createRoot(element);
    const id = plan('ready').id;
    const show = (seq: number) => createElement(OpenPlans, {
      board: board(), openId: id, focus: { id, seq }, busy: '',
    });
    await act(async () => { root.render(show(1)); });
    expect(scrolled).toHaveBeenCalledTimes(1);
    await act(async () => { root.render(show(1)); });
    expect(scrolled).toHaveBeenCalledTimes(1);
    await act(async () => { root.render(show(2)); });
    expect(scrolled).toHaveBeenCalledTimes(2);
    delete proto.closest;
    delete proto.getBoundingClientRect;
  });
});

describe('Fix 2: one plan per player', () => {
  it('offers Open bid plan on a row with a plan, and Plan a bid on one without', () => {
    const text = html(createElement(PoolFrame, { reading: pool(), refreshing: false }));
    expect(text).toContain('Open bid plan on Smith, John');
    expect(text).toContain('>Open bid plan<');
    expect(text).not.toContain('Plan a bid on Smith, John');
    expect(text).toContain('Plan a bid on Jones, Amy');
    expect(text).not.toContain('Open bid plan on Jones, Amy');
  });
  it('opens the held plan instead of a second form, even before the plan list has loaded', async () => {
    const { provider, calls } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    await state.change({ field: 'pool-plan', value: '13593' });
    expect(state.peek().form).toBeUndefined();
    expect(state.peek().openPlan).toBe(plan('ready').id);
    expect(calls.some((c) => c.startsWith('preview'))).toBe(false);
    const early = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await early.open();
    await early.change({ field: 'pool-plan', value: '13595' });
    expect(early.peek().form).toBeUndefined();
    expect(early.peek().openPlan).toBe(plan('blocked').id);
  });
});

describe('Fix 3: taking back a plan that was never posted', () => {
  const handedOff = (): Plan => ({
    ...plan('ready'), stateCode: 'handed_off', state: 'Handed off', rules: [],
    receipt: walk(['draft', 'ready', 'handed_off']),
    actions: [
      { id: 'copy', label: 'Copy post again', disabled: false, reason: '' },
      { id: 'posted', label: 'I posted it', disabled: false, reason: '' },
      { id: 'unposted', label: 'I didn\'t post it', disabled: false, reason: '' },
    ],
  });
  it('shows I didn\'t post it on a handed-off plan, and nothing to retract a posted bid', () => {
    const text = html(createElement(BidPlanCard, { plan: handedOff() }));
    expect(text).toContain('I didn&#x27;t post it');
    expect(text).toContain('I posted it');
    const pending = { ...handedOff(), stateCode: 'bid_pending', actions: handedOff().actions.slice(0, 1) };
    const after = html(createElement(BidPlanCard, { plan: pending }));
    expect(after).not.toContain('post it');
    expect(after).not.toContain('Withdraw');
  });
  it('sends it to Go\'s withdrawal and shows Go\'s words', async () => {
    const { provider, calls } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    await state.change({ field: 'pool-unposted', value: 'bid-x' });
    expect(calls.some((c) => c === 'withdraw:["bid-x"]')).toBe(true);
    expect(state.peek().notice).toBe('');
  });
  it('draws the withdrawal as the end of the plan, in the strip and in History', () => {
    const gone = plan('withdrawn');
    const text = html(createElement(BidPlanCard, { plan: gone }));
    expect(text).toMatch(/aria-current="step"[^>]*>(?:(?!<li).)*Withdrawn/s);
    expect(text).toContain('did not post it');
    expect(text).toContain('History');
    expect(text).not.toContain('class="act');
    expect(gone.receipt.audit.at(-1)).toMatchObject({ from: 'handed_off', to: 'withdrawn' });
  });
});

describe('Fix 4: passing rules fold into Go\'s line', () => {
  it('shows failures and notes in full, and one disclosure for the rest', () => {
    const text = html(createElement(BidPlanCard, { plan: plan('blocked') }));
    const fold = text.indexOf('<details class="bid-pass"><summary>');
    expect(fold).toBeGreaterThan(-1);
    expect(text).toContain(`<summary>${plan('blocked').ruleSummary}</summary>`);
    expect(text.indexOf('data-verdict="unknown"')).toBeLessThan(fold);
    expect(text.indexOf('data-verdict="info"')).toBeLessThan(fold);
    expect(text.indexOf('data-verdict="pass"')).toBeGreaterThan(fold);
    expect(text).not.toContain('<details class="bid-pass" open');
  });
  it('writes the summary in Go, for the plan and for the form\'s preview', () => {
    expect(plan('ready').ruleSummary).toBe('10 rules pass');
    expect(parsePreview(wire.preview).ruleSummary).toBe('10 rules pass');
    const text = html(createElement(BidForm, { form, preview: parsePreview(wire.preview), busy: '' }));
    expect(text).toContain('<summary>10 rules pass</summary>');
    expect(text.split('<li data-verdict="pass"').length - 1).toBe(10);
    expect(text.indexOf('data-verdict="info"')).toBeLessThan(text.indexOf('<summary>10 rules pass'));
  });
});

describe('Fix 5: unknown age and experience', () => {
  it('shows a dash for the age and a plain phrase for the experience, in separate cells', () => {
    const text = html(createElement(PoolFrame, { reading: pool(), refreshing: false }));
    expect(text).toContain('No score');
    expect(text).toContain('Not valued');
    expect(text).not.toContain('UnknownUnknown');
  });
  it('confines the six-column table to its work column, with a sticky player fallback', () => {
    const css = readFileSync('src/app/pool/pool.css', 'utf8');
    expect(css).toContain('min-width: 920px');
    expect(css).not.toContain('min-width: 1600px');
    expect(css).toContain('.pool, .twr-app .pool-reading { min-width: 0; max-width: 100%; }');
    expect(css).toContain('.pool-table-scroll { min-width: 0; overflow: auto; max-width: 100%;');
    expect(css.replace(/\s+/g, ' '))
      .toContain('position: sticky; left: 0; background: var(--surface-0)');
  });
});

describe('Fix 6: Bidding closes reads as empty until it is set', () => {
  const preview = parsePreview(wire.preview);
  it('offers a button, no date box, and Go\'s words about what the app does', () => {
    const text = html(createElement(BidForm, { form, preview, busy: '' }));
    expect(text).toContain('Set a closing time');
    expect(text).not.toContain('datetime-local');
    expect(text).not.toContain('type="date"');
    expect(text).toContain(preview.closesNote);
    expect(preview.closesNote).toContain('No closing time is set');
  });
  it('reveals the date and time with Clear once asked', async () => {
    const open = { ...form, closesOpen: true, closesDate: '2026-10-10', closesTime: '20:30' };
    const text = html(createElement(BidForm, { form: open, preview, busy: '' }));
    for (const piece of ['type="date"', 'type="time"', 'value="2026-10-10"', 'value="20:30"', 'Clear'])
      expect(text).toContain(piece);
    expect(text).not.toContain('Set a closing time');
    const { provider } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.change({ field: 'pool-plan', value: '13594' });
    expect(state.peek().form?.closesOpen).toBe(false);
    await state.change({ field: 'pool-closes-set', value: '' });
    state.edit('closesDate', '2026-10-10');
    state.edit('closesTime', '20:30');
    expect(state.peek().form).toMatchObject({
      closesOpen: true, closesDate: '2026-10-10', closesTime: '20:30',
    });
    await state.change({ field: 'pool-closes-clear', value: '' });
    expect(state.peek().form).toMatchObject({ closesOpen: false, closesDate: '', closesTime: '' });
  });
  it('shows a saved closing time again when the plan is changed', async () => {
    const saved = { ...plan('ready'), closes: '2026-10-11T01:30:00Z' };
    const { provider } = fakeProvider({ plans: async () => ({ ...board(), value: { ...board().value,
      plans: [saved] } }) });
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    await state.change({ field: 'pool-revise', value: saved.id });
    const shown = state.peek().form!;
    expect(shown.closesOpen).toBe(true);
    expect(shown.closesDate).toMatch(/^\d{4}-\d\d-\d\d$/);
    expect(shown.closesTime).toMatch(/^\d\d:\d\d$/);
  });
});

describe('Fix 7: the bid card uses the move rail\'s strip', () => {
  it('draws the bid steps, with time and note under each reached step', () => {
    const text = html(createElement(BidPlanCard, { plan: plan('ready') }));
    expect(text).toContain('aria-label="Move stages"');
    const labels = ['>Draft<', '>Ready<', '>Handed off<', '>Bid pending<', '>Landed<'];
    const at = labels.map((label) => text.indexOf(label));
    expect([...at].sort((a, b) => a - b)).toEqual(at);
    expect(at.every((index) => index > -1)).toBe(true);
    expect(text).toMatch(
      /aria-current="step"[^>]*><span[^>]*>Ready<\/span><time[^>]*>[^<]+<\/time><p>Bid rules/,
    );
    expect(text.match(/data-reached="false"/g)).toHaveLength(3);
  });
  it('follows the bid through posting to each ending', () => {
    const base = ['draft', 'ready', 'handed_off'];
    expect(railStages(walk([...base, 'bid_pending', 'landed']))).toEqual([...base, 'bid_pending', 'landed']);
    expect(railStages(walk([...base, 'bid_pending', 'awarded_off_app']))).toEqual(
      [...base, 'bid_pending', 'awarded_off_app']);
    expect(railStages(walk([...base, 'bid_pending', 'not_verified']))).toEqual(
      [...base, 'bid_pending', 'not_verified']);
    expect(railStages(walk([...base, 'withdrawn']))).toEqual([...base, 'withdrawn']);
    expect(railStages(walk(['draft', 'ready', 'withdrawn']))).toEqual(['draft', 'ready', 'withdrawn']);
    expect(railStages(walk([...base, 'not_yet_done', 'bid_pending']))).toEqual(
      [...base, 'not_yet_done', 'bid_pending', 'landed']);
    expect(railStages(walk(['draft', 'blocked']))).toEqual(['draft', 'blocked']);
    expect(railStages(walk(['draft', 'ready', 'handed_off', 'not_yet_done', 'landed'], 'roster.ir')))
      .toEqual(['draft', 'ready', 'handed_off', 'not_yet_done', 'landed']);
  });
  it('reuses the rail component rather than copying it', () => {
    const card = readFileSync('src/app/pool/BidPlans.tsx', 'utf8');
    expect(card).toContain("import { RailStrip } from '../shell/EnvelopeRail'");
    expect(card).not.toContain('envelope-rail');
    const rail = readFileSync('src/app/shell/EnvelopeRail.tsx', 'utf8');
    expect(rail.match(/<ol /g)).toHaveLength(1);
  });
});

describe('Fix 8: Go\'s wording reaches the screen', () => {
  it('reads the rule sentences and status lines as Go wrote them', () => {
    const words = (code: string, rule: string) =>
      plan(code).rules.find((r) => r.rule === rule)?.reason;
    expect(words('ready', 'flat-salary')).toBe('$5,000,000 a year, the same in each of the 3 years');
    expect(words('blocked', 'flat-salary')).toBe('$2,000,000 a year for its 1 year');
    expect(words('ready', 'tier-years')).toBe('Up to $12,000,000 needs at least 1 year');
    for (const code of ['ready', 'blocked', 'withdrawn']) {
      expect(plan(code).status.endsWith('.')).toBe(true);
      for (const entry of plan(code).history) expect(entry.note.endsWith('.')).toBe(true);
    }
  });
});

it('320px Inspector uses a compact plan with state, time and a card route, never a lifecycle strip', () => {
  const ready = plan('ready');
  const text = html(createElement(PoolInspectorView, { plans: [ready], busy: '' }));
  expect(text).toContain(ready.state);
  expect(text).toContain('<time');
  expect(text).toContain(ready.contract);
  expect(text).toContain('See bid plan');
  expect(text).not.toContain('Move stages');
  expect(text).not.toContain('bid-post');
  const code = readFileSync('src/app/pool/pool.css', 'utf8');
  expect(code).toContain('word-break: normal');
});

it('withdraw does not leave a duplicate pool notice', async () => {
  const { provider } = fakeProvider();
  const state = createPoolState({ provider, franchise: () => '0001', copy: async () => true });
  await state.open();
  await state.change({ field: 'pool-withdraw', value: plan('ready').id });
  expect(state.peek().notice).toBe('');
});

it('Inspector card link and plan-a-bid both navigate to the pool work column', () => {
  for (const field of ['pool-open', 'pool-plan'] as const) {
    const c = createCommands(() => undefined);
    c.dispatch('nav.open', { node: 'hq', workspace: 'my-moves' });
    c.dispatch('market', { field, value: 'bid-1' });
    expect(c.read()).toMatchObject({ node: 'war', workspace: { war: 'valuations-and-pool' } });
  }
});

it('pool form insertion uses the same row anchor as Contracts', () => {
  const source = readFileSync('src/app/pool/Pool.tsx', 'utf8');
  expect(source).toContain("useWorkAnchor(root, '.pool-table tbody tr:not(.pool-form-row), .bid-plan')");
});

it('the bid preview stays in the work column, outside the overflowing pool table', () => {
  const reading = pool();
  const row = reading.value.rows[1];
  const text = html(createElement(PoolScreen, { franchiseId: '0001', state: {
    ...poolState.peek(), reading, form: { ...form, playerId: row.id, playerName: row.name },
  } }));
  expect(text.match(/aria-label="Bid plan form"/g)).toHaveLength(1);
  expect(text.indexOf('Bid plan form')).toBeLessThan(text.indexOf('<table'));
  expect(text).not.toContain('<tr class="pool-form-row">');
});
it('I: terminal chosen plans stay out of the pool, including the finished list', () => {
  const chosen = plan('withdrawn');
  const reading = { ...board(), value: { ...board().value, plans: [chosen] } };
  const text = html(createElement(PoolScreen, { franchiseId: '0001', state: {
    ...poolState.peek(), reading: pool(), board: reading, openPlan: chosen.id,
  } }));
  expect(text).toContain('No open bid plans. Choose a player, then Plan a bid in the Inspector.');
  expect(text).not.toContain(`aria-label="Bid plan ${chosen.player}`);
  expect(text).not.toContain('Finished bid plans');
});
it('H: free-agent Inspector uses its normal 14px block rhythm', () => {
  const css = readFileSync('src/app/pool/pool.css', 'utf8');
  expect(css).toContain('.twr-app .pool-inspector { display: grid; gap: 14px; }');
});

it('G: saved Change figures keeps the same form beside its card, not in the distant pool table', () => {
  const chosen = plan('ready');
  const text = html(createElement(PoolScreen, { franchiseId: '0001', state: {
    ...poolState.peek(), reading: pool(), board: board(), openPlan: chosen.id,
    form: { ...form, planId: chosen.id, playerId: chosen.playerId },
  } }));
  expect(text.match(/aria-label="Bid plan form"/g)).toHaveLength(1);
  const card = text.indexOf(`aria-label="Bid plan ${chosen.player}`);
  expect(text.indexOf('Bid plan form')).toBeGreaterThan(card);
  expect(text.indexOf('Bid plan form')).toBeLessThan(text.indexOf('<table'));
  expect(readFileSync('src/app/pool/pool.css', 'utf8'))
    .toContain('.pool-table tr:not(.pool-form-row) td:nth-child(1) .act');
});


it('keeps MFL market lines under the player and the minimum bid, ranges only in the Inspector', () => {
  const reading = pool();
  const row = { ...reading.value.rows[0],
    marketLines: ['Proj 12.4 · Rank 45, Up 3', 'ADP 112 (88–140) · AAV 1.2%'] };
  const text = html(createElement(PoolFrame, {
    reading: { ...reading, value: { ...reading.value, rows: [row] } }, refreshing: false,
  }));
  expect(text.match(/<th[ >]/g)).toHaveLength(8);
  expect(text).toContain('Sort by Lab dynasty value');
  expect(text).toContain('Sort by Minimum bid');
  expect(text).toContain(row.marketLines.join(' · '));
  expect(text).toContain(row.minimum.text);
  expect(text).not.toContain('<svg');
});

it('orders intro, open plans, pool and reports in DOM order', () => {
  const source = readFileSync('src/app/pool/Pool.tsx', 'utf8');
  const screen = source.slice(
    source.indexOf('export function PoolScreen'), source.indexOf('export default function Pool'),
  );
  const elements = ['<header>', '<OpenPlans', '<Controls', '<PoolFrame', '<MarketReports'];
  const indices = elements.map(s => screen.indexOf(s));
  expect(indices.every(i => i >= 0)).toBe(true);
  expect(indices).toEqual([...indices].sort((a, b) => a - b));
});

it('offers all seven Go-supplied sorts and sends each market key and direction to Go', async () => {
  const reading = pool();
  const state = { ...poolState.peek(), reading };
  const text = html(createElement(PoolScreen, { state, franchiseId: '0001' }));
  for (const { key, label } of reading.value.sorts) {
    const selected = key === 'engine' ? ' selected=""' : '';
    expect(text).toContain(`<option value="${key}"${selected}>${label}</option>`);
  }
  expect(reading.value.sorts.map(s => s.key))
    .toEqual(['name', 'position', 'team', 'engine', 'ppg', 'lab', 'minimum',
      'projected', 'rank', 'adp', 'aav']);
  const { provider, calls } = fakeProvider();
  const store = createPoolState({ provider, franchise: () => '0001', delay: 0 });
  await store.open();
  for (const key of ['projected', 'rank', 'adp', 'aav']) {
    await store.change({ field: 'pool-sort', value: key });
    expect(calls.at(-1)).toContain(`"sort":"${key}","desc":false`);
    await store.change({ field: 'pool-sort', value: key });
    expect(calls.at(-1)).toContain(`"sort":"${key}","desc":true`);
  }
});

it('renders salary and points above the overflow region with a bounded preview value column', () => {
  const preview = parsePreview(wire.preview);
  const text = html(createElement(PoolScreen, { franchiseId: '0001', state: {
    ...poolState.peek(), reading: pool(), form, preview,
  } }));
  const start = text.indexOf('class="bid-preview"');
  const end = text.indexOf('</dl>', start);
  expect(start).toBeGreaterThan(-1);
  expect(end).toBeLessThan(text.indexOf('class="pool-table-scroll"'));
  expect(text.slice(start, end)).toContain(preview.salary.text);
  expect(text.slice(start, end)).toContain(preview.points);
  const css = readFileSync('src/app/pool/pool.css', 'utf8');
  expect(css).toMatch(/\.bid-preview \.kv \{ grid-template-columns: minmax\(0, 1fr\) auto;/);
});
