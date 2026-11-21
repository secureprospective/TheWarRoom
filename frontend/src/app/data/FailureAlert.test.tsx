// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import { AppErrorBoundary } from './AppErrorBoundary';
import { FailureAlert } from './FailureAlert';
import {
  recordAppRelease,
  registerLeagueProvider,
  resetFailureStateForTest,
  setFailureScreen,
} from './errorCodes';

let host: HTMLDivElement;
let root: Root;

function setupDom(): void {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
}

afterEach(async () => {
  if (root) await act(async () => root.unmount());
  host?.remove();
  resetFailureStateForTest();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('FailureAlert', () => {
  it('shows plain message, code, copies Central details with 2s Copied state, and sends report', async () => {
    vi.useFakeTimers();
    setupDom();
    const writeText = vi.fn(async () => {});
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText }, configurable: true,
    });
    setFailureScreen('League', 'phone/league');
    recordAppRelease(42);
    registerLeagueProvider(() => ({ league: 'native.hosted', name: 'Test League' }));

    const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
      nativeEnvelope(request);
      return {
        label: 'League server', writable: true, football: 'held', appRelease: 42,
        data: { message: 'Sent — thanks' },
      };
    });

    await act(async () => root.render(
      <FailureAlert
        message="The league could not accept this action."
        code="TWR-7K9M-2P4X"
        port={port}
        now={() => new Date('2026-11-20T20:15:30Z')}
      />,
    ));

    expect(host.querySelector('[role="alert"]')?.textContent)
      .toBe('The league could not accept this action.');
    expect(host.querySelector('[data-testid="failure-code"]')?.textContent).toBe('TWR-7K9M-2P4X');

    const findBtn = (label: string) =>
      [...host.querySelectorAll('button')].find(b => b.textContent === label);

    await act(async () => { findBtn('Copy')!.click(); });
    expect(writeText).toHaveBeenCalledTimes(1);
    const copiedText = (writeText.mock.calls as unknown as string[][])[0][0];
    expect(copiedText).toContain('Error code: TWR-7K9M-2P4X');
    expect(copiedText).toContain('2:15:30 PM C');
    expect(copiedText).toContain('Screen: League (phone/league)');
    expect(copiedText).toContain('App version: 0.5.0 (release 42)');
    expect(copiedText).toContain('League: Test League (native.hosted)');
    expect(findBtn('Copied')).toBeTruthy();

    await act(async () => { vi.advanceTimersByTime(1999); });
    expect(findBtn('Copied')).toBeTruthy();
    await act(async () => { vi.advanceTimersByTime(1); });
    expect(findBtn('Copy')).toBeTruthy();

    await act(async () => { findBtn('Send to commissioner')!.click(); });
    expect(port).toHaveBeenCalledTimes(1);
    const sent = port.mock.calls[0][0];
    expect(sent.command).toBe('feedback.report.create');
    expect(sent.mode).toBe('execute');
    const args = sent.arguments as Record<string, unknown>;
    expect(args.league).toBe('native.hosted');
    expect(args.route).toBe('phone/league');
    expect(args.appRelease).toBe(42);
    expect(String(args.text)).toContain('TWR-7K9M-2P4X');
    expect(String(args.text)).toContain('League (phone/league)');
    expect(findBtn('Sent — thanks')).toBeTruthy();
  });

  it('catches screen crashes in AppErrorBoundary with a client TWR-C code', async () => {
    setupDom();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    function Boom(): JSX.Element {
      throw new Error('render boom');
    }
    await act(async () => root.render(
      <AppErrorBoundary><Boom /></AppErrorBoundary>,
    ));
    expect(host.querySelector('[role="alert"]')?.textContent)
      .toContain('Something went wrong');
    expect(host.querySelector('[data-testid="failure-code"]')?.textContent)
      .toMatch(/^TWR-C[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{3}-/);
  });
});
