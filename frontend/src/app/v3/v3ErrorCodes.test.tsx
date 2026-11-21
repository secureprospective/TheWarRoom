// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import { createHostedLeagueRoom, type HostedPort } from '../commissioner/hostedRoom';
import type { HostedControlRoomState } from '../commissioner/HostedControlRoom';
import { createFailureError, resetFailureStateForTest } from '../data/errorCodes';
import { hostedReads, observeView } from '../data/hostedReads';
import type { WarRoomContext } from '../shell/warroomReads';
import { CtrlFeedbackV3 } from './CtrlFeedbackV3';
import { CtrlFixV3 } from './CtrlFixV3';
import { CtrlJobsV3 } from './CtrlJobsV3';
import { CtrlLeagueV3 } from './CtrlLeagueV3';
import { CtrlPeopleV3 } from './CtrlPeopleV3';
import { CtrlPicksV3 } from './CtrlPicksV3';
import { CtrlRulesV3 } from './CtrlRulesV3';
import { CtrlSeasonV3 } from './CtrlSeasonV3';
import { CtrlServerV3 } from './CtrlServerV3';
import V3Shell from './V3Shell';
import { WarDraftV3 } from './WarDraftV3';
import { WarPoolV3 } from './WarPoolV3';

vi.mock('../shell/warroomMarketReads', async importOriginal => {
  const actual = await importOriginal<typeof import('../shell/warroomMarketReads')>();
  return {
    ...actual,
    readWarRoomMarket: vi.fn(async () => ({
      players: [{
        id: '1234', name: 'Ace Runner', position: 'RB', freeAgent: true, waiverStatus: '',
        minimumCents: 100000000, minimum: '$1.00M',
      }],
      bids: [],
      wins: [],
      bidRevision: 7,
      capRoom: 500000000,
      writable: true,
      football: 'held',
      maxYears: 4,
      rosterRevision: 7,
      openBenchSlots: 2,
      rosterPlayers: [],
      waiverPriority: { place: 1, total: 12, source: 'standings' },
      errors: [],
    })),
    reviewWarRoomRosterAdd: vi.fn(async () => {
      throw createFailureError('Roster add refused by server.', 'TWR-PL23-4567');
    }),
  };
});

vi.mock('../shell/warroomDraftReads', async importOriginal => {
  const actual = await importOriginal<typeof import('../shell/warroomDraftReads')>();
  return {
    ...actual,
    readWarRoomDraft: vi.fn(async () => ({
      draft: {
        league, season: 2026, words: 'Live', started: true, exhausted: false, start: '',
        onClock: null, owners: [{ id: 'f1', name: 'Arizona Cardinals' }], order: [], picks: [],
        available: [{ id: '1234', name: 'Ace Runner', position: 'RB', team: 'ARI' }],
        players: { '1234': { id: '1234', name: 'Ace Runner', position: 'RB', team: 'ARI' } },
      },
      queue: [],
    })),
    saveWarRoomDraftQueue: vi.fn(async () => {
      throw createFailureError('Queue rejected by server.', 'TWR-DRFT-2345');
    }),
  };
});

const league = 'native.test';

let host: HTMLDivElement;
let root: Root;
let authGen = 0;

