import type { Snapshot } from '../data/contract';
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
) {
  let snapshot: Snapshot | undefined;
  const state = createShellState();
  const ambient = {
    roles: ['gm', 'commish', 'admin'] as const,
    gravity: 'G0' as const,
    undo: 'instant' as const,
    args: [] as const,
  };
  const registry = Object.freeze({
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
    if (fromBar && id !== 'commandbar.open') {
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
    loadSnapshot: (value: Snapshot) => {
      snapshot = value;
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
