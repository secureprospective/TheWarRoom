import { endpoints } from '../registry';
import type { Snapshot, Position } from '../data/contract';
import { nodes, nodeKeys } from '../shell/nodes';
import { presetIds, presets, presetAllowed } from '../shell/presets';
import type { createCommands, CommandId, CommandArgs } from './registry';

// The bar's own plumbing is never a search result: offering "Close command bar" inside the open
// command bar is noise, and Escape already does it.
const UNSEARCHABLE: ReadonlySet<CommandId> = new Set<CommandId>([
  'commandbar.open',
  'commandbar.close',
  'overlays.close',
  'surface.close',
]);

type Invocation = {
  [K in CommandId]: { verb: K; args: CommandArgs<K> };
}[CommandId];

export type SearchResult = {
  label: string;
  aliases: readonly string[];
  kind: 'command' | 'place' | 'endpoint' | 'player';
  position?: Position;
  invocation: Invocation;
};

export type IndexedResult = SearchResult & { texts: readonly string[]; words: readonly string[] };

export function indexResult(result: SearchResult): IndexedResult {
  const texts = [result.label, ...result.aliases].map((text) => text.toLowerCase());
  return { ...result, texts, words: texts.flatMap((text) => text.split(/\s+/)) };
}

type Executor = ReturnType<typeof createCommands>;

export function searchCandidates(executor: Executor, snapshot?: Snapshot): IndexedResult[] {
  const results: SearchResult[] = [];
  for (const id of Object.keys(executor.registry) as CommandId[]) {
    const command = executor.registry[id];
    if (command.args.length || !command.roles.includes('gm') || UNSEARCHABLE.has(id)) continue;
    results.push({
      label: command.label + (command.keys?.length ? ` · ${command.keys.map((k) => k.key).join(', ')}` : ''),
      aliases: command.aliases,
      kind: 'command',
      invocation: { verb: id, args: {} } as Invocation,
    });
  }
  for (const preset of presetIds.filter(presetAllowed)) {
    results.push({
      label: `Preset: ${presets[preset].label}`,
      aliases: [presets[preset].label],
      kind: 'command',
      invocation: { verb: 'preset.apply', args: { preset } },
    });
  }
  for (const density of ['narrative', 'tactical', 'matrix'] as const) {
    results.push({
      label: `Density: ${density} · ${['N', 'T', 'M'][['narrative', 'tactical', 'matrix'].indexOf(density)]}`,
      aliases: [density],
      kind: 'command',
      invocation: { verb: 'density.set', args: { density } },
    });
  }
  for (const node of nodeKeys) {
    for (const workspace of nodes[node].workspaces) {
      results.push({
        label: `${nodes[node].label} › ${workspace.label}`,
        aliases: [workspace.label],
        kind: 'place',
        invocation: {
          verb: 'nav.open',
          args: { node, workspace: workspace.slug },
        } as Invocation,
      });
    }
  }
  for (const row of endpoints.filter((e) => e.disposition === 'kept')) {
    results.push({
      label: `${row.id} · ${row.name} · ${row.placement}`,
      aliases: [row.name, row.id],
      kind: 'endpoint',
      invocation: { verb: 'endpoint.open', args: { id: row.id } },
    });
  }
  const owners = new Map((snapshot?.franchises.value ?? [])
    .map((franchise) => [franchise.id, franchise.name]));
  const players = new Map((snapshot?.players.value ?? []).map((player) => [player.id, player]));
  for (const roster of snapshot?.rosters.value ?? []) {
    for (const player of roster.players) {
      const identity = players.get(player.id);
      const name = identity?.name ?? player.id;
      const owner = owners.get(roster.franchiseId);
      const context = [name, identity?.position, identity?.team, owner ?? roster.franchiseId]
        .filter(Boolean).join(' · ');
      results.push({
        label: context,
        aliases: [],
        kind: 'player',
        position: identity?.position,
        invocation: {
          verb: 'inspector.open',
          args: {
            subject: { kind: 'player', id: player.id, franchiseId: roster.franchiseId },
          },
        },
      });
    }
  }
  return results.map(indexResult);
}

export function rankResults(
  query: string,
  candidates: readonly IndexedResult[],
): SearchResult[] {
  const normalized = query.trim().toLowerCase();
  const order = { command: 0, place: 1, endpoint: 2, player: 3 };
  return candidates
    .map((result) => ({
      result,
      score: result.texts.some((text) => text.startsWith(normalized)) ? 0 :
        result.words.some((word) => word.startsWith(normalized)) ? 1 :
        result.texts.some((text) => text.includes(normalized)) ? 2 : 3,
    }))
    .filter((entry) => entry.score < 3)
    .sort(
      (a, b) =>
        a.score - b.score ||
        order[a.result.kind] - order[b.result.kind] ||
        (a.result.label < b.result.label ? -1 : a.result.label > b.result.label ? 1 : 0),
    )
    .slice(0, 8)
    .map((entry) => entry.result);
}

export function runResult(executor: Executor, result: SearchResult) {
  const { verb, args } = result.invocation;
  executor.dispatch(verb, args);
}
