import type { EnvelopeState, Provenance, Receipt, TradeOffer, TradeReading } from './contract';
import { heldLineup } from './lineupTestData';

export const tradeDesk = 'https://www47.myfantasyleague.com/2026/options?L=14432&O=05';
export const dotNote = "offer left MFL's pending list; awaiting DOT and commissioner (or declined there)";

export function tradeOffer(direction: TradeOffer['direction'] = 'to_you', tradeId = '901'): TradeOffer {
  return {
    tradeId, direction, otherId: '0007', otherName: 'Rivals',
    give: [{ token: '16195', name: 'Given, Player', position: 'WR' }],
    get: [{ token: 'FP_0007_2027_1', name: '' }],
    expires: '2026-10-09T18:00:00Z', comments: 'Talked on ProBoards',
  };
}
export function tradeReading(offers = [tradeOffer()], provenance: Provenance = heldLineup().provenance) {
  return { offers, provenance } satisfies TradeReading;
}
// The Go JSON for the reading above: TargetTrades' wire shape.
export function goTradeReading(): unknown {
  return JSON.parse(JSON.stringify(tradeReading()));
}
export function tradeReceipt(state: EnvelopeState = 'ready', note = ''): Receipt {
  const blocked = state === 'blocked';
  const audit: Receipt['audit'] = [{
    at: '2026-10-07T21:00:00Z', from: 'draft', event: blocked ? 'checks_block' : 'checks_pass',
    to: blocked ? 'blocked' : 'ready', note: blocked ? note : 'offer 901 from Rivals',
  }];
  if (!blocked && state !== 'ready') {
    audit.push({
      at: '2026-10-07T21:01:00Z', from: 'ready', event: 'hand_off', to: 'handed_off', note: 'desk opened',
    });
  }
  if (!['blocked', 'ready', 'handed_off'].includes(state)) {
    const event = state === 'landed' ? 'match' : state === 'dot_review' ? 'await_dot' : 'no_change';
    audit.push({ at: '2026-10-07T21:02:00Z', from: 'handed_off', event, to: state, note });
  }
  return {
    correlationId: 'trade-901',
    spec: {
      intent: 'trade.accept', leagueId: '14432', franchiseId: '0025',
      subject: { players: ['16195'], picks: ['FP_0007_2027_1'] },
      expected: { trade: {
        tradeId: '901', offering: '0007', accepting: '0025',
        offeringGives: ['FP_0007_2027_1'], acceptingGives: ['16195'],
      } },
      gravity: 'G2', undo: 'irreversible',
      target: { kind: 'mapped', url: tradeDesk },
      deadline: '2026-10-09T18:00:00Z',
    },
    state,
    audit,
  };
}
