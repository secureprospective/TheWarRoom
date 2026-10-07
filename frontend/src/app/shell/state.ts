import type { EndpointId, Surface } from '../registry';
import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { shallow } from 'zustand/shallow';
import type { Density } from '../cards/Card';
import { nodes, nodeKeys, type Node, type Route, type WorkspaceMemory } from './nodes';
export type PlayerSubject = { kind: 'player'; id: string; franchiseId: string };

export type LineupDraft = { franchiseId: string; week: number; starters: string[] };
export type ShellState = {
  lineupDraft: LineupDraft | null;
  lineupReceiptId: string | null;
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
  endpointIds: readonly EndpointId[];
  endpointSurface: Surface | null;
  scrollRevision: number;
};
export function initialShellState(): ShellState {
  return {
    lineupDraft: null,
    lineupReceiptId: null,
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
    endpointIds: [],
    endpointSurface: null,
    scrollRevision: 0,
  };
}
export function renderState(
  { density: _density, lineupDraft: _draft, lineupReceiptId: _receipt, ...rest }: ShellState,
): Omit<ShellState, 'density' | 'lineupDraft' | 'lineupReceiptId'> {
  return rest;
}

export function createShellState() {
  const store = createStore<ShellState>(() => initialShellState());
  return {
    read: store.getState,
    use: () => useStore(store, renderState, shallow),
    subscribe: store.subscribe,
    useLineupEdit: () => useStore(store, (s) => ({
      draft: s.lineupDraft, receiptId: s.lineupReceiptId,
    }), shallow),
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
