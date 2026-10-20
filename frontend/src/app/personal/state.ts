import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { parsePersonal, type ListKind, type ListChange, type PersonalReading } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Private lists need the desktop app; no sample data is substituted.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
type PersonalProvider = {
  read(mine: string): Promise<PersonalReading>;
  change(mine: string, kind: ListKind, change: ListChange): Promise<void>;
  handoff(kind: ListKind | 'ir'): Promise<void>;
};
const live: PersonalProvider = {
  read: async mine => parsePersonal(await bindings().TargetPersonal(mine)),
  change: async (mine, kind, change) => bindings().TargetPersonalChange({
    mine, kind, ...change, text: change.text ?? '',
  }),
  handoff: async kind => bindings().TargetPersonalHandOff(kind),
};
export function createPersonalState(provider: PersonalProvider) {
  type Reading = { value?: PersonalReading; pending: boolean; error: string;
    savedNote?: ListChange & { at: string } };
  const store = createStore<Record<string, Reading>>(() => ({}));
  const empty: Reading = { pending: false, error: '' };
  const revisions = new Map<string, number>();
  const write = (mine: string, patch: Partial<Reading>) => store.setState(s =>
    ({ [mine]: { ...empty, ...s[mine], ...patch } }));
  async function load(mine: string) {
    const revision = (revisions.get(mine) ?? 0) + 1;
    revisions.set(mine, revision);
    write(mine, { pending: true, error: '' });
    try {
      const value = await provider.read(mine);
      if (revisions.get(mine) === revision) write(mine, { value, pending: false });
    } catch (cause) {
      if (revisions.get(mine) === revision) write(mine, { pending: false, error: String(cause) });
    }
  }
  async function change(mine: string, kind: ListKind, change: ListChange) {
    if (store.getState()[mine]?.pending) return;
    revisions.set(mine, (revisions.get(mine) ?? 0) + 1);
    write(mine, { pending: true, error: '' });
    try {
      await provider.change(mine, kind, change);
      if (kind === 'note' && change.action === 'add')
        write(mine, { savedNote: { ...change, at: new Date().toISOString() } });
      await load(mine);
    } catch (cause) { write(mine, { pending: false, error: String(cause) }); }
  }
  return { load, change, ensure: (mine: string) => {
    const s = store.getState()[mine];
    if (!s?.value && !s?.pending) return load(mine);
  }, handoff: provider.handoff,
    peek: (mine: string) => store.getState()[mine] ?? empty,
    useReading: (mine: string) => useStore(store, s => s[mine] ?? empty) };
}
export const personalState = createPersonalState(live);
