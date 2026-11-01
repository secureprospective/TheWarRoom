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
import assets from '../../../../testdata/wire/native/assets.json';
import lineup from '../../../../testdata/wire/native/lineup.json';
import week from '../../../../testdata/wire/native/week.json';
import rules from '../../../../testdata/wire/commissioner/rules.json';
import projections from '../data/projections.wire.testdata.json';
import cap from '../../../../testdata/wire/native/cap-check.json';

vi.mock('../data/provider', async (original) => {
  const actual = await original<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
  const baseClock = await fixture.clock();
  fixture.clock = async () => ({ ...baseClock, value: { ...baseClock.value, week: 8 } });
  return { ...actual, selectProvider: () => fixture };
});
vi.mock('../data/api', async (original) => ({
  ...await original<typeof import('../data/api')>(),
  targetLeagues: async () => ['native.seed', 'native.other'].map(id => ({
    id, name: id, teams: 12, app: true,
  })),
}));
vi.mock('./leaguePicker', async original => ({
  ...await original<typeof import('./leaguePicker')>(), readAppLeaguePeople: vi.fn(),
}));
vi.mock('./localWorkflow', () => ({ localWorkflow: vi.fn() }));

const people: People = {
  league: 'native.seed', name: 'App league', revision: 7,
  seats: [{ franchise: 'franchise.01', name: 'First team', abbreviation: 'A', owner: 'a' },
    { franchise: 'franchise.02', name: 'Second team', abbreviation: 'B', owner: 'b' }],
  roles: [], invites: [], dotCount: 0, dotStatus: '', votes: [], succession: '',
  yourRole: 'Commissioner', yourFranchise: '', mayChangeRules: true, mayGrantRoles: true,
  mayPostNotices: true,
};
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
async function settle() {
  for (let i = 0; i < 20; i++)
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
}
async function mount() {
  await act(async () => { root.render(<TargetApp />); });
  await settle();
}
async function choose(id: string) {
  const select = container.querySelector('.rail .me select') as HTMLSelectElement;
  expect(select).not.toBeNull();
  await act(async () => { select.value = id; select.dispatchEvent(new Event('change', { bubbles: true })); });
  await settle();
}
beforeEach(() => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
  localStorage.clear();
  localStorage.setItem(LEAGUE_KEY, 'native.seed');
  commands.dispatch('nav.open', { node: 'hq', workspace: 'lineup-and-roster' });
  vi.mocked(readAppLeaguePeople).mockImplementation(async league => ({ ...people, league }));
  vi.mocked(localWorkflow).mockReset().mockImplementation(async request => {
    const league = request.arguments.league;
    const franchise = 'franchise' in request.arguments ? request.arguments.franchise : 'franchise.02';
    const data: Record<string, unknown> = {
      'native.assets.read': { ...assets, league },
      'projections.read': { ...projections, league, week: request.command === 'projections.read' ?
        request.arguments.week : 1 },
      'commish.rules.read': { ...rules, league, revision: assets.revision },
      'native.cap.check': { ...cap, league },
      'week.read': { ...week, schedule: { ...week.schedule, league, franchise, revision: assets.revision } },
      'lineup.read': { ...lineup, league, franchise, revision: assets.revision },
    };
    if (!(request.command in data)) throw new Error(`Unavailable fixture: ${request.command}`);
    return { label: 'Fake local port', writable: false, football: '', data: data[request.command] };
  });
  container = document.createElement('div'); document.body.append(container); root = createRoot(container);
});
afterEach(async () => {
  await act(async () => { root.unmount(); }); container.remove();
  localStorage.clear(); vi.unstubAllGlobals(); delete (window as unknown as { go?: unknown }).go;
});

