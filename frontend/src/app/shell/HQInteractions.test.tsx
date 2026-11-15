import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, click, flush, mount, render, rerender, type, unmount }
  from '../commissioner/commissionerDom';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { HQInspector } from './HQInspector';
import { HQBoard } from './HQBoard';
import { useHQRead } from './hqReads';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
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
it('never calls local private-list reads for a native franchise', async () => {
  const notes = vi.fn();
  const snapshot = heldSnapshot();
  await mount(render(HQInspector, { native: true, snapshot, mine: 'franchise.01',
    id: snapshot.players.value[0].id }), {}, { offers: { API: { TargetPersonalNotes: notes } } });
  await flush(); expect(notes).not.toHaveBeenCalled();
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
it('selects a real native cut quote without borrowing the previous selection or executing a move',
  async () => {
  const { HQNativeWhatIf } = await import('./HQNativeWhatIf');
  const snapshot = heldSnapshot(), id = snapshot.rosters.value[0].players[0].id, select = vi.fn();
  const props = { snapshot, mine: '0025', chosen: id, select, beforeRoom: 800000000,
    cut: { player: id, revision: 1, season: 2026, deadNow: 100000000, roomAfter: 900000000, reason: '' },
    close: () => {} };
  const host = await mount(render(HQNativeWhatIf, props));
  expect(host.textContent).toContain('$9.00M');
  const other = snapshot.rosters.value[0].players[1].id;
  const name = snapshot.players.value.find(p => p.id === other)!.name!;
  await click(all(host, 'button').find(b => b.textContent.includes(name))!);
  expect(select).toHaveBeenCalledWith(other);
  await rerender(render(HQNativeWhatIf, { ...props, chosen: other }));
  expect(host.textContent).not.toContain('$9.00M');
  expect(host.textContent).toContain('Not available yet');
});
