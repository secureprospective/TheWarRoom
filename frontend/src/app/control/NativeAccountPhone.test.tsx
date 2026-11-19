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

it('lands on the new phone app after sign-in and changes placeholder tabs', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  goDescriptor = Object.getOwnPropertyDescriptor(window, 'go');
  Reflect.deleteProperty(window, 'go');
  await vi.resetModules();
  const { default: NativeAccount } = await import('./NativeAccount');
  const admission = { protocol: 1, admission: { league: 'native.test', name: 'Test League',
    franchise: 'franchise.01', franchiseName: 'My Team', role: 'Owner', seats: [] } };
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
  expect(host.querySelector('.phone-app__slot h1')?.textContent).toBe('Today');
  const live = [...host.querySelectorAll('.phone-app__tab')]
    .find(button => button.textContent?.includes('Live')) as HTMLButtonElement;
  await act(async () => live.click());
  expect(host.querySelector('.phone-app__slot h1')?.textContent).toBe('Live');
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
