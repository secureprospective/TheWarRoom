import { it, expect, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { readFileSync } from 'node:fs';
import snapshot from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { createCommands } from '../commands/registry';
import { FixtureProvider } from '../data/provider';
import { CounterReading, CounterControls, sortedRows, panelEndpoint } from './Counterparties';
import { TeamReading } from './TeamPanel';
import { EdgeBar, teamIdentity } from './Displays';
import { createCounterState, filtered } from './state';
import { parseCounterparties, parseTeamPanel, counterpartProvider } from './provider';
import { surfaceWire, teamWire, thinWire, emptyWire } from './wire.testdata';
import type { CounterpartiesReading, CounterScope } from './contract';
const scope: CounterScope = { from: 2017, to: 2026, tiers: [], windows: [], postures: [], team: '' };
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
  expect(() => parseTeamPanel({ ...teamWire, value: { ...teamWire.value, empty: 'false' } })).toThrow();
});
it('renders L-23–L-26 with archive disclosure and range bars', () => {
  const text = html(CounterReading, parseCounterparties(surfaceWire));
  for (const id of ['L-23', 'L-24', 'L-25', 'L-26']) expect(text).toContain(`data-endpoint="${id}"`);
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
    go: { main: { App: { TargetCounterparties: surface, TargetCounterpartTeam: team } } },
  } as unknown as Window & typeof globalThis;
  try {
    expect((await counterpartProvider.surface(scope)).status).toBe('ok');
    expect((await counterpartProvider.team(scope, '0001')).status).toBe('ok');
    surface.mockRejectedValueOnce(new Error('fingerprint mismatch'));
    expect((await counterpartProvider.surface(scope)).absenceReason).toContain('fingerprint mismatch');
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
    filtered({ ...scope, tiers: ['contender', 'hunt'] }, { field: 'counterpart-tier', value: 'rebuild' })
      .tiers,
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
  expect(teamIdentity(league, identity.id, 'old name')).toEqual({ name: identity.name, matched: true });
  expect(teamIdentity(league, '9999', 'Archive team')).toEqual({ name: 'Archive team', matched: false });
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
      createElement(EdgeBar, { edge: { n: 3, mean: 0.1, lo: -0.2, hi: 0.4, tone }, label: 'Reading' }),
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
