import { describe, it, expect, vi } from 'vitest';
import { dataState } from '../datasources/state';
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
      if (id === 'data.parameter') expect(command.roles).toEqual(['admin']);
      else expect(command.roles).toContain('gm');
    }
    expect(validGravity({ gravity: 'G3', undo: 'instant' })).toBe(false);
    expect(validGravity({ gravity: 'G0', undo: 'irreversible' })).toBe(false);
    expect(validGravity({ gravity: 'G3', undo: 'irreversible' })).toBe(true);
  });
  it('records completed and failed dispatches in a bounded immutable log', () => {
    const c = createCommands();
    const args: CommandArgs<'density.set'> = { density: 'matrix' };
    c.dispatch('density.set', args);
    args.density = 'narrative';
    expect(c.history()[0]).toEqual({ id: 'density.set', args: { density: 'matrix' }, status: 'done' });
    expect(() =>
      c.dispatch('nav.open', {
        node: 'home',
        workspace: 'market',
      } as unknown as CommandArgs<'nav.open'>),
    ).toThrow('invalid workspace');
    expect(c.history()).toHaveLength(2);
    expect(c.history()[1]).toMatchObject({ id: 'nav.open', status: 'failed',
      reason: 'nav.open: invalid workspace for home' });
    for (let i = 0; i < 40; i++) c.dispatch('inspector.close', {});
    expect(c.history()).toHaveLength(32);
    expect(Object.isFrozen(c.history())).toBe(true);
    expect(Object.isFrozen(c.history()[0].args)).toBe(true);
  });
  it('dispatches held roles and refuses unheld roles without effects', async () => {
    const run = vi.spyOn(dataState, 'run').mockResolvedValue(undefined);
    try {
      const action = { kind: 'parameter' as const, key: 'dynasty.discount',
        position: '', value: 0.75 };
      const admin = createCommands(undefined, undefined, undefined, ['admin']);
      await admin.dispatch('data.parameter', action);
      await vi.waitFor(() => expect(run).toHaveBeenCalledWith(action));
      expect(admin.history()).toEqual([{ id: 'data.parameter', args: action, status: 'done' }]);
      const gm = createCommands(undefined, undefined, undefined, ['gm']);
      expect(() => gm.dispatch('data.parameter', action)).toThrow('permission denied');
      expect(gm.history()).toHaveLength(0);
      expect(run).toHaveBeenCalledTimes(1);
      expect(() => createCommands(undefined, undefined, undefined, []).dispatch('inspector.close', {}))
        .toThrow('permission denied');
      const mixed = createCommands(undefined, undefined, undefined, ['gm', 'admin']);
      mixed.dispatch('data.parameter', action);
      await vi.waitFor(() => expect(run).toHaveBeenCalledTimes(2));
      expect(Object.keys(mixed.registry)).toHaveLength(62);
    } finally { run.mockRestore(); }
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

it('records async success/failure only after completion, with a visible failure reason', async () => {
  let finish!: () => void;
  let fail!: (cause: Error) => void;
  const run = vi.spyOn(dataState, 'run');
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    for (const status of ['done', 'failed'] as const) {
      const c = createCommands();
      run.mockImplementationOnce(() => new Promise<void>((resolve, reject) => {
        finish = resolve; fail = reject;
      }));
      const result = c.dispatch('refresh.run', { scope: 'players' });
      await vi.waitFor(() => expect(run).toHaveBeenCalled());
      expect(c.history()).toEqual([]);
      if (status === 'done') finish();
      else fail(new Error('The data source is unavailable. Try again later.'));
      await result;
      expect(c.history()).toEqual([expect.objectContaining({ id: 'refresh.run', status })]);
      if (status === 'failed') {
        expect(c.history()[0].reason).toBe('The data source is unavailable. Try again later.');
        expect(c.read().notice).toBe(c.history()[0].reason);
        expect(logged).toHaveBeenCalled();
      }
      run.mockClear();
    }
  } finally { run.mockRestore(); logged.mockRestore(); }
});

it('shows, logs and records an unknown mounted league action as failed', async () => {
  const c = createCommands();
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    await c.dispatch('commissioner.surface', { screen: 'unmounted', action: 'open', value: '' });
    expect(c.history()).toEqual([expect.objectContaining({ status: 'failed',
      reason: 'This league view is no longer open. Open it again and retry your action.' })]);
    expect(c.read().notice).toBe(c.history()[0].reason);
    expect(logged).toHaveBeenCalled();
  } finally { logged.mockRestore(); }
});
