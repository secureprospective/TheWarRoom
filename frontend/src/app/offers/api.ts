import { commands } from '../commands/registry';
import { selectProvider } from '../data/provider';
import type { BidFormState, PoolScope } from '../pool/contract';
import type { ListChange, ListKind } from '../personal/contract';

const unavailable = async (): Promise<never> => {
  throw new Error('The free-agent pool needs the desktop app');
};
export const poolWire = {
  refresh: (_force: boolean) => unavailable(),
  pool: (_mine: string, _scope: PoolScope) => unavailable(),
  plans: (_mine: string) => unavailable(),
  preview: (_mine: string, _form: BidFormState) => unavailable(),
  draft: (_mine: string, _form: BidFormState) => unavailable(),
  revise: (_id: string, _mine: string, _form: BidFormState) => unavailable(),
  handOff: (_id: string) => unavailable(),
  posted: (_id: string) => unavailable(),
  withdraw: (_id: string) => unavailable(),
};
const privateLists = async (): Promise<never> => {
  throw new Error('Private lists need the desktop app; no sample data is substituted.');
};
export const personalWire = {
  read: (_mine: string) => privateLists(),
  change: (_mine: string, _kind: ListKind, _change: ListChange): Promise<void> => privateLists(),
  handoff: (_kind: ListKind | 'ir') => privateLists(),
};

export const subscribeOffers = (changed: () => void) => selectProvider().onSeasonChange(changed);
export const subscribePersonal = (changed: () => void) => commands.alerts.onSeasonChange(changed);
export function subscribePool(changed: () => void): () => void {
  const moves = commands.onMovesChange(changed);
  const season = selectProvider().onSeasonChange(changed);
  return () => { moves(); season(); };
}

export const readPersonalNotes = (_mine: string, _player: string): Promise<unknown> => privateLists();
export const swapPersonalTargets = (_mine: string, _id: string, _withPlayer: string) => privateLists();
