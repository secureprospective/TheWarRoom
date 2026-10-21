import { parseBoard, report } from '../reports/contract';
import { parseMarket, unavailable } from '../market/provider';
import type { SurfaceReading } from '../market/contract';
import type { DraftScope } from '../market/draftState';
import type { ReportProvider } from './provider';
import type * as Bindings from '../../../wailsjs/go/draft/API';

function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Draft reports require the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { draft: { API: typeof Bindings } } }).go.draft.API;
}

export const provider: ReportProvider = {
  read: async () => parseBoard(await bindings().TargetDraftReports()),
  refresh: async force => bindings().TargetDraftReportsRefresh(force),
  openMFL: async () => bindings().TargetOpenMFLDraftRoom(),
};

export async function readNFLDraft() {
  return report(await bindings().TargetNFLDraft(), 'NFL draft');
}

export async function readDraftLab(scope: DraftScope): Promise<SurfaceReading> {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go) {
    return { ...unavailable(
      'Historical Draft needs the desktop app; no browser fixtures are substituted', 'not_wired',
    ), threshold: 40 };
  }
  try {
    return parseMarket(await bindings().TargetDraftLab(scope));
  } catch (cause) {
    const reason = `Draft read failed: ${String(cause)}. Retry the held archive reading.`;
    return { ...unavailable(reason), threshold: 40 };
  }
}

export function subscribeDraft(changed: () => void): () => void {
  let disposed = false;
  let stop: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(m => {
    if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
      stop = m.EventsOn('target:season', changed);
  });
  return () => { disposed = true; stop?.(); };
}
