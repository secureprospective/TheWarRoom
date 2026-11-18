// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from './HostedLeague';
import { HostedRules } from './HostedRules';
import { createHostedLeagueRoom, type HostedPort } from './hostedRoom';
import { parseRules } from './rules';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import NativeAccount from '../control/NativeAccount';
import type { NativeView } from '../control/nativeApi';

const league = 'native.test';
function rules(revision = 7, current = 'individual', nowOnly = false) {
  return { league, name: 'Test League', revision, rollover: 'At the next season rollover', sections: [
    { label: 'Lineup', rows: [{ field: 'lineup.lock', label: 'Lineup lock', value: current,
      current, schema: { kind: 'string', choices: [
        { value: 'individual', label: 'Individual kickoff' },
        { value: 'first_kickoff', label: 'First kickoff' },
      ], fields: [] }, state: 'Set', citation: '', changedAt: '', governed: null,
      competitive: true, nowOnly, pending: [], unit: '', safeguards: { nextRollover: 'confirm', now: 'hold' },
    }] },
  ] };
}
function service(blockers: string[] = [], nowOnly = false) {
  let revision = 7;
  let current = 'individual';
  const port = vi.fn<HostedPort>(async request => {
    nativeEnvelope(request);
    let data: unknown;
    switch (request.command) {
      case 'commish.rules.read': data = rules(revision, current, nowOnly); break;
      case 'commish.rules.set':
        if (request.mode === 'preview') data = { question: 'Change the lock?',
          target: { league, name: 'Test League', version: revision }, fields: [
            { label: 'Lineup lock', before: current, after: 'first_kickoff', beforeSource: '',
              afterSource: '', when: 'Now' },
          ], affects: { readings: [], count: 0, words: '' }, blockers, safeguard: 'hold' };
        else {
          revision++; current = 'first_kickoff';
          data = { entry: 'entry.8', league, revision, recordedAt: '2026-11-15T18:00:00Z',
            message: 'Recorded' };
        }
        break;
      case 'native.directory.read': data = { league, name: 'Test League', revision, rosterSeason: 2026,
        franchises: [{ franchise: 'a', name: 'Alpha', abbreviation: 'ALP' },
          { franchise: 'b', name: 'Beta', abbreviation: 'BET' }] }; break;
      case 'week.read': data = { schedule: { league, franchise: '', season: 2026, revision,
        games: [{ week: 1, venue: '' }] }, current: { week: 1, venue: '' } }; break;
      case 'schedule.read': data = { league, season: 2026, revision,
        weeks: [{ week: 1, games: [{ home: 'a', away: 'b' }] }] }; break;
      case 'results.read': data = { league, season: 2026, week: 1, revision,
        published: false, words: 'No results yet', games: [], scores: [], missed: [] }; break;
      case 'standings.read': data = { league, season: 2026, revision, name: 'Test League',
        status: 'Ready', words: 'Standings available', through: 0, decided: 0, pending: 0,
        criteria: [], noData: [], computed: [] }; break;
      default: throw new Error(`Read refused: ${request.command}`);
    }
    return { data, label: 'Server', writable: true, football: 'pin' };
  });
  return port;
}
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  vi.spyOn(console, 'warn').mockImplementation(() => {});
});
afterEach(async () => {
  await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks(); vi.unstubAllGlobals();
});
const settle = () => act(async () => { await new Promise(resolve => setTimeout(resolve, 40)); });
const buttons = (text: string) => [...host.querySelectorAll('button')]
  .filter(b => b.textContent === text || b.getAttribute('aria-label') === text);
