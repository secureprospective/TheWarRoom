import { describe, it, expect } from 'vitest';
import { createCommands } from './registry';
import type { Command, CommandId, CommandArgs } from './registry';

function validGravity(c: Pick<Command<{}>, 'gravity' | 'undo'>) {
  return (c.gravity === 'G3') === (c.undo === 'irreversible');
}
describe('Command registry', () => {
  it('has unique IDs, immutable metadata and consistent gravity/undo', () => {
    const { registry } = createCommands();
    const entries = Object.values(registry);
    expect(new Set(entries.map((c) => c.id)).size).toBe(entries.length);
    expect(Object.isFrozen(registry)).toBe(true);
    for (const [id, command] of Object.entries(registry)) {
      expect(command.id).toBe(id);
      expect(validGravity(command)).toBe(true);
      expect(Object.isFrozen(command)).toBe(true);
      expect(Object.isFrozen(command.roles)).toBe(true);
      expect(Object.isFrozen(command.args)).toBe(true);
      expect(Object.isFrozen(command.aliases)).toBe(true);
      expect(command.roles).toContain('gm');
    }
    expect(validGravity({ gravity: 'G3', undo: 'instant' })).toBe(false);
    expect(validGravity({ gravity: 'G0', undo: 'irreversible' })).toBe(false);
    expect(validGravity({ gravity: 'G3', undo: 'irreversible' })).toBe(true);
  });
  it('records only successful dispatches in a bounded immutable log', () => {
    const c = createCommands();
    const args: CommandArgs<'density.set'> = { density: 'matrix' };
    c.dispatch('density.set', args);
    args.density = 'narrative';
    expect(c.history()[0]).toEqual({ id: 'density.set', args: { density: 'matrix' } });
    expect(() =>
      c.dispatch('nav.open', {
        node: 'home',
        workspace: 'market',
      } as unknown as CommandArgs<'nav.open'>),
    ).toThrow('invalid workspace');
    expect(c.history()).toHaveLength(1);
    for (let i = 0; i < 40; i++) c.dispatch('inspector.close', {});
    expect(c.history()).toHaveLength(32);
    expect(Object.isFrozen(c.history())).toBe(true);
    expect(Object.isFrozen(c.history()[0].args)).toBe(true);
  });
  it('keeps IDs and args type-checked', () => {
    const id: CommandId = 'nav.open';
    expect(id).toBe('nav.open');
    // @ts-expect-error unknown verb
    const badId: CommandId = 'nav.missing';
    // @ts-expect-error density must use the card contract
    const badArgs: CommandArgs<'density.set'> = { density: 'compact' };
    // @ts-expect-error workspace must belong to its node
    const badRoute: CommandArgs<'nav.open'> = { node: 'home', workspace: 'market' };
    expect([badId, badArgs, badRoute]).toHaveLength(3);
  });
});
