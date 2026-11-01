import { describe, expect, it, vi } from 'vitest';
import assets from '../../../../testdata/wire/native/assets.json';
import reserves from '../../../../testdata/wire/native/reserve-check.json';
import cap from '../../../../testdata/wire/native/cap-check.json';
import rules from '../../../../testdata/wire/commissioner/rules.json';
import lineup from '../../../../testdata/wire/native/lineup.json';
import movesWire from '../data/native-moves.wire.testdata.json';
import week from '../../../../testdata/wire/native/week.json';
import projections from '../data/projections.wire.testdata.json';
import standings from '../../../../testdata/wire/native/standings.json';
import { parseRules } from './rules';
import { nativeHQAssets, nativeHQCap, nativeHQMoves, nativeHQSettings, readNativeHQ } from './hqReading';
import { selectProvider } from '../data/provider';
import clockWire from '../data/fixtures/clock.json';

vi.mock('../data/provider', () => ({ selectProvider: vi.fn() }));

import type { HostedPort } from './hostedRoom';

describe('native HQ existing reads', () => {
  it('maps recorded assets, settings and cap runway without an MFL fallback', () => {
    const snapshot = nativeHQAssets(assets, 'native.seed');
    expect(snapshot.rosters.value.find(r => r.franchiseId ===
      'franchise.02')?.players[0].salary).toBe(100_000_000);
    expect(snapshot.players.provenance.source).toBe('League ledger');
    const settings = nativeHQSettings(parseRules(rules));
    expect(settings.active).toBe(90); expect(settings.starters).toBe(21);
    const runway = nativeHQCap(cap, 'native.seed', 'franchise.01');
    expect(runway[0]).toEqual({ season: 2026, ir: 0, committed: 40_000_000, dead: 0, max: 100_000_000_000,
      room:
      99_960_000_000 });
  });
  it('reads an imported league whose MFL contract status was blank or held as written', () => {
    const imported = structuredClone(assets) as { snapshot: { players: Record<string, unknown>[] } };
    imported.snapshot.players[0].contractStatus = '';
    imported.snapshot.players[0].heldContractStatus = 'EXT (2024)';
    imported.snapshot.players[0].contractInfo = 'Imported salary contract';
    const snapshot = nativeHQAssets(imported, 'native.seed');
    const id = imported.snapshot.players[0].id;
    const player = snapshot.rosters.value.flatMap(r => r.players).find(p => p.id === id);
    expect(player?.contractStatus).toBe('FLAG');
  });
  it('changes the view model when cap, total slots, IR and taxi settings change', () => {
    const changed = parseRules(structuredClone(rules));
    const rows = changed.sections.flatMap(s => s.rows);
    rows.find(r => r.field === 'lineup.groups')!.current = { total: { Min: 9, Max: 9 }, offense: { Min: 4,
      Max: 4 }, defense: { Min: 5, Max: 5 } };
    rows.find(r => r.field === 'ir.slots')!.current = 4;
    rows.find(r => r.field === 'taxi.slots')!.current = 6;
    expect(nativeHQSettings(changed)).toMatchObject({ starters: 9, offence: 4, defence: 5, ir: 4, taxi: 6 });
    const next = structuredClone(cap); next.franchises[0].runway[0].capCents = 999;
    expect(nativeHQCap(next, 'native.seed', 'franchise.01')[0].max).toBe(999);
  });
  it('uses native move receipts with the same states and rejects another franchise', () => {
    const value = { league: 'native.seed', franchise: 'franchise.01', planned: [], reason:
      'native plans not yet stored', moves: [{ correlationId: 'act.1', spec: { intent: 'roster.cut',
      leagueId: 'native.seed', franchiseId: 'franchise.01', subject: { players: ['0042'], picks: [] },
      expected: {}, gravity: 'G0', undo: 'irreversible', target: { kind: 'native' } }, state: 'landed',
      audit: [{ at: '2026-10-31T10:00:00Z', from: 'draft', event: 'recorded', to: 'landed', note:
      'Recorded cut' }] }] };
    expect(nativeHQMoves(value, 'native.seed', 'franchise.01').moves[0]).toMatchObject({ native: true,
      state: 'landed', players: ['0042'] });
    expect(nativeHQMoves(movesWire, 'native.seed', 'franchise.01').moves.map(m => m.state))
      .toContain('needs_you');
    expect(() => nativeHQMoves(value, 'native.seed', 'franchise.02')).toThrow('another franchise');
  });
  it('calls the native week, lineup, cap, moves and standings reads; isolates a failed cap read', async () =>
     {
    const mine = 'franchise.02', league = 'native.seed';
    const data: Record<string, unknown> = {
      'native.assets.read': assets, 'commish.rules.read': { ...rules, league, revision: assets.revision },
      'native.reserve.check': reserves, 'native.cap.check': { ...cap, franchises: cap.franchises.map(f =>
        ({ ...f, contracts: [{ player: '1000', years: [{ season: 2026, cents: 100000000 },
          { season: 2027, cents: 100000000 }] }] })) }, 'moves.read': { league, franchise: mine,
        planned: [], moves: [], reason:
        'native plans not yet stored' },
      'standings.read': { ...standings, league, revision: assets.revision },
      'week.read': { ...week, schedule: { ...week.schedule, league, franchise: mine, revision:
        assets.revision } },
      'lineup.read': { ...lineup, revision: assets.revision },
    };
    const port = vi.fn<HostedPort>(async request => ({ label: 'Native', writable: false, football: '',
      data: request.command === 'projections.read' ? { ...projections, week: request.arguments.week } :
        data[request.command] }));
    const result = await readNativeHQ(league, mine, port);
    expect(result.errors).toEqual([]); expect(result.lineup?.starters[0].id).toBe('1000');
    expect(result.snapshot.players.value.find(p => p.id === '1008')?.position).toBe('K');
    expect(result.contracts?.get('1000')).toEqual([{ season: 2026, cents: 100000000 },
      { season: 2027, cents: 100000000 }]);
    expect(result.projections?.get('1000')).toBe(projections.players[0].median);
    expect(result.projections?.has('1008')).toBe(false);
    expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'projections.read', arguments:
      { league, season: 2026, week: 1, players: expect.stringContaining('1000') } }));
    expect(result.week).toBe(1); expect(result.moves).toEqual([]); expect(result.settings.active).toBe(90);
    expect(port.mock.calls.map(c => c[0].command)).toContain('moves.read');
    expect(port.mock.calls.every(c => c[0].mode === 'read')).toBe(true);
    const broken: HostedPort = async request => { if (request.command === 'native.cap.check') throw new
      Error('offline'); return port(request); };
    const partial = await readNativeHQ(league, mine, broken);
    expect(partial.runway).toBeUndefined(); expect(partial.lineup?.starters.length).toBeGreaterThan(0);
    expect(partial.errors).toContain('Error: offline');
    const imported = structuredClone(assets);
    const importedPlayer = imported.snapshot.players[0] as Record<string, unknown>;
    importedPlayer.contractStatus = ''; importedPlayer.heldContractStatus = 'EXT (2024)';
    importedPlayer.contractInfo = 'Imported salary contract';
    data['native.assets.read'] = imported;
    data['week.read'] = { schedule: { league, franchise: mine, season: assets.snapshot.season,
      revision: assets.revision, games: [], message: 'The schedule is not published yet.' },
      message: 'The schedule is not published yet.' };
    data['lineup.read'] = { ...lineup, starters: [], bench: [...lineup.starters, ...lineup.bench],
      revision: assets.revision };
    const clock = vi.fn().mockResolvedValue({ ...clockWire, value: { ...clockWire.value, week: 8 } });
    vi.mocked(selectProvider).mockReturnValue({ clock } as unknown as ReturnType<typeof selectProvider>);
    data['lineup.read'] = { ...data['lineup.read'] as object, week: 8 };
    const unscheduled = await readNativeHQ(league, mine, port);
    expect(unscheduled.errors).toEqual([]);
    expect(unscheduled.week).toBe(8);
    expect(unscheduled.weekNote).toBe('Viewing week 8 · The schedule is not published yet.');
    expect(unscheduled.lineup?.starters).toEqual([]);
    expect(unscheduled.lineup?.bench).toHaveLength(lineup.starters.length + lineup.bench.length);
    expect(unscheduled.snapshot.players.value.find(p => p.id === '1008')?.position).toBe('K');
    expect(unscheduled.lock).toBeUndefined();
    expect(port.mock.calls.map(([r]) => r)).toContainEqual(expect.objectContaining({
      command: 'lineup.read', arguments: expect.objectContaining({ week: 8 }) }));
    for (const invalid of [{ ...projections, league: 'native.other' },
      { ...projections, players: [{ ...projections.players[0], player: '9999' }] },
      { ...projections, players: [projections.players[1], projections.players[1]] },
      { ...projections, players: [{ ...projections.players[0], median: NaN }] }]) {
      const rejected = await readNativeHQ(league, mine, async request => request.command ===
        'projections.read' ? { label: '', writable: false, football: '',
          data: { ...invalid, week: request.arguments.week } } : port(request));
      expect(rejected.projections).toBeUndefined();
      expect(rejected.lineup).toBeDefined();
      expect(rejected.errors.length).toBeGreaterThan(0);
    }
    for (const value of [
      { ...clockWire, value: { ...clockWire.value, week: undefined } },
      { ...clockWire, value: { ...clockWire.value, season: 2025, week: 8 } },
      { ...clockWire, value: { ...clockWire.value, week: 8 }, provenance: { ...clockWire.provenance,
        freshness: { ...clockWire.provenance.freshness, state: 'fail' } } },
    ]) {
      clock.mockResolvedValue(value);
      const unknown = await readNativeHQ(league, mine, port);
      expect(unknown.week).toBeUndefined(); expect(unknown.lineup).toBeUndefined();
      expect(unknown.weekNote).toContain('Current NFL week is unknown');
    }
    const failedWeek = await readNativeHQ(league, mine, async request => {
      if (request.command === 'week.read') throw new Error('Week unavailable');
      return port(request);
    });
    expect(failedWeek.week).toBeUndefined(); expect(failedWeek.lineup).toBeUndefined();
    expect(failedWeek.errors).toContain('Error: Week unavailable');
    data['week.read'] = { schedule: { league, franchise: 'another', season: assets.snapshot.season,
      revision: assets.revision,
      games: [] } };
    const wrongScope = await readNativeHQ(league, mine, port);
    expect(wrongScope.week).toBeUndefined(); expect(wrongScope.lineup).toBeUndefined();
    expect(wrongScope.errors.join(' ')).toContain('another franchise');
    data['week.read'] = { ...week, schedule: { ...week.schedule, league, franchise: mine,
      revision: assets.revision } };
    data['lineup.read'] = { ...lineup, revision: assets.revision + 1 };
    const mixed = await readNativeHQ(league, mine, port);
    expect(mixed.lineup).toBeUndefined();
    expect(mixed.errors.join(' ')).toContain('League changed during the read');
  });
});

