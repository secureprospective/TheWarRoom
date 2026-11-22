// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import type { HostedPort } from './hostedRoom';
import { HostedControlRoomFeedback } from './HostedControlRoomFeedback';

let host: HTMLDivElement;
let root: Root;
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});

it('reads and displays reports in server-provided newest-first order', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
    nativeEnvelope(request);
    return { label: 'League server', writable: false, football: 'held', data: {
      league: 'native.test', reports: [
        { id: `feedback.${'b'.repeat(32)}`, league: 'native.test', principal: 'acct.owner',
          text: 'Newest report', route: 'control/league', appRelease: 42, viewportWidth: 1440,
          userAgent: 'Test browser', createdAt: '2026-11-17T12:00:00Z' },
        { id: `feedback.${'a'.repeat(32)}`, league: 'native.test', principal: 'acct.owner',
          text: 'Older report', route: 'home/overview', appRelease: 41, viewportWidth: 1280,
          userAgent: 'Test browser', createdAt: '2026-11-16T12:00:00Z' },
      ],
    } };
  });
  await act(async () => root.render(<HostedControlRoomFeedback league="native.test" port={port} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(port).toHaveBeenCalledWith({ command: 'commish.feedback.read', mode: 'read',
    arguments: { league: 'native.test' } });
  expect([...host.querySelectorAll('.hosted-feedback-text')].map(item => item.textContent))
    .toEqual(['Newest report', 'Older report']);
});
