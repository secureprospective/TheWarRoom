import { create } from 'zustand';
import { TargetTradeBuilder } from '../../../wailsjs/go/main/App';
import type { main } from '../../../wailsjs/go/models';
import { selectProvider } from '../data/provider';
import { array, object, text } from '../data/parseValues';
import { parseReceipt } from '../data/parseEnvelope';
import { parseReading } from '../contracts/provider';
import type { Reading } from '../contracts/contract';
import type { Receipt, TradeAsset } from '../data/contract';
import { commands } from '../commands/registry';

export type Figure = { label: string; before: string; after: string; change: string };
export type BuilderPlan = {
  id: string; post: string; reasons: string[]; resets: { franchise: string; label: string }[];
  receipt?: Receipt;
  teams: { team: string; figures: Figure[]; roster: TradeAsset[] }[];
};
export type BuilderView = {
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
    const r = object(v, p, ['token', 'name', 'position']);
    return { token: text(r.token, p), name: text(r.name, p),
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
export type BuilderAction =
  | { action: 'read' | 'preview' | 'save'; mine: string }
  | { action: 'other' | 'rationale'; value: string }
  | { action: 'give' | 'get'; value: string }
  | { action: 'reject' | 'revoke' | 'handoff'; mine: string; id: string; post?: string };
export const useBuilder = create<{
  mine: string; other: string; give: string[]; get: string[]; rationale: string;
  reading?: Reading<BuilderView>; busy: boolean; error: string;
}>(() => ({ mine: '', other: '', give: [], get: [], rationale: '', busy: false, error: '' }));
export async function builderAction(action: BuilderAction) {
  const s = useBuilder.getState();
  if ('value' in action) {
    useBuilder.setState({ reading: s.reading && { ...s.reading,
      value: s.reading.value && { ...s.reading.value, preview: undefined } } });
    if (action.action === 'other') {
      useBuilder.setState({ other: action.value, get: [], reading: s.reading &&
        { ...s.reading, value: s.reading.value && { ...s.reading.value, preview: undefined } } });
    } else if (action.action === 'rationale') useBuilder.setState({ rationale: action.value });
    else {
      const selected = s[action.action];
      useBuilder.setState({ [action.action]: selected.includes(action.value)
        ? selected.filter(v => v !== action.value) : [...selected, action.value] });
    }
    return;
  }
  if (s.busy) return;
  useBuilder.setState({ busy: true, error: '' });
  try {
    if (selectProvider().kind !== 'live') throw new Error('Trade builder needs the desktop app');
    if (action.action === 'handoff') {
      const reading = parseBuilder(await TargetTradeBuilder({
        mine: action.mine, action: 'read', other: '', give: [], get: [], rationale: '', id: '',
      } as main.RawTradeBuilder));
      if (reading.status !== 'ok') throw new Error(reading.absenceReason ?? 'Trade plan unavailable');
      const proposal = reading.value?.plans.find(p => p.id === action.id);
      const post = proposal?.post ?? action.post;
      if (post) {
      const { ClipboardSetText } = await import('../../../wailsjs/runtime/runtime');
      if (!await ClipboardSetText(post)) throw new Error('Clipboard unavailable; nothing opened');
      }
    }
    const reset = s.mine !== action.mine;
    const response = parseBuilder(await TargetTradeBuilder({
      mine: action.mine, other: reset ? '' : s.other, give: reset ? [] : s.give,
      get: reset ? [] : s.get, rationale: s.rationale, action: action.action,
      id: 'id' in action ? action.id : '',
    } as main.RawTradeBuilder));
    if (response.status !== 'ok') throw new Error(response.absenceReason ?? 'Trade builder unavailable');
    useBuilder.setState({ reading: response, mine: action.mine,
      ...(reset ? { other: '', give: [], get: [], rationale: '' } : {}) });
    if (action.action !== 'preview' && action.action !== 'read') await commands.loadMoves(action.mine);
  } catch (cause) { useBuilder.setState({ error: String(cause) }); }
  finally { useBuilder.setState({ busy: false }); }
}
