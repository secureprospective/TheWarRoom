import { describe, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import { endpointById } from '../registry';
import { placementRoute } from '../registry';
import { set as marketSet } from '../market/Market';
import { BidBoardView, BidPlanCard } from './BidPlans';
import { BidForm } from './BidForm';
import { PoolFrame, PoolInspectorView, PoolScreen } from './Pool';
import type { BidFormState, BoardReading, PoolReading } from './contract';
import { parseBoard, parsePlan, parsePool, parsePreview, rawBid, type PoolProvider } from './provider';
import { createPoolState, localInput, poolState } from './state';
import wire from './wire.testdata.json';

const pool = () => parsePool(wire.pool);
const board = () => parseBoard(wire.board);
const html = (node: Parameters<typeof renderToStaticMarkup>[0]) => renderToStaticMarkup(node);
const form: BidFormState = {
  planId: '', playerId: '13593', playerName: 'Smith, John', salary: '5.004', years: 3,
  snipe: false, beat: '', evidence: '', closesOpen: false, closesDate: '', closesTime: '',
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
    withdraw: log('withdraw', plan('ready')) as PoolProvider['withdraw'],
    ...overrides,
  };
  return { provider, calls };
}
const plan = (code: string) => board().value.plans.find((p) => p.stateCode === code)!;
const button = (markup: string, label: string) =>
  markup.split('<button').find((part) => part.includes(`aria-label="${label}"`)) ?? '';
const tick = () => new Promise((resolve) => setTimeout(resolve, 5));

describe('Go wire', () => {
  it('parses what Go sends: pool, empty pool, plans and previews', () => {
    expect(pool().status).toBe('ok');
    expect(pool().value.rows.map((r) => r.name)).toEqual(['Adams, Zed', 'Jones, Amy', 'Smith, John']);
    expect(parsePool(wire.empty).status).toBe('failed');
    expect(board().value.plans.map((p) => p.state)).toEqual(['Withdrawn', 'Blocked', 'Ready']);
    expect(plan('blocked').id).not.toBe(plan('ready').id);
    expect(parsePreview(wire.preview).rules).toHaveLength(11);
    expect(parsePreview(wire.problem).problem).toContain('Enter the annual salary in millions');
  });
  it('refuses a wire that drifted', () => {
    expect(() => parsePool({ ...wire.pool, extra: 1 })).toThrow(/unexpected field/);
    expect(() => parsePool({ ...wire.pool, status: 'thin', absenceReason: null })).toThrow(/absence reason/);
    expect(() => parsePlan({ ...wire.board.value.plans[0], extra: 1 })).toThrow(/unexpected field/);
    expect(() => parsePreview({ ...wire.preview, rules: [{ rule: 'x' }] })).toThrow();
  });
});

