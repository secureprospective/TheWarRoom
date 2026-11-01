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
import cap from '../../../../testdata/wire/native/cap-check.json';

vi.mock('../data/provider', async (original) => {
  const actual = await original<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
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

it('shows a no-seat shell chooser, then native roster and contracts from a fake port', async () => {
  await mount();
  expect(container.querySelector('.rail nav[aria-label="Workspaces"]')).not.toBeNull();
  expect(container.textContent).toContain('Choose a franchise to view');
  expect(localWorkflow).not.toHaveBeenCalled();
  await choose('franchise.02');
  expect(container.querySelector('.hq-v10 .board')?.textContent)
    .toContain('Player, 1000');
  expect(container.querySelector('.hq-v10')).not.toBeNull();
  expect(document.querySelector('body > .hq-v10')).toBeNull();
  expect(container.querySelectorAll('.rail')).toHaveLength(1);
  expect(localStorage.getItem(`${FRANCHISE_KEY}:native.seed`)).toBe('franchise.02');
  expect(vi.mocked(localWorkflow).mock.calls.some(([r]) => r.command === 'lineup.read' &&
    r.arguments.franchise === 'franchise.02')).toBe(true);
  const contracts = [...container.querySelectorAll('.hq-v10 button')].find(
    b => b.textContent === 'CONTRACTS AND CAP');
  await act(async () => { (contracts as HTMLButtonElement).click(); });
  expect(container.querySelector('.hq-v10')?.textContent).toContain('Player, 1000');
  expect(vi.mocked(localWorkflow).mock.calls.every(([r]) => r.mode === 'read')).toBe(true);
  await act(async () => { commands.dispatch('nav.open', { node: 'war' }); });
  await settle();
  expect(container.querySelector('.wr-v6.embedded')).not.toBeNull();
  expect(container.querySelectorAll('.rail')).toHaveLength(1);
  expect((container.querySelector('.me select') as HTMLSelectElement).value).toBe('franchise.02');
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
