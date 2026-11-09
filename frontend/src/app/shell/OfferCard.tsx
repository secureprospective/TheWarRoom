import type { TradeOffer } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { TradePlan } from './TradePlan';
import { tradeAssetLabel } from './tradeLabels';
import { localDate } from '../clock/labels';
import './trade.css';

const livePlan = new Set(['ready', 'handed_off', 'not_yet_done', 'not_verified', 'dot_review']);

export function tradeExpiry(expires: string, now = Date.now()): string {
  if (expires.startsWith('0001-')) return 'Expiry unknown';
  const hours = Math.floor((Date.parse(expires) - now) / 3_600_000);
  if (hours < 0) return 'expired';
  if (hours === 0) return 'expires in less than 1h';
  return `expires in ${Math.floor(hours / 24)}d ${hours % 24}h`;
}
export function OfferCard({ offer, franchiseId }: { offer: TradeOffer; franchiseId: string }) {
  const key = `trade:${franchiseId}:${offer.tradeId}`;
  const pending = commands.useMoves((s) => s.drafting[key]);
  const error = commands.useMoves((s) => s.draftErrors[key]);
  const receipt = commands.useMoves((s) => s.moves.find((r) =>
    r.spec.franchiseId === franchiseId && r.spec.intent === 'mfl.trade.accept' &&
    r.spec.expected.trade?.tradeId === offer.tradeId));
  const exits = commands.useMoves(s => s.moves).filter(r => r.spec.franchiseId === franchiseId &&
    r.spec.expected.trade?.tradeId === offer.tradeId && r.spec.intent !== 'mfl.trade.accept');
  // One live exit plan per offer: while one stands, the card offers its removal, not a second plan.
  const exitPlanned = exits.some(r => livePlan.has(r.state));
  const knownExpiry = !offer.expires.startsWith('0001-');
  return <Card gravity="G3" header={<h5><Act variant="text" verb="inspector.pin"
    args={{ entity: { kind: 'offer', id: offer.tradeId, franchiseId } }}>
    Offer with {offer.otherName || 'Franchise name unavailable'}</Act></h5>}>
    <p>Pending offer on MFL</p>
    <div className="trade-assets">
      {(['give', 'get'] as const).map((side) => <div key={side}>
        <h6>{side === 'give' ? 'You give' : 'You get'}</h6>
        <ul>{offer[side].map((asset) => <li key={asset.token}>
          <Act variant="text" verb="inspector.pin" args={{ entity: /^\d+$/.test(asset.token)
          ? { kind: 'player', id: asset.token, franchiseId }
          : { kind: 'pick', id: asset.token, name: tradeAssetLabel(asset) } }}>
          {tradeAssetLabel(asset)}</Act></li>)}</ul>
      </div>)}
    </div>
    <p>{knownExpiry && <time dateTime={offer.expires}>
      {localDate(offer.expires)}{' · '}
    </time>}{tradeExpiry(offer.expires)}</p>
    {offer.comments && <p className="trade-comments">{offer.comments}</p>}
    {offer.direction === 'to_you' && !(receipt && livePlan.has(receipt.state)) &&
      <div className="plan-actions">
      <Act variant="button" verb="trade.accept.plan" args={{ tradeId: offer.tradeId }}
        disabled={Boolean(pending)} label={pending ? 'Planning accept…' : 'Plan accept'} />
      {error && <p role="alert">Trade plan · {error}</p>}
    </div>}
    {!exitPlanned && <div className="plan-actions">
      <Act variant="button"
        verb="trade.builder" args={{ action: offer.direction === 'to_you' ? 'reject' : 'revoke',
        mine: franchiseId, id: offer.tradeId }}
        label={offer.direction === 'to_you' ? 'Plan rejection' : 'Plan withdrawal'} />
    </div>}
    {receipt && <TradePlan receipt={receipt} />}
    {exits.map(r => <TradePlan key={r.correlationId} receipt={r} />)}
  </Card>;
}
