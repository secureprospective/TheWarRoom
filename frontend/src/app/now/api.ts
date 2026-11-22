import type * as Bindings from '../../../wailsjs/go/now/API';

// The only file that touches the now bindings; callers parse the unknown wire values.
const pulse = 'League Pulse needs the desktop app; no browser fixture is substituted.';
const live = 'Live games need the desktop app; no fixture is substituted.';

function bindings(missing: string): typeof Bindings {
  const api = typeof window === 'undefined' ? undefined
    : (window as unknown as { go?: { now?: { API?: typeof Bindings } } }).go?.now?.API;
  if (!api) throw new Error(missing);
  return api;
}

export async function targetNow(week: number, reportWeek: number, mine: string): Promise<unknown> {
  return bindings(pulse).TargetNow(week, reportWeek, mine);
}
export async function targetNowRefresh(week: number, force: boolean): Promise<void> {
  return bindings(pulse).TargetNowRefresh(week, force);
}
export async function targetLiveGame(open: boolean, follow: string, mine: string): Promise<unknown> {
  return bindings(live).TargetLiveGame(open, follow, mine);
}
