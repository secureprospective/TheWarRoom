import { HQAct as WarRoomAct } from './HQAct';
import { useState, type ReactNode } from 'react';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import type { WarRoomAsset } from './warroomTradeReads';
import { readWarRoomInspector, changeWarRoomNote,
  type WarRoomInspectorReading } from './warroomInspectorReads';
import { researchFact } from './hqPlayerData';
import { money, pickPlayer, unavailable } from './hqModel';
import { CompactAbsent, WarRoomEmpty } from './WarRoomHeader';
import { lensValue, type WarRoomLens } from './warroomLens';
// What the gap between the two lenses means for this asset, in the owner's own terms.
function lensWhy(lens: WarRoomLens | undefined, token: string, theirs: boolean): string {
  const a = lens?.assets.get(token);
  if (!lens || lens.preparing || !a) return unavailable;
  if (!a.valued) return a.words || 'No value in this league yet.';
  const n = lens.starterN;
  if (!lens.them) return a.you < 0.05 ? `Outside your best ${n} today: depth you can move at no cost.` :
    `Holds ${a.you.toFixed(1)} of your best ${n}. Choose a partner to see what it is worth to them.`;
  const gap = Math.abs(a.you - a.them).toFixed(1);
  if (a.you < 0.05 && a.them < 0.05) return `Outside both sides' best ${n} today: depth, not value.`;
  if (theirs) return a.you > a.them ? `The gap is your edge: worth ${gap} more to you than to them. ` +
    'Ask for this one.' : `Worth ${gap} more to them than to you. Expensive to pry loose.`;
  return a.them > a.you ? `Worth ${gap} more to them than to you. Your best bait.` :
    `Worth ${gap} more to you than to them. Keep it out of the deal.`;
}
function Drawer({ label, detail, children, initial = true }: { label: string; detail?: ReactNode;
  children: ReactNode; initial?: boolean }) {
  const [open, setOpen] = useState(initial);
  return <div className={`idr ${open ? 'open' : ''}`}><WarRoomAct expanded={open}
    onAction={() => setOpen(!open)}>{label}{label === 'NOTES' && <span className="mine">MINE</span>}
    <span className="c">{detail ?? unavailable}</span></WarRoomAct><div className="body" hidden={!open}>
      {children}</div></div>;
}
export function WarRoomInspector({ context, asset, add, watch, lens, theirs }: { context: WarRoomContext;
  asset?: WarRoomAsset; add?: () => void; watch?: () => Promise<void>; lens?: WarRoomLens;
  theirs?: boolean }) {
  if (!asset) return <aside className="insp" aria-label="Inspector"><p className="empty"
    role="status">{pickPlayer}</p></aside>;
  return <LoadedInspector key={`${context.native?.league}:${context.mine}:${asset.token}`}
    context={context} asset={asset} add={add} watch={watch} lens={lens} theirs={theirs} />;
}
function LoadedInspector({ context, asset, add, watch, lens, theirs }: { context: WarRoomContext;
  asset?: WarRoomAsset; add?: () => void; watch?: () => Promise<void>; lens?: WarRoomLens;
  theirs?: boolean }) {
  const [revision, setRevision] = useState(0);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const reading = useWarRoomRead<WarRoomInspectorReading | undefined>(JSON.stringify([
    context.native?.league, context.mine, asset?.token, context.week, revision]), publish => asset && !asset.pick && /^\d+$/.test(asset.token) ?
    readWarRoomInspector(context, asset.token, publish) : Promise.resolve(undefined));
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
  const age = researchFact(research, 'Age') ?? facts?.arc?.age.toFixed(1);
  const arc = facts?.arc;
  const maxArc = Math.max(1, ...arc?.points.map(p => p.level) ?? []);
  const projection = facts?.projection;
  const value = facts?.value;
  const fit = facts?.fit;
  const fitNames = (ids: string[]) => ids.map(id => context.snapshot.players.value.find(p =>
    p.id === id)?.name ?? id).join(', ');
  const worth = asset && lens && !lens.preparing ? lens.assets.get(asset.token) : undefined;
  const position = asset?.position ?? projection?.position;
  const need = lens && !lens.preparing ? lens.you.needs.find(x => x.position === position) : undefined;
  const changedStarters = fit?.side.changes.filter(c => c.kind === (fit.removing ? 'promoted' : 'benched'))
    .map(c => c.player);
  return <aside className="insp" aria-label="Inspector"><div className="ihead"><div className="top">
    <span className="pg">{research?.position || asset?.position || projection?.position || '—'}</span>
    <span className="on">{asset ? 'PLAYER' : unavailable}</span></div><div className="nm">
      {research?.name ?? asset?.name ?? unavailable}</div><div className="sub">{research?.owner ??
        (asset?.pick ? 'Pick' : 'Player')} · age {age ?? '—'} · {asset?.contract ?? '—'}</div></div>
    {context.native && (facts?.valuesPreparing ? <p role="status">Values are being prepared</p> :
    <div className="nums"><div><b>{value?.available ? value.nowPPG.toFixed(1) :
      <CompactAbsent pending={reading.pending} />}</b><span>pts / game</span></div><div>
      <b title={value?.words}>{value?.available ? value.dynastyPPG.toFixed(1) :
        <CompactAbsent pending={reading.pending} />}</b><span>dynasty value</span></div></div>)}
    <div className="lensrow"><div><b>{lensValue(worth, 'you')}</b><span>to you</span></div>
      <div><b>{lens?.them ? lensValue(worth, 'them') : <CompactAbsent />}</b><span>to them</span></div></div>
    <div className="why" title={worth?.words}>{asset ? lensWhy(lens, asset.token, Boolean(theirs)) :
      unavailable}</div>
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
    </Drawer><Drawer label="VALUE ARC" detail={arc?.points.length ? `age ${Math.floor(arc.points[0].age)} → ${
      Math.floor(arc.points[arc.points.length - 1].age)}` : undefined}>
      {arc?.points.length ? <div className="arc" title={arc.reason ? unavailable : undefined}>
        {arc.points.map(p => <i
        key={p.season}
        className={p.season === context.snapshot.league.value.season ? 'now' : ''}
        style={{ height: `${Math.max(0, p.level / maxArc * 100)}%` }}
        title={`${p.season} · age ${p.age.toFixed(1)} · ${p.level.toFixed(1)}`} />)}</div> :
          <WarRoomEmpty />}</Drawer>
    <Drawer label="IN YOUR LINEUP"><div className="kv"><span>
      {fit?.removing ? 'Promotes' : 'Replaces'}</span><span>{fit ?
      fitNames(changedStarters ?? []) || 'No starter change' : unavailable}</span>
      </div>{fit && <><div className="kv"><span>
        {fit.removing ? 'Without him / week' : 'Adds / week'}</span><span
        title={fit.notes.join(' · ')}>{fit.side.unprojected.length ? unavailable :
          fit.side.perWeek.toFixed(1)}</span></div>
        <div className="kv"><span>This week</span><span>{fit.side.unprojected.length ? unavailable :
          fit.side.thisWeek.toFixed(1)}</span></div></>}<div className="kv"><span>L5 avg</span><span>
          {researchFact(research, 'L5 avg') ?? unavailable}
      </span></div><div className="kv"><span>Fills need</span><span>{!need ? '—' : need.state === 'ok' ?
        `No: ${need.position} is covered` : theirs ? `Yes: ${need.position} is a ${need.state === 'hole' ?
          'hole' : 'thin spot'} next season` : `Trading him leaves ${need.position} ${need.state === 'hole' ?
          'a hole' : 'thin'}`}</span></div>
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
