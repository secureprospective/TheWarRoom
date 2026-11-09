import { useEffect, useRef, useState } from 'react';
import { HQAct as WarRoomAct } from './HQAct';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import { readWarRoomInspector, changeWarRoomNote } from './warroomInspectorReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { unavailable } from './hqModel';
export type WarRoomProspect = { id: string; name: string; position?: string };
export function WarRoomScouting({ context, prospect, close }: { context: WarRoomContext;
  prospect: WarRoomProspect; close: () => void }) {
  const root = useRef<HTMLDivElement>(null);
  const [refresh, setRefresh] = useState(0);
  const [note, setNote] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, prospect.id,
    context.week, refresh]), () => readWarRoomInspector(context, prospect.id));
  useEffect(() => {
    const previous = document.activeElement;
    root.current?.querySelector<HTMLButtonElement>('button:not([disabled])')?.focus();
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); close(); }
      if (e.key !== 'Tab') return;
      const controls = root.current?.querySelectorAll<HTMLElement>('button:not([disabled]), textarea:not(' +
        '[disabled]), input:not([disabled])');
      if (!controls?.length) return;
      const first = controls[0], last = controls[controls.length - 1];
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    };
    document.addEventListener('keydown', key);
    return () => { document.removeEventListener('keydown', key);
      if (previous && 'focus' in previous) (previous as HTMLElement).focus(); };
  }, [close]);
  const save = async () => {
    if (pending || !note.trim() || context.native) return;
    setPending(true); setError('');
    try { await changeWarRoomNote(context, prospect.id, note); setNote(''); setRefresh(n => n + 1); }
    catch { setError(unavailable); } finally { setPending(false); }
  };
  const notes = reading.reading?.notes;
  return <div className="scrim open"><div className="scard" ref={root} role="dialog" aria-modal="true"
    aria-label={`${prospect.name} scouting report`}><div className="sc-h"><span className="pg">
      {prospect.position ?? '—'}</span><h2>{prospect.name}</h2><span className="sub">{unavailable}</span>
      <span className="pv">PRIVATE · MY REPORT</span><WarRoomAct className="x" onAction={close}>ESC
      </WarRoomAct></div><div className="sc-d"><div><span className="lbl">MY VERDICT · ONE LINE</span>
        <input disabled placeholder={unavailable} /></div>
      {['MY GRADE', 'MY CONFIDENCE', 'MY RANK', 'CONSENSUS ADP'].map(label => <div key={label}>
        <span className="lbl">{label}</span><span className="rk">{unavailable}</span></div>)}</div>
    <div className="sc-b"><div>{['STRENGTHS', 'WEAKNESSES', 'FANTASY ROLE', 'COMPARISON'].map(label =>
      <div key={label}><span className="lbl">{label}</span><textarea disabled placeholder={unavailable}
        /></div>)}</div><div><div className="cap2"><span className="lbl">QUICK NOTE</span>
      <div className="ctx">{unavailable}</div><div className="traits"><WarRoomEmpty /></div>
      <form className="row2" onSubmit={e => { e.preventDefault(); void save(); }}>
        <input aria-label="Private scouting note" value={note} maxLength={8000} disabled={!!context.native ||
          !notes || pending} onChange={e => setNote(e.target.value)} placeholder="Optional: one line" />
        <WarRoomAct className="btn solid" submit disabled={!!context.native || !notes || pending ||
          !note.trim()}>SAVE NOTE</WarRoomAct></form></div><span className="lbl">OBSERVATIONS</span>
      <div className="obs">{notes?.notes.length ? notes.notes.map(n => <div className="ob" key={n.at}>
        <span className="m"><span>{unavailable}</span><time dateTime={n.at}>{new Date(n.at)
          .toLocaleDateString()}</time></span><span className="t">{n.text}</span></div>) :
          <WarRoomEmpty reason={reading.error ?? reading.reading?.errors.join(' · ')} />}</div>
        {error && <p role="alert">{error}</p>}</div></div>
    <div className="sc-f"><span className="st">{notes ? 'Notes private to you' : unavailable}</span>
      <WarRoomAct className="btn" disabled title={unavailable}>COMPARE…</WarRoomAct>
      <WarRoomAct className="btn" disabled title={unavailable}>ADD TO QUEUE</WarRoomAct>
      <WarRoomAct className="btn solid" onAction={close}>DONE</WarRoomAct></div></div></div>;
}
