import { createMFLClient, type MFLContext } from '../commands/mfl';
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
const localOwner: MFLContext = { availability: 'desktop', role: 'gm' };
const ownerAction: MFLContext = { ...localOwner, ownerAction: true };
const mflClient = createMFLClient(request => {
  const go = poolGo();
  const args = request.arguments;
  switch (request.command) {
    case 'mfl.pool.refresh': return go.TargetPoolRefresh(request.arguments.force);
    case 'mfl.pool.read': return go.TargetPool(offers.RawPoolScope.createFrom(args));
    case 'mfl.bid.plans': return go.TargetBidPlans(request.arguments.franchiseId);
    case 'mfl.bid.preview': return go.TargetBidPreview(offers.RawBidInput.createFrom(args));
    case 'bid.submit': return go.TargetBidDraft(offers.RawBidInput.createFrom(args));
    case 'mfl.bid.revise': return go.TargetBidRevise(request.arguments.id, offers.RawBidInput.createFrom(
      Object.fromEntries(Object.entries(args).filter(([key]) => key !== 'id')),
    ));
    case 'mfl.bid.handoff': return go.TargetBidHandOff(request.arguments.id);
    case 'mfl.bid.posted': return go.TargetBidPosted(request.arguments.id);
    case 'mfl.bid.withdraw': return go.TargetBidWithdraw(request.arguments.id);
  }
});
export const poolWire = {
  refresh: (force: boolean) => mflClient.invoke({ command: 'mfl.pool.refresh', mode: 'execute', arguments: { force } }, localOwner),
  pool: (franchiseId: string, scope: PoolScope) => mflClient.invoke({ command: 'mfl.pool.read', mode: 'read', arguments: {
    franchiseId, position: scope.position, sort: scope.sort, desc: scope.desc, showNoTeam: scope.showNoTeam ?? false,
  } }, localOwner),
  plans: (franchiseId: string) => mflClient.invoke({ command: 'mfl.bid.plans', mode: 'read', arguments: { franchiseId } }, localOwner),
  preview: (mine: string, form: BidFormState) => mflClient.invoke({ command: 'mfl.bid.preview', mode: 'preview', arguments: rawBid(mine, form) }, localOwner),
  draft: (mine: string, form: BidFormState) => mflClient.invoke({ command: 'bid.submit', mode: 'execute', arguments: rawBid(mine, form) }, localOwner),
  revise: (id: string, mine: string, form: BidFormState) => mflClient.invoke({ command: 'mfl.bid.revise', mode: 'execute', arguments: { id, ...rawBid(mine, form) } }, localOwner),
  handOff: (id: string) => mflClient.invoke({ command: 'mfl.bid.handoff', mode: 'execute', arguments: { id } }, ownerAction),
  posted: (id: string) => mflClient.invoke({ command: 'mfl.bid.posted', mode: 'execute', arguments: { id } }, ownerAction),
  withdraw: (id: string) => mflClient.invoke({ command: 'mfl.bid.withdraw', mode: 'execute', arguments: { id } }, ownerAction),
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
