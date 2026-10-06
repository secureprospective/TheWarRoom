import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import type { Density } from '../cards/Card';
import { nodes, nodeKeys, type Node, type Route, type WorkspaceMemory } from './nodes';
export type ShellState = {
  node: Node; workspace: WorkspaceMemory; density: Density;
  inspector: 'closed' | 'rest' | 'expanded'; subject: string | null;
  summoned: 'calendar' | null; comms: boolean; harness: boolean; notice: string | null;
};
export function initialShellState(): ShellState {
  return {
    node: 'home', workspace: Object.fromEntries(nodeKeys.map(n => [n, nodes[n].workspaces[0].slug])) as WorkspaceMemory,
    density: 'tactical', inspector: 'rest', subject: null, summoned: null,
    comms: false, harness: false, notice: null,
  };
}
export function createShellState() {
  const store = createStore<ShellState>(() => initialShellState());
  return {
    read: store.getState,
    use: () => useStore(store),
    subscribe: store.subscribe,
    // The write capability is handed only to the command registry, never to views.
    write: (patch: Partial<ShellState>) => store.setState(patch),
    navigate: (route: Route) => store.setState(s => ({ node: route.node, workspace: { ...s.workspace, [route.node]: route.workspace } })),
  };
}