describe('Pool reading states', () => {
  const frame = (reading?: PoolReading, refreshing = false) =>
    html(createElement(PoolFrame, { reading, refreshing }));
  it('shows the table, source, as-of and how it is counted', () => {
    const text = frame(pool());
    for (const phrase of ['Adams, Zed', 'Smith, John', 'Plan a bid on Jones, Amy',
      'Open bid plan on Adams, Zed',
      'Source: MFL freeAgents', 'As of:', 'How this is counted', 'data-endpoint="M-056"',
      'Sort by Player', 'Sort by Pos', 'Sort by NFL team', 'Sort by Engine points',
      'Sort by This season points per game', 'Sort by Lab dynasty value', 'Not valued'])
      expect(text).toContain(phrase);
  });
  it('is honest when nothing was fetched, MFL is empty, or the copy is old', () => {
    const base = pool();
    expect(frame(undefined)).toContain('Reading the free-agent list…');
    const failed: PoolReading = {
      ...base, status: 'failed', absenceReason: 'MFL free agents unavailable: offline',
    };
    expect(frame(failed)).toContain('Failed · MFL free agents unavailable: offline');
    expect(frame(failed)).toContain('Try MFL again');
    expect(frame(failed, true)).toContain('Asking MFL for the free-agent list…');
    expect(frame(failed, true)).not.toContain('Failed');
    const empty: PoolReading = { ...base, status: 'thin', absenceReason: 'MFL lists no free agents right now',
      value: { rows: [], positions: [], sorts: [], summary: '0 free agents' } };
    expect(frame(empty)).toContain('Empty · MFL lists no free agents right now');
    expect(frame(empty)).not.toContain('<table');
    const none: PoolReading = { ...base, value: { ...base.value, rows: [] } };
    expect(frame(none)).toContain('No free agent matches this position.');
    const stale: PoolReading = { ...base, provenance: { ...base.provenance,
      freshness: { state: 'stale', fetchedAt: '2026-10-10T12:00:00Z', note: 'live fetch failed' } } };
    expect(frame(stale)).toContain('Stale: live fetch failed');
  });
  it('filters by Go-supplied positions and sorts by name, position or minimum', () => {
    const state = { ...poolState.peek(), reading: pool() };
    const text = html(createElement(PoolScreen, { state, franchiseId: '0001' }));
    for (const phrase of ['<option value="QB">QB</option>', '<option value="WR">WR</option>',
      '<option value="CB">CB</option>', 'All positions', '>Player</option>', '>Pos</option>',
      '>Minimum</option>', 'Refresh from MFL', 'Descending'])
      expect(text).toContain(phrase);
  });
  it('keeps pool sources free of broken words and sideways scroll', () => {
    const css = readFileSync('src/app/pool/pool.css', 'utf8');
    expect(css).not.toMatch(/break-all|anywhere|overflow-x|break-word/);
    expect(css).toContain('table-layout: fixed');
    expect(css).toMatch(/\.pool-money \{ white-space: nowrap/);
  });
});

describe('Bid form re-queries Go', () => {
  it('opens, then asks Go again after each change, keeping the last answer on screen', async () => {
    let release: (value: ReturnType<typeof parsePreview>) => void = () => undefined;
    const slow = new Promise<ReturnType<typeof parsePreview>>((resolve) => { release = resolve; });
    let calls = 0;
    const { provider, calls: log } = fakeProvider({
      preview: async (...args) => {
        log.push(`preview:${JSON.stringify(args[1].salary)}`);
        return ++calls === 2 ? slow : parsePreview(wire.preview);
      },
    });
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    expect(log.filter((c) => c.startsWith('refresh'))).toEqual(['refresh:[false]']);
    await state.change({ field: 'pool-plan', value: '13594' });
    await tick();
    expect(state.peek().form?.playerName).toBe('Jones, Amy');
    expect(state.peek().preview?.points).toBe('7.000');
    state.edit('salary', '6');
    state.edit('years', 2);
    await tick();
    expect(log.filter((c) => c.startsWith('preview'))).toEqual(['preview:""', 'preview:"6"']);
    expect(state.peek().preview?.points).toBe('7.000');
    release({ ...parsePreview(wire.preview), points: '7.200' });
    await tick();
    expect(state.peek().preview?.points).toBe('7.200');
    expect(state.peek().form).toMatchObject({ salary: '6', years: 2 });
  });
  it('saves a plan, closes the form and reloads the list; a revision goes through revise', async () => {
    const { provider, calls } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.open();
    await state.change({ field: 'pool-plan', value: '13594' });
    await state.change({ field: 'pool-save', value: '' });
    expect(calls.some((c) => c.startsWith('draft:["0001"'))).toBe(true);
    expect(state.peek().form).toBeUndefined();
    expect(state.peek().notice).toContain('Ready');
    await state.change({ field: 'pool-revise', value: plan('ready').id });
    expect(state.peek().form).toMatchObject({ salary: '5', years: 3, snipe: true, beat: '5.500' });
    await state.change({ field: 'pool-save', value: '' });
    expect(calls.some((c) => c.startsWith('revise:'))).toBe(true);
  });
  it('turns a failed act into the screen\'s error instead of a rejected promise', async () => {
    const refuse = async () => { throw new Error('the rules no longer pass'); };
    const { provider } = fakeProvider({ handOff: refuse });
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0, copy: async () => true });
    await state.open();
    await expect(state.change({ field: 'pool-handoff', value: plan('ready').id })).resolves.toBeUndefined();
    expect(state.peek().error).toContain('the rules no longer pass');
    await expect(state.change({ field: 'pool-revise', value: 'bid-unknown' })).resolves.toBeUndefined();
    expect(state.peek().error).toContain('not in the list');
  });
  it('needs a franchise before a plan can be drafted', async () => {
    const { provider } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '', delay: 0 });
    await state.change({ field: 'pool-plan', value: '13594' });
    expect(state.peek().form).toBeUndefined();
    expect(state.peek().error).toContain('Choose my franchise');
  });
  it('toggles direction on the same sort and filters by position through Go', async () => {
    const { provider, calls } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0 });
    await state.change({ field: 'pool-sort', value: 'minimum' });
    await state.change({ field: 'pool-sort', value: 'minimum' });
    await state.change({ field: 'pool-position', value: 'QB' });
    expect(state.peek().scope).toEqual({ position: 'QB', sort: 'minimum', desc: true, showNoTeam: false });
    expect(calls.filter((c) => c.startsWith('pool')).at(-1)).toContain('"position":"QB"');
  });
  it('renders the form, its verdicts, and holds Save until Go has answered', () => {
    const ready = html(createElement(BidForm, { form, preview: parsePreview(wire.preview), busy: '' }));
    for (const phrase of ['Plan a bid · Smith, John', 'Annual salary', 'Bid points', '7.000',
      'Entered $5,004,000', 'Minimum salary', 'Save bid plan', 'Snipe: this bid lands after the 20-hour mark',
      'cannot see the auction clock', 'ProBoards post preview', 'data-endpoint="M-194"',
      'data-endpoint="C-06.7"'])
      expect(ready).toContain(phrase);
    expect(button(ready, 'Save bid plan')).not.toContain('disabled');
    const waiting = html(createElement(BidForm, { form, preview: undefined, busy: '' }));
    expect(waiting).toContain('Checking the bid rules…');
    expect(button(waiting, 'Save bid plan')).toContain('disabled=""');
    const problem = html(createElement(BidForm, { form, preview: parsePreview(wire.problem), busy: '' }));
    expect(problem).toContain('Enter the annual salary in millions, such as 5.25');
    expect(button(problem, 'Save bid plan')).toContain('disabled=""');
  });
  it('sends the window close as UTC and shows it back as local time', () => {
    const set = { ...form, closesOpen: true, closesDate: '2026-10-10', closesTime: '20:30' };
    const raw = rawBid('0001', set);
    expect(raw.closes).toMatch(/Z$/);
    expect(localInput(raw.closes)).toBe('2026-10-10T20:30');
    expect(localInput('')).toBe('');
    expect(rawBid('0001', form).closes).toBe('');
    // A date without a time is no closing time yet.
    expect(rawBid('0001', { ...set, closesTime: '' }).closes).toBe('');
  });
});

