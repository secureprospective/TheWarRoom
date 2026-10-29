import { describe, expect, it } from 'vitest';
import { parseAssetBoard } from './contract';
import { createAssetState } from './state';

const wire = {
  values: [{ id: '16162',
    now: { text: '18.40 PPG', sub: 'WR 3/120 · top 3%', detail: '73.60 points in 4 scored weeks',
      held: true },
    dynasty: { text: 'Not valued yet', sub: 'no model run read yet', detail: '', held: false },
    market: { text: 'Not valued yet', sub: 'no model run read yet', detail: '', held: false },
    nowValue: 18.4, dynastyValue: 0 }],
  nowSource: 'Source: MFL fantasy points', dynastySource: 'Source: the Lab', how: 'Counted.',
};

describe('C-01 asset board', () => {
  it('parses Go cells and rejects unknown fields and non-player ids', () => {
    expect(parseAssetBoard(wire).values[0].now.sub).toBe('WR 3/120 · top 3%');
    expect(parseAssetBoard({ ...wire, values: [{ ...wire.values[0], nowValue: -1.5 }] })
      .values[0].nowValue).toBe(-1.5);
    expect(() => parseAssetBoard({ ...wire, values: [{ ...wire.values[0], nowValue: null }] }))
      .toThrow('expected number');
    expect(() => parseAssetBoard({ ...wire, extra: 1 })).toThrow('unexpected field');
    expect(() => parseAssetBoard({ ...wire, values: [{ ...wire.values[0], id: 'x' }] })).toThrow('player id');
    expect(() => parseAssetBoard({ ...wire, values: [{ ...wire.values[0],
      now: { ...wire.values[0].now, held: 'yes' } }] })).toThrow('expected boolean');
  });
  it('words loading, a held value, an unlisted player and a failed read', async () => {
    let fail = false;
    const s = createAssetState(async () => {
      if (fail) throw new Error('startup: not ready');
      return parseAssetBoard(wire);
    }, () => () => undefined);
    expect(s.cellFor(s.peek(), '16162', 'now').text).toBe('Reading…');
    await s.load();
    expect(s.cellFor(s.peek(), '16162', 'now'))
      .toMatchObject({ held: true, text: '18.40 PPG', sub: 'WR 3/120 · top 3%' });
    expect(s.cellFor(s.peek(), '16162', 'market').text).toBe('Not valued yet');
    expect(s.cellFor(s.peek(), '99999', 'now')).toMatchObject({ text: 'No score', sub: 'this season' });
    fail = true;
    await s.load();
    expect(s.cellFor(s.peek(), '99999', 'now')).toMatchObject({ text: 'Unavailable', held: false });
  });
});

describe('C-01 board view', async () => {
  const { createElement } = await import('react');
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { FixtureProvider } = await import('../data/provider');
  const { AssetBoardView, rosterRows } = await import('./AssetBoardView');
  const snapshot = await new FixtureProvider().snapshot();
  it('lists every rostered player once per roster, with franchise names', () => {
    const rows = rosterRows(snapshot);
    expect(rows).toHaveLength(snapshot.rosters.value.reduce((n, r) => n + r.players.length, 0));
    const named = new Set(snapshot.franchises.value.map(f => f.name));
    expect(rows.every(r => named.has(r.franchise) || r.franchise.startsWith('Franchise '))).toBe(true);
  });
  it('renders the board on its endpoint with every franchise and the position filter', () => {
    const html = renderToStaticMarkup(createElement(AssetBoardView, { snapshot }));
    expect(html).toContain('data-endpoint="C-01"');
    expect(html).toContain('All franchises');
    expect(html).toContain('All positions');
    expect(html).toContain('On-field-now');
    expect(html).toContain('Reading…');
  });
});

it('keeps table values whole and wraps the qualifier on a separate line', async () => {
  const { vi } = await import('vitest');
  const { createElement } = await import('react');
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { readFileSync } = await import('node:fs');
  const { AssetNumber } = await import('./AssetNumber');
  const { assetState } = await import('./state');
  const spy = vi.spyOn(assetState, 'useCell').mockReturnValue({
    text: '14.00 PPG', sub: 'QB 14/54 · top 26%', detail: '', held: true,
  });
  const markup = renderToStaticMarkup(createElement(AssetNumber,
    { playerId: '0042', which: 'now', inline: true }));
  expect(markup).toContain('14.00 PPG<span class="asset-sub asset-table-sub">QB 14/54 · top 26%</span>');
  expect(markup).toContain('title="14.00 PPG · QB 14/54 · top 26%"');
  const css = readFileSync(new URL('./assetNumber.css', import.meta.url), 'utf8');
  expect(css).toContain('white-space: normal; overflow: visible; text-overflow: clip;');
  expect(css).toContain('.matrix-roster .asset-table-sub { display: block; white-space: normal; }');
  spy.mockRestore();
});

it('fences overlapping board reads and keeps failed cell wording during retry', async () => {
  const waiting: ((board: ReturnType<typeof parseAssetBoard>) => void)[] = [];
  let fail = false;
  const state = createAssetState(async () => {
    if (fail) throw new Error('board unavailable');
    return new Promise(resolve => waiting.push(resolve));
  }, () => () => {});
  const old = state.load();
  const latest = state.load();
  const board = parseAssetBoard(wire);
  waiting[1](board);
  await latest;
  waiting[0]({ ...board, values: [] });
  await old;
  expect(state.peek().board).toBe(board);
  fail = true;
  await state.load();
  fail = false;
  const retry = state.load();
  expect(state.peek().pending).toBe(true);
  expect(state.cellFor(state.peek(), '99999', 'now').text).toBe('Unavailable');
  expect(state.cellFor(state.peek(), '16162', 'now').held).toBe(true);
  waiting[2](board);
  await retry;
  expect(state.peek().error).toBe('');
  expect(state.cellFor(state.peek(), '99999', 'now').text).toBe('No score');
});

it.each([
  { held: true, text: '12.00 PPG', sub: 'dynasty · next 3 seasons · reduced inputs',
    detail: 'league points for 2024 are not ready' },
  { held: false, text: 'Not valued', sub: 'No usable season: games played for 2025 are not ready', detail: '' },
])('keeps input gaps visible in player and pick cells: $text', async cell => {
  const { createElement } = await import('react');
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { AssetCellView } = await import('./AssetNumber');
  const board = parseAssetBoard({ ...wire, values: [{ ...wire.values[0], dynasty: cell }] });
  const html = renderToStaticMarkup(createElement(AssetCellView, { cell: board.values[0].dynasty }));
  expect(html).toContain(cell.text);
  expect(html).toContain(cell.sub);
  if (cell.detail) expect(html).toContain(cell.detail);
  if (!cell.held) expect(html).not.toContain('0.00');
});
