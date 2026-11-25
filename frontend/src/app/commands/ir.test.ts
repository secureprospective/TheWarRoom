import { expect, it } from 'vitest';
import fixture from '../data/fixtures/snapshot.json';
import { parseSnapshot } from '../data/parse';
import { FixtureProvider } from '../data/provider';
import { createCommands } from './registry';

it.each(['roster.ir', 'roster.taxi'] as const)('%s in preview', intent => {
  const subject = { kind: 'player' as const, id: '11675', franchiseId: '0001' };
  const snapshot = parseSnapshot(fixture);
  const c = createCommands(() => undefined, new FixtureProvider());
  c.loadSnapshot(snapshot);
  expect(c.draftReason(subject)).toBe('Drafting needs the desktop app');
  c.dispatch(intent, { subject });
  expect(c.readMoves().drafting).toEqual({});
  expect(c.readMoves().moves).toEqual([]);
});
