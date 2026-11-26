import { expect, it } from 'vitest';
import { oldestOwedPick, parseDraft, type Draft } from './draft';

// A league with commissioner draft corrections sends them on the draft read. HQ reads the draft in its one
// refresh, so refusing them left every owner's lineup read-only.
it('reads a draft that carries commissioner corrections', () => {
  const draft = parseDraft({ league: 'l', season: 2026, revision: 3, rules: [], owners: [], order: [],
    words: 'Draft closed', nextOwnedPick: null, closed: null, players: {}, available: [], checkpoints: [],
    draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: true, picks: [], onClock: null },
    corrections: [{ entry: 'e1', who: 'chris', when: '2026-05-02T00:00:00Z', why: 'MFL import',
      what: 'pick moved' }] });
  expect(draft.season).toBe(2026);
  expect(draft.exhausted).toBe(true);
});

it('finds only the acting franchise’s oldest pick while it remains eligible', () => {
  const owed = (round: number, owner = 'mine') => ({ round, slot: round, owner, original: owner,
    extra: 0, player: '', skipped: true, forfeited: false, madeAt: '', contract: null,
    how: 'expired' as const, changed: false });
  const draft = ({ closed: null, onClock: { paused: false },
    picks: [owed(1), owed(2), owed(1, 'other')] } as unknown as Draft);
  expect(oldestOwedPick(draft, 'mine')?.round).toBe(1);
  expect(oldestOwedPick(draft, 'other')?.owner).toBe('other');
  expect(oldestOwedPick({ ...draft, onClock: { paused: false, owner: 'other' } } as Draft,
    'mine')?.round).toBe(1);
  expect(oldestOwedPick({ ...draft, onClock: { paused: false, owner: 'mine' } } as Draft,
    'mine')?.round).toBe(1);
  expect(oldestOwedPick({ ...draft, onClock: null, exhausted: true } as Draft, 'mine')?.round).toBe(1);
  expect(oldestOwedPick(draft, '')).toBeUndefined();
  expect(oldestOwedPick({ ...draft, onClock: { paused: true } } as Draft, 'mine')).toBeUndefined();
  expect(oldestOwedPick({ ...draft, closed: {} } as Draft, 'mine')).toBeUndefined();
});
