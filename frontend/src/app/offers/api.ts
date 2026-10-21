import { TargetOffers } from '../../../wailsjs/go/offers/API';
import type * as Bindings from '../../../wailsjs/go/offers/API';
import { offers } from '../../../wailsjs/go/models';
import { commands } from '../commands/registry';
import { selectProvider } from '../data/provider';
import type { Scope } from './provider';
import type { BidFormState, PoolScope } from '../pool/contract';
import type { ListChange, ListKind } from '../personal/contract';

type Go = typeof Bindings;
function poolGo(): Go {
  if (selectProvider().kind !== 'live') throw new Error('The free-agent pool needs the desktop app');
  return (window as unknown as { go: { offers: { API: Go } } }).go.offers.API;
}
function personalGo(): Go {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Private lists need the desktop app; no sample data is substituted.');
  return (window as unknown as { go: { offers: { API: Go } } }).go.offers.API;
}
export async function readOffersWire(scope: Scope): Promise<unknown> {
  return TargetOffers(offers.RawOfferScope.createFrom(scope));
}
export function rawBid(franchiseId: string, form: BidFormState): offers.RawBidInput {
  const closes = form.closesDate && form.closesTime
    ? new Date(`${form.closesDate}T${form.closesTime}`) : undefined;
  return offers.RawBidInput.createFrom({
    franchiseId, player: form.playerId, salary: form.salary, years: form.years, snipe: form.snipe,
    conditionalDrop: form.conditionalDrop ?? '', beat: form.beat, evidence: form.evidence,
    closes: closes && !Number.isNaN(closes.getTime()) ? closes.toISOString() : '',
  });
}
export const poolWire = {
  refresh: (force: boolean) => poolGo().TargetPoolRefresh(force),
  pool: (franchiseId: string, scope: PoolScope) => poolGo().TargetPool(offers.RawPoolScope.createFrom({
    franchiseId, position: scope.position, sort: scope.sort, desc: scope.desc,
    showNoTeam: scope.showNoTeam ?? false,
  })),
  plans: (mine: string) => poolGo().TargetBidPlans(mine),
  preview: (mine: string, form: BidFormState) => poolGo().TargetBidPreview(rawBid(mine, form)),
  draft: (mine: string, form: BidFormState) => poolGo().TargetBidDraft(rawBid(mine, form)),
  revise: (id: string, mine: string, form: BidFormState) => poolGo().TargetBidRevise(id, rawBid(mine, form)),
  handOff: (id: string) => poolGo().TargetBidHandOff(id),
  posted: (id: string) => poolGo().TargetBidPosted(id),
  withdraw: (id: string) => poolGo().TargetBidWithdraw(id),
};
export const personalWire = {
  read: (mine: string) => personalGo().TargetPersonal(mine),
  change: (mine: string, kind: ListKind, change: ListChange) => personalGo().TargetPersonalChange({
    mine, kind, ...change, text: change.text ?? '',
  }),
  handoff: (kind: ListKind | 'ir') => personalGo().TargetPersonalHandOff(kind),
};

export const subscribeOffers = (changed: () => void) => selectProvider().onSeasonChange(changed);
export const subscribePersonal = (changed: () => void) => commands.alerts.onSeasonChange(changed);
export function subscribePool(changed: () => void): () => void {
  const moves = commands.onMovesChange(changed);
  const season = selectProvider().onSeasonChange(changed);
  return () => { moves(); season(); };
}
