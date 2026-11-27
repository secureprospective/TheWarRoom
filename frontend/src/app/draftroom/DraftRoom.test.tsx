// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import DraftRoom from './DraftRoom';
import type { DraftRoomPort } from './state';

const defaultLegacySession = () => ({
  choices: [] as { id: string; name: string }[],
  selected: 'native.stale',
  people: {
    yourFranchise: 'franchise.01',
    seats: [{ franchise: 'franchise.01', name: 'Owner' }],
    mayChangeRules: false,
  },
  outcome: { status: 'idle' } as { status: 'idle' } | {
    status: 'done'; form: string;
    receipt: { entry: string; league: string; revision: number; recordedAt: string; message: string };
  },
  error: '',
  pending: false,
});

const legacySession = vi.hoisted(() => ({
  value: {
    choices: [] as { id: string; name: string }[],
    selected: 'native.stale',
    people: {
      yourFranchise: 'franchise.01',
      seats: [{ franchise: 'franchise.01', name: 'Owner' }],
      mayChangeRules: false,
    },
    outcome: { status: 'idle' } as { status: 'idle' } | {
      status: 'done'; form: string;
      receipt: { entry: string; league: string; revision: number; recordedAt: string; message: string };
    },
    error: '',
    pending: false,
  },
}));
vi.mock('../commissioner/proxy', () => ({
  refusalText: (cause: unknown) => cause instanceof Error ? cause.message : String(cause),
  leagueState: { use: () => legacySession.value, choose: vi.fn() },
}));

let root: Root | undefined;
let host: HTMLDivElement | undefined;
beforeEach(() => {
  legacySession.value = defaultLegacySession();
});
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

function sampleDraft(league: string, words: string, owner = 'franchise.01') {
  const contract = { salaryCents: 900000, years: 3, rfaAfter: true };
  return {
    league, season: 2026, revision: 2, rules: [],
    owners: [{ id: 'franchise.01', name: 'Owner 1' }, { id: 'franchise.02', name: 'Owner 2' }],
    order: [], words, nextOwnedPick: null, closed: null, players: {},
    available: [{ id: '1001', name: 'Rookie One', position: 'RB', team: 'KC' }],
    checkpoints: [
      { entry: 'c1', words: 'Start', at: '2026-05-01T00:00:00Z' },
      { entry: 'c2', words: 'Pick 1', at: '2026-05-01T00:01:00Z' },
    ],
    draft: {
      start: '2026-05-01T00:00:00Z', started: true, exhausted: false,
      picks: [{ round: 1, original: owner, owner, slot: 1, extra: 0,
        player: '', skipped: false, forfeited: false, madeAt: '', contract,
        how: '', changed: false }],
      onClock: { round: 1, original: owner, owner, slot: 1,
        openedAt: '2026-05-01T00:00:00Z', deadline: '2099-05-01T00:10:00Z',
        contract, paused: false, remaining: 0 },
    },
  };
}

