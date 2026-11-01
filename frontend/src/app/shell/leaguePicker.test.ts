import { describe, expect, it } from 'vitest';
import {
  LEAGUE_KEY,
  chooseLeague,
  parseLeagues,
  resolveLeagueId,
  type LeagueChoice,
} from './leaguePicker';

const leagues: LeagueChoice[] = [
  { id: 'mirror', name: 'Legacy NFL', teams: 32, app: false },
  { id: 'native.one', name: 'One', teams: 12, app: true },
  { id: 'native.two', name: 'Two', teams: 14, app: true },
];

function fakeStorage(initial?: string) {
  const values = new Map<string, string>();
  if (initial !== undefined) values.set(LEAGUE_KEY, initial);
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
  };
}

describe('League picker selection', () => {
  it('restores the stored league when it is still available', () => {
    const storage = fakeStorage('native.two');
    expect(resolveLeagueId(leagues, storage.getItem(LEAGUE_KEY))).toBe('native.two');
  });

  it('falls back to the first league when nothing is stored', () => {
    expect(resolveLeagueId(leagues, fakeStorage().getItem(LEAGUE_KEY))).toBe('mirror');
  });

  it('falls back to the first league when the stored league was deleted', () => {
    expect(resolveLeagueId(leagues, 'native.deleted')).toBe('mirror');
  });

  it('accepts an empty league list without inventing a league', () => {
    expect(parseLeagues([])).toEqual([]);
    expect(resolveLeagueId([], 'native.deleted')).toBe('');
  });

  it('persists a pick and resolves it back after a relaunch', () => {
    const storage = fakeStorage();
    expect(chooseLeague(leagues, 'native.two', storage)).toBe('native.two');
    expect(storage.getItem(LEAGUE_KEY)).toBe('native.two');
    expect(resolveLeagueId(leagues, storage.getItem(LEAGUE_KEY))).toBe('native.two');
  });

  it('refuses an unknown league without writing it', () => {
    const storage = fakeStorage('mirror');
    expect(() => chooseLeague(leagues, 'native.deleted', storage)).toThrow('unknown league');
    expect(storage.getItem(LEAGUE_KEY)).toBe('mirror');
  });

  it('parses the desktop list and rejects an unreadable authority bit', () => {
    expect(parseLeagues(leagues)).toEqual(leagues);
    expect(() => parseLeagues([{ id: 'mirror', name: 'Legacy NFL', teams: 32, app: 'no' }])).toThrow(
      'expected boolean',
    );
  });
});

it('renders the Home choices as visible actions', async () => {
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { createElement } = await import('react');
  const { LeagueChoices } = await import('./leaguePicker');
  const html = renderToStaticMarkup(createElement(LeagueChoices, {
    leagues, selectedId: 'native.one', onSelect: () => {},
  }));
  expect(html).toContain('<button');
  expect(html).toContain('One — 12 teams');
  expect(html).not.toContain('<select');
});

it('uses the authorizer’s writable status rather than inferring rights from a franchise choice', async () => {
  const { vi } = await import('vitest');
  const api = await import('../commissioner/api');
  const { localWorkflow } = await import('./localWorkflow');
  vi.stubGlobal('window', { go: {} });
  const authority = vi.fn().mockResolvedValue({ label: 'Local league', writable: false,
    football: '', data: {} });
  const spy = vi.spyOn(api, 'nativeBindings').mockReturnValue({ TargetWorkflow: authority } as never);
  const request = { command: 'native.assets.read', mode: 'read',
    arguments: { league: 'native.one' } } as const;
  try {
    expect((await localWorkflow(request)).writable).toBe(false);
    authority.mockResolvedValue({ label: 'Local league', writable: true, football: '', data: {} });
    expect((await localWorkflow(request)).writable).toBe(true);
    authority.mockRejectedValue(new Error('Not authorized'));
    await expect(localWorkflow(request)).rejects.toThrow('Not authorized');
  } finally {
    spy.mockRestore();
    vi.unstubAllGlobals();
  }
});

it('uses only the signed-in person’s franchise, including an empty seat', async () => {
  const { vi } = await import('vitest');
  const api = await import('../commissioner/api');
  const { readAppLeaguePeople } = await import('./leaguePicker');
  const people = {
    league: 'native.one', name: 'One', revision: 1,
    seats: [{ franchise: 'other', name: 'Other', abbreviation: 'O', owner: 'someone' }],
    roles: [], invites: [], dotCount: 0, dotStatus: '', votes: [], succession: '',
    yourRole: 'Commissioner', yourFranchise: '', mayChangeRules: true, mayGrantRoles: true,
    mayPostNotices: true,
  };
  vi.stubGlobal('window', { go: {} });
  const read = vi.fn().mockResolvedValue(people);
  const spy = vi.spyOn(api, 'nativeBindings').mockReturnValue({ TargetCommissionerPeople: read } as never);
  try {
    expect((await readAppLeaguePeople('native.one')).yourFranchise).toBe('');
    read.mockResolvedValue({ ...people, yourFranchise: 'mine' });
    expect((await readAppLeaguePeople('native.one')).yourFranchise).toBe('mine');
  } finally {
    spy.mockRestore();
    vi.unstubAllGlobals();
  }
});

it('uses desktop native reads with league authority for regions outside the workflow catalog', async () => {
  const { vi } = await import('vitest');
  const api = await import('../commissioner/api');
  const { localWorkflow } = await import('./localWorkflow');
  vi.stubGlobal('window', { go: {} });
  const cap = vi.fn().mockResolvedValue({ league: 'native.one' });
  const authority = vi.fn().mockResolvedValue({ label: 'Local league', writable: true,
    football: '', data: {} });
  const spy = vi.spyOn(api, 'nativeBindings').mockReturnValue({ TargetNativeCapCheck: cap,
    TargetWorkflow: authority } as never);
  try {
    const result = await localWorkflow({ command: 'native.cap.check', mode: 'read',
      arguments: { league: 'native.one', viewerZone: 'UTC' } });
    expect(cap).toHaveBeenCalledWith('native.one', 'UTC');
    expect(JSON.parse(authority.mock.calls[0][0]).arguments.league).toBe('native.one');
    expect(result.data).toEqual({ league: 'native.one' });
  } finally {
    spy.mockRestore();
    vi.unstubAllGlobals();
  }
});

it('uses the selected league as the rail header without a static source label', async () => {
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { createElement } = await import('react');
  const { LeaguePicker } = await import('./leaguePicker');
  const html = renderToStaticMarkup(createElement(LeaguePicker, {
    leagues, selected: leagues[0], onSelect: () => {},
  }));
  expect(html).toContain('Legacy NFL');
  expect(html).not.toContain('From MFL (reference only)');
});

it('distinguishes same-named league choices by their available identity', async () => {
  const { renderToStaticMarkup } = await import('react-dom/server');
  const { createElement } = await import('react');
  const { LeagueChoices } = await import('./leaguePicker');
  const choices = [leagues[0], { ...leagues[1], name: 'Legacy NFL', teams: 32 }];
  const html = renderToStaticMarkup(createElement(LeagueChoices, {
    leagues: choices, selectedId: 'mirror', onSelect: () => {},
  }));
  expect(html).toContain('Synced from MFL');
  expect(html).toContain('native.one');
});
