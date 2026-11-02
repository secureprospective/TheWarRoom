import { describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { commands } from '../../commands/registry';
import { all, settle } from '../../commissioner/commissionerDom';
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
    const tabs = all(host, 'button')
      .filter((b) => b.getAttribute('role') === 'tab')
      .map((b) => b.textContent);
    expect(tabs).toEqual([
      'JOBS 3',
      'SITUATIONS',
      'RULES',
      'SEASON DESK',
      'FIX',
      'PICKS',
      'PEOPLE',
      'LEAGUE',
      'SERVER',
    ]);
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
    expect(runSearch(r, 'owner quit').some((h) => h.type === 'SITUATION')).toBe(true);
  });
  it('submits search and command through the shared surface command', async () => {
    const { host } = await room();
    const dispatch = vi.spyOn(commands, 'dispatch');
    const input = (label: string) => all(host, 'input').find((i) => i.getAttribute('aria-label') === label)!;
    await act(async () => {
      (input('Find anything').props().onChange as (e: unknown) => void)({ target: { value: 'roster size' } });
    });
    const submit = async (index: number) =>
      act(async () => {
        (all(host, 'form')[index]!.props().onSubmit as (e: unknown) => void)({ preventDefault() {} });
      });
    await submit(0);
    await settle();
    expect(dispatch).toHaveBeenCalledWith('commissioner.surface', {
      screen: 'control-room',
      action: 'find',
      value: '0',
    });
    expect(all(host, 'div').some((d) => d.getAttribute('aria-label') === 'Change a rule')).toBe(true);
    await act(async () => {
      (input('Command').props().onChange as (e: unknown) => void)({ target: { value: 'people' } });
    });
    await submit(1);
    await settle();
    expect(dispatch).toHaveBeenCalledWith('commissioner.surface', {
      screen: 'control-room',
      action: 'tray',
      value: '',
    });
    expect(
      all(host, 'button').find(
        (b) => b.getAttribute('role') === 'tab' && b.getAttribute('aria-selected') === 'true',
      )?.textContent,
    ).toBe('PEOPLE');
    dispatch.mockRestore();
  });
  it('reads each source once and keeps going when one fails', async () => {
    const calls: string[] = [];
    const ok =
      (name: string, v: unknown = {}) =>
      async (...a: unknown[]) => {
        calls.push(`${name}:${a.join(',')}`);
        return v;
      };
    const port = {
      TargetNativeDirectory: ok('directory', { rosterSeason: 2026, franchises }),
      TargetNativeAssets: ok('assets'),
      TargetCommissionerMarket: ok('market'),
      TargetCorrections: async () => {
        throw new Error('no corrections');
      },
      TargetCommissionerSchedule: ok('schedule'),
      TargetNativeHouseRules: ok('house'),
      TargetShellStatus: ok('status'),
      TargetStatQueue: ok('stat'),
      TargetLeagueCalendar: ok('calendar'),
      TargetNativeWeek: ok('week'),
      TargetNativeDraft: ok('draft'),
      TargetSignInFacts: ok('signins', []),
      GetParams: ok('params', {
        ok: true,
        error: '',
        params: [],
      }),
    } as unknown as RoomPort;
    const d = await readRoom(league, port);
    expect(d.errors).toEqual({ corrections: 'no corrections' });
    expect(calls).toContain(`market:${league},trade.review`);
    expect(calls).toContain(`stat:${league},2026`);
  });
});
