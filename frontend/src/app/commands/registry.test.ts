import { describe, it, expect, vi } from 'vitest';
import { dataState } from '../datasources/state';
import { createCommands } from './registry';
import type { CommandId, CommandArgs } from './registry';

describe('Command registry', () => {
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

it('records a stored read failure instead of successful completion', async () => {
  const { cacheState } = await import('../datasources/cacheState');
  const reason = 'The saved reading could not be read again. Try again later.';
  const run = vi.spyOn(cacheState, 'run').mockImplementation(async () => {});
  const peek = vi.spyOn(cacheState, 'peek').mockReturnValue({ scopes: [], busy: '',
    error: reason, outcomes: { race: reason } });
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    const c = createCommands();
    await c.dispatch('cache.run', { scope: 'race' });
    expect(c.history()).toEqual([{ id: 'cache.run', args: { scope: 'race' }, status: 'failed', reason }]);
    expect(c.read().notice).toBe(reason);
  } finally { run.mockRestore(); peek.mockRestore(); logged.mockRestore(); }
});

it.each(['done', 'failed'] as const)('records market filter %s after its readings settle', async status => {
  const { marketState } = await import('../market/state');
  const { unavailable } = await import('../market/provider');
  const reason = 'The trade archive cannot be read. Try again.';
  let finish!: () => void;
  const change = vi.spyOn(marketState, 'change').mockImplementation(() =>
    new Promise<void>(resolve => { finish = resolve; }));
  const before = marketState.peek();
  const peek = vi.spyOn(marketState, 'peek').mockReturnValue({ ...before,
    lists: status === 'failed' ? unavailable(reason) : undefined, clock: undefined, stretches: undefined });
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    const c = createCommands();
    const result = c.dispatch('market', { field: 'retry', value: '' });
    await vi.waitFor(() => expect(change).toHaveBeenCalledTimes(1));
    expect(c.history()).toEqual([]);
    finish();
    await result;
    expect(c.history()).toEqual([{ id: 'market', args: { field: 'retry', value: '' }, status,
      ...(status === 'failed' ? { reason } : {}) }]);
  } finally { change.mockRestore(); peek.mockRestore(); logged.mockRestore(); }
});

it('records a returned offer-history failure in plain English', async () => {
  const { offerState } = await import('../offers/state');
  const reason = 'The offer history cannot be read. Try again.';
  const change = vi.spyOn(offerState, 'change').mockResolvedValue(undefined);
  const before = offerState.peek();
  const peek = vi.spyOn(offerState, 'peek').mockReturnValue({ ...before,
    reading: { status: 'failed', absenceReason: reason } as NonNullable<typeof before.reading> });
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    const c = createCommands();
    await c.dispatch('offers.filter', { field: 'retry', value: '' });
    expect(c.history()).toEqual([{ id: 'offers.filter', args: { field: 'retry', value: '' },
      status: 'failed', reason }]);
    expect(c.read().notice).toBe(reason);
  } finally { change.mockRestore(); peek.mockRestore(); logged.mockRestore(); }
});
