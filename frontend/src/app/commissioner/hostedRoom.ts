import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { hostedWorkflow, type HostedResponse } from './api';
import type { NativeRequest } from '../commands/native';
import { parseDraft, type Draft } from '../draftroom/draft';
import { parseRules, type Rules } from './rules';
import { parsePreview, type Preview } from './preview';
import { parseReceipt } from './proxy';
import { parseHostedLineup, parseHostedReserves, parseHostedRoster,
  type HostedLineup, type HostedReserves, type HostedRoster } from './hostedReadings';

export type HostedReading = { lineup: HostedLineup; roster: HostedRoster; reserves: HostedReserves;
  draft: Draft; rules: Rules; label: string; writable: boolean; football: string; readAt: number };
type Review = { request: NativeRequest; preview: Preview | null; summary: string };
type State = { reading?: HostedReading; stale: boolean; pending: boolean; error: string;
  message: string; review?: Review };
export type HostedPort = (request: NativeRequest, football?: string) => Promise<HostedResponse>;

// One signed-in franchise, held only in memory. No local store, queue or authoritative fallback.
export function createHostedRoom(league: string, franchise: string, port: HostedPort = hostedWorkflow) {
  const store = createStore<State>(() => ({ stale: true, pending: false, error: '', message: '' }));
  let generation = 0;
  let activeWeek = 1;
  let held: Review | undefined;
  async function read(week = activeWeek) {
    if (store.getState().pending) return;
    activeWeek = week;
    const turn = ++generation;
    store.setState({ pending: true, review: undefined });
    try {
      const requests: NativeRequest[] = [
        { command: 'lineup.read', mode: 'read', arguments: { league, franchise, week, at: '' } },
        { command: 'native.assets.read', mode: 'read', arguments: { league } },
        { command: 'native.reserve.check', mode: 'read', arguments: { league } },
        { command: 'draft.read', mode: 'read', arguments: { league, franchise, season: 0, at: '0001-01-01T00:00:00Z' } },
        { command: 'commish.rules.read', mode: 'read', arguments: { league } },
      ];
      const results = await Promise.all(requests.map(request => port(request)));
      const [lineup, roster, reserves, draft, rules] = results;
      const reading: HostedReading = { lineup: parseHostedLineup(lineup.data), roster: parseHostedRoster(roster.data),
        reserves: parseHostedReserves(reserves.data), draft: parseDraft(draft.data), rules: parseRules(rules.data),
        label: lineup.label, writable: results.every(r => r.writable), football: lineup.football, readAt: Date.now() };
      if (reading.lineup.league !== league || reading.lineup.franchise !== franchise
        || reading.draft.league !== league || reading.rules.league !== league
        || results.some(r => r.football !== reading.football)) throw new Error('League readings changed. Refresh again.');
      if (turn === generation) { held = undefined; store.setState({ reading, stale: false, error: '' }); }
    } catch {
      if (turn === generation) store.setState({ stale: true,
        error: 'Could not refresh your league. Saved readings are read-only. Check your connection, seat and authenticator, then refresh.' });
    } finally { if (turn === generation) store.setState({ pending: false }); }
  }
  const canChange = () => {
    const s = store.getState();
    return !s.pending && !s.stale && Boolean(s.reading?.writable && s.reading.football);
  };
  async function review(request: NativeRequest, summary: string) {
    if (!canChange()) return;
    if (request.mode !== 'preview' && request.command !== 'draft.pick') return;
    store.setState({ pending: true, error: '', message: '', review: undefined });
    try {
      const preview = request.mode === 'preview'
        ? parsePreview((await port(request, store.getState().reading?.football)).data) : null;
      // Retain the exact request key/revision across an ambiguous delivery, never silently retry.
      const signature = JSON.stringify({ ...request, arguments: { ...request.arguments, request: '' } });
      const prior = held && JSON.stringify({ ...held.request, arguments: { ...held.request.arguments, request: '' } });
      const next = { request: signature === prior ? held!.request : request, preview, summary };
      held = next;
      store.setState({ review: next });
    } catch {
      store.setState({ stale: true, error: 'Could not check this change. Nothing is queued. Refresh your league before trying again.' });
    } finally { store.setState({ pending: false }); }
  }
  async function confirm() {
    const s = store.getState();
    if (!canChange() || !s.review || s.review.preview?.blockers.length) return;
    const review = s.review;
    store.setState({ pending: true, review: undefined, error: '' });
    try {
      const request = { ...review.request, mode: 'execute' } as NativeRequest;
      const receipt = parseReceipt((await port(request, s.reading?.football)).data);
      store.setState({ message: 'The server recorded your change. Refreshing the league.' });
      if (receipt.league !== league) throw new Error('Wrong league receipt');
    } catch {
      store.setState({ stale: true, error: 'The change was not confirmed. It may have reached the server. Refresh before trying again; nothing will be sent automatically.' });
      return;
    } finally { store.setState({ pending: false }); }
    await read();
  }
  const cancel = () => store.setState({ review: undefined });
  const dispose = () => { generation++; held = undefined; store.setState({ reading: undefined, stale: true, pending: false, review: undefined }); };
  return { read, review, confirm, cancel, dispose, canChange, peek: store.getState, use: () => useStore(store) };
}
export type HostedRoom = ReturnType<typeof createHostedRoom>;
