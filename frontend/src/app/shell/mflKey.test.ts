import { afterEach, describe, expect, it, vi } from 'vitest';
import { randomBytes } from 'node:crypto';
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { createCommands } from '../commands/registry';
import * as registry from '../commands/registry';
import { FixtureProvider, LiveProvider, type Provider } from '../data/provider';
import type { MFLKeyStatus } from '../data/contract';
import MFLConnection from './MFLConnection';
import { descendants, KeyDocument, type KeyInputElement } from './mflKeyTestDom';

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'].map((key) => ({
  key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key),
}));
let mounted: Root | undefined;
afterEach(() => {
  if (mounted) act(() => mounted?.unmount());
  mounted = undefined;
  vi.restoreAllMocks();
  for (const { key, descriptor } of globals) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
});
const absent: MFLKeyStatus = { state: 'absent', league: '14432', season: 2026 };
const connected: MFLKeyStatus = {
  ...absent, state: 'connected', verifiedAt: '2026-10-07T14:00:00Z',
};

function assertAbsent(sentinel: string, surfaces: string[]) {
  if (surfaces.some((surface) => surface.includes(sentinel))) throw new Error('Secret leak');
}
async function settle() {
  await act(async () => { await new Promise((done) => setTimeout(done, 0)); });
}
async function mount(provider: Provider = new LiveProvider(), status: MFLKeyStatus = absent,
  initialRead?: () => Promise<MFLKeyStatus>) {
  const read = vi.spyOn(provider, 'mflKey').mockResolvedValue(status);
  if (initialRead) read.mockImplementation(initialRead);
  const c = createCommands(() => undefined, provider);
  vi.spyOn(registry, 'commands', 'get').mockReturnValue(c);
  vi.spyOn(registry, 'dispatch').mockImplementation(c.dispatch);
  const document = new KeyDocument();
  const container = document.createElement('div');
  vi.stubGlobal('document', document);
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {} });
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  mounted = createRoot(container as unknown as HTMLElement);
  let outerRenders = 0;
  function SettingsProbe() {
    outerRenders += 1;
    return createElement(MFLConnection);
  }
  act(() => mounted?.render(createElement(SettingsProbe)));
  await settle();
  const all = () => descendants(container);
  const input = all().find((element) => element.tagName === 'INPUT')!;
  const button = (label: string) => all().find((element) => element.getAttribute('aria-label') === label);
  return { c, container, all, input, button, read, document, outerRenders: () => outerRenders };
}
function invoke(element: KeyInputElement | undefined) {
  if (!element || element.disabled) throw new Error('Control not enabled');
  (element.props().onClick as () => void)();
}

describe('MFL key leak gate (§M3)', () => {
  it('detects a deliberate leak', () => {
    const sentinel = 'SENTINEL-' + randomBytes(24).toString('hex');
    expect(() => assertAbsent(sentinel, [`prefix ${sentinel}`])).toThrow('Secret leak');
  });
  it('takes the field exactly once and leaks nowhere, including while pending', async () => {
    const sentinel = 'SENTINEL-' + randomBytes(24).toString('hex');
    const output: unknown[][] = [];
    const methods = Object.getOwnPropertyNames(console).filter((key) =>
      typeof (console as unknown as Record<string, unknown>)[key] === 'function');
    for (const method of methods) {
      vi.spyOn(console, method as 'log').mockImplementation((...args: unknown[]) => { output.push(args); });
    }
    const provider = new LiveProvider();
    let finish!: (status: MFLKeyStatus) => void;
    const connect = vi.spyOn(provider, 'connectMFL').mockImplementation(() =>
      new Promise<MFLKeyStatus>((done) => { finish = done; }));
    const { c, input, container, all, outerRenders } = await mount(provider);
    const initialRenders = outerRenders();
    const shellChanged = vi.fn();
    c.subscribe(shellChanged);
    act(() => {
      input.input(sentinel);
      c.dispatch('mflkey.connect', {});
      c.dispatch('mflkey.connect', {});
    });
    expect(connect).toHaveBeenCalledTimes(1);
    expect(connect).toHaveBeenCalledWith(sentinel);
    const check = () => assertAbsent(sentinel, [
      JSON.stringify(c.history()), JSON.stringify(c.read()), JSON.stringify(c.readMFLKey()),
      container.textContent, ...all().filter((element) => element.tagName === 'INPUT')
        .map((element) => element.value), JSON.stringify(output),
    ]);
    check();
    await act(async () => { finish(connected); });
    check();
    expect(shellChanged).not.toHaveBeenCalled();
    expect(outerRenders()).toBe(initialRenders);
    expect(c.readMFLKey().pending).toBe(false);
    expect(input.value).toBe('');
    const props = input.props();
    for (const forbidden of ['value', 'name', 'onChange']) expect(props).not.toHaveProperty(forbidden);
    expect(props).toMatchObject({ type: 'password', autoComplete: 'off', spellCheck: false });
  });
});

