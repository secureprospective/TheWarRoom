import type { CommandArguments } from './arguments.gen';
import { describe, expect, it } from 'vitest';
import { nativeCommands } from './native.gen';
import { nativeEnvelope, type NativeArguments, type NativeRequest } from './native';

const read: NativeRequest = {
  command: 'commish.ledger.read', mode: 'read', arguments: { league: 'league.one' },
};
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
  // @ts-expect-error Browser arguments preserve the local domain choices.
  const browser: CommandArguments<'browser', 'density.set'> = { density: 'unknown' };
  // @ts-expect-error Wails keeps positional arguments and their original scalar types.
  const desktop: CommandArguments<'wails', 'race.TargetRace'> = ['5', 'team', 'mine'];
  // @ts-expect-error A local MFL command cannot acquire a hosted workflow door.
  const remote: CommandArguments<'workflow', 'mfl.bid.revise'> = {};
  // @ts-expect-error MFL plan IDs stay text.
  const mfl: CommandArguments<'mfl', 'mfl.bid.withdraw'> = { id: 1 };
  return [upgrade, badLeague, badCommand, outcome, browser, desktop, remote, mfl];
}
void typeChecks;

describe('catalog-generated native envelope', () => {
  it('preserves command, version, mode and all supplied fields for the live hosted path', () => {
    expect(nativeEnvelope(read)).toEqual({ ...read, version: '1' });
    const args: NativeArguments<'commish.league.create'> = {
      league: '', command: 'commish.league.create',
      args: { name: 'Our league', preset: 'empty', franchiseCount: 12, season: 2027 },
      reason: 'We agreed to start', expected: 0, request: 'request.one',
      effective: 'now', visibility: 'league',
    };
    expect(nativeEnvelope({ command: 'commish.league.create', mode: 'execute', arguments: args }))
      .toEqual({ command: 'commish.league.create', mode: 'execute', version: '1', arguments: args });
    expect(nativeCommands['commish.league.create'].definition.Safeguards)
      .toBe('RawCommand.EffectiveDefinition+MissingReason');
  });

  it('rejects unsupported modes, unknown commands and unavailable doors', () => {
    for (const request of [
      { ...read, command: 'unknown' }, { ...read, mode: 'execute' },
      { ...read, command: 'commish.league.seed_held', mode: 'execute' },
    ]) expect(() => nativeEnvelope(request as NativeRequest)).toThrow('not available here');
  });

  it('rejects malformed, extra and missing arguments before a transport can run', () => {
    for (const args of [{}, { league: 3 }, { league: null }, { league: 'one', principal: 'admin' }])
      expect(() => nativeEnvelope({ ...read, arguments: args } as NativeRequest)).toThrow();
  });
});
