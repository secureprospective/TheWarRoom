import { commands } from '../commands/registry';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, button, click, control, mount, render, settle, type, unmount }
  from '../commissioner/commissionerDom';
import NativeAccount from './NativeAccount';
import { nativeAction, nativeStartup, parseNativeView, type NativeView } from './nativeApi';

// Franchise workflow requests are covered by the hosted-room tests, not account admission.
vi.mock('../commissioner/HostedLeague', () => ({ default: () => null }));

const initial: NativeView = { server: 'https://league.example', league: 'native.test', message: '',
  name: '', role: '', franchise: '', franchiseName: '', seed: '', connection: '', helperId: '',
  confirmation: '', seats: [] };
function reply(value: NativeView) {
  return { ok: true, headers: new Headers({ 'Content-Type': 'application/json' }), json: async () => value };
}
afterEach(async () => { await unmount(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });
const submit = async (host: Awaited<ReturnType<typeof mount>>) => {
  await act(async () => {
    const form = all(host, 'form')[0];
    (form.props().onSubmit as (event: { preventDefault(): void }) => void)({ preventDefault() {} });
  });
  await settle();
};

describe('native desktop account access', () => {
  it('restores saved sign-in controls and revokes without sending passwords', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply({
      ...initial, message: 'All device sessions revoked.',
    }));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeAccount, {
      initial: { ...initial, message: 'Saved device session available.' },
    }));
    await click(button(host, 'Revoke all devices')); await settle();
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({
      action: 'revoke-all', username: '', password: '', code: '',
    });
    expect(host.textContent).toContain('All device sessions revoked.');
    expect(host.textContent).not.toContain('Open my franchise');
  });

  it('joins without MFL, signs in and opens only the server-assigned franchise', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply(initial));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeAccount, { initial }));
    expect(host.textContent).toContain('No MFL account or key needed');
    await click(button(host, 'Join with a code'));
    await type(control(host, 'Username'), 'owner');
    await type(control(host, 'Password'), 'private-password');
    await type(control(host, 'Personally handed enrollment code'), 'private-invite');
    await submit(host);
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({
      action: 'enroll', username: 'owner', password: 'private-password', code: 'private-invite',
    });
    expect(control(host, 'Password').value).toBe('');
    await type(control(host, 'Password'), 'private-password');
    await submit(host);
    fetcher.mockResolvedValue(reply({ ...initial, name: 'League', role: 'Owner',
      franchise: 'franchise.01', franchiseName: 'My club' }));
    await click(button(host, 'Open my franchise'));
    await settle();
    expect(host.textContent).toContain('My club');
    expect(JSON.parse(fetcher.mock.calls[2][1].body)).toEqual({
      action: 'enter', username: '', password: '', code: '',
    });
    // A refused reread must remove the previously private view, not show it as current.
    fetcher.mockResolvedValue({ ok: false, text: async () => 'Session revoked. Sign in again.' });
    await click(button(host, 'Open my franchise')); await settle();
    expect(host.textContent).not.toContain('My club');
    expect(host.textContent).toContain('Sign in again');
  });

  it('makes expired invitations and lost factors recoverable without promising a seat reset', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false,
      text: async () => 'Code expired; ask your commissioner for a new code.' }));
    const host = await mount(render(NativeAccount, { initial }));
    await click(button(host, 'Join with a code'));
    await type(control(host, 'Username'), 'owner');
    await type(control(host, 'Password'), 'private-password');
    await type(control(host, 'Personally handed enrollment code'), 'old-code');
    await submit(host);
    expect(all(host, 'p').some(p => p.getAttribute('role') === 'alert'
      && p.textContent.includes('Code expired'))).toBe(true);
    await click(button(host, 'Account recovery'));
    expect(host.textContent).toContain('48 hours');
    expect(host.textContent).toContain('eight printed one-time recovery codes');
    expect(host.textContent).toContain('audited console recovery');
    expect(host.textContent).toContain('cannot reset your account by reassigning a seat');
    expect(all(host, 'input')).toHaveLength(3);
    await click(button(host, 'Back to sign in'));
    expect(control(host, 'Password').value).toBe('');
  });

  it('resets with a printed code, clears secrets and returns to sign-in', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      reply({ ...initial, message: 'Password reset. Sign in next.' }));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeAccount, { initial }));
    await click(button(host, 'Account recovery'));
    await type(control(host, 'Username'), 'owner');
    await type(control(host, 'Unused recovery code'), 'a'.repeat(43));
    await type(control(host, 'New password'), 'replacement-password');
    await submit(host);
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({ action: 'recover', username: 'owner',
      password: 'replacement-password', code: 'a'.repeat(43) });
    expect(host.textContent).toContain('Password reset');
    expect(control(host, 'Password').value).toBe('');
    expect(all(host, 'input').some(input => input.value === 'a'.repeat(43))).toBe(false);
  });

  it('shows the private setup key but never requests bearer or proof from JavaScript', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply({ ...initial, seed: 'PRIVATESETUPKEY' }));
    vi.stubGlobal('fetch', fetcher);
    const host = await mount(render(NativeAccount, { initial }));
    await click(button(host, 'Authenticator'));
    await type(control(host, 'Username'), 'commissioner');
    await type(control(host, 'Password'), 'private-password');
    await click(button(host, 'Get setup key')); await settle();
    expect(host.textContent).toContain('PRIVATESETUPKEY');
    expect(control(host, 'Password').value).toBe('');
    await click(button(host, 'Account recovery'));
    expect(host.textContent).not.toContain('PRIVATESETUPKEY');
    expect(fetcher.mock.calls[0][0]).toBe('/desktop/native');
    expect(fetcher.mock.calls[0][1].headers).toEqual({ 'Content-Type': 'application/json' });
  });

  it('validates the local transport and distinguishes local startup without fetching a server', async () => {
    const fetcher = vi.fn().mockResolvedValue(reply(initial));
    vi.stubGlobal('fetch', fetcher);
    await expect(nativeStartup()).resolves.toEqual(initial);
    expect(fetcher.mock.calls[0][0]).toBe('/desktop/native');
    fetcher.mockResolvedValue({ ok: true, headers: new Headers({ 'Content-Type': 'text/html' }) });
    await expect(nativeStartup()).resolves.toBeNull();
    fetcher.mockResolvedValue({ ok: false, status: 404 });
    await expect(nativeStartup()).resolves.toBeNull();
    expect(() => parseNativeView({ ...initial, bearer: 'must-not-cross' })).toThrow();
    expect(() => parseNativeView({ ...initial, league: 3 })).toThrow();
    fetcher.mockResolvedValue({ ok: false, text: async () => 'Fresh code required' });
    await expect(nativeAction('totp-step-up', 'owner', 'password', '123456'))
      .rejects.toThrow('Fresh code required');
  });
});

it.each(['done', 'failed'] as const)('records account %s after admission', async status => {
  let finish!: (value: ReturnType<typeof reply>) => void;
  let fail!: (cause: Error) => void;
  const reason = 'Your league cannot be reached. Try again.';
  vi.stubGlobal('fetch', vi.fn(() => new Promise<ReturnType<typeof reply>>((resolve, reject) => {
    finish = resolve; fail = reject;
  })));
  vi.spyOn(console, 'error').mockImplementation(() => {});
  await mount(render(NativeAccount, { initial }));
  const before = commands.history();
  let result: void | Promise<void>;
  await act(async () => {
    result = commands.dispatch('commissioner.surface', {
      screen: 'native-account', action: 'check', value: 'submit',
    });
  });
  expect(commands.history()).toEqual(before);
  await act(async () => {
    if (status === 'done') finish(reply(initial));
    else fail(new Error(reason));
    await result;
  });
  expect(commands.history()[commands.history().length - 1]).toEqual({ id: 'commissioner.surface',
    args: { screen: 'native-account', action: 'check', value: 'submit' }, status,
    ...(status === 'failed' ? { reason } : {}) });
});
