import { describe, expect, it, vi } from 'vitest';
import assets from '../../../../testdata/wire/native/assets.json';
import cap from '../../../../testdata/wire/native/cap-check.json';
import rules from '../../../../testdata/wire/commissioner/rules.json';
import lineup from '../../../../testdata/wire/native/lineup.json';
import week from '../../../../testdata/wire/native/week.json';
import standings from '../../../../testdata/wire/native/standings.json';
import { parseRules } from './rules';
import { nativeHQAssets, nativeHQCap, nativeHQMoves, nativeHQSettings, readNativeHQ } from './hqReading';
import type { HostedPort } from './hostedRoom';

describe('native HQ existing reads', () => {
  it('maps recorded assets, settings and cap runway without an MFL fallback', () => {
    const snapshot = nativeHQAssets(assets, 'native.seed');
    expect(snapshot.rosters.value.find(r => r.franchiseId === 'franchise.02')?.players[0].salary).toBe(100_000_000);
    expect(snapshot.players.provenance.source).toBe('League ledger');
    const settings = nativeHQSettings(parseRules(rules));
    expect(settings.active).toBe(90); expect(settings.starters).toBe(21);
    const runway = nativeHQCap(cap, 'native.seed', 'franchise.01');
    expect(runway[0]).toEqual({ season: 2026, committed: 40_000_000, dead: 0, max: 100_000_000_000, room: 99_960_000_000 });
  });
  it('changes the view model when cap, total slots, IR and taxi settings change', () => {
    const changed = parseRules(structuredClone(rules));
    const rows = changed.sections.flatMap(s => s.rows);
    rows.find(r => r.field === 'lineup.groups')!.current = { total: { Min: 9, Max: 9 }, offense: { Min: 4, Max: 4 }, defense: { Min: 5, Max: 5 } };
    rows.find(r => r.field === 'ir.slots')!.current = 4;
    rows.find(r => r.field === 'taxi.slots')!.current = 6;
    expect(nativeHQSettings(changed)).toMatchObject({ starters: 9, offence: 4, defence: 5, ir: 4, taxi: 6 });
    const next = structuredClone(cap); next.franchises[0].runway[0].capCents = 999;
    expect(nativeHQCap(next, 'native.seed', 'franchise.01')[0].max).toBe(999);
  });
  it('uses native move receipts with the same states and rejects another franchise', () => {
    const value = { league: 'native.seed', franchise: 'franchise.01', planned: [], reason: 'native plans not yet stored', moves: [{ correlationId: 'act.1', spec: { intent: 'roster.cut', leagueId: 'native.seed', franchiseId: 'franchise.01', subject: { players: ['0042'], picks: [] }, expected: {}, gravity: 'G0', undo: 'irreversible', target: { kind: 'native' } }, state: 'landed', audit: [{ at: '2026-10-31T10:00:00Z', from: 'draft', event: 'recorded', to: 'landed', note: 'Recorded cut' }] }] };
    expect(nativeHQMoves(value, 'native.seed', 'franchise.01').moves[0]).toMatchObject({ native: true, state: 'landed', players: ['0042'] });
    expect(() => nativeHQMoves(value, 'native.seed', 'franchise.02')).toThrow('another franchise');
  });
  it('calls the native week, lineup, cap, moves and standings reads; isolates a failed cap read', async () => {
    const mine = 'franchise.02', league = 'native.seed';
    const data: Record<string, unknown> = {
      'native.assets.read': assets, 'commish.rules.read': { ...rules, league },
      'native.cap.check': cap, 'moves.read': { league, franchise: mine, planned: [], moves: [], reason: 'native plans not yet stored' },
      'standings.read': { ...standings, league },
      'week.read': { ...week, schedule: { ...week.schedule, league, franchise: mine } },
      'lineup.read': lineup,
    };
    const port = vi.fn<HostedPort>(async request => ({ label: 'Native', writable: false, football: '', data: data[request.command] }));
    const result = await readNativeHQ(league, mine, port);
    expect(result.errors).toEqual([]); expect(result.lineup?.starters[0].id).toBe('1000');
    expect(result.week).toBe(1); expect(result.moves).toEqual([]); expect(result.settings.active).toBe(90);
    expect(port.mock.calls.map(c => c[0].command)).toContain('moves.read');
    expect(port.mock.calls.every(c => c[0].mode === 'read')).toBe(true);
    const broken: HostedPort = async request => { if (request.command === 'native.cap.check') throw new Error('offline'); return port(request); };
    const partial = await readNativeHQ(league, mine, broken);
    expect(partial.runway).toBeUndefined(); expect(partial.lineup?.starters.length).toBeGreaterThan(0);
    expect(partial.errors).toContain('Error: offline');
  });
});
