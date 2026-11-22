// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { LeagueChoice } from '../shell/leaguePicker';
import type { Snapshot } from '../data/contract';
import type { WarRoomContext } from '../shell/warroomReads';
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
async function show(read: () => Promise<MovesReading>, port = quietPort) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const readContext = async (): Promise<WarRoomContext> => ({ mine: 'mine', week: 11,
    snapshot: { players: { value: [] }, rosters: { value: [] }, league: { value: { season: 2026 } },
      franchises: { value: [] } } as unknown as Snapshot,
    native: { league: 'league', mine: 'mine', port, owner: true, leave: () => undefined } });
  await act(async () => root!.render(<MovesTab league={league} port={port} read={read}
    readContext={readContext} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
  return host;
}
const tab = (label: RegExp) => act(async () =>
  ([...host!.querySelectorAll('[role="tab"]')]
    .find(t => label.test(t.textContent ?? '')) as HTMLElement).click());
const offer = { id: 't1', status: 'offered', proposer: 'other', parties: ['mine', 'other'], accepted: [],
  teams: [['mine', 'My Team'], ['other', 'Gridlock']], words: 'Waiting for you', expires: 'Fri 6:00 PM',
  details: ['Ace: My Team → Gridlock', 'Bolt: Gridlock → My Team'],
  legs: [{ from: 'mine', to: 'other', label: 'Ace' }, { from: 'other', to: 'mine', label: 'Bolt' }] };
const loaded = (): MovesReading => ({
  pool: { value: [{ id: 'p1', name: 'Ace Runner', position: 'RB' },
    { id: 'p2', name: 'Bolt Catcher', position: 'WR', waiver: 'On waivers', window: 'w2' }] },
  roster: { revision: 7, writable: true, football: 'pin', mine: [{ id: '0109', name: 'Old Guard' }] },
  waivers: { value: { revision: 5, place: 3, total: 10, source: 'Reverse standings', method: 'rolling',
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
    expect(view.textContent)
      .toContain('Claims run top first. You are 3rd of 10 in waiver order · Reverse standings.');
    expect(view.textContent).toContain('+ Bolt Catcher');
    expect(view.textContent).toContain('Pending');
    expect(view.querySelector('[aria-label="Cancel claim"]')).not.toBeNull();
    await tab(/Trades/);
    expect(view.textContent).toContain('Offer from Gridlock');
    const sides = [...view.querySelectorAll('.mv4-sides > div')].map(d => d.textContent);
    expect(sides).toEqual(['YOU GIVEAce', 'YOU GETBolt']);
    expect(view.querySelector('.mv4-to.need')).not.toBeNull();
    expect(view.textContent).toContain('Accept');
    expect(view.textContent).toContain('Decline');
    expect(view.textContent).not.toContain('Withdraw');
  });
  it('empty: says the pool, claims and offers are empty', async () => {
    const view = await show(async () => ({ pool: { value: [] },
      waivers: { value: { revision: 5, total: 10, source: '', method: '', claims: [] } },
      trades: { value: { revision: 1, notes: [], offers: [] } } }));
    expect(view.textContent).toContain('No free agents in the pool.');
    await tab(/Waivers/);
    expect(view.textContent).toContain('No waiver claims. Players with a CLAIM key in Players');
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
  it('pool rows join the NFL games read for team · kickoff and show the season AVG column', async () => {
    const board = { league: 'league', board: { stale: false, source: 'ESPN',
      at: '2026-11-19T18:00:00Z', failure: '', games: [{ id: 'g1', season: 2026, week: 11, seasonType: 2,
        kickoff: '2026-11-22T18:00:00Z', home: { id: '1', abbreviation: 'LAR', score: 0 },
        away: { id: '2', abbreviation: 'SF', score: 0 }, status: 'pre', clock: '', period: 0,
        state: null, last: null, box: [], plays: [] }] } };
    const port = vi.fn(async (request: NativeRequest) => {
      if (request.command !== 'nfl.games.read') throw new Error('held');
      return { data: board, writable: false, football: '', label: 'Server' };
    }) as unknown as HostedPort;
    const view = await show(async () => ({ ...loaded(), pool: { value: [{ id: 'p1', name: 'Ace Runner',
      position: 'RB', team: 'LAR', average: '12.4' },
      { id: 'p3', name: 'No Stats', position: 'TE' }] } }), port);
    const rows = [...view.querySelectorAll('.mv4-fa')];
    expect(rows[0].querySelector('em.tm')?.textContent).toMatch(/^LAR · \w{3} /);
    expect(rows[0].querySelector('.mv4-avg')?.textContent).toBe('12.4AVG');
    expect(rows[1].querySelector('em.tm')).toBeNull();
    expect(rows[1].querySelector('.mv4-avg')?.textContent).toBe('—AVG');
  });
  it('readMoves carries team and average from the assets pool', async () => {
    const port = vi.fn(async (request: NativeRequest) => {
      if (request.command !== 'native.assets.read') throw new Error('held');
      return { writable: true, football: 'pin', data: { league: 'league', revision: 1, seeded: true,
        entry: 'e', source: 'held', identities: [],
        pool: [{ player: 'p1', name: 'Ace', position: 'RB', string: '—', points: '40.0', team: 'LAR',
          average: '10.0' }],
        snapshot: { season: 2026, franchises: [], players: [], picks: [], pool: [] } } };
    }) as unknown as HostedPort;
    const reading = await readMoves('league', 'mine', port);
    expect(reading.pool.value?.[0]).toMatchObject({ team: 'LAR', average: '10.0' });
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

  it('player card opens from a pool row and closes', async () => {
    const view = await show(async () => loaded());
    await press('Open Ace Runner');
    expect(view.querySelector('dialog[aria-label="Ace Runner"]')).not.toBeNull();
    await press('Close');
    expect(view.querySelector('dialog')).toBeNull();
  });
  it('add sheet: reviews roster.add, holds to confirm, closes and rereads', async () => {
    const port = service();
    const read = vi.fn(async () => loaded());
    const view = await show(read, port);
    await press('Add Ace Runner');
    expect(view.querySelector('dialog[aria-label="Add Ace Runner"]')).not.toBeNull();
    const select = view.querySelector('dialog select') as HTMLSelectElement;
    await act(async () => { setValue(select, '0109'); });
    const reason = view.querySelector('dialog input') as HTMLInputElement;
    await act(async () => { setValue(reason, 'Need a back'); });
    await pressText('Review add');
    await hold('Confirm add');
    const sent = port.mock.calls.map(([r]) => r).filter(r => r.command === 'roster.add');
    expect(sent.map(r => r.mode)).toEqual(['preview', 'execute']);
    expect(sent[0].arguments).toMatchObject({
      args: { waiver: { player: 'p1', franchise: 'mine', drop: '0109' } },
      reason: 'Need a back', expected: 7 });
    expect(view.querySelector('dialog')).toBeNull();
    expect(view.textContent).toContain('Ace Runner added');
    expect(read).toHaveBeenCalledTimes(2);
  });
  it('claim sheet: reviews private waivers.claim, confirms, closes; blind bid not available', async () => {
    const port = service();
    const blind = loaded();
    blind.waivers.value!.method = 'Blind bidding';
    const view = await show(async () => blind, port);
    await press('Claim Bolt Catcher');
    expect(view.textContent).toContain('Bid amount: Not available yet');
    await pressText('Review claim');
    await hold('Confirm league change');
    const sent = port.mock.calls.map(([r]) => r).filter(r => r.command === 'waivers.claim');
    expect(sent.map(r => r.mode)).toEqual(['preview', 'execute']);
    expect(sent[0].arguments).toMatchObject({ args: { waiver: { window: 'w2', franchise: 'mine' } },
      expected: 5, visibility: 'private' });
    expect(view.querySelector('dialog')).toBeNull();
    expect(view.textContent).toContain('Claim placed for Bolt Catcher');
  });
  it('trade sheet: partner, both sides, review trade.offer, hold to send, closes', async () => {
    const port = service();
    const view = await show(async () => loaded(), port);
    await tab(/Trades/);
    await pressText('Start a trade');
    await pressText('Gridlock');
    const send = () => [...view.querySelectorAll('dialog [aria-disabled], dialog button')]
      .find(b => b.textContent === 'Review offer') as HTMLButtonElement;
    expect(send().disabled || send().getAttribute('aria-disabled') === 'true').toBe(true);
    await pressText('Their Back');
    await pressText('Old Guard');
    expect(view.textContent).toContain('1 selected');
    await pressText('Review offer');
    await hold('Send offer');
    const sent = port.mock.calls.map(([r]) => r).filter(r => r.command === 'trade.offer');
    expect(sent.map(r => r.mode)).toEqual(['preview', 'execute']);
    expect(view.querySelector('dialog')).toBeNull();
    expect(view.textContent).toContain('Offer sent to Gridlock');
  });
  it('claims: reorder sends waivers.claim.order.set, cancel sends waivers.claim.cancel', async () => {
    const port = service();
    const two = loaded();
    two.waivers.value!.claims.push({ id: 'c2', player: 'p1', name: 'Ace Runner', words: 'Pending' });
    const view = await show(async () => two, port);
    await tab(/Waivers/);
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    await press('Move down');
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    const order = port.mock.calls.map(([r]) => r).filter(r => r.command === 'waivers.claim.order.set');
    expect(order[0]?.mode).toBe('preview');
    expect(order[0]?.arguments).toMatchObject({ args: { waiver: { franchise: 'mine', order: ['c2', 'c1'] } },
      expected: 5 });
  });
  it('claims: cancel previews waivers.claim.cancel for that claim', async () => {
    const port = service();
    const view = await show(async () => loaded(), port);
    await tab(/Waivers/);
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    await press('Cancel claim');
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    const sent = port.mock.calls.map(([r]) => r).filter(r => r.command === 'waivers.claim.cancel');
    expect(sent[0]?.arguments).toMatchObject({ args: { waiver: { franchise: 'mine', claim: 'c1' } } });
    expect(view.textContent).toContain('Pending');
  });
  it('trades: a held trade shows its public hold reason; bench capacity shows open slots', async () => {
    const held = loaded();
    held.openBench = 2;
    held.trades.value!.offers = [{ ...offer,
      reasons: ['Review held for 24 hours: checking both owners'] } as never];
    const view = await show(async () => held);
    expect(view.textContent).toContain('2 open bench spots');
    await tab(/Trades/);
    expect(view.textContent).toContain('Review held for 24 hours: checking both owners');
  });
  it('trades: my own open offer says its status with a Withdraw key; bench full names the drop', async () => {
    const mineOut = loaded();
    mineOut.openBench = 0;
    mineOut.trades.value!.offers = [{ ...offer, proposer: 'mine', words: 'Waiting on Gridlock' } as never];
    const view = await show(async () => mineOut);
    expect(view.textContent).toContain('Bench full: adding means dropping someone');
    await tab(/Trades/);
    expect(view.textContent).toContain('Your offer to Gridlock');
    expect(view.querySelector('.mv4-res.wait')?.textContent).toBe('Waiting on GridlockWithdraw');
    expect(view.textContent).not.toContain('Accept');
    expect([...view.querySelectorAll('.mv4-sides > div')].map(d => d.textContent))
      .toEqual(['YOU GIVEAce', 'YOU GETBolt']);
  });
  it('counter opens the trade sheet loaded with the offer', async () => {
    const port = service();
    const withAssets = loaded();
    withAssets.trades.value!.offers = [{ ...offer, assets: [{ from: 'other', to: 'mine', token: '0108' },
      { from: 'mine', to: 'other', token: '0109' }] } as never];
    const view = await show(async () => withAssets, port);
    await tab(/Trades/);
    await pressText('Counter');
    expect(view.querySelector('dialog[aria-label="Trade with Gridlock"]')).not.toBeNull();
    expect(view.textContent).toContain('✓ Their Back');
  });
});

const reply = (data: unknown) => ({ label: '', writable: true, football: 'pin', data });
const previewOf = (version: number) => ({ question: 'Review',
  target: { league: 'league', name: 'L', version },
  fields: [], affects: { readings: [], count: 0, words: '' }, blockers: [], safeguard: 'hold' });
const receipt = { entry: 'entry-1', league: 'league', revision: 8, recordedAt: '2026-11-17T12:01:00Z',
  message: 'Recorded' };
const sentOffer = { id: 'entry-1', proposer: 'mine', parties: ['mine', 'other'],
  assets: [{ from: 'mine', to: 'other', player: '0109' }, { from: 'other', to: 'mine', player: '0108' }],
  assetDetails: [{ fromTeam: 'My Team', toTeam: 'Gridlock', label: 'Old Guard' },
    { fromTeam: 'Gridlock', toTeam: 'My Team', label: 'Their Back' }], accepted: [], policy: {},
  offeredAt: '2026-11-17T12:00:00Z', expiresAt: '2026-12-01T12:00:00Z', status: 'offered',
  words: 'Waiting', expires: 'Dec 1', votes: [], recusals: [], appeals: [], overrides: [],
  vetoNumber: 0, appealedVetoNumber: 0 };
function service() {
  let offered = false;
  return vi.fn(async (request: NativeRequest) => {
    const { command, mode } = request;
    if (command === 'commish.rules.read') return reply({ league: 'league', name: 'L', revision: 7,
      rollover: '', sections: [] });
    if (command === 'native.assets.read') return reply({ league: 'league', revision: 7, seeded: true,
      source: 'held', pool: [], identities: [], snapshot: { season: 2026, franchises: [
        { id: 'mine', heldId: 'mine', name: 'My Team', abbreviation: 'MY', division: '' },
        { id: 'other', heldId: 'other', name: 'Gridlock', abbreviation: 'GR', division: '' }],
      players: [{ id: '0109', name: 'Old Guard', franchise: 'mine', status: 'ROSTER', contractStatus: '',
        salaryCents: 100,
        expirationYear: 2027 }, { id: '0108', name: 'Their Back', franchise: 'other', status: 'ROSTER',
        contractStatus: '',
        salaryCents: 100, expirationYear: 2027 }], picks: [], pool: [] } });
    if (command === 'trades.read') return reply({ league: 'league', viewer: 'mine', revision: 8, rules: [],
      offers: offered ? [sentOffer] : [], notes: [] });
    if (command === 'trade.offer' && mode === 'execute') offered = true;
    if (['roster.add', 'waivers.claim', 'trade.offer'].includes(command))
      return reply(mode === 'preview' ? previewOf(command === 'waivers.claim' ? 5 : 7)
        : receipt);
    throw new Error(`${command} not served`);
  }) as unknown as HostedPort & ReturnType<typeof vi.fn>;
}
const press = (label: string) => act(async () =>
  (host!.querySelector(`[aria-label="${label}"]`) as HTMLElement).click());
const pressText = (text: string) => act(async () => {
  const found = [...host!.querySelectorAll('button, [role="tab"]')]
    .find(b => b.textContent?.includes(text)) as HTMLElement;
  found.click();
  await new Promise(resolve => setTimeout(resolve, 20));
});
async function hold(label: string) {
  const button = host!.querySelector(`button[aria-label="${label}"]`)!;
  await act(async () => button.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 700)); });
}
function setValue(field: HTMLInputElement | HTMLSelectElement, value: string) {
  const proto = Object.getPrototypeOf(field) as object;
  Object.getOwnPropertyDescriptor(proto, 'value')!.set!.call(field, value);
  field.dispatchEvent(new Event(field instanceof HTMLSelectElement ? 'change' : 'input', { bubbles: true }));
}
