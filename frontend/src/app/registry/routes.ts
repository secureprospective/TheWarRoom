import { isSurface, placementRoute, type Endpoint } from './index';
import type { IndexedResult } from '../commands/search';
import { rankResults } from '../commands/search';
export type EndpointRoute = 'nav' | 'command' | 'merged-into';

export function endpointRoutes(rows: readonly Endpoint[], candidates: readonly IndexedResult[]) {
  const routes = new Map<string, Set<EndpointRoute>>();
  for (const row of rows) {
    const set = new Set<EndpointRoute>();
    if (row.disposition === 'kept') {
      if (placementRoute(row.placement) || isSurface(row.placement)) set.add('nav');
    }
    if (rankResults(row.name, candidates).some((r) =>
      r.invocation.verb === 'endpoint.open' && r.invocation.args.id === row.id)) set.add('command');
    routes.set(row.id, set);
  }
  for (const row of rows.filter((r) => r.disposition === 'merged')) {
    const reachable = row.merged_place
      ? Boolean(placementRoute(row.merged_place) || isSurface(row.merged_place))
      : row.merged_into.length > 0 && row.merged_into.every((id) => (routes.get(id)?.size ?? 0) >= 2);
    if (reachable) routes.get(row.id)!.add('merged-into');
  }
  return routes;
}
export function assertEndpointRoutes(rows: readonly Endpoint[], candidates: readonly IndexedResult[]) {
  const routes = endpointRoutes(rows, candidates);
  for (const row of rows) {
    const count = routes.get(row.id)!.size;
    if (row.disposition === 'kept' && count < 2) throw new Error(`${row.id}: fewer than two routes`);
    if (row.disposition === 'retired' && count) throw new Error(`${row.id}: retired row reachable`);
    if (row.disposition === 'merged') {
      if (Boolean(row.merged_place) === Boolean(row.merged_into.length)) {
        throw new Error(`${row.id}: merge target not kept and reachable`);
      }
      if (!count || row.merged_into.some((id) => rows.find((r) => r.id === id)?.disposition !== 'kept')) {
        throw new Error(`${row.id}: merge target not kept and reachable`);
      }
    }
  }
  return routes;
}
