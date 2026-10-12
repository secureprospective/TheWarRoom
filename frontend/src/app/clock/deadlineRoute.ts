import type { Route } from '../shell/nodes';

type DeadlineRoute = { verb: 'nav.open'; args: Route } |
  { verb: 'calendar.summon'; args: Record<string, never> };
export function deadlineRoute(id: string): DeadlineRoute {
  if (id.startsWith('lineup-w')) {
    return { verb: 'nav.open', args: { node: 'hq', workspace: 'lineup-and-roster' } };
  }
  if (id === 'window:trade_deadline') {
    return { verb: 'nav.open', args: { node: 'trade', workspace: 'trade-desk' } };
  }
  if (['window:playoff_bids_only', 'window:multiyear_bids_close', 'window:ufa_bidding'].includes(id)) {
    return { verb: 'nav.open', args: { node: 'war', workspace: 'valuations-and-pool' } };
  }
  return { verb: 'calendar.summon', args: {} };
}
