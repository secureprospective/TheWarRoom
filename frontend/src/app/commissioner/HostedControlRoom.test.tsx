// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import HostedLeague from './HostedLeague';
import type { HostedPort } from './hostedRoom';
import type { NativeRequest } from '../commands/native';

const league = 'native.test';
const ruleReading = {
  league, name: 'Test League', revision: 7, rollover: 'No rollover date is set', sections: [],
};
function jobs(count: number) {
  return { league, name: 'Test League', revision: 7, jobs: Array.from({ length: count }, (_, i) => ({
    id: i === 0 ? 'rollover' : `job-${i + 1}`, name: i === 0 ? 'Rollover' : `Job ${i + 1}`,
    state: 'upcoming', words: 'Open the commissioner tool.', opens: '', closes: '',
    controls: i === 0 ? ['CM-027'] : [],
  })) };
}
function ledger() {
  return { league, name: 'Test League', revision: 7,
    entries: Array.from({ length: 32 }, (_, i) => ({
      id: `entry-${i}`, who: 'Commissioner', what: `Change ${i + 1}`, why: 'Test reason',
      when: '2026-11-17T12:00:00Z', gravity: 'G1', undo: '',
    })) };
}
function port(count: number): HostedPort & ReturnType<typeof vi.fn> {
  return vi.fn(async (request: NativeRequest) => {
    let data: unknown;
    if (request.command === 'commish.rules.read') data = ruleReading;
    else if (request.command === 'commish.jobs.read') data = jobs(count);
    else if (request.command === 'commish.ledger.read') data = ledger();
    else throw new Error(`Unexpected hosted request: ${request.command}`);
    return { data, label: 'League server', writable: true, football: 'held-pin' };
  }) as HostedPort & ReturnType<typeof vi.fn>;
}
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
});
afterEach(async () => { await act(async () => root.unmount()); host.remove(); vi.restoreAllMocks(); });
async function settle() { await act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); }); }

it.each(['Commissioner', 'Co-commissioner'])(
  'mounts the v3 hosted tabs for %s and reuses one jobs read for the count', async role => {
  const read = port(32);
  await act(async () => root.render(<HostedLeague league={league} franchise="" name="Test League"
    role={role} port={read} />));
  const controlRoom = [...host.querySelectorAll('button')]
    .find(button => button.textContent === 'Control Room');
  expect(controlRoom).toBeDefined();
  await act(async () => controlRoom!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
  });
  const tabs = [...host.querySelectorAll('[role="tab"]')].map(button =>
    button.textContent?.replace(/\s+\d+$/, '').trim());
  expect(tabs).toEqual([
    'Jobs', 'Situations', 'Rules', 'Season', 'Fix', 'Picks', 'People', 'League', 'Server',
  ]);
  expect(host.textContent).toContain('32 from the league job read');
  expect(host.querySelector('[aria-label="League health"]')?.textContent).toContain('32');
  expect(read.mock.calls.filter(([request]) => request.command === 'commish.jobs.read')).toHaveLength(1);

  const rollover = host.querySelector<HTMLButtonElement>('[data-job="rollover"]');
  expect(rollover).not.toBeNull();
  await act(async () => rollover!.click());
  await settle();
  expect(host.textContent).toContain('Preview move to the next season');
  },
);

it('shows the hosted server ledger at 32 entries and leaves operator actions unavailable', async () => {
  const read = port(32);
  await act(async () => root.render(<HostedLeague league={league} franchise=""
    name="Test League" role="Commissioner" port={read} />));
  const controlRoom = [...host.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent === 'Control Room');
  await act(async () => controlRoom!.click());
  await settle();
  const serverTab = [...host.querySelectorAll<HTMLButtonElement>('[role="tab"]')]
    .find(button => button.textContent?.includes('Server'));
  await act(async () => serverTab!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.textContent).toContain('Change 32');
  });
  expect(read.mock.calls.some(([request]) => request.command === 'commish.ledger.read')).toBe(true);
  expect(host.textContent).toContain('operator-only');
  expect(host.querySelectorAll('[aria-label="Server status"] ol li')).toHaveLength(32);
});

it('opens the hosted frame for an assigned commissioner too', async () => {
  const read = port(0);
  await act(async () => root.render(<HostedLeague league={league} franchise="franchise.01"
    name="Franchise One" role="Commissioner" port={read} />));
  const controlRoom = [...host.querySelectorAll('button')]
    .find(button => button.textContent === 'Control Room');
  expect(controlRoom).toBeDefined();
  await act(async () => controlRoom!.click());
  await vi.waitFor(async () => {
    await settle();
    expect(host.querySelector('[data-testid="hosted-control-room"]')).not.toBeNull();
  });
  expect(read.mock.calls.some(([request]) => request.command === 'commish.jobs.read')).toBe(true);
});

it('does not mount or read Control Room jobs for a plain hosted owner', async () => {
  const read = port(0);
  await act(async () => root.render(<HostedLeague league={league} franchise="franchise.01"
    name="Franchise One" role="Owner" port={read} />));
  await settle();
  expect(host.querySelector('[data-testid="hosted-control-room"]')).toBeNull();
  const visibleControlRoom = [...host.querySelectorAll('button')]
    .some(button => button.textContent?.includes('Control Room'));
  expect(visibleControlRoom).toBe(false);
  expect(read.mock.calls.some(([request]) => request.command === 'commish.jobs.read')).toBe(false);
});
