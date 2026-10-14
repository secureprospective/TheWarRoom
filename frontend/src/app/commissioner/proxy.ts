import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import { parsePeople, type People } from './people';
import { parseRules, type Rules } from './rules';
import type * as Bindings from '../../../wailsjs/go/main/App';

export type Receipt = {
  entry: string; league: string; revision: number; recordedAt: string; message: string;
};
export type LedgerEntry = {
  id: string; who: string; what: string; why: string; when: string; gravity: string;
  // Where Go lets this principal reverse the entry, and who may read the reversal.
  undo: '' | 'league' | 'private'; replacedAt?: string;
};
export type Ledger = { league: string; name: string; revision: number; entries: LedgerEntry[] };
export type NativeLeague = { id: string; name: string };
export type Port = {
  execute(body: string): Promise<unknown>;
  ledger(league: string): Promise<unknown>;
  leagues(): Promise<unknown>;
  rules(league: string): Promise<unknown>;
  people(league: string): Promise<unknown>;
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
  rules: league => bindings().TargetCommissionerRules(league),
  people: league => bindings().TargetCommissionerPeople(league),
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
      const e = object(v, p, ['id', 'who', 'what', 'why', 'when', 'gravity', 'undo', 'replacedAt']);
      return {
        id: text(e.id, p), who: text(e.who, p), what: text(e.what, p), why: text(e.why, p),
        when: timestamp(e.when, p), gravity: text(e.gravity, p),
        replacedAt: e.replacedAt === undefined ? '' : timestamp(e.replacedAt, `${p}.replacedAt`),
        undo: choice(e.undo, `${p}.undo`, ['', 'league', 'private'] as const),
      };
    }),
  };
}
export function createProxy(port: Port) {
  return {
    execute: async (body: string) => parseReceipt(await port.execute(body)),
    ledger: async (league: string) => parseLedger(await port.ledger(league)),
    rules: async (league: string) => parseRules(await port.rules(league)),
    people: async (league: string) => parsePeople(await port.people(league)),
    leagues: async () => array(await port.leagues(), 'App leagues', (v, p): NativeLeague => {
      const r = object(v, p, ['id', 'name']);
      return { id: text(r.id, p), name: text(r.name, p) };
    }),
  };
}

// A form is told apart by its command, so the registry command needs no extra argument.
export function formOf(body: string): string {
  try {
    const sent = JSON.parse(body) as { command?: string; args?: { entry?: string } };
    if (sent.command === 'commish.league.create') return 'create';
    if (sent.command === 'commish.rules.set') return 'rule';
    if (sent.command === 'commish.correction.revert') return `undo:${sent.args?.entry ?? ''}`;
    return sent.command?.startsWith('commish.role.') ? 'role' : 'other';
  } catch {
    return 'other';
  }
}
export const staleWords = 'The league changed while you were editing; review and confirm again.';
// Go words a stale league version as "reread before changing it".
const isStale = (message: string) => message.includes('reread before changing it');
export const refusalText = (cause: unknown) =>
  (cause instanceof Error ? cause.message : String(cause)).replace(/^Error: /, '');

export type Outcome =
  | { status: 'idle' }
  | { status: 'sending'; form: string }
  | { status: 'done'; form: string; receipt: Receipt }
  | { status: 'refused'; form: string; message: string };
type LeagueState = {
  choices: NativeLeague[]; selected: string; ledger?: Ledger; rules?: Rules; people?: People;
  error: string; pending: boolean; outcome: Outcome;
};
// One state serves the picker, the three readings and every form, so a receipt reloads them together.
export function createLeagueState(proxy = createProxy(live)) {
  const store = createStore<LeagueState>(() => ({
    choices: [], selected: '', error: '', pending: true, outcome: { status: 'idle' },
  }));
  let generation = 0;
  async function choose(league = '') {
    const request = ++generation;
    const was = store.getState().selected;
    store.setState({ pending: true, error: '', ...(league && league !== was ?
      { ledger: undefined, rules: undefined, people: undefined } : {}) });
    try {
      const choices = await proxy.leagues();
      const selected = league || was || choices[0]?.id || '';
      if (selected && !choices.some(choice => choice.id === selected))
        throw new Error('This app league is no longer available. Choose another league.');
      const [ledger, rules, people] = selected ? await Promise.all([
        proxy.ledger(selected), proxy.rules(selected), proxy.people(selected)]) : [];
      if (request === generation) store.setState({ choices, selected, ledger, rules, people });
    } catch (cause) {
      if (request === generation)
        store.setState({ error: String(cause), ledger: undefined, rules: undefined, people: undefined });
    } finally {
      if (request === generation) store.setState({ pending: false });
    }
  }
  async function execute(body: string) {
    const form = formOf(body);
    store.setState({ outcome: { status: 'sending', form } });
    try {
      const receipt = await proxy.execute(body);
      store.setState({ outcome: { status: 'done', form, receipt } });
      await choose(receipt.league);
      return receipt;
    } catch (cause) {
      const stale = isStale(refusalText(cause));
      const message = stale ? staleWords : refusalText(cause);
      store.setState({ outcome: { status: 'refused', form, message } });
      if (stale) await choose();
      throw new Error(message);
    }
  }
  const forget = () => store.setState({ outcome: { status: 'idle' } });
  return { choose, execute, forget, use: () => useStore(store), peek: store.getState };
}
export const leagueState = createLeagueState();
export const execute = leagueState.execute;
