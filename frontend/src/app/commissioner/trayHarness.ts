import { act } from 'react';
import { KeyInputElement, descendants } from '../shell/mflKeyTestDom';
import type { Element } from './commissionerDom';

// Model only the browser focus APIs the tray consumes; the form harness already models rendering.
// Returns the call that puts the headless element back as it was.
export function modelFocusApis(): () => void {
  const keys = ['querySelectorAll', 'hasAttribute', 'tabIndex', 'closest', 'contains'];
  const held = new Map(keys.map(key =>
    [key, Object.getOwnPropertyDescriptor(KeyInputElement.prototype, key)]));
  Object.defineProperties(KeyInputElement.prototype, {
    querySelectorAll: { configurable: true, value(this: Element, selector: string) {
      return descendants(this).filter(e => selector === '[hidden]' ? e.attributes.has('hidden') :
        ['BUTTON', 'INPUT', 'SELECT', 'TEXTAREA'].includes(e.tagName) || e.attributes.has('tabindex'));
    } },
    hasAttribute: { configurable: true, value(this: Element, key: string) {
      return this.attributes.has(key);
    } },
    tabIndex: { configurable: true, get(this: Element) {
      return Number(this.getAttribute('tabindex') ?? 0);
    } },
    closest: { configurable: true, value() { return null; } },
    contains: { configurable: true, value(this: Element, other: Element) {
      return this === other || descendants(this).includes(other);
    } },
  });
  return () => {
    for (const [key, descriptor] of held) {
      if (descriptor) Object.defineProperty(KeyInputElement.prototype, key, descriptor);
      else Reflect.deleteProperty(KeyInputElement.prototype, key);
    }
  };
}

export const focused = (host: Element) => (host.ownerDocument as unknown as { focused: Element }).focused;
export function active(host: Element) {
  Object.defineProperty(host.ownerDocument, 'activeElement', { configurable: true,
    get: () => focused(host) });
}
export const key = (panel: Element, name: string, shiftKey = false) => act(async () => {
  (panel.props().onKeyDown as (event: object) => void)({ key: name, shiftKey,
    preventDefault() {}, stopPropagation() {} });
});
export const dialogOf = (host: Element) =>
  descendants(host).find(e => e.tagName === 'SECTION' && e.getAttribute('role') === 'dialog');
