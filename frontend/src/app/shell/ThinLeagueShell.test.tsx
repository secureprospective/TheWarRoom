// @vitest-environment jsdom
import { act, useState } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { heldSnapshot } from '../data/lineupTestData';
import { WarRoomInspector } from './WarRoomInspector';
import { ThinLeagueShell } from './ThinLeagueShell';
import { findPlayers, type PlayerCandidate, type ThinScreenRoute } from './ThinPlayerSearch';

vi.mock('./warroomReads', async importOriginal => ({
  ...await importOriginal<typeof import('./warroomReads')>(),
  useWarRoomRead: () => ({}),
}));

it('matches Last, First queries against First Last player names', () => {
  expect(findPlayers([{ id: 'pearson', name: 'Jachour Pearson' }], 'Pearson, Jachour'))
    .toEqual([{ id: 'pearson', name: 'Jachour Pearson' }]);
});

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  vi.restoreAllMocks();
});

it('puts Home first and routes Home from another thin view', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  function Probe() {
    const [route, setRoute] = useState<ThinScreenRoute>({ node: 'hq', workspace: 'lineup-and-roster' });
    return <ThinLeagueShell leagueName="Test league" route={route} commissioner
      players={[]} navigate={setRoute} openPlayer={() => {}}>
      <p>{route.node === 'home' ? 'Thin home brief' : 'Franchise HQ'}</p>
    </ThinLeagueShell>;
  }
  await act(async () => root!.render(<Probe />));
  const nav = host.querySelector('nav[aria-label="Workspaces"]')!;
  const nodes = [...nav.querySelectorAll('.node>.nb')];
  expect(nodes.map(node => node.textContent?.replace(/[0-9]/g, '').trim())).toEqual([
    'Home›', 'Franchise HQ›', 'War Room›', 'League Pulse›', 'Control Room›',
  ]);
  await act(async () => { (nodes[0] as HTMLButtonElement).click(); await new Promise(resolve =>
    setTimeout(resolve, 0)); });
  expect(host.textContent).toContain('Thin home brief');
});

it('shows the report entry only outside the phone shell', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const shell = (phoneShell: boolean) => <ThinLeagueShell league="native.test"
    leagueName="Test league" route={{ node: 'home', workspace: 'overview' }} commissioner players={[]}
    navigate={() => {}} openPlayer={() => {}} phoneShell={phoneShell}>
    <p>League Home</p>
  </ThinLeagueShell>;
  await act(async () => root!.render(shell(false)));
  expect([...host.querySelectorAll('button')].some(button =>
    button.textContent === 'Report a problem')).toBe(true);
  await act(async () => root!.render(shell(true)));
  expect([...host.querySelectorAll('button')].some(button =>
    button.textContent === 'Report a problem')).toBe(false);
});

it('matches case-insensitive name fragments and opens the card', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const player: PlayerCandidate = { id: '0007', name: 'Rookie Runningback', position: 'RB' };
  const snapshot = heldSnapshot();
  function Probe() {
    const [route, setRoute] = useState<ThinScreenRoute>({ node: 'home', workspace: 'overview' });
    const [selected, setSelected] = useState<PlayerCandidate>();
    const openPlayer = (found: PlayerCandidate) => {
      setSelected(found); setRoute({ node: 'war', workspace: 'valuations-and-pool' });
    };
    return <ThinLeagueShell leagueName="Test league" route={route} commissioner players={[player]}
      navigate={setRoute} openPlayer={openPlayer}>
      {selected ? <WarRoomInspector context={{ snapshot, mine: 'franchise.01', week: 1 }}
        asset={{ token: selected.id, name: selected.name }} /> : <p>League Home</p>}
    </ThinLeagueShell>;
  }
  await act(async () => root!.render(<Probe />));
  const input = host.querySelector('#thin-player-search') as HTMLInputElement;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(input, 'RUNN');
    input.dispatchEvent(new Event('input', { bubbles: true }));
  });
  const result = host.querySelector('button[aria-label="Open Rookie Runningback player card"]');
  expect(result).not.toBeNull();
  await act(async () => { (result as HTMLButtonElement).click(); await new Promise(resolve =>
    setTimeout(resolve, 0)); });
  expect(host.querySelector('.insp .nm')?.textContent).toBe('Rookie Runningback');
  expect(host.querySelector('.node.open .nb')?.textContent).toContain('War Room');
});
