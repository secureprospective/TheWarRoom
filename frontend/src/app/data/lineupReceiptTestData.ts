import type { Receipt, EnvelopeState } from './contract';
import { heldLineup } from './lineupTestData';

export function lineupReceipt(state: EnvelopeState = 'ready'): Receipt {
  const saved = heldLineup().starters.map((p) => p.id);
  const starters = saved.filter((id) => id !== '16195').concat('16428');
  return {
    correlationId: 'lineup-test',
    spec: {
      intent: 'lineup.set', leagueId: '14432', franchiseId: '0025',
      subject: { players: starters, picks: [] },
      expected: { lineup: { week: 5, starters, baseline: saved } },
      gravity: 'G1', undo: 'reversible',
      target: { kind: 'mapped', url: 'https://fixture.invalid/lineup?WEEK=5' },
      deadline: '2026-10-13T00:15:00Z',
    },
    state,
    audit: [{
      at: '2026-10-07T14:00:00Z', from: 'draft',
      event: state === 'blocked' ? 'checks_block' : 'checks_pass', to: state,
      note: state === 'blocked' ? 'week 5 lineups are locked' : 'legal',
    }],
  };
}
