import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { copyHelperConnection } from './api';
import { all, button, click, control, mount, render, settle, type, unmount }
  from '../commissioner/commissionerDom';
import NativeHelpers from './NativeHelpers';
import type { NativeView } from './nativeApi';

vi.mock('./api', () => ({ copyHelperConnection: vi.fn() }));

const view: NativeView = { server: 'https://league.example', league: 'native.test', message: '',
  name: '', role: '', franchise: '', franchiseName: '', seed: '', connection: '', helperId: '',
  confirmation: '', seats: [] };
const reply = (value: Partial<NativeView>) => ({ ok: true,
  headers: new Headers({ 'Content-Type': 'application/json' }), json: async () => ({ ...view, ...value }) });
const id = 'A'.repeat(43);
const props = { server: view.server, league: view.league, franchise: 'franchise.02',
  seats: [{ franchise: 'franchise.01', name: 'Club one' }, { franchise: 'franchise.02', name: 'Club two' }] };
afterEach(async () => { await unmount(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });

describe('native AI helpers', () => {
  it('connects a helper for the chosen franchise and shows its line once', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply({ message: 'Copy this connection line now.',
      connection: 'twr1.' + id + '.' + 'Q'.repeat(43), helperId: id }));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeHelpers, props));
    await click(button(host, 'Connect a helper')); await settle();
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({
      action: 'helper-issue', helper: 'suggest', franchise: 'franchise.02',
    });
    expect(host.textContent).toContain('twr connect --address https://league.example --league native.test');
    expect(host.textContent).toContain('Connection line (shown once):');
    const line = all(host, 'code').find(code => code.textContent.startsWith('twr1.'));
    expect(line?.getAttribute('class')).toBe('native-seed');
    const css = readFileSync('src/app/control/nativeAccount.css', 'utf8');
    expect(css).toMatch(/\.native-account \.native-seed\s*\{\s*overflow-wrap: anywhere;/);
    expect(css).toMatch(/\.native-account code\.native-seed\s*\{[^}]*white-space: pre-wrap;/);
    expect(css).toMatch(/\.native-account code\.native-seed\s*\{[^}]*display: block;/);
    vi.mocked(copyHelperConnection).mockResolvedValue(true);
    await click(button(host, 'Copy connection line')); await settle();
    expect(copyHelperConnection).toHaveBeenCalledWith('twr1.' + id + '.' + 'Q'.repeat(43));
    expect(host.textContent).toContain('Connection line copied.');
    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(control(host, 'Helper ID').value).toBe(id);
    fetcher.mockResolvedValue(reply({ message: 'AI helper disconnected.' }));
    await click(button(host, 'Disconnect helper')); await settle();
    expect(JSON.parse(fetcher.mock.calls[1][1].body)).toEqual({ action: 'helper-revoke', code: id });
    expect(host.textContent).not.toContain('twr1.' + id);
  });

  it.each(['unavailable', 'rejected'])('keeps the line for manual copy when clipboard is %s', async kind => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(reply({ connection: 'twr1.' + id })));
    if (kind === 'unavailable') vi.mocked(copyHelperConnection).mockResolvedValue(false);
    else vi.mocked(copyHelperConnection).mockRejectedValue(new Error('Clipboard denied'));
    const host = await mount(render(NativeHelpers, props));
    await click(button(host, 'Connect a helper')); await settle();
    await click(button(host, 'Copy connection line')); await settle();
    expect(host.textContent).toContain('Could not copy. Select the connection line and copy it yourself.');
    expect(host.textContent).toContain('twr1.' + id);
    expect(host.textContent).not.toContain('Connection line copied.');
  });

  it('shows the exact change before confirming it, and refuses a partial paste', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply({ message: 'Give this code to your AI helper.',
      confirmation: '123.signed' }));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeHelpers, props));
    await type(control(host, 'Confirm request from your helper'), 'twrc1.broken');
    expect(host.textContent).toContain('not a whole confirm request');
    expect(button(host, 'Confirm this change').disabled).toBe(true);
    const request = 'twrc1.' + btoa(JSON.stringify({ tokenId: id,
      change: { command: 'commish.franchise.edit', reason: 'New name', args: { people: { name: 'HT' } } } }))
      .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    await type(control(host, 'Confirm request from your helper'), request);
    expect(host.textContent).toContain('You are confirming commish.franchise.edit because “New name”');
    await click(button(host, 'Confirm this change')); await settle();
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({ action: 'helper-confirm', change: request });
    expect(host.textContent).toContain('Code for your helper: 123.signed');
  });
});
