import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { array, object, text } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

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
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Control Room requires the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
const live: ControlReader = {
  settings: async () => report(await bindings().TargetLeagueSettings(), 'League settings'),
  bylaws: async () => parseBylaws(await bindings().TargetBylaws()),
  accounting: async () => report(await bindings().TargetAccounting(), 'Accounting'),
  news: async () => report(await bindings().TargetSiteNews(), "What's New"),
  openAccounting: async () => bindings().TargetOpenMFLAccounting(),
};
export type ControlAction = 'league' | 'news' | 'search' | 'open-accounting';
type State = { settings?: Report; bylaws?: Bylaw[]; accounting?: Report; news?: Report; search: string;
  pending: boolean; error: string };
export function createControlState(reader: ControlReader) {
  const store = createStore<State>(() => ({ search: '', pending: false, error: '' }));
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
    if (action === 'news') { await run(async () => ({ news: await reader.news() })); return; }
    await run(async () => {
      const [settings, bylaws, accounting] =
        await Promise.all([reader.settings(), reader.bylaws(), reader.accounting()]);
      return { settings, bylaws, accounting };
    });
  }
  return { use: () => useStore(store), peek: store.getState, choose };
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
