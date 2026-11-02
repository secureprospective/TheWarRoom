import { useEffect, useMemo } from 'react';
import { createReadQuery } from '../data/query';
import type { ReadArguments, ReadCommandId } from '../commands/reads.gen';
import { selectProvider } from '../data/provider';
import { commands } from '../commands/registry';
import { controlAPI } from '../control/api';
import { report } from '../reports/contract';
import { raceAPI } from '../race/api';

export function useHQRead<K extends ReadCommandId, T>(command: K, args: ReadArguments<K>,
  read: (...args: ReadArguments<K>) => Promise<T>, enabled = true) {
  const query = useMemo(() => createReadQuery(command, {
    initial: (): { value?: T; scope?: string } => ({}), read,
    accept: (value, accepted) => ({ value, scope: JSON.stringify(accepted) }),
  }), [command, read]);
  const key = JSON.stringify(args);
  useEffect(() => {
    if (!enabled) return;
    const load = () => { void query.load(args); };
    void query.load(args, { overlap: 'join' });
    const season = selectProvider().onSeasonChange(load);
    const moves = commands.onMovesChange(load);
    return () => { query.fence(); season(); moves(); };
    // The canonical binding arguments, not the array's render identity, own the request lifetime.
  }, [query, key, enabled]);
  const state = query.use();
  // A re-read of the same scope keeps showing what was read: a slow reading must not blank the screen.
  return { ...state, value: enabled && state.scope === key && !state.error ? state.value : undefined };
}
export const readHQSettings = async () => report(await controlAPI.settings(), 'League settings');
export const readHQRace = (week: number, team: string, mine: string, mode: string) =>
  raceAPI.read(week, team, mine, mode);

export const hqEvents = { onChange: (load: () => void) => {
  const season = selectProvider().onSeasonChange(load);
  const moves = commands.onMovesChange(load);
  return () => { season(); moves(); };
} };

export async function addHQTrade(mine: string, player: string): Promise<void> {
  const { builderAction, useBuilder } = await import('./tradeBuilderState');
  await builderAction({ action: 'read', mine });
  const state = useBuilder.getState();
  if (state.error) throw new Error(state.error);
  const asset = state.reading?.value?.teams.find(t => t.id === mine)?.assets
    .find(a => a.token === player);
  if (!asset) throw new Error('This player is not on your held trade roster. Refresh before retrying.');
  if (!state.give.includes(asset.token)) await builderAction({ action: 'give', value: asset.token });
  await commands.dispatch('nav.open', { node: 'trade', workspace: 'trade-desk' });
}
