export const nodes = {
  home: { label: 'Home', workspaces: [
    { slug: 'seasonal-card', label: 'Seasonal card', ring: 'Ring 1' },
    { slug: 'alert-tray', label: 'Alert tray', ring: 'Ring 1' },
    { slug: 'digests', label: 'Digests', ring: 'Ring 3' },
  ] },
  war: { label: 'War Room', workspaces: [
    { slug: 'market', label: 'Market', ring: 'Ring 2' },
    { slug: 'valuations-and-pool', label: 'Valuations and pool', ring: 'Ring 2' },
    { slug: 'research', label: 'Research', ring: 'Ring 2' },
  ] },
  hq: { label: 'Franchise HQ', workspaces: [
    { slug: 'lineup-and-roster', label: 'Lineup and roster', ring: 'Ring 1' },
    { slug: 'contracts-and-cap', label: 'Contracts and cap', ring: 'Ring 2' },
    { slug: 'my-moves', label: 'My moves', ring: 'Ring 1' },
  ] },
  trade: { label: 'Trade Floor', workspaces: [
    { slug: 'trade-desk-and-offers', label: 'Trade desk and offers', ring: 'Ring 1' },
    { slug: 'counterparties', label: 'Counterparties', ring: 'Ring 2' },
    { slug: 'draft-room', label: 'Draft room', ring: 'Ring 2' },
  ] },
  pulse: { label: 'League Pulse', workspaces: [
    { slug: 'now', label: 'Now', ring: 'Ring 1' },
    { slug: 'race', label: 'Race', ring: 'Ring 3' },
    { slug: 'pools-archive', label: 'Pools / Archive', ring: 'Ring 3' },
  ] },
  control: { label: 'Control Room', workspaces: [
    { slug: 'app', label: 'App', ring: 'Ring 3' },
    { slug: 'data-and-sources', label: 'Data and sources', ring: 'Ring 3' },
    { slug: 'league', label: 'League (role-gated)', ring: 'Ring 3' },
  ] },
} as const;
export type Node = keyof typeof nodes;
export type Route = { [N in Node]: { node: N; workspace: typeof nodes[N]['workspaces'][number]['slug'] } }[Node];
export type WorkspaceMemory = { [N in Node]: typeof nodes[N]['workspaces'][number]['slug'] };
export const nodeKeys = Object.keys(nodes) as Node[];
export const homeRoute: Route = { node: 'home', workspace: 'seasonal-card' };
export function isNode(value: string): value is Node { return Object.prototype.hasOwnProperty.call(nodes, value); }
export function routeFor(node: Node, workspace: string): Route | undefined {
  return nodes[node].workspaces.some(w => w.slug === workspace) ? { node, workspace } as Route : undefined;
}
