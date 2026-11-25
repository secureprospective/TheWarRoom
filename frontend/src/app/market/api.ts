import { hostedPlayerReadingUnavailable, hostedWorkflow, type HostedPlayerReadingUnavailable } from
  '../control/nativeApi';

const unavailable = async (): Promise<never> => {
  throw new Error('Player values require the desktop app; no browser fixture is substituted.');
};
const reportsUnavailable = async (): Promise<never> => {
  throw new Error('Player market reports require the desktop app; no browser fixture is substituted.');
};
export const marketAPI = {
  assetBoard: unavailable,
  reports: reportsUnavailable,
  refreshReports: (_force: boolean) => reportsUnavailable(),
};
export const subscribeSeason = (_changed: () => void) => () => {};

export const readAgeArc = async (id: string, league?: string): Promise<
  unknown | HostedPlayerReadingUnavailable> => {
  if (!league) return unavailable();
  const { data } = await hostedWorkflow({ command: 'player.agearc.read', mode: 'read',
    arguments: { league, player: id } });
  return hostedPlayerReadingUnavailable(data) ?? data;
};
