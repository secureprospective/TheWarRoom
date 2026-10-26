import { afterEach, expect, it, vi } from 'vitest';
import { all, button, click, disabled, mount, render, settle, type, unmount }
  from '../commissioner/commissionerDom';
import { commands, createCommands } from '../commands/registry';
import { Parameters } from './DataSources';
import { dataState } from './state';
import { parseSources } from './provider';
import type { ParamsResult } from '../control/api';
import wire from './wire.testdata.json';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

it.each([['admin', true], ['gm', false], ['commish', false]] as const)(
  '%s uses the dispatch role check for parameter inputs and Apply', async (role, allowed) => {
    const caller = createCommands(undefined, undefined, undefined, [role]);
    vi.spyOn(commands, 'canDispatch').mockImplementation(caller.canDispatch);
    const run = vi.spyOn(dataState, 'run').mockResolvedValue(undefined);
    const words = parseSources(wire.data).words;
    const host = await mount(render(Parameters, { params: wire.params as ParamsResult,
      busy: '', words }));
    const input = all(host, 'input').find(e => e.getAttribute('type') === 'number')!;
    expect(disabled(input)).toBe(!allowed);
    expect(disabled(button(host, words.apply))).toBe(true);
    if (allowed) {
      await type(input, '0.7');
      expect(disabled(button(host, words.apply))).toBe(false);
      await click(button(host, words.apply));
      await settle();
      expect(run).toHaveBeenCalledExactlyOnceWith({ kind: 'parameter', key: 'test.weight',
        position: 'WR', value: 0.7 });
      expect(commands.history().at(-1)).toMatchObject({ id: 'data.parameter', status: 'done' });
    } else {
      expect(host.textContent).toContain(words.permission);
      expect(() => caller.dispatch('data.parameter', { kind: 'parameter', key: 'test.weight',
        position: 'WR', value: 0.7 })).toThrow('permission denied');
      expect(run).not.toHaveBeenCalled();
    }
  });