describe('Hand-off', () => {
  it('copies the post before Go records the hand-off, and says so', async () => {
    const order: string[] = [];
    const { provider } = fakeProvider({
      handOff: async () => { order.push('record'); return plan('ready'); },
    });
    const copy = vi.fn(async (value: string) => { order.push(`copy:${value.split('\n')[0]}`); return true; });
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0, copy });
    await state.open();
    await state.change({ field: 'pool-handoff', value: plan('ready').id });
    expect(order).toEqual(['copy:Player: Smith, John (WR, DAL)', 'record']);
    expect(state.peek().notice).toContain('Nothing was posted for you');
  });
  it('still records the hand-off when this machine cannot copy, and tells the owner', async () => {
    const { provider, calls } = fakeProvider();
    const state = createPoolState({ provider, franchise: () => '0001', delay: 0, copy: async () => false });
    await state.open();
    await state.change({ field: 'pool-handoff', value: plan('ready').id });
    expect(calls.some((c) => c.startsWith('handOff'))).toBe(true);
    expect(state.peek().notice).toContain('select the post text above');
  });
  it('shows the exact ProBoards post, with Snipe when it is one', () => {
    const text = html(createElement(BidPlanCard, { plan: plan('ready') }));
    for (const line of ['Player: Smith, John (WR, DAL)', 'Contract: 3 years at $5,000,000 a year, flat',
      'Cap hit: $5,000,000 a year', 'Points: 7.000'])
      expect(text).toContain(line);
    expect(text).toMatch(/Points: 7\.000\nSnipe<\/pre>/);
    expect(text).toContain('Copy ProBoards post');
    expect(text).toContain('Bid to beat 5.500 · Snipe');
  });
  it('leaves the Act off with Go\'s reason when the app cannot tell playoff status', () => {
    const text = html(createElement(BidPlanCard, { plan: plan('blocked') }));
    expect(button(text, 'Copy ProBoards post: Adams, Zed (CB, MIA)')).toContain('disabled=""');
    expect(text).toContain('Copy ProBoards post is off: After Week 12 only playoff teams may bid');
    expect(text).toContain('cannot tell whether yours is one');
  });
});

