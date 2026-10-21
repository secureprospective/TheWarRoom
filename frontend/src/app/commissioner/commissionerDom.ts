import { act, createElement, type ReactElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { vi } from 'vitest';
import { KeyDocument, descendants } from '../shell/mflKeyTestDom';
import type { KeyInputElement } from '../shell/mflKeyTestDom';

// A headless page for the commissioner forms: React renders into a fake DOM, the typed Go bindings are
// stubbed, and a test types, clicks and reads exactly what a person would.
export type Element = KeyInputElement;
type Binding = (...args: string[]) => Promise<unknown>;
export type App = Record<'TargetCommissionerExecute' | 'TargetCommissionerLedger'
  | 'TargetCommissionerLeagues' | 'TargetCommissionerRules' | 'TargetCommissionerPeople'
  | 'TargetCommissionerPreview' | 'TargetCommissionerLedgerFiltered' | 'TargetCommissionerJobs'
  | 'TargetNativeAssets' | 'TargetNativeDirectory' | 'TargetNativeBracket' | 'TargetNativeTrades'
  | 'TargetNativeWaivers' | 'TargetNativeBids' | 'TargetCommissionerActor' | 'TargetCommissionerMarket'
  | 'TargetCommissionerSchedule' | 'TargetCommissionerControls' | 'TargetCommissionerNotices'
  | 'TargetCommissionerDigests', Binding>;

const globals = ['window', 'document', 'IS_REACT_ACT_ENVIRONMENT'];
const held = globals.map(key => ({ key, descriptor: Object.getOwnPropertyDescriptor(globalThis, key) }));
let root: Root | undefined;

export const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
// features stubs the bound feature APIs (window.go.<pkg>.API) alongside the root App.
export async function mount(node: ReactElement, app: Partial<App> = {},
  features: Record<string, unknown> = {}) {
  const document = new KeyDocument();
  const host = document.createElement('div');
  vi.stubGlobal('document', document);
  const go = { commissioner: { API: app }, ...features };
  vi.stubGlobal('window', { document, HTMLIFrameElement: class {}, go });
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  root = createRoot(host as unknown as HTMLElement);
  await act(async () => root?.render(node));
  return host as unknown as Element;
}
export async function unmount() {
  if (root) await act(async () => root?.unmount());
  root = undefined;
  for (const { key, descriptor } of held) {
    if (descriptor) Object.defineProperty(globalThis, key, descriptor);
    else Reflect.deleteProperty(globalThis, key);
  }
}
export const all = (host: Element, tag: string) =>
  descendants(host).filter(e => e.tagName === tag.toUpperCase());
export function control(host: Element, label: string): Element {
  const found = all(host, 'label').find(l => l.textContent.startsWith(label));
  const field = found && descendants(found).find(e => ['INPUT', 'SELECT', 'TEXTAREA'].includes(e.tagName));
  if (!field) throw new Error(`No field labelled “${label}”`);
  return field;
}
export function button(host: Element, label: string): Element {
  const found = all(host, 'button').find(b =>
    b.textContent === label || b.getAttribute('aria-label') === label);
  if (!found) throw new Error(`No button “${label}”`);
  return found;
}
// A hold-to-fire button's events are handled by the element around it.
export const holder = (host: Element, label: string) => button(host, label).parentNode as unknown as Element;
export const present = (host: Element, text: string) => host.textContent.includes(text);
export const disabled = (e: Element) => e.attributes.has('disabled');
export const type = (e: Element, value: string) =>
  act(async () => { (e.props().onChange as (event: unknown) => void)({ target: { value } }); });
export const check = (e: Element, checked: boolean) =>
  act(async () => { (e.props().onChange as (event: unknown) => void)({ target: { checked } }); });
export const click = (e: Element) => act(async () => { (e.props().onClick as () => void)(); });
export const render = createElement;
export const flush = () => act(async () => { for (let i = 0; i < 40; i++) await Promise.resolve(); });
export const rerender = (node: ReactElement) => act(async () => root?.render(node));
