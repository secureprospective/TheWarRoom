import { describe, expect, it, vi } from 'vitest';
import { createCommands } from './registry';
import { execute } from '../commissioner/proxy';

vi.mock('../commissioner/proxy', () => ({ execute: vi.fn() }));
const settle = () => new Promise(resolve => setTimeout(resolve, 30));

describe('Commissioner registry proxy', () => {
  it('dispatches to the lazy Go proxy and shows its landed receipt or plain refusal', async () => {
    const proxy = vi.mocked(execute);
    proxy.mockResolvedValue({
      entry: 'act.one', league: 'native.one', revision: 1, recordedAt: '2026-10-13T12:00:00Z',
      message: 'Recorded in the league ledger.',
    });
    const commands = createCommands();
    const body = '{"command":"commish.league.create","reason":"start our league"}';
    commands.dispatch('commissioner.execute', { body });
    await settle();
    expect(proxy).toHaveBeenCalledWith(body);
    expect(commands.read().notice).toBe('Recorded in the league ledger.');
    proxy.mockRejectedValue(new Error('only this league’s commissioner may make this change'));
    commands.dispatch('commissioner.execute', { body: '{}' });
    await settle();
    expect(commands.read().notice).toContain('only this league’s commissioner');
    expect(commands.read().notice).not.toContain('Recorded in');
  });
});
