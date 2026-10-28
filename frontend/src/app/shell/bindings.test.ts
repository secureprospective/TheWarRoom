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
it('ignores typing and modified shortcuts without dispatching', () => {
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
});

it('normalizes removed Archive hashes', () => {
  const c = createCommands();
  const location = { hash: '#/pulse/pools-archive' };
  const binding = connectRoutes(c, location, hash => { location.hash = hash; });
  expect(location.hash).toBe('#/home/seasonal-card');
  binding.unsubscribe();
});

it('rejects removed workspaces while keeping the draft room and value board', () => {
  const c = createCommands();
  for (const hash of ['#/war/market', '#/trade/counterparties', '#/trade/offers-history',
    '#/pulse/archive', '#/control/data-and-sources']) {
    const location = { hash };
    const binding = connectRoutes(c, location, value => { location.hash = value; });
    expect(location.hash).toBe('#/home/seasonal-card');
    binding.unsubscribe();
  }
  for (const hash of ['#/trade/draft-room', '#/war/valuations-and-pool']) {
    const location = { hash };
    const binding = connectRoutes(c, location, value => { location.hash = value; });
    expect(location.hash).toBe(hash);
    binding.unsubscribe();
  }
});
