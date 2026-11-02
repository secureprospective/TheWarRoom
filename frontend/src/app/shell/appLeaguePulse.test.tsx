// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { TargetApp } from './TargetApp';
import { commands } from '../commands/registry';
import { LEAGUE_KEY, readAppLeaguePeople } from './leaguePicker';
import { FRANCHISE_KEY } from './franchise';
import { localWorkflow } from './localWorkflow';
import type { People } from '../commissioner/people';
import pulse from './pulse.wire.testdata.json';

vi.mock('../data/provider', async original => {
  const actual = await original<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
  return { ...actual, selectProvider: () => fixture };
});
vi.mock('../data/api', async original => ({
  ...await original<typeof import('../data/api')>(),
  targetLeagues: async () => [{ id: 'native.seed', name: 'Held league', teams: 32, app: true }],
}));
vi.mock('./leaguePicker', async original => ({
  ...await original<typeof import('./leaguePicker')>(), readAppLeaguePeople: vi.fn(),
}));
vi.mock('./localWorkflow', () => ({ localWorkflow: vi.fn() }));

const LEAGUE = 'native.seed';
const people: People = {
  league: LEAGUE, name: 'Held league', revision: 8,
  seats: pulse.directory.franchises.map(f => ({ franchise: f.franchise, name: f.name,
    abbreviation: f.abbreviation, owner: f.owner })),
  roles: [], invites: [], dotCount: 0, dotStatus: '', votes: [], succession: '',
  yourRole: 'Owner', yourFranchise: 'franchise.01', mayChangeRules: false, mayGrantRoles: false,
  mayPostNotices: false,
};
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
async function settle() {
  for (let i = 0; i < 25; i++)
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
}
async function mount() {
  await act(async () => { root.render(<TargetApp />); });
  await settle();
}
function servePulse(results: unknown) {
  vi.mocked(localWorkflow).mockReset().mockImplementation(async request => {
    const league = request.arguments.league;
    const franchise = 'franchise' in request.arguments ? String(request.arguments.franchise) : '';
    const lineups = pulse.lineups as Record<string, { league: string }>;
    const data: Record<string, unknown> = {
      'native.directory.read': { ...pulse.directory, league },
      'week.read': { ...pulse.week,
        schedule: { ...pulse.week.schedule, league, franchise } },
      'standings.read': { ...pulse.standings, league },
      'bracket.read': { ...pulse.bracket, league },
      'results.read': { ...(results as object), league },
      'projections.read': { ...pulse.projections, league,
        players: pulse.projections.players.filter(p =>
          String((request.arguments as { players?: string }).players ?? '').split(',').includes(p.player)) },
      'lineup.read': lineups[franchise] ? { ...lineups[franchise], league } : undefined,
    };
    if (!(request.command in data) || data[request.command] === undefined)
      throw new Error(`Unavailable fixture: ${request.command} ${franchise}`);
    return { label: 'Fake local port', writable: false, football: '', data: data[request.command] };
  });
}
beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
  localStorage.clear();
  localStorage.setItem(LEAGUE_KEY, LEAGUE);
  localStorage.setItem(`${FRANCHISE_KEY}:${LEAGUE}`, 'franchise.01');
  commands.dispatch('nav.open', { node: 'pulse', workspace: 'now' });
  vi.mocked(readAppLeaguePeople).mockImplementation(async id => ({ ...people, league: id }));
  servePulse(pulse.results);
  container = document.createElement('div'); document.body.append(container); root = createRoot(container);
});
afterEach(async () => {
  await act(async () => { root.unmount(); }); container.remove();
  localStorage.clear(); vi.unstubAllGlobals(); delete (window as unknown as { go?: unknown }).go;
});

it('renders the Now tab from the week the league has actually scored', async () => {
  await mount();
  await vi.waitFor(async () => {
    await settle();
    expect(container.querySelector('.lp-v14 #ws-live .scroll')?.textContent)
      .toContain('Held team 32');
  }, { timeout: 4000 });
  const list = container.querySelector('#ws-live .scroll')!;
  // results.json: week 2, franchise.32 beat franchise.01 3 to 1.5, every other game tied.
  expect(list.textContent).toContain('MY MATCHUP');
  expect(list.textContent).toContain('3.0');
  expect(list.textContent).toContain('1.5');
  expect(list.textContent).toContain('FINAL');
  expect(container.querySelector('.sline')?.textContent).toContain('WEEK 2');
  expect(container.querySelector('.sline .slate')?.textContent).toContain('16 with a result');
  expect(container.querySelector('.sline .mybox')?.textContent).toContain('1.5');
  // The starters spine reads the week score's starters and the lineup's NFL team.
  const spine = container.querySelector('#ws-live .spine')!;
  expect(spine.textContent).toContain('franchise.01 starter');
  expect(spine.textContent).toContain('AAA');
  expect(spine.textContent).toContain('LOCKED');
  expect(spine.textContent).toContain('proj 9.5');
  // Nothing claims a fact the league does not hold.
  expect(container.textContent).not.toContain('Not available yet');
  expect(container.textContent).not.toContain('Not wired');
  // The NFL side reads the held live view; with none held it says so instead of inventing a game.
  expect(container.querySelector('#ws-live .follow')?.textContent).toContain('FOLLOWING');
  expect(container.querySelector('#ws-live')?.textContent).not.toContain('needs the NFL read');
  const nfl = [...container.querySelectorAll('#ws-live .seg button')]
    .find(b => b.textContent === 'NFL GAMES')!;
  await act(async () => { (nfl as HTMLButtonElement).click(); });
  await settle();
  expect(container.querySelector('#ws-live .scroll .idle')?.textContent)
    .toMatch(/NFL scoreboard|NFL games/);
});

it('renders the Race tab standings, playoff field and road from the same reads', async () => {
  commands.dispatch('nav.open', { node: 'pulse', workspace: 'race' });
  await mount();
  await vi.waitFor(async () => {
    await settle();
    expect(container.querySelector('.lp-v14 #ws-race .stand')?.textContent).toContain('Held team');
  }, { timeout: 4000 });
  const standings = container.querySelector('#ws-race .stand')!;
  expect(standings.textContent).toContain('2-1');
  expect(standings.textContent).toContain('7.5 points scored');
  expect(container.querySelector('#ws-race #racec')?.textContent).toContain('MY SEED TODAY');
  expect(container.querySelector('#ws-race #racec')?.textContent).toContain('PLACES ON THIS SIDE');
  expect(container.querySelector('#ws-race #road')?.textContent).toContain('MY ROAD');
  expect(container.querySelector('#ws-race #road')?.textContent).toContain('Held team');
  expect(container.textContent).not.toContain('Not wired');
  expect(container.textContent).not.toContain('Not available yet');
});

it('says plainly that no schedule is published instead of reporting a failure', async () => {
  servePulse(pulse.unpublished);
  await mount();
  await vi.waitFor(async () => {
    await settle();
    expect(container.querySelector('.lp-v14 #ws-live')?.textContent)
      .toContain('schedule is not published yet');
  }, { timeout: 4000 });
  expect(container.querySelector('#ws-live .scroll')?.textContent)
    .toContain('Games appear here as soon as');
  expect(container.querySelector('.sline .slate')?.textContent)
    .toContain('schedule is not published yet');
  expect(container.textContent).not.toContain('Not available yet');
  expect(container.textContent).not.toContain('unavailable');
});
