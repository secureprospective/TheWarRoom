import { create } from 'zustand';
import { array, integer, object, text } from '../data/parseValues';
import { parseSchema, type RuleSchema } from '../commissioner/rules';
import { blankDraft, toNatural, type Draft } from '../commissioner/ruleDraft';
import { leagueState, parseReceipt } from '../commissioner/proxy';
import { runMigration } from './api';

export type Waiting = { field: string; label: string; schema: RuleSchema };
export type Preparation = {
  league: string; name: string; season: number; franchises: number; players: number; picks: number;
  settings: number; notes: string[]; problems: string[]; ready: boolean; preset: string; waiting: Waiting[];
};
export function parsePreparation(value: unknown): Preparation {
  const r = object(value, 'MFL league', ['league', 'name', 'season', 'franchises', 'players', 'picks',
    'settings', 'notes', 'problems', 'ready', 'preset', 'waiting']);
  if (typeof r.ready !== 'boolean') throw new Error('The MFL league could not be read. Prepare again.');
  return {
    league: text(r.league, 'League'), name: text(r.name, 'Name'), season: integer(r.season, 'Season'),
    franchises: integer(r.franchises, 'Teams'), players: integer(r.players, 'Players'),
    picks: integer(r.picks, 'Picks'), settings: integer(r.settings, 'Settings'), ready: r.ready,
    preset: text(r.preset, 'Starting settings'), notes: array(r.notes, 'Notes', text),
    problems: array(r.problems, 'Problems', text),
    waiting: array(r.waiting ?? [], 'Settings waiting', (value, path) => {
      const f = object(value, path, ['field', 'label', 'schema']);
      return { field: text(f.field, path), label: text(f.label, path), schema: parseSchema(f.schema, path) };
    }),
  };
}
export type MigrationPort = {
  run(command: string, body: string): Promise<unknown>; open(league: string): Promise<void>;
};
type State = {
  id: string; key: string; preset: string; reason: string; prep?: Preparation;
  drafts: Record<string, Draft>; pending: boolean; error: string; done: string;
};
export function createMigration(port: MigrationPort) {
  const use = create<State>(() => ({ id: '', key: '', preset: '', reason: '', drafts: {},
    pending: false, error: '', done: '' }));
  const set = use.setState;
  function input(id: string) {
    if (!use.getState().pending) set({ id, key: '', reason: '', prep: undefined,
      drafts: {}, preset: '', done: '', error: '' });
  }
  async function prepare(preset = '') {
    const s = use.getState();
    if (s.pending) return;
    set({ pending: true, error: '', prep: undefined, done: '' });
    try {
      const prep = parsePreparation(await port.run('league.migrate.prepare', JSON.stringify({
        mflLeague: s.id.trim(), preset, ...(s.key.trim() ? { apiKey: s.key.trim() } : {}),
      })));
      set({ prep, id: s.id.trim(), key: '', preset: prep.preset,
        drafts: Object.fromEntries(prep.waiting.map(f => [f.field, blankDraft(f.schema)])) });
    } catch (cause) { set({ error: String(cause) }); }
    finally { set({ pending: false }); }
  }
  function answers() {
    const s = use.getState();
    return (s.prep?.waiting ?? []).map(f => ({ field: f.field,
      typed: toNatural(f.schema, s.drafts[f.field] ?? blankDraft(f.schema), f.label) }));
  }
  function blocked() {
    const s = use.getState();
    if (s.pending || s.done || !s.prep?.ready || s.prep.problems.length)
      return 'Prepare a league without problems first.';
    for (const answer of answers()) if (!answer.typed.ok) return answer.typed.error;
    return s.reason.trim() ? '' : 'Type a reason for importing this league.';
  }
  async function importLeague() {
    if (blocked()) return;
    const s = use.getState();
    const values = answers().map(a => ({ field: a.field, value: a.typed.ok ? a.typed.value : null }));
    set({ pending: true, error: '' });
    try {
      const receipts = array(await port.run('league.migrate.import', JSON.stringify({
        mflLeague: s.id, preset: s.preset, reason: s.reason.trim(), answers: values,
      })), 'Import receipts', parseReceipt);
      const league = receipts[0]?.league;
      if (!league) throw new Error('No league was recorded. Prepare again before retrying.');
      // Once recorded, do not offer a duplicate import if opening its readings fails.
      set({ done: league, key: '' });
      await port.open(league);
    } catch (cause) { set({ error: String(cause) }); }
    finally { set({ pending: false }); }
  }
  return { use, input, prepare, importLeague, blocked,
    key: (key: string) => { if (!use.getState().pending) set({ key, prep: undefined, drafts: {} }); },
    reason: (reason: string) => set({ reason }),
    answer: (field: string, draft: Draft) => set(s => ({ drafts: { ...s.drafts, [field]: draft } })),
  };
}
export const migration = createMigration({ run: runMigration, open: league => leagueState.choose(league) });
