// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { LiveTab, surname } from './LiveTab';

const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
const state = { possession: '12', down: 2, distance: 7, yardLine: 57, toEndzone: 57 };
const play = { id: 'p1', sequence: 1, text: 'Mahomes pass short right', clock: '8:42', period: 2,
  at: '2026-11-15T18:00:00Z', homeScore: 7, awayScore: 3, start: state, end: state, participants: [] };
const nflGame = { season: 2026, week: 11, seasonType: 2, id: '401', kickoff: '2026-11-15T18:00:00Z',
  home: { id: '12', abbreviation: 'KC', score: 7, record: '8-2' },
  away: { id: '2', abbreviation: 'BUF', score: 3 },
  status: 'in', clock: '8:42', period: 2, state, last: play, plays: [play],
  box: [{ team: 'KC', category: 'passing', labels: ['C/ATT', 'YDS'], rows: [{ name: 'P. Mahomes',
    stats: ['6/10', '95'] }] }] };
const reply = (data: unknown) => ({ data, football: '', writable: false, label: 'Server' });
function fakePort(nfl: 'ok' | 'fail' = 'ok') {
  return vi.fn<HostedPort>(async request => {
    if (nfl === 'fail') throw new Error('scoreboard down');
    if (request.command === 'nfl.games.read') return reply({ league: 'league', board: { stale: false,
      source: 'ESPN', at: '2026-11-15T18:00:00Z', failure: '', games: [nflGame] } });
    if (request.command === 'nfl.game.read') return reply({ league: 'league', game: nflGame,
      at: '2026-11-15T18:00:00Z', stale: false, words: '' });
    throw new Error(`Unexpected read ${request.command}`);
  });
}
const starter = (player: string, name: string, position: string, provisional: number) =>
  ({ player, name, position, words: '', provisional });
const brief = (games: unknown[]): HomeBrief => ({ pulse: { league: 'league', name: 'Test League',
  season: 2026, week: 11, lastWeek: 18, weekNote: '', seats: [], fixtures: [], forms: {}, errors: [],
  odds: { games: [{ week: 11, home: 'mine', away: 'other', homeWin: 0.44 }], rows: [] } },
results: { league: 'league', season: 2026, week: 11, revision: 1, published: true, words: '',
  games, missed: [], scores: [
    { franchise: 'mine', name: 'Iron Curtain', status: '', provisional: 24.2, words: '', adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false,
      starters: [starter('m1', 'Mahomes, Patrick', 'QB', 18.1)] },
    { franchise: 'other', name: 'Gridlock', status: '', provisional: 25.8, words: '', adjustment: 0,
      lineupStatus: '', lineupWords: '', flagged: false, recorded: false,
      starters: [starter('o1', 'Allen, Josh', 'QB', 20.4)] }] },
matchup: { lineups: { mine: [{ player: 'm1', team: 'KC', locked: true }],
  other: [{ player: 'o1', team: 'BUF', locked: true }] }, projections: {}, errors: [] },
at: {}, errors: {} } as unknown as HomeBrief);
const myGameRow = { week: 11, home: 'mine', away: 'other', homeName: 'Iron Curtain', awayName: 'Gridlock',
  status: 'in', homeProvisional: 24.2, awayProvisional: 25.8, words: '', why: [] };

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});
async function show(readBrief: () => Promise<HomeBrief>, port: HostedPort = fakePort()) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  HTMLDialogElement.prototype.showModal ??= function (this: HTMLDialogElement) { this.open = true; };
  HTMLDialogElement.prototype.close ??= function (this: HTMLDialogElement) { this.open = false; };
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<LiveTab league={league} port={port} readBrief={readBrief}
    subscribe={() => () => {}} />));
  await act(async () => { await new Promise(r => setTimeout(r, 0)); });
  return host;
}
const tap = async (label: string) => {
  const key = [...host!.querySelectorAll<HTMLElement>('button')]
    .find(b => b.getAttribute('aria-label') === label || b.textContent === label);
  if (!key) throw new Error(`No control ${label}`);
  await act(async () => key.click());
};
const region = (name: string) =>
  host!.querySelector(`:is(section,nav)[aria-label="${name}"]`)?.textContent ?? '';

