// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it } from 'vitest';
import { parseDraft } from './draft';
import { DraftClock } from './DraftClock';

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove();
});

it('does not say an expired pick closes when the league waits on its owner', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const draft = parseDraft({ league: 'native.test', season: 2026, revision: 2,
    rules: [{ label: 'Expired draft clock', words: 'Wait for the owner or commissioner (no expiry)' }],
    owners: [{ id: 'franchise.01', name: 'Team One' }], order: [], words: 'Draft running',
    nextOwnedPick: null, closed: null, players: {}, available: [], checkpoints: [],
    draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: false,
      picks: [], onClock: { round: 1, original: 'franchise.01', owner: 'franchise.01', slot: 1,
        openedAt: '2026-05-01T00:00:00Z', deadline: '2026-05-01T00:01:00Z',
        contract: { salaryCents: 1000000, years: 3, rfaAfter: false }, paused: false, remaining: 0 },
    },
  });
  await act(async () => root!.render(<DraftClock draft={draft} may={false}
    now={Date.parse(draft.onClock!.deadline)} />));
  expect(host.textContent).toContain('The pick stays open until its owner or commissioner acts.');
  expect(host.textContent).not.toContain('The pick closes');
});
