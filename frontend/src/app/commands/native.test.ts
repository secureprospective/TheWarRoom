import { describe, expect, it, vi } from 'vitest';
import { nativeCommands } from './native.gen';
import { createNativeClient, nativeCapability, nativeGrammar,
  type NativeArguments, type NativeContext, type NativeRequest } from './native';

const read: NativeRequest = {
  command: 'commish.ledger.read', mode: 'read', arguments: { league: 'league.one' },
};
const context = (): NativeContext => ({ availability: 'server', capabilities: [
  { command: read.command, mode: 'read', allowed: true, reason: '' },
] });

// Checked by tsc, not executed: argument and mode types come from the same Go schema as metadata.
function typeChecks() {
  // @ts-expect-error A read cannot become an execute command.
  const upgrade: NativeRequest = { ...read, mode: 'execute' };
  // @ts-expect-error League must be text.
  const badLeague: NativeArguments<'commish.ledger.read'> = { league: 12 };
  // @ts-expect-error Logical commands cannot be substituted in a mutation envelope.
  const badCommand: NativeArguments<'commish.league.create'>['command'] = 'bids.place';
  // @ts-expect-error Server-owned outcomes are not client arguments.
  const outcome: NativeArguments<'commish.league.create'>['args'] = { tradeOutcome: {} };
  return [upgrade, badLeague, badCommand, outcome];
}
void typeChecks;

describe('catalog-generated native client', () => {
  it('sends exactly the catalog command/version/mode and named arguments', async () => {
    const port = vi.fn().mockResolvedValue({ revision: 4 });
    const client = createNativeClient(port, context);
    await expect(client.invoke(read)).resolves.toEqual({ revision: 4 });
    expect(port).toHaveBeenCalledExactlyOnceWith({ ...read, version: '1' });
  });

  it('never upgrades missing, read-only, denied, duplicate or revoked capabilities', async () => {
    const port = vi.fn();
    let current = context();
    const client = createNativeClient(port, () => current);
    await client.invoke(read);
    port.mockClear();
    for (const capabilities of [[],
      [{ ...context().capabilities[0], allowed: false, reason: 'Seat removed.' }],
      [context().capabilities[0], context().capabilities[0]],
      [{ ...context().capabilities[0], mode: 'preview' as const }]]) {
      current = { ...current, capabilities };
      await expect(client.invoke(read)).rejects.toThrow();
      expect(port).not.toHaveBeenCalled();
    }
    current = context();
    await expect(client.invoke({ ...read, mode: 'execute' } as unknown as NativeRequest))
      .rejects.toThrow('not available');
    expect(port).not.toHaveBeenCalled();
  });

  it('rejects malformed, extra and missing arguments before invoking a transport', async () => {
    const port = vi.fn();
    const client = createNativeClient(port, context);
    for (const args of [{}, { league: 3 }, { league: null }, { league: 'one', principal: 'admin' }]) {
      await expect(client.invoke({ ...read, arguments: args } as NativeRequest)).rejects.toThrow();
    }
    expect(port).not.toHaveBeenCalled();
  });

  it('renders modes and named grammar from every catalog entry, without role inference', () => {
    const rows = nativeGrammar({ availability: 'server', capabilities: [] });
    const count = Object.values(nativeCommands).reduce((n, c) => n + c.definition.Modes.length, 0);
    expect(rows.length).toBe(count);
    expect(rows.every(row => !row.enabled && row.reason)).toBe(true);
    expect(nativeGrammar(context(), 'read league ledger')).toContainEqual(expect.objectContaining({
      command: read.command, mode: 'read', enabled: true,
      arguments: [{ name: 'league', required: true }],
    }));
    expect(nativeGrammar(context(), 'admin')).toEqual([]);
  });

  it('cannot turn preview authority into execution or publish a desktop-only import', () => {
    const preview: NativeContext = { availability: 'server', capabilities: [
      { command: 'commish.league.create', mode: 'preview', allowed: true, reason: '' },
      { command: 'commish.league.seed_held', mode: 'execute', allowed: true, reason: '' },
    ] };
    expect(nativeCapability('commish.league.create', 'preview', preview).enabled).toBe(true);
    expect(nativeCapability('commish.league.create', 'execute', preview).enabled).toBe(false);
    expect(nativeCapability('commish.league.seed_held', 'execute', preview)).toEqual({
      enabled: false, reason: 'This action needs the desktop app.',
    });
  });

  it('retains reason, request key and expected revision without inventing approval', async () => {
    type Create = NativeArguments<'commish.league.create'>;
    const args: Create = { league: '', command: 'commish.league.create',
      args: { name: 'Our league', preset: 'empty', franchiseCount: 12, season: 2027 },
      reason: 'We agreed to start', expected: 0, request: 'request.one',
      effective: 'now', visibility: 'league' };
    const port = vi.fn();
    const client = createNativeClient(port, () => ({ availability: 'server', capabilities: [
      { command: 'commish.league.create', mode: 'execute', allowed: true, reason: '' },
    ] }));
    await client.invoke({ command: 'commish.league.create', mode: 'execute', arguments: args });
    expect(port).toHaveBeenCalledWith({ command: 'commish.league.create', version: '1',
      mode: 'execute', arguments: args });
    expect(nativeCommands['commish.league.create'].definition.Safeguards)
      .toBe('RawCommand.EffectiveDefinition+MissingReason');
  });
});
