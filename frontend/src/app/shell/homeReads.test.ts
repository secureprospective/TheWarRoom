import { describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { readHomeBrief, readLiveBrief, readThinHomeBrief } from './homeReads';

const rejectingPort = () => vi.fn<HostedPort>(async () => { throw new Error('not configured'); });
const count = (port: ReturnType<typeof rejectingPort>, command: string) =>
  port.mock.calls.filter(([request]) => request.command === command).length;

const HOME_ONLY_COMMANDS = [
  'native.assets.read', 'trades.read', 'native.cap.check', 'commish.ledger.filtered.read',
  'bids.read', 'draft.read', 'commish.jobs.read', 'job.snoozes.read', 'calendar.read',
  'schedule.read',
] as const;

function livePort(opts: { failResults?: string } = {}) {
  const common = { league: 'league', season: 2026, revision: 1, name: 'Test League', words: 'Published' };
  const teams = [
    { franchise: 'mine', name: 'Iron Curtain', abbreviation: 'IRN' },
    { franchise: 'other', name: 'Gridlock', abbreviation: 'GRD' },
  ];
  return vi.fn<HostedPort>(async request => {
    const args = request.arguments;
    let data: unknown;
    switch (request.command) {
      case 'native.directory.read':
        data = { league: 'league', name: 'Test League', revision: 1, rosterSeason: 2026,
          franchises: teams }; break;
      case 'week.read':
        data = { schedule: { league: 'league', season: 2026, revision: 1, franchise: 'mine',
          games: [{ week: 11, venue: 'home' }] }, current: { week: 11 }, message: '' }; break;
      case 'schedule.franchise.read':
        data = { league: 'league', franchise: 'mine', season: 2026, revision: 1, message: '',
          games: [{ week: 11, game: 'g1', opponent: 'other', venue: 'Home' }] }; break;
      case 'standings.read':
        data = { ...common, status: 'Ready', through: 0, decided: 0, pending: 1, criteria: [], noData: [],
          computed: teams.map((t, i) => ({ franchise: t.franchise, name: t.name, rank: i + 1,
            wins: 0, losses: 0, ties: 0, record: '0-0', pointsFor: 0, pointsWords: '', why: [] })) };
        break;
      case 'bracket.read':
        data = { ...common, status: 'Projected', started: false, fields: [], projected: [],
          rules: { startWeek: 14, finalWeek: 17, perConference: 2, byeSeeds: 0, shape: [] } }; break;
      case 'odds.read':
        data = { ...common, status: 'Ready', powerWords: '', gameWords: '', oddsWords: '', week: 11,
          runs: 100, seed: 1, rows: [],
          games: [{ week: 11, home: 'mine', away: 'other', homeName: 'Iron Curtain',
            awayName: 'Gridlock', homeWin: 0.44 }] }; break;
      case 'results.read':
        if (opts.failResults) throw new Error(opts.failResults);
        data = { league: 'league', season: 2026, revision: 1, words: 'Published', week: 11,
          published: true, missed: [],
          games: [{ week: 11, home: 'mine', away: 'other', homeName: 'Iron Curtain',
            awayName: 'Gridlock', status: 'in', homeProvisional: 242000, awayProvisional: 258000,
            words: '', why: [] }],
          scores: [
            { franchise: 'mine', name: 'Iron Curtain', status: 'Provisional', provisional: 242000,
              words: '', adjustment: 0, lineup: { status: 'Valid', words: '', flagged: false },
              starters: [{ player: 'm1', name: 'Mahomes, Patrick', position: 'QB', words: '',
                provisional: 181000 }] },
            { franchise: 'other', name: 'Gridlock', status: 'Provisional', provisional: 258000,
              words: '', adjustment: 0, lineup: { status: 'Valid', words: '', flagged: false },
              starters: [{ player: 'o1', name: 'Allen, Josh', position: 'QB', words: '',
                provisional: 204000 }] },
          ] }; break;
      case 'lineup.read': {
        const franchise = 'franchise' in args ? String(args.franchise) : '';
        data = { ...common, franchise, week: 11,
          starters: [{ player: franchise === 'mine' ? 'm1' : 'o1',
            team: franchise === 'mine' ? 'KC' : 'BUF', locked: true }] }; break;
      }
      case 'projections.read':
        data = { league: 'league', season: 2026, week: 11, players: [] }; break;
      default:
        throw new Error(`Unexpected read ${request.command}`);
    }
    return { data, football: '', writable: false, label: 'Server' };
  });
}

describe('Home commissioner-only jobs read', () => {
  it('does not ask an owner session for commissioner jobs', async () => {
    const port = rejectingPort();
    await readHomeBrief('league', 'franchise.1', port, undefined, null, true, false);
    expect(count(port, 'commish.jobs.read')).toBe(0);
  });

  it('reads jobs for a commissioner with a franchise and the unassigned commissioner view', async () => {
    const assigned = rejectingPort();
    await readHomeBrief('league', 'franchise.1', assigned, undefined, null, true, true, true);
    expect(count(assigned, 'commish.jobs.read')).toBe(1);
    expect(count(assigned, 'job.snoozes.read')).toBe(1);

    const unassigned = rejectingPort();
    await readThinHomeBrief('league', '', unassigned, undefined, true);
    expect(count(unassigned, 'commish.jobs.read')).toBe(1);
    expect(count(unassigned, 'job.snoozes.read')).toBe(1);

    const visitor = rejectingPort();
    await readThinHomeBrief('league', '', visitor, undefined, false);
    expect(count(visitor, 'commish.jobs.read')).toBe(0);
    expect(count(visitor, 'job.snoozes.read')).toBe(0);
  });

  it('keeps local home layout mode off the hosted account preference route', async () => {
    const local = rejectingPort();
    await readHomeBrief('league', 'franchise.1', local, undefined, null, true, true);
    expect(count(local, 'job.snoozes.read')).toBe(0);
  });
});

describe('readLiveBrief', () => {
  it('reads pulse, weekly results, and matchup while skipping all Home-only commands', async () => {
    const port = livePort();
    const brief = await readLiveBrief('league', 'mine', port);
    expect(brief.pulse?.week).toBe(11);
    expect(brief.pulse?.errors).toEqual([]);
    expect(brief.results?.week).toBe(11);
    expect(brief.matchup?.lineups.mine).toHaveLength(1);
    expect(brief.matchup?.lineups.other).toHaveLength(1);
    expect(brief.errors).toEqual({});
    for (const command of HOME_ONLY_COMMANDS) {
      expect(count(port, command), command).toBe(0);
    }
  });

  it('surfaces affected results errors identically to readHomeBrief without Home-only calls', async () => {
    const live = livePort({ failResults: 'results.read refused' });
    const home = livePort({ failResults: 'results.read refused' });
    const liveBrief = await readLiveBrief('league', 'mine', live);
    const homeBrief = await readHomeBrief('league', 'mine', home, undefined, null, true, false, true);
    expect(liveBrief.errors.results).toBe('results.read refused');
    expect(liveBrief.errors.results).toBe(homeBrief.errors.results);
    expect(liveBrief.errors.matchup).toBe(homeBrief.errors.matchup);
    for (const command of HOME_ONLY_COMMANDS) {
      expect(count(live, command), command).toBe(0);
    }
  });

  it('keeps full Thin Home fanout intact for a seated owner', async () => {
    const port = rejectingPort();
    await readThinHomeBrief('league', 'mine', port);
    for (const command of ['native.assets.read', 'trades.read', 'native.cap.check',
      'commish.ledger.filtered.read', 'bids.read', 'draft.read']) {
      expect(count(port, command), command).toBe(1);
    }
  });
});
