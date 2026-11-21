// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { NativeHQ } from '../commissioner/hqReading';
import type { HostedPort } from '../commissioner/hostedRoom';
import { PlayerCardDrawer } from './PlayerCardDrawer';

const readHQ = vi.hoisted(() => ({ value: undefined as NativeHQ | undefined }));
vi.mock('../commissioner/hqReading', async original => ({
  ...(await original<typeof import('../commissioner/hqReading')>()),
  readNativeHQ: async (_l: string, _m: string, _p: unknown, publish?: (r: NativeHQ) => void) => {
    publish?.(readHQ.value!);
    return readHQ.value!;
  },
  readNativeCut: async () => ({
    player: '16162', revision: 1, season: 2026, deadNow: 450_000_000, reason: 'Cut penalty',
  }),
}));

const { default: NativeFranchiseHQ } = await import('../shell/NativeFranchiseHQ');

const provenance = {
  source: 'League ledger', kind: 'live' as const,
  freshness: { state: 'live' as const, fetchedAt: '2026-11-20T12:00:00Z', note: 'Recorded' },
};

function fixtureReading(): NativeHQ {
  return {
    revision: 1,
    errors: [],
    week: 11,
    record: '8-2',
    searchPlayers: [],
    settings: { positions: [{ code: 'RB', min: 1, max: 2 }] },
    snapshot: {
      league: { value: { season: 2026, franchiseCount: 12 }, provenance },
      franchises: { value: [{ id: 'f1', name: 'Motor City' }], provenance },
      rosters: {
        value: [{
          franchiseId: 'f1',
          players: [
            {
              id: '16162', salary: 1_350_000_000, yearsRemaining: 4,
              contractStatus: 'RFA', rosterStatus: 'ROSTER',
            },
            {
              id: '16200', salary: 200_000_000, yearsRemaining: 1,
              contractStatus: 'UFA', rosterStatus: 'ROSTER',
            },
          ],
        }],
        provenance,
      },
      players: {
        value: [
          {
            id: '16162', name: 'Jahmyr Gibbs', position: 'RB', team: 'DET',
            birthdate: Date.UTC(2002, 2, 20) / 1000,
          },
          { id: '16200', name: 'Expiring Back', position: 'RB', team: 'CHI' },
        ],
        provenance,
      },
      eligibility: { players: {}, feeds: { injuries: provenance, players: provenance, rosters: provenance } },
    },
    contracts: new Map([
      ['16162', [2026, 2027, 2028, 2029].map(season => ({ season, cents: 1_350_000_000 }))],
      ['16200', [{ season: 2026, cents: 200_000_000 }]],
    ]),
    projections: new Map([['16162', 19.4], ['16200', 8.2]]),
    thisWeek: {
      league: 'L', season: 2026, week: 11, revision: 1, words: '', games: [],
      scores: [{
        franchise: 'f1', name: 'Motor City', points: 24.8,
        starters: [{ player: '16162', name: 'Jahmyr Gibbs', points: 24.8, words: '' }],
      }],
    },
    playerHistory: {
      '16162': [12, 18.2, 22.5, 30, 15.4, 26.1].map((points, i) => ({ week: 5 + i, points })),
    },
    playerForm: {
      '16162': [18.2, 22.5, 30, 15.4, 26.1].map((points, i) => ({ week: 6 + i, points })),
    },
    values: {
      league: 'L', franchise: 'f1', season: 2026, preparing: false, words: '',
      players: [{
        player: '16162', available: true, nowPPG: 18.5, dynastyPPG: 21.2, words: 'Model value',
      }],
    },
  } as unknown as NativeHQ;
}

