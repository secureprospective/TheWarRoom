// Desktop v3 War Room, Draft room, after TheWarRoom-mock-v3-FINAL.html (screens.js war.draft): the clock
// and picks made, the board of players still available, my picks and my auto-pick queue. Reads are the
// War Room's own (draft.read, draft.queue.read); making a pick stays in the league draft room, opened below.
import { lazy, Suspense, useEffect, useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { useWarRoomRead, type WarRoomContext } from '../shell/warroomReads';
import { readWarRoomDraft, saveWarRoomDraftQueue } from '../shell/warroomDraftReads';
import type { Draft } from '../draftroom/draft';

const DraftRoom = lazy(() => import('../draftroom/DraftRoom'));
const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
// The server sends year 1 for a date it does not have yet.
const when = (at?: string) => at && Date.parse(at) > 0
  ? new Date(at).toLocaleString('en-US', { month: 'short', day: 'numeric',
  hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' }) + ' CT' : '';
function Panel({ title, badge, children, foot, style }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" /></div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** Picks made, newest first, with the drafted player. */
export function madePicks(draft: Draft) {
  const owner = (id: string) => draft.owners.find(o => o.id === id)?.name ?? id;
  return draft.picks.filter(p => p.player || p.skipped || p.forfeited)
    .sort((a, b) => b.round - a.round || b.slot - a.slot)
    .map(p => ({ key: `${p.round}:${p.original}`, no: `${p.round}.${String(p.slot).padStart(2, '0')}`,
      team: owner(p.owner), player: p.skipped ? 'skipped' : p.forfeited ? 'forfeited'
        : draft.players[p.player]?.name ?? p.player, position: draft.players[p.player]?.position }));
}

export function WarDraftV3({ context, badge }: { context: WarRoomContext; badge: ReactNode }) {
  const native = context.native;
  const reading = useWarRoomRead(JSON.stringify([native?.league, context.mine]), () =>
    readWarRoomDraft(context));
  const draft = reading.reading?.draft;
  const owner = native?.owner !== false;
  const queueKey = JSON.stringify([native?.league, context.mine]);
  const [held, setHeld] = useState<{ key: string; players: string[] }>();
  const [saving, setSaving] = useState(false);
  const [problem, setProblem] = useState('');
  const [room, setRoom] = useState(false);
  useEffect(() => { setHeld(undefined); setProblem(''); }, [queueKey]);
  const queue = held?.key === queueKey ? held.players : reading.reading?.queue;
  const save = async (players: string[]) => {
    if (!native || !queue || saving) return;
    setSaving(true); setProblem('');
    try { setHeld({ key: queueKey, players: await saveWarRoomDraftQueue(native, players) }); }
    catch { setProblem('The queue was not saved. Try again.'); }
    finally { setSaving(false); }
  };
  const move = (i: number, d: number) => {
    if (!queue) return;
    const next = [...queue]; const [p] = next.splice(i, 1); next.splice(i + d, 0, p); void save(next);
  };
  const name = (id: string) => draft?.owners.find(o => o.id === id)?.name ?? id;
  const mine = draft?.order.flatMap(r => r.picks).filter(p => p.owner === context.mine) ?? [];
  const made = draft ? madePicks(draft) : [];
  const clock = draft?.onClock;
  return <div className="wbody scr scr-war"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1fr) minmax(0,1.3fr) minmax(0,1fr) minmax(0,.9fr)',
    gridTemplateRows: room ? 'minmax(0,1fr) minmax(0,1.2fr)' : 'minmax(0,1fr)' }}>
    <Panel title="Draft clock and board" badge={badge} foot={draft?.words || 'newest pick first'}>
      {!draft ? <St>{reading.error ? `The draft could not be read. ${reading.error}`
        : 'Reading the draft…'}</St>
        : <>{clock ? <div className="row" style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto' }}>
          <span className="tag o">ON THE CLOCK</span><span className="ell">{clock.round}.{clock.slot} {name(
            clock.owner)}</span>
          <span className="m">{clock.paused ? 'paused' : when(clock.deadline)}</span></div>
          : <St>{draft.started ? draft.exhausted ? 'The draft is complete.' : 'No pick is on the clock.'
            : `The draft has not started.${when(draft.start)
              ? ` It opens ${when(draft.start)}.` : ''}`}</St>}
          {made.map(p => <div key={p.key} className="row"
            style={{ gridTemplateColumns: '38px minmax(0,1fr) minmax(0,1fr)' }}>
            <span className="m">{p.no}</span><span className="ell">{p.team}</span>
            <span className="ell">{p.player}{p.position ? ` · ${p.position}` : ''}</span></div>)}</>}
    </Panel>
    <Panel title="Player board" badge={badge}
      foot={draft ? `${draft.available.length} available · ${made.length} taken` : ''}>
      {!draft ? <St>Reading the board…</St> : !draft.available.length ? <St>No players are available.</St>
        : draft.available.map(p => <div key={p.id} className="row"
          style={{ gridTemplateColumns: '30px minmax(0,1fr) 44px auto' }}>
          <span className="pos">{p.position}</span><span className="ell">{p.name}</span>
          <span className="m">{p.team}</span><span className="acts">{queue?.includes(p.id)
            ? <span className="tag">QUEUED</span>
            : <HQAct className="btn sm" disabled={!owner || !queue || saving}
              title={!owner ? 'Only this franchise\'s owner keeps a queue' : undefined}
              onAction={() => { if (queue) void save([...queue, p.id]); }}>+ Queue</HQAct>}</span></div>)}
    </Panel>
    <Panel title="My picks" badge={badge} foot="from the league's draft order">
      {!draft ? <St>Reading…</St> : !draft.order.some(r => r.picks.length)
        ? <St>The draft order is not set yet.</St>
        : !mine.length ? <St>This franchise holds no picks in this draft.</St>
        : mine.map(p => <div key={`${p.round}:${p.original}`} className="row"
          style={{ gridTemplateColumns: '44px minmax(0,1fr)' }}>
          <span className="m">{p.round}.{String(p.slot).padStart(2, '0')}</span>
          <span className="ell">{p.original === context.mine ? 'own'
            : `from ${name(p.original)}`}</span></div>)}
    </Panel>
    <Panel title="My queue" badge={badge} foot="top of the queue is auto-picked when your clock runs out">
      {!owner ? <St>The queue is visible only to this franchise's owner.</St>
        : !queue ? <St>{reading.reading?.queueError ?? 'Reading your queue…'}</St>
          : !queue.length ? <St>Queue players from the board.</St>
            : queue.map((id, i) => <div key={id} className="row"
              style={{ gridTemplateColumns: '18px 30px minmax(0,1fr) auto' }}>
              <span className="m">{i + 1}</span>
              <span className="pos">{draft?.players[id]?.position ?? '—'}</span>
              <span className="ell">{draft?.players[id]?.name ?? id}</span><span className="acts">
                <HQAct className="btn sm" disabled={saving || i === 0} onAction={() => move(i, -1)}>▲</HQAct>
                <HQAct className="btn sm" disabled={saving || i === queue.length - 1}
                  onAction={() => move(i, 1)}>▼</HQAct>
                <HQAct className="btn sm" disabled={saving}
                  onAction={() => void save(queue.filter(x => x !== id))}>✕</HQAct></span></div>)}
      {problem && <St><span role="alert">{problem}</span></St>}
      {native && <div className="tbar"><HQAct className="btn sm" onAction={() => setRoom(v => !v)}>
        {room ? 'Close the draft room' : 'Open the draft room to pick ▾'}</HQAct></div>}
    </Panel>
    {room && native && <section className="pc panel sz-L" style={{ gridColumn: '1/-1', overflow: 'auto' }}>
      <div className="ph">{badge}<b className="pt">League draft room</b><span className="sp" /></div>
      <div className="pb"><Suspense fallback={<St>Opening the draft room…</St>}>
        <DraftRoom league={native.league} /></Suspense></div></section>}
  </div></div>;
}
