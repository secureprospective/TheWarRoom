import { HQAct as WarRoomAct } from './HQAct';
import { useEffect, useMemo, useState } from 'react';
import { targetTrades } from '../trades/api';
import { parseTrades } from '../data/parseTrades';
import { hostedWorkflow } from '../commissioner/api';
import { createHostedLeagueRoom } from '../commissioner/hostedRoom';
import { reviewHostedOwnerTrade } from '../commissioner/HostedWorkflows';
import { Review } from '../commissioner/HostedReview';
import { parseHostedTrades } from '../commissioner/hostedMarkets';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { OWNER_ONLY_TRADES } from './homeReads';
import { isEndedTrade } from './warroomPlansReads';
import { deleteWarRoomDraft, readWarRoomDrafts, type WarRoomSavedDraft } from './warroomDrafts';
import { dealFromWarRoomOffer, type WarRoomDeal, type WarRoomOffer } from './warroomTradeReads';

export type WarRoomDrawerOffer = { id: string; name: string; status: string; details: string[]; note: string;
  source?: WarRoomOffer };
export type WarRoomOffers = { revision?: number; notes: string[]; offers: WarRoomDrawerOffer[] };

export async function readWarRoomOffers(c: WarRoomContext): Promise<WarRoomOffers> {
  if (!c.native) {
    const reading = parseTrades(await targetTrades(c.mine));
    if (reading.provenance.freshness.state === 'fail') throw new Error('Pending offers unavailable');
    return { notes: [], offers: reading.offers.map(o => ({ id: o.tradeId, name: o.otherName,
      status: o.direction === 'to_you' ? 'Incoming' : 'Outgoing',
      details: [...o.give.map(a => `You send ${a.name}`), ...o.get.map(a => `You get ${a.name}`)],
      note: o.comments })) };
  }
  if (c.native.owner === false) throw new Error(OWNER_ONLY_TRADES);
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'trades.read', mode: 'read',
    arguments: { league: c.native.league, viewer: c.mine } });
  const reading = parseHostedTrades(data);
  if (reading.league !== c.native.league || reading.franchise !== c.mine)
    throw new Error('Trade offers belong to another viewer');
  return { revision: reading.revision, notes: reading.notes, offers: reading.offers
    .filter(o => !isEndedTrade(o.status)).map(o => ({ id: o.id, name: partnerName(o, c.mine),
      status: o.words, details: [...o.details, ...o.reasons], note: o.expires, source: o })) };
}

/** partnerName is the other team's held name; the trade id only when the read names no partner. */
function partnerName(offer: WarRoomOffer, mine: string): string {
  const names = new Map(offer.teams);
  const others = offer.parties.filter(id => id !== mine).map(id => names.get(id) ?? id);
  return others.length ? others.join(' · ') : offer.id;
}

function NativeOffers({ context, reading, onLoadOffer, onChanged }: {
  context: WarRoomContext; reading: WarRoomOffers; onLoadOffer?: (deal: WarRoomDeal) => void;
  onChanged: () => void }) {
  const native = context.native!;
  const room = useMemo(() => createHostedLeagueRoom(native.league, native.port ?? hostedWorkflow),
    [native.league, native.port]);
  const state = room.use();
  const [actionError, setActionError] = useState('');
  useEffect(() => {
    void room.refresh();
    return room.dispose;
  }, [room]);
  const act = (offer: WarRoomOffer, action: 'trade.accept' | 'trade.decline' | 'trade.withdraw') => {
    if (reading.revision === undefined) return;
    setActionError('');
    void reviewHostedOwnerTrade(room, native.league, context.mine, offer.id, reading.revision, action)
      .catch(cause => setActionError(String(cause)));
  };
  return <>
    {reading.notes.map(note => <p className="offer-note" key={note}>{note}</p>)}
    {!reading.offers.length && <WarRoomEmpty reason="No trade offers" />}
    {reading.offers.map(offer => {
      const source = offer.source;
      const inLeague = source?.parties.includes(context.mine) ?? false;
      const open = source?.status === 'offered' && inLeague;
      const outgoing = open && source?.proposer === context.mine;
      const incoming = open && source?.proposer !== context.mine;
      const needsAccept = incoming && !source?.accepted.includes(context.mine);
      return <div className="pk" key={offer.id}>
        <div className="top">{offer.status}</div>
        <div className="ttl">{offer.name}</div>
        <div className="legs">{offer.details.map((d, i) => <span key={i}>{d}</span>)}</div>
        <div className="stale">Expires {offer.note}</div>
        {onLoadOffer && source && <WarRoomAct className="btn" onAction={() => {
          try { onLoadOffer(dealFromWarRoomOffer(source, context.mine)); setActionError(''); }
          catch (cause) { setActionError(String(cause)); }
        }}>LOAD ONTO DESK</WarRoomAct>}
        {incoming && <div className="offer-actions">
          {needsAccept && <WarRoomAct className="btn"
            disabled={!room.canChange() || reading.revision === undefined}
            onAction={() => act(source!, 'trade.accept')}>ACCEPT</WarRoomAct>}
          <WarRoomAct className="btn" disabled={!room.canChange() || reading.revision === undefined}
            onAction={() => act(source!, 'trade.decline')}>DECLINE</WarRoomAct>
        </div>}
        {outgoing && <WarRoomAct className="btn"
          disabled={!room.canChange() || reading.revision === undefined}
          onAction={() => act(source!, 'trade.withdraw')}>WITHDRAW</WarRoomAct>}
      </div>;
    })}
    {state.error && <p role="alert">{state.error}</p>}
    {actionError && <p role="alert">{actionError}</p>}
    {state.message && <p role="status">{state.message}</p>}
    {state.review && <Review room={room} onConfirmed={onChanged} />}
  </>;
}

export function WarRoomDrawers({ context, drawer, close, openOffer, onOpenDraft, onLoadOffer,
  refreshKey = 0 }: {
  context: WarRoomContext; drawer: 'pocket' | 'offers'; close: () => void;
  openOffer: (id: string) => void; onOpenDraft?: (draft: WarRoomSavedDraft) => void;
  onLoadOffer?: (deal: WarRoomDeal) => void; refreshKey?: number }) {
  const [offersRefresh, setOffersRefresh] = useState(0);
  const offers = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, drawer, refreshKey,
    offersRefresh]), () => drawer === 'offers' ? readWarRoomOffers(context) : Promise.resolve(undefined));
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
      context.native && offers.reading ? <NativeOffers context={context} reading={offers.reading}
        onLoadOffer={onLoadOffer} onChanged={() => setOffersRefresh(value => value + 1)} /> :
      offers.reading?.offers.length ? offers.reading.offers.map(o => <div className="pk" key={o.id}>
        <div className="top">{o.status}</div>
        <WarRoomAct className="ttl" onAction={() => openOffer(o.id)}>{o.name}</WarRoomAct>
        <div className="legs">{o.details.map((d, i) => <span key={i}>{d}</span>)}</div>
        <div className="stale">{o.note}</div></div>) :
        <WarRoomEmpty reason={offers.error} pending={offers.pending} />}
    <div className="pkside">{showPrivatePocket ?
      <WarRoomEmpty reason="Private proposals are visible only to your team" /> :
      <WarRoomEmpty reason="Saved versions and negotiation rounds have no read" />}
    </div></div></div>;
}
