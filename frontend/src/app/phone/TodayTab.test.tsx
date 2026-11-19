// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';
import { defaultLayout, type Layout } from '../shell/homeLayout';
import type { HomeBrief } from '../shell/homeReads';
import type { LeagueChoice } from '../shell/leaguePicker';
import { TodayTab } from './TodayTab';

const now = Date.parse('2026-11-15T12:00:00Z');
const league: LeagueChoice = { id: 'league', name: 'Test League', app: true, franchise: 'mine',
  franchiseName: 'My Team', role: 'Owner' };
const loaded = (): HomeBrief => ({ pulse: { league: 'league', name: 'Test League', season: 2026, week: 11,
  lastWeek: 18, weekNote: '', seats: [], fixtures: [], forms: {}, errors: [],
  odds: { games: [{ week: 11, home: 'mine', away: 'other', homeWin: 0.44 }], rows: [] } },
results: { league: 'league', season: 2026, week: 11, revision: 1, published: false, words: '',
  games: [{ week: 11, home: 'mine', away: 'other', homeName: 'Iron Curtain', awayName: 'Gridlock',
    status: 'in', homeProvisional: 24.2, awayProvisional: 25.8, words: '', why: [] }],
  scores: [], missed: [] },
trades: { league: 'league', franchise: 'mine', offers: [{ id: 'o1', status: 'offered',
  parties: ['mine', 'other'], accepted: ['other'], expiresAt: '2026-11-15T13:00:00Z',
  teams: [['mine', 'My Team'], ['other', 'Gridlock']] }] },
calendar: [], at: { pulse: '2026-11-15T12:00:00Z', results: '2026-11-15T12:00:00Z' }, errors: {},
} as unknown as HomeBrief);
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined; window.localStorage.clear();
});
function layoutPort(requests: NativeRequest[], state: { saved: Layout; revision: number }) {
  return vi.fn(async (request: NativeRequest) => {
    requests.push(request);
    if (request.command === 'home.layout.read') return { label: 'Home layout', writable: true,
      football: '', data: { layout: state.saved, revision: state.revision } };
    if (request.command === 'home.layout.set') {
      state.saved = (request.arguments as { layout: Layout }).layout; state.revision++;
      return { label: 'Home layout', writable: true, football: '',
        data: { layout: state.saved, revision: state.revision } };
    }
    throw new Error(`Unexpected request: ${request.command}`);
  }) as unknown as HostedPort;
}
async function show(readBrief: () => Promise<HomeBrief>, port: HostedPort, onOpen = vi.fn()) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(<TodayTab leagues={[league]} onOpen={onOpen} port={port}
    readBrief={readBrief} now={now} />));
  await act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
  return host;
}
const press = (label: string) => act(async () =>
  (host!.querySelector(`[aria-label="${label}"]`) as HTMLButtonElement).click());

