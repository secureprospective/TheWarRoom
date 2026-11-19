// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import { ThinHome } from './ThinHome';
import type { HomeBrief } from './homeReads';
import { defaultLayout, type Layout } from './homeLayout';
import type { LeagueChoice } from './leaguePicker';

const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
const brief = { pulse: { league: 'league', name: 'Test League', season: 2026, week: 8, lastWeek: 18,
  weekNote: '', seats: [], fixtures: [], forms: {}, errors: [] },
  results: { league: 'league', season: 2026, week: 8, revision: 1, published: true, words: '',
    games: [{ week: 8, home: 'mine', away: 'other', homeName: 'My Team', awayName: 'Rivals',
      status: 'in', words: '', why: [] }], scores: [
      { franchise: 'mine', name: 'My Team', points: 28.4, starters: [] },
      { franchise: 'other', name: 'Rivals', provisional: 24.1, starters: [] }], missed: [] },
  at: { pulse: '2026-11-18T12:00:00Z', results: '2026-11-18T12:00:00Z' }, errors: {},
} as unknown as HomeBrief;
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  Reflect.deleteProperty(window, 'matchMedia');
  Reflect.deleteProperty(window, 'innerWidth');
  window.localStorage.clear();
});
function setPhoneViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
  Object.defineProperty(window, 'matchMedia', { configurable: true, value: vi.fn((media: string) => ({
    media, matches: width < 700, addEventListener: vi.fn(), removeEventListener: vi.fn(),
  })) });
}

it('shows the synced Home boards on Today and writes phone edits to the account preference', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setPhoneViewport(390);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  const requests: NativeRequest[] = [];
  let saved = defaultLayout();
  let revision = 0;
  const port = vi.fn(async (request: NativeRequest) => {
    requests.push(request);
    if (request.command === 'home.layout.read') return { label: 'Home layout', writable: true,
      football: '', data: { layout: saved, revision } };
    if (request.command === 'home.layout.set') {
      const args = request.arguments as { layout: Layout; expectedRevision: number };
      if (args.expectedRevision !== revision) return { label: 'Home layout', writable: true,
        football: '', data: { layout: saved, revision, conflict: true } };
      saved = args.layout; revision++;
      return { label: 'Home layout', writable: true, football: '', data: { layout: saved, revision } };
    }
    throw new Error(`Unexpected request: ${request.command}`);
  }) as unknown as HostedPort;
  const readBrief = vi.fn(async () => brief);
  await act(async () => root!.render(<ThinHome leagues={[league]} onOpen={vi.fn()} port={port}
    readBrief={readBrief} now={Date.parse('2026-11-18T12:00:00Z')} />));
  await act(async () => {
    for (let attempt = 0; attempt < 50 && !host!.querySelector('[data-testid="home-brief"]'); attempt++)
      await new Promise(resolve => setTimeout(resolve, 10));
  });

  expect(host.textContent).toContain('MY BOARDS');
  expect(host.querySelector('[data-testid="home-brief"]')).not.toBeNull();
  expect(requests.map(request => request.command)).toEqual(['home.layout.read']);
  const ask = host.querySelector('.phone-ask__open') as HTMLButtonElement;
  await act(async () => ask.click());
  const suggestion = [...host.querySelectorAll('.phone-ask__suggestions button')][0] as HTMLButtonElement;
  await act(async () => suggestion.click());
  expect(host.querySelector('.phone-ask__answer')?.textContent).toContain('You are ahead of Rivals');
  expect(requests.map(request => request.command)).toEqual(['home.layout.read']);

  const customize = [...host.querySelectorAll('button')].find(button =>
    button.textContent?.includes('CUSTOMIZE'))!;
  await act(async () => customize.click());
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
  const move = host.querySelector('[aria-label="Move THIS WEEK down"]') as HTMLButtonElement;
  expect(move).not.toBeNull();
  await act(async () => move.click());
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
  expect(requests.map(request => request.command)).toEqual(['home.layout.read', 'home.layout.set']);
  expect(saved.tiles.slice(0, 2).map(([id]) => id)).toEqual(['news', 'mu']);
  expect(revision).toBe(1);
});
