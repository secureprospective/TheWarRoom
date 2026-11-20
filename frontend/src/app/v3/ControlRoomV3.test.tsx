// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import { hostedReads, observeView } from '../data/hostedReads';
import { createHostedLeagueRoom, type HostedPort } from '../commissioner/hostedRoom';
import type { HostedControlRoomState } from '../commissioner/HostedControlRoom';
import type { People } from '../commissioner/people';
import ControlRoomV3 from './ControlRoomV3';
import { CtrlLeagueV3 } from './CtrlLeagueV3';
import { CtrlPeopleV3 } from './CtrlPeopleV3';
import { CtrlSeasonV3 } from './CtrlSeasonV3';

const league = 'native.test';

const peopleFixture = (ownerTeam = 'Arizona Cardinals'): People => ({
  league,
  name: 'Test League',
  revision: 7,
  seats: [
    { franchise: 'f2', name: 'Detroit Lions', abbreviation: 'DET', owner: '', caretaker: 'acct.az' },
    { franchise: 'f1', name: ownerTeam, abbreviation: 'ARI', owner: 'acct.az' },
  ],
  roles: [{ person: 'acct.az', role: 'Commissioner' }],
  invites: [],
  dotCount: 0,
  dotStatus: 'No DOT seats',
  votes: [],
  succession: 'Commissioner active',
  actors: [{ person: 'acct.az', effectivePermissions: ['edit-lineups'] }],
  yourPermissions: ['edit-lineups'],
  yourRole: 'Commissioner',
  yourFranchise: 'f1',
  mayChangeRules: true,
  mayGrantRoles: true,
  mayPostNotices: true,
});

const bracketFixture = {
  league, name: 'Test League', season: 2026, revision: 7, status: 'Projected',
  words: 'Playoff bracket is not set.', preset: '', presets: [], started: false, startKnown: true,
  fields: [], projected: [], rounds: [], champion: null, title: {}, history: [], changes: [],
  rules: { words: '', perConference: 2, divisionWinners: 0, wildCards: 0, byeSeeds: 0,
    seeding: 'rules', reseed: false, startWeek: 15, finalWeek: 17, homeBonus: '0', shape: [] },
};

let host: HTMLDivElement;
let root: Root;
let authGeneration = 0;

beforeEach(() => {
  hostedReads.endSession();
  authGeneration++;
  observeView({
    authGeneration: String(authGeneration), server: 'https://league.test', league,
    message: '', name: 'Commissioner', role: 'Commissioner',
    franchise: 'f1', franchiseName: 'Arizona Cardinals',
    seed: '', connection: '', helperId: '', confirmation: '', seats: [],
  });
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});

afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});

function stubRoomState(port: HostedPort): HostedControlRoomState {
  const room = createHostedLeagueRoom(league, port);
  return {
    room,
    roomState: room.use(),
    jobsState: { pending: false, error: '' },
    fixState: { pending: false, error: '' },
    tab: 'people',
    definition: undefined,
    openedTab: undefined,
    initialValues: undefined,
    job: undefined,
    routeWords: '',
    feedbackRefreshVersion: 0,
    jobSnoozes: {},
    snoozeError: '',
    snoozePending: '',
    jobNow: vi.fn(),
    snoozeJob: vi.fn(),
    wakeJob: vi.fn(),
    openTool: vi.fn(),
    openSeasonTool: vi.fn(),
    showRoute: vi.fn(),
    navigateTo: vi.fn(),
    jobCount: 0,
    refreshAll: vi.fn(),
  } as unknown as HostedControlRoomState;
}

