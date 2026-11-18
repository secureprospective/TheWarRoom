// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { situations } from '../control/room/guide';
import { HostedControlRoomSituations } from './HostedControlRoomSituations';
import { resolveSituationStep, hostedControlTabs } from './hostedControlRoomData';
import { hostedCommandAvailable } from '../commands/native';
import { actTable } from './actTable';

let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); });

it('follows every built guide route and leaves ruled or gap steps disabled', async () => {
  const onTab = vi.fn();
  const onTool = vi.fn();
  const onUnavailable = vi.fn();
  await act(async () => root.render(<HostedControlRoomSituations onTab={onTab}
    onTool={onTool} onUnavailable={onUnavailable} />));
  for (const [index, story] of situations.entries()) {
    const card = host.querySelector<HTMLButtonElement>(`[data-situation="${index}"]`);
    expect(card, story[1]).not.toBeNull();
    await act(async () => card!.click());
    for (const [stepIndex, [words, command, mark]] of story[2].entries()) {
      const route = resolveSituationStep(command, mark);
      expect(route.kind, `${story[1]} / ${command}`).not.toBe('unknown');
      if (route.kind === 'tab')
        expect(hostedControlTabs.some(([tab]) => tab === route.tab), command).toBe(true);
      if (route.kind === 'tool') {
        expect(actTable.includes(route.definition), command).toBe(true);
        expect(hostedCommandAvailable(route.definition.command), command).toBe(true);
      }
      if (route.kind === 'unavailable') expect(route.words, command).not.toBe('');
      const open = host.querySelector<HTMLButtonElement>(`[data-step="${stepIndex}"]`);
      expect(open, `${story[1]} / ${words}`).not.toBeNull();
      if (mark !== 'B') {
        expect(open!.disabled, command).toBe(true);
        continue;
      }
      const calls = onTab.mock.calls.length + onTool.mock.calls.length + onUnavailable.mock.calls.length;
      await act(async () => open!.click());
      const nextCalls = onTab.mock.calls.length + onTool.mock.calls.length + onUnavailable.mock.calls.length;
      expect(nextCalls, `${story[1]} / ${command}`).toBe(calls + 1);
      if (route.kind === 'tab') expect(onTab).toHaveBeenLastCalledWith(route.tab);
      else if (route.kind === 'tool') expect(onTool).toHaveBeenLastCalledWith(route.definition);
      else if (route.kind === 'unavailable') expect(onUnavailable).toHaveBeenLastCalledWith(route.words);
      else throw new Error(`Dead situation route: ${command}`);
    }
  }
});
