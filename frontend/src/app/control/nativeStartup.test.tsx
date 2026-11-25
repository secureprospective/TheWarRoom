// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { NativeStartup } from './NativeStartup';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
async function mount() {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
  await act(async () => root?.render(<NativeStartup />));
  return host;
}
afterEach(async () => {
  await act(async () => root?.unmount());
  root = undefined;
  host?.remove();
  vi.unstubAllGlobals();
});

it('offers sign-in when the cookie session is absent', async () => {
  const fetcher = vi.fn().mockResolvedValue(new Response(null, { status: 401 }));
  vi.stubGlobal('fetch', fetcher);
  const page = await mount();
  await vi.waitFor(() => expect(page.textContent).toContain('No MFL account or key needed'),
    { timeout: 5000 });
  expect(page.textContent).not.toContain('Join with a code');
  expect(fetcher.mock.calls.map(call => call[0])).toEqual(['/app/session']);
});

it.each([400, 404, 500, 503])('shows a startup failure for HTTP %s', async status => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status })));
  const page = await mount();
  await vi.waitFor(() => expect(page.textContent).toContain('Your league server could not be reached'),
    { timeout: 5000 });
});
