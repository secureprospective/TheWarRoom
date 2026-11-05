import { useEffect } from 'react';
import type { Snapshot, TradeReading } from '../data/contract';
import { commands } from '../commands/registry';
import { Card } from '../cards/Card';
import { SignalChip } from '../look/Slots';
import { provenanceSlot } from '../cards/playerModel';
import { FranchisePicker } from './FranchisePicker';
import { useReading } from './lineups';
import { TradePlan } from './TradePlan';
import { tradeRailLabels } from './tradeLabels';
import './lineup.css';
import './trade.css';
import TradeBuilder from './TradeBuilder';
import { OfferCard } from './OfferCard';

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
    <TradeBuilder mine={franchiseId} />
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
    if (receipt.spec.intent === 'trade.propose') return false;
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
  return <section className="trade-desk" aria-label="Trade desk">
    {franchiseId ? (
      <FranchiseTrades key={franchiseId} franchiseId={franchiseId} snapshot={snapshot} />
    ) : <FranchisePicker snapshot={snapshot} />}
  </section>;
}
