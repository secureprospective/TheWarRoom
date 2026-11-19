// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { LeagueChoice } from '../shell/leaguePicker';
import { MovesTab, readMoves, type MovesReading } from './MovesTab';

const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});
const quietPort = vi.fn(async () => { throw new Error('offline'); }) as unknown as HostedPort;
async function show(read: () => Promise<MovesReading>) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<MovesTab league={league} port={quietPort} read={read} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
  return host;
}
const tab = (label: RegExp) => act(async () =>
  ([...host!.querySelectorAll('[role="tab"]')]
    .find(t => label.test(t.textContent ?? '')) as HTMLElement).click());
const offer = { id: 't1', status: 'offered', proposer: 'other', parties: ['mine', 'other'], accepted: [],
  teams: [['mine', 'My Team'], ['other', 'Gridlock']], words: 'Waiting for you', expires: 'Fri 6:00 PM',
  details: ['You give Ace', 'You get Bolt'] };
const loaded = (): MovesReading => ({
  pool: { value: [{ id: 'p1', name: 'Ace Runner', position: 'RB' },
    { id: 'p2', name: 'Bolt Catcher', position: 'WR', waiver: 'On waivers' }] },
  waivers: { value: { place: 3, total: 10, source: 'Reverse standings', method: 'rolling',
    claims: [{ id: 'c1', player: 'p2', name: 'Bolt Catcher', words: 'Pending' }] } },
  trades: { value: { revision: 4, notes: [], offers: [offer as never] } },
});

describe('MovesTab', () => {
  it('loaded: pool filters, claim counts, claims and incoming offers', async () => {
    const view = await show(async () => loaded());
    expect(view.textContent).toContain('Ace Runner');
    expect(view.textContent).toContain('On waivers');
    expect([...view.querySelectorAll('[role="tab"]')].map(t => t.textContent)).toEqual(
      ['Players', 'Waivers1', 'Trades1']);
    await act(async () => ([...view.querySelectorAll('.mv4-chips > *')]
      .find(c => c.textContent === 'WR') as HTMLElement).click());
    expect(view.textContent).not.toContain('Ace Runner');
    await tab(/Waivers/);
    expect(view.textContent).toContain('Waiver order: 3 of 10 · Reverse standings');
    expect(view.textContent).toContain('Pending');
    expect(view.textContent).toContain('Reorder and cancel: Not available yet');
    await tab(/Trades/);
    expect(view.textContent).toContain('From Gridlock');
    expect(view.textContent).toContain('You get Bolt');
    expect(view.textContent).toContain('Accept');
    expect(view.textContent).toContain('Decline');
    expect(view.textContent).not.toContain('Withdraw');
  });
  it('empty: says the pool, claims and offers are empty', async () => {
    const view = await show(async () => ({ pool: { value: [] },
      waivers: { value: { total: 10, source: '', method: '', claims: [] } },
      trades: { value: { revision: 1, notes: [], offers: [] } } }));
    expect(view.textContent).toContain('No free agents in the pool.');
    await tab(/Waivers/);
    expect(view.textContent).toContain('No claims. Your claims stay private until waivers run.');
    await tab(/Trades/);
    expect(view.textContent).toContain('No trade offers.');
  });
  it('failed: each failed read shows its own words; a failed reading shows unavailable', async () => {
    const view = await show(async () => ({ pool: { error: 'Pool held' }, waivers: { error: 'Waivers held' },
      trades: { error: 'Trades held' } }));
    expect(view.querySelector('[role="alert"]')?.textContent).toBe('Unavailable: Pool held');
    await tab(/Waivers/);
    expect(view.textContent).toContain('Unavailable: Waivers held');
    await root!.unmount(); host!.remove();
    const whole = await show(async () => { throw new Error('Server asleep'); });
    expect(whole.textContent).toContain('Unavailable: Server asleep');
  });
  it('readMoves keeps a failed read separate from the others', async () => {
    const requests: string[] = [];
    const port = vi.fn(async (request: NativeRequest) => {
      requests.push(request.command);
      if (request.command === 'trades.read') return { data: { league: 'league', viewer: 'mine', revision: 2,
        rules: {}, offers: [], notes: [] } };
      throw new Error(`${request.command} held`);
    }) as unknown as HostedPort;
    const reading = await readMoves('league', 'mine', port);
    expect(requests.sort()).toEqual(['native.assets.read', 'trades.read', 'waivers.read']);
    expect(reading.pool.error).toBe('native.assets.read held');
    expect(reading.trades.value?.offers).toEqual([]);
  });
});
