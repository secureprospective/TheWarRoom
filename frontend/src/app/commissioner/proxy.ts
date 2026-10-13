import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { array, integer, object, text, timestamp } from '../data/parseValues';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type Receipt = {
  entry: string; league: string; revision: number; recordedAt: string; message: string;
};
export type LedgerEntry = {
  id: string; who: string; what: string; why: string; when: string; gravity: string;
};
export type Ledger = { league: string; name: string; revision: number; entries: LedgerEntry[] };
export type NativeLeague = { id: string; name: string };
export type Port = {
  execute(body: string): Promise<unknown>;
  ledger(league: string): Promise<unknown>;
  leagues(): Promise<unknown>;
};

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('The league ledger requires the desktop app.');
  return (window as unknown as { go: { main: { App: typeof Bindings } } }).go.main.App;
}
const live: Port = {
  execute: body => bindings().TargetCommissionerExecute(body),
  ledger: league => bindings().TargetCommissionerLedger(league),
  leagues: () => bindings().TargetCommissionerLeagues(),
};
export function parseReceipt(v: unknown): Receipt {
  const r = object(v, 'League receipt', ['entry', 'league', 'revision', 'recordedAt', 'message']);
  return {
    entry: text(r.entry, 'Entry'), league: text(r.league, 'League'),
    revision: integer(r.revision, 'League version', 1), recordedAt: timestamp(r.recordedAt, 'Recorded time'),
    message: text(r.message, 'Receipt'),
  };
}
export function parseLedger(v: unknown): Ledger {
  const r = object(v, 'League ledger', ['league', 'name', 'revision', 'entries']);
  return {
    league: text(r.league, 'League'), name: text(r.name, 'League name'),
    revision: integer(r.revision, 'League version'),
    entries: array(r.entries, 'Entries', (v, p) => {
      const e = object(v, p, ['id', 'who', 'what', 'why', 'when', 'gravity']);
      return {
        id: text(e.id, p), who: text(e.who, p), what: text(e.what, p), why: text(e.why, p),
        when: timestamp(e.when, p), gravity: text(e.gravity, p),
      };
    }),
  };
}
export function createProxy(port: Port) {
  return {
    execute: async (body: string) => parseReceipt(await port.execute(body)),
    ledger: async (league: string) => parseLedger(await port.ledger(league)),
    leagues: async () => array(await port.leagues(), 'App leagues', (v, p): NativeLeague => {
      const r = object(v, p, ['id', 'name']);
      return { id: text(r.id, p), name: text(r.name, p) };
    }),
  };
}
export const { execute, ledger, leagues } = createProxy(live);

type LedgerState = {
  choices: NativeLeague[]; selected: string; reading?: Ledger; error: string; pending: boolean;
};
export function createLedgerState(proxy = createProxy(live)) {
  const store = createStore<LedgerState>(() => ({ choices: [], selected: '', error: '', pending: true }));
  let generation = 0;
  async function choose(league = '') {
    const request = ++generation;
    store.setState({ pending: true, error: '', reading: undefined });
    try {
      const choices = await proxy.leagues();
      const selected = league || store.getState().selected || choices[0]?.id || '';
      if (selected && !choices.some(choice => choice.id === selected))
        throw new Error('This app league is no longer available. Choose another league.');
      const reading = selected ? await proxy.ledger(selected) : undefined;
      if (request === generation) store.setState({ choices, selected, reading });
    } catch (cause) {
      if (request === generation) store.setState({ error: String(cause) });
    } finally {
      if (request === generation) store.setState({ pending: false });
    }
  }
  return { choose, use: () => useStore(store), peek: store.getState };
}
export const ledgerState = createLedgerState();