beforeEach(() => {
  hostedReads.endSession();
  resetFailureStateForTest();
  authGen++;
  observeView({
    authGeneration: String(authGen), server: 'https://league.test', league,
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
  resetFailureStateForTest();
  vi.restoreAllMocks();
});

function expectFailureAlert(code: string, container: ParentNode = host): void {
  const codes = [...container.querySelectorAll('[data-testid="failure-code"]')]
    .map(el => el.textContent);
  expect(codes).toContain(code);
  const copyBtn = [...container.querySelectorAll('button')]
    .find(btn => btn.textContent?.trim() === 'Copy');
  expect(copyBtn).toBeTruthy();
}

function stubControlRoom(
  port: HostedPort,
  overrides: Partial<HostedControlRoomState> = {},
): HostedControlRoomState {
  const room = createHostedLeagueRoom(league, port);
  return {
    room,
    roomState: {
      pending: false, error: '', errorCode: undefined, stale: false, message: '',
    },
    jobsState: { pending: false, error: '', errorCode: undefined },
    fixState: { pending: false, error: '', errorCode: undefined },
    tab: 'jobs',
    definition: undefined,
    openedTab: undefined,
    initialValues: undefined,
    job: undefined,
    routeWords: '',
    feedbackRefreshVersion: 0,
    jobSnoozes: {},
    snoozeError: '',
    snoozeErrorCode: undefined,
    snoozePending: '',
    jobNow: Date.now(),
    snoozeJob: vi.fn(),
    wakeJob: vi.fn(),
    openTool: vi.fn(),
    openSeasonTool: vi.fn(),
    showRoute: vi.fn(),
    navigateTo: vi.fn(),
    jobCount: 0,
    refreshAll: vi.fn(),
    ...overrides,
  } as unknown as HostedControlRoomState;
}

const noopPort: HostedPort = async (req: NativeRequest) => {
  throw new Error(`Unexpected read ${req.command}`);
};

const failingPort = (message: string, code: string): HostedPort =>
  vi.fn<HostedPort>(async () => { throw createFailureError(message, code); });

describe('v3 error codes with Copy button', () => {
  it('renders FailureAlert with code and Copy across Control Room v3 tabs', async () => {
    const cJobs = stubControlRoom(noopPort, {
      jobsState: { pending: false, error: 'Jobs read failed.', errorCode: 'TWR-JBS2-3456' },
      snoozeError: 'Snooze save failed.',
      snoozeErrorCode: 'TWR-SNZ2-3456',
    });
    await act(async () => root.render(
      <CtrlJobsV3 c={cJobs} league={league} port={noopPort} badge={null} />,
    ));
    expectFailureAlert('TWR-JBS2-3456');
    expectFailureAlert('TWR-SNZ2-3456');

    const cRules = stubControlRoom(noopPort, {
      roomState: {
        pending: false, error: 'Rules read failed.', errorCode: 'TWR-RLS2-3456',
        stale: false, message: '',
      },
    } as Partial<HostedControlRoomState>);
    await act(async () => root.render(
      <CtrlRulesV3 c={cRules} league={league} port={noopPort} badge={null} />,
    ));
    expectFailureAlert('TWR-RLS2-3456');

    const cFix = stubControlRoom(noopPort, {
      fixState: { pending: false, error: 'Fix read failed.', errorCode: 'TWR-FX23-4567' },
    });
    await act(async () => root.render(
      <CtrlFixV3 c={cFix} league={league} port={noopPort} badge={null} />,
    ));
    expectFailureAlert('TWR-FX23-4567');

    await act(async () => root.render(
      <CtrlPeopleV3 c={stubControlRoom(noopPort)} league={league} port={noopPort} badge={null}
        peopleError="People unavailable." peopleErrorCode="TWR-PPL2-3456" />,
    ));
    expectFailureAlert('TWR-PPL2-3456');

    const fbPort = failingPort('Feedback read failed.', 'TWR-FDBK-2345');
    await act(async () => root.render(
      <CtrlFeedbackV3 c={stubControlRoom(fbPort)} league={league} port={fbPort} badge={null} />,
    ));
    await vi.waitFor(() => expectFailureAlert('TWR-FDBK-2345'));

    const leaguePort = failingPort('League ledger failed.', 'TWR-LGR2-3456');
    await act(async () => root.render(
      <CtrlLeagueV3 c={stubControlRoom(leaguePort)} league={league} port={leaguePort}
        badge={null} />,
    ));
    await vi.waitFor(() => expectFailureAlert('TWR-LGR2-3456'));

    const picksPort = failingPort('Assets read failed.', 'TWR-PCKS-2345');
    await act(async () => root.render(
      <CtrlPicksV3 c={stubControlRoom(picksPort)} league={league} port={picksPort} badge={null} />,
    ));
    await vi.waitFor(() => expectFailureAlert('TWR-PCKS-2345'));

    const seasonPort = failingPort('Season read failed.', 'TWR-SSN2-3456');
    await act(async () => root.render(
      <CtrlSeasonV3 c={stubControlRoom(seasonPort)} league={league} port={seasonPort}
        badge={null} />,
    ));
    await vi.waitFor(() => expectFailureAlert('TWR-SSN2-3456'));

    const serverPort = failingPort('Server ledger failed.', 'TWR-SRV2-3456');
    await act(async () => root.render(
      <CtrlServerV3 c={stubControlRoom(serverPort)} league={league} port={serverPort}
        badge={null} />,
    ));
    await vi.waitFor(() => expectFailureAlert('TWR-SRV2-3456'));
  });

  it('renders FailureAlert with code and Copy on WarDraftV3 queue save failure', async () => {
    const context = {
      snapshot: {}, mine: 'f1', week: 1,
      native: { league, mine: 'f1', port: noopPort, leave: vi.fn(), owner: true },
    } as unknown as WarRoomContext;

    await act(async () => root.render(<WarDraftV3 context={context} badge={null} />));
    await vi.waitFor(() => expect(host.textContent).toContain('Ace Runner'));
    const queueBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === '+ Queue');
    expect(queueBtn).toBeTruthy();
    await act(async () => { queueBtn!.click(); });
    await vi.waitFor(() => expectFailureAlert('TWR-DRFT-2345'));
  });

  it('renders FailureAlert with code and Copy on WarPoolV3 action failure', async () => {
    const context = {
      snapshot: {}, mine: 'f1', week: 1,
      native: { league, mine: 'f1', port: noopPort, leave: vi.fn(), owner: true },
    } as unknown as WarRoomContext;

    await act(async () => root.render(<WarPoolV3 context={context} badge={null} />));
    await vi.waitFor(() => expect(host.textContent).toContain('Ace Runner'));
    const addBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === 'Add');
    expect(addBtn).toBeTruthy();
    await act(async () => { addBtn!.click(); });

    const reasonInput = host.querySelector<HTMLInputElement>('div[aria-label="Add free agent"] input');
    expect(reasonInput).toBeTruthy();
    const setValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
    await act(async () => {
      setValue?.call(reasonInput, 'Need RB depth');
      reasonInput!.dispatchEvent(new Event('input', { bubbles: true }));
    });
    const reviewBtn = [...host.querySelectorAll('button')]
      .find(btn => btn.textContent?.trim() === 'Review add');
    expect(reviewBtn).toBeTruthy();
    await act(async () => { reviewBtn!.click(); });
    await vi.waitFor(() => expectFailureAlert('TWR-PL23-4567'));
  });

  it('renders FailureAlert with code and Copy in V3Shell footer status', async () => {
    await act(async () => root.render(
      <V3Shell
        leagues={[]}
        onOpen={vi.fn()}
        onHome={vi.fn()}
        onSignOut={vi.fn()}
        port={noopPort}
        status="Session check failed."
        statusCode="TWR-SHLL-9876"
      />,
    ));
    const footer = host.querySelector('footer.status');
    expect(footer).toBeTruthy();
    expectFailureAlert('TWR-SHLL-9876', footer!);
  });
});
