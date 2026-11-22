import { createReadQuery } from '../data/query';
import { parsePersonal, type ListKind, type ListChange, type PersonalReading } from './contract';
import { personalWire } from '../offers/api';

type PersonalProvider = {
  read(mine: string): Promise<PersonalReading>;
  change(mine: string, kind: ListKind, change: ListChange): Promise<void>;
  handoff(kind: ListKind | 'ir'): Promise<void>;
};
const live: PersonalProvider = {
  read: async mine => parsePersonal(await personalWire.read(mine)),
  change: async (mine, kind, change) => { await personalWire.change(mine, kind, change); },
  handoff: async kind => { await personalWire.handoff(kind); },
};
export function createPersonalState(provider: PersonalProvider) {
  type Reading = { value?: PersonalReading; pending: boolean; error: string;
    savedNote?: ListChange & { at: string } };
  const empty: Reading = { pending: false, error: '' };
  function createReading() {
    return createReadQuery('offers.TargetPersonal', {
      initial: (): Reading => ({ ...empty }),
      read: (mine: string) => provider.read(mine), accept: value => ({ value }),
    });
  }
  const readings = new Map<string, ReturnType<typeof createReading>>();
  function reading(mine: string) {
    let query = readings.get(mine);
    if (!query) { query = createReading(); readings.set(mine, query); }
    return query;
  }
  const write = (mine: string, patch: Partial<Reading>) => reading(mine).set(patch);
  const load = (mine: string) => reading(mine).load([mine]);
  async function change(mine: string, kind: ListKind, change: ListChange) {
    if (reading(mine).peek().pending) return;
    reading(mine).fence();
    write(mine, { pending: true, error: '' });
    try {
      await provider.change(mine, kind, change);
      if (kind === 'note' && change.action === 'add')
        write(mine, { savedNote: { ...change, at: new Date().toISOString() } });
      await load(mine);
    } catch (cause) { write(mine, { pending: false, error: String(cause) }); }
  }
  return { load, change, ensure: (mine: string) => {
    const s = readings.get(mine)?.peek();
    if (!s?.value && !s?.pending) return load(mine);
  }, handoff: provider.handoff,
    peek: (mine: string) => readings.get(mine)?.peek() ?? empty,
    useReading: (mine: string) => reading(mine).use() };
}
export const personalState = createPersonalState(live);
