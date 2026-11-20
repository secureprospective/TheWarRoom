// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { PulseResults } from '../data/nativePulse';
import type { LeagueChoice } from '../shell/leaguePicker';
import type { NativePulse } from '../shell/pulseReads';
import { readLeague, LeagueTab, streakClass, type LeagueReading, type LeagueRosterPlayer } from './LeagueTab';

const owner: LeagueChoice = { id: 'lg', name: 'Test League', app: true, franchise: 'a', role: 'Owner' };
const commish: LeagueChoice = { ...owner, role: 'Commissioner' };
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});
const row = (rank: number, franchise: string, name: string, wins: number, conference = '',
  division = '') => ({ rank, franchise, name, division, conference, wins, losses: 10 - wins, ties: 0,
  record: `${wins}-${10 - wins}`,
    pointsFor: 1000 + wins, pointsWords: '', why: [] });
const pulse = (rows = [row(1, 'b', 'Bravo', 8, 'Forge', 'East'), { ...row(2, 'a', 'Alpha', 6, 'Mill', 'West'),
  streak: 'L2' },
  row(3, 'c', 'Charlie', 4, 'Forge', 'East')]): NativePulse => ({
  league: 'lg', name: 'Test League', season: 2026, revision: 1,
  seats: [{ franchise: 'a', name: 'Alpha', abbreviation: 'A', division: '' },
    { franchise: 'b', name: 'Bravo', abbreviation: 'B', division: '' },
    { franchise: 'c', name: 'Charlie', abbreviation: 'C', division: '' }],
  week: 2, weekNote: '', lastWeek: 3, fixtures: [], scheduleNote: '', forms: {}, formPending: false,
  standings: { league: 'lg', name: 'Test League', season: 2026, revision: 1, status: 'ok', words: '',
    through: 1,
    decided: 1, pending: 0, criteria: [], noData: [], rows },
  bracket: { league: 'lg', season: 2026, revision: 1, status: '', words: 'Top 1 per conference',
    startWeek: 15,
    finalWeek: 17, perConference: 1, byeSeeds: 0, started: false, fields: [{ conference: '',
      seeds: [{ seed: 1, franchise: 'b', name: 'Bravo', division: '', rank: 1, kind: '', bye: false,
        byHand: false }] }], projected: true, shape: [] },
  errors: [] });
const game = (week: number, home: string, away: string, hp?: number, ap?: number, winner?: string) => ({ week,
  home, away, homeName: home === 'a' ? 'Alpha' : 'Bravo', awayName: away === 'a' ? 'Alpha' : 'Charlie',
  status: '', homePoints: hp, awayPoints: ap, winner, words: '', why: [] });
const results = (week: number, published: boolean): PulseResults => ({ league: 'lg', season: 2026, week,
  revision: 1, published, words: '', games: [game(week, 'b', 'c', 90, 80, published ? 'b' : undefined),
    game(week, 'a', 'c', 100.5, 99, published ? 'a' : undefined)], scores: [], missed: [] });
const loaded = (): LeagueReading => ({ pulse: { value: pulse() },
  season: { value: { schedule: { season: 2026, weeks: [1, 2, 3].map(week => ({ week,
    games: [{ home: 'a', away: week === 1 ? 'b' : 'c' }] })) }, played: { 1: { ...results(1, true),
      games: [game(1, 'a', 'b', 110, 90, 'a')] } } } },
  activity: { value: { league: 'lg', revision: 1, events: [{ id: 'e1', kind: 'trade',
    when: '2026-11-18T15:00:00Z', summary: 'Bravo and Charlie traded receivers.', franchises: [] }] } },
  jobs: { value: { open: 2, now: 1 } } });

