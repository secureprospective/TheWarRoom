import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import type { Research, Scope } from './contract';
import { researchAPI, type ResearchReader } from './api';
import { createReadQuery, type ReadStatus } from '../data/query';
export type { ResearchReader } from './api';

export function createResearchState(reader: ResearchReader, mine: () => string) {
  const initial: Scope = { mine: '', search: '', team: '', position: '', group: 'QB',
    trend: 'topAdds', depth: 'team', injuries: 'all' };
  type State = {
    research?: Research; error: string; status: string;
    scope: Scope; pages: Record<string, number>;
  };
  const empty = (): State => ({ error: '', status: '', scope: initial, pages: {} });
  const store = createStore<State & ReadStatus>(() => ({ ...empty(), pending: false }));
  const query = createReadQuery('research.TargetResearch', {
    initial: empty, store, read: scope => reader.read(scope),
    accept: research => ({ research }),
    failure: () => 'Research could not be read. Please try refreshing again.',
  });
  let polls = 0;
  async function load(refresh = false, news = false, league = false, force = false) {
    const request = ++polls;
    let refreshFailed = false;
    let polling: ReturnType<typeof setInterval> | undefined;
    store.setState({ status: '' });
    // Only refresh/news are steps before the read; a plain scope change reads immediately.
    const before = refresh || news ? async () => {
      if (refresh && reader.status) {
        store.setState({ status: 'Waiting for the MFL refresh to finish' });
        polling = setInterval(() => {
          void reader.status!().then(status => {
            if (request === polls && store.getState().pending) store.setState({ status });
          }).catch(() => {
            if (request === polls) store.setState({ status: 'Waiting for the refresh to finish' });
          });
        }, 250);
      }
      if (refresh) {
        try { await reader.refresh(force); } catch { refreshFailed = true; }
      }
      if (news) await reader.news(league ? '' : mine());
    } : undefined;
    const reading = query.load(() => [{ ...store.getState().scope, mine: mine() }], { before });
    try {
      await reading;
      if (request === polls && refresh && !store.getState().error) {
        const research = store.getState().research;
        const missing = [...new Set(research?.board.reports
          .filter(report => ['failed', 'stale'].includes(report.status))
          .map(report => report.title) ?? [])];
        const status = missing.length
          ? `Refresh finished. Missing or out of date: ${missing.join(', ')}. Try refreshing again.`
          : refreshFailed ? 'Refresh finished, but research sources could not all be updated. Try refreshing again.'
            : 'Refresh complete.';
        store.setState({ status });
      }
    } finally { if (polling) clearInterval(polling); }
  }
  async function choose(action: string, value: string) {
    if (action === 'more') {
      const pages = store.getState().pages;
      store.setState({ pages: { ...pages, [value]: (pages[value] ?? 25) + 25 } });
      return;
    }
    if (action === 'refresh') return load(true, false, false, true);
    if (action === 'news') return load(false, true, value === 'league');
    if (action === 'open') {
      let article = store.getState().research?.articles.find(a => `${a.id}:${a.article}` === value);
      const [id, requested, extra] = value.split(':');
      if (!article && reader.player && /^\d{4,5}$/.test(id) && requested && !extra) {
        const player = await reader.player(id);
        article = player.articles.find(a => `${a.id}:${a.article}` === value);
      }
      if (!article) throw new Error('News: article is not held');
      return reader.open(article.id, article.article);
    }
    if (!['search', 'position', 'team', 'group', 'trend', 'depth', 'injuries'].includes(action))
      throw new Error('Research: invalid choice');
    store.setState({ scope: { ...store.getState().scope, [action]: value } });
    return load();
  }
  return { use: () => useStore(store), peek: store.getState, load, choose };
}
export const researchState = createResearchState(researchAPI, () => commands.read().franchiseId ?? '');
