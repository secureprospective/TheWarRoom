import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { shallow } from 'zustand/shallow';
import type { Density } from '../cards/Card';
import { nodes, nodeKeys, type Node, type Route, type WorkspaceMemory } from './nodes';
export type PlayerSubject = { kind: 'player'; id: string; franchiseId: string };

export type ShellState = {
  node: Node;
  workspace: WorkspaceMemory;
  density: Density;
  inspector: 'closed' | 'rest' | 'expanded';
  subject: PlayerSubject | null;
  franchiseId: string | null;
  commandbar: boolean;
  summoned: 'calendar' | null;
  comms: boolean;
  harness: boolean;
  notice: string | null;
  scrollRevision: number;
};
export function initialShellState(): ShellState {
  return {
    node: 'home',
    workspace: Object.fromEntries(
      nodeKeys.map((n) => [n, nodes[n].workspaces[0].slug]),
    ) as WorkspaceMemory,
    density: 'tactical',
    inspector: 'rest',
    subject: null,
    franchiseId: null,
    commandbar: false,
    summoned: null,
    comms: false,
    harness: false,
    notice: null,
    scrollRevision: 0,
  };
}
export function renderState({ density: _density, ...rest }: ShellState): Omit<ShellState, 'density'> {
  return rest;
}

export function createShellState() {
  const store = createStore<ShellState>(() => initialShellState());
  return {
    read: store.getState,
    use: () => useStore(store, renderState, shallow),
    subscribe: store.subscribe,
    // The write capability is handed only to the command registry, never to views.
    write: (patch: Partial<ShellState>) => store.setState(patch),
    navigate: (route: Route) =>
      store.setState((s) => ({
        scrollRevision: s.scrollRevision + Number(
          s.node !== route.node || s.workspace[s.node] !== route.workspace,
        ),
        node: route.node,
        workspace: { ...s.workspace, [route.node]: route.workspace },
      })),
  };
}
