import { describe, it, expect } from 'vitest';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { createCommands } from '../commands/registry';
import { FRANCHISE_KEY, loadFranchise } from './franchise';
import { franchiseRoster } from './roster';
import { presetIds, presets, presetAllowed } from './presets';
import { routeFor } from './nodes';
import { dispatchKey } from './bindings';

const snapshot = parseSnapshot(fixture);

function fakeStorage(initial?: string) {
  const values = new Map<string, string>();
  if (initial !== undefined) values.set(FRANCHISE_KEY, initial);
  return {
    values,
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
  };
}

describe('App-local franchise', () => {
  it('validates persisted ids and persists only known franchises through dispatch', () => {
    const storage = fakeStorage('not-a-franchise');
    const c = createCommands(() => storage);
    c.loadSnapshot(snapshot);
    expect(c.read().franchiseId).toBeNull();
    expect(loadFranchise(snapshot, fakeStorage())).toBeNull();
    const id = snapshot.franchises.value[0].id;
    c.dispatch('franchise.set', { franchiseId: id });
    expect(storage.values).toEqual(new Map([[FRANCHISE_KEY, id]]));
    expect(c.read().franchiseId).toBe(id);
    const reloaded = createCommands(() => storage);
    reloaded.loadSnapshot(snapshot);
    expect(reloaded.read().franchiseId).toBe(id);
    expect(() => c.dispatch('franchise.set', { franchiseId: 'unknown' })).toThrow(
      'unknown franchise',
    );
    expect(storage.getItem(FRANCHISE_KEY)).toBe(id);
  });
  it('does not hide storage failures or apply an unpersisted selection', () => {
    const storage = {
      getItem: () => null,
      setItem: () => {
        throw new Error('storage denied');
      },
    };
    const c = createCommands(() => storage);
    c.loadSnapshot(snapshot);
    expect(() =>
      c.dispatch('franchise.set', {
        franchiseId: snapshot.franchises.value[0].id,
      }),
    ).toThrow('storage denied');
    expect(c.read().franchiseId).toBeNull();
  });
});

describe('Fixture roster', () => {
  it('groups the Jets Active/IR/Taxi and uses snapshot cap figures', () => {
    const jets = snapshot.franchises.value.find((f) => f.name === 'New York Jets')!;
    const result = franchiseRoster(snapshot, jets.id);
    expect(result.groups.map((g) => [g.label, g.players.length])).toEqual([
      ['Active', 38],
      ['IR', 2],
      ['Taxi', 5],
    ]);
    expect(result).toMatchObject({
      capUsed: 11894450000,
      capRoom: 605550000,
      salaryCap: 12500000000,
    });
    expect(result.groups[0].players[0]).toMatchObject({
      id: '13593',
      salary: 2177000000,
    });
    const names = new Map(snapshot.players.value.map((p) => [p.id, p.name ?? p.id]));
    let ties = 0;
    for (const group of result.groups) {
      for (let i = 1; i < group.players.length; i++) {
        const previous = group.players[i - 1];
        const current = group.players[i];
        expect(previous.salary).toBeGreaterThanOrEqual(current.salary);
        if (previous.salary === current.salary) {
          ties++;
          expect(
            names.get(previous.id)!.localeCompare(names.get(current.id)!),
          ).toBeLessThanOrEqual(0);
        }
      }
    }
    expect(ties).toBeGreaterThan(0);
    expect(() => franchiseRoster(snapshot, 'unknown')).toThrow('unknown franchise');
  });
});

it('opens, expands, collapses and closes a real player via dispatch', () => {
  const c = createCommands(() => undefined);
  c.loadSnapshot(snapshot);
  const roster = snapshot.rosters.value[0];
  const subject = {
    kind: 'player' as const,
    id: roster.players[0].id,
    franchiseId: roster.franchiseId,
  };
  c.dispatch('inspector.open', { subject });
  expect(c.read()).toMatchObject({ inspector: 'rest', subject });
  c.dispatch('inspector.expand', {});
  expect(c.read().inspector).toBe('expanded');
  c.dispatch('inspector.collapse', {});
  expect(c.read().inspector).toBe('rest');
  c.dispatch('inspector.close', {});
  expect(c.read().inspector).toBe('closed');
  expect(() =>
    c.dispatch('inspector.open', {
      subject: { ...subject, id: 'unknown' },
    }),
  ).toThrow('unknown roster player');
});

it('presets reference real workspaces, apply all four fields, and deny Admin for GM', () => {
  const c = createCommands(() => undefined);
  for (const id of presetIds) {
    const p = presets[id];
    expect(routeFor(p.route.node, p.route.workspace)).toEqual(p.route);
    if (!presetAllowed(id)) {
      const before = c.read();
      expect(() => c.dispatch('preset.apply', { preset: id })).toThrow(
        'GM permission denied',
      );
      expect(c.read()).toEqual(before);
      continue;
    }
    c.dispatch('preset.apply', { preset: id });
    expect(c.read()).toMatchObject({
      node: p.route.node,
      workspace: { [p.route.node]: p.route.workspace },
      density: p.density,
      inspector: p.inspector,
      comms: p.comms,
    });
  }
});

it('opens with / and Ctrl+K but never intercepts typing', () => {
  const c = createCommands(() => undefined);
  const event = { key: '/', target: null, ctrlKey: false, metaKey: false, altKey: false };
  expect(dispatchKey(event, c)).toBe(true);
  expect(c.read().commandbar).toBe(true);
  c.dispatch('commandbar.close', {});
  expect(dispatchKey({ ...event, key: 'k', ctrlKey: true }, c)).toBe(true);
  expect(c.read().commandbar).toBe(true);
  c.dispatch('commandbar.close', {});
  for (const key of ['/', 'k']) {
    expect(
      dispatchKey(
        {
          ...event,
          key,
          ctrlKey: key === 'k',
          target: { tagName: 'INPUT' } as unknown as EventTarget,
        },
        c,
      ),
    ).toBe(false);
  }
  expect(c.read().commandbar).toBe(false);
});
