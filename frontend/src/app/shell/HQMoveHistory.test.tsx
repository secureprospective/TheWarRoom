import { afterEach, expect, it, vi } from 'vitest';
import { HQMoveHistory } from './HQMoveHistory';
import { HQMoves } from './HQMoves';
import { heldSnapshot } from '../data/lineupTestData';
import { mount, flush, unmount, button, click, rerender } from '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';

const entry = (id: string, what: string) => ({ id, what, who: 'via AI/CLI, acting for owner',
  when: '2026-11-17T12:00:00Z', why: 'Recorded reason', gravity: 'Recorded reason required', undo: '' });
const page = (entries = [entry('original', 'Released a player'),
  entry('correction', 'Reversed player release')]) =>
  ({ league: 'native.test', name: 'League', revision: 7, entries });
const response = (data: unknown) => ({ data, label: 'Hosted league', writable: true, football: '' });
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });
it('renders hosted HQ history and correction rows, with season paging and no local read', async () => {
  const port = vi.fn<HostedPort>(async request => response(
    request.command === 'franchise.moves.history.read' && request.arguments.season === 2026
      ? page() : page([])));
  const host = await mount(<HQMoves native moves={[]} snapshot={heldSnapshot()}
    history={<HQMoveHistory league="native.test" franchise="franchise.01" season={2026} port={port} />} />);
  await flush();
  expect(host.textContent).toContain('Released a player');
  expect(host.textContent).toContain('Reversed player release');
  expect(host.textContent).toContain('via AI/CLI, acting for owner');
  expect(port).toHaveBeenCalledWith({ command: 'franchise.moves.history.read', mode: 'read',
    arguments: { league: 'native.test', franchise: 'franchise.01', season: 2026 } });
  expect(button(host, 'Next season').attributes.has('disabled')).toBe(true);
  await click(button(host, 'Previous season')); await flush();
  expect(host.textContent).toContain('No changes visible to you for this franchise in 2025.');
  expect(host.textContent).not.toContain('Released a player');
  await click(button(host, 'Next season')); await flush();
  expect(host.textContent).toContain('Released a player');
});
it('shows refusal with retry, and fences a late prior-season response', async () => {
  let finish: (value: ReturnType<typeof response>) => void = () => {};
  const port = vi.fn<HostedPort>().mockImplementationOnce(() =>
    new Promise(resolve => { finish = resolve; }))
    .mockRejectedValueOnce(new Error('Seat refused')).mockResolvedValue(response(page([])));
  const host = await mount(<HQMoveHistory league="native.test" franchise="franchise.01"
    season={2026} port={port} />);
  expect(host.textContent).toContain('Reading season history');
  await click(button(host, 'Previous season')); await flush();
  expect(host.textContent).toContain('History unavailable. Refresh history to try again.');
  finish(response(page())); await flush();
  expect(host.textContent).not.toContain('Released a player');
  await click(button(host, 'Refresh history')); await flush();
  expect(host.textContent).toContain('No changes visible to you');
});
it('does not let a delayed prior franchise history replace the new selection', async () => {
  let finish: (value: ReturnType<typeof response>) => void = () => {};
  const port = vi.fn<HostedPort>().mockImplementationOnce(() =>
    new Promise(resolve => { finish = resolve; }))
    .mockResolvedValue(response(page([])));
  const host = await mount(<HQMoveHistory league="native.test" franchise="franchise.01"
    season={2026} port={port} />);
  await rerender(<HQMoveHistory league="native.test" franchise="franchise.02" season={2026} port={port} />);
  await flush(); finish(response(page())); await flush();
  expect(host.textContent).not.toContain('Released a player');
  expect(host.textContent).toContain('No changes visible to you');
});
it('leaves the legacy desktop empty ledger path unchanged', async () => {
  const host = await mount(<HQMoves moves={[]} snapshot={heldSnapshot()} />);
  expect(host.textContent).toContain('every move you made, as recorded');
  expect(host.textContent).not.toContain('Previous season');
});
