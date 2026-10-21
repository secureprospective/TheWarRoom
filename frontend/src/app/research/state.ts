import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import type { Research, Scope } from './contract';
import { researchAPI, type ResearchReader } from './api';
export type { ResearchReader } from './api';

export function createResearchState(reader: ResearchReader, mine: () => string) {
  const initial: Scope = { mine: '', search: '', team: '', position: '', group: 'QB',
    trend: 'topAdds', depth: 'team', injuries: 'all' };
  type State = {
    research?: Research; pending: boolean; error: string; status: string;
    scope: Scope; pages: Record<string, number>;
  };
  const store = createStore<State>(() => ({
    pending: false, error: '', status: '', scope: initial, pages: {},
  }));
  let revision = 0;
  async function load(refresh = false, news = false, league = false, force = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '', status: '' });
    let polling: ReturnType<typeof setInterval> | undefined;
    if (refresh && reader.status) {
      store.setState({ status: 'Waiting for the MFL refresh to finish' });
      polling = setInterval(() => {
        void reader.status!().then(status => {
          if (request === revision && store.getState().pending) store.setState({ status });
        }).catch(cause => {
          if (request === revision) store.setState({ error: String(cause) });
        });
      }, 250);
    }
    try {
      if (refresh) await reader.refresh(force);
      if (news) await reader.news(league ? '' : mine());
      const research = await reader.read({ ...store.getState().scope, mine: mine() });
      if (request === revision) store.setState({ research });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (polling) clearInterval(polling);
      if (request === revision) store.setState({ pending: false });
    }
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
