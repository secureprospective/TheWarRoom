import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HostedPort } from '../commissioner/hostedRoom';
import { all, button, click, disabled, mount, render, settle, unmount }
  from '../commissioner/commissionerDom';
import type { HomeBrief } from './homeReads';
import { ThinHome } from './ThinHome';
import type { LeagueChoice } from './leaguePicker';

const now = Date.parse('2026-11-18T00:00:00Z');
const at = (hour: number) => `2026-11-18T${String(hour).padStart(2, '0')}:00:00Z`;
const leagues: LeagueChoice[] = [
  { id: 'alpha', name: 'Alpha League', app: true, franchise: 'alpha-team',
    franchiseName: 'Alpha Team', role: 'Owner' },
  { id: 'beta', name: 'Beta League', app: true, franchise: 'beta-team',
    franchiseName: 'Beta Team', role: 'Commissioner' },
];
const brief = (league: string, franchise: string): HomeBrief => ({ at: {}, errors: {},
  pulse: { league, name: league, season: 2026, revision: 1, seats: [], week: 8, weekNote: '',
    lastWeek: 8, fixtures: [], errors: [], standings: { rows: [{ franchise, record: '3-2' }] } } as never,
  calendar: [{ id: 'trade-window', label: 'Trade window', at: at(2) }] as never,
});
afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

describe('ThinHome', () => {
  it('renders multiple leagues, sorted Needs-you items and commissioner-only shortcuts', async () => {
    const onOpen = vi.fn();
    const readBrief = vi.fn(async (league: string, _mine: string) => {
      const reading = brief(league, `${league}-team`);
      if (league === 'alpha') reading.trades = { league, franchise: 'alpha-team', offers: [{
        id: 'offer-1', status: 'offered', parties: ['alpha-team', 'other'], accepted: ['other'],
        expiresAt: at(4), teams: [['alpha-team', 'Alpha Team'], ['other', 'Other Team']],
      }] } as never;
      if (league === 'beta') {
        reading.calendar = [];
        reading.jobs = { league, jobs: [{ id: 'job-1', name: 'Review the season',
          state: 'blocked', words: 'Commissioner action required', closes: '' }] } as never;
      }
      return reading;
    });
    const host = await mount(render(ThinHome, { leagues, onOpen, readBrief, now }));
    await settle();

    expect(host.textContent).toContain('Alpha League');
    expect(host.textContent).toContain('Beta League');
    expect(host.textContent).toContain('Alpha Team');
    expect(host.textContent).toContain('3-2');
    expect(host.textContent).toContain('Trade window');
    const items = all(host, 'li').map(item => item.textContent);
    expect(items[0]).toContain('Trade window');
    expect(items[1]).toContain('Trade offer from Other Team');
    expect(items[2]).toContain('Review the season');
    expect(all(host, 'button').filter(item => item.getAttribute('aria-label')
      === 'Open Control Room for Alpha League')).toHaveLength(0);
    expect(button(host, 'Open Control Room for Beta League')).toBeTruthy();

    await click(button(host, 'Open Beta League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[1], 'hq');
    onOpen.mockClear();
    await click(button(host, 'Open Trade window · League deadline this week in Alpha League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[0], 'trades');
    onOpen.mockClear();
    await click(button(host, 'Open Franchise HQ for Alpha League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[0], 'hq');
    onOpen.mockClear();
    await click(button(host, 'Open War Room for Alpha League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[0], 'war');
    onOpen.mockClear();
    await click(button(host, 'Open Pulse for Beta League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[1], 'pulse');
    onOpen.mockClear();
    await click(button(host, 'Open Control Room for Beta League'));
    expect(onOpen).toHaveBeenCalledWith(leagues[1], 'control');
  });

  it('keeps a failed league unavailable without hiding the other league', async () => {
    const readBrief = vi.fn(async (league: string) => {
      if (league === 'alpha') throw new Error('Alpha read was refused');
      return brief(league, 'beta-team');
    });
    const host = await mount(render(ThinHome, { leagues, onOpen: vi.fn(), readBrief, now }));
    await settle();

    expect(host.textContent).toContain('Not available yet · Alpha read was refused');
    expect(host.textContent).toContain('Beta League');
    expect(host.textContent).toContain('3-2');
    expect(all(host, 'article')).toHaveLength(2);
  });

  it('keeps Needs-you actions and failed reads usable at the 390px phone breakpoint', async () => {
    const phoneWidth = 390;
    expect(phoneWidth).toBeLessThan(700);
    const readBrief = vi.fn(async () => { throw new Error('Calendar read refused'); });
    const host = await mount(render(ThinHome, { leagues: [leagues[0]], onOpen: vi.fn(),
      readBrief, now }));
    await settle();

    expect(all(host, 'h2').some(heading => heading.textContent === 'NEEDS YOU')).toBe(true);
    expect(host.textContent).toContain('Not available yet · Calendar read refused');
    const unavailableItem = all(host, 'button').find(item => item.textContent
      .includes('Calendar read refused'));
    expect(unavailableItem).toBeTruthy();
    expect(disabled(unavailableItem!)).toBe(true);
  });

  it('uses only the admitted league and never requests the server-wide league list', async () => {
    const requests: { command: string; arguments: Record<string, unknown> }[] = [];
    const port = vi.fn(async request => {
      requests.push(request as typeof requests[number]);
      throw new Error('read unavailable');
    }) as unknown as HostedPort;
    const host = await mount(render(ThinHome, { leagues: [leagues[0]], onOpen: vi.fn(), port, now }));
    await settle();

    expect(requests.length).toBeGreaterThan(0);
    expect(requests.some(request => request.command === 'commish.leagues.read')).toBe(false);
    expect(new Set(requests.map(request => request.arguments.league))).toEqual(new Set(['alpha']));
    expect(host.textContent).toContain('Alpha League');
  });
});
