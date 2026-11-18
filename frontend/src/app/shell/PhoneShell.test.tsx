// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import { ThinLeagueShell } from './ThinLeagueShell';
import { phoneTabEntry, type PhoneTab } from './PhoneShell';
import type { ThinScreenRoute } from './ThinPlayerSearch';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  Reflect.deleteProperty(window, 'matchMedia');
  Reflect.deleteProperty(window, 'innerWidth');
});
function setViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
  Object.defineProperty(window, 'matchMedia', { configurable: true,
    value: vi.fn((media: string) => ({ media, matches: width < 700,
      addEventListener: vi.fn(), removeEventListener: vi.fn() })) });
}
function mountShell(props: { tab?: PhoneTab; onTab?: (tab: PhoneTab) => void; control?: () => void;
  leagueName?: string; phoneLeagueName?: string } = {}) {
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const route: ThinScreenRoute = { node: 'pulse', workspace: 'now' };
  return act(async () => root!.render(<ThinLeagueShell leagueName={props.leagueName ?? 'Test league'}
    phoneLeagueName={props.phoneLeagueName} route={route}
    commissioner players={[]} navigate={() => {}} openPlayer={() => {}} phoneShell
    phoneTab={props.tab ?? 'today'} onPhoneTab={props.onTab} onPhoneControlRoom={props.control}>
    <p>League screen</p>
  </ThinLeagueShell>));
}

it('shows five routed tabs, the admitted league name and commissioner access on phone', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setViewport(390);
  const onTab = vi.fn(); const control = vi.fn();
  await mountShell({ onTab, control, leagueName: 'Franchise name', phoneLeagueName: 'Test league' });
  const tabs = [...host!.querySelectorAll('.phone-shell__tabs button')];
  expect(tabs.map(tab => tab.textContent)).toEqual(['Today', 'Live', 'Team', 'Moves', 'League']);
  expect(host!.querySelector('.phone-shell__league-name')?.textContent).toBe('Test league');
  expect(host!.querySelector('.rail .ph')?.textContent).toBe('Franchise name · hosted');
  expect(host!.querySelector('.phone-shell__appbar select')).toBeNull();
  for (const tab of tabs) await act(async () => (tab as HTMLButtonElement).click());
  expect(onTab.mock.calls.map(([tab]) => tab)).toEqual(['today', 'live', 'team', 'moves', 'league']);
  expect((['today', 'live', 'team', 'moves', 'league'] as const).map(phoneTabEntry))
    .toEqual(['home', 'pulse', 'hq', 'war', 'league']);

  await act(async () => root!.render(<ThinLeagueShell leagueName="Test league"
    route={{ node: 'control', workspace: 'league' }} commissioner players={[]}
    navigate={() => {}} openPlayer={() => {}} phoneShell phoneTab="league"
    onPhoneTab={onTab} onPhoneControlRoom={control}><p>Control Room</p></ThinLeagueShell>));
  const room = host!.querySelector('.phone-shell__control') as HTMLButtonElement;
  expect(room?.textContent).toBe('Control Room');
  await act(async () => room.click());
  expect(control).toHaveBeenCalledOnce();
});

it('keeps the desktop rail and omits phone tabs at 1280px', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setViewport(1280);
  await mountShell();
  expect(host!.querySelector('.rail.mock nav[aria-label="Workspaces"]')).not.toBeNull();
  expect(host!.querySelector('.phone-shell__tabs')).toBeNull();
  expect(host!.textContent).toContain('League screen');
});
