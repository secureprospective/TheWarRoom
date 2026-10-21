import { subscribePersonal } from '../offers/api';
import { useEffect, useRef, useState } from 'react';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import type { Snapshot } from '../data/contract';
import { localDate } from '../clock/labels';
import { Glyph } from '../look/Glyph';
import { ReportView } from '../reports/ReportView';
import { personalState } from './state';
import type { Entry, ListKind } from './contract';
import './personal.css';

const labels: Record<ListKind, string> = {
  watch: 'Watch list', bait: 'My trade bait', note: 'Scratchpad', draft: 'My draft list',
};
const verbs = { watch: 'watch.add', bait: 'bait.set', note: 'note.add', draft: 'draft.list' } as const;

function newestNote(a: Entry, b: Entry) {
  return b.at.localeCompare(a.at) || b.order - a.order;
}
export function NoteEditor({ id, entity, showNotes = true }: {
  id: string; entity: 'player' | 'franchise'; showNotes?: boolean;
}) {
  const [text, setText] = useState('');
  const mine = commands.use().franchiseId ?? '';
  const s = personalState.useReading(mine);
  const seen = useRef(s.savedNote);
  const [kept, setKept] = useState('');
  useEffect(() => {
    const saved = s.savedNote;
    if (saved !== seen.current && saved?.id === id && saved.entity === entity) {
      setText('');
      setKept(saved.at);
    }
    seen.current = saved;
  }, [s.savedNote, id, entity]);
  const notes = s.value?.entries.filter(e => e.kind === 'note' && e.id === id && e.entity === entity)
    .slice().sort(newestNote) ?? [];
  return <div className="personal-note">
    <label>Plain-text note<textarea value={text} maxLength={8000} disabled={s.pending}
      onChange={e => setText(e.target.value)} /></label>
    <Act variant="button" verb="note.add" args={{ id, entity, action: 'add', text }}
      disabled={!mine || s.pending || !text.trim()}>
      Keep dated note</Act>
    {kept && <p role="status">Kept · <time dateTime={kept}>{localDate(kept)}</time></p>}
    {showNotes && <ol aria-label="Kept notes">{notes.map(note => <li key={`${note.at}:${note.order}`}>
      <time dateTime={note.at}>{localDate(note.at)}</time><p className="personal-text">{note.text}</p>
    </li>)}</ol>}
    {s.error && <p role="alert">{s.error}</p>}
  </div>;
}
export function PlayerLists({ id }: { id: string }) {
  const mine = commands.use().franchiseId ?? '';
  const s = personalState.useReading(mine);
  useEffect(() => { if (mine) void personalState.load(mine); }, [mine]);
  return <section className="personal-actions" aria-label="Private player lists">
    <h5>My private lists</h5>
    {(['watch', 'bait', 'draft'] as const).map(kind => {
      const listed = s.value?.entries.some(e => e.kind === kind && e.id === id);
      return <Act variant="button" key={kind} verb={verbs[kind]} disabled={!mine || s.pending}
        args={{ id, entity: 'player', action: listed ? 'remove' : 'add' }}>
        {listed && kind !== 'draft' && <Glyph name={kind} />}
        {listed ? 'Remove from' : 'Add to'} {labels[kind].toLowerCase()}</Act>;
    })}
    <NoteEditor key={id} id={id} entity="player" />
    {s.error && <p role="alert">{s.error}</p>}
  </section>;
}
function EntryRow({ entry, snapshot, pending, facts }: {
  entry: Entry; snapshot: Snapshot; pending: boolean;
  facts?: { status: string; market: string[] };
}) {
  const name = entry.entity === 'player' ? snapshot.players.value.find(p => p.id === entry.id)?.name
    : snapshot.franchises.value.find(f => f.id === entry.id)?.name;
  return <li className="personal-entry">
    <Act variant="text" verb="inspector.pin" args={{ entity: entry.entity === 'player'
      ? { kind: 'player', id: entry.id, franchiseId: '' } : { kind: 'franchise', id: entry.id } }}>
      {(entry.kind === 'watch' || entry.kind === 'bait') && <Glyph name={entry.kind} />}
      {name ?? 'Name unavailable'}</Act>
    {entry.kind === 'note' ? <>
      <time dateTime={entry.at}>{localDate(entry.at)}</time><p className="personal-text">{entry.text}</p>
    </> : <>
      {facts && <><p>{facts.status}</p><p>{facts.market.join(' · ')}</p></>}
      <div className="plan-actions">
        {entry.kind === 'draft' && (['up', 'down'] as const).map(action =>
          <Act variant="button" key={action} verb="draft.list" disabled={pending}
            args={{ id: entry.id, entity: 'player', action }}>Move {action}</Act>)}
        <Act variant="button" verb={verbs[entry.kind]} disabled={pending}
          args={{ id: entry.id, entity: entry.entity, action: 'remove' }}>Remove</Act>
      </div>
    </>}
  </li>;
}
export default function PersonalLists({ snapshot, draft = false }: { snapshot: Snapshot; draft?: boolean }) {
  const mine = commands.use().franchiseId ?? '';
  const s = personalState.useReading(mine);
  const [player, setPlayer] = useState('');
  const [query, setQuery] = useState('');
  useEffect(() => {
    if (!mine) return;
    void personalState.load(mine);
    return subscribePersonal(() => personalState.load(mine));
  }, [mine, snapshot]);
  if (!mine) return <p>Choose my franchise before keeping private lists.</p>;
  const kinds: ListKind[] = draft ? ['draft'] : ['watch', 'bait', 'note'];
  const choices = snapshot.players.value.filter(p => p.name?.toLowerCase().includes(query.toLowerCase()))
    .slice(0, 40);
  return <section className="personal-lists" data-endpoint="S-11"
    aria-label={draft ? 'My draft list' : 'Private lists'}>
    <p>Private to this franchise in the app. Open MFL to maintain its separate copy; nothing is submitted.</p>
    <label>Find a player<input value={query} onChange={e => setQuery(e.target.value)} /></label>
    <label>Player<select value={player} onChange={e => setPlayer(e.target.value)}>
      <option value="">Choose a player</option>
      {choices.map(p => <option key={p.id} value={p.id}>{p.name} · {p.position} · {p.team}</option>)}
    </select></label>
    {s.pending && <p role="status">Reading private lists…</p>}
    {s.error && <p role="alert">{s.error}</p>}
    {kinds.map(kind => <section key={kind} data-endpoint={kind === 'watch' ? 'M-117'
      : kind === 'bait' ? 'M-116' : kind === 'note' ? 'M-118' : 'M-103'}>
      <h4>{labels[kind]}</h4>
      <div className="plan-actions">
        {kind !== 'note' && <Act variant="button" verb={verbs[kind]} disabled={!player || s.pending}
          args={{ id: player, entity: 'player', action: 'add' }}>Add selected player</Act>}
        <Act variant="button"
          verb="personal.handoff" args={{ kind }}>Open {labels[kind].toLowerCase()} on MFL</Act>
      </div>
      {kind === 'note' && <>
        <p>Keep a dated note for the selected player, or for your franchise when no player is chosen.</p>
        <NoteEditor key={player || mine} id={player || mine}
          entity={player ? 'player' : 'franchise'} showNotes={false} />
      </>}
      {!s.value?.entries.some(e => e.kind === kind) && <p>No entries kept.</p>}
      <ol>{s.value?.entries.filter(e => e.kind === kind)
        .sort((a, b) => kind === 'note' ? newestNote(a, b) : 0).map(e =>
        <EntryRow key={`${e.kind}:${e.id}:${e.at}`} entry={e} snapshot={snapshot} pending={s.pending}
          facts={s.value?.players.find(p => p.id === e.id)} />)}</ol>
    </section>)}
    {!draft && s.value && <ReportView report={s.value.bait} howLabel="How this is read" />}
  </section>;
}
