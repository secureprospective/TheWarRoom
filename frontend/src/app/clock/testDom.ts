// A headless DOM surface for React's host renderer; no browser or extra dependency is needed.
export class TestElement {
  readonly nodeType = 1;
  readonly namespaceURI = 'http://www.w3.org/1999/xhtml';
  readonly style = {};
  readonly attributes = new Map<string, string>();
  readonly children: (TestElement | TestText)[] = [];
  parentNode: TestElement | null = null;
  constructor(readonly tagName: string, readonly ownerDocument: TestDocument) {}
  get nodeName() { return this.tagName; }
  get textContent(): string {
    return this.children.map((child) => child.textContent).join('');
  }
  set textContent(value: string) {
    this.children.splice(0);
    if (value) this.appendChild(new TestText(value));
  }
  appendChild(child: TestElement | TestText) {
    child.parentNode?.removeChild(child);
    child.parentNode = this;
    this.children.push(child);
    return child;
  }
  removeChild(child: TestElement | TestText) {
    this.children.splice(this.children.indexOf(child), 1);
    child.parentNode = null;
    return child;
  }
  insertBefore(child: TestElement | TestText, before: TestElement | TestText) {
    child.parentNode?.removeChild(child);
    child.parentNode = this;
    this.children.splice(this.children.indexOf(before), 0, child);
    return child;
  }
  setAttribute(name: string, value: string) { this.attributes.set(name, String(value)); }
  removeAttribute(name: string) { this.attributes.delete(name); }
  getAttribute(name: string) { return this.attributes.get(name) ?? null; }
  addEventListener() {}
  removeEventListener() {}
  querySelectorAll(selector: string): TestElement[] {
    const attribute = selector.slice(1, -1);
    return this.children.flatMap((child): TestElement[] => {
      if (!(child instanceof TestElement)) return [];
      const nested = child.querySelectorAll(selector);
      return child.attributes.has(attribute) ? [child, ...nested] : nested;
    });
  }
  querySelector(selector: string) { return this.querySelectorAll(selector)[0] ?? null; }
}

class TestText {
  readonly nodeType = 3;
  parentNode: TestElement | null = null;
  constructor(public nodeValue: string) {}
  get textContent() { return this.nodeValue; }
  set textContent(value: string) { this.nodeValue = value; }
}

export class TestDocument {
  readonly nodeType = 9;
  hidden = false;
  readonly documentElement = this.createElement('html');
  readonly body = this.createElement('body');
  readonly activeElement = this.body;
  readonly listeners = new Set<() => void>();
  createElement(tag: string) { return new TestElement(tag.toUpperCase(), this); }
  createElementNS(_namespace: string, tag: string) { return this.createElement(tag); }
  createTextNode(text: string) { return new TestText(text); }
  addEventListener(name: string, listener: () => void) {
    if (name === 'visibilitychange') this.listeners.add(listener);
  }
  removeEventListener(name: string, listener: () => void) {
    if (name === 'visibilitychange') this.listeners.delete(listener);
  }
  visibility(hidden: boolean) {
    this.hidden = hidden;
    for (const listener of this.listeners) listener();
  }
}

export function clockDom(ids: string[]) {
  const document = new TestDocument();
  const root = document.createElement('div');
  for (const id of ids) {
    const node = document.createElement('span');
    node.setAttribute('data-deadline', id);
    const text = document.createElement('span');
    text.setAttribute('data-countdown', '');
    node.appendChild(text);
    root.appendChild(node);
  }
  return { document, root, element: root as unknown as HTMLElement };
}
