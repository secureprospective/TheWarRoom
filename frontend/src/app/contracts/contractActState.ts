import { create } from 'zustand';
import { commands } from '../commands/registry';
import { targetContractActs } from './api';
import { array, object, text, integer } from '../data/parseValues';
import { parseRailReceipt } from '../shell/railReceipt';
import type { RailReceipt } from '../shell/EnvelopeRail';
import { parseReading } from './provider';
import type { Reading } from './contract';
import { selectProvider } from '../data/provider';

export type ContractActRequest = {
  mine: string; player?: string;
      action: 'read' | 'options' | 'preview' | 'draft' | 'handoff' | 'posted' | 'withdraw' | 'rebase';
  intent?: string; id?: string; level?: number; years?: number; source?: number; destination?: number;
      move?: string;
};
export type ContractActPlan = {
  id: string; intent: string; act: string; levels: { value: number; label: string }[]; years: number[];
  player: string; subject: string; label: string; price: string; post: string;
  how: string;
  teamFigures: { label: string; before: string; after: string; change: string }[];
  reasons: string[]; figures: { season: number; before: string; after: string; change: string }[];
  receipt?: RailReceipt; canHandOff: boolean; canPost: boolean; canWithdraw: boolean; canRebase: boolean;
};
function flag(value: unknown): boolean {
  if (typeof value !== 'boolean') throw new Error('Contract Act: expected boolean');
  return value;
}
export function parseContractActs(value: unknown): Reading<ContractActPlan[]> {
  return parseReading(value, (v, p) => array(v, p, (v, p) => {
    const r = object(v, p, ['id', 'intent', 'act', 'levels', 'years', 'teamFigures', 'player', 'subject',
      'label', 'price', 'post', 'how', 'reasons', 'figures', 'receipt',
      'canHandOff', 'canPost', 'canWithdraw', 'canRebase']);
    const id = text(r.id, p);
    return { id, intent: text(r.intent, p), act: text(r.act, p),
      levels: array(r.levels, p, (v, p) => {
        const l = object(v, p, ['value', 'label']);
        return { value: integer(l.value, p), label: text(l.label, p) };
      }), years: array(r.years, p, integer), teamFigures: array(r.teamFigures ?? [], p, (v, p) => {
        const f = object(v, p, ['label', 'before', 'after', 'change']);
        return { label: text(f.label, p), before: text(f.before, p), after: text(f.after, p),
          change: text(f.change, p) };
      }), player: text(r.player, p), subject: text(r.subject, p), label: text(r.label, p),
      price: text(r.price, p),
      post: text(r.post, p), how: text(r.how, p), reasons: array(r.reasons, p, text),
      figures: array(r.figures, p, (v, p) => {
        const f = object(v, p, ['season', 'before', 'after', 'change']);
        return { season: integer(f.season, p), before: text(f.before, p), after: text(f.after, p),
      change: text(f.change, p) };
      }), receipt: id ? parseRailReceipt(r.receipt, p).rail : undefined,
      canHandOff: flag(r.canHandOff), canPost: flag(r.canPost), canWithdraw: flag(r.canWithdraw),
      canRebase: flag(r.canRebase),
    };
  }));
}
const listeners = new Set<() => void>();
export function onContractActsChange(listener: () => void) {
  listeners.add(listener); return () => { listeners.delete(listener); };
}
export async function contractActAction(request: ContractActRequest) {
  if (selectProvider().kind !== 'live') throw new Error('Contract plans need the desktop app');
  const raw = { player: '', intent: '', id: '', level: 1, years: 1, source: 0, destination: 0, move: '',
      ...request };
  const reading = parseContractActs(await targetContractActs(raw));
  if (reading.status !== 'ok') throw new Error(reading.absenceReason ?? 'Contract plans unavailable');
  if (!['read', 'options', 'preview'].includes(request.action)) for (const listener of listeners) listener();
  return reading;
}
export async function runContractAct(request: ContractActRequest) {
  if (request.action === 'handoff') {
    const reading = await contractActAction({ mine: request.mine, action: 'read' });
    const plan = reading.value?.find(plan => plan.id === request.id);
    if (!plan?.canHandOff) throw new Error('Contract plan is no longer ready');
    const { copyReleaseText } = await import('./releases');
    if (!await copyReleaseText(plan.post)) throw new Error('Clipboard unavailable; nothing handed off');
  }
  await contractActAction(request);
}

export const useContractMove = create<{
  mine: string; player: string; label: string;
  kept?: ContractActPlan; keptTerms: string; saving: boolean; error: string; reveal: string;
}>(() => ({ mine: '', player: '', label: '', keptTerms: '', saving: false, error: '', reveal: '' }));
export function termsKey({ intent, level, years, source, destination, move }: ContractActRequest) {
  return JSON.stringify([intent, level, years, source, destination, move]);
}
export function keepLabel(request: ContractActRequest, state = useContractMove.getState()) {
  if (state.saving) return 'Keeping…';
  return state.kept && state.keptTerms === termsKey(request) ? 'Kept' : 'Keep plan';
}
export function openContractMove(mine: string, player: string) {
  useContractMove.setState({ mine, player, label: '', kept: undefined, keptTerms: '', saving: false,
    error: '' });
  commands.dispatch('nav.open', { node: 'hq', workspace: 'contracts-and-cap' });
}
export function revealContractPlan(id: string) {
  useContractMove.setState({ reveal: id });
  commands.dispatch('nav.open', { node: 'hq', workspace: 'my-moves' });
}
export async function keepContractMove(request: ContractActRequest, action = contractActAction) {
  if (useContractMove.getState().saving) return;
  useContractMove.setState({ saving: true, error: '' });
  try {
    const before = await action({ mine: request.mine, action: 'read' });
    const reading = await action({ ...request, action: 'draft' });
    const kept = reading.value?.find(p => !before.value?.some(old => old.id === p.id));
    if (kept) useContractMove.setState({ kept, keptTerms: termsKey(request) });
  } catch (cause) {
    useContractMove.setState({ error: cause instanceof Error ? cause.message : String(cause) });
  }
  finally { useContractMove.setState({ saving: false }); }
}


export function reconcileContractMove(mine: string, plans: ContractActPlan[]) {
  const current = useContractMove.getState();
  if (current.mine !== mine || !current.kept) return;
  const live = plans.find(plan => plan.id === current.kept?.id);
  const terminal = !live || ['withdrawn', 'landed', 'failed', 'awarded_off_app']
    .includes(live.receipt?.state ?? '');
  useContractMove.setState(terminal ? { kept: undefined, keptTerms: '', error: '' } : { kept: live });
}
