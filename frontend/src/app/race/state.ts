import { createReadQuery } from '../data/query';
import { commands } from '../commands/registry';
import type { Race } from './contract';
import { raceAPI, type RaceReader } from './api';
export type { RaceReader } from './api';

// Week and team are requests; Go validates them and the board echoes what it accepted.
export function createRaceState(reader: RaceReader, mine: () => string) {
  type Selection = { race?: Race; week: number; team: string | null; aggMode: string };
  const query = createReadQuery('race.TargetRace', {
    initial: (): Selection => ({ week: 0, team: null, aggMode: 'lineup' }),
    read: (...args) => reader.read(...args),
    accept: race => ({ race, week: race.week, team: race.team }),
    failure: () => 'Power rankings could not be updated. Try refreshing again.',
  });
  const load = (refresh: boolean, force = false) => query.load(() => {
    const { week, team, aggMode } = query.peek();
    return [week, team ?? mine(), mine(), aggMode];
  }, { before: refresh ? () => reader.refresh(force) : undefined });
  function choose(action: 'refresh' | 'week' | 'team' | 'aggMode', value: string) {
    if (action === 'refresh') return load(true, true);
    if (action === 'aggMode') {
      if (!['sum', 'lineup', 'topn'].includes(value)) throw new Error('League race: invalid roster count');
      query.set({ aggMode: value });
    } else if (action === 'week') {
      if (!/^\d{1,2}$/.test(value)) throw new Error('League race: invalid week');
      query.set({ week: Number(value) });
    } else {
      if (!/^\d{4}$/.test(value)) throw new Error('League race: invalid franchise');
      query.set({ team: value });
    }
    return load(false);
  }
  return { use: query.use, peek: query.peek, load, choose };
}
export const raceState = createRaceState(raceAPI, () => commands.read().franchiseId ?? '');