describe('ControlRoomV3 rendered tabs', () => {
  it('renders member names in People deputies/roles, plain copy, refresh via shell', async () => {
    const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
      throw new Error(`Unexpected read ${request.command}`);
    });
    const refreshPeople = vi.fn();
    const Wrapper = () => (
      <CtrlPeopleV3
        c={stubRoomState(port)}
        league={league}
        port={port}
        badge={null}
        people={peopleFixture()}
        peoplePending={false}
        peopleError=""
        refreshPeople={refreshPeople}
      />
    );
    await act(async () => root.render(<Wrapper />));
    expect(host.textContent).toContain('Arizona Cardinals');
    expect(host.textContent).toContain('Quiet / last-seen and staff filters aren’t available yet.');
    expect(host.textContent).not.toContain('commish.people.read');
    expect(port).not.toHaveBeenCalled();

    const refreshBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === 'Refresh people');
    expect(refreshBtn).toBeDefined();
    await act(async () => refreshBtn!.click());
    expect(refreshPeople).toHaveBeenCalledTimes(1);
  });

  it('renders member names in the League ledger and notices, with plain newsletter copy', async () => {
    const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
      if (request.command === 'commish.ledger.read' || request.command === 'commish.ledger.filtered.read') {
        return {
          data: {
            league, name: 'Test League', revision: 7,
            entries: [{
              id: 'e1', who: 'acct.az', what: 'Scoring updated', why: 'Owner vote',
              when: '2026-11-17T12:00:00Z', gravity: 'G1', undo: '', replacedAt: '',
            }],
          },
          label: 'League server', writable: true, football: 'held-pin',
        };
      }
      if (request.command === 'commish.notices.read') {
        return {
          data: {
            league, revision: 7,
            notices: [{
              id: 'n1', kind: 'notice', text: 'Draft moved', pinned: false,
              who: 'acct.az', when: '2026-11-17T12:00:00Z', withdrawn: false, change: '',
            }],
            notes: [], releases: [], mail: { words: '', recorded: [] },
          },
          label: 'League server', writable: true, football: 'held-pin',
        };
      }
      throw new Error(`Unexpected read ${request.command}`);
    });

    const Wrapper = () => (
      <CtrlLeagueV3
        c={stubRoomState(port)}
        league={league}
        port={port}
        badge={null}
        people={peopleFixture()}
      />
    );
    await act(async () => root.render(<Wrapper />));
    await vi.waitFor(() => expect(host.textContent).toContain('Scoring updated'));
    expect(host.textContent).toContain('Arizona Cardinals · Owner vote');
    expect(host.textContent).not.toContain('acct.az');
    expect(host.textContent).not.toContain('not on the server app yet');

    const newsletterBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === 'NEWSLETTER');
    expect(newsletterBtn).toBeDefined();
    await act(async () => newsletterBtn!.click());
    expect(host.textContent).toContain('Publishing the newsletter isn’t available yet.');
    expect(host.textContent).not.toContain('hosted command');
  });

  it('renders Contracts as the first Season calendar window row when opens is empty', async () => {
    const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
      if (request.command === 'native.directory.read') {
        return {
          data: {
            league, name: 'Test League', revision: 7, rosterSeason: 2026, source: 'held',
            franchises: [{ franchise: 'f1', name: 'Arizona Cardinals', abbreviation: 'ARI' }],
          },
          label: 'Server', writable: true, football: 'held',
        };
      }
      if (request.command === 'commish.schedule.read') {
        return {
          data: { league, revision: 7, draft: [], published: [] },
          label: 'Server', writable: true, football: 'held',
        };
      }
      if (request.command === 'bracket.read') {
        return { data: bracketFixture, label: 'Server', writable: true, football: 'held' };
      }
      if (request.command === 'calendar.read') {
        return {
          data: {
            league, revision: 7, decision: {},
            clock: {
              windows: [
                { id: 'contracts', opens: [], start: '2026-03-01T00:00:00Z', live: true },
                { id: 'lineups', opens: ['lineup'], start: '2026-09-01T00:00:00Z', live: false },
              ],
              exceptions: [], tradeDeadline: null, secondsToTradeDeadline: null,
            },
          },
          label: 'Server', writable: true, football: 'held',
        };
      }
      throw new Error(`Unexpected read ${request.command}`);
    });

    const Wrapper = () => (
      <CtrlSeasonV3 c={stubRoomState(port)} league={league} port={port} badge={null} />
    );
    await act(async () => root.render(<Wrapper />));
    await vi.waitFor(() => expect(host.textContent).toContain('ACTION WINDOWS · 2'));
    const firstWindowRow = host.querySelector('.row b');
    expect(firstWindowRow?.textContent).toBe('Contracts');
  });

  it('uses one shared people query and propagates Refresh people to other tabs', async () => {
    let teamName = 'Arizona Cardinals';
    const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
      if (request.command === 'commish.rules.read') {
        return {
          data: { league, name: 'Test League', revision: 7, rollover: '', sections: [] },
          label: 'Server', writable: true, football: 'held-pin',
        };
      }
      if (request.command === 'commish.jobs.read') {
        return {
          data: { league, name: 'Test League', revision: 7, jobs: [] },
          label: 'Server', writable: true, football: 'held-pin',
        };
      }
      if (request.command === 'commish.people.read') {
        return { data: peopleFixture(teamName), label: 'Server', writable: true, football: 'held-pin' };
      }
      if (request.command === 'commish.ledger.read' || request.command === 'commish.ledger.filtered.read') {
        return {
          data: {
            league, name: 'Test League', revision: 7,
            entries: [{
              id: 'e1', who: 'acct.az', what: 'Scoring updated', why: '',
              when: '2026-11-17T12:00:00Z', gravity: 'G1', undo: '', replacedAt: '',
            }],
          },
          label: 'Server', writable: true, football: 'held-pin',
        };
      }
      if (request.command === 'commish.notices.read') {
        return {
          data: {
            league, revision: 7, notices: [], notes: [], releases: [],
            mail: { words: '', recorded: [] },
          },
          label: 'Server', writable: true, football: 'held-pin',
        };
      }
      throw new Error(`Unexpected read ${request.command}`);
    });

    const setTab = vi.fn();
    await act(async () => root.render(
      <ControlRoomV3 league={league} port={port} tab="people" setTab={setTab} badge={null} />,
    ));
    await vi.waitFor(() => expect(host.textContent).toContain('Arizona Cardinals'));
    const peopleCalls = () => port.mock.calls.filter(([req]) => req.command === 'commish.people.read').length;
    expect(peopleCalls()).toBe(1);

    teamName = 'Desert Birds';
    const refreshBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === 'Refresh people');
    await act(async () => refreshBtn!.click());
    await vi.waitFor(() => expect(host.textContent).toContain('Desert Birds'));
    expect(peopleCalls()).toBe(2);

    await act(async () => root.render(
      <ControlRoomV3 league={league} port={port} tab="league" setTab={setTab} badge={null} />,
    ));
    await vi.waitFor(() => expect(host.textContent).toContain('Scoring updated'));
    expect(host.textContent).toContain('Desert Birds');
  });
});
