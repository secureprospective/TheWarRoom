import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { marketAPI, subscribeSeason } from '../market/api';
import { parseAssetBoard, type AssetBoard, type AssetCell } from './contract';

type AssetState = { board?: AssetBoard; byId: ReadonlyMap<string, AssetBoard['values'][number]>;
  pending: boolean; error: string };
export type AssetKey = 'now' | 'dynasty' | 'market';

// The asset board loads once on first use and again whenever the season's held feeds change.
export function createAssetState(read: () => Promise<AssetBoard>,
  onChange: (listener: () => void) => () => void) {
  const store = createStore<AssetState>(() => ({ byId: new Map(), pending: false, error: '' }));
  let revision = 0;
  let started = false;
  async function load() {
    const request = ++revision;
    store.setState({ pending: true });
    try {
      const board = await read();
      if (request === revision)
        store.setState({ board, byId: new Map(board.values.map(v => [v.id, v])), error: '' });
    } catch (cause) {
      if (request === revision) store.setState({ error: String(cause) });
    } finally {
      if (request === revision) store.setState({ pending: false });
    }
  }
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