describe('My moves and the Inspector', () => {
  it('lists bid plans with state, status and history like every other plan', () => {
    const text = html(createElement(BidBoardView, { board: board(), busy: '' }));
    for (const phrase of ['Bid plans', '2 open of 3 bid plans', 'Blocked', 'Ready', 'Bid rules checked',
      'History', 'Only your own plans are kept; nothing about other franchises', 'data-endpoint="M-194"',
      'How this is counted', 'the bid to beat is points you typed'])
      expect(text).toContain(phrase);
  });
  it('shows an award to someone else as the end of a plan, not an error', () => {
    const ready = plan('ready');
    const note = 'Awarded off-app: he is on another franchise\'s roster.';
    const awarded = { ...ready, stateCode: 'awarded_off_app', state: 'Awarded off-app', status: note,
      actions: [], receipt: { ...ready.receipt, state: 'awarded_off_app' as const, audit: [
        ...ready.receipt.audit, { at: '2026-10-10T13:00:00Z', from: 'ready' as const,
          to: 'awarded_off_app' as const, note }] } };
    const text = html(createElement(BidPlanCard, { plan: awarded }));
    expect(text).toContain('Awarded off-app');
    expect(text).not.toContain('role="alert"');
    expect(text.indexOf('ProBoards post')).toBeGreaterThan(text.indexOf('<summary>History</summary>'));
    expect(text).not.toContain('class="act');
  });
  it.each(['withdrawn', 'landed', 'failed'] as const)('folds %s bid post under History', state => {
    const ready = plan('ready');
    const finished = { ...ready, stateCode: state, actions: [],
      receipt: { ...ready.receipt, state } };
    const text = html(createElement(BidPlanCard, { plan: finished }));
    expect(text.indexOf('ProBoards post')).toBeGreaterThan(text.indexOf('<summary>History</summary>'));
    expect(text).not.toContain('class="bid-line"');
    expect(text).not.toContain('class="act');
  });
  it('shows a failed list as failed and an empty one as empty', () => {
    const failed = { ...board(), status: 'failed' as const, absenceReason: 'plan store unavailable' };
    const view = (board?: BoardReading) => html(createElement(BidBoardView, { board, busy: '' }));
    expect(view(failed)).toContain('Failed · plan store unavailable');
    const none = { ...board(), value: { plans: [], summary: '0 open of 0 bid plans' } };
    expect(view(none)).toContain('No bid plans yet');
    expect(view(undefined)).toContain('Reading your bid plans…');
  });
  it('calls Go through window.go, so no generated wrapper is added to the entry chunk', () => {
    const source = readFileSync('src/app/pool/provider.ts', 'utf8');
    expect(source).toContain("import type * as Bindings from '../../../wailsjs/go/main/App'");
    expect(source).not.toMatch(/import \{[^}]*\} from '[./]*wailsjs\/go\/main\/App'/);
  });
  it('loads the plans into My moves lazily, so the entry bundle does not carry them', () => {
    const source = readFileSync('src/app/shell/MyMoves.tsx', 'utf8');
    expect(source).toContain("lazy(() => import('../pool/BidPlans'))");
    expect(source).not.toMatch(/from '\.\.\/pool\//);
  });
  it('shows a pool player\'s open bid plan in the Inspector', () => {
    const row = pool().value.rows[2];
    const open = html(createElement(PoolInspectorView, { row, plans: [plan('ready')], busy: '' }));
    for (const phrase of ['Smith, John', 'Minimum salary', 'Bid plan Smith, John', 'Ready'])
      expect(open).toContain(phrase);
    expect(open).not.toContain('Plan a bid');
    const none = html(createElement(PoolInspectorView, { row, plans: [], busy: '' }));
    expect(none).toContain('No open bid plan on this player.');
    expect(none).toContain('Plan a bid');
  });
  it('counts the Inspector\'s minimum salary with the pool table\'s own Go-written words', () => {
    const reading = { ...pool(), howCounted: 'Minimum salary as Go counts it, from the fixture' };
    const row = reading.value.rows[2];
    const inspector = html(createElement(PoolInspectorView, { row, plans: [], busy: '', reading }));
    const table = html(createElement(PoolFrame, { reading, refreshing: false }));
    const how = (markup: string) =>
      /<details><summary>How this is counted<\/summary>.*?<\/details>/.exec(markup)?.[0];
    expect(how(inspector)).toContain('Minimum salary as Go counts it, from the fixture');
    expect(how(inspector)).toBe(how(table));
    expect(inspector.indexOf('How this is counted')).toBeGreaterThan(inspector.indexOf('$530,000'));
  });
});

describe('Reachability', () => {
  it('puts the four registry rows on the pool workspace, built, reachable by menu and command', () => {
    for (const id of ['M-039', 'M-056', 'M-194', 'C-06.7'] as const) {
      const row = endpointById(id);
      expect(row.status).not.toBe('spec');
      expect(row.placement).toBe('War Room › Valuations and pool');
    }
    expect(placementRoute('War Room › Valuations and pool')).toEqual({
      node: 'war', workspace: 'valuations-and-pool',
    });
  });
  it('routes pool fields from the Market verb to the pool state', async () => {
    await marketSet({ field: 'pool-cancel', value: '' } as Parameters<typeof marketSet>[0]);
    expect(poolState.peek().form).toBeUndefined();
  });
});
