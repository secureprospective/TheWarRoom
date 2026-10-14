import { act } from 'react';
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { commands } from '../commands/registry';
import { ChangeBar } from './ChangeBar';
import { all, disabled, flush, holder, mount, render, unmount, type Element } from './commissionerDom';
import type { Change } from './forms';
import { leagueState } from './proxy';
import type { Safeguard } from './rules';
import wire from './commands.wire.testdata.json';

beforeEach(() => { leagueState.forget(); vi.useFakeTimers(); });
afterEach(async () => { await unmount(); vi.useRealTimers(); vi.restoreAllMocks(); });

const event = async (node: Element, name: string, args: object = {}) => {
  const handler = node.props()[name] as ((event: object) => void) | undefined;
  await act(async () => { handler?.({ preventDefault() {}, ...args }); });
};
const wait = async (ms: number) => { await act(async () => { vi.advanceTimersByTime(ms); }); };

// Every supported command comes from Go, including commands whose form is not built yet.
describe.each(wire.filter(row => row.safeguard !== 'confirm'))(
  '$command $effective $safeguard', row => {
    it('cannot record by a click or a single Enter or Space; requires Go’s safeguard', async () => {
      const sent = vi.spyOn(commands, 'dispatch').mockImplementation(() => undefined);
      const change: Change = { league: 'native.proof', command: row.command, expected: 1,
        args: { field: row.field }, reason: '', effective: row.effective as Change['effective'],
        visibility: 'league' };
      const safeguard = row.safeguard as Safeguard;
      const host = await mount(render(ChangeBar, { form: 'proof', change, safeguard,
        reason: '', onReason() {}, confirmLabel: 'Record this change' }));
      const confirm = all(host, 'button')[0];
      if (safeguard === 'reason') {
        expect(disabled(confirm)).toBe(true);
        expect(host.textContent).toContain('Type a reason');
      } else {
        expect(disabled(confirm)).toBe(false);
        expect(confirm.props().onClick).toBeUndefined();
        // Native buttons synthesize click for Enter and Space. A hold Act must have no click handler.
        await event(confirm, 'onClick');
        for (const key of ['Enter', ' ']) {
          await event(holder(host, 'Record this change'), 'onKeyDown', { key });
          await wait(1);
          await event(holder(host, 'Record this change'), 'onKeyUp', { key });
          await event(confirm, 'onClick');
          await wait(1000);
        }
        expect(sent).not.toHaveBeenCalled();
        await event(holder(host, 'Record this change'), 'onKeyDown', { key: 'Enter' });
        await wait(699);
        expect(sent).not.toHaveBeenCalled();
        await wait(1);
        await flush();
        expect(sent).toHaveBeenCalledTimes(1);
        expect(sent).toHaveBeenCalledWith('commissioner.execute',
          { body: expect.stringContaining(`"command":"${row.command}"`) });
      }
    });
  },
);

function executeControls(dir: string, root = dir): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const file = join(dir, entry.name);
    if (entry.isDirectory()) return executeControls(file, root);
    if (!file.endsWith('.tsx') || file.includes('.test.')) return [];
    return /(?:verb=|dispatch\()["']commissioner\.execute["']/.test(readFileSync(file, 'utf8'))
      ? [relative(root, file)] : [];
  });
}
it('has no other commissioner execute control that could bypass the shared safeguard', () => {
  expect(executeControls(resolve('src/app'))).toEqual([
    'commissioner/ChangeBar.tsx', 'commissioner/HoldToFire.tsx',
  ]);
});