describe('MFL connection surface', () => {
  it('does not read at launch; reads once on mount and names the keyring while pending', async () => {
    const provider = new LiveProvider();
    const before = vi.spyOn(provider, 'mflKey');
    const c = createCommands(() => undefined, provider);
    expect(before).not.toHaveBeenCalled();
    expect(c.registry['mflkey.connect']).toMatchObject({
      args: [], aliases: ['mfl', 'api key'], gravity: 'G1', undo: 'reversible',
    });
    expect(c.registry['mflkey.forget']).toMatchObject({ args: [], gravity: 'G1', undo: 'reversible' });
    let finish!: (status: MFLKeyStatus) => void;
    const { container, read, button } = await mount(provider, absent, () =>
      new Promise<MFLKeyStatus>((done) => { finish = done; }));
    expect(read).toHaveBeenCalledTimes(1);
    expect(container.textContent).toContain('Checking the keyring…');
    expect(button('Connect MFL')?.disabled).toBe(true);
    await act(async () => { finish(absent); });
    expect(container.textContent).toContain('Not connected');
  });
  it('focuses a newly mounted field after an empty command-bar invocation', async () => {
    const provider = new LiveProvider();
    const connect = vi.spyOn(provider, 'connectMFL');
    const c = createCommands(() => undefined, provider);
    c.dispatch('commandbar.open', {});
    c.dispatch('mflkey.connect', {});
    await settle();
    const document = new KeyDocument();
    const input = document.createElement('input');
    c.mflField.mount(input as unknown as HTMLInputElement);
    expect(document.focused).toBe(input);
    expect(connect).not.toHaveBeenCalled();
    input.input('temporary');
    c.mflField.mount(null);
    expect(input.value).toBe('');
    expect(input.events.get('input')?.size).toBe(0);
  });
  it.each([
    ['absent', 'Not connected · paste your API key from MFL: Help › Developer\'s API'],
    ['connected', 'Connected · league 14432 · 2026'],
    ['unavailable', 'Keyring unavailable · Unlock the keyring'],
    ['rejected', 'Key not accepted · Unlock the keyring'],
    ['unreachable', 'Could not reach MFL · Unlock the keyring'],
  ] as const)('renders %s plainly', async (state, line) => {
    const { container, button } = await mount(new LiveProvider(), {
      ...connected, state, detail: 'Unlock the keyring',
    });
    expect(container.textContent).toContain(line);
    expect(Boolean(button('Forget MFL key'))).toBe(state === 'connected');
    if (state === 'connected') {
      expect(container.textContent).toContain(new Date(connected.verifiedAt!).toLocaleString());
    }
  });
  it.each(['rejected', 'unreachable'] as const)('preserves the old key on %s', async (state) => {
    const provider = new LiveProvider();
    vi.spyOn(provider, 'connectMFL').mockResolvedValue({
      ...absent, state, detail: 'MFL verdict; nothing was stored',
    });
    const { c, input, container } = await mount(provider, connected);
    act(() => {
      input.input('candidate');
      c.dispatch('mflkey.connect', {});
    });
    await settle();
    expect(container.textContent).toContain('Previously stored key is unchanged');
    expect(container.textContent).toContain('MFL verdict; nothing was stored');
    expect(input.value).toBe('');
  });
  it('disables fixtures with the reason and never reaches mutations', async () => {
    const provider = new FixtureProvider();
    const connect = vi.spyOn(provider, 'connectMFL');
    const forget = vi.spyOn(provider, 'forgetMFL');
    const { c, button, input, container } = await mount(provider);
    expect(container.textContent).toContain('Connecting MFL needs the desktop app');
    expect(input.disabled).toBe(true);
    expect(button('Connect MFL')?.disabled).toBe(true);
    expect(button('Forget MFL key')?.disabled).toBe(true);
    c.dispatch('mflkey.connect', {});
    c.dispatch('mflkey.forget', {});
    expect(connect).not.toHaveBeenCalled();
    expect(forget).not.toHaveBeenCalled();
  });
  it('disables empty Connect and dispatches Enter without mirroring the key', async () => {
    const provider = new LiveProvider();
    const connect = vi.spyOn(provider, 'connectMFL').mockResolvedValue(connected);
    const { c, input, button } = await mount(provider);
    expect(button('Connect MFL')?.disabled).toBe(true);
    act(() => { input.input('candidate'); });
    expect(button('Connect MFL')?.disabled).toBe(false);
    const preventDefault = vi.fn();
    act(() => {
      (input.props().onKeyDown as (event: unknown) => void)({ key: 'Enter', preventDefault });
    });
    await settle();
    expect(preventDefault).toHaveBeenCalledTimes(1);
    expect(connect).toHaveBeenCalledWith('candidate');
    expect(c.history()).toEqual([{ id: 'mflkey.connect', args: {} }]);
    expect(button('Connect MFL')?.disabled).toBe(true);
  });
  it('forgets only when connected, through Act', async () => {
    const provider = new LiveProvider();
    const forget = vi.spyOn(provider, 'forgetMFL').mockResolvedValue(absent);
    const { c, button, container } = await mount(provider, connected);
    act(() => { invoke(button('Forget MFL key')); });
    await settle();
    expect(forget).toHaveBeenCalledTimes(1);
    expect(container.textContent).toContain('Not connected');
    c.dispatch('mflkey.forget', {});
    expect(forget).toHaveBeenCalledTimes(1);
  });
  it('clears pending and the field on a rejected startup binding promise', async () => {
    const provider = new LiveProvider();
    vi.spyOn(provider, 'connectMFL').mockRejectedValue(new Error('app not started'));
    const { c, input, container } = await mount(provider);
    act(() => {
      input.input('candidate');
      c.dispatch('mflkey.connect', {});
    });
    await settle();
    expect(c.readMFLKey().pending).toBe(false);
    expect(container.textContent).toContain('app not started');
    expect(container.textContent).not.toContain('Keyring unavailable');
    expect(input.value).toBe('');
  });
  it('names MFL, not the keyring, while a key is being verified', async () => {
    const provider = new LiveProvider();
    vi.spyOn(provider, 'connectMFL').mockImplementation(() => new Promise<MFLKeyStatus>(() => {}));
    const { c, input, container } = await mount(provider);
    act(() => {
      input.input('candidate');
      c.dispatch('mflkey.connect', {});
    });
    expect(container.textContent).toContain('Checking the key with MFL…');
    expect(container.textContent).not.toContain('Checking the keyring');
  });
  it('an empty command-bar invocation navigates and focuses only the Settings field', async () => {
    const { c, input, document } = await mount();
    c.dispatch('commandbar.open', {});
    c.dispatch('mflkey.connect', {});
    expect(c.read().node).toBe('control');
    expect(c.read().workspace.control).toBe('app');
    expect(c.read().commandbar).toBe(false);
    await settle();
    expect(document.focused).toBe(input);
  });
});
