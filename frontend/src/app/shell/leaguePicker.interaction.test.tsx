// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { TargetApp } from './TargetApp';
import { commands } from '../commands/registry';
import { LEAGUE_KEY } from './leaguePicker';

vi.mock('../data/provider', async (original) => {
  const actual = await original<typeof import('../data/provider')>();
  const fixture = new actual.FixtureProvider();
  return { ...actual, selectProvider: () => fixture };
});
vi.mock('../data/api', async (original) => ({
  ...await original<typeof import('../data/api')>(),
  targetLeagues: async () => [
    { id: 'mirror', name: 'Reference', teams: 32, app: false },
    { id: 'native.one', name: 'App league', teams: 12, app: true },
  ],
}));
vi.mock('./TradeDesk', () => ({ default: () => <p>Mirror trade desk</p> }));

afterEach(() => { localStorage.clear(); vi.unstubAllGlobals(); });

it('switches the rendered route and stores the selected league', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
  localStorage.setItem(LEAGUE_KEY, 'mirror');
  commands.dispatch('nav.open', { node: 'trade' });
  const container = document.createElement('div');
  document.body.append(container);
  const root = createRoot(container);
  try {
    await act(async () => { root.render(<TargetApp />); });
    for (let i = 0; i < 30 && !container.textContent?.includes('Mirror trade desk'); i++)
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
    expect(container.textContent).toContain('Mirror trade desk');
    const switcher = [...container.querySelectorAll('button')].find(b => b.textContent === 'Switch league');
    expect(switcher).toBeDefined();
    await act(async () => { switcher!.click(); });
    const app = [...container.querySelectorAll('button')].find(b => b.textContent?.includes('App league'));
    expect(app).toBeDefined();
    await act(async () => { app!.click(); });
    expect(container.textContent).not.toContain('Mirror trade desk');
    expect(container.textContent).toContain('Not wired');
    expect(localStorage.getItem(LEAGUE_KEY)).toBe('native.one');
  } finally {
    await act(async () => { root.unmount(); });
    container.remove();
    delete (window as unknown as { go?: unknown }).go;
  }
});

it('keeps app HQ and War Room failures inside the navigable shell', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
  localStorage.setItem(LEAGUE_KEY, 'native.one');
  commands.dispatch('nav.open', { node: 'hq' });
  const container = document.createElement('div');
  document.body.append(container);
  const root = createRoot(container);
  try {
    await act(async () => { root.render(<TargetApp />); });
    expect(container.querySelector('.rail nav[aria-label="Workspaces"]')).not.toBeNull();
    expect(container.querySelector('.rail .me')).not.toBeNull();
    expect(container.querySelector('.ws')).not.toBeNull();
    await act(async () => { commands.dispatch('nav.open', { node: 'war' }); });
    expect(container.querySelector('.rail nav[aria-label="Workspaces"]')).not.toBeNull();
    const home = [...container.querySelectorAll('.rail nav button')].find(b => b.textContent === 'Home');
    expect(home).toBeDefined();
    await act(async () => { (home as HTMLButtonElement).click(); });
    expect(container.querySelector('.ws')?.getAttribute('aria-label')).toBe('Home');
  } finally {
    await act(async () => { root.unmount(); });
    container.remove();
    delete (window as unknown as { go?: unknown }).go;
  }
});

it('opens the Control Room on the locked mock for a picked app league', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
  localStorage.setItem(LEAGUE_KEY, 'native.one');
  commands.dispatch('nav.open', { node: 'control', workspace: 'app' });
  const container = document.createElement('div');
  const root = createRoot(container);
  try {
    await act(async () => { root.render(<TargetApp />); });
    for (let i = 0; i < 50 && !container.querySelector('[data-testid="control-room"]'); i++)
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
    expect(container.querySelector('[data-testid="control-room"]')).not.toBeNull();
    expect(container.querySelector('.shell-unwired')).toBeNull();
    expect([...container.querySelectorAll('[data-testid="control-room"] [role="tab"]')].map(b =>
      b.textContent))
      .toEqual(['JOBS', 'SITUATIONS', 'RULES', 'SEASON DESK', 'FIX', 'PICKS', 'PEOPLE', 'LEAGUE', 'SERVER']);
    expect(container.querySelectorAll('.wtabs button')).toHaveLength(0);
  } finally {
    await act(async () => { root.unmount(); });
    delete (window as unknown as { go?: unknown }).go;
  }
});
