// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it } from 'vitest';
import { parseDraft } from './draft';
import { YourPick } from './YourPick';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove();
});

it('shows the missed slot and its server-projected terms in the shared chooser', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const draft = parseDraft({ league: 'native.test', season: 2026, revision: 2,
    rules: [], owners: [], order: [],
    words: 'Draft running', nextOwnedPick: null, closed: null,
    players: {}, available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }], checkpoints: [],
    draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: false, onClock: null,
      picks: [{ round: 1, original: 'franchise.01', owner: 'franchise.01', slot: 3, extra: 0,
        player: '', skipped: true, forfeited: false, how: 'expired', contract: {
          salaryCents: 1234500, years: 3, rfaAfter: true,
        } }],
    },
  });
  await act(async () => root!.render(
    <YourPick draft={draft} owed={draft.picks[0]} outcome={{ status: 'idle' }} />));
  expect(host.textContent).toContain('You missed round 1 pick 3. Pick any time.');
  expect(host.textContent).toContain('This pick signs for $12,345 for 3 years, then restricted free agency.');
  expect(host.querySelectorAll('select')[1]?.querySelector('option')?.textContent).toBe('Choose…');
});
