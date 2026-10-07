import { createMovesState } from './moves';
import { createMFLKeyState } from './mflKey';
import { createMFLKeyField } from '../shell/mflKeyField';
import { createMovesPrefetch } from '../shell/MovesMount';
import { selectProvider, type Provider } from '../data/provider';
import {
  resolve, placementRoute, isSurface, type EndpointId, type Placement, type Surface,
} from '../registry';
import type { MFLKeyStatus, Snapshot } from '../data/contract';
import type { PlayerSubject } from '../shell/state';
import {
  browserStorage,
  loadFranchise,
  persistFranchise,
  validFranchise,
  type SettingsStorage,
} from '../shell/franchise';
import { presets, presetAllowed, type PresetId } from '../shell/presets';
import type { Density } from '../cards/Card';
import type { Gravity } from '../look/channels';
import { createShellState } from '../shell/state';
import { routeFor, type Node, type Route } from '../shell/nodes';
export type Role = 'gm' | 'commish' | 'admin';
export type Undo = 'instant' | 'reversible' | 'irreversible';
export type Command<A> = {
  id: string;
  label: string;
  aliases: readonly string[];
  roles: readonly Role[];
  gravity: Gravity;
  undo: Undo;
  args: readonly (keyof A)[];
  keys?: readonly { key: string; args: A }[];
  run: (args: A) => void;
};
function command<A>(definition: Command<A>): Readonly<Command<A>> {
  return Object.freeze({
    ...definition,
    aliases: Object.freeze([...definition.aliases]),
    roles: Object.freeze([...definition.roles]),
    args: Object.freeze([...definition.args]),
    keys:
      definition.keys &&
      Object.freeze(
        definition.keys.map((k) => Object.freeze({ ...k, args: Object.freeze(k.args) })),
      ),
  });
}
export function createCommands(
  storage: () => SettingsStorage | undefined = browserStorage,
  provider: Provider = selectProvider(),
  mflField = createMFLKeyField(),
) {
  let snapshot: Snapshot | undefined;
  const state = createShellState();
  const moves = createMovesState();
  const mflKey = createMFLKeyState();
  const prefetchMoves = createMovesPrefetch();
  const ambient = {
    roles: ['gm', 'commish', 'admin'] as const,
    gravity: 'G0' as const,
    undo: 'instant' as const,
    args: [] as const,
  };
  function openPlace(place: Placement) {
    const route = placementRoute(place);
    if (route) state.navigate(route);
    else if (isSurface(place)) {
      if (place === 'Inspector') state.write({ inspector: 'rest' });
      if (place === 'Comms') state.write({ comms: true });
      if (place === 'Calendar') state.write({ summoned: 'calendar' });
      if (place === 'Command bar') state.write({ commandbar: true });
      state.write({ endpointSurface: place });
    } else throw new Error(`endpoint.open: unreachable place ${place}`);
  }
  const subjectKey = (subject: PlayerSubject) => `${subject.franchiseId}:${subject.id}`;
  function draftReason(subject: PlayerSubject): string | undefined {
    if (provider.kind === 'fixture') return provider.reason;
    if (subject.franchiseId !== state.read().franchiseId) return 'Only players on my franchise';
    if (moves.read().drafting[subjectKey(subject)]) return 'Drafting…';
    return undefined;
  }
  async function draftIR(subject: PlayerSubject) {
    if (draftReason(subject) || provider.kind !== 'live') return;
    const key = subjectKey(subject);
    moves.write({
      drafting: { ...moves.read().drafting, [key]: true },
      draftErrors: { ...moves.read().draftErrors, [key]: '' },
    });
    try {
      const { finishDraft } = await import('./sessionMoves');
      await finishDraft(moves, provider, subject, key);
    } catch (cause) {
      moves.write({
        drafting: { ...moves.read().drafting, [key]: false },
        draftErrors: { ...moves.read().draftErrors, [key]: String(cause) },
      });
    }
  }
  async function loadMoves(franchiseId: string) {
    try {
      const { loadSessionMoves } = await import('./sessionMoves');
      await loadSessionMoves(moves, provider, franchiseId);
    } catch (cause) {
      moves.write({
        movesErrors: { ...moves.read().movesErrors, [franchiseId]: String(cause) },
        movesLoading: { ...moves.read().movesLoading, [franchiseId]: false },
      });
    }
  }
  function mflReason(forget = false): string | undefined {
    if (provider.kind === 'fixture') return 'Connecting MFL needs the desktop app';
    if (mflKey.read().pending) return 'Checking MFL…';
    if (forget) return mflKey.read().state === 'connected' ? undefined : 'Not connected';
    return mflField.empty() ? 'Paste your API key in Control Room › App' : undefined;
  }
  async function requestMFL(operation: () => Promise<MFLKeyStatus>, activity: string) {
    if (mflKey.read().pending) return;
    const before = mflKey.read();
    mflKey.write({ ...before, pending: true, activity, error: undefined });
    try {
      const status = await operation();
      if (before.state === 'connected' && ['rejected', 'unreachable'].includes(status.state)) {
        status.detail = `${status.detail ?? ''} · Previously stored key is unchanged`;
      }
      mflKey.write({ ...status, pending: false });
    } catch (cause) {
      mflKey.write({
        ...before, pending: false,
        error: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      mflField.take();
    }
  }
  function connectMFL() {
    if (provider.kind !== 'live' || mflKey.read().pending) return;
    if (mflField.empty()) {
      state.navigate({ node: 'control', workspace: 'app' });
      setTimeout(mflField.focus, 0);
      return;
    }
    void requestMFL(() => provider.connectMFL(mflField.take()), 'Checking the key with MFL…');
  }
  const registry = Object.freeze({
    'mflkey.connect': command({
      ...ambient, id: 'mflkey.connect', label: 'Connect MFL', aliases: ['mfl', 'api key'],
      gravity: 'G1', undo: 'reversible',
      run: (_args: Record<string, never>) => connectMFL(),
    }),
    'mflkey.forget': command({
      ...ambient, id: 'mflkey.forget', label: 'Forget MFL key', aliases: [],
      gravity: 'G1', undo: 'reversible',
      run: (_args: Record<string, never>) => {
        if (!mflReason(true)) {
          void requestMFL(() => provider.forgetMFL(), 'Removing the key from the keyring…');
        }
      },
    }),
    'roster.ir': command({
      id: 'roster.ir',
      label: 'Draft IR placement',
      aliases: [],
      roles: ['gm'],
      gravity: 'G2',
      undo: 'reversible',
      args: ['subject'],
      run: (args: { subject: PlayerSubject }) => { void draftIR(args.subject); },
    }),
    'surface.open': command({
      ...ambient,
      id: 'surface.open',
      label: 'Open shell surface',
      aliases: ['surface'],
      args: ['place'],
      run: (args: { place: Surface }) => openPlace(args.place),
    }),
    'surface.close': command({
      ...ambient,
      id: 'surface.close',
      label: 'Close endpoint surface',
      aliases: [],
      run: (_args: Record<string, never>) => state.write({ endpointSurface: null }),
    }),
    'endpoint.open': command({
      ...ambient,
      id: 'endpoint.open',
      label: 'Open endpoint',
      aliases: ['endpoint'],
      args: ['id'],
      run: (args: { id: EndpointId }) => {
        const target = resolve(args.id);
        if (target.kind === 'retired') throw new Error(`endpoint.open: retired ${args.id}`);
        const ids = target.kind === 'merged' && 'into' in target ? target.into : [args.id];
        state.write({ endpointIds: ids, endpointSurface: null });
        if (target.kind === 'view') openPlace(target.placement);
        else if ('place' in target) openPlace(target.place);
        else for (const id of target.into) {
          const view = resolve(id);
          if (view.kind !== 'view') throw new Error(`endpoint.open: target ${id} is not kept`);
          openPlace(view.placement);
        }
      },
    }),
    'franchise.set': command({
      ...ambient,
      id: 'franchise.set',
      label: 'Choose my franchise',
      aliases: ['my team'],
      args: ['franchiseId'],
      run: (args: { franchiseId: string }) => {
        if (!snapshot || !validFranchise(snapshot, args.franchiseId)) {
          throw new Error('franchise.set: unknown franchise');
        }
        persistFranchise(args.franchiseId, storage());
        state.write({
          franchiseId: args.franchiseId,
          scrollRevision: state.read().scrollRevision + Number(
            state.read().franchiseId !== args.franchiseId,
          ),
        });
      },
    }),
    'preset.apply': command({
      ...ambient,
      id: 'preset.apply',
      label: 'Apply preset',
      aliases: ['view'],
      args: ['preset'],
      run: (args: { preset: PresetId }) => {
        if (!presetAllowed(args.preset))
          throw new Error('preset.apply: GM permission denied');
        const p = presets[args.preset];
        state.navigate(p.route);
        state.write({ density: p.density, inspector: p.inspector, comms: p.comms });
      },
    }),
    'inspector.open': command({
      ...ambient,
      id: 'inspector.open',
      label: 'Inspect player',
      aliases: ['player'],
      args: ['subject'],
      run: (args: { subject: PlayerSubject }) => {
        const subject = args.subject;
        if (
          !snapshot?.rosters.value.some(
            (r) =>
              r.franchiseId === subject.franchiseId &&
              r.players.some((p) => p.id === subject.id),
          )
        ) {
          throw new Error('inspector.open: unknown roster player');
        }
        state.write({ subject: { ...subject }, inspector: 'rest' });
      },
    }),
    'inspector.collapse': command({
      ...ambient,
      id: 'inspector.collapse',
      label: 'Collapse inspector',
      aliases: ['collapse'],
      run: (_args: Record<string, never>) => state.write({ inspector: 'rest' }),
    }),
    'commandbar.close': command({
      ...ambient,
      id: 'commandbar.close',
      label: 'Close command bar',
      aliases: [],
      run: (_args: Record<string, never>) => state.write({ commandbar: false }),
    }),
    'nav.open': command({
      ...ambient,
      id: 'nav.open',
      label: 'Open workspace',
      args: ['node', 'workspace'],
      aliases: ['goto', 'navigate'],
      run: (args: Route | { node: Node; workspace?: undefined }) => {
        const route = routeFor(
          args.node,
          args.workspace ?? state.read().workspace[args.node],
        );
        if (!route) throw new Error(`nav.open: invalid workspace for ${args.node}`);
        state.navigate(route);
      },
    }),
    'density.set': command({
      ...ambient,
      id: 'density.set',
      label: 'Set density',
      args: ['density'],
      aliases: ['density'],
      keys: [
        { key: 'N', args: { density: 'narrative' as Density } },
        { key: 'T', args: { density: 'tactical' as Density } },
        { key: 'M', args: { density: 'matrix' as Density } },
        { key: '1', args: { density: 'narrative' as Density } },
        { key: '2', args: { density: 'tactical' as Density } },
        { key: '3', args: { density: 'matrix' as Density } },
      ],
      run: (args: { density: Density }) => state.write({ density: args.density }),
    }),
    'inspector.expand': command({
      ...ambient,
      id: 'inspector.expand',
      label: 'Expand inspector',
      aliases: ['expand'],
      run: (_args: Record<string, never>) => state.write({ inspector: 'expanded' }),
    }),
    'inspector.toggle': command({
      ...ambient,
      id: 'inspector.toggle',
      label: 'Toggle inspector',
      aliases: ['inspect'],
      keys: [{ key: 'I', args: {} }],
      run: (_args: Record<string, never>) =>
        state.write({
          inspector: state.read().inspector === 'closed' ? 'rest' : 'closed',
        }),
    }),
    'inspector.close': command({
      ...ambient,
      id: 'inspector.close',
      label: 'Close inspector',
      aliases: ['close inspector'],
      run: (_args: Record<string, never>) => state.write({ inspector: 'closed' }),
    }),
    'calendar.summon': command({
      ...ambient,
      id: 'calendar.summon',
      label: 'Summon calendar',
      aliases: ['calendar'],
      run: (_args: Record<string, never>) =>
        state.write({ summoned: state.read().summoned ? null : 'calendar' }),
    }),
    'comms.toggle': command({
      ...ambient,
      id: 'comms.toggle',
      label: 'Toggle comms',
      aliases: ['comms', 'chat'],
      run: (_args: Record<string, never>) => state.write({ comms: !state.read().comms }),
    }),
    'overlays.close': command({
      ...ambient,
      id: 'overlays.close',
      label: 'Close overlays',
      aliases: ['escape'],
      keys: [{ key: 'Escape', args: {} }],
      run: (_args: Record<string, never>) =>
        state.write({
          inspector: 'closed',
          summoned: null,
          comms: false,
          notice: null,
          endpointSurface: null,
          commandbar: false,
        }),
    }),
    'commandbar.open': command({
      ...ambient,
      id: 'commandbar.open',
      label: 'Type a command',
      aliases: ['command', '/'],
      keys: [
        { key: '/', args: {} },
        { key: 'Ctrl+K', args: {} },
      ],
      run: (_args: Record<string, never>) => state.write({ commandbar: true }),
    }),
    'harness.open': command({
      ...ambient,
      id: 'harness.open',
      label: 'Open harness',
      aliases: ['harness'],
      run: (_args: Record<string, never>) => state.write({ harness: true }),
    }),
    'target.open': command({
      ...ambient,
      id: 'target.open',
      label: 'Target UI',
      aliases: ['target'],
      run: (_args: Record<string, never>) => state.write({ harness: false }),
    }),
  });
  type Id = keyof typeof registry;
  type Args<K extends Id> = Parameters<(typeof registry)[K]['run']>[0];
  type Entry = { [K in Id]: { id: K; args: Args<K> } }[Id];
  const history: Readonly<Entry>[] = [];
  function dispatch<K extends Id>(id: K, args: Args<K>) {
    const selected = registry[id] as Command<Args<K>>;
    if (!selected.roles.includes('gm')) throw new Error(`${id}: GM permission denied`);
    const fromBar = state.read().commandbar;
    selected.run(args);
    if (fromBar && id !== 'commandbar.open' &&
      !(id === 'endpoint.open' && state.read().endpointSurface === 'Command bar')) {
      if (
        id === 'inspector.open' &&
        state.read().subject?.franchiseId === state.read().franchiseId
      ) {
        dispatch('nav.open', { node: 'hq', workspace: 'lineup-and-roster' });
      }
      state.write({ commandbar: false });
    }
    history.push(Object.freeze({ id, args: Object.freeze({ ...args }) }) as Entry);
    if (history.length > 32) history.shift();
  }
  function key(key: string): boolean {
    for (const id of Object.keys(registry) as Id[]) {
      const selected: Command<Args<Id>> = registry[id] as Command<Args<Id>>;
      const binding = selected.keys?.find(
        (k) => k.key.toLowerCase() === key.toLowerCase(),
      );
      if (binding) {
        dispatch(id, binding.args);
        return true;
      }
    }
    return false;
  }
  return {
    registry,
    mflField,
    mflReason,
    loadMFLKey: () => requestMFL(() => provider.mflKey(), 'Checking the keyring…'),
    readMFLKey: mflKey.read,
    useMFLKey: mflKey.use,
    providerKind: provider.kind,
    draftReason,
    loadMoves,
    readMoves: moves.read,
    useMoves: moves.use,
    loadSnapshot: (value: Snapshot) => {
      snapshot = value;
      prefetchMoves();
      state.write({ franchiseId: loadFranchise(value, storage()) });
    },
    dispatch,
    key,
    history: () => Object.freeze([...history]),
    read: state.read,
    use: state.use,
    subscribe: state.subscribe,
  };
}
export const commands = createCommands();
export type CommandId = keyof typeof commands.registry;
export type CommandArgs<K extends CommandId> = Parameters<
  (typeof commands.registry)[K]['run']
>[0];
export const dispatch = commands.dispatch;
