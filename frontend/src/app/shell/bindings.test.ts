import { it, expect } from 'vitest';
import { createCommands } from '../commands/registry';
import { connectRoutes, dispatchKey } from './bindings';
it('opens a deep link, replaces on navigation, normalizes bad hashes and unsubscribes', () => {
  const c = createCommands();
  const location = { hash: '#/hq/contracts-and-cap' };
  const replaced: string[] = [];
  const binding = connectRoutes(c, location, (hash) => {
    location.hash = hash;
    replaced.push(hash);
  });
  expect(c.read()).toMatchObject({ node: 'hq', workspace: { hq: 'contracts-and-cap' } });
  expect(replaced).toEqual([]);
  c.dispatch('nav.open', { node: 'war', workspace: 'draft-room' });
  expect(location.hash).toBe('#/war/draft-room');
  location.hash = '#/bad/unknown';
  binding.fromHash();
  expect(location.hash).toBe('#/home/overview');
  c.dispatch('nav.open', { node: 'hq' });
  expect(location.hash).toBe('#/hq/contracts-and-cap');
  binding.unsubscribe();
  c.dispatch('nav.open', { node: 'war' });
  expect(location.hash).toBe('#/hq/contracts-and-cap');
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
  expect(location.hash).toBe('#/home/overview');
  binding.unsubscribe();
});

it('rejects removed workspaces while keeping the draft room and value board', () => {
  const c = createCommands();
  for (const hash of ['#/home/alert-tray', '#/home/seasonal-card', '#/war/market',
    '#/trade/trade-desk', '#/trade/draft-room', '#/trade/counterparties', '#/trade/offers-history',
    '#/pulse/archive', '#/control/data-and-sources']) {
    const location = { hash };
    const binding = connectRoutes(c, location, value => { location.hash = value; });
    expect(location.hash).toBe('#/home/overview');
    binding.unsubscribe();
  }
  for (const hash of ['#/war/draft-room', '#/war/valuations-and-pool']) {
    const location = { hash };
    const binding = connectRoutes(c, location, value => { location.hash = value; });
    expect(location.hash).toBe(hash);
    binding.unsubscribe();
  }
});
