import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { hostedWorkflow, type HostedResponse } from './api';
import type { NativeRequest } from '../commands/native';
import { parseDraft, type Draft } from '../draftroom/draft';
import { parseRules, type Rules } from './rules';
import { parsePreview, type Preview } from './preview';
import { parseReceipt } from './proxy';
import {
  parseHostedLineup,
  parseHostedReserves,
  parseHostedRoster,
  type HostedLineup,
  type HostedReserves,
  type HostedRoster,
} from './hostedReadings';

import {
  parseHostedMarket,
  parseHostedBids,
  parseHostedWaivers,
  parseHostedTrades,
  parseHostedResults,
  parseHostedStandings,
  type HostedMarkets,
} from './hostedMarkets';

export type HostedReading = {
  markets: HostedMarkets;
  lineup: HostedLineup;
  roster: HostedRoster;
  reserves: HostedReserves;
  draft: Draft;
  rules: Rules;
  label: string;
  writable: boolean;
  football: string;
  readAt: number;
};
type Review = {
  request: NativeRequest;
  preview: Preview | null;
  summary: string;
};
type State = {
  reading?: HostedReading;
  stale: boolean;
  pending: boolean;
  error: string;
  message: string;
  review?: Review;
  market?: ReturnType<typeof parseHostedMarket>;
};
export type HostedPort = (request: NativeRequest, football?: string) => Promise<HostedResponse>;

