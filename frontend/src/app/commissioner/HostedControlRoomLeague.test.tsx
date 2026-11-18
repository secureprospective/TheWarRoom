// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import { hostedReads, observeView } from '../data/hostedReads';
import { HostedControlRoom } from './HostedControlRoom';
import { HostedControlRoomLeague } from './HostedControlRoomLeague';
import { createHostedLeagueRoom, type HostedPort } from './hostedRoom';

const league = 'native.test';
function ledger(revision: number) {
  return { league, name: 'Test League', revision, entries: Array.from({ length: 32 }, (_, index) => ({
    id: `entry.${index + 1}`, who: `Owner ${index + 1}`, what: `League change ${index + 1}`,
    why: 'League decision', when: new Date(Date.UTC(2026, 10, index + 1)).toISOString(),
    gravity: 'G1', undo: '', replacedAt: '',
  })) };
}
function notices(revision: number) {
  return { league, revision, notices: Array.from({ length: 32 }, (_, index) => ({
    id: `notice.${index + 1}`, kind: 'notice', text: `Notice ${index + 1}`, pinned: false,
    who: 'Commissioner', when: '2026-11-17T12:00:00Z', withdrawn: false, change: '',
  })), notes: [], releases: [], mail: { words: '', recorded: [] } };
}
function service() {
  let revision = 7;
  const port = vi.fn<HostedPort>(async (request: NativeRequest) => {
    nativeEnvelope(request);
    let data: unknown;
    if (request.command === 'commish.rules.read') data = {
      league, name: 'Test League', revision, rollover: 'No rollover date is set', sections: [],
    };
    else if (request.command === 'commish.jobs.read') data = {
      league, name: 'Test League', revision, jobs: [],
    };
    else if (request.command === 'commish.ledger.read'
      || request.command === 'commish.ledger.filtered.read') data = ledger(revision);
    else if (request.command === 'commish.notices.read') data = notices(revision);
    else if (request.mode === 'preview') data = {
      question: 'Record this message?', target: { league, name: 'Test League', version: revision },
      fields: [{ label: 'Message', before: 'Nothing posted', after: 'New message',
        beforeSource: '', afterSource: '', when: 'Now' }],
      affects: { readings: [], count: 0, words: '' }, blockers: [], safeguard: 'confirm',
    };
    else if (request.mode === 'execute') {
      revision++;
      data = { entry: `entry.${revision}`, league, revision, recordedAt: '2026-11-17T12:00:00Z',
        message: 'Recorded', plan: null };
    } else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  });
  return { port, advance: () => { revision++; }, get revision() { return revision; } };
}

let host: HTMLDivElement;
let root: Root;
let authGeneration = 0;
beforeEach(() => {
  hostedReads.endSession();
  authGeneration++;
  observeView({ authGeneration: String(authGeneration), server: 'https://league.test', league,
    message: '', name: 'Commissioner', role: 'Commissioner', franchise: '', franchiseName: '',
    seed: '', connection: '', helperId: '', confirmation: '', seats: [] });
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks();
});
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 35)); });
function button(label: string) {
  return [...host.querySelectorAll('button')].find(item => item.textContent?.trim() === label);
}
async function click(label: string) {
  const item = button(label);
  expect(item, label).toBeDefined();
  await act(async () => { item!.focus(); item!.click(); });
  await settle();
}
async function fill(label: string, value: string) {
  const element = [...host.querySelectorAll('label')].find(item =>
    item.firstChild?.textContent?.trim() === label)?.querySelector('textarea');
  expect(element, label).toBeDefined();
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')!.set!.call(element, value);
    element!.dispatchEvent(new Event('input', { bubbles: true }));
  });
}
async function mount() {
  const fixture = service();
  const room = createHostedLeagueRoom(league, fixture.port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(
    <HostedControlRoomLeague league={league} room={room} port={fixture.port} />));
  await settle();
  return { ...fixture, room };
}

