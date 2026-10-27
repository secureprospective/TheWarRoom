import { createReadQuery } from '../data/query';
import { array, object, text } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import { controlAPI } from './api';

export type Bylaw = { title: string; lines: string[] };
export type ControlReader = {
  settings(): Promise<Report>;
  bylaws(): Promise<Bylaw[]>;
  accounting(): Promise<Report>;
  news(): Promise<Report>;
  openAccounting(): Promise<void>;
};
export function parseBylaws(v: unknown): Bylaw[] {
  return array(v, 'By-laws', (s, p) => {
    const r = object(s, p, ['title', 'lines']);
    return { title: text(r.title, p), lines: array(r.lines, p, (l, q) => text(l, q)) };
  });
}
const live: ControlReader = {
  settings: async () => report(await controlAPI.settings(), 'League settings'),
  bylaws: async () => parseBylaws(await controlAPI.bylaws()),
  accounting: async () => report(await controlAPI.accounting(), 'Accounting'),
  news: async () => report(await controlAPI.news(), "What's New"),
  openAccounting: async () => controlAPI.openAccounting(),
};
type ControlAction = 'league' | 'news' | 'search' | 'open-accounting';
type State = { settings?: Report; bylaws?: Bylaw[]; accounting?: Report; news?: Report; search: string;
  pending: boolean; error: string };
export function createControlState(reader: ControlReader) {
  const initial = (): State => ({ search: '', pending: false, error: '' });
  const league = createReadQuery('control.TargetLeagueSettings', {
    initial, companions: ['control.TargetBylaws', 'control.TargetAccounting'],
    read: async () => {
      const [settings, bylaws, accounting] =
        await Promise.all([reader.settings(), reader.bylaws(), reader.accounting()]);
      return { settings, bylaws, accounting };
    }, accept: value => value,
  });
  const news = createReadQuery('control.TargetSiteNews', {
    initial, store: league.store, read: () => reader.news(), accept: value => ({ news: value }),
  });
  const store = league.store;
  async function run(work: () => Promise<Partial<State>>) {
    store.setState({ pending: true, error: '' });
    try {
      store.setState(await work());
    } catch (cause) {
      store.setState({ error: String(cause) });
    } finally {
      store.setState({ pending: false });
    }
  }
  async function choose(action: ControlAction, value = '') {
    if (action === 'search') { store.setState({ search: value }); return; }
    if (action === 'open-accounting') {
      await run(async () => { await reader.openAccounting(); return {}; });
      return;
    }
    if (action === 'news') { await news.load([]); return; }
    await league.load([]);
  }
  return { use: league.use, peek: store.getState, choose };
}
export const controlState = createControlState(live);

// MFL's words for what the rulebook names differently: the app's settings say "Taxi squad".
const rulebookWords: Record<string, string> = { taxi: 'practice squad' };

// matching keeps the by-laws sections whose title or lines contain every word of the search,
// reading MFL's word for a thing as the rulebook's.
export function matching(sections: Bylaw[], search: string): Bylaw[] {
  const words = search.toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) return sections;
  return sections.filter(s => {
    const body = (s.title + '\n' + s.lines.join('\n')).toLowerCase();
    return words.every(w => body.includes(w) || (w in rulebookWords && body.includes(rulebookWords[w])));
  });
}
