import { expect, it } from 'vitest';
import { parseDraft } from './draft';

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
