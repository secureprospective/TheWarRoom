import { parseAssetCell } from '../assets/contract';
import { create } from 'zustand';
import { array, object, text } from '../data/parseValues';
import { parseReceipt } from '../data/parseEnvelope';
import { parseReading } from '../contracts/provider';
import type { Reading } from '../contracts/contract';
import type { Receipt, TradeAsset } from '../data/contract';

export type Figure = { label: string; before: string; after: string; change: string };
export type BuilderPlan = {
  id: string; post: string; reasons: string[]; resets: { franchise: string; label: string }[];
  receipt?: Receipt;
  teams: { team: string; figures: Figure[]; roster: TradeAsset[] }[];
};
type BuilderView = {
  teams: { id: string; name: string; assets: TradeAsset[] }[];
  plans: BuilderPlan[]; preview?: BuilderPlan;
};
export function parseFigures(v: unknown, p: string): Figure[] {
  return array(v, p, (v, p) => {
    const r = object(v, p, ['label', 'before', 'after', 'change']);
    return { label: text(r.label, p), before: text(r.before, p), after: text(r.after, p),
      change: text(r.change, p) };
  });
}
export function parseAssets(v: unknown, p: string): TradeAsset[] {
  return array(v, p, (v, p) => {
    const r = object(v, p, ['token', 'name', 'position', 'team', 'contract', 'now', 'dynasty']);
    return { token: text(r.token, p), name: text(r.name, p),
      ...(r.team === undefined ? {} : { team: text(r.team, p) }),
      ...(r.contract === undefined ? {} : { contract: text(r.contract, p) }),
      ...(r.now === undefined ? {} : { now: parseAssetCell(r.now, p) }),
      ...(r.dynasty === undefined ? {} : { dynasty: parseAssetCell(r.dynasty, p) }),
      ...(r.position === undefined ? {} : { position: text(r.position, p) as TradeAsset['position'] }) };
  });
}
export function parseBuilder(v: unknown): Reading<BuilderView> {
  return parseReading(v, (v, p) => {
    const r = object(v, p, ['teams', 'plans', 'preview']);
    return { teams: array(r.teams, p, (v, p) => {
      const t = object(v, p, ['id', 'name', 'assets']);
      return { id: text(t.id, p), name: text(t.name, p), assets: parseAssets(t.assets, p) };
    }), plans: array(r.plans, p, parsePlan),
    ...(r.preview === undefined ? {} : { preview: parsePlan(r.preview, p) }) };
  });
}
function parsePlan(v: unknown, p: string): BuilderPlan {
  const r = object(v, p, ['id', 'post', 'reasons', 'resets', 'teams', 'receipt']);
  return { id: text(r.id, p), post: text(r.post, p), reasons: array(r.reasons, p, text),
    resets: r.resets === undefined ? [] : array(r.resets, p, (v, p) => {
      const x = object(v, p, ['franchise', 'label']);
      return { franchise: text(x.franchise, p), label: text(x.label, p) };
    }),
    teams: array(r.teams, p, (v, p) => {
      const t = object(v, p, ['team', 'figures', 'roster']);
      return { team: text(t.team, p), figures: parseFigures(t.figures, p), roster: parseAssets(t.roster, p) };
    }), ...(r.receipt === undefined ? {} : { receipt: parseReceipt(r.receipt) }) };
}
export type BuilderAction = {
  action: 'reject' | 'revoke' | 'handoff' | 'remove'; mine: string; id: string; post?: string;
};
export const useBuilder = create<{ busy: boolean; error: string }>(() => ({ busy: false, error: '' }));

export async function builderAction(_action: BuilderAction) {
  const s = useBuilder.getState();
  if (s.busy) return;
  useBuilder.setState({ busy: true, error: '' });
  try {
    throw new Error('Trade builder needs the desktop app');
  } catch (cause) { useBuilder.setState({ error: String(cause) }); }
  finally { useBuilder.setState({ busy: false }); }
}
