import { describe, expect, it } from 'vitest';
import { parseAssetBoard } from './contract';
import { createAssetState } from './state';

const wire = {
  values: [{ id: '16162',
    now: { text: '18.40 PPG', sub: 'WR 3/120 · top 3%', detail: '73.60 points in 4 scored weeks',
      held: true },
    dynasty: { text: 'Not valued yet', sub: 'Lab model not compiled', detail: '', held: false },
    market: { text: 'Not valued yet', sub: 'Lab model not compiled', detail: '', held: false },
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