it.each(['hq', 'war'] as const)('renders %s cap and roster while values are pending, then publishes values once', async node => {
  const original = vi.mocked(localWorkflow).getMockImplementation()!;
  let finish: () => void = () => {};
  const pending = new Promise<void>(resolve => { finish = resolve; });
  vi.mocked(localWorkflow).mockImplementation(async request => {
    if (request.command !== 'value.read') return original(request);
    await pending;
    const { league, franchise, season } = request.arguments;
    return { label: 'Local', writable: false, football: '', data: { league, franchise, season,
      players: [], best: { franchise, starterN: 21, value: 243.3, age: 27, ageKnown: true,
        rank: 3, of: 32, available: true, words: '35 of 35 players valued' }, age: [], notes: [] } };
  });
  localStorage.setItem(`${FRANCHISE_KEY}:native.seed`, 'franchise.02');
  commands.dispatch('nav.open', { node });
  await mount();
  expect(container.querySelector(node === 'war' ? '.wr-v6 .shelf' : '.hq-v10 .board')?.textContent)
    .toContain('Player, 1000');
  expect(container.textContent).toContain('2026');
  if (node === 'war') expect(container.querySelector('.runw')?.textContent).not.toContain('Not available yet');
  expect(vi.mocked(localWorkflow).mock.calls.filter(([r]) => r.command === 'value.read')).toHaveLength(1);
  await act(async () => { finish(); }); await settle();
  if (node === 'war') expect(container.querySelector('.franchise-context')?.textContent).toContain('243.3');
  expect(vi.mocked(localWorkflow).mock.calls.filter(([r]) => r.command === 'value.read')).toHaveLength(1);
});

it('shows a no-seat shell chooser, then native roster and contracts from a fake port', async () => {
  await mount();
  expect(container.querySelector('.rail nav[aria-label="Workspaces"]')).not.toBeNull();
  expect(container.textContent).toContain('Choose a franchise to view');
  expect(localWorkflow).not.toHaveBeenCalled();
  await choose('franchise.02');
  expect(container.querySelector('.hq-v10 .board')?.textContent)
    .toContain('Player, 1000');
  expect(container.querySelector('.hq-v10')).not.toBeNull();
  expect(container.querySelector('.hq-v10 .insp')?.textContent).toContain('Projection10.0');
  expect(container.querySelector('.hq-v10 .tray')?.textContent).not.toContain('OPEN MFL');
  expect(document.querySelector('body > .hq-v10')).toBeNull();
  expect(container.querySelectorAll('.rail')).toHaveLength(1);
  expect(localStorage.getItem(`${FRANCHISE_KEY}:native.seed`)).toBe('franchise.02');
  expect(vi.mocked(localWorkflow).mock.calls.some(([r]) => r.command === 'lineup.read' &&
    r.arguments.franchise === 'franchise.02')).toBe(true);
  expect(container.querySelector('.hq-v10 .ws-tabs')).toBeNull();
  expect(container.querySelector('.target-frame')?.getAttribute('data-native-workspace')).toBe('true');
  expect(container.querySelector('.target-frame > .insp')).toBeNull();
  const contracts = [...container.querySelectorAll('.wtabs button')].find(
    b => b.textContent?.toLowerCase() === 'contracts and cap');
  await act(async () => { (contracts as HTMLButtonElement).click(); });
  expect(container.querySelector('.hq-v10')?.textContent).toContain('Player, 1000');
  expect(vi.mocked(localWorkflow).mock.calls.every(([r]) => r.mode === 'read')).toBe(true);
  await act(async () => { commands.dispatch('nav.open', { node: 'war' }); });
  await settle();
  expect(container.querySelector('.wr-v6.embedded')).not.toBeNull();
  expect(container.querySelector('.wr-v6 .ws-tabs')).toBeNull();
  expect(container.querySelector('.wr-v6 .work.external-tabs')).not.toBeNull();
  expect(container.querySelector('.wr-v6 .franchise-context')?.classList.contains('fixed')).toBe(false);
  expect(container.querySelector('.target-frame > .insp')).toBeNull();
  const tabs = [...container.querySelectorAll<HTMLButtonElement>('.wtabs [role="tab"]')];
  expect(tabs.map(t => t.textContent)).toEqual(['TRADE DESK', 'MARKET', 'DRAFT ROOM', 'PLANS']);
  await act(async () => { tabs[1].click(); }); await settle();
  expect(tabs[1].getAttribute('aria-selected')).toBe('true');
  expect(container.querySelector('.wr-v6 [aria-label="Market"]')).not.toBeNull();
  await act(async () => { tabs[0].click(); }); await settle();
  expect(container.querySelector('.wr-v6 .shelf')?.textContent).toContain('Player, 1000');
  expect(container.querySelectorAll('.rail')).toHaveLength(1);
  expect((container.querySelector('.me select') as HTMLSelectElement).value).toBe('franchise.02');
});

