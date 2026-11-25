import { type Research, type Scope, type PlayerReading } from './contract';
import type { HostedPort } from '../commissioner/hostedRoom';
import { hostedPlayerReadingUnavailable, hostedWorkflow, type HostedPlayerReadingUnavailable } from
  '../control/nativeApi';

export type ResearchReader = {
  read(scope: Scope): Promise<Research>; refresh(force: boolean): Promise<void>;
  status?(): Promise<string>; player?(id: string): Promise<PlayerReading>;
  news(mine: string): Promise<void>; open(id: string, article: string): Promise<void>;
};
const unavailable = async (): Promise<never> => {
  throw new Error('Player research requires the desktop app; no browser fixture is substituted.');
};
export const researchAPI: ResearchReader = {
  player: unavailable, status: unavailable, read: unavailable, refresh: unavailable,
  news: unavailable, open: unavailable,
};

export const subscribeResearch = (_changed: () => void) => () => {};
export const readPlayerUsage = async (
  id: string, league?: string, port: HostedPort = hostedWorkflow,
): Promise<unknown | HostedPlayerReadingUnavailable> => {
  if (!league) return unavailable();
  const { data } = await port({ command: 'player.usage.read', mode: 'read',
    arguments: { league, player: id } });
  return hostedPlayerReadingUnavailable(data) ?? data;
};
