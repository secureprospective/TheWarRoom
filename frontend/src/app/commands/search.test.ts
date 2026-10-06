import { it, expect } from 'vitest';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { createCommands } from './registry';
import {
  rankResults, runResult, searchCandidates, indexResult, type SearchResult, type IndexedResult,
} from './search';

const snapshot = parseSnapshot(fixture);

function candidate(
  label: string,
  kind: SearchResult['kind'] = 'command',
  aliases: string[] = [],
): IndexedResult {
  return indexResult({ label, kind, aliases, invocation: { verb: 'inspector.close', args: {} } });
}

it('ranks prefix before word-start before substring; ties by kind then label', () => {
  const candidates = [
    candidate('Xtrade'),
    candidate('Open trade'),
    candidate('Trade Z', 'player'),
    candidate('Trade B', 'command'),
    candidate('Trade A', 'place'),
    candidate('Trade A', 'command'),
    candidate('Nothing'),
  ];
  expect(rankResults(' TRADE ', candidates).map((r) => `${r.kind}:${r.label}`)).toEqual([
    'command:Trade A',
    'command:Trade B',
    'place:Trade A',
    'player:Trade Z',
    'command:Open trade',
    'command:Xtrade',
  ]);
  expect(rankResults('deal', [candidate('Trade', 'command', ['deal'])])).toHaveLength(1);
  expect(rankResults('none', candidates)).toEqual([]);
  expect(
    rankResults(
      '',
      Array.from({ length: 12 }, (_, i) => candidate(`Item ${i}`)),
    ),
  ).toHaveLength(8);
  expect(candidates[0].label).toBe('Xtrade');
});

it('maps places, named presets, density and rostered players to registered verbs', () => {
  const c = createCommands(() => undefined);
  c.loadSnapshot(snapshot);
  const results = searchCandidates(c, snapshot);
  expect(results.filter((r) => r.kind === 'place')).toHaveLength(18);
  expect(results.filter((r) => r.kind === 'player')).toHaveLength(
    snapshot.rosters.value.reduce((sum, r) => sum + r.players.length, 0),
  );
  expect(results.find((r) => r.label === 'Preset: Gameday')?.invocation).toEqual({
    verb: 'preset.apply',
    args: { preset: 'gameday' },
  });
  expect(results.find((r) => r.label === 'Density: matrix · M')?.invocation).toEqual({
    verb: 'density.set',
    args: { density: 'matrix' },
  });
  expect(results.some((r) => r.invocation.verb === 'franchise.set')).toBe(false);
  expect(results.some((r) => r.label === 'Preset: Admin')).toBe(false);
  for (const result of results) {
    const command = c.registry[result.invocation.verb];
    expect(command.roles).toContain('gm');
    for (const arg of command.args) expect(result.invocation.args).toHaveProperty(arg);
  }
  const place = results.find((r) => r.label === 'War Room › Research')!;
  c.dispatch('commandbar.open', {});
  runResult(c, place);
  expect(c.read()).toMatchObject({
    node: 'war',
    workspace: { war: 'research' },
    commandbar: false,
  });
  expect(c.history().some((entry) => entry.id === 'nav.open')).toBe(true);
});

it('navigates only for a chosen-franchise player and treats pointer and Enter alike', () => {
  const c = createCommands(() => undefined);
  c.loadSnapshot(snapshot);
  const roster = snapshot.rosters.value[0];
  c.dispatch('franchise.set', { franchiseId: roster.franchiseId });
  const results = searchCandidates(c, snapshot);
  const own = results.find(
    (r) =>
      r.invocation.verb === 'inspector.open' &&
      r.invocation.args.subject.franchiseId === roster.franchiseId,
  )!;
  const other = results.find(
    (r) =>
      r.invocation.verb === 'inspector.open' &&
      r.invocation.args.subject.franchiseId !== roster.franchiseId,
  )!;
  c.dispatch('nav.open', { node: 'war', workspace: 'research' });
  c.dispatch('commandbar.open', {});
  runResult(c, other);
  expect(c.read()).toMatchObject({ node: 'war', inspector: 'rest', commandbar: false });
  c.dispatch('commandbar.open', {});
  runResult(c, own);
  expect(c.read()).toMatchObject({ node: 'hq', workspace: { hq: 'lineup-and-roster' } });
  c.dispatch('nav.open', { node: 'war', workspace: 'research' });
  c.dispatch('commandbar.open', {});
  c.dispatch(own.invocation.verb, own.invocation.args);
  expect(c.read()).toMatchObject({ node: 'hq', inspector: 'rest', commandbar: false });
});