function makePort() {
  const saved: string[] = [];
  const port = vi.fn(async (request: { command: string; arguments: Record<string, unknown> }) => {
    if (request.command === 'player.usage.read') {
      return {
        label: 'L', writable: false, football: '',
        data: {
          snapShare: 0.72, targetShare: 0.16, carryShare: 0.62, weeksUsed: 4,
          opponent: 'MIN', allowedPoints: 18.4, allowedRank: 24, teams: 32,
          label: 'Soft', reason: 'Last 4 weeks',
        },
      };
    }
    if (request.command === 'nfl.player-status.read') {
      return {
        label: 'L', writable: false, football: '',
        data: {
          league: 'L', state: 'available', season: 2026, fetchedAt: '2026-11-20T12:00:00Z',
          players: [{
            player: '16162', designation: 'Questionable', bodyPart: 'Hamstring',
            expectedReturnWeek: 12,
          }],
        },
      };
    }
    if (request.command === 'player.news.read') {
      if (request.arguments.player === '16200') {
        return {
          label: 'L', writable: false, football: '',
          data: {
            league: 'L', player: '16200', espnId: '', bio: {}, items: [], asOf: '', stale: false,
            reason: 'No ESPN player mapping is on file for this player.',
          },
        };
      }
      return {
        label: 'L', writable: false, football: '',
        data: {
          league: 'L', player: String(request.arguments.player), espnId: '4429795',
          bio: {
            height: `5' 9"`, weight: '200 lbs', born: '2002-03-20', age: 24,
            college: 'Alabama', draft: '2023: Rd 1, Pk 12 (DET)',
            headshot: 'https://a.espncdn.com/i/headshots/nfl/players/full/4429795.png',
            team: 'DET', position: 'RB', status: 'Active',
          },
          items: [
            {
              id: '505001', headline: 'Gibbs explodes for two TDs in Lions win',
              description: 'Jahmyr Gibbs led Detroit with 142 scrimmage yards.',
              published: new Date(Date.now() - 5 * 86_400_000).toISOString(),
              url: 'https://www.espn.com/nfl/story/_/id/505001/gibbs-two-tds',
              source: 'ESPN', type: 'Story',
            },
          ],
          asOf: '2026-11-21T12:00:00Z', stale: false,
        },
      };
    }
    if (request.command === 'player.note.read') {
      const text = saved.at(-1) ?? 'Hold as core RB1';
      return {
        label: 'L', writable: true, football: '',
        data: {
          league: 'L', player: String(request.arguments.player), count: 1, watched: false,
          notes: [{
            kind: 'note', entity: 'player', id: String(request.arguments.player),
            text, at: '2026-11-19T10:00:00Z', order: 1,
          }],
        },
      };
    }
    if (request.command === 'player.note.set') {
      if (request.arguments.text) saved.push(String(request.arguments.text));
      return { label: 'L', writable: true, football: '', data: {} };
    }
    throw new Error(`Unexpected command ${request.command}`);
  });
  return { port: port as unknown as HostedPort, mock: port, saved };
}

let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  root = undefined;
  host?.remove();
  host = undefined;
  document.getElementById('layer')?.remove();
  vi.restoreAllMocks();
});

async function flush() {
  await act(async () => { await new Promise(r => setTimeout(r, 10)); });
}