async function click(text: string) {
  expect(buttons(text), text).toHaveLength(1);
  await act(async () => { buttons(text)[0].focus(); buttons(text)[0].click(); }); await settle();
}
async function clickNode(text: string) {
  const button = [...host.querySelectorAll('.thin-league-shell .node>.nb')]
    .find(node => node.textContent?.includes(text));
  expect(button, text).toBeDefined();
  await act(async () => { (button as HTMLButtonElement).click(); }); await settle();
}
async function input(selector: string, value: string) {
  const element = host.querySelector(selector)! as HTMLInputElement;
  const prototype = element.tagName === 'SELECT'
    ? HTMLSelectElement.prototype : HTMLTextAreaElement.prototype;
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(element, value);
    element.dispatchEvent(new Event(element.tagName === 'SELECT' ? 'change' : 'input', { bubbles: true }));
  });
}
async function openRules(port: HostedPort, role = 'Commissioner') {
  await act(async () => root.render(<HostedLeague league={league} franchise="" name=""
    role={role} port={port} />));
  await settle(); await clickNode('Control Room'); await click('Change Lineup lock');
}
async function preview() {
  await input('select[aria-label="New value for Lineup lock"]', 'first_kickoff');
  await input('textarea', 'League approved this change');
  await click('Preview rule change');
}
it('opens a named tray and Escape closes it, returning focus to the rule button', async () => {
  const port = service(); await openRules(port);
  const opener = buttons('Change Lineup lock')[0];
  const dialog = host.querySelector<HTMLElement>('[role="dialog"]')!;
  expect(dialog).not.toBeNull();
  expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)?.textContent)
    .toBe('Change Lineup lock');
  expect([...dialog.querySelectorAll('h1, h2, h3, h4, h5, h6')]
    .filter(heading => heading.textContent === 'Change Lineup lock')).toHaveLength(1);
  expect(opener.closest('[inert]')).not.toBeNull();
  await act(async () => dialog.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })));
  expect(host.querySelector('[role="dialog"]')).toBeNull();
  expect(opener.closest('[inert]')).toBeNull();
  expect(document.activeElement).toBe(opener);
});
it('the tray Close cancels the review and restores focus', async () => {
  const port = service(); await openRules(port); await preview();
  await click('Close rule editor');
  expect(host.querySelector('[role="dialog"]')).toBeNull();
  expect(host.querySelector('[aria-label="Confirm league change"]')).toBeNull();
  expect(document.activeElement).toBe(buttons('Change Lineup lock')[0]);
});
it('closing a loading preview discards its late reply', async () => {
  const serve = service();
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  const port: HostedPort = async request => {
    if (request.mode === 'preview') await gate;
    return serve(request);
  };
  const room = createHostedLeagueRoom(league, port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(<HostedRules rules={parseRules(rules())} commissioner room={room} />));
  await click('Change Lineup lock'); await preview();
  expect(room.peek().pending).toBe(true);
  await click('Close rule editor');
  expect(room.peek().pending).toBe(false);
  expect(room.peek().review).toBeUndefined();
  await click('Change Lineup lock');
  await act(async () => { release(); await gate; });
  expect(room.peek().pending).toBe(false);
  expect(room.peek().review).toBeUndefined();
  expect(host.querySelector('[aria-label="Confirm league change"]')).toBeNull();
});
it('cancel leaves an in-flight refresh pending and accepts its result', async () => {
  const serve = service();
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  const room = createHostedLeagueRoom(league, async request => { await gate; return serve(request); });
  const loading = room.refresh();
  room.cancel();
  expect(room.peek().pending).toBe(true);
  release(); await loading;
  expect(room.peek().pending).toBe(false);
  expect(room.peek().reading?.rules.league).toBe(league);
  expect(room.peek().stale).toBe(false);
});
it('closing during a submitted change still records and refreshes it', async () => {
  const serve = service();
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  const port: HostedPort = async request => {
    if (request.mode === 'execute') await gate;
    return serve(request);
  };
  const room = createHostedLeagueRoom(league, port);
  await act(async () => { await room.refresh(); });
  await act(async () => root.render(<HostedRules rules={parseRules(rules())} commissioner room={room} />));
  await click('Change Lineup lock'); await preview();
  let confirming!: Promise<void>;
  await act(async () => { confirming = room.confirm(); });
  await click('Close rule editor');
  expect(room.peek().pending).toBe(true);
  await act(async () => { release(); await confirming; });
  expect(room.peek().pending).toBe(false);
  expect(room.peek().reading?.rules.revision).toBe(8);
  expect(room.peek().message).toContain('server recorded your change');
});
it('commissioner previews and confirms a rule through the hosted port, then rereads the value', async () => {
  const port = service(); await openRules(port); await preview();
  const request = port.mock.calls.find(([r]) => r.mode === 'preview')![0];
  expect(request).toMatchObject({ command: 'commish.rules.set', mode: 'preview', arguments: {
    league, command: 'commish.rules.set', args: { field: 'lineup.lock', value: 'first_kickoff' },
    expected: 7, effective: 'next_rollover', reason: 'League approved this change', visibility: 'league',
  } });
  const dialog = host.querySelector('[role=dialog]')!;
  const confirm = buttons('Confirm league change')[0];
  expect(dialog.contains(confirm)).toBe(true);
  expect(confirm.closest('[inert]')).toBeNull();
  await act(async () => confirm.focus());
  expect(document.activeElement).toBe(confirm);
  await act(async () => confirm.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 650)); });
  await settle();
  expect(port.mock.calls.find(([r]) => r.mode === 'execute')![0]).toEqual({ ...request, mode: 'execute' });
  expect(host.querySelector('dd')?.textContent).toContain('first_kickoff');
  expect(port.mock.calls.filter(([r]) => r.command === 'commish.rules.read')).toHaveLength(2);
});
it.each([
  'A starter was not active on the roster at kickoff',
  'native change refused; reread before retrying',
])('shows the server refusal verbatim: %s', async reason => {
  const serve = service();
  const port: HostedPort = async request => {
    if (request.mode === 'execute') throw new Error(reason);
    return serve(request);
  };
  await openRules(port); await preview();
  const confirm = buttons('Confirm league change')[0];
  await act(async () => confirm.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 650)); });
  expect(host.textContent).toContain(reason);
  expect(host.textContent).not.toContain('The change was not confirmed');
});
it('a blocked preview shows the blocker and offers no confirmation or execute', async () => {
  const port = service(['Lineups are already locked']); await openRules(port); await preview();
  expect(host.textContent).toContain('Lineups are already locked');
  expect(buttons('Confirm league change')).toHaveLength(0);
  expect(port.mock.calls.some(([r]) => r.mode === 'execute')).toBe(false);
  await click('Cancel change');
  expect(host.querySelector('[aria-label="Confirm league change"]')).toBeNull();
});
it('owners see rule values but no Change controls', async () => {
  const room = createHostedLeagueRoom(league, service());
  await act(async () => root.render(
    <HostedRules rules={parseRules(rules())} commissioner={false} room={room} />));
  expect(host.textContent).toContain('individual');
  expect(buttons('Change Lineup lock')).toHaveLength(0);
  expect(host.querySelector('textarea')).toBeNull();
});
it('now-only rules offer only Now and submit that timing', async () => {
  const port = service([], true); await openRules(port);
  const selects = host.querySelectorAll('select');
  expect([...selects[1].options].map(o => o.value)).toEqual(['now']);
  await preview();
  const request = port.mock.calls.find(([r]) => r.mode === 'preview')![0];
  expect(request.arguments).toMatchObject({ effective: 'now' });
});
it('a refused rules refresh disables editing and never previews', async () => {
  const port = service(); await openRules(port);
  port.mockRejectedValue(new Error('Disconnected'));
  await click('Refresh league server');
  expect(buttons('Change Lineup lock')[0].disabled).toBe(true);
  expect(host.textContent).toContain('read-only');
  expect(port.mock.calls.some(([r]) => r.mode === 'preview')).toBe(false);
});
// Lazy Home and HostedLeague content can take longer to resolve during the full parallel suite.
it.each(['Commissioner', 'Co-commissioner'])(
  'NativeAccount admits a teamless %s to Pulse and rules without franchise reads', async role => {
    const port = service();
    const admission: NativeView = { server: 'https://league.example', league, name: 'Test League', role,
      franchise: '', franchiseName: '', message: 'Connected', seed: '', connection: '', helperId: '',
      confirmation: '', authGeneration: '1', seats: [] };
    vi.stubGlobal('fetch', vi.fn(async (url: string, init?: RequestInit) => {
      if (url === '/desktop/changes') return new Promise(() => {});
      if (url === '/desktop/workflow') {
        const request = JSON.parse(String(init?.body)) as NativeRequest;
        return { ok: true, json: async () => await port(request) };
      }
      return { ok: true, headers: new Headers({ 'Content-Type': 'application/json' }),
        json: async () => admission };
    }));
    await act(async () => root.render(<NativeAccount initial={admission} />));
    await vi.waitFor(() => expect(buttons('Open Pulse for Test League')).toHaveLength(1),
      { timeout: 10_000 });
    await click('Open Pulse for Test League');
    await vi.waitFor(() => expect(host.textContent).toContain('League Pulse'), { timeout: 10_000 });
    expect(host.textContent).not.toContain('MY MATCHUP');
    expect(host.querySelector('.mybox')).toBeNull();
    expect(port.mock.calls.some(([r]) => r.command === 'native.directory.read')).toBe(true);
    expect(port.mock.calls.some(([r]) => r.command === 'standings.read')).toBe(true);
    await clickNode('Control Room');
    expect(buttons('Change Lineup lock')).toHaveLength(1);
    await click('Change Lineup lock'); await preview();
    const dialog = host.querySelector('[role=dialog]')!;
    const confirm = buttons('Confirm league change')[0];
    expect(dialog.contains(confirm)).toBe(true);
    expect(confirm.closest('[inert]')).toBeNull();
    await act(async () => confirm.focus());
    expect(document.activeElement).toBe(confirm);
    await click('Cancel change'); await click('Close rule editor');
    for (const [r] of port.mock.calls) {
      expect(['lineup.read', 'native.assets.read', 'native.reserve.check', 'draft.read',
        'bids.read', 'waivers.read', 'trades.read']).not.toContain(r.command);
      if (r.command === 'week.read') expect(r.arguments).toMatchObject({ franchise: '', at: '' });
      else expect(r.arguments).not.toHaveProperty('franchise');
    }
    expect(buttons('Set lineup')).toHaveLength(0);
    expect(buttons('Roster and reserves')).toHaveLength(0);
    expect(buttons('Draft room')).toHaveLength(0);
    await click('← Home');
    await vi.waitFor(() => expect(buttons('Open Control Room for Test League')).toHaveLength(1),
      { timeout: 10_000 });
    await click('Open Control Room for Test League');
    expect(buttons('Change Lineup lock')).toHaveLength(1);
  }, 15_000);
it('a refused rules read leaves league-wide Pulse available', async () => {
  const port = service();
  const serve = port.getMockImplementation()!;
  port.mockImplementation(request => request.command === 'commish.rules.read'
    ? Promise.reject(new Error('Rules refused')) : serve(request));
  await act(async () => root.render(<HostedLeague league={league} franchise="" name=""
    role="Commissioner" port={port} />));
  await settle(); await settle();
  expect(host.textContent).toContain('Could not refresh your league');
  expect(host.textContent).toContain('Standings available');
  expect(port.mock.calls.some(([r]) => r.command === 'results.read')).toBe(true);
});
