// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from './HostedLeague';
import { actTable } from './actTable';
import type { HostedPort } from './hostedRoom';
import type { NativeRequest } from '../commands/native';

const league = 'native.people';
const seats = Array.from({ length: 32 }, (_, index) => ({
  franchise: `franchise.${String(index + 1).padStart(2, '0')}`,
  name: `Team ${String(index + 1).padStart(2, '0')}`,
  abbreviation: `T${String(index + 1).padStart(2, '0')}`,
  owner: index < 30 ? `owner.${index + 1}` : '',
  ...(index === 30 ? { caretaker: 'caretaker.31' } : {}),
}));
const people = {
  league, name: 'People Test', revision: 7, seats,
  roles: [{ person: 'owner.1', role: 'commissioner' },
    { person: 'owner.2', role: 'co-commissioner' }, { person: 'owner.3', role: 'DOT' }],
  invites: [], dotCount: 1, dotStatus: 'Ready', votes: [], succession: 'Commissioner active',
  actors: [{ person: 'owner.1', effectivePermissions: ['approve-stat-fixes', 'rule-on-trades',
    'edit-lineups', 'move-picks', 'change-rules', 'run-rollover'] }],
  yourPermissions: ['approve-stat-fixes', 'rule-on-trades', 'edit-lineups', 'move-picks',
    'change-rules', 'run-rollover'],
  yourRole: 'commissioner', yourFranchise: seats[0]!.franchise, mayChangeRules: true,
  mayGrantRoles: true, mayPostNotices: true,
};
const directory = {
  league, name: 'People Test', revision: 7, rosterSeason: 2026, source: 'held',
  franchises: seats.map(seat => ({ ...seat, owner: seat.owner || '', caretaker: seat.caretaker || '',
    division: '' })),
};
function response(data: unknown) {
  return { data, label: 'League server', writable: true, football: 'held-pin' };
}
function rules(revision = 7) {
  return { league, name: 'People Test', revision, rollover: 'No rollover date is set', sections: [] };
}
function makePort(refuseCaretaker = false): HostedPort & ReturnType<typeof vi.fn> {
  let revision = 7;
  return vi.fn(async (request: NativeRequest) => {
    switch (request.command) {
      case 'commish.rules.read': return response(rules(revision));
      case 'commish.jobs.read': return response({ league, name: 'People Test', revision, jobs: [] });
      case 'commish.people.read': return response({ ...people, revision });
      case 'native.directory.read': return response({ ...directory, revision });
      case 'commish.actor.read': return response('owner.1');
      case 'commish.seat.caretaker':
        if (request.mode === 'preview') return response({ question: 'Seat a caretaker?',
          target: { league, name: 'People Test', version: revision }, fields: [
            { label: 'Franchise', before: 'Vacant', beforeSource: '', after: 'Team 32',
              afterSource: '', when: 'Now' },
          ], affects: { readings: [], count: 0, words: '' },
          blockers: refuseCaretaker ? ['The acting person lacks permission.'] : [],
          safeguard: 'confirm' });
        revision++;
        return response({ entry: 'entry.8', league, revision,
          recordedAt: '2026-11-18T18:00:00Z', message: 'Recorded', plan: null });
      default: throw new Error(`Unexpected hosted request: ${request.command} ${request.mode}`);
    }
  }) as HostedPort & ReturnType<typeof vi.fn>;
}

