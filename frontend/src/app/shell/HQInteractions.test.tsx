import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, click, flush, mount, render, rerender, type, unmount }
  from '../commissioner/commissionerDom';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { HQInspector } from './HQInspector';
import { HQBoard } from './HQBoard';
import { useHQRead } from './hqReads';
import { hostedReads } from '../data/hostedReads';
import type { HostedResponse } from '../control/nativeApi';

afterEach(async () => { await unmount(); hostedReads.endSession(); vi.restoreAllMocks(); });
const usage = { snapShare: .7, targetShare: .25, carryShare: null, weeksUsed: 4, opponent: 'DEN',
  allowedPoints: 15, allowedRank: 24, teams: 32, label: 'Soft', reason: 'Held weekly facts' };
const arc = { age: 25, points: [{ season: 2026, age: 25, level: 80, survival: .9 }],
  peakStart: 2026, peakEnd: 2026 };
it('adds and removes dated notes through the existing backend, then re-reads the thread', async () => {
  const snapshot = heldSnapshot(), id = snapshot.players.value[0].id;
  let notes: { kind: string; entity: string; id: string; text: string; at: string; order: number }[] = [];
  const read = vi.fn(async () => ({ player: id, notes, count: notes.length }));
  const change = vi.fn(async (raw: { action: string; at: string; text: string }) => {
    if (raw.action === 'add') notes = [{ kind: 'note', entity: 'player', id, text: raw.text,
      at: '2026-10-31T10:00:00Z', order: 1 }];
    else notes = notes.filter(n => n.at !== raw.at);
  });
  const host = await mount(render(HQInspector, { snapshot, mine: '0025', id }), {}, {
    offers: { API: { TargetPersonalNotes: read, TargetPersonalChange: change } },
    market: { API: { TargetAgeArc: async () => arc, TargetAssetBoard: async () => ({ values: [],
      nowSource: '', dynastySource: '', how: '' }) } },
    research: { API: { TargetPlayerUsage: async () => usage } },
    contracts: { API: { TargetPlayerContract: async () => { throw new Error('Not held'); } } },
  });
  await flush();
  await type(all(host, 'textarea')[0], 'Keep through deadline');
  await act(async () => { const submit = all(host, 'form')[0].props().onSubmit;
    (submit as (e: { preventDefault(): void }) => void)({ preventDefault() {} }); });
  await flush();
  expect(host.textContent).toContain('Keep through deadline');
  expect(change).toHaveBeenCalledWith(expect.objectContaining({ mine: '0025', id, action: 'add',
    kind: 'note', text: 'Keep through deadline' }));
  await click(button(host, 'REMOVE')); await flush();
  expect(host.textContent).not.toContain('Keep through deadline');
  expect(change).toHaveBeenLastCalledWith(expect.objectContaining({ action: 'remove',
    at: '2026-10-31T10:00:00Z' }));
  expect(read).toHaveBeenCalledTimes(3);
});
it('shows unavailable rather than an internal notes read error', async () => {
  const snapshot = heldSnapshot();
  const host = await mount(render(HQInspector, { snapshot, mine: '0025', id: snapshot.players.value[0].id }),
    {}, { offers: { API: { TargetPersonalNotes: async () => {
      throw new Error('No recorded read: offers.TargetPersonalNotes');
    } } } });
  await flush();
  expect(host.textContent).toContain('MY NOTES');
  expect(host.textContent).toContain('Not available yet');
  expect(host.textContent).not.toContain('offers.TargetPersonalNotes');
  expect(host.textContent).not.toContain('No recorded read');
});
it('does not expose a failed what-if read in the release panel', async () => {
  const whatIf = await import('../contracts/whatif');
  vi.spyOn(whatIf, 'whatIfProvider').mockReturnValue({
    query: async () => { throw new Error('No recorded read: contracts.TargetWhatIf'); },
    change: async () => {}, onChange: () => () => {},
  });
  const { HQRelease } = await import('./HQRelease');
  const snapshot = heldSnapshot();
  const host = await mount(render(HQRelease, { snapshot, mine: '0025', id: snapshot.players.value[0].id,
    close: () => {} }));
  await flush();
  expect(host.textContent).toContain('Not available yet');
  expect(host.textContent).not.toContain('contracts.TargetWhatIf');
  expect(host.textContent).not.toContain('No recorded read');
});
it('picks a bench player, places him through a command-backed row, and filters positions', async () => {
  const snapshot = heldSnapshot(), reading = heldLineup(), place = vi.fn(), select = vi.fn();
  const host = await mount(render(HQBoard, { snapshot, mine: '0025', reading,
    settings: { positions: [] }, values: new Map(), canEdit: true, place, select }));
  const name = (id: string) => snapshot.players.value.find(p => p.id === id)!.name!;
  const incoming = reading.bench[0], outgoing = reading.starters[0];
  await click(button(host, `Bench · ${name(incoming.id)}`));
  await click(button(host, `${outgoing.position} · ${name(outgoing.id)}`));
  expect(place).toHaveBeenCalledWith(incoming.id, outgoing.id);
  expect(select).toHaveBeenCalledWith(outgoing.id);
  await click(button(host, 'QB'));
  const other = reading.bench.find(p => p.position !== 'QB')!;
  expect(host.textContent).not.toContain(name(other.id));
});
it('fences slow reads and never paints another selected player’s data', async () => {
  let resolve!: (value: string) => void;
  const first = new Promise<string>(yes => { resolve = yes; });
  const read = vi.fn((id: string) => id === '0042' ? first : Promise.resolve('Second player'));
  function Probe({ id }: { id: string }) {
    const state = useHQRead('market.TargetAgeArc', [id], read);
    return <p>{state.value ?? 'Not available yet'}</p>;
  }
  const host = await mount(render(Probe, { id: '0042' }));
  expect(host.textContent).toBe('Not available yet');
  await rerender(render(Probe, { id: '0043' })); await flush();
  expect(host.textContent).toBe('Second player');
  await act(async () => { resolve('First player'); });
  expect(host.textContent).toBe('Second player');
});
it('makes no desktop data-binding calls for a hosted franchise', async () => {
  const notes = vi.fn(), age = vi.fn(), usage = vi.fn(), assets = vi.fn(), contracts = vi.fn();
  const snapshot = heldSnapshot();
  await mount(render(HQInspector, { native: true, snapshot, mine: 'franchise.01',
    id: snapshot.players.value[0].id }), {}, {
    offers: { API: { TargetPersonalNotes: notes } },
    market: { API: { TargetAgeArc: age, TargetAssetBoard: assets } },
    research: { API: { TargetPlayerUsage: usage } },
    contracts: { API: { TargetPlayerContract: contracts } },
  });
  await flush();
  for (const binding of [notes, age, usage, assets, contracts]) expect(binding).not.toHaveBeenCalled();
});
it('promotes a native reserve through its hosted review callback', async () => {
  const { HQReserves } = await import('./HQPanels');
  const snapshot = heldSnapshot();
  const reserve = snapshot.rosters.value[0].players.find(p => p.rosterStatus === 'IR')!;
  const activate = vi.fn();
  const host = await mount(render(HQReserves, { native: true, kind: 'ir', snapshot, mine: '0025',
    settings: { positions: [], ir: 12 }, close: () => {}, activate,
    reserveNotes: new Map([[reserve.id, 'League reserve verdict']]) }));
  expect(host.textContent).toContain('League reserve verdict');
  const actions = all(host, 'button').filter(b => b.textContent === 'ACTIVATE');
  await click(actions[0]);
  expect(activate).toHaveBeenCalledWith(reserve.id);
});
it('adds, removes and resets stacked moves, rereading each ordered stack', async () => {
  const { HQNativeWhatIf } = await import('./HQNativeWhatIf');
  const snapshot = heldSnapshot(), first = snapshot.rosters.value[0].players[0].id;
  const second = snapshot.rosters.value[0].players[1].id;
  hostedReads.setIdentity({ serverId: 'https://league.test', serverEpoch: 'epoch', accountId: 'owner',
    authGeneration: '1', league: 'native.test', view: 'owner/0025' });
  const tags = { serverEpoch: 'epoch', league: {}, football: 'pin', live: '', values: '' };
  const read = vi.fn(async (request: import('../commands/native').NativeRequest): Promise<HostedResponse> => {
    const args = request.arguments as { league: string; moves: { player: string; action: string }[] };
    const quote = (move: { player: string; action: string }) => ({
      league: args.league, revision: 1, season: 2026, player: move.player,
      name: `Player ${move.player}`, franchise: 'franchise.01', action: move.action,
      actionName: 'Release', allowed: true, priceable: true, reason: '', needs: [],
      years: 0, salaryCents: null,
      salaries: [], deadMoney: [{ season: 2026, cents: 100000000 }], reliefCents: 0,
      cap: null, basis: [], notes: [], source: 'server',
    });
    const room = args.moves.length ? 800000000 : 900000000;
    const year = { season: 2026, committedCents: 100000000, deadCents: args.moves.length ? 100000000 : 0,
      capCents: 1000000000, roomCents: room };
    return { label: 'League', writable: false, football: 'pin', tags, data: { league: args.league,
      revision: 1, season: 2026, before: [{ ...year, deadCents: 0, roomCents: 900000000 }], after: [year],
      moves: args.moves.map(quote) } };
  });
  const props = { snapshot, league: 'native.test', mine: '0025', chosen: first, select: vi.fn(), port: read,
    close: () => {} };
  const host = await mount(render(HQNativeWhatIf, props)); await flush();
  expect(host.textContent).toContain('$9.00M');
  const firstName = snapshot.players.value.find(p => p.id === first)!.name!;
  await click(button(host, `Select ${firstName}`));
  await click(button(host, 'ADD MOVE')); await flush();
  expect(host.textContent).toContain('$8.00M');
  await click(button(host, `Remove ${firstName} cut`)); await flush();
  expect(host.textContent).toContain('$9.00M');
  const secondName = snapshot.players.value.find(p => p.id === second)!.name!;
  await click(button(host, `Select ${secondName}`));
  await click(button(host, 'ADD MOVE')); await flush();
  expect(host.textContent).toContain('$8.00M');
  await click(button(host, 'RESET MOVES')); await flush();
  const stacks = read.mock.calls.map(([request]) =>
    (request.arguments as { moves: { player: string }[] }).moves.map(move => move.player));
  expect(stacks).toEqual([[], [first], [], [second], []]);
  expect(host.textContent).toContain('$9.00M');
});
