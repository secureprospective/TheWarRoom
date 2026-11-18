// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { HostedControlRoom } from './HostedControlRoom';
import { HostedControlRoomPicks } from './HostedControlRoomPicks';
import { createHostedLeagueRoom, type HostedPort } from './hostedRoom';
import type { NativeRequest } from '../commands/native';

const league = 'native.test';
const teams = Array.from({ length: 32 }, (_, index) => {
  const number = String(index + 1).padStart(2, '0');
  return { franchise: `franchise.${number}`, name: `Team ${number}`, abbreviation: `T${number}` };
});
const picks = [2027, 2028].flatMap(season => [1, 2, 3, 4].flatMap(round =>
  teams.map((team, index) => ({ compensatory: false, correction: '', season, round, slot: index + 1,
    original: team.franchise, owner: season === 2027 && round === 1 && index === 1
      ? teams[0]!.franchise : team.franchise, player: '', skipped: false }))));
const assets = {
  pool: [], identities: [], corrections: [], league, revision: 7, seeded: true, entry: '', source: 'held',
  snapshot: { season: 2026, franchises: teams.map(team => ({ id: team.franchise, heldId: team.franchise,
    name: team.name, abbreviation: team.abbreviation, division: 'Division 1' })), players: [], picks,
    pool: [], standingsOrder: [], pickHistory: [], adjustments: [], sides: {} },
};
const rules = { league, name: 'Test League', revision: 7, rollover: 'No rollover date is set', sections: [] };
const directory = { league, name: 'Test League', revision: 7, rosterSeason: 2026, source: 'held',
  franchises: teams.map(team => ({ franchise: team.franchise, name: team.name,
    abbreviation: team.abbreviation, owner: '', caretaker: '', division: 'Division 1' })) };
const preview = { question: 'Correct the pick owner?', target: { league, name: 'Test League', version: 7 },
  fields: [{ label: 'Pick owner', before: 'Team 02', beforeSource: 'ledger', after: 'Team 03',
    afterSource: 'ledger', when: 'now' }], affects: { readings: ['picks'], count: 1, words: 'One pick' },
  blockers: [], safeguard: 'confirm' };
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks(); });
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
function testPort() {
  return vi.fn<HostedPort>(async (request: NativeRequest) => {
    let data: unknown;
    if (request.command === 'commish.rules.read') data = rules;
    else if (request.command === 'commish.jobs.read') data = { ...rules, jobs: [] };
    else if (request.command === 'native.assets.read') data = assets;
    else if (request.command === 'native.directory.read') data = directory;
    else if (request.mode === 'preview') data = preview;
    else if (request.mode === 'execute') data = { entry: 'entry.1', league, revision: 8,
      recordedAt: '2026-11-17T12:00:00Z', message: 'Recorded' };
    else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  });
}
async function change(select: HTMLSelectElement, value: string) {
  await act(async () => {
    select.value = value; select.dispatchEvent(new Event('change', { bubbles: true }));
  });
}

it('registers Picks in the hosted frame and renders the full 32-team list with filters', async () => {
  const port = testPort();
  await act(async () => root.render(<HostedControlRoom league={league} port={port} />));
  await settle();
  const picksTab = [...host.querySelectorAll('[role="tab"]')]
    .find(button => button.textContent?.includes('Picks'));
  expect(picksTab).toBeDefined();
  await act(async () => picksTab!.dispatchEvent(new MouseEvent('click', { bubbles: true })));
  await vi.waitFor(async () => { await settle(); expect(host.querySelector('tbody tr')).not.toBeNull(); });
  expect(port.mock.calls.filter(([request]) => request.command === 'native.assets.read')).toHaveLength(1);
  expect(host.textContent).toContain('128 of 256 picks');
  await change(host.querySelector<HTMLSelectElement>('[aria-label="Filter picks by year"]')!, 'all');
  await settle();
  expect(host.querySelectorAll('tbody tr')).toHaveLength(256);
  expect(host.textContent).toContain('256 of 256 picks');

  await change(host.querySelector<HTMLSelectElement>('[aria-label="Filter picks by round"]')!, '2');
  await settle();
  expect(host.querySelectorAll('tbody tr')).toHaveLength(64);
  await change(host.querySelector<HTMLSelectElement>('[aria-label="Filter picks by round"]')!, 'all');
  await change(host.querySelector<HTMLSelectElement>(
    '[aria-label="Filter picks by trade status"]')!, 'traded');
  await settle();
  expect(host.querySelectorAll('tbody tr')).toHaveLength(1);
  expect(host.textContent).toContain('Team 02');
  expect(host.querySelector<HTMLOptionElement>('[value="conditioned"]')?.disabled).toBe(true);

  await change(host.querySelector<HTMLSelectElement>('[aria-label="Filter picks by trade status"]')!, 'all');
  const search = host.querySelector<HTMLInputElement>('[aria-label="Search picks by team"]')!;
  await act(async () => {
    search.value = 'Team 01'; search.dispatchEvent(new Event('input', { bubbles: true }));
  });
  await settle();
  expect(host.querySelectorAll('tbody tr').length).toBeGreaterThan(0);
  expect(host.textContent).toContain('Condition');
  expect(host.textContent).toContain('not on the server app yet');
});

