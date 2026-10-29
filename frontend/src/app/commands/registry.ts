import type { BrowserArguments, BrowserArgumentMap } from './arguments.gen';
import { browserCommands } from './browser.gen';
import { loadMarket } from '../market/load';
import { prefetchHQ } from '../shell/FranchiseHQMount';
import { createLineups, createReadings } from '../shell/lineups';
import { createMovesState } from './moves';
import { createMFLKeyState } from './mflKey';
import { createMFLKeyField } from '../shell/mflKeyField';
import { createMovesPrefetch } from '../shell/MovesMount';
import { selectProvider, type Provider } from '../data/provider';
import {
  resolve,
  placementRoute,
  isSurface,
  type Placement,
} from '../registry';
import type { MFLKeyStatus, Snapshot, TradeReading } from '../data/contract';
import type { PlayerSubject } from '../shell/state';
import {
  browserStorage,
  loadFranchise,
  persistFranchise,
  validFranchise,
  type SettingsStorage,
} from '../shell/franchise';
import { presets, presetAllowed, holdsRole, type Role } from '../shell/presets';
import type { Gravity } from '../look/channels';
import { createShellState } from '../shell/state';
import { routeFor } from '../shell/nodes';
export type { Role } from '../shell/presets';
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
  run: (args: A) => void | Promise<unknown>;
};
function command<K extends keyof BrowserArgumentMap>(definition: {
  id: K; run: (args: BrowserArguments<K>) => void | Promise<unknown>;
}): Readonly<Command<BrowserArguments<K>>> {
  const metadata = browserCommands[definition.id as keyof typeof browserCommands];
  const keys = 'keys' in metadata
    ? metadata.keys as unknown as Command<BrowserArguments<K>>['keys'] : undefined;
  return Object.freeze({
    ...metadata,
    aliases: Object.freeze([...metadata.aliases]),
    roles: Object.freeze([...metadata.roles]),
    args: Object.freeze([...metadata.args]) as readonly (keyof BrowserArguments<K>)[],
    keys: keys && Object.freeze(keys.map(k => Object.freeze({
      ...k, args: Object.freeze(k.args) as BrowserArguments<K>,
    }))),
    run: definition.run,
  });
}
export function createCommands(
  storage: () => SettingsStorage | undefined = browserStorage,
  provider: Provider = selectProvider(),
  mflField = createMFLKeyField(),
  // The local desktop operator holds all roles; restricted callers supply their own.
  roles: readonly Role[] = ['gm', 'commish', 'admin'],
) {
  let snapshot: Snapshot | undefined;
  const state = createShellState();
  const lineups = createLineups(provider);
  const trades = createReadings<TradeReading>(provider, (id) => provider.trades(id), 'trades.TargetTrades');
  const alerts = createReadings(provider, (id) => provider.alerts(id), 'control.TargetAlerts');
  const moves = createMovesState();
  const mflKey = createMFLKeyState();
  const prefetchMoves = createMovesPrefetch();
  const hqFailure = (cause: unknown) => console.error('HQ prefetch failed:', cause);
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
  async function draftRoster(subject: PlayerSubject, intent: 'roster.ir' | 'roster.taxi') {
    if (draftReason(subject) || provider.kind !== 'live') return;
    const key = subjectKey(subject);
    moves.write({
      drafting: { ...moves.read().drafting, [key]: true },
      draftErrors: { ...moves.read().draftErrors, [key]: '' },
    });
    try {
      const { finishDraft } = await import('./sessionMoves');
      await finishDraft(moves, provider, subject, key, intent);
    } catch (cause) {
      moves.write({
        drafting: { ...moves.read().drafting, [key]: false },
        draftErrors: { ...moves.read().draftErrors, [key]: String(cause) },
      });
      throw cause;
    }
  }
  function rosterCommand(intent: 'roster.ir' | 'roster.taxi') {
    return command({
      id: intent,
      run: ({ subject }: BrowserArguments<'roster.ir'>) => {
        return draftRoster(subject, intent);
      },
    });
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
  async function completed(work: Promise<unknown>, failure: () => string | undefined) {
    await work;
    const reason = failure();
    if (reason) throw new Error(reason);
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
        ...before,
        pending: false,
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
    return completed(requestMFL(() => provider.connectMFL(mflField.take()), 'Checking the key with MFL…'),
      () => mflKey.read().error || (['rejected', 'unreachable'].includes(mflKey.read().state)
        ? mflKey.read().detail || 'MFL did not accept the connection. Check your key and try again.'
        : undefined));
  }
  function lineupAction(action: import('./lineupEdit').LineupAction) {
    return import('./lineupEdit')
      .then(({ runLineupAction }) =>
        runLineupAction(action, state, moves, lineups, provider, snapshot),
      );
  }
  function editCommand<K extends
    'lineup.edit' | 'lineup.toggle' | 'lineup.reset' | 'lineup.cancel' |
    'lineup.draft' | 'move.withdraw' | 'move.handoff'>(
    id: K, kind: import('./lineupEdit').LineupAction['kind'],
  ) {
    return command({
      id,
      run: (values: BrowserArguments<K>) =>
        lineupAction({ kind, ...values } as import('./lineupEdit').LineupAction),
    });
  }
  const dataRun = (action: import('../datasources/state').Action) => {
    return import('../datasources/state').then(async m => {
      await m.dataState.run(action);
      if (m.dataState.read().error) throw new Error(m.dataState.read().error);
    });
  };
  const phaseRoutes = {
    OFFSEASON: { node: 'hq', workspace: 'contracts-and-cap' },
    DRAFT: { node: 'trade', workspace: 'draft-room' },
    REGULAR_SEASON: { node: 'hq', workspace: 'lineup-and-roster' },
    PLAYOFFS: { node: 'pulse', workspace: 'race' },
  } as const;
  function personalCommand(kind: import('../personal/contract').ListKind) {
    return command({
      id: kind === 'watch' ? 'watch.add' : kind === 'bait' ? 'bait.set'
        : kind === 'note' ? 'note.add' : 'draft.list',
      run: (change: BrowserArguments<'watch.add'>) => {
        const mine = state.read().franchiseId;
        if (!mine) {
          const message = 'Choose my franchise before keeping a private list';
          state.write({ notice: message });
          throw new Error(message);
        }
        return import('../personal/state').then(m => completed(m.personalState.change(mine, kind, change),
          () => m.personalState.peek(mine).error));
      },
    });
  }
  const registry = Object.freeze({
    'watch.add': personalCommand('watch'),
    'bait.set': personalCommand('bait'),
    'note.add': personalCommand('note'),
    'draft.list': personalCommand('draft'),
    'personal.handoff': command({
      id: 'personal.handoff',
      run: ({ kind }: BrowserArguments<'personal.handoff'>) => {
        return import('../personal/state').then(m => m.personalState.handoff(kind));
      },
    }),
    'window.set': command({
      id: 'window.set',
      run: ({ kind, at }: BrowserArguments<'window.set'>) => {
        return import('../clock/windowState').then(m => m.setWindow(provider, kind, at));
      },
    }),
    'window.edit': command({
      id: 'window.edit',
      run: ({ kind }: BrowserArguments<'window.edit'>) => {
        return import('../clock/windowState').then(m => m.editWindow(kind));
      },
    }),
    'phase.goto': command({
      id: 'phase.goto',
      run: ({ phase }: BrowserArguments<'phase.goto'>) => {
        const route = phaseRoutes[phase];
        if (!route) throw new Error('phase.goto: unknown phase');
        state.navigate(route);
      },
    }),
    'inspector.pin': command({
      id: 'inspector.pin',
      run: ({ entity }: BrowserArguments<'inspector.pin'>) => {
        if (entity.kind === 'player') {
          const roster = snapshot?.rosters.value.find(r => r.players.some(p => p.id === entity.id));
          if (roster) state.write({ subject: { ...entity, franchiseId: roster.franchiseId },
            asset: null, entity: null, inspector: 'rest' });
          else if (snapshot?.players.value.some(p => p.id === entity.id))
            state.write({ asset: `pool:${entity.id}`, subject: null, entity: null, inspector: 'rest' });
          else throw new Error('inspector.pin: unknown player');
        } else {
          if (entity.kind === 'franchise' && !snapshot?.franchises.value.some(f => f.id === entity.id))
            throw new Error('inspector.pin: unknown franchise');
          state.write({ entity, subject: null, asset: null, inspector: 'rest' });
        }
      },
    }),
    'refresh.run': command({
      id: 'refresh.run',
      run: ({ scope }: BrowserArguments<'refresh.run'>) => dataRun({ kind: 'refresh', scope }),
    }),
    'cache.run': command({
      id: 'cache.run',
      run: ({ scope }: BrowserArguments<'cache.run'>) => {
        return import('../datasources/cacheState').then(m =>
          completed(m.cacheState.run(scope), () => m.cacheState.peek().error));
      },
    }),
    'data.run': command({
      id: 'data.run',
      run: (action: BrowserArguments<'data.run'>) => dataRun(action),
    }),
    'data.parameter': command({
      id: 'data.parameter',
      run: (action: BrowserArguments<'data.parameter'>) => dataRun(action),
    }),
    'contracts.set': command({
      id: 'contracts.set',
      run: (change: BrowserArguments<'contracts.set'>) => {
        return import('../contracts/state').then(m => m.runContractsAction(change));
      },
    }),
    market: command({
      id: 'market',
      run: (change: BrowserArguments<'market'>) => {
        if (['pool-plan', 'pool-open'].includes(change.field))
          state.navigate({ node: 'war', workspace: 'valuations-and-pool' });
        if (change.field === 'inspect' || change.field === 'asset')
          state.write({ asset: change.value, entity: null, subject: null, inspector: 'rest' });
        if (change.field !== 'inspect')
          return loadMarket().then(
            (m) => m.set(change),
          );
      },
    }),
    'trade.builder': command({
      id: 'trade.builder',
      run: (action: BrowserArguments<'trade.builder'>) => {
        return import('../shell/tradeBuilderState').then(m =>
          completed(m.builderAction(action), () => m.useBuilder.getState().error));
      },
    }),
    acquisition: command({
      id: 'acquisition',
      run: (action: BrowserArguments<'acquisition'>) => {
        state.navigate({ node: 'hq', workspace: 'contracts-and-cap' });
        return import('../contracts/acquisition').then(m =>
          completed(m.openAcquisition(action), () => m.useAcquisition.getState().error));
      },
    }),
    'trade.accept.plan': command({
      id: 'trade.accept.plan',
      run: ({ tradeId }: BrowserArguments<'trade.accept.plan'>) => {
        const franchiseId = state.read().franchiseId;
        if (franchiseId)
          return import('./tradeAccept').then(
            ({ planTradeAccept }) => planTradeAccept(moves, provider, franchiseId, tradeId),
            (cause) => {
              moves.write({
                draftErrors: {
                  ...moves.read().draftErrors,
                  [`trade:${franchiseId}:${tradeId}`]: String(cause),
                },
              });
              throw cause;
            },
          );
      },
    }),
    'lineup.edit': editCommand('lineup.edit', 'edit'),
    'lineup.toggle': editCommand('lineup.toggle', 'toggle'),
    'lineup.reset': editCommand('lineup.reset', 'reset'),
    'lineup.cancel': editCommand('lineup.cancel', 'cancel'),
    'lineup.draft': editCommand('lineup.draft', 'draft'),
    'move.withdraw': editCommand('move.withdraw', 'withdraw'),
    'move.handoff': editCommand('move.handoff', 'handoff'),
    'mflkey.connect': command({
      id: 'mflkey.connect',
      run: (_args: BrowserArguments<'mflkey.connect'>) => connectMFL(),
    }),
    'mflkey.forget': command({
      id: 'mflkey.forget',
      run: (_args: BrowserArguments<'mflkey.forget'>) => {
        if (!mflReason(true)) {
          return completed(requestMFL(() => provider.forgetMFL(), 'Removing the key from the keyring…'),
            () => mflKey.read().error);
        }
      },
    }),
    'roster.ir': rosterCommand('roster.ir'),
    'roster.taxi': rosterCommand('roster.taxi'),
    'surface.open': command({
      id: 'surface.open',
      run: (args: BrowserArguments<'surface.open'>) => openPlace(args.place),
    }),
    'surface.close': command({
      id: 'surface.close',
      run: (_args: BrowserArguments<'surface.close'>) => state.write({ endpointSurface: null }),
    }),
    'report.sort': command({
      id: 'report.sort',
      run: (args: BrowserArguments<'report.sort'>) => {
        return import('../reports/sortState').then(m => m.reportSort.set(args));
      },
    }),
    'market.reports': command({
      id: 'market.reports',
      run: (args: BrowserArguments<'market.reports'>) => {
        if (args.action !== 'refresh') throw new Error('Player market reports: invalid action');
        return import('../reports/marketState').then(m => completed(m.marketReportState.refresh(true),
          () => m.marketReportState.peek().error));
      },
    }),
    'comms.read': command({
      id: 'comms.read',
      run: (args: BrowserArguments<'comms.read'>) => {
        if (!['refresh', 'team', 'kind', 'more', 'open'].includes(args.action))
          throw new Error('Comms: invalid action');
        return import('../comms/state').then(m => completed(m.commsState.choose(args.action, args.value),
          () => ['more', 'open'].includes(args.action) ? undefined : m.commsState.peek().error));
      },
    }),
    'backup.now': command({
      id: 'backup.now',
      run: (_args: BrowserArguments<'backup.now'>) => {
        state.navigate({ node: 'control', workspace: 'app' });
        return import('../commissioner/backupState').then(m =>
          completed(m.backupState.run(), () => m.backupState.peek().error));
      },
    }),
    'commissioner.surface': command({
      id: 'commissioner.surface',
      run: (args: BrowserArguments<'commissioner.surface'>) => {
        return import('../commissioner/surfaceActions').then(m => m.surfaceActions.run(args));
      },
    }),
    'commissioner.read': command({
      id: 'commissioner.read',
      run: (args: BrowserArguments<'commissioner.read'>) => {
        return import('../commissioner/proxy').then(m =>
          completed(m.leagueState.choose(args.league), () => m.leagueState.peek().error));
      },
    }),
    'commissioner.draft': command({
      id: 'commissioner.draft',
      run: (args: BrowserArguments<'commissioner.draft'>) => {
        return import('../commissioner/draftEditor').then(m => m.draftEditor.run(args));
      },
    }),
    'commissioner.execute': command({
      id: 'commissioner.execute',
      run: (args: BrowserArguments<'commissioner.execute'>) => {
        return import('../commissioner/proxy').then(m => m.execute(args.body))
          .then(receipt => state.write({ notice: receipt.message }));
      },
    }),
    'control.read': command({
      id: 'control.read',
      run: (args: BrowserArguments<'control.read'>) => {
        if (!['league', 'news', 'search', 'open-accounting'].includes(args.action))
          throw new Error('Control Room: invalid action');
        return import('../control/state').then(m => completed(m.controlState.choose(args.action, args.value),
          () => args.action === 'search' ? undefined : m.controlState.peek().error));
      },
    }),
    'research.read': command({
      id: 'research.read',
      run: (args: BrowserArguments<'research.read'>) => {
        return import('../research/state').then(m =>
          completed(m.researchState.choose(args.action, args.value),
          () => ['more', 'open'].includes(args.action) ? undefined : m.researchState.peek().error));
      },
    }),
    'race.reports': command({
      id: 'race.reports',
      run: (args: BrowserArguments<'race.reports'>) => {
        if (!['refresh', 'week', 'team', 'aggMode'].includes(args.action))
          throw new Error('League race: invalid action');
        return import('../race/state').then(m =>
          completed(m.raceState.choose(args.action, args.value), () => m.raceState.peek().error));
      },
    }),
    'game.follow': command({
      id: 'game.follow',
      run: ({ id }: BrowserArguments<'game.follow'>) => {
        state.navigate({ node: 'pulse', workspace: 'now' });
        return import('../now/liveState').then(m => m.liveState.follow(id));
      },
    }),
    'now.reports': command({
      id: 'now.reports',
      run: (args: BrowserArguments<'now.reports'>) => {
        if (!['refresh', 'week', 'reports-week'].includes(args.action))
          throw new Error('League Pulse: invalid action');
        return import('../now/state').then(m =>
          completed(m.nowState.choose(args.action, args.value), () => m.nowState.peek().error));
      },
    }),
    'draft.reports': command({
      id: 'draft.reports',
      run: (args: BrowserArguments<'draft.reports'>) => {
        if (!['refresh', 'mfl'].includes(args.action)) throw new Error('Draft reports: invalid action');
        return import('../draftreports/state').then(m =>
          completed(m.reportState.action(args.action), () => m.reportState.peek().error));
      },
    }),
    'endpoint.open': command({
      id: 'endpoint.open',
      run: (args: BrowserArguments<'endpoint.open'>) => {
        const target = resolve(args.id);
        if (target.kind === 'retired') throw new Error(`endpoint.open: retired ${args.id}`);
        const ids = target.kind === 'merged' && 'into' in target ? target.into : [args.id];
        state.write({ endpointIds: ids, endpointSurface: null });
        if (target.kind === 'view') openPlace(target.placement);
        else if ('place' in target) openPlace(target.place);
        else
          for (const id of target.into) {
            const view = resolve(id);
            if (view.kind !== 'view') throw new Error(`endpoint.open: target ${id} is not kept`);
            openPlace(view.placement);
          }
      },
    }),
    'franchise.set': command({
      id: 'franchise.set',
      run: (args: BrowserArguments<'franchise.set'>) => {
        if (!snapshot || !validFranchise(snapshot, args.franchiseId)) {
          throw new Error('franchise.set: unknown franchise');
        }
        void prefetchHQ().catch(hqFailure);
        lineups.refresh(args.franchiseId);
        trades.refresh(args.franchiseId);
        alerts.refresh(args.franchiseId);
        persistFranchise(args.franchiseId, storage());
        state.write({
          franchiseId: args.franchiseId,
          scrollRevision:
            state.read().scrollRevision + Number(state.read().franchiseId !== args.franchiseId),
        });

      },
    }),
    'preset.apply': command({
      id: 'preset.apply',
      run: (args: BrowserArguments<'preset.apply'>) => {
        if (!presetAllowed(args.preset, roles)) throw new Error('preset.apply: GM permission denied');
        const p = presets[args.preset];
        state.navigate(p.route);
        state.write({ density: p.density, inspector: p.inspector, comms: p.comms });
      },
    }),
    'inspector.open': command({
      id: 'inspector.open',
      run: (args: BrowserArguments<'inspector.open'>) => {
        const subject = args.subject;
        if (
          !snapshot?.rosters.value.some(
            (r) =>
              r.franchiseId === subject.franchiseId && r.players.some((p) => p.id === subject.id),
          ) &&
          !lineups.peek(subject.franchiseId).reading?.starters.some((p) => p.id === subject.id)
        ) {
          throw new Error('inspector.open: unknown roster player');
        }
        state.write({ subject: { ...subject }, asset: null, entity: null, inspector: 'rest' });
      },
    }),
    'inspector.collapse': command({
      id: 'inspector.collapse',
      run: (_args: BrowserArguments<'inspector.collapse'>) => state.write({ inspector: 'rest' }),
    }),
    'commandbar.close': command({
      id: 'commandbar.close',
      run: (_args: BrowserArguments<'commandbar.close'>) => state.write({ commandbar: false }),
    }),
    'nav.open': command({
      id: 'nav.open',
      run: (args: BrowserArguments<'nav.open'>) => {
        const route = routeFor(args.node, args.workspace ?? state.read().workspace[args.node]);
        if (!route) throw new Error(`nav.open: invalid workspace for ${args.node}`);
        state.navigate(route);
      },
    }),
    'density.set': command({
      id: 'density.set',
      run: (args: BrowserArguments<'density.set'>) => state.write({ density: args.density }),
    }),
    'inspector.expand': command({
      id: 'inspector.expand',
      run: (_args: BrowserArguments<'inspector.expand'>) => state.write({ inspector: 'expanded' }),
    }),
    'inspector.toggle': command({
      id: 'inspector.toggle',
      run: (_args: BrowserArguments<'inspector.toggle'>) =>
        state.write({
          inspector: state.read().inspector === 'closed' ? 'rest' : 'closed',
        }),
    }),
    'inspector.close': command({
      id: 'inspector.close',
      run: (_args: BrowserArguments<'inspector.close'>) => state.write({ inspector: 'closed' }),
    }),
    'calendar.summon': command({
      id: 'calendar.summon',
      run: (_args: BrowserArguments<'calendar.summon'>) =>
        state.write({ summoned: state.read().summoned ? null : 'calendar' }),
    }),
    'comms.toggle': command({
      id: 'comms.toggle',
      run: (_args: BrowserArguments<'comms.toggle'>) => state.write({ comms: !state.read().comms }),
    }),
    'overlays.close': command({
      id: 'overlays.close',
      run: (_args: BrowserArguments<'overlays.close'>) =>
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
      id: 'commandbar.open',
      run: (_args: BrowserArguments<'commandbar.open'>) => state.write({ commandbar: true }),
    }),
  } satisfies Record<keyof typeof browserCommands, { id: string }>);
  type Id = keyof typeof registry;
  type Args<K extends Id> = Parameters<(typeof registry)[K]['run']>[0];
  function teamActionFranchise(id: Id, args: Args<Id>): string | null | undefined {
    let mine = state.read().franchiseId;
    if (id === 'contracts.set' && 'field' in args) {
      if (!['whatif', 'release', 'release-confirm', 'contract-open', 'contract-keep',
        'contract-act'].includes(args.field)) return undefined;
      if (typeof args.value === 'object' && args.value !== null && 'mine' in args.value)
        mine = args.value.mine;
    } else if (['acquisition', 'trade.builder'].includes(id) && 'mine' in args) {
      mine = args.mine;
    } else if (['roster.ir', 'roster.taxi'].includes(id) && 'subject' in args) {
      mine = args.subject.franchiseId;
    } else if (id === 'lineup.edit' && 'franchiseId' in args) {
      mine = args.franchiseId;
    } else if (!['lineup.toggle', 'lineup.reset', 'lineup.draft', 'move.withdraw', 'move.handoff',
      'trade.accept.plan'].includes(id)) {
      return undefined;
    }
    return mine;
  }
  function teamActionReason(id: Id, args: Args<Id>): string | undefined {
    if (provider.kind !== 'live') return undefined;
    const mine = teamActionFranchise(id, args);
    if (mine === undefined) return undefined;
    const status = mflKey.read();
    if (mine && mine === status.myFranchise) return undefined;
    if (status.myFranchise) return 'Switch the franchise view to the team linked to your MFL login. ' +
      'If it is wrong, reconnect the correct MFL account.';
    return status.teamActionRefusal || 'Checking MFL…';
  }
  type Receipt = { id: Id; args: Readonly<Args<Id>>; status: 'done' | 'failed'; reason?: string };
  const history: Readonly<Receipt>[] = [];
  function dispatch<K extends Id>(id: K, args: Args<K>): void | Promise<void> {
    const selected = registry[id] as Command<Args<K>> | undefined;
    if (!selected) throw new Error('This action is not available. Refresh the app and try again.');
    if (!holdsRole(selected.roles, roles)) throw new Error(`${id}: permission denied`);
    const fromBar = state.read().commandbar;
    const savedArgs = Object.freeze({ ...args });
    function record(status: 'done' | 'failed', cause?: unknown, show = false) {
      const reason = status === 'failed'
        ? cause instanceof Error ? cause.message : String(cause) : undefined;
      history.push(Object.freeze({ id, args: savedArgs, status,
        ...(reason === undefined ? {} : { reason }) }));
      if (history.length > 32) history.shift();
      if (reason !== undefined) {
        if (show) state.write({ notice: reason });
        console.error(`Command ${id} failed:`, reason);
      }
    }
    try {
      const result = selected.run(args);
      if (fromBar && id !== 'commandbar.open' &&
        !(id === 'endpoint.open' && state.read().endpointSurface === 'Command bar')) {
        state.write({ commandbar: false });
      }
      if (result instanceof Promise) {
        const action = (savedArgs as { action?: string }).action ?? '';
        return result.then(() => record('done'), cause => record('failed', cause,
          id !== 'research.read' || ['open', 'more'].includes(action)));
      }
      record('done');
    } catch (cause) {
      record('failed', cause);
      throw cause;
    }
  }
  function key(key: string): boolean {
    for (const id of Object.keys(registry) as Id[]) {
      const selected: Command<Args<Id>> = registry[id] as Command<Args<Id>>;
      const binding = selected.keys?.find((k) => k.key.toLowerCase() === key.toLowerCase());
      if (binding) {
        dispatch(id, binding.args);
        return true;
      }
    }
    return false;
  }
  if (provider.kind === 'live') {
    void requestMFL(() => provider.mflKey(), 'Checking the keyring…');
  }
  return {
    registry,
    canDispatch: (id: Id) => holdsRole(registry[id].roles, roles),
    hasRoles: (required: readonly Role[]) => holdsRole(required, roles),
    teamActionReason,
    isTeamAction: (id: Id, args: Args<Id>) =>
      provider.kind === 'live' && teamActionFranchise(id, args) !== undefined,
    useTeamActionReason: (id: Id, args: Args<Id>) => {
      state.use();
      mflKey.use();
      return teamActionReason(id, args);
    },
    checkLineup: provider.checkLineup.bind(provider),
    onMovesChange: lineups.onMovesChange,
    useLineupEdit: state.useLineupEdit,
    lineups,
    trades,
    alerts,
    mflField,
    mflReason,
    loadMFLKey: () => requestMFL(() => provider.mflKey(), 'Checking the keyring…'),
    readMFLKey: mflKey.read,
    useMFLKey: mflKey.use,
    providerKind: provider.kind,
    draftReason,
    eligibility: (subject: PlayerSubject) => snapshot?.eligibility.players[subject.id],
    eligibilityFeeds: () => snapshot?.eligibility.feeds,
    rosterStatus: (subject: PlayerSubject) =>
      snapshot?.rosters.value
        .find((r) => r.franchiseId === subject.franchiseId)
        ?.players.find((p) => p.id === subject.id)?.rosterStatus,
    loadMoves,
    readMoves: moves.read,
    useMoves: moves.use,
    loadSnapshot: (value: Snapshot) => {
      snapshot = value;
      prefetchMoves();
      const franchiseId = loadFranchise(value, storage());
      if (franchiseId) {
        void prefetchHQ().catch(hqFailure);
        lineups.refresh(franchiseId);
        trades.refresh(franchiseId);
        alerts.refresh(franchiseId);
      }
      state.write({ franchiseId });
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
export type CommandArgs<K extends CommandId> = BrowserArguments<K>;
export const dispatch = commands.dispatch;
