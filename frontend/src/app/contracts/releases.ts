import type { Reading } from './contract';
import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { targetReleasePlans } from './api';
import { array, object, text } from '../data/parseValues';
import { selectProvider } from '../data/provider';
import { parseRailReceipt } from '../shell/railReceipt';
import type { RailReceipt } from '../shell/EnvelopeRail';
import { parseReading } from './provider';

export type ReleaseAction = {
  mine: string; action: 'read' | 'draft' | 'handoff' | 'rebase' | 'withdraw'; move?: string; id?: string;
  confirmed?: boolean;
};
export type ReleasePlan = {
  id: string; player: string; label: string; charge: string; confirmation: string;
  post: string; channel: string; receipt: RailReceipt; canHandOff: boolean; canRebase: boolean;
  cut: boolean; rebaseLabel: string; withdrawLabel: string;
};
function flag(v: unknown, p: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${p}: expected boolean`);
  return v;
}
export function parseReleases(v: unknown): Reading<ReleasePlan[]> {
  return parseReading(v, (v, p) => array(v, p, (v, p) => {
    const r = object(v, p, ['id', 'player', 'label', 'charge', 'confirmation', 'post', 'channel',
      'receipt', 'canHandOff', 'canRebase', 'cut', 'rebaseLabel', 'withdrawLabel']);
    return {
      id: text(r.id, p), player: text(r.player, p), label: text(r.label, p), charge: text(r.charge, p),
      confirmation: text(r.confirmation, p), post: text(r.post, p), channel: text(r.channel, p),
      receipt: parseRailReceipt(r.receipt, p).rail, canHandOff: flag(r.canHandOff, p),
      canRebase: flag(r.canRebase, p), cut: flag(r.cut, p),
      rebaseLabel: text(r.rebaseLabel, p), withdrawLabel: text(r.withdrawLabel, p),
    };
  }));
}
export async function releaseAction(action: ReleaseAction): Promise<Reading<ReleasePlan[]>> {
  if (selectProvider().kind !== 'live') throw new Error('Release plans need the desktop app');
  const result = parseReleases(await targetReleasePlans({
    ...action, move: action.move ?? '', id: action.id ?? '', confirmed: action.confirmed ?? false,
  }));
  if (result.status !== 'ok') throw new Error(result.absenceReason ?? 'Release plans failed');
  if (action.action !== 'read') for (const listener of listeners) listener();
  return result;
}

const listeners = new Set<() => void>();
export function onReleaseChange(listener: () => void) {
 listeners.add(listener); return () => { listeners.delete(listener); };
}

const confirmation = createStore<{ token: string }>(() => ({ token: '' }));
export function confirmRelease(token: string) { confirmation.setState({ token }); }
export function useReleaseConfirmation() { return useStore(confirmation, state => state.token); }

export async function copyReleaseText(text: string): Promise<boolean> {
  const runtime = (window as unknown as {
    runtime: { ClipboardSetText(text: string): Promise<boolean> };
  }).runtime;
  return runtime.ClipboardSetText(text);
}

