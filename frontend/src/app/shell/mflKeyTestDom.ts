import { TestDocument, TestElement } from '../clock/testDom';

export class KeyInputElement extends TestElement {
  value = '';
  defaultValue = '';
  get disabled() { return this.attributes.has('disabled'); }
  type = '';
  readonly events = new Map<string, Set<() => void>>();
  addEventListener(name?: string, listener?: () => void) {
    if (!name || !listener) return;
    if (!this.events.has(name)) this.events.set(name, new Set());
    this.events.get(name)?.add(listener);
  }
  removeEventListener(name?: string, listener?: () => void) {
    if (name && listener) this.events.get(name)?.delete(listener);
  }
  focus() { (this.ownerDocument as KeyDocument).focused = this; }
  input(text: string) {
    this.value = text;
    this.events.get('input')?.forEach((listener) => listener());
  }
  props(): Record<string, unknown> {
    const key = Object.keys(this).find((name) => name.startsWith('__reactProps$'));
    if (!key) throw new Error('Missing React host props');
    return (this as unknown as Record<string, Record<string, unknown>>)[key];
  }
}

export class KeyDocument extends TestDocument {
  focused: KeyInputElement | undefined;
  createElement(tag: string) { return new KeyInputElement(tag.toUpperCase(), this); }
}

export function descendants(root: TestElement): KeyInputElement[] {
  return root.children.flatMap((child): KeyInputElement[] => {
    if (!(child instanceof KeyInputElement)) return [];
    return [child, ...descendants(child)];
  });
}
