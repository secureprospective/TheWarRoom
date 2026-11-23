// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import type { NativeView } from './nativeApi';

const initial: NativeView = { server: 'https://league.example', league: '', message: '', name: '',
  role: '', franchise: '', franchiseName: '', seed: '', connection: '', helperId: '',
  confirmation: '', authGeneration: '0', seats: [] };
let root: Root | undefined;
let host: HTMLDivElement | undefined;
let goDescriptor: PropertyDescriptor | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  Reflect.deleteProperty(window, 'matchMedia');
  Reflect.deleteProperty(window, 'innerWidth');
  if (goDescriptor) Object.defineProperty(window, 'go', goDescriptor);
  else Reflect.deleteProperty(window, 'go');
  goDescriptor = undefined;
  vi.unstubAllGlobals(); vi.restoreAllMocks();
});
function setPhoneViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
  Object.defineProperty(window, 'matchMedia', { configurable: true, value: vi.fn((media: string) => ({
    media, matches: width < 700, addEventListener: vi.fn(), removeEventListener: vi.fn(),
  })) });
}
function setValue(input: HTMLInputElement, value: string) {
  const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
  setter?.call(input, value);
  input.dispatchEvent(new Event('input', { bubbles: true }));
  input.dispatchEvent(new Event('change', { bubbles: true }));
}

it('lands on the new phone app after sign-in and shows the ported Today tab', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  goDescriptor = Object.getOwnPropertyDescriptor(window, 'go');
  Reflect.deleteProperty(window, 'go');
  await vi.resetModules();
  const { default: NativeAccount } = await import('./NativeAccount');
  const admission = { protocol: 1, admission: { league: 'native.test', name: 'Test League',
    franchise: 'franchise.01', franchiseName: 'My Team', role: 'Owner', seats: [] },
    leagues: [{ id: 'native.test', name: 'Test League', role: 'Owner', franchise: 'franchise.01',
      franchiseName: 'My Team' }] };
  const fetcher = vi.fn(async (url: string, init?: RequestInit) => {
    if (url === '/app/session' && init?.method === 'POST')
      return new Response(JSON.stringify(admission), { status: 200,
        headers: { 'Content-Type': 'application/json' } });
    if (url === '/app/session' && init?.method === 'DELETE')
      return new Response(null, { status: 204 });
    if (url === '/app/session')
      return new Response(JSON.stringify(admission), { status: 200,
        headers: { 'Content-Type': 'application/json' } });
    return new Response('not found', { status: 404 });
  });
  vi.stubGlobal('fetch', fetcher);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<NativeAccount initial={initial} />));
  const username = host.querySelector('input[autocomplete="username"]') as HTMLInputElement;
  const password = host.querySelector('input[autocomplete="current-password"]') as HTMLInputElement;
  setValue(username, 'owner'); setValue(password, 'private-password');
  const form = password.form!;
  await act(async () => form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })));
  await act(async () => {
    for (let attempt = 0; attempt < 80 && !host!.querySelector('.phone-app'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });

  expect(host.querySelector('.phone-app')).not.toBeNull();
  expect(host.querySelector('.phone-app__league')?.textContent).toContain('Test League');
  // The frame owns the phone screen: nothing from the account page's own chrome may render.
  expect(host.querySelector('.native-account > header')).toBeNull();
  expect(host.querySelector('nav[aria-label="Account access"]')).toBeNull();
  expect(host.querySelector('.native-account > .native-actions')).toBeNull();
  expect(host.querySelector('.native-account > footer')).toBeNull();
  expect([...host.querySelectorAll('button')].map(button => button.textContent))
    .not.toContain('Open my franchise');
  expect(host.querySelector('.phone-app__slot')?.getAttribute('aria-label')).toBe('Today');
  expect(host.querySelector('.phone-app__slot h1')).toBeNull();
  const live = [...host.querySelectorAll('.phone-app__tab')]
    .find(button => button.textContent?.includes('Live')) as HTMLButtonElement;
  await act(async () => live.click());
  expect(host.querySelector('.phone-app__slot')?.getAttribute('aria-label')).toBe('Live');
  expect(host.querySelector('.phone-app__slot h1')).toBeNull();
  expect(host.querySelector('.phone-app__bar')?.textContent).not.toContain('Mock');
  await act(async () => host!.querySelector<HTMLButtonElement>('.phone-app__league')!.click());
  expect(host.querySelector('.phone-app__account-name')?.textContent).toContain('owner');
  await act(async () => [...host!.querySelectorAll<HTMLButtonElement>('.phone-app__account button')]
    .find(button => button.textContent === 'Authenticator')!.click());
  expect(host.querySelector('.native-account > form > h2')?.textContent).toBe('Authenticator (optional)');
  const back = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('Back to league'))!;
  await act(async () => back.click());
  expect(host.querySelector('.phone-app')).not.toBeNull();
  await act(async () => host!.querySelector<HTMLButtonElement>('.phone-app__league')!.click());
  await act(async () => [...host!.querySelectorAll<HTMLButtonElement>('.phone-app__account button')]
    .find(button => button.textContent === 'Sign out')!.click());
  await act(async () => {
    for (let attempt = 0; attempt < 80 && host!.querySelector('.phone-app'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });
  expect(fetcher.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(true);
  expect(host.querySelector('.phone-app')).toBeNull();
});

it('says No team of yours in the Team tab when the account holds no franchise', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  goDescriptor = Object.getOwnPropertyDescriptor(window, 'go');
  Reflect.deleteProperty(window, 'go');
  await vi.resetModules();
  const { default: NativeAccount } = await import('./NativeAccount');
  const admission = { protocol: 1, admission: { league: 'native.test', name: 'Test League',
    franchise: '', franchiseName: '', role: 'Commissioner', seats: [] },
    leagues: [{ id: 'native.test', name: 'Test League', role: 'Commissioner', franchise: '',
      franchiseName: '' }] };
  const fetcher = vi.fn(async (url: string, init?: RequestInit) => {
    if (url === '/app/session' && init?.method === 'POST')
      return new Response(JSON.stringify(admission), { status: 200,
        headers: { 'Content-Type': 'application/json' } });
    if (url === '/app/session' && init?.method === 'DELETE')
      return new Response(null, { status: 204 });
    if (url === '/app/session')
      return new Response(JSON.stringify(admission), { status: 200,
        headers: { 'Content-Type': 'application/json' } });
    return new Response('not found', { status: 404 });
  });
  vi.stubGlobal('fetch', fetcher);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<NativeAccount initial={initial} />));
  const username = host.querySelector('input[autocomplete="username"]') as HTMLInputElement;
  const password = host.querySelector('input[autocomplete="current-password"]') as HTMLInputElement;
  setValue(username, 'owner'); setValue(password, 'private-password');
  const form = password.form!;
  await act(async () => form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })));
  await act(async () => {
    for (let attempt = 0; attempt < 80 && !host!.querySelector('.phone-app'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });

  expect(host.querySelector('.phone-app')).not.toBeNull();
  const team = [...host.querySelectorAll('.phone-app__tab')]
    .find(button => button.textContent?.includes('Team')) as HTMLButtonElement;
  await act(async () => team.click());
  expect(host.querySelector('.phone-app__slot')?.textContent)
    .toContain('No team of yours in this league');
});


