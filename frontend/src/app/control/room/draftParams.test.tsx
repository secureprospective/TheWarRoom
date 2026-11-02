import { describe, expect, it, vi } from 'vitest';
import { button, click, present } from '../../commissioner/commissionerDom';
import { commands } from '../../commands/registry';
import { data, league, room, roomHooks, tab } from './roomTestKit';
import { readRoom, type RoomData, type RoomPort } from './roomReads';
import { paramWords } from './LeagueView';
import { roundOne } from './SeasonView';

roomHooks();
const draft = { league, season: 2027, order: [{ round: 1, words: '', picks: [
  { round: 1, slot: 1, original: 'franchise.02', owner: 'franchise.02' },
  { round: 1, slot: 2, original: 'franchise.01', owner: 'franchise.01' }] }] } as unknown as RoomData['draft'];
const param = (overridden: boolean) => ({ key: 'k', overridden }) as never;

describe('Control Room · draft order and engine parameters', () => {
  it('round 1 comes from draft.read by slot and team name', () => {
    expect(roundOne({ ...data, draft })).toEqual(['1 Held Bison', '2 Held Owls']);
  });
  it('engine parameters summarise GetParams, and say a failed read', () => {
    expect(paramWords({ ...data, params: { ok: true, error: '', params: [] } as never })).toBe('none defined');
    expect(paramWords({ ...data, params: { ok: true, error: '', params: [param(true), param(false)] } as never }))
      .toBe('2 defined · 1 changed');
    expect(paramWords({ ...data, errors: { params: 'no params' } })).toBe('no params');
  });
  it('readRoom asks draft.read for the next draft and keeps a failed GetParams as words', async () => {
    const asked: unknown[] = [];
    const ok = (v: unknown = {}) => async () => v;
    const port = {
      TargetNativeDirectory: ok({ rosterSeason: 2026, franchises: [{ franchise: 'franchise.01' }] }),
      TargetNativeAssets: ok(), TargetCommissionerMarket: ok(), TargetCorrections: ok(), TargetCommissionerSchedule: ok(),
      TargetNativeHouseRules: ok(), TargetShellStatus: ok(), TargetStatQueue: ok(), TargetLeagueCalendar: ok(),
      TargetNativeWeek: ok(), TargetNativeDraft: async (s: unknown) => { asked.push(s); return draft; },
      GetParams: ok({ ok: false, error: 'params store closed', params: [] }),
    } as unknown as RoomPort;
    const d = await readRoom(league, port);
    expect(asked[0]).toMatchObject({ league, season: 0, franchise: 'franchise.01' });
    expect(d.draft).toBe(draft);
    expect(d.errors.params).toBe('params store closed');
  });
  it('the Season desk shows round 1; OPEN DRAFT ROOM TRAY opens the Draft room', async () => {
    const nav = vi.spyOn(commands, 'dispatch');
    const { host } = await room();
    await tab(host, 'SEASON');
    expect(present(host, 'ROUND 1 ORDER')).toBe(true);
    await click(button(host, 'OPEN DRAFT ROOM TRAY ›'));
    expect(nav).toHaveBeenCalledWith('nav.open', { node: 'trade', workspace: 'draft-room' });
  });
});
