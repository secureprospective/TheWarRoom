import { ledgerBindings as bindings } from './api';
import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import { parseControls, type Controls } from './controls';
import { parseJobs, type Jobs } from './jobs';
import { parseDigests, parseNotes, type Digests, type LeagueNotes } from './notes';
import { parsePeople, type People } from './people';
import { parseRules, type Rules } from './rules';
import { parsePreview } from './preview';

type Receipt = {
  entry: string; league: string; revision: number; recordedAt: string; message: string;
};
export type LedgerEntry = {
  id: string; who: string; what: string; why: string; when: string; gravity: string;
  // Where Go lets this principal reverse the entry, and who may read the reversal.
  undo: '' | 'league' | 'private'; replacedAt?: string;
};
export type Ledger = { league: string; name: string; revision: number; entries: LedgerEntry[] };
type NativeLeague = { id: string; name: string };
export type Port = {
  execute(body: string): Promise<unknown>;
  preview(body: string): Promise<unknown>;
  ledger(league: string): Promise<unknown>;
  leagues(): Promise<unknown>;
  rules(league: string): Promise<unknown>;
  people(league: string): Promise<unknown>;
  jobs(league: string): Promise<unknown>;
  controls(): Promise<unknown>;
  notices(league: string): Promise<unknown>;
  digests(league: string): Promise<unknown>;
};


const live: Port = {
  execute: body => bindings().TargetCommissionerExecute(body),
  preview: body => bindings().TargetCommissionerPreview(body),
  ledger: league => bindings().TargetCommissionerLedger(league),
  leagues: () => bindings().TargetCommissionerLeagues(),
  rules: league => bindings().TargetCommissionerRules(league),
  people: league => bindings().TargetCommissionerPeople(league),
  jobs: league => bindings().TargetCommissionerJobs(league),
  controls: async () => bindings().TargetCommissionerControls(),
  notices: league => bindings().TargetCommissionerNotices(league),
  digests: league => bindings().TargetCommissionerDigests(league),
};
// A confirmed rollover receipt also carries the plan it applied; the plan was already reviewed in its preview.
export function parseReceipt(v: unknown): Receipt {
  const r = object(v, 'League receipt', ['entry', 'league', 'revision', 'recordedAt', 'message', 'plan']);
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
    preview: async (body: string) => parsePreview(await port.preview(body)),
    ledger: async (league: string) => parseLedger(await port.ledger(league)),
    rules: async (league: string) => parseRules(await port.rules(league)),
    people: async (league: string) => parsePeople(await port.people(league)),
    jobs: async (league: string) => parseJobs(await port.jobs(league)),
    controls: async () => parseControls(await port.controls()),
    notices: async (league: string) => parseNotes(await port.notices(league)),
    digests: async (league: string) => parseDigests(await port.digests(league)),
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
    if (sent.command === 'commish.notice.publish') return 'notice';
    if (sent.command === 'commish.whatsnew.post') return 'note';
    if (sent.command?.startsWith('draft.')) return sent.command;
    return sent.command?.startsWith('commish.role.') ? 'role' : sent.command ?? 'other';
  } catch {
    return 'other';
  }
}
export const staleWords = 'The league changed while you were editing; review and confirm again.';
// Go words a stale league version as "reread before changing it".
const isStale = (message: string) => message.includes('reread before changing it');
export const refusalText = (cause: unknown) =>
  (cause instanceof Error ? cause.message : String(cause)).replace(/^Error: /, '');

type Outcome =
  | { status: 'idle' }
  | { status: 'sending'; form: string }
  | { status: 'done'; form: string; receipt: Receipt }
  | { status: 'refused'; form: string; message: string };
