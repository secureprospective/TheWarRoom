import { POSITIONS } from './contract';
import type { TradeAsset, TradeOffer, TradeReading } from './contract';
import { object, array, choice, id, text, requiredText, timestamp } from './parseValues';
import { provenance } from './parse';

function asset(value: unknown, path: string): TradeAsset {
  const r = object(value, path, ['token', 'name', 'position']);
  return {
    token: requiredText(r.token, `${path}.token`),
    name: text(r.name, `${path}.name`),
    ...(r.position === undefined ? {} : { position: choice(r.position, `${path}.position`, POSITIONS) }),
  };
}
function offer(value: unknown, path: string): TradeOffer {
  const r = object(value, path, [
    'tradeId', 'direction', 'otherId', 'otherName', 'give', 'get', 'expires', 'comments',
  ]);
  const expires = timestamp(r.expires, `${path}.expires`);
  if (!expires) throw new Error(`${path}.expires: expected timestamp`);
  return {
    tradeId: requiredText(r.tradeId, `${path}.tradeId`),
    direction: choice(r.direction, `${path}.direction`, ['to_you', 'by_you'] as const),
    otherId: id(r.otherId, `${path}.otherId`),
    otherName: text(r.otherName, `${path}.otherName`),
    give: array(r.give, `${path}.give`, asset),
    get: array(r.get, `${path}.get`, asset),
    expires,
    comments: text(r.comments, `${path}.comments`),
  };
}
export function parseTrades(value: unknown): TradeReading {
  const r = object(value, 'trades', ['offers', 'provenance']);
  return {
    offers: array(r.offers, 'trades.offers', offer),
    provenance: provenance(r.provenance, 'trades.provenance'),
  };
}
