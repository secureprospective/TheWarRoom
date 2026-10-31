import { useEffect } from 'react';
import { targetTrades } from '../trades/api';
import { parseTrades } from '../data/parseTrades';
import { hostedWorkflow } from '../commissioner/api';
import { parseHostedTrades } from '../commissioner/hostedMarkets';
import { useWarRoomRead, type WarRoomContext } from './warroomReads';
import { WarRoomEmpty } from './WarRoomHeader';
export async function readWarRoomOffers(c: WarRoomContext) {
  if (!c.native) {
    const reading = parseTrades(await targetTrades(c.mine));
    if (reading.provenance.freshness.state === 'fail') throw new Error('Pending offers unavailable');
    return reading.offers.map(o => ({ id: o.tradeId, name: o.otherName, status: o.direction === 'to_you' ?
      'Incoming' : 'Outgoing', details: [...o.give.map(a => `You send ${a.name}`),
        ...o.get.map(a => `You get ${a.name}`)], note: o.comments }));
  }
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'trades.read', mode: 'read',
    arguments: { league: c.native.league, viewer: c.mine } });
  const reading = parseHostedTrades(data);
  if (reading.league !== c.native.league || reading.franchise !== c.mine)
    throw new Error('Trade offers belong to another viewer');
  return reading.offers.map(o => ({ id: o.id, name: o.id, status: o.words, details: o.details,
    note: o.expires }));
}
export function WarRoomDrawers({ context, drawer, close }: { context: WarRoomContext;
  drawer: 'pocket' | 'offers'; close: () => void }) {
  const offers = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, drawer]), () =>
    drawer === 'offers' ? readWarRoomOffers(context) : Promise.resolve(undefined));
  useEffect(() => { const key = (e: KeyboardEvent) => { if (e.key === 'Escape') close(); };
    document.addEventListener('keydown', key);
    return () => document.removeEventListener('keydown', key);
  }, [close]);
  return <div className="bdrawer open" role="region" aria-label={drawer === 'pocket' ? 'Pocket' :
    'Negotiations'}><div className="bd-head"><h2>{drawer === 'pocket' ? 'Pocket' : 'Negotiations'}</h2>
    <span className="sub">{drawer === 'pocket' ?
      "Trades you built and haven't sent. Only you can see them. Each one re-checks itself." :
      'Every live trade talk. Open one to answer it on the desk.'}</span>
    <button className="x" onClick={close}>ESC</button></div><div className="bd-body pocket">
    {drawer === 'offers' && offers.reading?.length ? offers.reading.map(o => <div className="pk"
      key={o.id}><div className="top">{o.status}</div><div className="ttl">{o.name}</div>
      <div className="legs">{o.details.map((d, i) => <span key={i}>{d}</span>)}</div>
      <div className="stale">{o.note}</div></div>) : <WarRoomEmpty reason={offers.error} />}
    <div className="pkside"><WarRoomEmpty reason="Saved versions and negotiation rounds have no read" />
    </div></div></div>;
}
