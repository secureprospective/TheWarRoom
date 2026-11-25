import type { Race } from './contract';

export type RaceReader = {
  read(week: number, team: string, mine: string, aggMode?: string): Promise<Race>;
  refresh(force: boolean): Promise<void>;
};

const unavailable = async (): Promise<never> => {
  throw new Error('League race reports require the desktop app; no browser fixture is substituted.');
};

export const raceAPI: RaceReader = {
  read: unavailable,
  refresh: unavailable,
};

export const subscribeRace = (_changed: () => void) => () => {};
