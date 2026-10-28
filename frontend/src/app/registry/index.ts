import { endpoints as generated, type EndpointId, type Endpoint, type Placement } from './endpoints.gen';
import { nodeKeys, nodes, routeFor, type Route } from '../shell/nodes';
export type { Endpoint, EndpointId, Placement } from './endpoints.gen';

export const surfacePlaces = [
  'Inspector', 'Comms', 'Calendar', 'Command bar', 'Status strip', 'Report drawer',
] as const;
export type Surface = (typeof surfacePlaces)[number];
export function isSurface(place: string): place is Surface {
  return surfacePlaces.some((surface) => surface === place);
}
export function placementRoute(place: string): Route | undefined {
  for (const node of nodeKeys) {
    const workspaces: readonly { label: string; slug: string }[] = nodes[node].workspaces;
    const workspace = workspaces.find((w) =>
      `${nodes[node].label} › ${w.label}` === place);
    if (workspace) return routeFor(node, workspace.slug);
  }
  return undefined;
}
const removedPlaces = new Set([
  'War Room › Market', 'Trade Floor › Offers history', 'Trade Floor › Counterparties',
  'League Pulse › Archive', 'Control Room › Data and sources',
]);
const removedIds = new Set(generated.filter(row =>
  row.disposition === 'kept' && removedPlaces.has(row.placement)).map(row => row.id));
export const endpoints: readonly Endpoint[] = Object.freeze(generated.map(row => {
  const removed = removedIds.has(row.id) || (row.disposition === 'merged' &&
    (removedPlaces.has(row.merged_place) || row.merged_into.some(id => removedIds.has(id))));
  return removed ? Object.freeze({ ...row, disposition: 'retired', placement: '', merged_place: '',
    merged_into: Object.freeze([] as const),
    disposition_note: 'This workspace is no longer available.' }) : row;
}));
const byId = new Map(endpoints.map((row) => [row.id, row]));
const byPlace = new Map<Placement, readonly Endpoint[]>();
for (const row of endpoints) {
  if (row.disposition !== 'kept' || !row.placement || byPlace.has(row.placement)) continue;
  byPlace.set(row.placement, Object.freeze(endpoints
    .filter((e) => e.disposition === 'kept' && e.placement === row.placement)
    .sort((a, b) => a.ring - b.ring || b.gravity.localeCompare(a.gravity) || a.id.localeCompare(b.id))));
}
export function endpointsAt(placement: Placement): readonly Endpoint[] {
  return byPlace.get(placement) ?? [];
}
export function endpointById(id: EndpointId): Endpoint {
  const row = byId.get(id);
  if (!row) throw new Error(`Unknown endpoint: ${id}`);
  return row;
}
type Resolution =
  | { kind: 'view'; placement: Placement }
  | { kind: 'merged'; into: readonly EndpointId[] }
  | { kind: 'merged'; place: Placement }
  | { kind: 'retired'; note: string };
export function resolve(id: EndpointId): Resolution {
  const row = endpointById(id);
  if (row.disposition === 'retired') return { kind: 'retired', note: row.disposition_note };
  if (row.disposition === 'merged') {
    return row.merged_place
      ? { kind: 'merged', place: row.merged_place }
      : { kind: 'merged', into: row.merged_into };
  }
  if (!row.placement) throw new Error(`${id}: kept endpoint has no placement`);
  return { kind: 'view', placement: row.placement };
}