it('moves a pick through correct.pick preview, review and execute with its exact slot', async () => {
  const port = testPort();
  const room = createHostedLeagueRoom(league, port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(<HostedControlRoomPicks league={league} room={room} port={port} />));
  await vi.waitFor(async () => { await settle(); expect(host.querySelector('tbody tr')).not.toBeNull(); });
  const move = [...host.querySelectorAll('button')]
    .find(button => button.textContent === 'Correct a pick owner');
  expect(move).toBeDefined();
  await act(async () => move!.click());
  await vi.waitFor(async () => { await settle(); expect(host.querySelector('textarea')).not.toBeNull(); });

  const pickSelect = [...host.querySelectorAll('select')].find(select =>
    [...select.labels].some(label => label.textContent?.startsWith('Pick')))!;
  const pickValue = [...pickSelect.options].find(option => {
    if (!option.value) return false;
    const pick = JSON.parse(option.value) as { slot?: number; season?: number };
    return pick.season === 2027 && pick.slot === 2;
  })!.value;
  await change(pickSelect, pickValue);
  const ownerSelect = [...host.querySelectorAll('select')].find(select =>
    [...select.labels].some(label => label.textContent?.startsWith('New owner')))!;
  await change(ownerSelect, 'franchise.03');
  const reason = host.querySelector('textarea')!;
  reason.value = 'Correct the recorded pick owner';
  reason.dispatchEvent(new Event('input', { bubbles: true }));
  reason.dispatchEvent(new Event('change', { bubbles: true }));
  const previewButton = [...host.querySelectorAll('button')].find(button =>
    button.textContent === 'Preview correct a pick owner');
  expect(previewButton).toBeDefined();
  await act(async () => previewButton!.click());
  await vi.waitFor(() => expect(port.mock.calls.some(([request]) =>
    request.command === 'correct.pick' && request.mode === 'preview')).toBe(true));
  const previewRequest = port.mock.calls.find(([request]) =>
    request.command === 'correct.pick' && request.mode === 'preview')![0];
  expect(previewRequest.arguments).toMatchObject({ args: { correction: { owner: 'franchise.03',
    pick: { season: 2027, round: 1, original: 'franchise.02', slot: 2 } } } });
  expect(host.querySelector('[aria-label="Confirm league change"]')).not.toBeNull();
  const confirm = [...host.querySelectorAll<HTMLButtonElement>('button')].find(button =>
    button.getAttribute('aria-label') === 'Confirm league change');
  expect(confirm).toBeDefined();
  await vi.waitFor(() => expect(confirm!.disabled).toBe(false));
  await act(async () => {
    confirm!.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, button: 0 }));
    await new Promise(resolve => setTimeout(resolve, 650));
    confirm!.dispatchEvent(new MouseEvent('pointerup', { bubbles: true, button: 0 }));
  });
  await settle();
  const writes = port.mock.calls.filter(([request]) => request.command === 'correct.pick');
  expect(writes.map(([request]) => request.mode)).toEqual(['preview', 'execute']);
  expect(writes[1]![0]).toEqual({ ...previewRequest, mode: 'execute' });
  await act(async () => room.dispose());
});
