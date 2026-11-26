// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import DraftRoom from './DraftRoom';
import type { DraftRoomPort } from './state';

const legacySession = vi.hoisted(() => ({
  value: { choices: [], selected: 'native.stale',
    people: { yourFranchise: 'franchise.01', seats: [{ franchise: 'franchise.01', name: 'Owner' }],
      mayChangeRules: false }, outcome: { status: 'idle' }, error: '', pending: false },
}));
vi.mock('../commissioner/proxy', () => ({
  refusalText: (cause: unknown) => cause instanceof Error ? cause.message : String(cause),
  leagueState: { use: () => legacySession.value, choose: vi.fn() },
}));

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove();
});

it('never falls through to a live pick when debt terms are missing, including commissioner view-as',
  async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    const reads: { franchise: string }[] = [];
    const port: DraftRoomPort = {
      read: async scope => {
        reads.push({ franchise: scope.franchise });
        return { league: 'native.test', season: 2026, revision: 3, rules: [],
          owners: [{ id: 'franchise.01', name: 'Owner' }, { id: 'franchise.02', name: 'Viewed team' }],
          order: [], words: 'Draft running', nextOwnedPick: null, closed: null,
          players: {}, available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }],
          checkpoints: [], draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: false,
            picks: [
              { round: 1, original: 'franchise.01', owner: 'franchise.01', slot: 1, extra: 0,
                player: '', skipped: true, forfeited: false, madeAt: '', contract: null,
                how: 'expired', changed: false },
              { round: 2, original: 'franchise.01', owner: 'franchise.01', slot: 1, extra: 0,
                player: '', skipped: false, forfeited: false, madeAt: '', contract: null,
                how: '', changed: false },
            ], onClock: { round: 2, original: 'franchise.01', owner: 'franchise.01', slot: 1,
              openedAt: '2026-05-01T00:00:00Z', deadline: '2026-05-01T00:10:00Z',
              contract: { salaryCents: 900000, years: 3, rfaAfter: true }, paused: false, remaining: 0 } },
        };
      },
      pick: async () => { throw new Error('view-as mode cannot submit picks'); },
    };
    await act(async () => root!.render(<DraftRoom league="native.test" franchise="franchise.02"
      sessionFranchise="franchise.01" port={port} />));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
    expect(reads).toEqual([{ franchise: 'franchise.02' }]);
    expect(host.querySelector('.draft-your-pick')).toBeNull();
    expect(host.textContent).not.toContain('You missed round 1 pick 1. Pick any time.');
    await act(async () => root!.unmount());
    root = createRoot(host);
    await act(async () => root!.render(<DraftRoom league="native.test" franchise="franchise.01"
      sessionFranchise="franchise.01" port={port} />));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
    expect(host.querySelector('.draft-your-pick')).toBeNull();
  });

it('hides the prior league draft before the next league read settles, even for the same franchise id',
  async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    const oldDraft = {
      league: 'native.old', season: 2026, revision: 3, rules: [],
      owners: [{ id: 'franchise.01', name: 'Owner' }], order: [], words: 'Draft open',
      nextOwnedPick: null, closed: null, players: {},
      available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }], checkpoints: [],
      draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: true,
        picks: [{ round: 1, original: 'franchise.01', owner: 'franchise.01', slot: 1, extra: 0,
          player: '', skipped: true, forfeited: false, madeAt: '',
          contract: { salaryCents: 900000, years: 3, rfaAfter: true }, how: 'expired', changed: false }],
        onClock: null },
    };
    const port: DraftRoomPort = {
      read: async scope => scope.league === 'native.old' ? oldDraft : new Promise(() => {}),
      pick: async () => { throw new Error('the previous league must not be submitted'); },
    };
    await act(async () => root!.render(<DraftRoom league="native.old" franchise="franchise.01"
      sessionFranchise="franchise.01" port={port} />));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
    expect(host.querySelector('.draft-your-pick')).not.toBeNull();
    await act(async () => root!.render(<DraftRoom league="native.new" franchise="franchise.01"
      sessionFranchise="franchise.01" port={port} />));
    expect(host.querySelector('.draft-your-pick')).toBeNull();
    expect(host.textContent).not.toContain('You missed round 1 pick 1. Pick any time.');
  });

it('does not fall back to a stale legacy league or owner for an explicitly empty session', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const reads: { league: string; franchise: string }[] = [];
  const port: DraftRoomPort = {
    read: async scope => {
      reads.push({ league: scope.league, franchise: scope.franchise });
      return { league: 'native.target', season: 2026, revision: 2, rules: [], owners: [], order: [],
        words: 'Draft open', nextOwnedPick: null, closed: null, players: {},
        available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }], checkpoints: [],
        draft: { start: '2026-05-01T00:00:00Z', started: true, exhausted: true,
          picks: [{ round: 1, original: 'franchise.01', owner: 'franchise.01', slot: 1, extra: 0,
            player: '', skipped: true, forfeited: false, madeAt: '',
            contract: { salaryCents: 900000, years: 3, rfaAfter: true }, how: 'expired', changed: false }],
          onClock: null },
      };
    },
    pick: async () => { throw new Error('an empty session cannot submit'); },
  };
  await act(async () => root!.render(<DraftRoom league="native.target" franchise="franchise.01"
    sessionFranchise="" port={port} mayChangeRules={false} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(reads).toEqual([{ league: 'native.target', franchise: 'franchise.01' }]);
  expect(host.querySelector('.draft-your-pick')).toBeNull();
  expect(host.textContent).not.toContain('You missed round 1 pick 1. Pick any time.');
});
