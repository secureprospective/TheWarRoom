import { parseResearch, type Research, type Scope, parsePlayerReading, type PlayerReading } from './contract';
import { commands } from '../commands/registry';
import type * as Bindings from '../../../wailsjs/go/research/API';

export type ResearchReader = {
  read(scope: Scope): Promise<Research>; refresh(force: boolean): Promise<void>;
  status?(): Promise<string>; player?(id: string): Promise<PlayerReading>;
  news(mine: string): Promise<void>; open(id: string, article: string): Promise<void>;
};
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Player research requires the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { research: { API: typeof Bindings } } }).go.research.API;
}
export const researchAPI: ResearchReader = {
  player: async id => parsePlayerReading(await bindings().TargetResearchPlayer(id)),
  status: async () => bindings().TargetResearchStatus(),
  read: async scope => parseResearch(await bindings().TargetResearch(scope)),
  refresh: async force => bindings().TargetResearchRefresh(force),
  news: async mine => bindings().TargetResearchNewsRefresh(mine),
  open: async (id, article) => bindings().TargetOpenResearchArticle(id, article),
};

export function researchAvailable(): boolean {
  return typeof window !== 'undefined' &&
    !!(window as { go?: { research?: { API?: unknown } } }).go?.research?.API;
}

export function subscribeResearch(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}

export function subscribeResearchPlayer(changed: () => void): () => void {
  return commands.alerts.onSeasonChange(changed);
}