it('shows Live for the league the bar has selected, not the first league', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  goDescriptor = Object.getOwnPropertyDescriptor(window, 'go');
  Reflect.deleteProperty(window, 'go');
  await vi.resetModules();
  vi.doMock('../phone/LiveTab', () => ({ default: ({ league }: { league?: { id: string } }) =>
    <p className="live-league">{league?.id ?? 'none'}</p> }));
  const { default: NativeAccount } = await import('./NativeAccount');
  const admission = { protocol: 1, admission: { league: 'native.second', name: 'Second League',
    franchise: 'franchise.02', franchiseName: 'My Team', role: 'Owner', seats: [] },
    leagues: [{ id: 'native.first', name: 'First League', role: 'Owner', franchise: 'franchise.01',
      franchiseName: 'Other Team' }, { id: 'native.second', name: 'Second League', role: 'Owner',
      franchise: 'franchise.02', franchiseName: 'My Team' }] };
  vi.stubGlobal('fetch', vi.fn(async (url: string) => url === '/app/session'
    ? new Response(JSON.stringify(admission), { status: 200,
      headers: { 'Content-Type': 'application/json' } })
    : new Response('not found', { status: 404 })));
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<NativeAccount initial={initial} />));
  const username = host.querySelector('input[autocomplete="username"]') as HTMLInputElement;
  const password = host.querySelector('input[autocomplete="current-password"]') as HTMLInputElement;
  setValue(username, 'owner'); setValue(password, 'private-password');
  await act(async () => password.form!.dispatchEvent(
    new Event('submit', { bubbles: true, cancelable: true })));
  await act(async () => {
    for (let attempt = 0; attempt < 80 && !host!.querySelector('.phone-app'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });
  const live = [...host.querySelectorAll('.phone-app__tab')]
    .find(button => button.textContent?.includes('Live')) as HTMLButtonElement;
  await act(async () => live.click());
  await act(async () => {
    for (let attempt = 0; attempt < 80 && !host!.querySelector('.live-league'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });
  expect(host.querySelector('.live-league')?.textContent).toBe('native.second');
  vi.doUnmock('../phone/LiveTab');
});

it('signed-out phone sign-in shows one Sign in button and no cookie jargon', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  goDescriptor = Object.getOwnPropertyDescriptor(window, 'go');
  Reflect.deleteProperty(window, 'go');
  await vi.resetModules();
  const { default: NativeAccount } = await import('./NativeAccount');
  vi.stubGlobal('fetch', vi.fn(async () => new Response('unauthorized', { status: 401 })));
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<NativeAccount initial={initial} />));
  const signIns = [...host.querySelectorAll('button')].filter(b => b.textContent?.trim() === 'Sign in');
  expect(signIns).toHaveLength(1);
  expect(host.textContent).not.toMatch(/HttpOnly|cookie/i);
  expect(host.textContent).toContain('You stay signed in on this device.');
});
