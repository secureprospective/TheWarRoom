import type { Snapshot } from '../data/contract';
import { nodes, nodeKeys } from '../shell/nodes';
import { presetIds, presets, presetAllowed } from '../shell/presets';
import type { createCommands, CommandId, CommandArgs } from './registry';

type Invocation = {
  [K in CommandId]: { verb: K; args: CommandArgs<K> };
}[CommandId];

export type SearchResult = {
  label: string;
  aliases: readonly string[];
  kind: 'command' | 'place' | 'player';
  invocation: Invocation;
};

type Executor = ReturnType<typeof createCommands>;

export function searchCandidates(executor: Executor, snapshot: Snapshot): SearchResult[] {
  const results: SearchResult[] = [];
  for (const id of Object.keys(executor.registry) as CommandId[]) {
    const command = executor.registry[id];
    if (command.args.length || !command.roles.includes('gm')) continue;
    results.push({
      label: command.label,
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
      label: `Density: ${density}`,
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
  const players = new Map(snapshot.players.value.map((player) => [player.id, player]));
  for (const roster of snapshot.rosters.value) {
    for (const player of roster.players) {
      const name = players.get(player.id)?.name ?? player.id;
      results.push({
        label: name,
        aliases: [],
        kind: 'player',
        invocation: {
          verb: 'inspector.open',
          args: {
            subject: { kind: 'player', id: player.id, franchiseId: roster.franchiseId },
          },
        },
      });
    }
  }
  return results;
}

function match(text: string, query: string): number {
  const value = text.toLowerCase();
  if (value.startsWith(query)) return 0;
  if (value.split(/\s+/).some((word) => word.startsWith(query))) return 1;
  return value.includes(query) ? 2 : 3;
}

export function rankResults(
  query: string,
  candidates: readonly SearchResult[],
): SearchResult[] {
  const normalized = query.trim().toLowerCase();
  const order = { command: 0, place: 1, player: 2 };
  return candidates
    .map((result) => ({
      result,
      score: Math.min(
        ...[result.label, ...result.aliases].map((text) => match(text, normalized)),
      ),
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
