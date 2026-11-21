// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ResearchPlayerInspector from './PlayerInspector';
import type { HostedPort } from '../commissioner/hostedRoom';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  root = undefined;
  host?.remove();
  host = undefined;
  delete (window as { go?: unknown }).go;
  vi.restoreAllMocks();
});

async function flush() {
  await act(async () => { await new Promise(r => setTimeout(r, 10)); });
}

describe('ResearchPlayerInspector', () => {
  it('reads hosted player.usage.read when a hosted league is provided without desktop bindings', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    const port = vi.fn(async () => ({
      label: 'L', writable: false, football: '',
      data: {
        snapShare: 0.81, targetShare: 0.24, carryShare: null, weeksUsed: 4,
        opponent: 'KC', allowedPoints: 14.2, allowedRank: 19, teams: 32,
        label: 'Average', reason: 'Last 4 weeks',
      },
    })) as unknown as HostedPort;
    host = document.createElement('div');
    document.body.append(host);
    root = createRoot(host);
    await act(async () => root!.render(
      <ResearchPlayerInspector id="16162" identity={false} league="L" port={port} position="WR" />,
    ));
    await flush();
    expect(host.textContent).toContain('Player news');
    expect(host.textContent).toContain('Not available');
    expect(host.textContent).toContain('KC');
    expect(host.textContent).toContain('24%');
    expect(host.querySelector('[role="alert"]')).toBeNull();
  });

  it('keeps desktop-required error when neither desktop bindings nor a hosted league exist', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    host = document.createElement('div');
    document.body.append(host);
    root = createRoot(host);
    await act(async () => root!.render(<ResearchPlayerInspector id="16162" />));
    await flush();
    expect(host.querySelector('[role="alert"]')?.textContent)
      .toContain('Player research requires the desktop app.');
  });
});
