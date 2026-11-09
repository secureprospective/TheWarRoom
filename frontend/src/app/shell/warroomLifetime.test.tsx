// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import AppLeagueMount from './AppLeagueMount';
import { localWorkflow } from './localWorkflow';
import assets from '../../../../testdata/wire/native/assets.json';
import pulse from './pulse.wire.testdata.json';

const season = vi.hoisted(() => ({ changed: new Set<() => void>() }));
vi.mock('../data/provider', async real => {
  const actual = await real<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
  fixture.onSeasonChange = f => { season.changed.add(f); return () => { season.changed.delete(f); }; };
  return { ...actual, selectProvider: () => fixture };
});
vi.mock('../market/api', async real => ({ ...await real<typeof import('../market/api')>(),
  subscribeSeason: (f: () => void) => { season.changed.add(f); return () => {
    season.changed.delete(f); }; } }));
vi.mock('./leaguePicker', async real => ({ ...await real<typeof import('./leaguePicker')>(),
  readAppLeaguePeople: async (league: string) => ({ league, yourFranchise: 'franchise.01',
    seats: [{ franchise: 'franchise.01', name: 'One' }, { franchise: 'franchise.02', name: 'Two' }] }) }));
vi.mock('./localWorkflow', () => ({ localWorkflow: vi.fn() }));
vi.mock('../control/room/ControlRoom', () => ({ default: () => <p>Control</p> }));
it.each([
  ['war', '[aria-label="Trade desk"]'], ['hq', '.hq-v10'],
  ['home', '[data-testid="home-brief"]'], ['pulse', '.lp-v14'],
] as const)('keeps actual %s reads across navigation, refreshes hidden reads, and resets scope',
  async (screen, selector) => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.mocked(localWorkflow).mockClear();
  vi.mocked(localWorkflow).mockImplementation(async request => {
    const league = request.arguments.league;
    const data: Record<string, unknown> = {
      'native.assets.read': { ...assets, league },
      'native.directory.read': { ...pulse.directory, league },
      'week.read': { ...pulse.week, schedule: { ...pulse.week.schedule, league,
        franchise: 'franchise' in request.arguments ? request.arguments.franchise : '' } },
      'standings.read': { ...pulse.standings, league },
      'bracket.read': { ...pulse.bracket, league },
      'results.read': { ...pulse.results, league },
    };
    if (!(request.command in data)) throw new Error('Region not held');
    return { label: 'Local', writable: false, football: '', data: data[request.command] };
  });
  const host = document.createElement('div'); document.body.append(host);
  const root = createRoot(host);
  const render = async (view: 'war' | 'hq' | 'home' | 'pulse' | 'control', league = 'native.seed') => {
    await act(async () => { root.render(<AppLeagueMount key={league} league={{ id: league, name: 'League',
      teams: 32, app: true }} view={view} leave={() => {}} />); });
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 150)); });
  };
  try {
    await render(screen);
    await vi.waitFor(async () => {
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
      expect(host.querySelector(selector)).not.toBeNull();
    }, { timeout: 5000 });
    const element = host.querySelector(selector);
    const reads = vi.mocked(localWorkflow).mock.calls.length;
    expect(reads).toBeGreaterThan(0);
    await render('control'); await render(screen);
    expect(vi.mocked(localWorkflow).mock.calls.length).toBe(reads);
    expect(host.querySelector(selector)).toBe(element);
    await render('control');
    await act(async () => { season.changed.forEach(f => f()); });
    expect(vi.mocked(localWorkflow).mock.calls.length).toBeGreaterThan(reads);
    await render(screen);
    const select = host.querySelector('select')!;
    const before = vi.mocked(localWorkflow).mock.calls.length;
    await act(async () => {
      select.value = 'franchise.02'; select.dispatchEvent(new Event('change', { bubbles: true }));
    });
    expect(host.querySelector(selector)).not.toBe(element);
    expect(vi.mocked(localWorkflow).mock.calls.length).toBeGreaterThan(before);
    for (const view of ['home', 'hq', 'pulse', 'war', 'control'] as const) await render(view);
    expect(host.querySelectorAll(':scope > div')).toHaveLength(5);
    const second = host.querySelector(selector);
    await render(screen, 'native.other');
    expect(host.querySelector(selector)).not.toBe(second);
    expect(vi.mocked(localWorkflow).mock.calls.some(([r]) => r.arguments.league ===
      'native.other')).toBe(true);
    expect(host.querySelectorAll(':scope > div')).toHaveLength(1);
  } finally {
    await act(async () => root.unmount()); host.remove(); localStorage.clear(); vi.unstubAllGlobals();
    expect(season.changed.size).toBe(0);
  }
});
