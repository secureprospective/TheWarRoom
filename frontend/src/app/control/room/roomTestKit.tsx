// The Control Room's region tests share one headless room on the Go-written wire fixtures.
import { afterEach, beforeEach, vi } from 'vitest';

import { all, click, mount, render, settle, unmount, type Element } from '../../commissioner/commissionerDom';
import { open, stub } from '../../commissioner/commissionerStub';
import { leagueState } from '../../commissioner/proxy';
import jobsWire from '../../commissioner/jobs.wire.testdata.json';
import rulesWire from '../../commissioner/rules.wire.testdata.json';
import ControlRoom from './ControlRoom';
import { type RoomData } from './roomReads';

export const league = rulesWire.league;
const jobs = { ...jobsWire.before, league };
export const franchises = [
  { franchise: 'franchise.01', name: 'Held Owls' },
  { franchise: 'franchise.02', name: 'Held Bison' },
];
export const data: RoomData = {
  errors: {},
  directory: {
    league,
    name: 'Rules league',
    revision: 3,
    rosterSeason: 2026,
    source: 'native',
    franchises,
  } as never,
  assets: {
    league,
    revision: 3,
    seeded: true,
    entry: '',
    source: '',
    snapshot: {
      season: 2026,
      picks: [
        {
          season: 2027,
          round: 1,
          slot: 1,
          original: 'franchise.01',
          owner: 'franchise.01',
          player: '',
          skipped: false,
        },
        {
          season: 2027,
          round: 1,
          slot: 2,
          original: 'franchise.02',
          owner: 'franchise.01',
          player: '',
          skipped: false,
        },
      ],
    },
  } as never,
  waivers: {
    league,
    revision: 3,
    windows: [],
    claims: [{ id: 'c1', words: 'K. Shaheed · Held Owls #3' }],
    players: [],
    bids: [],
    trades: [],
  } as never,
  trades: {
    league,
    revision: 3,
    windows: [],
    claims: [],
    players: [],
    bids: [],
    trades: [{ id: 't1', words: 'Held Owls ⇄ Held Bison · 2 of 3 approvals' }],
  } as never,
  statQueue: {
    league,
    season: 2026,
    revision: 3,
    words: '',
    decided: [],
    open: [
      {
        id: 's1',
        player: '1',
        name: 'J. Kincaid',
        season: 2026,
        week: 8,
        changes: [{ stat: 'ry', label: 'Rushing yards', before: '61', after: '64' }],
        flip: '',
        flipWords: '',
        words: 'Held Owls +0.30 · no result changes',
        arrived: '',
        sha256: '',
      },
    ],
  } as never,
  calendar: {
    league,
    revision: 3,
    decision: {},
    clock: { windows: [], exceptions: [], tradeDeadline: '2026-11-08T18:00:00Z' },
  } as never,
  corrections: {
    league,
    revision: 3,
    items: [
      {
        entry: 'act.x',
        who: 'creator',
        when: '2026-10-20T00:00:00Z',
        why: 'feed',
        what: 'Exception: roster over the maximum',
        franchise: 'franchise.02',
        until: '2026-11-08T18:00:00Z',
      },
    ],
  } as never,
};

/** Each region test file calls this once. */
export function roomHooks() {
  beforeEach(() => leagueState.forget());
  afterEach(async () => {
    await unmount();
    vi.useRealTimers();
  });
}

export async function room(
  over: Parameters<typeof stub>[0] = {},
  readings: Parameters<typeof stub>[1] = {},
  initial: RoomData = data,
) {
  const s = stub({ TargetCommissionerJobs: async () => jobs, ...over }, readings);
  const host = await mount(render(ControlRoom, { league, initial }), s.app);
  await open(league);
  await settle();
  return { host, ...s };
}
export const tab = (host: Element, label: string) =>
  click(all(host, 'button').find((b) => b.textContent.startsWith(label))!);
export const region = (host: Element, label: string) =>
  all(host, 'div').find((d) => d.getAttribute('role') === 'region' && d.getAttribute('aria-label') === label);