type LeagueState = {
  choices: NativeLeague[]; selected: string; ledger?: Ledger; rules?: Rules; people?: People;
  // The jobs and the control list are read apart from the rest: if either fails, the screens still work
  // and the landing says what it could not read.
  jobs?: Jobs; controls?: Controls; landingError: string;
  // The notices and digests are read apart too, so the rules and the ledger never wait on them.
  notes?: LeagueNotes; digests?: Digests; opsError: string;
  error: string; pending: boolean; outcome: Outcome;
};
// One state serves the picker, the three readings and every form, so a receipt reloads them together.
export function createLeagueState(proxy = createProxy(live)) {
  const store = createStore<LeagueState>(() => ({
    choices: [], selected: '', error: '', landingError: '', opsError: '', pending: true,
    outcome: { status: 'idle' },
  }));
  let generation = 0;
  // The control list does not depend on the league, so it is read once.
  async function readLanding(selected: string) {
    type Landing = Pick<LeagueState, 'jobs' | 'controls' | 'landingError'>;
    const landing: Landing = { jobs: undefined, landingError: '' };
    const failed: string[] = [];
    const [jobs, controls] = await Promise.allSettled([
      selected ? proxy.jobs(selected) : Promise.resolve(undefined),
      store.getState().controls ?? proxy.controls()]);
    if (jobs.status === 'fulfilled') landing.jobs = jobs.value;
    else failed.push(`the jobs (${refusalText(jobs.reason)})`);
    if (controls.status === 'fulfilled') landing.controls = controls.value;
    else failed.push(`the list of controls (${refusalText(controls.reason)})`);
    if (failed.length > 0) landing.landingError = `The league page could not read ${failed.join(' or ')}.`;
    return landing;
  }
  async function readOps(selected: string) {
    type Ops = Pick<LeagueState, 'notes' | 'digests' | 'opsError'>;
    const ops: Ops = { notes: undefined, digests: undefined, opsError: '' };
    if (!selected) return ops;
    const [notes, digests] = await Promise.allSettled([proxy.notices(selected), proxy.digests(selected)]);
    const failed: string[] = [];
    if (notes.status === 'fulfilled') ops.notes = notes.value;
    else failed.push(`the league notices (${refusalText(notes.reason)})`);
    if (digests.status === 'fulfilled') ops.digests = digests.value;
    else failed.push(`the digest of changes (${refusalText(digests.reason)})`);
    if (failed.length > 0) ops.opsError = `The league page could not read ${failed.join(' or ')}.`;
    return ops;
  }
  async function readCore(selected: string) {
    type Core = Pick<LeagueState, 'ledger' | 'rules' | 'people' | 'error'>;
    const core: Core = { ledger: undefined, rules: undefined, people: undefined, error: '' };
    if (!selected) return core;
    const [ledger, rules, people] = await Promise.allSettled([
      proxy.ledger(selected), proxy.rules(selected), proxy.people(selected)]);
    const failed: string[] = [];
    if (ledger.status === 'fulfilled') core.ledger = ledger.value;
    else failed.push(`the ledger (${refusalText(ledger.reason)})`);
    if (rules.status === 'fulfilled') core.rules = rules.value;
    else failed.push(`the league rules (${refusalText(rules.reason)})`);
    if (people.status === 'fulfilled') core.people = people.value;
    else failed.push(`the people (${refusalText(people.reason)})`);
    if (failed.length > 0) core.error = `This league could not be read: ${failed.join(', ')}.`;
    return core;
  }
  async function choose(league = '') {
    const request = ++generation;
    const was = store.getState().selected;
    store.setState({ pending: true, error: '', ...(league && league !== was ?
      { ledger: undefined, rules: undefined, people: undefined, jobs: undefined, notes: undefined,
        digests: undefined } : {}) });
    try {
      const choices = await proxy.leagues();
      const selected = league || was || choices[0]?.id || '';
      if (selected && !choices.some(choice => choice.id === selected))
        throw new Error('This app league is no longer available. Choose another league.');
      // The picker is kept even when one league cannot be read, so the commissioner can still choose another
      // league or fix this one; each reading that failed is named.
      if (request === generation) store.setState({ choices, selected });
      const [core, landing, ops] = await Promise.all([
        readCore(selected), readLanding(selected), readOps(selected)]);
      if (request === generation)
        store.setState({ choices, selected, ...core, ...landing, ...ops });
    } catch (cause) {
      if (request === generation)
        store.setState({ error: String(cause), ledger: undefined, rules: undefined, people: undefined,
          jobs: undefined, notes: undefined, digests: undefined });
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
      const existingForm = ['rule', 'role', 'create', 'notice', 'note'].includes(form) ||
        form.startsWith('undo:');
      const message = stale && existingForm ? staleWords : refusalText(cause);
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

export type LedgerFilter = { person: string; kind: string; from: string; before: string };
export async function readLedgerFiltered(league: string, filter: LedgerFilter): Promise<Ledger> {
  return parseLedger(await bindings().TargetCommissionerLedgerFiltered(league, JSON.stringify(filter)));
}

export const readPreview = createProxy(live).preview;

// A rollover dry run returns a plan, not a ledger receipt. It must never enter the receipt refresh path.
export const nativeRolloverDryRun = (body: string) => live.execute(body);