// One signed-in franchise, held only in memory. No local store, queue or authoritative fallback.
export function createHostedRoom(
  league: string,
  franchise: string,
  port: HostedPort = hostedWorkflow,
) {
  const store = createStore<State>(() => ({
    stale: true,
    pending: false,
    error: '',
    message: '',
  }));
  let generation = 0;
  let activeWeek = 1;
  async function refresh(week = activeWeek) {
    if (store.getState().pending) return;
    activeWeek = week;
    const turn = ++generation;
    store.setState({ pending: true, review: undefined, market: undefined });
    try {
      const assets = await port({
        command: 'native.assets.read',
        mode: 'read',
        arguments: { league },
      });
      const roster = parseHostedRoster(assets.data);
      const season = roster.season;
      const requests: NativeRequest[] = [
        {
          command: 'lineup.read',
          mode: 'read',
          arguments: { league, franchise, week, at: '' },
        },
        {
          command: 'native.reserve.check',
          mode: 'read',
          arguments: { league },
        },
        {
          command: 'draft.read',
          mode: 'read',
          arguments: {
            league,
            franchise,
            season: 0,
            at: '0001-01-01T00:00:00Z',
          },
        },
        { command: 'commish.rules.read', mode: 'read', arguments: { league } },
        {
          command: 'bids.read',
          mode: 'read',
          arguments: { league, franchise },
        },
        {
          command: 'waivers.read',
          mode: 'read',
          arguments: { league, franchise, at: new Date().toISOString() },
        },
        {
          command: 'trades.read',
          mode: 'read',
          arguments: { league, viewer: franchise },
        },
        {
          command: 'results.read',
          mode: 'read',
          arguments: { league, season, week },
        },
        {
          command: 'standings.read',
          mode: 'read',
          arguments: { league, season },
        },
      ];
      const results = await Promise.all(
        requests.map((request) =>
          request.command === 'lineup.read'
            ? port(request).catch(() => ({ ...assets, data: null }))
            : port(request),
        ),
      );
      const [lineup, reserves, draft, rules, bids, waivers, trades, games, standings] = results;
      const reading: HostedReading = {
        markets: {
          bids: parseHostedBids(bids.data),
          waivers: parseHostedWaivers(waivers.data),
          trades: parseHostedTrades(trades.data),
          results: parseHostedResults(games.data),
          standings: parseHostedStandings(standings.data),
        },
        lineup:
          lineup.data === null
            ? {
                available: false,
                league,
                franchise,
                season,
                week,
                revision: roster.revision,
                words:
                  'The lineup could not be read. Ask the server operator to check football evidence ' +
                'for this season, then refresh.',
                verdict: 'Lineup changes are unavailable.',
                starters: [],
                bench: [],
              }
            : parseHostedLineup(lineup.data),
        roster,
        reserves: parseHostedReserves(reserves.data),
        draft: parseDraft(draft.data),
        rules: parseRules(rules.data),
        label: lineup.label,
        writable: results.every((r) => r.writable),
        football: lineup.football,
        readAt: Date.now(),
      };
      if (
        reading.lineup.league !== league ||
        reading.lineup.franchise !== franchise ||
        reading.draft.league !== league ||
        reading.rules.league !== league ||
        reading.roster.league !== league ||
        [reading.lineup, reading.draft, reading.rules, ...Object.values(reading.markets)].some(
          (r) => r.revision !== roster.revision,
        ) ||
        Object.values(reading.markets).some((r) => r.league !== league) ||
        [reading.markets.bids, reading.markets.waivers, reading.markets.trades].some(
          (r) => r.franchise !== franchise,
        ) ||
        results.some((r) => r.football !== reading.football)
      )
        throw new Error('League readings changed. Refresh again.');
      if (turn === generation) {
        store.setState({ reading, stale: false, error: '' });
      }
    } catch {
      if (turn === generation)
        store.setState({
          stale: true,
          error:
            'Could not refresh your league. Saved readings are read-only. ' +
            'Check your connection, seat and authenticator, then refresh.',
        });
    } finally {
      if (turn === generation) store.setState({ pending: false });
    }
  }
  const canChange = () => {
    const s = store.getState();
    return !s.pending && !s.stale && Boolean(s.reading?.writable && s.reading.football);
  };
  async function review(request: NativeRequest, summary: string) {
    if (!canChange()) return;
    if (request.mode !== 'preview' && request.command !== 'draft.pick') return;
    const turn = generation;
    store.setState({
      pending: true,
      error: '',
      message: '',
      review: undefined,
    });
    try {
      const preview =
        request.mode === 'preview'
          ? parsePreview((await port(request, store.getState().reading?.football)).data)
          : null;
      if (turn !== generation) return;
      const next = { request, preview, summary };
      store.setState({ review: next });
    } catch {
      if (turn !== generation) return;
      store.setState({
        stale: true,
        error:
          'Could not check this change. Nothing is queued. Refresh your league before trying again.',
      });
    } finally {
      if (turn === generation) store.setState({ pending: false });
    }
  }
  async function market(command: string) {
    if (!canChange()) return;
    const turn = generation;
    store.setState({ pending: true, review: undefined, market: undefined });
    try {
      const response = await port({
        command: 'commish.market.read',
        mode: 'read',
        arguments: { league, command },
      });
      const market = parseHostedMarket(response.data);
      if (turn !== generation) return;
      if (
        market.league !== league ||
        market.revision !== store.getState().reading?.roster.revision ||
        response.football !== store.getState().reading?.football ||
        !response.writable
      )
        throw new Error('League changed');
      store.setState({ market });
    } catch {
      if (turn === generation)
        store.setState({
          stale: true,
          error: 'Could not read commissioner choices. Refresh before choosing another action.',
        });
    } finally {
      if (turn === generation) store.setState({ pending: false });
    }
  }
  async function confirm() {
    const s = store.getState();
    if (!canChange() || !s.review || s.review.preview?.blockers.length) return;
    const review = s.review;
    const turn = generation;
    store.setState({ pending: true, review: undefined, error: '' });
    try {
      const request = { ...review.request, mode: 'execute' } as NativeRequest;
      const receipt = parseReceipt((await port(request, s.reading?.football)).data);
      if (turn !== generation) return;
      if (receipt.league !== league) throw new Error('Wrong league receipt');
      store.setState({
        message: 'The server recorded your change. Refreshing the league.',
      });
    } catch {
      if (turn !== generation) return;
      store.setState({
        stale: true,
        error:
          'The change was not confirmed. It may have reached the server. ' +
          'Refresh before trying again; nothing will be sent automatically.',
      });
      return;
    } finally {
      if (turn === generation) store.setState({ pending: false });
    }
    await refresh();
  }
  const cancel = () => store.setState({ review: undefined });
  const dispose = () => {
    generation++;
    store.setState({
      reading: undefined,
      stale: true,
      pending: false,
      review: undefined,
      market: undefined,
    });
  };
  return {
    refresh,
    review,
    confirm,
    market,
    cancel,
    dispose,
    canChange,
    peek: store.getState,
    use: () => useStore(store),
  };
}
export type HostedRoom = ReturnType<typeof createHostedRoom>;
