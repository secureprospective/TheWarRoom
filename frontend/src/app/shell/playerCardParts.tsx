import { useEffect, useRef, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { NativeHQ } from '../commissioner/hqReading';
import type { PulsePlayerFormPoint } from '../data/nativePulse';
import { statusWords, type PlayerStatus } from '../phone/playerStatus';
import { unavailable } from './hqModel';
import { readHostedPlayerNotes, setHostedPlayerNote } from './playerPersonal';

export const fixedPoints = (n?: number) => (n === undefined ? '–' : n.toFixed(1));

export function rosterStatusLabel(status?: string, empty = unavailable): string {
  if (status === 'IR') return 'Injured reserve';
  if (status === 'TAXI_SQUAD') return 'Taxi squad';
  if (status === 'ROSTER') return 'Active roster';
  return empty;
}

export function injuryLabel(id: string, injuries?: PlayerStatus, empty = unavailable): string {
  if (!injuries) return empty;
  const fact = injuries.facts.get(id);
  if (fact) return statusWords(fact);
  return injuries.state === 'unavailable' ? empty : 'No injury designation';
}

export function playerRange(history?: PulsePlayerFormPoint[]): [number, number] | undefined {
  if (!history?.length) return undefined;
  const pts = history.map(w => w.points);
  return [Math.min(...pts), Math.max(...pts)];
}

export function playerWeekPoints(reading: NativeHQ, mine: string, id: string): number | undefined {
  const own = reading.thisWeek?.scores.find(s => s.franchise === mine);
  const entry = own?.starters.find(s => s.player === id)
    ?? reading.thisWeek?.scores.flatMap(s => s.starters).find(s => s.player === id);
  return entry?.points ?? entry?.provisional;
}

export function PlayerFormBars({ weeks, empty, className = 'team-card__w',
  emptyClassName = 'team-tab__note' }: {
  weeks: PulsePlayerFormPoint[]; empty: string; className?: string; emptyClassName?: string;
}) {
  if (!weeks.length) return <p className={emptyClassName}>{empty}</p>;
  const max = Math.max(1, ...weeks.map(w => Math.abs(w.points)));
  return <div className={className}>{weeks.map(w => <div key={w.week}>{w.points.toFixed(1)}
    <i style={{ height: `${Math.max(2, (Math.max(0, w.points) / max) * 40)}px` }} />WK{w.week}</div>)}</div>;
}

export function PlayerStatusRows({ id, rosterStatus, injuries, empty = unavailable,
  rowClassName = 'team-card__s' }: {
  id: string; rosterStatus?: string; injuries?: PlayerStatus; empty?: string; rowClassName?: string;
}) {
  return <>
    <div className={rowClassName}><em>ROSTER</em><p><b>Status</b> {rosterStatusLabel(rosterStatus, empty)}
    </p></div>
    <div className={rowClassName}><em>NFL</em><p><b>Injury</b> {injuryLabel(id, injuries, empty)}</p></div>
  </>;
}

export function PlayerNoteField({ league, id, port = hostedWorkflow, inputId = 'team-note',
  labelClassName = 'team-lbl', fieldClassName = 'team-fld' }: {
  league: string; id: string; port?: HostedPort; inputId?: string;
  labelClassName?: string; fieldClassName?: string;
}) {
  const [note, setNote] = useState<{ text: string; at?: string; saved: string }>();
  const latest = useRef<string>();
  useEffect(() => {
    let live = true;
    latest.current = undefined;
    setNote(undefined);
    readHostedPlayerNotes(league, id, port).then(r => {
      const last = r.notes.at(-1);
      if (live) setNote({ text: last?.text ?? '', at: last?.at, saved: '' });
    }).catch(() => { if (live) setNote({ text: '', saved: 'Notes are unavailable' }); });
    return () => { live = false; };
  }, [league, id, port]);
  const save = async () => {
    if (!note || note.text === latest.current) return;
    latest.current = note.text;
    try {
      // One note per player: add the new words, then remove the note they replace.
      if (note.text.trim()) await setHostedPlayerNote(league, id, note.text.trim(), port);
      if (note.at) await setHostedPlayerNote(league, id, '', port, note.at);
      const fresh = await readHostedPlayerNotes(league, id, port);
      setNote({ text: note.text, at: fresh.notes.at(-1)?.at, saved: 'Saved' });
    } catch { setNote({ ...note, saved: 'Not saved. Try again.' }); }
  };
  return <>
    <div className={labelClassName}><label htmlFor={inputId}>My note</label></div>
    <div className={fieldClassName}><textarea id={inputId} placeholder="Only you see this."
      disabled={!note} value={note?.text ?? ''}
      onChange={e => setNote({ ...(note ?? { saved: '' }), text: e.target.value, saved: '' })}
      onBlur={() => { void save(); }} />{note?.saved && <small role="status">{note.saved}</small>}</div>
  </>;
}
