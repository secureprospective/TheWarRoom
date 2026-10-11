import type { EnvelopeState } from '../data/contract';
import type { Signal } from '../look/channels';

// Bid plans end in two states the move rail's own receipts never reach.
export type RailState = EnvelopeState | 'awarded_off_app' | 'withdrawn';

export const stateLabels: Record<RailState, string> = {
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
  awarded_off_app: 'Awarded off-app',
  withdrawn: 'Withdrawn',
};
export const railStates = Object.keys(stateLabels) as RailState[];
export const intentLabels: Record<string, string> = {
  'roster.ir': 'IR placement', 'roster.taxi': 'Taxi move',
  'lineup.set': 'Lineup change', 'trade.accept': 'Trade accept',
};
export const stateSignals: Record<RailState, Signal> = {
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
  awarded_off_app: 'grey',
  withdrawn: 'grey',
};
