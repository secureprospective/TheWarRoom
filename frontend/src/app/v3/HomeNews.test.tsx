// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Brief, V3League } from './model';
import { HomeBoard } from './pieces';

let root: Root | undefined;
let host: HTMLDivElement | undefined;

afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  root = undefined;
  host?.remove();
  host = undefined;
  vi.restoreAllMocks();
});

const prov = {
  source: 'League ledger', kind: 'live' as const,
  freshness: { state: 'live' as const, fetchedAt: '2026-11-21T12:00:00Z', note: '' },
};

const league = {
  id: 'native.legacy', name: 'Legacy', app: true, franchise: 'f1', franchiseName: 'Mine',
  bd: 'LG', c: 'hsl(172,30%,52%)', commish: false, team: true,
} as unknown as V3League;

const now = Date.parse('2026-11-21T16:00:00Z');

const brief: Brief = {
  brief: {
    at: {}, errors: {},
    assets: {
      league: { value: { season: 2026, franchiseCount: 12 }, provenance: prov },
      franchises: { value: [{ id: 'f1', name: 'Mine' }, { id: 'f2', name: 'Rival' }], provenance: prov },
      rosters: {
        value: [
          {
            franchiseId: 'f1',
            players: [{ id: '13116', rosterStatus: 'ROSTER', salary: 500_000_000, contractStatus: 'RFA' }],
          },
          {
            franchiseId: 'f2',
            players: [{ id: '14836', rosterStatus: 'ROSTER', salary: 600_000_000, contractStatus: 'UFA' }],
          },
        ],
        provenance: prov,
      },
      players: {
        value: [
          { id: '13116', name: 'Patrick Mahomes', position: 'QB', team: 'KC' },
          { id: '14836', name: 'Justin Jefferson', position: 'WR', team: 'MIN' },
          { id: '16900', name: 'Wire Sleeper', position: 'RB', team: 'BUF' },
        ],
        provenance: prov,
      },
      eligibility: { players: {}, feeds: { injuries: prov, players: prov, rosters: prov } },
    },
  },
  news: {
    league: 'native.legacy',
    asOf: '2026-11-21T15:30:00Z',
    stale: false,
    items: [
      {
        id: 'n1', headline: 'Mahomes sharp in Friday practice',
        description: 'Kansas City QB ready for Sunday.',
        published: '2026-11-21T14:00:00Z',
        url: 'https://www.espn.com/nfl/story/_/id/n1',
        source: 'ESPN', type: 'Story', players: ['13116'],
      },
      {
        id: 'n2', headline: 'Jefferson questionable with hamstring issue',
        description: 'Listed questionable on Friday injury report.',
        published: '2026-11-21T13:00:00Z',
        url: 'https://www.espn.com/nfl/story/_/id/n2',
        source: 'ESPN', type: 'HeadlineNews', injury: true, players: ['14836'],
      },
      {
        id: 'n3', headline: 'Wire Sleeper elevated for Week 11',
        description: 'Unrostered back could see touches.',
        published: '2026-11-21T12:00:00Z',
        url: 'https://www.espn.com/nfl/story/_/id/n3',
        source: 'ESPN', type: 'Story', players: ['16900'],
      },
    ],
  },
};

describe('Home NEWS panel', () => {
  it('shows real chip counts, filters All / My players / Injuries / On the wire, and opens player card',
    async () => {
      Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
      const openPlayer = vi.fn();
      host = document.createElement('div');
      document.body.append(host);
      root = createRoot(host);

      await act(async () => root!.render(
        <HomeBoard c={{
          leagues: [league], briefs: { 'native.legacy': brief }, docket: [],
          now, open: () => {}, openPlayer, loading: false,
        }} />,
      ));

      const panel = host.querySelector('[data-iid="nflnews"]')!;
      expect(panel).toBeTruthy();
      const chips = [...panel.querySelectorAll('.fbar button')];
      expect(chips.map(c => c.textContent)).toEqual([
        'All 3', 'My players 1', 'Injuries 1', 'On the wire 1',
      ]);
      expect(panel.textContent).toContain('CT · ESPN');
      expect(panel.querySelectorAll('.news')).toHaveLength(3);

      // Click My players chip
      await act(async () => (chips[1] as HTMLElement).click());
      expect(panel.querySelectorAll('.news')).toHaveLength(1);
      expect(panel.textContent).toContain('Mahomes sharp in Friday practice');
      expect(panel.textContent).not.toContain('Wire Sleeper elevated');

      // Click Injuries chip
      await act(async () => (chips[2] as HTMLElement).click());
      expect(panel.querySelectorAll('.news')).toHaveLength(1);
      expect(panel.textContent).toContain('Jefferson questionable');

      // Click On the wire chip
      await act(async () => (chips[3] as HTMLElement).click());
      expect(panel.querySelectorAll('.news')).toHaveLength(1);
      expect(panel.textContent).toContain('Wire Sleeper elevated');

      // Click player name button -> opens that player's card
      const playerBtn = panel.querySelector<HTMLButtonElement>('[data-a="player"]')!;
      expect(playerBtn.textContent).toBe('Wire Sleeper');
      await act(async () => playerBtn.click());
      expect(openPlayer).toHaveBeenCalledWith(league, '16900');

      // Article button opens in a new tab
      const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
      const linkBtn = panel.querySelector<HTMLButtonElement>('.news .espn-news-link')!;
      expect(linkBtn.dataset.url).toBe('https://www.espn.com/nfl/story/_/id/n3');
      await act(async () => linkBtn.click());
      expect(openSpy).toHaveBeenCalledWith(
        'https://www.espn.com/nfl/story/_/id/n3', '_blank', 'noopener,noreferrer',
      );
    });
});
