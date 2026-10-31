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
