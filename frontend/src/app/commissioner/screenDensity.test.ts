import { afterEach, expect, it, vi } from 'vitest';
import { KeyDocument } from '../shell/mflKeyTestDom';
import { createShellState } from '../shell/state';
import { connectDensity } from '../shell/density';
import { connectScreenDensity } from './screenDensity';
import { surfaceActions } from './surfaceActions';

const held = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
afterEach(() => {
  if (held) Object.defineProperty(globalThis, 'localStorage', held);
  else Reflect.deleteProperty(globalThis, 'localStorage');
});

it('keeps nested commissioner density buttons independent of the owner shell', () => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined });
  const document = new KeyDocument();
  const root = document.createElement('section');
  const nested = document.createElement('section');
  const ownerButton = document.createElement('button');
  const nestedButton = document.createElement('button');
  for (const button of [ownerButton, nestedButton]) button.setAttribute('class', 'act act-density');
  ownerButton.setAttribute('data-d', 'narrative');
  nestedButton.setAttribute('data-d', 'tactical');
  root.appendChild(ownerButton);
  root.appendChild(nested);
  nested.appendChild(nestedButton);
  const source = createShellState();
  const stopOwner = connectDensity(source, root as unknown as HTMLElement);
  const stopScreen = connectScreenDensity('ledger', nested as unknown as HTMLElement);
  try {
    source.write({ density: 'narrative' });
    expect(root.getAttribute('data-density')).toBe('narrative');
    expect(nested.getAttribute('data-density')).toBe('tactical');
    expect(ownerButton.getAttribute('aria-pressed')).toBe('true');
    expect(nestedButton.getAttribute('aria-pressed')).toBe('true');
    surfaceActions.run({ screen: 'density.ledger', action: 'view', value: 'matrix' });
    expect(nested.getAttribute('data-density')).toBe('matrix');
    expect(root.getAttribute('data-density')).toBe('narrative');
    expect(nestedButton.getAttribute('aria-pressed')).toBe('false');
  } finally { stopScreen(); stopOwner(); }
  surfaceActions.run({ screen: 'density.ledger', action: 'view', value: 'narrative' });
  expect(nested.getAttribute('data-density')).toBe('matrix');
});
