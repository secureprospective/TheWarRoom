import { array, choice, integer, object, text, timestamp } from '../data/parseValues';
import type { Governed } from '../governed/governed';

// The Go notices reading (TargetCommissionerNotices) and digests reading (TargetCommissionerDigests).
// Go orders both: pinned notices lead, each group newest first.
export type Note = {
  id: string; kind: 'notice' | 'note'; text: string; pinned: boolean; who: string; when: string;
  withdrawn: boolean; change: string;
};
export type Release = { date: string; title: string; words: string };
export type LeagueNotes = {
  league: string; revision: number; notices: Note[]; notes: Note[]; releases: Release[];
  mail: { words: string; recorded: Note[] };
};
// A posted note is a value the commissioner set: the glyph beside it says who, when and which change.
export const governedBy = (note: Note): Governed =>
  ({ entry: note.id, who: note.who, when: note.when, change: note.change });
export type Digest = { id: string; sentence: string; when: string };
export type Digests = { league: string; digests: Digest[] };

function flag(v: unknown, path: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${path}: expected true or false`);
  return v;
}
function parseNote(v: unknown, p: string): Note {
  const o = object(v, p, ['id', 'kind', 'text', 'pinned', 'who', 'when', 'withdrawn', 'change']);
  return {
    id: text(o.id, p), kind: choice(o.kind, `${p}.kind`, ['notice', 'note'] as const), text: text(o.text, p),
    pinned: flag(o.pinned, `${p}.pinned`), who: text(o.who, p), when: timestamp(o.when, `${p}.when`),
    withdrawn: flag(o.withdrawn, `${p}.withdrawn`), change: text(o.change, p),
  };
}
export function parseNotes(v: unknown): LeagueNotes {
  const r = object(v, 'League notices', ['league', 'revision', 'notices', 'notes', 'releases', 'mail']);
  const mail = object(r.mail, 'Mail log', ['words', 'recorded']);
  return {
    league: text(r.league, 'League'), revision: integer(r.revision, 'League version'),
    notices: array(r.notices, 'Notices', parseNote), notes: array(r.notes, 'League notes', parseNote),
    releases: array(r.releases, 'Release notes', (x, p) => {
      const o = object(x, p, ['date', 'title', 'words']);
      return { date: text(o.date, p), title: text(o.title, p), words: text(o.words, p) };
    }),
    mail: { words: text(mail.words, 'Mail log'),
      recorded: array(mail.recorded, 'Recorded notices', parseNote) },
  };
}
export function parseDigests(v: unknown): Digests {
  const r = object(v, 'League digests', ['league', 'digests']);
  return {
    league: text(r.league, 'League'),
    digests: array(r.digests, 'Digests', (d, p) => {
      const o = object(d, p, ['id', 'sentence', 'when']);
      return { id: text(o.id, p), sentence: text(o.sentence, p), when: timestamp(o.when, `${p}.when`) };
    }),
  };
}