it('reads native cut costs and before/after cap without changing the contract', async () => {
  const { default: quote } = await import('../../../../testdata/wire/native/contract-quote.json');
  const { parseNativeCut, readNativeCut } = await import('./hqReading');
  const result = parseNativeCut(quote, 'native.seed', '0099');
  expect(result).toMatchObject({ deadNow: 1_050_000_000, deadNext: 0, before: 1_850_000_000,
    after: 1_900_000_000, roomAfter: 10_600_000_000 });
  const port = vi.fn<HostedPort>(async () => ({ label: 'Native', writable: false, football: '', data: quote
    }));
  await readNativeCut('native.seed', 'franchise.02', '0099', port);
  expect(port.mock.calls[0][0]).toMatchObject({ command: 'contract.quote', mode: 'read', arguments: {
    league: 'native.seed', franchise: 'franchise.02', player: '0099', action: 'cut', years: 0,
  } });
  expect(() => parseNativeCut(quote, 'native.seed', '0042')).toThrow('another player');
});

it('resolves human division standings from alignment settings, never ledger ids', async () => {
  const { nativeHQStanding } = await import('./hqReading');
  const changed = parseRules(structuredClone(rules));
  const alignment = changed.sections.flatMap(s => s.rows).find(r => r.field === 'league.alignment')!;
  const rows = [{ franchise: 'franchise.01', division: 'division.1' },
    { franchise: 'franchise.02', division: 'division.1' }];
  expect(nativeHQStanding(changed, rows, 'franchise.02')).toBeUndefined();
  alignment.current = { 'franchise.01': 'NFC West', 'franchise.02': 'NFC West' };
  expect(nativeHQStanding(changed, rows, 'franchise.02')).toBe('2nd NFC West');
  alignment.current = { 'franchise.01': 'NFC West', 'franchise.02': 'NFC East' };
  expect(nativeHQStanding(changed, rows, 'franchise.02')).toBe('1st NFC East');
});

it('keeps a known zero cut price behind a closed gate, but never invents an unpriced zero', async () => {
  const { default: quote } = await import('../../../../testdata/wire/native/contract-quote.json');
  const { parseNativeCut } = await import('./hqReading');
  const shut = { ...quote, allowed: false, reason: 'Cannot date this action', deadMoney: [] };
  expect(parseNativeCut(shut, 'native.seed', '0099')).toMatchObject({ deadNow: 0, deadNext: 0,
    reason: 'Cannot date this action' });
  expect(parseNativeCut({ ...shut, cap: null, needs: ['Release rate'] }, 'native.seed', '0099'))
    .toMatchObject({ deadNow: undefined, deadNext: undefined });
});
