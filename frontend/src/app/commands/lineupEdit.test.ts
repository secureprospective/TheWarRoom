import { describe, expect, it, vi } from 'vitest';
import { FixtureProvider } from '../data/provider';
import { heldLineup, heldSnapshot } from '../data/lineupTestData';
import { lineupReceipt } from '../data/lineupReceiptTestData';
import { createCommands } from './registry';
import { createMovesState } from './moves';
import { loadSessionMoves } from './sessionMoves';
import type { Provider } from '../data/provider';

async function setup(snapshot = heldSnapshot()) {
  const provider = new FixtureProvider();
  provider.lineup = vi.fn().mockResolvedValue(heldLineup());
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: '0025' });
  await c.lineups.refresh('0025');
  await c.registry['lineup.edit'].run({ franchiseId: '0025' });
  await waitFor(() => expect(c.read().lineupDraft).not.toBeNull());
  return { c, provider };
}
async function flush() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}
async function waitFor(assertion: () => void) {
  for (let attempt = 0; attempt < 50; attempt++) {
    try {
      assertion();
      return;
    } catch {
      await flush();
    }
  }
  assertion();
}
describe('registered lineup commands', () => {
  it('seeds, toggles both ways, refuses reserves, resets and discards', async () => {
    const { c } = await setup();
    const baseline = heldLineup().starters.map((p) => p.id);
    expect(c.read().lineupDraft).toEqual({ franchiseId: '0025', week: 5, starters: baseline });
    c.dispatch('lineup.toggle', { playerId: '16195' });
    await flush();
    expect(c.read().lineupDraft?.starters).not.toContain('16195');
    c.dispatch('lineup.toggle', { playerId: '16195' });
    await flush();
    expect(c.read().lineupDraft?.starters).toContain('16195');
    for (const player of heldSnapshot().rosters.value[0].players.filter((p) => p.rosterStatus !== 'ROSTER')) {
      c.dispatch('lineup.toggle', { playerId: player.id });
    }
    await flush();
    expect(new Set(c.read().lineupDraft?.starters)).toEqual(new Set(baseline));
    c.dispatch('lineup.toggle', { playerId: '16428' });
    await flush();
    c.dispatch('lineup.reset', {});
    await flush();
    expect(c.read().lineupDraft?.starters).toEqual(baseline);
    c.dispatch('lineup.cancel', {});
    await flush();
    expect(c.read().lineupDraft).toBeNull();
  });
  it('stores the plan, hands off only Ready, and retains it after cancel', async () => {
    const { c, provider } = await setup();
    const ready = lineupReceipt();
    provider.draftLineup = vi.fn().mockResolvedValue(ready);
    c.dispatch('lineup.draft', {});
    await flush();
    expect(provider.draftLineup).not.toHaveBeenCalled();
    c.dispatch('lineup.toggle', { playerId: '16195' });
    c.dispatch('lineup.toggle', { playerId: '16428' });
    await flush();
    c.dispatch('lineup.draft', {});
    await waitFor(() => expect(c.read().lineupReceiptId).toBe(ready.correlationId));
    expect(c.readMoves().moves).toEqual([ready]);
    const handed = { ...ready, state: 'handed_off' as const, audit: [...ready.audit, {
      at: '2026-10-07T14:01:00Z', from: 'ready' as const, event: 'hand_off' as const,
      to: 'handed_off' as const, note: 'Opened MFL lineup page',
    }] };
    provider.handOff = vi.fn().mockResolvedValue(handed);
    c.dispatch('move.handoff', { correlationId: 'unknown' });
    await flush();
    expect(provider.handOff).not.toHaveBeenCalled();
    c.dispatch('move.handoff', { correlationId: ready.correlationId });
    await waitFor(() => expect(c.readMoves().moves[0].state).toBe('handed_off'));
    c.dispatch('move.handoff', { correlationId: ready.correlationId });
    await flush();
    expect(provider.handOff).toHaveBeenCalledTimes(1);
    c.dispatch('lineup.cancel', {});
    await flush();
    expect(c.read().lineupReceiptId).toBe(ready.correlationId);
  });
  it('rejects failed or weekless readings', async () => {
    const { c, provider } = await setup();
    c.dispatch('lineup.cancel', {});
    await flush();
    for (const reading of [
      { ...heldLineup(), week: 0 },
      { ...heldLineup(), provenance: { ...heldLineup().provenance,
        freshness: { state: 'fail' as const, fetchedAt: '', note: 'offline' } } },
    ]) {
      provider.lineup = vi.fn().mockResolvedValue(reading);
      await c.lineups.refresh('0025');
      c.dispatch('lineup.edit', { franchiseId: '0025' });
      await flush();
      expect(c.read().lineupDraft).toBeNull();
    }
  });
  it('benches a saved starter who left the active roster, but never starts him again', async () => {
    const snapshot = heldSnapshot();
    snapshot.rosters.value[0].players.find((p) => p.id === '16195')!.rosterStatus = 'IR';
    const { c } = await setup(snapshot);
    c.dispatch('lineup.toggle', { playerId: '16195' });
    await flush();
    expect(c.read().lineupDraft?.starters).not.toContain('16195');
    c.dispatch('lineup.toggle', { playerId: '16195' });
    await flush();
    expect(c.read().lineupDraft?.starters).not.toContain('16195');
  });
  it('a late, older load never rolls a receipt back; a newer one wins', async () => {
    const ready = lineupReceipt();
    const step = (r: typeof ready, to: 'handed_off' | 'landed') => ({ ...r, state: to, audit: [
      ...r.audit, { at: '2026-10-07T14:01:00Z', from: r.state, event: to === 'landed' ? 'match' as const
        : 'hand_off' as const, to, note: to },
    ] });
    const handed = step(ready, 'handed_off');
    const moves = createMovesState();
    moves.write({ moves: [handed] });
    const read = vi.fn().mockResolvedValue([ready]);
    const provider = { kind: 'live', moves: read } as unknown as Provider;
    await loadSessionMoves(moves, provider, '0025');
    expect(moves.read().moves).toEqual([handed]);
    read.mockResolvedValue([step(handed, 'landed')]);
    await loadSessionMoves(moves, provider, '0025');
    expect(moves.read().moves[0].state).toBe('landed');
  });
  it('fixture mutations reject with the desktop reason', async () => {
    const provider = new FixtureProvider();
    for (const operation of [
      () => provider.checkLineup('0025', []), () => provider.draftLineup('0025', []),
      () => provider.handOff('test'),
    ]) await expect(operation()).rejects.toThrow('Lineup changes need the desktop app');
  });
});
