import { createReadQuery } from '../data/query';
import { useStore } from 'zustand';
import { marketAPI, subscribeSeason } from '../market/api';
import { parseAssetBoard, type AssetBoard, type AssetCell } from './contract';

type AssetState = { board?: AssetBoard; byId: ReadonlyMap<string, AssetBoard['values'][number]>;
  pending: boolean; error: string };
export type AssetKey = 'now' | 'dynasty' | 'market';

// The asset board loads once on first use and again whenever the season's held feeds change.
export function createAssetState(read: () => Promise<AssetBoard>,
  onChange: (listener: () => void) => () => void) {
  const query = createReadQuery('market.TargetAssetBoard', {
    initial: (): AssetState => ({ byId: new Map(), pending: false, error: '' }),
    retainErrorWhilePending: true,
    read, accept: board => ({ board, byId: new Map(board.values.map(v => [v.id, v])) }),
  });
  const store = query.store;
  let started = false;
  const load = () => query.load([]);
  function start() {
    if (started) return;
    started = true;
    onChange(() => { void load(); });
    void load();
  }
  // cellFor words a player's cell, including while the board is still loading or failed.
  function cellFor(s: AssetState, playerId: string, key: AssetKey): AssetCell {
    const value = s.byId.get(playerId);
    if (value) return value[key];
    if (s.error) return { text: 'Unavailable', sub: '', detail: s.error, held: false };
    if (!s.board) return { text: 'Reading…', sub: '', detail: '', held: false };
    return key === 'now' ? { text: 'No score', sub: 'this season', detail: '', held: false }
      : { text: 'Not valued', sub: 'not on a held roster', detail: '', held: false };
  }
  function useCell(playerId: string, key: AssetKey): AssetCell {
    start();
    return cellFor(useStore(store), playerId, key);
  }
  function useBoard() {
    start();
    return useStore(store);
  }
  return { useCell, useBoard, load, peek: store.getState, cellFor };
}

export const assetState = createAssetState(
  async () => parseAssetBoard(await marketAPI.assetBoard()),
  subscribeSeason,
);