describe('PlayerCardDrawer', () => {
  it('renders header, News, Stats (last five weeks + projection), contract, status, and note', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    const reading = fixtureReading();
    const { port } = makePort();
    host = document.createElement('div');
    document.body.append(host);
    root = createRoot(host);

    await act(async () => root!.render(
      <PlayerCardDrawer id="16162" reading={reading} league="L" mine="f1" port={port}
        cut={{ player: '16162', revision: 1, season: 2026, deadNow: 450_000_000, reason: '' }}
        close={() => {}} />,
    ));
    await flush();

    const dialog = host.querySelector('[role="dialog"]')!;
    expect(dialog).toBeTruthy();
    const text = dialog.textContent ?? '';
    expect(text).toContain('JAHMYR GIBBS');
    expect(text).toContain('DET · RB · Motor City');
    expect(text).toContain(`5' 9"`);
    expect(text).toContain('200 lbs');
    expect(text).toContain('Mar 20, 2002 (24)');
    expect(text).toContain('Alabama');
    expect(text).toContain('2023: Rd 1, Pk 12 (DET)');
    expect(text).toContain('$13.5M');
    expect(text).toContain('2029');
    expect(text).toContain('RFA');
    expect(text).toContain('Not available');
    expect(text).not.toContain('Career Stats');
    const photoImg = dialog.querySelector<HTMLImageElement>('.pcd-photo img');
    expect(photoImg?.getAttribute("src")).toBe("/v1/headshot/16162");

    // News tab (default) shows ESPN items (bold headline link in new tab, age, description, source tag)
    expect(text).toContain('Player news');
    expect(text).toContain('Gibbs explodes for two TDs in Lions win');
    expect(text).toContain('5 days ago');
    expect(text).toContain('Jahmyr Gibbs led Detroit with 142 scrimmage yards.');
    expect(text).toContain('ESPN');
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    const articleBtn = dialog.querySelector<HTMLButtonElement>('.espn-news-title .espn-news-link')!;
    expect(articleBtn.dataset.url).toContain('espn.com/nfl/story/_/id/505001');
    await act(async () => articleBtn.click());
    expect(openSpy).toHaveBeenCalledWith(
      'https://www.espn.com/nfl/story/_/id/505001/gibbs-two-tds', '_blank', 'noopener,noreferrer',
    );
    expect(text).toContain('MIN');
    expect(text).toContain('62%');

    // Side pane shows ContractPanel years, dead cap, status, injury, and loaded private note
    expect(text).toContain('Contract years');
    expect(text).toContain('2026');
    expect(text).toContain('2029');
    expect(text).toContain('$4.5M');
    expect(text).toContain('Active roster');
    expect(text).toContain('Questionable · Hamstring · back week 12');
    const noteBox = dialog.querySelector<HTMLTextAreaElement>('#card-note')!;
    expect(noteBox.value).toBe('Hold as core RB1');

    // Switch to Stats tab: last five weeks bars + this week + projection + range
    const statsBtn = [...dialog.querySelectorAll('button')].find(b => b.textContent === '2026 Stats')!;
    await act(async () => statsBtn.click());
    const statsText = dialog.textContent ?? '';
    expect(statsText).toContain('24.8');
    expect(statsText).toContain('19.4');
    expect(statsText).toContain('12–30');
    expect(statsText).toContain('WK6');
    expect(statsText).toContain('WK10');
    expect(statsText).toContain('26.1');
    expect(statsText).toContain('18.5');
    expect(statsText).toContain('21.2');

    // Switch to Projections tab
    const projBtn = [...dialog.querySelectorAll('button')].find(b => b.textContent === 'Projections')!;
    await act(async () => projBtn.click());
    expect(dialog.textContent).toContain('WEEK 11 PROJ');
    expect(dialog.textContent).toContain('Soft · 24');
  });

  it('opens from a Contracts name click and closes on Escape', async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    readHQ.value = fixtureReading();
    const { port } = makePort();
    const layer = document.createElement('div');
    layer.id = 'layer';
    document.body.append(layer);
    host = document.createElement('div');
    document.body.append(host);
    root = createRoot(host);

    await act(async () => root!.render(
      <NativeFranchiseHQ league="L" mine="f1" name="Motor City" port={port} embedded
        tab="contracts-and-cap" leave={() => {}} v3={{ badge: <span>L</span>, kind: 'OWNER' }} />,
    ));
    await flush();

    const nameBtn = [...host.querySelectorAll('button.lk')].find(b => b.textContent === 'Jahmyr Gibbs')!;
    expect(nameBtn).toBeTruthy();
    await act(async () => (nameBtn as HTMLElement).click());
    await flush();

    const dialog = layer.querySelector('[role="dialog"]')!;
    expect(dialog).toBeTruthy();
    expect(dialog.textContent).toContain('JAHMYR GIBBS');
    expect(dialog.textContent).toContain('Contract years');
    expect(dialog.textContent).toContain('Questionable · Hamstring · back week 12');

    await act(async () => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    });
    expect(layer.querySelector('[role="dialog"]')).toBeNull();

    // Expiring contracts panel (.lk) also opens the card; unmapped player shows plain reason
    const expiringBtn = [...host.querySelectorAll('button.lk')]
      .filter(b => b.textContent === 'Expiring Back').at(-1)!;
    await act(async () => (expiringBtn as HTMLElement).click());
    await flush();
    const unmappedDialog = layer.querySelector('[role="dialog"]')!;
    expect(unmappedDialog.textContent).toContain('EXPIRING BACK');
    expect(unmappedDialog.textContent).toContain('No ESPN player mapping is on file for this player.');
  });
});
