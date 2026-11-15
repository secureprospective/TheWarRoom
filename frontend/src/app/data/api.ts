import type { platform } from '../../../wailsjs/go/models';
import {
  TargetClock, TargetDraftIR, TargetDraftTaxi, TargetHandOff, TargetLeagues, TargetMoves, TargetSnapshot,
  TargetWithdrawRoster,
} from '../../../wailsjs/go/platform/API';

// The only file that touches the platform bindings: the snapshot, the league list, the clock,
// the move rail and
// player photos. Their callers keep their own desktop checks.
export const targetSnapshot = TargetSnapshot;
export const targetLeagues = TargetLeagues;
export const targetClock = TargetClock;
export const targetDraftIR = TargetDraftIR;
export const targetDraftTaxi = TargetDraftTaxi;
export const targetMoves = TargetMoves;
export const targetHandOff = TargetHandOff;
export const targetWithdrawRoster = TargetWithdrawRoster;

export type PhotoReader = (id: string) => Promise<platform.PhotoView>;
type GoWindow = { go?: { platform?: { API?: { TargetPhoto?: PhotoReader } } } };

// photoReader is Go's photo verdict, or none outside the desktop app, so no request is made.
export function photoReader(): PhotoReader | undefined {
  if (typeof window === 'undefined') return undefined;
  return (window as unknown as GoWindow).go?.platform?.API?.TargetPhoto;
}

export { TargetInjuries as targetInjuries } from '../../../wailsjs/go/platform/API';
