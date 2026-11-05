// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import AppLeagueMount from './AppLeagueMount';
import { localWorkflow } from './localWorkflow';
import assets from '../../../../testdata/wire/native/assets.json';

const season = vi.hoisted(() => ({ changed: new Set<() => void>() }));
vi.mock('../data/provider', async real => {
  const actual = await real<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
  fixture.onSeasonChange = f => { season.changed.add(f); return () => { season.changed.delete(f); }; };
  return { ...actual, selectProvider: () => fixture };
});
vi.mock('./leaguePicker', async real => ({ ...await real<typeof import('./leaguePicker')>(),
  readAppLeaguePeople: async (league: string) => ({ league, yourFranchise: 'franchise.01',
    seats: [{ franchise: 'franchise.01', name: 'One' }] }) }));
vi.mock('./localWorkflow', () => ({ localWorkflow: vi.fn() }));
vi.mock('./NativeFranchiseHQ', () => ({ default: () => <p>HQ</p> }));
it('reuses actual War Room readings on revisit and refreshes them on a season revision', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  vi.mocked(localWorkflow).mockImplementation(async request => {
    if (request.command !== 'native.assets.read') throw new Error('Region not held');
    return { label: 'Local', writable: false, football: '', data: { ...assets, league: 'native.seed' } };
  });
  const host = document.createElement('div'); document.body.append(host);
  const root = createRoot(host);
  const render = async (view: 'war' | 'hq') => {
    await act(async () => { root.render(<AppLeagueMount league={{ id: 'native.seed', name: 'League',
      teams: 32, app: true }} view={view} leave={() => {}} />); });
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 150)); });
  };
  try {
    await render('war');
    await vi.waitFor(async () => {
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
      expect(host.querySelector('[aria-label="Trade desk"]')).not.toBeNull();
    }, { timeout: 5000 });
    const reads = vi.mocked(localWorkflow).mock.calls.length;
    await render('hq'); await render('war');
    expect(vi.mocked(localWorkflow).mock.calls.length).toBe(reads);
    expect(host.querySelector('[aria-label="Trade desk"]')).not.toBeNull();
    await act(async () => { season.changed.forEach(f => f()); });
    expect(vi.mocked(localWorkflow).mock.calls.length).toBeGreaterThan(reads);
  } finally {
    await act(async () => root.unmount()); host.remove(); vi.unstubAllGlobals();
  }
});
