// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import { ProblemReport } from './ProblemReport';

let host: HTMLDivElement;
let root: Root;
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});

it('refuses an empty report, then sends the route and server release with member text', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
    nativeEnvelope(request);
    return { label: 'League server', writable: true, football: 'held', appRelease: 42, data: {} };
  });
  await act(async () => root.render(<ProblemReport league="native.test"
    route={{ node: 'control', workspace: 'league' }} onClose={() => {}} port={port} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });

  const send = () => [...host.querySelectorAll('button')]
    .find(button => button.textContent === 'Send report')!;
  await act(async () => send().click());
  expect(host.querySelector('[role="alert"]')?.textContent).toContain('Write a short description');
  expect(port).toHaveBeenCalledTimes(1);

  const textarea = host.querySelector('textarea')!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')!.set!.call(textarea,
      'The standings page is blank.');
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
  });
  await act(async () => send().click());
  expect(host.textContent).toContain('Sent — thanks');
  expect(port).toHaveBeenCalledTimes(2);
  expect(port.mock.calls[1][0]).toEqual({
    command: 'feedback.report.create', mode: 'execute', arguments: {
      league: 'native.test', text: 'The standings page is blank.', route: 'control/league',
      appRelease: 42, viewportWidth: window.innerWidth,
    },
  });
});