it('restarts draft polling with the new client when port changes on a mounted DraftRoom', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const readsA: string[] = [];
  const readsB: string[] = [];
  const portA: DraftRoomPort = {
    read: async scope => {
      readsA.push(scope.league);
      return sampleDraft(scope.league, 'Draft from port A');
    },
    pick: async () => ({}),
  };
  const portB: DraftRoomPort = {
    read: async scope => {
      readsB.push(scope.league);
      return sampleDraft(scope.league, 'Draft from port B');
    },
    pick: async () => ({}),
  };
  await act(async () => root!.render(<DraftRoom league="native.test" franchise="franchise.01"
    sessionFranchise="franchise.01" port={portA} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(readsA.length).toBe(1);
  expect(host.textContent).toContain('Draft from port A');

  await act(async () => root!.render(<DraftRoom league="native.test" franchise="franchise.01"
    sessionFranchise="franchise.01" port={portB} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(readsB.length).toBe(1);
  expect(host.textContent).toContain('Draft from port B');
  expect(host.textContent).not.toContain('Reading the draft…');
});

it('ignores legacy leagueState people, errors, and outcomes in scoped hosted sessions', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  legacySession.value = {
    choices: [{ id: 'native.scoped', name: 'Legacy' }],
    selected: 'native.scoped',
    people: {
      yourFranchise: 'franchise.01',
      seats: [
        { franchise: 'franchise.01', name: 'Legacy 1' },
        { franchise: 'franchise.02', name: 'Legacy 2' },
      ],
      mayChangeRules: true,
    },
    outcome: {
      status: 'done',
      form: 'draft.skip',
      receipt: {
        entry: 'legacy.entry.1', league: 'native.scoped', revision: 9,
        recordedAt: '2026-05-01T00:00:00Z', message: 'Legacy draft receipt',
      },
    },
    error: 'Legacy desktop error',
    pending: false,
  };
  let resolveRead!: (value: unknown) => void;
  const port: DraftRoomPort = {
    read: () => new Promise(resolve => { resolveRead = resolve; }),
    pick: async () => ({}),
  };
  await act(async () => root!.render(<DraftRoom league="native.scoped" franchise="franchise.01"
    sessionFranchise="franchise.01" port={port} />));
  expect(host.textContent).toContain('Reading the draft…');
  expect(host.textContent).not.toContain('Legacy desktop error');

  await act(async () => { resolveRead(sampleDraft('native.scoped', 'Hosted draft ready')); });
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(host.textContent).toContain('Hosted draft ready');
  expect(host.textContent).not.toContain('Legacy draft receipt');
  expect(host.textContent).not.toContain('Franchise to view');
  expect(host.textContent).not.toContain('Skip this pick');
  expect(host.textContent).not.toContain('Close the draft');
});

it('clears the dropdown viewing franchise when viewFranchise or league changes', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  legacySession.value = {
    choices: [{ id: 'legacy.one', name: 'One' }, { id: 'legacy.two', name: 'Two' }],
    selected: 'legacy.one',
    people: {
      yourFranchise: 'franchise.01',
      seats: [
        { franchise: 'franchise.01', name: 'Team 1' },
        { franchise: 'franchise.02', name: 'Team 2' },
        { franchise: 'franchise.03', name: 'Team 3' },
      ],
      mayChangeRules: false,
    },
    outcome: { status: 'idle' },
    error: '',
    pending: false,
  };
  const reads: { league: string; franchise: string }[] = [];
  const port: DraftRoomPort = {
    read: async scope => {
      reads.push({ league: scope.league, franchise: scope.franchise });
      return sampleDraft(scope.league, `Draft ${scope.league}`, scope.franchise);
    },
    pick: async () => ({}),
  };
  await act(async () => root!.render(<DraftRoom league="legacy.one" franchise="franchise.01" port={port} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  const select = host.querySelector('select') as HTMLSelectElement;
  expect(select.value).toBe('franchise.01');

  await act(async () => {
    select.value = 'franchise.02';
    select.dispatchEvent(new Event('change', { bubbles: true }));
  });
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(reads[reads.length - 1]).toEqual({ league: 'legacy.one', franchise: 'franchise.02' });

  await act(async () => root!.render(<DraftRoom league="legacy.one" franchise="franchise.03" port={port} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(reads[reads.length - 1]).toEqual({ league: 'legacy.one', franchise: 'franchise.03' });
  expect((host.querySelector('select') as HTMLSelectElement).value).toBe('franchise.03');

  await act(async () => {
    const currentSelect = host!.querySelector('select') as HTMLSelectElement;
    currentSelect.value = 'franchise.02';
    currentSelect.dispatchEvent(new Event('change', { bubbles: true }));
  });
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(reads[reads.length - 1]).toEqual({ league: 'legacy.one', franchise: 'franchise.02' });

  legacySession.value = { ...legacySession.value, selected: 'legacy.two' };
  await act(async () => root!.render(<DraftRoom league="legacy.two" franchise="franchise.01" port={port} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 0)); });
  expect(reads[reads.length - 1]).toEqual({ league: 'legacy.two', franchise: 'franchise.01' });
  expect((host.querySelector('select') as HTMLSelectElement).value).toBe('franchise.01');
});
