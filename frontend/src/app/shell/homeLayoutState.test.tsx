// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { defaultLayout, type Layout } from './homeLayout';
import { HQAct } from './HQAct';
import { useHomeLayout } from './homeLayoutState';
import type { SettingsStorage } from './franchise';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  vi.unstubAllGlobals();
});

function LayoutProbe({ port, storage }: { port: HostedPort; storage: SettingsStorage }) {
  const state = useHomeLayout(port, 'home', storage, true, () => {});
  return <><output data-testid="layout">{JSON.stringify(state.layout)}</output>
    <HQAct label="Save layout" onAction={() => state.save({ mode: 'custom',
      tiles: [['cap', '1x1']], over: [], pins: {} })}>Save</HQAct></>;
}

it('does not read or write hosted account layouts through browser storage', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const browserLayout: Layout = { mode: 'custom', tiles: [['wire', '1x1']], over: [], pins: {} };
  const storage: SettingsStorage = {
    getItem: vi.fn(() => JSON.stringify(browserLayout)), setItem: vi.fn(),
  };
  const port: HostedPort = vi.fn(async request => ({
    label: 'Account home layout', writable: true, football: '',
    data: { layout: defaultLayout(), revision: request.command === 'home.layout.set' ? 1 : 0 },
  }));
  await act(async () => root!.render(<LayoutProbe port={port} storage={storage} />));
  expect(storage.getItem).not.toHaveBeenCalled();
  expect(JSON.parse(host.querySelector('[data-testid="layout"]')!.textContent!)).toEqual(defaultLayout());

  await act(async () => {
    (host!.querySelector('button') as HTMLButtonElement).click();
    await Promise.resolve();
  });
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'home.layout.set' }));
  expect(storage.setItem).not.toHaveBeenCalled();

  const offlinePort: HostedPort = vi.fn(async () => { throw new Error('offline'); });
  await act(async () => root!.render(<LayoutProbe port={offlinePort} storage={storage} />));
  expect(storage.getItem).not.toHaveBeenCalled();
  expect(storage.setItem).not.toHaveBeenCalled();
  await act(async () => {
    (host!.querySelector('button') as HTMLButtonElement).click();
    await Promise.resolve();
  });
  expect(storage.setItem).not.toHaveBeenCalled();
  expect(JSON.parse(host.querySelector('[data-testid="layout"]')!.textContent!))
    .toMatchObject({ mode: 'custom', tiles: [['cap', '1x1']] });
});
