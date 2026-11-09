import { afterEach, expect, it, vi } from 'vitest';
import { WarRoomDraft } from '../shell/WarRoomDraft';
import { parseSnapshot } from '../data/parse';
import snapshot from '../data/fixtures/snapshot.json';
import { button, click, control, render, settle, type, unmount } from '../commissioner/commissionerDom';
import { dialogOf } from '../commissioner/trayHarness';
import { openRoom, people, wire } from './draftHarness';

let restore: (() => void) | undefined;
afterEach(async () => { await unmount(); restore?.(); vi.restoreAllMocks(); });
const view = () => render(WarRoomDraft, { context: { snapshot: parseSnapshot(snapshot),
  mine: 'franchise.01', week: 0, native: { league: 'native.rules', mine: 'franchise.01',
    embedded: true, leave: () => {} } }, select: () => {} });

it('mounts the existing commissioner clock, board and tray in War Room for the selected league', async () => {
  const opened = await openRoom({ view: view(), leagues: [
    { id: 'other.league', name: 'Other league' }, { id: 'native.rules', name: 'Chosen league' }] });
  restore = opened.restore;
  const { host, reads, sent } = opened;
  expect(reads.length).toBeGreaterThan(0);
  expect(reads.every(r => r.league === 'native.rules')).toBe(true);
  for (const label of ['Skip this pick', 'Pause the clock', 'Set the clock', 'Close the draft',
    'Restore the draft to an earlier point', 'Choose a rookie for round 1 pick 2, Held team 1',
    'Change round 1 pick 1, Held team 3', 'Add an extra pick after round 1 pick 2, Held team 1'])
    expect(button(host, label)).toBeDefined();
  const pause = button(host, 'Pause the clock');
  pause.focus();
  await click(pause);
  await settle();
  const panel = dialogOf(host)!;
  await type(control(panel, 'Reason'), 'Temporary interruption');
  await settle();
  await click(button(panel, 'Pause the clock'));
  await settle();
  expect(sent).toEqual([expect.objectContaining({ league: 'native.rules', command: 'draft.timer',
    reason: 'Temporary interruption', args: { draft: { season: 2027, act: { timer: 'pause' } } } })]);
  expect(dialogOf(host)).toBeUndefined();
});

it.each([
  { reading: wire.paused, label: 'Resume the clock' },
  { reading: { ...wire.running, startup: true,
    draft: { ...wire.running.draft, started: false, onClock: null } },
    label: 'Open the startup draft' },
])('mounts the existing $label launcher for its draft state', async ({ reading, label }) => {
  const opened = await openRoom({ view: view(), reading });
  restore = opened.restore;
  expect(button(opened.host, label)).toBeDefined();
  expect(opened.sent).toEqual([]);
});

it('keeps commissioner actions out of the owner view and submits the existing owner pick', async () => {
  const opened = await openRoom({ view: view(), people: people.owner });
  restore = opened.restore;
  const { host, picks, sent } = opened;
  expect(() => button(host, 'Pause the clock')).toThrow();
  expect(() => button(host, 'Skip this pick')).toThrow();
  await type(control(host, 'Rookie'), '5002');
  await settle();
  await click(button(host, 'Pick Player, 5002'));
  await settle();
  expect(picks).toEqual([expect.objectContaining({ league: 'native.rules',
    franchise: 'franchise.01', player: '5002' })]);
  expect(sent).toEqual([]);
});