it('registers the League tab in the hosted frame and loads its server readings', async () => {
  const fixture = service();
  const room = createHostedLeagueRoom(league, fixture.port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(
    <HostedControlRoom league={league} port={fixture.port} heldRoom={room} />));
  await settle();
  await click('League');
  expect(host.querySelector('[data-testid="hosted-control-league"]')).not.toBeNull();
  expect(host.textContent).toContain('League change 32');
  expect(fixture.port.mock.calls.some(([request]) => request.command === 'commish.notices.read')).toBe(true);
  room.dispose();
});

it('filters the ledger and rereads after a newer hosted league topic', async () => {
  const fixture = service();
  const room = createHostedLeagueRoom(league, fixture.port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(
    <HostedControlRoomLeague league={league} room={room} port={fixture.port} />));
  await settle();
  await click('RULES');
  expect(fixture.port.mock.calls.filter(([request]) =>
    request.command === 'commish.ledger.filtered.read')).toHaveLength(1);
  fixture.advance();
  await act(async () => hostedReads.applyTopics([{ kind: 'league', league, version: '8' }]));
  await vi.waitFor(() => expect(fixture.port.mock.calls.filter(([request]) =>
    request.command === 'commish.ledger.filtered.read')).toHaveLength(2));
  expect(host.textContent).toContain('League change 32');
  room.dispose();
});

it('renders every hosted ledger row and notice at 32-row depth without a client cap', async () => {
  const { room } = await mount();
  expect(host.querySelectorAll('.cr-led > div:not(.cr-chips)')).toHaveLength(32);
  expect(host.textContent).toContain('League change 32');
  expect(host.querySelectorAll('.hosted-control-league article')).toHaveLength(32);
  expect(host.textContent).toContain('Notice 32');
  expect(button('TRADES')?.disabled).toBe(true);
  room.dispose();
});

it.each([
  ['NOTICE', 'Notice', 'Post an official notice', 'commish.notice.publish'],
  ['LEAGUE NOTE', 'League note', 'Post a league note', 'commish.whatsnew.post'],
] as const)('%s is previewed, reviewed, executed and reread through the hosted path',
  async (tab, field, title, command) => {
    const { port, room } = await mount();
    await click(tab);
    await fill(field, 'A commissioner message for owners');
    await fill('Reason for this change', 'Owners need this information');
    await click(`Preview ${title.toLowerCase()}`);

    const preview = port.mock.calls.find(([request]) => request.command === command
      && request.mode === 'preview')?.[0];
    expect(preview).toMatchObject({ command, mode: 'preview', arguments: {
      league, command, args: { note: { text: 'A commissioner message for owners', pin: false } },
      reason: 'Owners need this information', expected: 7, effective: 'now', visibility: 'league',
    } });
    expect(host.querySelector('[aria-label="Confirm league change"]')).not.toBeNull();
    expect(port.mock.calls.some(([request]) => request.command === command
      && request.mode === 'execute')).toBe(false);
    const noticesBefore = port.mock.calls.filter(([request]) =>
      request.command === 'commish.notices.read').length;
    const ledgerBefore = port.mock.calls.filter(([request]) =>
      request.command === 'commish.ledger.read').length;

    const confirm = button('Confirm league change');
    expect(confirm).toBeDefined();
    await act(async () => confirm!.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Enter', bubbles: true,
    })));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 700)); });
    await settle();

    expect(port.mock.calls.find(([request]) => request.command === command
      && request.mode === 'execute')?.[0]).toEqual({ ...preview, mode: 'execute' });
    expect(port.mock.calls.filter(([request]) =>
      request.command === 'commish.notices.read').length).toBeGreaterThan(noticesBefore);
    expect(port.mock.calls.filter(([request]) =>
      request.command === 'commish.ledger.read').length).toBeGreaterThan(ledgerBefore);
    expect(room.peek().reading?.rules.revision).toBe(8);
    room.dispose();
  });

it('leaves newsletter publishing and unsupported local tools explicitly unavailable', async () => {
  const { room } = await mount();
  await click('NEWSLETTER');
  expect(host.textContent).toContain('Newsletter publishing has no hosted command.');
  expect(host.textContent).toContain('Feed and engine controls, league creation or migration, '
    + 'and house-rule editing');
  room.dispose();
});
