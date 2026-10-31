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
    initial: (): { value?: T } => ({}), read, accept: value => ({ value }),
  }), [command, read]);
  const key = JSON.stringify(args);
  useEffect(() => {
    if (!enabled) return;
    const load = () => { void query.load(args); };
    load();
    const season = selectProvider().onSeasonChange(load);
    const moves = commands.onMovesChange(load);
    return () => { query.fence(); season(); moves(); };
    // The canonical binding arguments, not the array's render identity, own the request lifetime.
  }, [query, key, enabled]);
  const state = query.use();
  return { ...state, value: enabled && !state.error && !state.pending ? state.value : undefined };
}
export const readHQSettings = async () => report(await controlAPI.settings(), 'League settings');
export const readHQRace = (week: number, team: string, mine: string, mode: string) =>
  raceAPI.read(week, team, mine, mode);