describe('LiveTab', () => {
  it('lays out the mock Live page from the held reads when loaded', async () => {
    await show(async () => brief([myGameRow]));
    expect(region('My matchup')).toContain('My matchup · wk 11');
    expect(region('My matchup')).toContain('Iron Curtain · YOU');
    expect(region('My matchup')).toContain('44% WIN');
    expect(region('My matchup')).toContain('Mahomes');
    expect(region('NFL games')).toContain('BUF3KC7Q2 8:42');
    expect(region('League matchups')).toBe('');
    expect(region('NFL live')).toContain('2nd & 7');
    expect(region('NFL live')).toContain('KC 43');
    expect(region('NFL live')).toContain('57 YD');
    expect(host!.querySelector('.live6__fresh')?.textContent).toContain('updated');
    expect(region('NFL box score')).toContain('P. Mahomes');
    expect(region('Play by play')).toContain('Mahomes pass short right');
    await tap('Matchup box score');
    expect(region('Matchup box score')).toContain('Patrick Mahomes');
    expect(region('Matchup box score')).toContain('Josh Allen');
  });
  it('opens and closes the all-matchups sheet, and opens a starter row detail', async () => {
    await show(async () => brief([myGameRow]));
    await tap('All matchups');
    const sheet = host!.querySelector('dialog');
    expect(sheet?.textContent).toContain('Week 11 matchups');
    expect(sheet?.textContent).toContain('YOUR MATCHUP');
    await tap('Close Week 11 matchups');
    expect(host!.querySelector('dialog')).toBeNull();
    await tap('Watch Patrick Mahomes\'s game');
    expect(host!.querySelector('dialog')?.textContent).toContain('18.1');
    await act(async () => host!.querySelector('dialog')!.dispatchEvent(new Event('cancel')));
    expect(host!.querySelector('dialog')).toBeNull();
  });
  it('says there is no game when the week holds none for me', async () => {
    await show(async () => brief([]));
    expect(region('My matchup')).toContain('No game this week');
    expect(region('My matchup')).not.toContain('WIN');
  });
  it('shows each read\'s own failure words when the reads fail', async () => {
    await show(async () => { throw new Error('results.read refused'); }, fakePort('fail'));
    expect(host!.querySelector('[aria-label="My matchup"] [role="alert"]')?.textContent)
      .toContain('results.read refused');
    expect(region('League matchups')).toBe('');
  });
  it('keeps the matchup header as one row: title, then both controls', async () => {
    await show(async () => brief([myGameRow]));
    const row = host!.querySelector<HTMLElement>('[aria-label="My matchup"] .live6__head')!;
    expect(row.children[0].tagName).toBe('B');
    expect(row.querySelectorAll(':scope > span > .live6__link')).toHaveLength(2);
  });
  it('names starter tiles by surname, as the mock does', async () => {
    expect(surname('Patrick Mahomes')).toBe('Mahomes');
    expect(surname('Amon-Ra St. Brown')).toBe('St. Brown');
    expect(surname('Chiefs')).toBe('Chiefs');
    await show(async () => brief([myGameRow]));
    expect(host!.querySelector('.live6__pl span')?.textContent).toBe('Mahomes');
  });
  it('draws the field graphic only when the field position is known', async () => {
    await show(async () => brief([myGameRow]));
    expect(host!.querySelector('[aria-label="NFL live"] .live-field')).not.toBeNull();
  });
  it('shows no field graphic when the game holds no field position', async () => {
    const noPosition = { ...nflGame, state: null };
    const port = vi.fn<HostedPort>(async request => request.command === 'nfl.games.read'
      ? reply({ league: 'league',
        board: { stale: false, source: 'ESPN', at: '', failure: '', games: [noPosition] } })
      : reply({ league: 'league', game: noPosition, at: '', stale: false, words: '' }));
    await show(async () => brief([myGameRow]), port);
    expect(host!.querySelector('[aria-label="NFL live"] .live-field')).toBeNull();
    expect(region('NFL live')).toContain('Field position not supplied');
  });
  it('shows the mock score header, drive bar and last play from the followed game', async () => {
    await show(async () => brief([myGameRow]));
    const live = host!.querySelector('[aria-label="NFL live"]')!;
    expect(live.querySelector('.live6__side.is-r .live6__id')?.textContent).toBe('KC8-2');
    expect(live.querySelector('.live6__side.is-r [aria-label="has the ball"]')).not.toBeNull();
    expect(live.querySelector('.live6__side:not(.is-r) [aria-label="has the ball"]')).toBeNull();
    const bar = host!.querySelector<HTMLElement>('[data-testid="live-drive-bar"]')!;
    expect(bar.dataset.dir).toBe('l');
    expect(bar.querySelector<HTMLElement>('.live6__ball')!.style.left).toBe(`${(100 / 120) * 67}%`);
    expect(bar.querySelector<HTMLElement>('.live6__first')!.style.left).toBe(`${(100 / 120) * 60}%`);
    expect(bar.textContent).toBe('BUF205020KC');
    expect(region('Latest play')).toContain('Q2 8:42');
    expect(region('Latest play')).toContain('Mahomes pass short right');
  });
});