describe('TodayTab', () => {
  it('loaded: lanes, live matchup and the account boards that save edits', async () => {
    const requests: NativeRequest[] = [];
    const state = { saved: defaultLayout(), revision: 0 };
    const onOpen = vi.fn();
    const view = await show(async () => loaded(), layoutPort(requests, state), onOpen);
    await act(async () => {
      for (let i = 0; i < 50 && !view.querySelector('[data-testid="home-brief"]'); i++)
        await new Promise(resolve => setTimeout(resolve, 10));
    });
    expect(view.querySelector('h2')!.textContent).toBe('Sunday · week 11');
    expect(view.textContent).toContain('1 league · 1 playing now');
    expect(view.textContent).toContain('1 open');
    expect(view.querySelector('.today6__job.is-now small')!.textContent).toBe('NOW · TEST LEAGUE');
    expect(view.textContent).not.toContain('Done');
    expect(view.querySelector('[aria-label="Snooze until tomorrow"]')).toBeNull();
    const live = view.querySelector('.today6__live')!;
    expect(live.textContent).toContain('Iron Curtain vs Gridlock');
    expect(live.textContent).toContain('24.2–25.8');
    expect(live.textContent).toContain('44%');
    await press('Open Live for Test League');
    expect(onOpen).toHaveBeenCalledWith(league, 'pulse');
    const customize = [...view.querySelectorAll('button')].find(b => b.textContent?.includes('CUSTOMIZE'))!;
    await act(async () => customize.click());
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
    await press('Move THIS WEEK down');
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
    expect(requests.map(r => r.command)).toEqual(['home.layout.read', 'home.layout.set']);
    expect(state.revision).toBe(1);
  });

  it('empty: a clean read says nothing needs you', async () => {
    const empty = { ...loaded(),
      trades: { league: 'league', franchise: 'mine', offers: [] } } as unknown as HomeBrief;
    const view = await show(async () => empty, layoutPort([], { saved: defaultLayout(), revision: 0 }));
    expect(view.textContent).toContain('Nothing needs you right now.');
  });

  it('unavailable: a failed read never shows the all-clear', async () => {
    const view = await show(async () => { throw new Error('Server unreachable'); },
      layoutPort([], { saved: defaultLayout(), revision: 0 }));
    expect(view.textContent).not.toContain('Nothing needs you right now.');
    expect(view.textContent).toContain('Server unreachable');
    expect(view.querySelector('h2')!.textContent).toBe('Sunday · week unavailable');
  });

  it('snooze: a commissioner job gets the clock, which writes job.snooze.set and hides it', async () => {
    const commish = { ...league, role: 'Commissioner' };
    const brief = { ...loaded(), trades: { league: 'league', franchise: 'mine', offers: [] },
      jobs: { league: 'league', name: 'Test League', revision: 1, jobs: [{ id: 'j1', name: 'Approve trade',
        state: 'blocked', words: 'Waiting', opens: '', closes: '2026-11-15T13:00:00Z', controls: [] }] },
    } as unknown as HomeBrief;
    const requests: NativeRequest[] = [];
    const base = layoutPort(requests, { saved: defaultLayout(), revision: 0 });
    const port = vi.fn(async (request: NativeRequest) => {
      if (request.command === 'job.snooze.set') { requests.push(request);
        return { label: '', writable: true, football: '', data: {} }; }
      return base(request);
    }) as unknown as HostedPort;
    const onNow = vi.fn();
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    await act(async () => root!.render(<TodayTab leagues={[commish]} onOpen={vi.fn()} port={port}
      readBrief={async () => brief} now={now} onNow={onNow} />));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
    expect(host.textContent).toContain('Approve trade');
    expect(onNow).toHaveBeenLastCalledWith(1);
    await press('Snooze until tomorrow');
    const sent = requests.find(r => r.command === 'job.snooze.set');
    expect(sent?.arguments).toMatchObject({ league: 'league', jobId: 'j1' });
    expect(host.querySelector('[aria-label="Needs you"]')!.textContent).not.toContain('Approve trade');
    expect(onNow).toHaveBeenLastCalledWith(0);
  });

  it('live now: one row per admitted league, each with its own read or failure words', async () => {
    const other: LeagueChoice = { ...league, id: 'f32', name: 'Foundry 32' };
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    host = document.createElement('div'); document.body.append(host); root = createRoot(host);
    const onOpen = vi.fn();
    await act(async () => root!.render(<TodayTab leagues={[league, other]} onOpen={onOpen}
      port={layoutPort([], { saved: defaultLayout(), revision: 0 })} now={now}
      readBrief={async id => { if (id === 'f32') throw new Error('Not admitted to this league');
        return loaded(); }} />));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 30)); });
    const rows = [...host.querySelectorAll('[aria-label="Live now"] .today6__live')];
    expect(rows.map(r => r.textContent)).toEqual([expect.stringContaining('Iron Curtain vs Gridlock'),
      'Foundry 32Open to see scores›']);
    expect(host.querySelector('[aria-label="Live now"]')!.textContent).not.toContain('Not admitted');
    await press('Open Foundry 32 to see scores');
    expect(onOpen).toHaveBeenCalledWith(other, 'pulse');
    expect(host.textContent).toContain('2 leagues');
  });
});
