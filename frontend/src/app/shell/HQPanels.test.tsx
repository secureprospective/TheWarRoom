import { describe, expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { AgeArc, NotesThread, UsageRows } from './HQInspector';
import { CapRunway, HQReserves } from './HQPanels';
import { HQMoves, HQQueue, moveLane, type HQMove } from './HQMoves';
import { heldSnapshot } from '../data/lineupTestData';
import { parseArc, parseInjuries, parseNotes, parseUsage } from './hqPlayerData';
import { unavailable } from './hqModel';

describe('HQ inspector reads', () => {
  it('renders usage shares, the actual matchup, and missing values without invented zeroes', () => {
    const usage = parseUsage({ snapShare: .8, targetShare: .27, carryShare: null, weeksUsed: 3,
      opponent: 'DEN', allowedPoints: 14.5, allowedRank: 24, teams: 32, label: 'Soft', reason:
        '3 weeks held' });
    const html = renderToStaticMarkup(<UsageRows usage={usage} projection={13.75} position="TE" />);
    expect(html).toContain('27%'); expect(html).toContain('Soft · 24'); expect(html).toContain('13.8');
    expect(html).toContain('3 weeks used');
    expect(renderToStaticMarkup(<UsageRows position="RB" usage={usage} />)).toContain(unavailable);
    expect(() => parseUsage({ ...usage, snapShare: 2 })).toThrow();
  });
  it('renders actual model arc seasons, highlight and peak window, never the sample arc', () => {
    const arc = parseArc({ age: 26.3, points: [{ age: 26.3, season: 2027, level: 50, survival: .9 },
      { age: 27.3, season: 2028, level: 55, survival: .8 }], peakStart: 2027, peakEnd: 2028 });
    const html = renderToStaticMarkup(<AgeArc arc={arc} season={2027} />);
    expect(html).toContain('2027–2028'); expect(html).toContain('class="now"');
      expect(html).toContain('age 26.3');
    expect(renderToStaticMarkup(<AgeArc season={2027} />)).toContain(unavailable);
  });
  it('renders private dated notes and checks their selected-player scope and count', () => {
    const thread = parseNotes({ player: '0042', count: 1, notes: [{ kind: 'note', entity: 'player', id:
      '0042', text: 'Keep through deadline', at: '2026-10-31T10:00:00Z', order: 1 }] });
    const html = renderToStaticMarkup(<NotesThread {...thread} add={async () => {}} remove={async () => {}}
      />);
    expect(html).toContain('Keep through deadline'); expect(html).toContain('REMOVE');
      expect(html).toContain('ADD NOTE');
    expect(renderToStaticMarkup(<NotesThread />)).toContain(unavailable);
    expect(() => parseNotes({ player: '0042', notes: [], count: 2 })).toThrow('count mismatch');
  });
});
describe('HQ reserves and runway', () => {
  it('renders BACK from returnSeason/returnWeek, retaining the reason for an unknown return', () => {
    const snapshot = heldSnapshot(); const ir = snapshot.rosters.value[0].players.find(p => p.rosterStatus
      === 'IR')!;
    const injuries = parseInjuries([{ id: ir.id, status: 'IR', details: 'Ankle', returnSeason: 2026,
      returnWeek: 11, returnReason: 'Week from held report' }]);
    const html = renderToStaticMarkup(<HQReserves kind="ir" snapshot={snapshot} mine="0025" settings={{
      positions: [], ir: 12, active: 44 }} injuries={injuries} close={() => {}} />);
    expect(html).toContain('2026 · Wk 11'); expect(html).toContain('Ankle');
      expect(html).toContain('Week from held report');
    expect(html).toContain('4 / 12'); expect(html).toContain('38 / 44');
    expect(renderToStaticMarkup(<HQReserves kind="ir" snapshot={snapshot} mine="0025" settings={{
      positions: [] }} close={() => {}} />)).toContain(unavailable);
  });
  it('uses changed taxi capacity and salary reads instead of the mock totals', () => {
    const snapshot = heldSnapshot(); const view = (taxi: number) => renderToStaticMarkup(<HQReserves
      kind="taxi" snapshot={snapshot} mine="0025" settings={{ positions: [], taxi }} close={() => {}} />);
    expect(view(8)).toContain('6 / 8'); expect(view(10)).toContain('6 / 10');
  });
  it('renders each runway year, cap split and room from the read; absence stays unknown', () => {
    const html = renderToStaticMarkup(<CapRunway years={[{ season: 2028, committed: 6_000_000_000, dead:
      100_000_000, max: 9_000_000_000, room: 2_900_000_000 }]} />);
    expect(html).toContain('2028'); expect(html).toContain('$29.00M'); expect(html).toContain('Dead $1.00M');
    expect(renderToStaticMarkup(<CapRunway />)).toContain(unavailable);
  });
});
describe('HQ move pipeline', () => {
  it('maps ready/waiting/landed/stale into the four mock lanes and queue counts for either origin', () => {
    const snapshot = heldSnapshot();
    const moves: HQMove[] = ['ready', 'handed_off', 'landed', 'stale'].map((state, i) => ({ id: String(i),
      state: state as HQMove['state'], intent: 'roster.ir', note: `Real note ${i}`, at:
      '2026-10-31T10:00:00Z', players: [], native: true }));
    expect(moves.map(m => moveLane(m.state))).toEqual([0, 1, 2, 3]);
    const html = renderToStaticMarkup(<HQMoves snapshot={snapshot} moves={moves} />);
    for (const m of moves) expect(html).toContain(m.note);
    const queue = renderToStaticMarkup(<HQQueue moves={moves} available />);
    expect(queue).toContain('1 NEED YOU'); expect(queue).toContain('1 WAITING');
      expect(queue).toContain('1 LANDED');
    expect(renderToStaticMarkup(<HQMoves snapshot={snapshot} moves={[]} available={false}
      />)).toContain(unavailable);
  });
});
