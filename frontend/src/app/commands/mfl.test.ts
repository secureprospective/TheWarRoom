import { describe, expect, it, vi } from 'vitest';
import { createMFLClient, mflEnvelope, type MFLContext, type MFLRequest } from './mfl';
import { mflCommands } from './mfl.gen';

const desktop: MFLContext = { availability: 'desktop', role: 'gm' };
const read: MFLRequest = { command: 'mfl.bid.plans', mode: 'read', arguments: { franchiseId: '0001' } };

function typeChecks() {
  // @ts-expect-error Reading plans cannot execute.
  const upgrade: MFLRequest = { ...read, mode: 'execute' };
  // @ts-expect-error No hosted credential, principal or MFL key belongs in plan arguments.
  const key: MFLRequest = { ...read, arguments: { franchiseId: '0001', apiKey: 'secret' } };
  return [upgrade, key];
}
void typeChecks;

describe('local MFL reference command transport', () => {
  it('dispatches both renamed plans and refuses the old names', async () => {
    const port = vi.fn().mockResolvedValue({ state: 'ready' });
    const client = createMFLClient(port);
    const requests: MFLRequest[] = [
      { command: 'mfl.lineup.set', mode: 'execute', arguments: { franchiseId: '0025', starters: ['0042'] } },
      { command: 'mfl.trade.accept', mode: 'execute', arguments: { franchiseId: '0025', tradeId: '77' } },
    ];
    for (const request of requests) {
      await client.invoke(request, { ...desktop, ownerAction: true });
      expect(port).toHaveBeenLastCalledWith({ ...request, version: '1' });
      expect(() => client.invoke({ ...request, command: request.command.slice(4) } as unknown as MFLRequest,
        { ...desktop, ownerAction: true })).toThrow();
    }
    expect(port).toHaveBeenCalledTimes(2);
  });
  it('retains the catalog version, roles and read-only request', async () => {
    const port = vi.fn().mockResolvedValue({ plans: [] });
    const client = createMFLClient(port);
    for (const role of ['gm', 'commish', 'admin'] as const) {
      await client.invoke(read, { ...desktop, role });
    }
    expect(port).toHaveBeenCalledTimes(3);
    expect(port).toHaveBeenLastCalledWith({ ...read, version: '1' });
    expect(Object.keys(mflCommands)).toHaveLength(11);
  });

  it('fails closed for every hosted route, including execute and MFL keys', () => {
    const port = vi.fn();
    const client = createMFLClient(port);
    for (const definition of Object.values(mflCommands)) {
      expect(definition.definition.Availability).toBe('desktop');
      expect(() => client.invoke({ command: definition.definition.ID, mode: definition.definition.Modes[0],
        arguments: {} } as MFLRequest, { ...desktop, availability: 'server', ownerAction: true }))
        .toThrow('stay on your computer');
    }
    for (const args of [{ franchiseId: '0001', apiKey: 'secret' }, {}, { franchiseId: 1 }]) {
      expect(() => client.invoke({ ...read, arguments: args } as MFLRequest, desktop)).toThrow();
    }
    expect(port).not.toHaveBeenCalled();
  });

  it('does not infer owner confirmation from preview, role or gravity', async () => {
    const port = vi.fn().mockResolvedValue({ state: 'Handed off' });
    const client = createMFLClient(port);
    for (const command of ['mfl.bid.handoff', 'mfl.bid.posted', 'mfl.bid.withdraw'] as const) {
      const request: MFLRequest = { command, mode: 'execute', arguments: { id: 'saved-plan' } };
      expect(() => client.invoke(request, { ...desktop, role: 'admin' })).toThrow('Nothing has changed');
      expect(port).not.toHaveBeenCalled();
      await client.invoke(request, { ...desktop, ownerAction: true });
      expect(port).toHaveBeenCalledExactlyOnceWith({ ...request, version: '1' });
      port.mockClear();
    }
  });

  it('rejects unknown commands, roles, mode upgrades and server-owned arguments', () => {
    expect(() => mflEnvelope({ ...read, command: 'bids.place' } as unknown as MFLRequest, desktop)).toThrow();
    expect(() => mflEnvelope(read, { ...desktop, role: 'viewer' } as unknown as MFLContext))
      .toThrow('not available');
    expect(() => mflEnvelope({ ...read, mode: 'execute' } as unknown as MFLRequest, desktop)).toThrow();
    const forged = { ...read, arguments: { franchiseId: '0001', principal: 'admin' } } as MFLRequest;
    expect(() => mflEnvelope(forged, desktop)).toThrow();
  });
});
