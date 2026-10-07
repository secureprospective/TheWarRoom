import { useEffect } from 'react';
import type { Snapshot, TradeOffer, TradeReading } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { SignalChip } from '../look/Slots';
import { provenanceSlot } from '../cards/playerModel';
import { FranchisePicker } from './FranchisePicker';
import { useReading } from './lineups';
import { TradePlan } from './TradePlan';
import { tradeAssetLabel, tradeRailLabels } from './tradeLabels';
import './lineup.css';
import './trade.css';

const livePlan = new Set(['ready', 'handed_off', 'not_yet_done', 'not_verified', 'dot_review']);

export function tradeExpiry(expires: string, now = Date.now()): string {
  if (expires.startsWith('0001-')) return 'Expiry unknown';
  const hours = Math.floor((Date.parse(expires) - now) / 3_600_000);
  if (hours < 0) return 'expired';
  if (hours === 0) return 'expires in less than 1h';
  return `expires in ${Math.floor(hours / 24)}d ${hours % 24}h`;
}
function OfferCard({ offer, franchiseId }: { offer: TradeOffer; franchiseId: string }) {
  const key = `trade:${franchiseId}:${offer.tradeId}`;
  const pending = commands.useMoves((s) => s.drafting[key]);
  const error = commands.useMoves((s) => s.draftErrors[key]);
  const receipt = commands.useMoves((s) => s.moves.find((r) =>
    r.spec.franchiseId === franchiseId && r.spec.expected.trade?.tradeId === offer.tradeId));
  const knownExpiry = !offer.expires.startsWith('0001-');
  return <Card gravity="G2" header={<h5>{offer.otherName || offer.otherId}</h5>}>
    <div className="trade-assets">
      {(['give', 'get'] as const).map((side) => <div key={side}>
        <h6>{side === 'give' ? 'You give' : 'You get'}</h6>
        <ul>{offer[side].map((asset) => <li key={asset.token}>{tradeAssetLabel(asset)}</li>)}</ul>
      </div>)}
    </div>
    <p>{knownExpiry && <time dateTime={offer.expires}>
      {new Date(offer.expires).toLocaleString()}{' · '}
    </time>}{tradeExpiry(offer.expires)}</p>
    {offer.comments && <p className="trade-comments">{offer.comments}</p>}
    {offer.direction === 'to_you' && !(receipt && livePlan.has(receipt.state)) &&
      <div className="plan-actions">
      <Act verb="trade.accept.plan" args={{ tradeId: offer.tradeId }}
        disabled={Boolean(pending)} label={pending ? 'Planning accept…' : 'Plan accept'} />
      {error && <p role="alert">Trade plan · {error}</p>}
    </div>}
    {receipt && offer.direction === 'to_you' && <TradePlan receipt={receipt} />}
  </Card>;
}
function FranchiseTrades({ franchiseId, snapshot }: { franchiseId: string; snapshot: Snapshot }) {
  const { reading, error } = useReading(commands.trades, franchiseId, snapshot);
  useEffect(() => {
    const read = () => { void commands.loadMoves(franchiseId); };
    read();
    return commands.onMovesChange(read);
  }, [franchiseId]);
  const freshness = reading?.provenance.freshness;
  const failed = freshness?.state === 'fail' || (freshness?.state === 'stale' && freshness.note);
  const reason = error || (failed && (freshness?.note || 'feed failed'));
  return <>
    <PendingOffers reading={reading} reason={reason || undefined} franchiseId={franchiseId} />
    <SavedTradePlans franchiseId={franchiseId} snapshot={snapshot}
      visibleOffers={reason ? [] : reading?.offers.map((offer) => offer.tradeId) ?? []} />
  </>;
}
function PendingOffers({ reading, reason, franchiseId }: {
  reading?: TradeReading; reason?: string; franchiseId: string;
}) {
  if (reason) return <p role="alert">Pending trades unavailable · {reason}</p>;
  if (!reading) return <p role="status">Reading pending trades…</p>;
  return <>
    <div className="trade-provenance">
      {!reading.offers.length && <p>No pending offers</p>}
      <SignalChip {...provenanceSlot([['Pending trades', reading.provenance]])} />
    </div>
    {(['to_you', 'by_you'] as const).map((direction) => {
      const offers = reading.offers.filter((offer) => offer.direction === direction);
      return offers.length > 0 && <section key={direction} className="trade-group">
        <h4>{direction === 'to_you' ? 'Offered to you' : 'Your offers'} · {offers.length}</h4>
        {offers.map((offer) => <OfferCard key={offer.tradeId} offer={offer} franchiseId={franchiseId} />)}
      </section>;
    })}
  </>;
}
function SavedTradePlans({ franchiseId, snapshot, visibleOffers }: {
  franchiseId: string; snapshot: Snapshot; visibleOffers: string[];
}) {
  const moves = commands.useMoves((s) => s.moves);
  const seen = new Set(visibleOffers);
  const plans = moves.filter((receipt) => {
    const tradeId = receipt.spec.expected.trade?.tradeId;
    if (receipt.spec.franchiseId !== franchiseId || !tradeId || seen.has(tradeId)) return false;
    if (!receipt.audit.some((entry) => entry.to === 'handed_off')) return false;
    seen.add(tradeId);
    return true;
  });
  if (!plans.length) return null;
  return <section className="trade-group" aria-label="Saved trade plans">
    <h4>Saved trade plans</h4>
    {plans.map((receipt) => {
      const labels = tradeRailLabels(receipt, snapshot)!;
      return <Card key={receipt.correlationId} gravity="G2" header={<h5>{labels.header}</h5>}>
        <p>{labels.summary}</p>
        <TradePlan receipt={receipt} />
      </Card>;
    })}
  </section>;
}
export default function TradeDesk({ snapshot }: { snapshot: Snapshot }) {
  const franchiseId = commands.use().franchiseId;
  if (!franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return <section className="trade-desk" aria-label="Trade desk and offers">
    <FranchiseTrades key={franchiseId} franchiseId={franchiseId} snapshot={snapshot} />
  </section>;
}
