import type { EnvelopeState } from '../data/contract';
import type { Signal } from '../look/channels';

export const stateLabels: Record<EnvelopeState, string> = {
  draft: 'Draft',
  ready: 'Ready',
  handed_off: 'Handed off',
  not_yet_done: 'Not yet done',
  landed: 'Landed',
  blocked: 'Blocked',
  failed: 'Failed',
  stale: 'Stale',
  not_verified: 'Not verified',
  dot_review: 'DOT review',
  bid_pending: 'Bid pending',
  waiver_pending: 'Waiver pending',
};
export const intentLabels: Record<string, string> = {
  'roster.ir': 'IR placement', 'lineup.set': 'Lineup change', 'trade.accept': 'Trade accept',
};
export const stateSignals: Record<EnvelopeState, Signal> = {
  draft: 'grey',
  ready: 'blue',
  handed_off: 'blue',
  not_yet_done: 'amber',
  landed: 'green',
  blocked: 'red',
  failed: 'red',
  stale: 'amber',
  not_verified: 'amber',
  dot_review: 'amber',
  bid_pending: 'amber',
  waiver_pending: 'amber',
};
