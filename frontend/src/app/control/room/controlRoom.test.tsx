import { describe, expect, it } from 'vitest';
import { all } from '../../commissioner/commissionerDom';
import { leagueState } from '../../commissioner/proxy';
import { readRoom, type RoomPort } from './roomReads';
import { routeCommand } from './route';
import { runSearch } from './search';
import type { Room } from './room';
import { data, franchises, league, room, roomHooks } from './roomTestKit';


roomHooks();

describe('Control Room · shell', () => {
  it('shows the nine branches and counts the live jobs on JOBS', async () => {
    const { host } = await room();
    const tabs = all(host, 'button').filter(b => b.getAttribute('role') === 'tab').map(b => b.textContent);
    expect(tabs).toEqual(['JOBS 3', 'SITUATIONS', 'RULES', 'SEASON DESK', 'FIX', 'PICKS', 'PEOPLE', 'LEAGUE', 'SERVER']);
  });
  it('routes every mock command to an existing form, or says it is not built', () => {
    expect(routeCommand('waivers.process').kind).toBe('act');
    expect(routeCommand('commish.notice.publish').kind).toBe('action');
    expect(routeCommand('commish.score.player').kind).toBe('act');
    expect(routeCommand('pick.grant')).toEqual({ kind: 'unbuilt', command: 'pick.grant' });
  });
  it('finds a rule by its current value and a situation by a synonym', async () => {
    const { host } = await room();
    void host;
    const r = { state: leagueState.peek(), data } as unknown as Room;
    expect(runSearch(r, '20 to 70')[0]).toMatchObject({ type: 'RULE', label: 'Active roster size' });
    expect(runSearch(r, 'owner quit').some(h => h.type === 'SITUATION')).toBe(true);
  });
  it('reads each source once and keeps going when one fails', async () => {
    const calls: string[] = [];
    const ok = (name: string, v: unknown = {}) => async (...a: unknown[]) => { calls.push(`${name}:${a.join(',')}`); return v; };
    const port = {
      TargetNativeDirectory: ok('directory', { rosterSeason: 2026, franchises }), TargetNativeAssets: ok('assets'),
      TargetCommissionerMarket: ok('market'), TargetCorrections: async () => { throw new Error('no corrections'); },
      TargetCommissionerSchedule: ok('schedule'), TargetNativeHouseRules: ok('house'), TargetShellStatus: ok('status'),
      TargetStatQueue: ok('stat'), TargetLeagueCalendar: ok('calendar'), TargetNativeWeek: ok('week'),
      TargetNativeDraft: ok('draft'), TargetSignInFacts: ok('signins', []), GetParams: ok('params', { ok: true, error: '', params: [] }),
    } as unknown as RoomPort;
    const d = await readRoom(league, port);
    expect(d.errors).toEqual({ corrections: 'no corrections' });
    expect(calls).toContain(`market:${league},trade.review`);
    expect(calls).toContain(`stat:${league},2026`);
  });
});
