import { describe, expect, it } from 'vitest';
import { actChange } from '../../commissioner/actBody';
import { actTable } from '../../commissioner/actTable';
import { movedStart } from './SeasonView';
import { act } from 'react';
import { all, settle } from '../../commissioner/commissionerDom';
import { data, room, roomHooks, tab, region } from './roomTestKit';

const now = Date.parse('2026-11-02T12:00:00Z');
const bar = 1200; // px for the league year Mar 1 2026 - Mar 1 2027 (365 days)

describe('Control Room · dragging a season window', () => {
  it('a drag lands on the day it covers; under a day is a click', () => {
    expect(movedStart('2026-09-01T00:00:00Z', 0, bar, now)).toBeNull();
    expect(movedStart('2026-09-01T00:00:00Z', 1, bar, now)).toBeNull();
    const week = (7 / 365) * bar;
    expect(movedStart('2026-09-01T00:00:00Z', week, bar, now)).toEqual({ month: 9, day: 8 });
    expect(movedStart('2026-09-01T00:00:00Z', -week, bar, now)).toEqual({ month: 8, day: 25 });
  });
  it('the move sends calendar.window.move with the window and its new month and day', () => {
    const def = actTable.find(a => a.command === 'calendar.window.move')!;
    expect(actChange(def, { id: 'waivers', 'start.month': '9', 'start.day': '8' },
      { league: 'L', revision: 2, principal: 'c' }, 'moved')).toMatchObject({
      command: 'calendar.window.move', args: { window: { id: 'waivers', start: { month: 9, day: 8 } } },
      effective: 'now', visibility: 'league' });
  });
});


roomHooks();
describe('Control Room · drag interaction', () => {
  it('dragging a window opens the move form with the window and its new day; a still click opens the rule', async () => {
    const start = '2026-09-01T00:00:00Z';
    const windows = [{ id: 'waivers', opens: ['waivers'], start, live: false }];
    const { host } = await room({ TargetNativeAssets: async () => data.assets, TargetNativeDirectory: async () => data.directory } as never, {}, { ...data, calendar: { ...data.calendar!, clock: { ...data.calendar!.clock, windows } } } as never);
    await tab(host, 'SEASON');
    const win = () => all(host, 'div').find(d => d.getAttribute('data-win') === 'waivers')!;
    const target = { setPointerCapture() {}, parentElement: { getBoundingClientRect: () => ({ width: 3650 }) } };
    const ev = (x: number) => ({ clientX: x, pointerId: 1, currentTarget: target });
    await act(async () => { (win().props().onPointerDown as (e: unknown) => void)(ev(100)); });
    await act(async () => { (win().props().onPointerMove as (e: unknown) => void)(ev(170)); });
    await act(async () => { (win().props().onPointerUp as (e: unknown) => void)(ev(170)); });
    await settle();
    const form = region(host, 'Move a league window')!;
    expect(form).toBeTruthy();
    await settle(); await settle();
    const inputs = all(region(host, 'Move a league window')!, 'input').map(i => String(i.props().value ?? ''));
    expect(inputs.slice(0, 3)).toEqual(['waivers', '9', '8']);
    await act(async () => { (win().props().onPointerDown as (e: unknown) => void)(ev(100)); });
    await act(async () => { (win().props().onPointerUp as (e: unknown) => void)(ev(100)); });
    await settle();
    expect(region(host, 'Change a rule')).toBeTruthy();
  });
});