it('discards partial and final publications from a franchise that was left', async () => {
  const original = vi.mocked(localWorkflow).getMockImplementation()!;
  let finish: () => void = () => {};
  const pending = new Promise<void>(resolve => { finish = resolve; });
  vi.mocked(localWorkflow).mockImplementation(async request => {
    if (request.command !== 'value.read') return original(request);
    const { league, franchise, season } = request.arguments;
    if (franchise === 'franchise.01') await pending;
    return { label: 'Local', writable: false, football: '', data: { league, franchise, season,
      players: [], best: { franchise, starterN: 21, value: franchise === 'franchise.01' ? 999 : 243.3,
        age: 27, ageKnown: true, rank: 3, of: 32, available: true, words: 'Held values' }, age: [], notes: [] } };
  });
  localStorage.setItem(`${FRANCHISE_KEY}:native.seed`, 'franchise.01');
  commands.dispatch('nav.open', { node: 'war' });
  await mount(); await choose('franchise.02');
  expect(container.querySelector('.franchise-context')?.textContent).toContain('243.3');
  await act(async () => { finish(); }); await settle();
  expect(container.querySelector('.franchise-context')?.textContent).toContain('243.3');
  expect(container.querySelector('.franchise-context')?.textContent).not.toContain('999');
});

it('defaults to a held seat but restores only a valid choice for the selected league', async () => {
  vi.mocked(readAppLeaguePeople).mockImplementation(async league => ({ ...people, league,
    yourFranchise: 'franchise.01' }));
  localStorage.setItem(`${FRANCHISE_KEY}:native.seed`, 'removed-seat');
  localStorage.setItem(`${FRANCHISE_KEY}:native.other`, 'franchise.01');
  await mount();
  expect((container.querySelector('.me select') as HTMLSelectElement).value).toBe('franchise.01');
  await choose('franchise.02');
  await act(async () => { commands.dispatch('nav.open', { node: 'home' }); });
  await act(async () => { commands.dispatch('nav.open', { node: 'hq' }); });
  await settle();
  expect((container.querySelector('.me select') as HTMLSelectElement).value).toBe('franchise.02');
  await act(async () => { container.querySelector<HTMLButtonElement>('.league .pick')!.click(); });
  const other = [...container.querySelectorAll('.league li button')].find(
    b => b.textContent?.includes('native.other'));
  await act(async () => { (other as HTMLButtonElement).click(); });
  await settle();
  expect((container.querySelector('.me select') as HTMLSelectElement).value).toBe('franchise.01');
  expect(localStorage.getItem(`${FRANCHISE_KEY}:native.seed`)).toBe('franchise.02');
});

it('shows an imported-shaped 35-player roster before any schedule is published', async () => {
  const original = vi.mocked(localWorkflow).getMockImplementation()!;
  const imported = structuredClone(assets);
  const player = imported.snapshot.players[0];
  const players = Array.from({ length: 35 }, (_, i) => ({ ...player, id: String(2000 + i),
    franchise: 'franchise.02', name: `Imported player ${i}`, salaryCents: 300_000_000,
    expirationYear: 2027, contractStatus: '',
    heldContractStatus: 'EXT (2024)', contractInfo: 'Imported contract' }));
  imported.snapshot.players = players;
  vi.mocked(localWorkflow).mockImplementation(async request => {
    const data = request.command === 'native.assets.read' ? imported : request.command === 'week.read' ?
      { schedule: { league: 'native.seed', franchise: 'franchise.02', season: imported.snapshot.season,
        revision: imported.revision, games: [] }, message: 'The schedule is not published yet.' } :
      request.command === 'lineup.read' ? { ...lineup, week: 8, revision: imported.revision, starters: [],
        bench: players.map(p => ({ player: p.id, name: p.name, position: 'QB',
          locked: false, words: 'No NFL game this week' })) } : undefined;
    return data ? { label: 'Import shape', writable: false, football: '', data } : original(request);
  });
  await mount(); await choose('franchise.02');
  expect(container.textContent).toContain('Viewing week 8 · The schedule is not published yet.');
  expect(container.querySelectorAll('.hq-v10 .blist [data-player]')).toHaveLength(35);
  expect(container.querySelector('.hq-v10 .blist')?.textContent).toContain('Imported player 34');
  expect(container.querySelector('.hq-v10 .blist .roster-contract')?.textContent).toBe('$3.00M · 2y');
  await act(async () => { commands.dispatch('nav.open', { node: 'war' }); }); await settle();
  expect(container.querySelectorAll('.wr-v6 .shelf:first-child .blk')).toHaveLength(35);
  expect(container.querySelector('.wr-v6 .shelf')?.textContent).toContain('Imported player 34');
  expect(container.querySelector('.wr-v6 .shelf .nm small')?.textContent).toBe('$3.00M · 2y');
});
