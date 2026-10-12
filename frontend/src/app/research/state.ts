import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { commands } from '../commands/registry';
import { parseResearch, type Research, type Scope } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type ResearchReader = {
  read(scope: Scope): Promise<Research>; refresh(force: boolean): Promise<void>;
  news(mine: string): Promise<void>; open(id: string, article: string): Promise<void>;
};
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Player research requires the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
const live: ResearchReader = {
  read: async scope => parseResearch(await bindings().TargetResearch(scope)),
  refresh: async force => bindings().TargetResearchRefresh(force),
  news: async mine => bindings().TargetResearchNewsRefresh(mine),
  open: async (id, article) => bindings().TargetOpenResearchArticle(id, article),
};
export function createResearchState(reader: ResearchReader, mine: () => string) {
  const initial: Scope = { mine: '', search: '', team: '', position: '', group: 'QB',
    trend: 'topAdds', depth: 'team', injuries: 'all' };
  type State = { research?: Research; pending: boolean; error: string; scope: Scope };
  const store = createStore<State>(() => ({ pending: false, error: '', scope: initial }));
  let revision = 0;
  async function load(refresh = false, news = false, league = false, force = false) {
    const request = ++revision;
    store.setState({ pending: true, error: '' });
    try {
      if (refresh) await reader.refresh(force);
      if (news) await reader.news(league ? '' : mine());
      const research = await reader.read({ ...store.getState().scope, mine: mine() });
      if (request === revision) store.setState({ research });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
  async function choose(action: string, value: string) {
    if (action === 'refresh') return load(true, false, false, true);
    if (action === 'news') return load(false, true, value === 'league');
    if (action === 'open') {
      const article = store.getState().research?.articles.find(a => `${a.id}:${a.article}` === value);
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
export const researchState = createResearchState(live, () => commands.read().franchiseId ?? '');
