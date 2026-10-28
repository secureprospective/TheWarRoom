import { describe, expect, it } from 'vitest';
import { browserCommands } from './browser.gen';
import before from './browser-before.json';
import { createCommands, type CommandId, type Role } from './registry';
import { searchCandidates } from './search';

const local = createCommands();
const roles: Role[] = ['gm', 'commish', 'admin'];
const hidden = new Set(['commandbar.open', 'commandbar.close', 'overlays.close', 'surface.close']);

const retained = before.filter(c => c.id !== 'offers.filter');

it('retains the browser inventory except removed offer history, with immutable metadata', () => {
  const ids = retained.map(c => c.id).sort();
  expect(Object.keys(local.registry).sort()).toEqual(ids);
  expect(Object.keys(browserCommands).sort()).toEqual(ids);
  expect(Object.isFrozen(local.registry)).toBe(true);
  for (const id of Object.keys(local.registry) as CommandId[]) {
    const { run: _run, ...metadata } = local.registry[id];
    expect(metadata).toEqual(before.find(c => c.id === id));
    expect(metadata.id).toBe(id);
    expect(Object.isFrozen(local.registry[id])).toBe(true);
    for (const field of ['aliases', 'roles', 'args'] as const)
      expect(Object.isFrozen(metadata[field])).toBe(true);
    expect(metadata.gravity === 'G3').toBe(metadata.undo === 'irreversible');
  }
});

describe.each(roles)('%s browser door parity', role => {
  const caller = createCommands(undefined, undefined, undefined, [role]);
  const searchable = new Set(searchCandidates(caller)
    .filter(c => c.kind === 'command' && Object.keys(c.invocation.args).length === 0)
    .map(c => c.invocation.verb));
  it.each(retained)('$id keeps admission and direct command-bar exposure', old => {
    const id = old.id as CommandId;
    const allowed = old.roles.includes(role);
    expect(caller.canDispatch(id)).toBe(allowed);
    expect(searchable.has(id)).toBe(allowed && old.args.length === 0 && !hidden.has(id));
  });
});
