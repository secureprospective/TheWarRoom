import { HQAct as WarRoomAct } from './HQAct';
import { useState, type ReactNode } from 'react';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import type { WarRoomAsset } from './warroomTradeReads';
import { readWarRoomInspector, changeWarRoomNote } from './warroomInspectorReads';
import { researchFact } from './hqPlayerData';
import { money, unavailable } from './hqModel';
import { CompactAbsent, WarRoomEmpty } from './WarRoomHeader';
function Drawer({ label, detail, children, initial = true }: { label: string; detail?: ReactNode;
  children: ReactNode; initial?: boolean }) {
  const [open, setOpen] = useState(initial);
  return <div className={`idr ${open ? 'open' : ''}`}><WarRoomAct expanded={open}
    onAction={() => setOpen(!open)}>{label}{label === 'NOTES' && <span className="mine">MINE</span>}
    <span className="c">{detail ?? unavailable}</span></WarRoomAct><div className="body" hidden={!open}>
      {children}</div></div>;
}
export function WarRoomInspector({ context, asset, add, watch }: { context: WarRoomContext;
  asset?: WarRoomAsset; add?: () => void; watch?: () => Promise<void> }) {
  return <LoadedInspector key={`${context.native?.league}:${context.mine}:${asset?.token}`}
    context={context} asset={asset} add={add} watch={watch} />;
}
function LoadedInspector({ context, asset, add, watch }: { context: WarRoomContext; asset?: WarRoomAsset;
  add?: () => void; watch?: () => Promise<void> }) {
  const [revision, setRevision] = useState(0);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, asset?.token,
    context.week, revision]), () => asset && !asset.pick && /^\d+$/.test(asset.token) ?
    readWarRoomInspector(context, asset.token) : Promise.resolve(undefined));
  const facts = reading.reading;
  const changeNote = async (text: string, at?: string) => {
    if (!asset || pending) return;
    setPending(true); setError('');
    try { await changeWarRoomNote(context, asset.token, text, at); setNote('');
      setRevision(n => n + 1); } catch { setError(unavailable); } finally { setPending(false); }
  };
  const research = facts?.research;
  const nativeContract = context.native ? context.snapshot.rosters.value.flatMap(r => r.players)
    .find(p => p.id === asset?.token) : undefined;
  const age = researchFact(research, 'Age');
  const arc = facts?.arc;
  const maxArc = Math.max(1, ...arc?.points.map(p => p.level) ?? []);
  const projection = facts?.projection;
  return <aside className="insp" aria-label="Inspector"><div className="ihead"><div className="top">
    <span className="pg">{research?.position || asset?.position || projection?.position || '—'}</span>
    <span className="on">{asset ? 'PLAYER' : unavailable}</span></div><div className="nm">
      {research?.name ?? asset?.name ?? unavailable}</div><div className="sub">{research?.owner ??
        unavailable} · age {age ?? unavailable} · {asset?.contract ?? unavailable}</div></div>
    <div className="lensrow"><div><b><CompactAbsent /></b><span>to you</span></div>
      <div><b><CompactAbsent /></b><span>to them</span></div></div><div className="why"
        title={reading.error || facts?.errors.length ? unavailable : undefined}>{unavailable}</div>
    <div className="idrawers"><Drawer label="CONTRACT" detail={nativeContract?.yearsRemaining === undefined ?
      undefined : `${nativeContract.yearsRemaining} yrs`}>
      {facts?.contract?.value?.years.length ? facts.contract.value.years.map(y => <div className="kv"
        key={y.season}><span>{y.season}</span><span title={y.reason ? unavailable : undefined}>
          {y.cost.text} · {y.tag}</span>
        </div>) :
        nativeContract ? <div className="kv"><span>{context.snapshot.league.value.season}</span>
          <span>{money(nativeContract.salary)} · {nativeContract.contractStatus} · {
            nativeContract.yearsRemaining ?? unavailable} yrs</span></div> : <WarRoomEmpty />}
      <div className="kv">
      <span>Dead if cut</span>
      <span>{context.native ? money(facts?.cut?.deadNow) :
        researchFact(research, 'Dead cap if cut today') ?? unavailable}</span>
      </div>
    </Drawer><Drawer label="VALUE ARC" detail={arc?.points.length ? `age ${arc.points[0].age} → ${
      arc.points[arc.points.length - 1].age}` : undefined}>
      {arc?.points.length ? <div className="arc" title={arc.reason ? unavailable : undefined}>
        {arc.points.map(p => <i
        key={p.season}
        className={p.season === context.snapshot.league.value.season ? 'now' : ''}
        style={{ height: `${Math.max(0, p.level / maxArc * 100)}%` }}
        title={`${p.season} · age ${p.age} · ${p.level}`} />)}</div> : <WarRoomEmpty />}</Drawer>
    <Drawer label="IN YOUR LINEUP"><div className="kv"><span>Replaces</span><span>{unavailable}</span>
      </div><div className="kv"><span>L5 avg</span><span>{researchFact(research, 'L5 avg') ?? unavailable}
      </span></div><div className="kv"><span>Fills need</span><span>{unavailable}</span></div>
      {projection && <div className="kv"><span>Floor · median · ceiling</span><span
        title={`${projection.games} games · ${projection.words}`}>{projection.available ?
          `${projection.floor.toFixed(1)} · ${projection.median.toFixed(1)} · ${
            projection.ceiling.toFixed(1)}` : unavailable}</span></div>}</Drawer>
    <Drawer label="NOTES" detail={facts?.notes?.count} initial={false}>
      {facts?.notes ? facts.notes.notes.map(n => <div key={n.at}><time dateTime={n.at}>{new Date(n.at)
        .toLocaleDateString()}</time><p>{n.text}</p><WarRoomAct className="btn" disabled={pending}
        onAction={() => { void changeNote(n.text, n.at); }}>REMOVE</WarRoomAct></div>) : <WarRoomEmpty />}
      <form onSubmit={e => { e.preventDefault(); if (note.trim()) void changeNote(note); }}>
        <textarea aria-label="Private player note" maxLength={8000} value={note}
          disabled={!!context.native || !facts?.notes || pending} onChange={e => setNote(e.target.value)}
          /><WarRoomAct className="btn" submit disabled={!!context.native || !facts?.notes || pending ||
            !note.trim()}>SAVE NOTE</WarRoomAct></form>{error && <p role="alert">{error}</p>}</Drawer></div>
    <div className="iact"><span className="h">MOVE</span><div className="ia"><WarRoomAct className="btn"
      disabled={!add} onAction={add}>ADD TO TRADE</WarRoomAct>
      <WarRoomAct className="btn" disabled={!watch || pending}
      onAction={() => { if (watch) { setPending(true); setError('');
        void watch().catch(() => setError(unavailable))
        .finally(() => setPending(false)); } }}>WATCH</WarRoomAct></div></div>
    <div className="ifoot">
      <WarRoomAct disabled title={unavailable}>
      + DRAWER</WarRoomAct></div>
  </aside>;
}