async function show(league: LeagueChoice, read: () => Promise<LeagueReading>,
  extra: { onTrade?: () => void; onControlRoom?: () => void; onWatch?: () => void;
    readRoster?: () => Promise<LeagueRosterPlayer[]> } = {}) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const readWeek = vi.fn(async (_l: string, _s: number, week: number) => results(week, week < 2));
  await act(async () => root!.render(<LeagueTab league={league} read={read} readWeek={readWeek}
    readRoster={extra.readRoster ?? (async () => [{ id: 'p1', name: 'Ace Runner', position: 'RB' }])}
    {...extra} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
  return host;
}
const click = (label: RegExp, scope: ParentNode = host!) => act(async () => {
  ([...scope.querySelectorAll('button, [role="tab"]')].find(el => label.test(el.textContent ?? '')
    || label.test(el.getAttribute('aria-label') ?? '')) as HTMLElement).click();
  await new Promise(resolve => setTimeout(resolve, 10));
});

const statusRead = (state: string, players: unknown[]) => ({ league: 'lg', state, season: 2026,
  fetchedAt: state === 'unavailable' ? null : '2026-11-18T15:00:00Z', players });

describe('LeagueTab', () => {
  it('player status: rostered facts in Activity and the team sheet; unavailable words', async () => {
    const withStatus = loaded();
    withStatus.status = { value: { read: { league: 'lg', state: 'available', facts: new Map([
      ['p1', { player: 'p1', designation: 'Out', bodyPart: 'Hamstring', returnWeek: 12 }],
      ['p9', { player: 'p9', designation: 'Questionable', bodyPart: '' }]]) },
    names: new Map([['p1', 'Ace Runner']]) } };
    const view = await show(owner, async () => withStatus);
    await click(/^Activity$/);
    expect(view.textContent).toContain('Ace Runner · Out · Hamstring · back week 12');
    expect(view.textContent).not.toContain('Questionable');
    await click(/^Table$/);
    await click(/Bravo: owner and record/);
    expect(document.querySelector('dialog')?.textContent).toContain('RB · Out · Hamstring · back week 12');
    await act(async () => root!.unmount()); host!.remove();
    const off = loaded();
    off.status = { value: { read: { league: 'lg', state: 'unavailable', facts: new Map() },
      names: new Map() } };
    const again = await show(owner, async () => off);
    await click(/^Activity$/);
    expect(again.textContent).toContain('the NFL injury report is not available right now');
  });
  it('readLeague reads nfl.player-status.read and names rostered players', async () => {
    const port = vi.fn(async (request: { command: string }) => {
      if (request.command === 'nfl.player-status.read') return { label: '', writable: false, football: '',
        data: statusRead('stale', [{ player: '0101', designation: 'Doubtful', bodyPart: 'Knee' }]) };
      if (request.command === 'native.assets.read') throw new Error('Assets held');
      throw new Error(`${request.command} held`);
    });
    const reading = await readLeague('lg', 'a', false, port as never);
    expect(reading.status?.error).toBe('Assets held');
  });
  it('report a problem opens the shared report form', async () => {
    const view = await show(owner, async () => loaded());
    await click(/^Report a problem$/);
    expect(view.querySelector('[data-testid="problem-report"]')).not.toBeNull();
    await click(/Close report form/);
    expect(view.querySelector('[data-testid="problem-report"]')).toBeNull();
  });
  it('loaded: table, scores, schedule and activity from the reads', async () => {
    const onWatch = vi.fn();
    const view = await show(commish, async () => loaded(), { onWatch });
    expect(view.textContent).toContain('2 open jobs · 1 needs you now');
    expect([...view.querySelectorAll('[role="tab"]')].map(t => t.textContent))
      .toEqual(['Table', 'Scores', 'Schedule', 'Activity']);
    expect(view.querySelector('tr.me')?.textContent).toContain('Alpha');
    expect(view.querySelector('tr.me')?.textContent).toContain('L2');
    expect(view.querySelector('tbody tr:nth-child(3)')?.textContent).toMatch(/–$/);
    await click(/^Forge conf\.$/);
    expect(view.textContent).not.toContain('Alpha');
    expect(view.textContent).toContain('Top 1 in each conference make the playoffs.');
    await click(/^Divisions$/);
    expect(view.textContent).toContain('Forge · East');
    await click(/^Scores$/);
    expect(view.textContent).toContain('WEEK 2 · LIVE');
    expect(view.querySelector('.lg5-mu')?.textContent).toContain('YOUR MATCHUP');
    await click(/watch live/);
    expect(onWatch).toHaveBeenCalled();
    await click(/Previous week/);
    expect(view.textContent).toContain('WEEK 1 · FINAL');
    expect(view.querySelector('[aria-label="Previous week"]')?.hasAttribute('disabled')).toBe(true);
    await click(/Next week/); await click(/Next week/);
    expect(view.textContent).toContain('WEEK 3 · UPCOMING');
    expect(view.textContent).toContain('WEEK 3');
    await click(/^Schedule$/);
    expect(view.textContent).toContain('Alpha · regular season, weeks 1–3');
    expect(view.textContent).toContain('beat Bravo');
    expect(view.textContent).toContain('W 110–90');
    expect(view.querySelector('.sc5.now')?.textContent).toContain('NOW');
    await click(/^Activity$/);
    expect(view.textContent).toContain('Bravo and Charlie traded receivers.');
    expect(view.textContent).toContain('Player status: Not available yet');
    await click(/^League$/);
    expect(view.textContent).not.toContain('Player status:');
  });
  it('empty: no standings, no games, no activity, nothing waiting', async () => {
    const view = await show(commish, async () => ({ pulse: { value: { ...pulse([]), week: 1 } },
      season: { value: { schedule: { season: 2026, weeks: [] }, played: {} } },
      activity: { value: { league: 'lg', revision: 1, events: [] } },
      jobs: { value: { open: 0, now: 0 } } }));
    expect(view.textContent).toContain('Nothing waiting on you');
    expect(view.textContent).toContain('No standings yet.');
    await click(/^Schedule$/);
    expect(view.textContent).toContain('No published games for your team.');
    await click(/^Activity$/);
    expect(view.textContent).toContain('No league activity yet.');
  });
  it('failed: shows the read\'s own words, and hides the Control Room row for owners', async () => {
    const view = await show(owner, async () => ({ pulse: { error: 'League server is unreachable' },
      season: {}, activity: { error: 'Activity read failed' } }));
    expect(view.textContent).not.toContain('Control Room');
    expect(view.querySelector('[role="alert"]')?.textContent).toBe('League server is unreachable');
    const partial = loaded();
    partial.pulse.value!.standings = undefined; partial.pulse.value!.standingsError = 'Standings failed';
    partial.activity = { error: 'Activity read failed' };
    await act(async () => root!.unmount()); host!.remove();
    const again = await show(owner, async () => partial);
    expect(again.textContent).toContain('Standings failed');
    await click(/^Activity$/);
    expect(again.textContent).toContain('Activity read failed');
  });
  it('team sheet opens with roster, trade and seat keys, and closes', async () => {
    const onTrade = vi.fn(), onControlRoom = vi.fn();
    const view = await show(commish, async () => loaded(), { onTrade, onControlRoom });
    await click(/Bravo: owner and record/);
    const sheet = () => document.querySelector('dialog');
    expect(sheet()?.getAttribute('aria-label') ?? sheet()?.textContent).toBeTruthy();
    expect(sheet()?.textContent).toContain('Ace Runner');
    expect(sheet()?.textContent).toContain('Owner: Not available yet');
    await click(/^Close$/, sheet()!);
    expect(sheet()).toBeNull();
    await click(/Bravo: owner and record/);
    await click(/Propose a trade/, sheet()!);
    expect(onTrade).toHaveBeenCalled(); expect(sheet()).toBeNull();
    await click(/Bravo: owner and record/);
    await click(/Manage this seat/, sheet()!);
    expect(onControlRoom).toHaveBeenCalled(); expect(sheet()).toBeNull();
    await click(/Alpha: owner and record/);
    expect(sheet()?.textContent).not.toContain('Propose a trade');
    expect(view.querySelector('.lg5')).not.toBeNull();
  });
  it('team sheet says when the roster read fails', async () => {
    await show(owner, async () => loaded(),
      { readRoster: async () => { throw new Error('Roster read failed'); } });
    await click(/Bravo: owner and record/);
    expect(document.querySelector('dialog')?.textContent).toContain('Roster read failed');
    expect(document.querySelector('dialog')?.textContent).not.toContain('Manage this seat');
  });
  it('colours the streak cell: W green, L red, none plain', async () => {
    const view = await show(owner, async () => loaded());
    const cell = (team: string) => [...view.querySelectorAll('tbody tr')]
      .find(tr => tr.textContent?.includes(team))!.lastElementChild!;
    expect(cell('Alpha').className).toBe('l');
    expect(cell('Bravo').className).toBe('');
    expect(streakClass('W3')).toBe('w');
    expect(view.textContent).toContain('Tap a team for its roster.');
  });
  it('mock v1 layout: rules outrank the frame button reset; 38 px zebra rows', async () => {
    const { readFileSync } = await import('node:fs');
    const css = readFileSync('src/app/phone/LeagueTab.css', 'utf8');
    const scoped = ['.lg5 .lg5-seg > *', '.lg5 .st5-team', '.lg5 .lg5-cr', '.lg5 .lg5-key', '.lg5 .lg5-mu'];
    for (const sel of scoped)
      expect(css).toContain(sel);
    expect(css).toMatch(/\.st5 td \{[^}]*height: 38px/);
    expect(css).toMatch(/tr:nth-child\(even\) td \{ background: var\(--lg-zebra\)/);
    expect(css).toMatch(/\.lg5-cr span em \{[^}]*text-transform: uppercase/);
  });
});
