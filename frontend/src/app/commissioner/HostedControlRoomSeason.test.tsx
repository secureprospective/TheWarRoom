// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { HostedControlRoomSeason } from './HostedControlRoomSeason';
import type { HostedPort } from './hostedRoom';
import type { NativeRequest } from '../commands/native';
import type { ActDefinition } from './actTable';

const league = 'native.test';
const teams = Array.from({ length: 32 }, (_, index) => ({ franchise: `team-${index + 1}`,
  name: `Team ${index + 1}`, abbreviation: `T${index + 1}` }));
const games = Array.from({ length: 16 }, (_, index) => ({ id: `game-${index + 1}`,
  home: `team-${index * 2 + 1}`, away: `team-${index * 2 + 2}` }));
const backupPreview = { question: 'Save and verify a league backup now?',
  target: { league, name: 'Test League', version: 7 }, fields: [{ label: 'Off-site league backup',
    before: 'No manual backup started', beforeSource: 'Hosted server',
    after: 'Save and check the league backup in Cloudflare R2',
    afterSource: 'Hosted server backup service', when: 'now' }],
  affects: { readings: [], count: null, words: 'No league settings or records change' },
  blockers: [], safeguard: 'confirm' };
const bracket = { league, name: 'Test League', season: 2026, revision: 7, status: 'Projected',
  words: 'Playoff bracket is not set.', preset: '', presets: [], started: false, startKnown: true,
  fields: [], projected: [], rounds: [], champion: null, title: {}, history: [], changes: [],
  rules: { words: '', perConference: 2, divisionWinners: 0, wildCards: 0, byeSeeds: 0,
    seeding: 'rules', reseed: false, startWeek: 15, finalWeek: 17, homeBonus: '0', shape: [] } };
let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks(); });

it('renders hosted schedule and playoffs at 32 teams and routes writes to the shared form',
  async () => {
  const read = vi.fn(async (request: NativeRequest) => {
    if (request.command === 'native.directory.read') return { data: { league, name: 'Test League',
      revision: 7, rosterSeason: 2026, source: 'held', franchises: teams },
  label: 'Server', writable: true, football: 'held' };
    if (request.command === 'commish.schedule.read') return { data: { league, revision: 7,
      draft: [], published: [{ week: 1, games }, { week: 2, games }] },
      label: 'Server', writable: true, football: 'held' };
    if (request.command === 'bracket.read') return { data: bracket,
      label: 'Server', writable: true, football: 'held' };
    if (request.command === 'calendar.read') return { data: { league, revision: 7, decision: {}, clock: {
      windows: [{ id: 'lineup', opens: ['lineup'], start: '2026-09-01T00:00:00Z', live: true }],
      exceptions: [], tradeDeadline: null, secondsToTradeDeadline: null } },
      label: 'Server', writable: true, football: 'held' };
    if (request.command === 'backup.now' && request.mode === 'preview') return { data: backupPreview,
      label: 'Server', writable: true, football: 'held' };
    if (request.command === 'backup.now') return { data: { league, captured: '2026-11-18T12:00:00Z',
      verified: '2026-11-18T12:00:01Z', where: 'Cloudflare R2', checked: true,
      message: 'League backup saved 2026-11-18 12:00 UTC and checked in Cloudflare R2.' },
      label: 'Server', writable: true, football: 'held' };
    throw new Error(`Unexpected read ${request.command}`);
  }) as HostedPort & ReturnType<typeof vi.fn>;
  const onTool = vi.fn<(definition: ActDefinition) => void>();
  await act(async () => root.render(<HostedControlRoomSeason league={league} port={read} onTool={onTool} />));
  await vi.waitFor(() => expect(host.textContent).toContain('32 games'));
  expect(host.textContent).toContain('2 published weeks');
  expect(host.textContent).toContain('lineup · 2026-09-01T00:00:00Z · Live');
  expect(host.textContent).toContain('Rollover starts with a fresh verified backup.');
  expect(host.textContent).toContain('Playoff bracket is not set.');
  expect(host.textContent).toContain('Divisions');
  expect(host.textContent).toContain('not on the server app yet');
  const edit = [...host.querySelectorAll('button')].find(button => button.textContent === 'Edit a game');
  expect(edit).toBeDefined();
  await act(async () => edit!.click());
  expect(onTool.mock.calls[0]?.[0].command).toBe('schedule.edit');
  expect(read.mock.calls.map(([request]) => request.command)).toEqual([
    'native.directory.read', 'commish.schedule.read', 'bracket.read', 'calendar.read',
  ]);
  onTool.mockClear();
  const rollover = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent === 'Roll over season')!;
  await act(async () => rollover.click());
  await vi.waitFor(() => expect(host.textContent).toContain('Save and verify a league backup now?'));
  expect(read.mock.calls.filter(([request]) => request.command === 'backup.now')
    .map(([request]) => request.mode)).toEqual(['preview']);
  expect(onTool).not.toHaveBeenCalled();
  const confirm = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('Confirm back up now'))!;
  await act(async () => {
    confirm.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }));
    await new Promise(resolve => setTimeout(resolve, 650));
    confirm.dispatchEvent(new KeyboardEvent('keyup', { key: ' ', bubbles: true }));
  });
  await vi.waitFor(() => expect(onTool).toHaveBeenCalledTimes(1));
  expect(onTool.mock.calls[0]?.[0].command).toBe('season.rollover');
  expect(read.mock.calls.filter(([request]) => request.command === 'backup.now')
    .map(([request]) => request.mode)).toEqual(['preview', 'execute']);
});
