import { it, expect } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { PlayerCard } from '../cards/PlayerCard';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { createCommands } from '../commands/registry';
import { searchCandidates } from '../commands/search';
import { matrixColumns, matrixCells } from '../cards/matrix';
import { playerCardModel, inheritedProvenance } from '../cards/playerModel';
import { franchiseRoster } from './roster';

const snapshot = parseSnapshot(fixture);
const roster = snapshot.rosters.value[0];

it('aligns all nine matrix columns, showing unwired numbers as em dashes only', () => {
  expect(matrixColumns.map((column) => column.label)).toEqual([
    'Pos', 'Player', 'Team', 'Age', 'On-field-now', 'Dynasty value', 'Salary', 'Years', 'Status',
  ]);
  const model = playerCardModel(snapshot, roster.franchiseId, roster.players[0].id, new Date());
  const cells = matrixCells(model);
  expect(cells).toHaveLength(matrixColumns.length);
  expect(cells.slice(4, 6)).toEqual(['—', '—']);
  expect(matrixColumns.filter((column) => 'note' in column).map((column) => column.key))
    .toEqual(['now', 'dynasty']);
  expect(cells[1]).toBe(model.name);
  expect(cells[6]).toBe(model.salary);
  expect(cells[8]).toBe(model.contractStatus);
  expect(matrixCells({ ...model, age: undefined, years: undefined })[7]).toBe('—');
});

it('inherits identical list provenance, but keeps differing source, date or severity own', () => {
  const list = franchiseRoster(snapshot, roster.franchiseId);
  for (const group of list.groups) {
    for (const card of group.cards) {
      expect(card.provenance).toBe('inherited');
      expect(card.model.provenance).toEqual(list.provenance);
    }
  }
  const own = list.groups[0].cards[0].model.provenance;
  expect(inheritedProvenance(own, { ...own })).toBe('inherited');
  for (const different of [
    { ...own, detail: 'other source' },
    { ...own, label: 'fixture · 10-04' },
    { ...own, signal: 'red' as const },
  ]) expect(inheritedProvenance(different, own)).toBe('own');
});

it('resets scroll only for franchise changes and different workspace navigation', () => {
  const c = createCommands(() => undefined);
  c.loadSnapshot(snapshot);
  const revision = () => c.read().scrollRevision;
  expect(revision()).toBe(0);
  c.dispatch('density.set', { density: 'matrix' });
  c.dispatch('inspector.expand', {});
  c.dispatch('commandbar.open', {});
  expect(revision()).toBe(0);
  c.dispatch('franchise.set', { franchiseId: roster.franchiseId });
  expect(revision()).toBe(1);
  c.dispatch('franchise.set', { franchiseId: roster.franchiseId });
  expect(revision()).toBe(1);
  c.dispatch('nav.open', { node: 'hq', workspace: 'lineup-and-roster' });
  expect(revision()).toBe(2);
  c.dispatch('nav.open', { node: 'hq', workspace: 'lineup-and-roster' });
  expect(revision()).toBe(2);
  c.dispatch('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
  expect(revision()).toBe(3);
  expect(() => c.dispatch('franchise.set', { franchiseId: 'unknown' })).toThrow();
  expect(revision()).toBe(3);
  c.dispatch('franchise.set', { franchiseId: snapshot.rosters.value[1].franchiseId });
  expect(revision()).toBe(4);
});

it('carries player ownership and NFL context, place hierarchy and command bindings', () => {
  const c = createCommands(() => undefined);
  const candidates = searchCandidates(c, snapshot);
  const murray = candidates.find((result) => result.label.startsWith('Murray, Kyler ·'));
  expect(murray?.label).toBe('Murray, Kyler · QB · MIN · Cincinnati Bengals');
  expect(candidates.find((result) => result.label === 'War Room › Research')?.kind).toBe('place');
  expect(candidates.find((result) => result.invocation.verb === 'inspector.toggle')?.label)
    .toBe('Toggle inspector · I');
  for (const plumbing of ['commandbar.open', 'commandbar.close', 'overlays.close']) {
    expect(candidates.some((result) => result.invocation.verb === plumbing)).toBe(false);
  }
  expect(candidates.find((result) => result.label === 'Density: matrix · M')?.invocation)
    .toEqual({ verb: 'density.set', args: { density: 'matrix' } });
  expect(murray?.texts).toContain(murray!.label.toLowerCase());
  expect(murray?.words).toContain('kyler');
});

it('renders a card dot only for its own provenance', () => {
  const props = {
    snapshot, franchiseId: roster.franchiseId, playerId: roster.players[0].id, asOf: new Date(),
  };
  const inherited = renderToStaticMarkup(createElement(PlayerCard, { ...props, provenance: 'inherited' }));
  const own = renderToStaticMarkup(createElement(PlayerCard, { ...props, provenance: 'own' }));
  expect(inherited).not.toContain('card-provenance');
  expect(own).toContain('card-provenance');
});
