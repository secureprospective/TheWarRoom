// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { LiveTab } from './LiveTab';

const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
const port = vi.fn() as unknown as HostedPort;
const brief = (games: unknown[]): HomeBrief => ({ pulse: { league: 'league', name: 'Test League',
  season: 2026, week: 11, lastWeek: 18, weekNote: '', seats: [], fixtures: [], forms: {}, errors: [],
  odds: { games: [{ week: 11, home: 'mine', away: 'other', homeWin: 0.44 }], rows: [] } },
results: { league: 'league', season: 2026, week: 11, revision: 1, published: false, words: '',
  games, scores: [], missed: [] }, at: {}, errors: {} } as unknown as HomeBrief);
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
});
async function show(readBrief: () => Promise<HomeBrief>) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<LiveTab league={league} port={port} readBrief={readBrief}>
    <p>pulse now</p></LiveTab>));
  return host.querySelector('[aria-label="My matchup"]')!;
}

describe('LiveTab', () => {
  it('shows my matchup score and the held win chance when loaded', async () => {
    const card = await show(async () => brief([{ week: 11, home: 'mine', away: 'other',
      homeName: 'Iron Curtain', awayName: 'Gridlock', status: 'in', homeProvisional: 24.2,
      awayProvisional: 25.8, words: '', why: [] }]));
    expect(card.textContent).toContain('My matchup · wk 11');
    expect(card.textContent).toContain('Iron Curtain · YOU');
    expect(card.textContent).toContain('24.2');
    expect(card.textContent).toContain('25.8');
    expect(card.textContent).toContain('44% WIN');
    expect(host!.textContent).toContain('pulse now');
  });
  it('says there is no game when the week holds none for me', async () => {
    const card = await show(async () => brief([]));
    expect(card.textContent).toContain('No game this week');
    expect(card.textContent).not.toContain('WIN');
  });
  it('shows the read failure words when the read fails', async () => {
    const card = await show(async () => { throw new Error('results.read refused'); });
    expect(card.querySelector('[role="alert"]')?.textContent).toContain('results.read refused');
  });
});
