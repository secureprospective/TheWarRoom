import { it, expect } from 'vitest';
import { createCommands } from '../commands/registry';
import { connectRoutes, dispatchKey } from './bindings';
it('opens a deep link, replaces on navigation, normalizes bad hashes and unsubscribes', () => {
  const c = createCommands();
  const location = { hash: '#/war/research' };
  const replaced: string[] = [];
  const binding = connectRoutes(c, location, (hash) => {
    location.hash = hash;
    replaced.push(hash);
  });
  expect(c.read()).toMatchObject({ node: 'war', workspace: { war: 'research' } });
  expect(replaced).toEqual([]);
  c.dispatch('nav.open', { node: 'trade', workspace: 'draft-room' });
  expect(location.hash).toBe('#/trade/draft-room');
  location.hash = '#/bad/unknown';
  binding.fromHash();
  expect(location.hash).toBe('#/home/seasonal-card');
  c.dispatch('nav.open', { node: 'war' });
  expect(location.hash).toBe('#/war/research');
  binding.unsubscribe();
  c.dispatch('nav.open', { node: 'hq' });
  expect(location.hash).toBe('#/war/research');
});
it('ignores typing, modified shortcuts and the harness without dispatching', () => {
  const c = createCommands();
  const event = { key: '1', target: null, ctrlKey: false, metaKey: false, altKey: false };
  for (const target of [
    { tagName: 'INPUT' },
    { tagName: 'TEXTAREA' },
    { tagName: 'DIV', isContentEditable: true },
  ]) {
    expect(dispatchKey({ ...event, target: target as unknown as EventTarget }, c)).toBe(
      false,
    );
  }
  for (const modifier of ['ctrlKey', 'metaKey', 'altKey'])
    expect(dispatchKey({ ...event, [modifier]: true }, c)).toBe(false);
  expect(c.history()).toHaveLength(0);
  expect(dispatchKey(event, c)).toBe(true);
  expect(c.read().density).toBe('narrative');
  c.dispatch('harness.open', {});
  expect(dispatchKey({ ...event, key: '3' }, c)).toBe(false);
  expect(c.read().density).toBe('narrative');
});