let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks(); });
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
async function setValue(element: HTMLInputElement | HTMLSelectElement, value: string) {
  const prototype = element instanceof HTMLSelectElement
    ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(element, value);
    element.dispatchEvent(new Event(element instanceof HTMLSelectElement ? 'change' : 'input',
      { bubbles: true }));
  });
}
function button(text: string) {
  return [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(candidate => candidate.textContent?.trim() === text ||
      candidate.getAttribute('aria-label') === text);
}
async function openPeople(port: HostedPort) {
  await act(async () => root.render(<HostedLeague league={league} franchise="" name="People Test"
    role="Commissioner" port={port} />));
  await settle();
  const control = button('Control Room');
  expect(control).toBeDefined();
  await act(async () => control!.click());
  await settle();
  const peopleTab = [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')]
    .find(candidate => candidate.textContent?.trim() === 'People');
  expect(peopleTab).toBeDefined();
  await act(async () => peopleTab!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[aria-label="Franchises and seats"]')).not.toBeNull();
  });
}

it('renders 32 seats, supports search and vacancy filtering, and states missing filters honestly',
  async () => {
  const port = makePort();
  await openPeople(port);
  expect(host.querySelectorAll('[data-seat]')).toHaveLength(32);
  expect(port.mock.calls.filter(([request]) => request.command === 'commish.people.read')).toHaveLength(1);

  const search = host.querySelector<HTMLInputElement>('input[aria-label="Search seats"]')!;
  await setValue(search, 'Team 32');
  expect(host.querySelectorAll('[data-seat]')).toHaveLength(1);
  expect(host.querySelector('[data-seat="franchise.32"]')).not.toBeNull();

  await setValue(search, '');
  const vacant = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(candidate => candidate.textContent?.trim().startsWith('Vacant'))!;
  await act(async () => vacant.click());
  expect(host.querySelectorAll('[data-seat]')).toHaveLength(1);
  expect(host.querySelector('[data-seat="franchise.32"]')).not.toBeNull();
  expect(button('Quiet · unavailable')?.disabled).toBe(true);
  expect(button('Staff · unavailable')?.disabled).toBe(true);
  expect(host.textContent).toContain('last-seen and staff status are not returned');
});

it('routes hosted People tools through preview, review, hold-to-confirm execute', async () => {
  const port = makePort();
  await openPeople(port);
  const caretaker = host.querySelector<HTMLButtonElement>(
    '[data-people-command="commish.seat.caretaker"]');
  expect(caretaker).not.toBeNull();
  await act(async () => caretaker!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(port.mock.calls.some(([request]) => request.command === 'native.directory.read')).toBe(true);
  });
  const franchise = host.querySelector<HTMLSelectElement>('.hosted-people-tool select')!;
  await setValue(franchise, 'franchise.32');
  const preview = button('Preview seat a caretaker');
  expect(preview).toBeDefined();
  await act(async () => preview!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[aria-label="Confirm league change"]')).not.toBeNull();
  });
  const requests = () => port.mock.calls.map(([request]) => request);
  expect(requests().filter(request => request.command === 'commish.seat.caretaker')
    .map(request => request.mode)).toEqual(['preview']);

  const confirm = host.querySelector<HTMLButtonElement>('button[aria-label="Confirm league change"]')!;
  await act(async () => {
    const down = new Event('pointerdown', { bubbles: true });
    Object.defineProperty(down, 'button', { value: 0 });
    confirm.dispatchEvent(down);
    await new Promise(resolve => setTimeout(resolve, 650));
    confirm.dispatchEvent(new Event('pointerup', { bubbles: true }));
  });
  await vi.waitFor(async () => {
    await settle();
    expect(requests().filter(request => request.command === 'commish.seat.caretaker')
      .map(request => request.mode)).toEqual(['preview', 'execute']);
  });
  expect(requests().findIndex(request => request.command === 'commish.seat.caretaker'
    && request.mode === 'preview')).toBeLessThan(requests().findIndex(request =>
      request.command === 'commish.seat.caretaker' && request.mode === 'execute'));
});

it('does not execute a People change when the server preview denies the acting person', async () => {
  const port = makePort(true);
  await openPeople(port);
  await act(async () => host.querySelector<HTMLButtonElement>(
    '[data-people-command="commish.seat.caretaker"]')!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(port.mock.calls.some(([request]) => request.command === 'native.directory.read')).toBe(true);
  });
  const franchise = host.querySelector<HTMLSelectElement>('.hosted-people-tool select')!;
  await setValue(franchise, 'franchise.32');
  await act(async () => button('Preview seat a caretaker')!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.textContent).toContain('The acting person lacks permission.');
  });
  expect(button('Confirm league change')).toBeUndefined();
  expect(port.mock.calls.filter(([request]) => request.command === 'commish.seat.caretaker')
    .map(([request]) => request.mode)).toEqual(['preview']);
});

it('routes every People tool to a registered hosted form and leaves D65 unavailable', async () => {
  const port = makePort();
  await openPeople(port);
  const forms = new Set(actTable.map(action => action.command));
  const commands = () => [...host.querySelectorAll('[data-people-command]')].map(node =>
    node.getAttribute('data-people-command'));
  const seatsCommands = [
    'commish.seat.invite', 'commish.seat.caretaker', 'commish.seat.replace', 'commish.seat.remove',
  ];
  expect(commands()).toEqual(seatsCommands);
  for (const command of seatsCommands) expect(forms.has(command), command).toBe(true);

  await act(async () => button('Roles & succession')!.click());
  const roleCommands = [
    'commish.role.grant', 'commish.role.revoke', 'commish.deputy.order',
    'commish.succession.stepdown', 'commish.succession.inactivity.set',
    'commish.succession.voteout.open', 'commish.succession.election.open',
    'commish.succession.periods.set',
  ];
  expect(commands()).toEqual(roleCommands);
  for (const command of roleCommands) expect(forms.has(command), command).toBe(true);
  await act(async () => host.querySelector<HTMLButtonElement>(
    '[data-people-command="commish.role.grant"]')!.click());
  expect(button('Preview grant a league role')).toBeDefined();

  await act(async () => button('Permissions')!.click());
  expect(commands()).toEqual(['commish.abilities.set']);
  expect(forms.has('commish.abilities.set')).toBe(true);
  expect(host.textContent).toContain('Permission sets and their six-permission grid are not available');
  expect(forms.has('commish.permissions.grant')).toBe(false);
});
