import { it, expect, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import snapshot from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { createCommands } from '../commands/registry';
import { FixtureProvider } from '../data/provider';
import { CounterReading, CounterControls, TeamTable, sortedRows, panelEndpoint } from './Counterparties';
import { TeamReading } from './TeamPanel';
import { EdgeBar, teamIdentity } from './Displays';
import { createCounterState, filtered } from './state';
import { parseCounterparties, parseTeamPanel } from './provider';
import { counterpartProvider } from './api';
import { surfaceWire, teamWire, thinWire, emptyWire } from './wire.testdata';
import type { CounterpartiesReading, CounterScope } from './contract';
const scope: CounterScope = {
  from: 2017,
  to: 2026,
  tiers: [],
  windows: [],
  postures: [],
  team: '',
};
const league = parseSnapshot(snapshot);
const html = (component: typeof CounterReading | typeof TeamReading, reading: unknown) =>
  renderToStaticMarkup(
    createElement(component as typeof CounterReading, {
      reading: reading as CounterpartiesReading,
      snapshot: league,
    }),
  );

it('accepts Go payloads with omitted fields and null collections', () => {
  const surface = parseCounterparties(surfaceWire),
    team = parseTeamPanel(teamWire);
  expect(surface.value?.cards).toHaveLength(3);
  expect(surface.value?.pairs).toHaveLength(9);
  expect(team.value?.behaviour.n).toBeGreaterThan(0);
  expect(team.value?.day.total).toBeDefined();
  expect(parseCounterparties(thinWire).value?.cards).toEqual([]);
  expect(parseTeamPanel(emptyWire).value?.empty).toBe(true);
});
it('rejects malformed IPC data rather than rendering inventions', () => {
  expect(() => parseCounterparties({ ...surfaceWire, extra: true })).toThrow();
  expect(() => parseCounterparties({ ...surfaceWire, inputSHA: '' })).toThrow();
  const card = surfaceWire.value.cards[0];
  expect(() =>
    parseCounterparties({
      ...surfaceWire,
      value: { ...surfaceWire.value, cards: [{ ...card, thin: 'false' }] },
    }),
  ).toThrow(/boolean/);
  const tier = surfaceWire.value.trends.tiers[0];
  expect(() =>
    parseCounterparties({
      ...surfaceWire,
      value: {
        ...surfaceWire.value,
        trends: {
          ...surfaceWire.value.trends,
          tiers: [
            {
              ...tier,
              shift: {
                now: {
                  early: 0.1,
                  late: 0.2,
                  from: [2017],
                  to: [2022, 2026],
                  clear: true,
                },
                later: null,
              },
            },
          ],
        },
      },
    }),
  ).toThrow(/two years/);
  expect(() =>
    parseTeamPanel({ ...teamWire, value: { ...teamWire.value, empty: 'false' } }),
  ).toThrow();
});
it('renders L-23–L-26 with archive disclosure and range bars', () => {
  const text = html(CounterReading, parseCounterparties(surfaceWire));
  for (const id of ['L-23', 'L-24', 'L-25', 'L-26'])
    expect(text).toContain(`data-endpoint="${id}"`);
  expect(text).toContain('2026-10-06');
  expect(text).toContain('not today&#x27;s live standings');
  expect(text).toContain('How this is counted');
  expect(text).toContain('zero marked');
  expect(text).toContain('aria-sort="descending"');
  expect(text).not.toMatch(/· (up|down|even|swap|now|later|hunt|rebuild)\b|\bsolid\b|\brough\b/);
});
it('renders L-27 through L-29 with marked seasons, both asset directions and inspectable partners', () => {
  const reading = parseTeamPanel(teamWire);
  const text = html(TeamReading, reading);
  for (const id of ['L-27', 'L-28', 'L-29']) expect(text).toContain(`data-endpoint="${id}"`);
  expect(text).toContain('data-flagged="true"');
  expect(text).toContain('Against its standing');
  expect(text).toContain('Sent ·');
  expect(text).toContain('Taken ·');
  expect(text).toContain('Inspect');
  expect(text).toContain('How this is counted');
});
it('both surfaces handle requested, thin, failed and not-wired readings', async () => {
  for (const component of [CounterReading, TeamReading]) {
    expect(html(component, undefined)).toContain('No figures yet');
    for (const status of ['failed', 'not_wired'] as const) {
      const text = html(component, {
        ...parseCounterparties(surfaceWire),
        value: undefined,
        status,
        absenceReason: 'hold unavailable',
      });
      expect(text).toContain(status === 'failed' ? 'Failed' : 'Not wired');
      expect(text).toContain('hold unavailable');
      expect(text).not.toContain('<svg');
      expect(text).not.toContain('<table');
    }
  }
  const thin = html(CounterReading, parseCounterparties(thinWire));
  expect(thin).toContain('Too few to judge');
  expect(thin).not.toContain('<svg');
  expect(thin).not.toContain('How each team trades');
  const empty = html(TeamReading, parseTeamPanel(emptyWire));
  expect(empty).toContain('no valued trades');
  expect(empty).not.toContain('<meter');
  expect((await counterpartProvider.surface(scope)).status).toBe('not_wired');
  expect((await counterpartProvider.team(scope, '0001')).status).toBe('not_wired');
});
it('awaits desktop calls before parsing and reports invalid or rejected responses failed', async () => {
  const old = globalThis.window;
  const surface = vi.fn(async () => surfaceWire),
    team = vi.fn(async () => teamWire);
  globalThis.window = {
    go: { counterparties: { API: { TargetCounterparties: surface, TargetCounterpartTeam: team } } },
  } as unknown as Window & typeof globalThis;
  try {
    expect((await counterpartProvider.surface(scope)).status).toBe('ok');
    expect((await counterpartProvider.team(scope, '0001')).status).toBe('ok');
    expect(surface).toHaveBeenLastCalledWith(scope);
    expect(team).toHaveBeenLastCalledWith(expect.objectContaining({ scope, fid: '0001' }));
    surface.mockRejectedValueOnce(new Error('fingerprint mismatch'));
    expect((await counterpartProvider.surface(scope)).absenceReason).toContain(
      'fingerprint mismatch',
    );
    team.mockResolvedValueOnce({ ...teamWire, inputSHA: '' });
    expect((await counterpartProvider.team(scope, '0001')).status).toBe('failed');
  } finally {
    globalThis.window = old;
  }
});
it('filters validate Go scope and preserve prior readings while pending', async () => {
  const reader = vi.fn(async () => parseCounterparties(surfaceWire));
  const state = createCounterState({ surface: reader, team: async () => parseTeamPanel(teamWire) });
  await state.refresh();
  for (const change of [
    { field: 'counterpart-tier', value: 'rebuild' },
    { field: 'counterpart-window', value: 'season' },
    { field: 'counterpart-posture', value: 'later' },
    { field: 'counterpart-team', value: '0001' },
  ] as const) {
    const before = state.peek().reading;
    const request = state.change(change);
    expect(state.peek().reading).toBe(before);
    expect(state.peek().pending).toBe(true);
    await request;
    expect(reader).toHaveBeenLastCalledWith(state.peek().scope);
  }
  expect(state.peek().scope).toEqual({
    ...scope,
    tiers: ['rebuild'],
    windows: ['season'],
    postures: ['later'],
    team: '0001',
  });
  await state.change({ field: 'counterpart-reset', value: '' });
  expect(state.peek().scope).toEqual(scope);
  expect(() => filtered(scope, { field: 'counterpart-tier', value: 'guess' })).toThrow();
  expect(
    filtered(
      { ...scope, tiers: ['contender', 'hunt'] },
      { field: 'counterpart-tier', value: 'rebuild' },
    ).tiers,
  ).toEqual([]);
  const text = renderToStaticMarkup(createElement(CounterControls));
  expect(text).toContain('aria-label="Filter Contenders"');
  expect(text).toContain('>Contenders</button>');
});
it('drops stale surface replies and allows team retry with the held scope, not the team filter', async () => {
  let finish!: (r: CounterpartiesReading) => void;
  const reader = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValue(parseCounterparties(surfaceWire));
  const team = vi.fn(async () => parseTeamPanel(teamWire));
  const state = createCounterState({ surface: reader, team });
  const stale = state.refresh();
  await state.change({ field: 'counterpart-team', value: '0001' });
  finish({ ...parseCounterparties(surfaceWire), absenceReason: 'stale' });
  await stale;
  expect(state.peek().reading?.absenceReason).not.toBe('stale');
  await state.team('0001');
  expect(team).toHaveBeenLastCalledWith(scope, '0001');
  await state.team('0001');
  expect(team).toHaveBeenCalledTimes(1);
  const applied = state.peek().applied;
  await state.change({ field: 'counterpart-retry', value: '' });
  expect(state.peek().applied).not.toBe(applied);
  await state.team('0001');
  expect(team).toHaveBeenCalledTimes(2);
});
it('sorts only supplied fields with stable ties and nulls last in either direction', () => {
  const rows = parseCounterparties(surfaceWire).value!.rows;
  const tied = [
    { ...rows[0], fid: 'first', n: 7 },
    { ...rows[1], fid: 'second', n: 7 },
  ];
  expect(sortedRows(tied, 'n', true).map((r) => r.fid)).toEqual(['first', 'second']);
  const missing = { ...rows[0], fid: 'missing', day: { ...rows[0].day, total: undefined } };
  for (const descending of [true, false])
    expect(sortedRows([missing, rows[1]], 'day', descending).at(-1)?.fid).toBe('missing');
  expect(rows[0].fid).not.toBe('missing');
});
it('resolves franchise ids and inspects teams independently of players', () => {
  const identity = league.franchises.value[0];
  expect(teamIdentity(league, identity.id, 'old name')).toEqual({
    name: identity.name,
    matched: true,
  });
  expect(teamIdentity(league, '9999', 'Archive team')).toEqual({
    name: 'Archive team',
    matched: false,
  });
  const c = createCommands(() => undefined, new FixtureProvider());
  c.loadSnapshot(league);
  c.dispatch('market', { field: 'inspect', value: `team:${identity.id}` });
  expect(c.read().asset).toBe(`team:${identity.id}`);
  expect(c.read().subject).toBeNull();
  for (let n = 23; n <= 29; n++) {
    c.dispatch('endpoint.open', { id: `L-${n}` as 'L-23' });
    expect(c.read().node).toBe('trade');
    expect(c.read().workspace.trade).toBe('counterparties');
  }
});
it('range rendering uses all supplied tones without showing enum text or inventing absent ranges', () => {
  for (const tone of ['up', 'down', 'even']) {
    const text = renderToStaticMarkup(
      createElement(EdgeBar, {
        edge: { n: 3, mean: 0.1, lo: -0.2, hi: 0.4, tone },
        label: 'Reading',
      }),
    );
    expect(text).toContain(`data-tone="${tone}"`);
    expect(text).not.toMatch(new RegExp(`>${tone}<`));
    expect(text).toContain('<svg');
  }
  expect(
    renderToStaticMarkup(
      createElement(EdgeBar, {
        edge: { n: 1, mean: 0.1, lo: null, hi: null, tone: 'even' },
        label: 'Reading',
      }),
    ),
  ).not.toContain('<svg');
  for (const file of ['Counterparties', 'TeamPanel', 'Displays']) {
    const code = readFileSync(`src/app/counterparties/${file}.tsx`, 'utf8');
    expect(code).not.toMatch(/reduce\(|fetch\(|confidence|\bsolid\b|\brough\b/);
  }
});

it('team endpoint doors distinguish panel requests without inventing a franchise', () => {
  for (const id of ['L-27', 'L-28', 'L-29']) expect(panelEndpoint([id])).toBe(true);
  expect(panelEndpoint(['L-23', 'L-24', 'L-25', 'L-26'])).toBe(false);
});

it('contains wide tables and reflows tiers by column rather than viewport', () => {
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  const text = html(CounterReading, parseCounterparties(surfaceWire));
  expect(text).toContain('class="counter-tiers"');
  expect(text).toContain('class="counter-scroll counter-team-scroll"');
  expect(text).toContain('class="counter-table counter-matrix"');
  expect(css).toContain('repeat(auto-fit, minmax(min(100%, 280px), 1fr))');
  expect(css).not.toContain('@media');
  expect(css).toMatch(/\.counter-scroll\s*\{[^}]*overflow: auto;[^}]*max-width: 100%;/);
  expect(css).toMatch(/\.counter-table th:first-child\s*\{[^}]*position: sticky;[^}]*left: 0;/);
});
it('wraps all workspace and panel prose including the headline and archive metadata', () => {
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  expect(css).toMatch(/overflow-wrap: normal;\s*word-break: normal;\s*white-space: normal;/);
  expect(css).toContain('max-width: min(100%, 72ch)');
  expect(css).toContain('grid-template-columns: minmax(0, 1fr) minmax(0, 2fr)');
  expect(html(CounterReading, parseCounterparties(surfaceWire))).toContain('25.3%');
});
it('shows three short standing trend tables with shares and no repeated standing column', () => {
  const text = html(CounterReading, parseCounterparties(surfaceWire));
  const trend = text.slice(
    text.indexOf('data-endpoint="L-24"'),
    text.indexOf('data-endpoint="L-25"'),
  );
  expect(trend).toContain('class="counter-trends"');
  expect(trend.match(/counter-trend-table/g)).toHaveLength(3);
  expect(trend).not.toContain('<th scope="col">Standing</th>');
  for (const label of [
    'Contenders',
    'In the hunt',
    'Rebuilding',
    'Bought for now',
    'Built for later',
  ])
    expect(trend).toContain(label);
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  expect(css).toMatch(/\.counter-trend-table\s*\{\s*table-layout: fixed;/);
});
it('each matrix range carries its own Go-authored visible label, even when absent', () => {
  const r = parseCounterparties(surfaceWire);
  r.value!.words.day = 'Day label from Go';
  r.value!.words.since = 'Since label from Go';
  const text = html(CounterReading, r);
  for (const id of ['L-25']) {
    const section = text.slice(text.indexOf(`data-endpoint="${id}"`));
    expect(section).toContain('Day label from Go:');
    expect(section).toContain('Since label from Go:');
  }
  expect(
    renderToStaticMarkup(
      createElement(EdgeBar, {
        edge: { n: 0, mean: null, lo: null, hi: null, tone: 'even' },
        label: 'Since label from Go',
      }),
    ),
  ).toContain('Since label from Go: —');
});
it('keeps team headings sticky in a bounded scroll box and explains red with a non-colour mark', () => {
  const r = parseCounterparties(surfaceWire);
  r.value!.rows[0].forNow.tone = 'down';
  r.value!.rows[0].forNow.p = 0.28;
  const text = html(CounterReading, r);
  expect(text).toContain('counter-sticky-header');
  expect(text).toContain('28% ↓');
  expect(text).toContain(r.value!.words.legend);
  expect(html(TeamReading, parseTeamPanel(teamWire))).toContain('↓ (red)');
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  expect(css).toMatch(/\.counter-team-scroll\s*\{\s*max-height: 480px;/);
  expect(css).toMatch(/\.counter-sticky-header th\s*\{[^}]*position: sticky;[^}]*top: 0;/);
});
it('renders Go-formatted large counts and never a rounded negative zero', () => {
  const r = parseCounterparties(surfaceWire);
  r.value!.rows[0].nText = '3,810';
  r.value!.rows[0].forNow.countText = '1,000 of 3,810';
  r.value!.rows[0].n = 3810;
  r.value!.rows[0].day.mean = -0.0001;
  const text = html(CounterReading, r);
  expect(text).toContain('3,810 trade sides');
  expect(text).toContain('<td class="counter-nowrap" data-column="n">3,810</td>');
  expect(text).toContain('Bought for now: 1,000 of 3,810');
  expect(text).toContain('3,810 / 40');
  expect(text).not.toContain('-0.00');
  expect(html(TeamReading, parseTeamPanel(teamWire))).not.toContain('-0.00');
});
it('themes accessible panel meters and keeps Go tier casing consistent', () => {
  const text = html(TeamReading, parseTeamPanel(teamWire));
  expect(text).toContain('<meter min="0"');
  expect(text).toContain('>In the hunt</p>');
  expect(text).toContain('>Rebuilding</p>');
  expect(text).toContain('As Contenders');
  expect(text).not.toContain('>contender</p>');
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  expect(css).toContain('meter::-webkit-meter-bar');
  expect(css).toContain('meter::-webkit-meter-optimum-value');
  expect(css).toContain('meter::-moz-meter-bar');
  expect(css).toContain('background: var(--surface-tx2)');
});

it('team totals are a single Go-authored text node with no range bars', () => {
  const v = parseCounterparties(surfaceWire).value!;
  v.rows[0].dayText = 'Day total and range from Go';
  v.rows[0].sinceText = 'Since total and range from Go';
  const text = renderToStaticMarkup(
    createElement(TeamTable, { rows: v.rows, words: v.words, snapshot: league }),
  );
  const cells = [...text.matchAll(/<td class="counter-total"[^>]*>([^<]*)<\/td>/g)];
  expect(cells).toHaveLength(v.rows.length * 2);
  expect(text).toContain('>Day total and range from Go</td>');
  expect(text).toContain('>Since total and range from Go</td>');
  expect(text).not.toContain('<svg');
  expect(text).not.toContain('counter-edge');
  expect(text).toContain('counter-standing');
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  expect(css).toMatch(/\.counter-team-table\s*\{[^}]*table-layout: auto;/);
});
it('matrix headers use the phrases sent by Go', () => {
  const r = parseCounterparties(surfaceWire);
  for (const pick of r.value!.picks) expect(html(CounterReading, r)).toContain(pick.with);
  r.value!.picks[1].with = 'Custom with phrase from Go';
  expect(html(CounterReading, r)).toContain('Custom with phrase from Go');
  expect(html(CounterReading, r)).not.toContain('With In the hunt');
});

it('keeps all ten wrapping columns and nests the frequent partner under Partners', () => {
  const v = parseCounterparties(surfaceWire).value!;
  const text = renderToStaticMarkup(createElement(TeamTable, { rows: v.rows, words: v.words }));
  const css = readFileSync('src/app/counterparties/counterparties.css', 'utf8');
  const headers = [...text.slice(text.indexOf('<thead'), text.indexOf('</thead>'))
    .matchAll(/<th\b[^>]*data-column="([^"]+)"/g)].map(m => m[1]);
  expect(headers).toEqual(['name', 'n', 'perSeason', 'now', 'later', 'day', 'since', 'picks',
    'age', 'partners']);
  expect(text).toContain(v.words.partner);
  expect(css).not.toMatch(/display:\s*none/);
  expect(css).toContain('.counter-team-table { table-layout: fixed; }');
  const controls = text.slice(text.indexOf('aria-label="Sort by"'), text.indexOf('counter-scroll'));
  expect(controls.match(/<button/g)).toHaveLength(10);
});
it('keeps the five narrow omissions available in the team panel from supplied fields', () => {
  const r = parseTeamPanel(teamWire);
  r.value!.words.picks = 'Go pick heading';
  r.value!.words.age = 'Go age heading';
  r.value!.partnersText = '31';
  const text = html(TeamReading, r);
  expect(text).toContain(`${r.value!.behaviour.perSeason.toFixed(1)} a season`);
  expect(text).toContain(`Go pick heading: ${r.value!.behaviour.picks.mean!.toFixed(2)}`);
  expect(text).toContain(`Go age heading: ${r.value!.behaviour.age.mean!.toFixed(1)}`);
  expect(text).toContain('Partners: 31');
  expect(text).toContain('Most frequent partner:');
  expect(text).toContain(r.value!.partners[0].tradesText);
});

it('renders filtered shares without chance arrows and excluded tiers without thin warnings', () => {
  const r = parseCounterparties(surfaceWire);
  const v = r.value!;
  v.summary = 'Showing only trades that bought for now.';
  v.headline = 'Contenders come out ahead on the day.';
  v.words.legend = 'Ranges show gains and losses beyond chance.';
  v.trendSay = [];
  for (const card of v.cards) {
    card.excluded = card.id !== 'contender';
    card.thin = false;
    for (const cell of card.posture) { cell.tone = ''; cell.title = 'Share of filtered trades'; }
  }
  for (const row of v.rows) { row.forNow.tone = ''; row.forLater.tone = ''; }
  v.trends.tiers = [];
  for (const [i, pair] of v.pairs.entries()) {
    pair.excluded = i >= 3;
    if (pair.excluded) pair.trades = 'Not in the current filter';
  }
  const shown = html(CounterReading, r);
  expect(shown).toContain('Showing only trades that bought for now.');
  expect(shown).not.toContain('buy for now in');
  expect(shown).not.toMatch(/\d+%\s+[↑↓]/);
  expect(shown).not.toMatch(/data-tone="(?:up|down)"[^>]*>Bought for now/);
  expect(shown.match(/Not in the current filter/g)).toHaveLength(2 + v.pairs.length - 3);
  expect(shown.match(/<p>Not in the current filter<\/p><\/td>/g)).toHaveLength(v.pairs.length - 3);
  const edge = { n: 0, mean: null, lo: null, hi: null, tone: '' };
  const empty = renderToStaticMarkup(createElement(EdgeBar, { edge, label: 'Since' }));
  expect(empty).toBe('<div class="counter-edge"><span>Since: —</span></div>');
  expect(shown).not.toContain('Too few to judge');
  const panel = parseTeamPanel(teamWire);
  for (const tier of panel.value!.asTier) {
    tier.excluded = true;
    tier.text = 'Not in the current filter';
  }
  expect(html(TeamReading, panel)).toContain('Not in the current filter');
  expect(html(TeamReading, panel)).not.toMatch(/As [^<]+: .*likely/);
});

it('keeps browser-without-go and missing-feature transport paths honest', async () => {
  const old = globalThis.window;
  globalThis.window = {} as Window & typeof globalThis;
  try {
    for (const reading of [await counterpartProvider.surface(scope),
      await counterpartProvider.team(scope, '0001')]) {
      expect(reading.status).toBe('not_wired');
      expect(reading.absenceReason).toBe(
        'Counterparties needs the desktop app; no browser fixtures are substituted',
      );
    }
    globalThis.window = { go: {} } as unknown as Window & typeof globalThis;
    expect((await counterpartProvider.surface(scope)).status).toBe('failed');
    expect((await counterpartProvider.team(scope, '0001')).status).toBe('failed');
  } finally {
    globalThis.window = old;
  }
});
