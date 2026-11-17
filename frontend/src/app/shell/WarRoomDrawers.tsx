import { HQAct as WarRoomAct } from './HQAct';
import { useEffect, useState } from 'react';
import { targetTrades } from '../trades/api';
import { parseTrades } from '../data/parseTrades';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedTrades } from '../commissioner/hostedMarkets';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { OWNER_ONLY_TRADES } from './homeReads';
import { deleteWarRoomDraft, readWarRoomDrafts, type WarRoomSavedDraft } from './warroomDrafts';

export async function readWarRoomOffers(c: WarRoomContext) {
  if (!c.native) {
    const reading = parseTrades(await targetTrades(c.mine));
    if (reading.provenance.freshness.state === 'fail') throw new Error('Pending offers unavailable');
    return reading.offers.map(o => ({ id: o.tradeId, name: o.otherName, status: o.direction === 'to_you' ?
      'Incoming' : 'Outgoing', details: [...o.give.map(a => `You send ${a.name}`),
        ...o.get.map(a => `You get ${a.name}`)], note: o.comments }));
  }
  if (c.native.owner === false) throw new Error(OWNER_ONLY_TRADES);
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'trades.read', mode: 'read',
    arguments: { league: c.native.league, viewer: c.mine } });
  const reading = parseHostedTrades(data);
  if (reading.league !== c.native.league || reading.franchise !== c.mine)
    throw new Error('Trade offers belong to another viewer');
  return reading.offers.map(o => ({ id: o.id, name: o.id, status: o.words, details: o.details,
    note: o.expires }));
}

export function WarRoomDrawers({ context, drawer, close, openOffer, onOpenDraft, refreshKey = 0 }: {
  context: WarRoomContext; drawer: 'pocket' | 'offers'; close: () => void;
  openOffer: (id: string) => void; onOpenDraft?: (draft: WarRoomSavedDraft) => void;
  refreshKey?: number }) {
  const offers = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, drawer]), () =>
    drawer === 'offers' ? readWarRoomOffers(context) : Promise.resolve(undefined));
  const drafts = useWarRoomRead(
    JSON.stringify([context.native?.league, context.mine, drawer, refreshKey]),
    () => drawer === 'pocket' && context.native && context.native.owner !== false ?
      readWarRoomDrafts(context) : Promise.resolve(undefined));
  const [draftsAfterDelete, setDraftsAfterDelete] = useState<WarRoomSavedDraft[]>();
  const [deleting, setDeleting] = useState('');
  const [deleteError, setDeleteError] = useState('');
  useEffect(() => { setDraftsAfterDelete(undefined); setDeleteError(''); }, [drawer, refreshKey]);
  useEffect(() => { const key = (e: KeyboardEvent) => { if (e.key === 'Escape') close(); };
    document.addEventListener('keydown', key);
    return () => document.removeEventListener('keydown', key);
  }, [close]);
  const removeDraft = async (id: string) => {
    setDeleting(id); setDeleteError('');
    try { setDraftsAfterDelete(await deleteWarRoomDraft(context, id)); }
    catch (cause) { setDeleteError(String(cause)); }
    finally { setDeleting(''); }
  };
  const saved = draftsAfterDelete ?? drafts.reading;
  const showPrivatePocket = drawer === 'pocket' && context.native !== undefined;
  return <div className="bdrawer open" role="region" aria-label={drawer === 'pocket' ? 'Pocket' :
    'Negotiations'}><div className="bd-head"><h2>{drawer === 'pocket' ? 'Pocket' : 'Negotiations'}</h2>
    <span className="sub">{drawer === 'pocket' ?
      "Trades you built and haven't sent. Only you can see them. Each one re-checks itself." :
      'Every live trade talk. Open one to answer it on the desk.'}</span>
    <WarRoomAct className="x" onAction={close}>ESC</WarRoomAct></div><div className="bd-body pocket">
    {drawer === 'pocket' ? context.native === undefined ?
      <WarRoomEmpty reason={offers.error} pending={offers.pending} /> :
      context.native.owner === false ?
        <WarRoomEmpty reason="Only the seated owner can open this private pocket" /> :
        saved?.length ? saved.map(draft => <div className="pk" key={draft.id}>
          <div className="top">Saved · {draft.at}</div>
          <div className="ttl">{draft.label || `Trade with ${draft.partner}`}</div>
          <div className="legs"><span>Send: {draft.give.join(', ') || 'none'}</span>
            <span>Get: {draft.get.join(', ') || 'none'}</span></div>
          <WarRoomAct className="btn" disabled={Boolean(deleting)} onAction={() => onOpenDraft?.(draft)}>
            REOPEN ON DESK</WarRoomAct>
          <WarRoomAct className="btn" disabled={Boolean(deleting)}
            onAction={() => void removeDraft(draft.id)}>
            {deleting === draft.id ? 'DELETING…' : 'DELETE'}</WarRoomAct>
        </div>) : <WarRoomEmpty
          reason={deleteError || drafts.error || (saved ? 'No saved proposals yet' : undefined)}
          pending={drafts.pending} /> :
      offers.reading?.length ? offers.reading.map(o => <div className="pk" key={o.id}>
        <div className="top">{o.status}</div>
        <WarRoomAct className="ttl" disabled={Boolean(context.native)}
          onAction={() => openOffer(o.id)}>{o.name}</WarRoomAct>
        <div className="legs">{o.details.map((d, i) => <span key={i}>{d}</span>)}</div>
        <div className="stale">{o.note}</div></div>) : <WarRoomEmpty reason={offers.error} />}
    <div className="pkside">{showPrivatePocket ?
      <WarRoomEmpty reason="Private proposals are visible only to your team" /> :
      <WarRoomEmpty reason="Saved versions and negotiation rounds have no read" />}
    </div></div></div>;
}
