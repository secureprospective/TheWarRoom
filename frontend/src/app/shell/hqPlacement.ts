import type { HQSlot } from './HQBoard';
import type { HQSettings } from './hqModel';

export type PlacementPlayer = { id: string; position?: string };
export function fitsSlot(player: PlacementPlayer | undefined, slot: HQSlot, settings: HQSettings): boolean {
  if (!player?.position) return false;
  const position = player.position === 'PK' ? 'K' : player.position;
  const rule = settings.positions.find(r => r.code === position);
  if (rule?.max === 0) return false;
  if (slot.code === position) return true;
  const defense = ['DT', 'DE', 'LB', 'CB', 'S', 'DL', 'DB'].includes(position);
  return slot.code === 'D·FLX' ? defense : slot.code === 'FLEX' ? !defense : false;
}

// Slot choice is presentation; the existing lineup command still checks league legality.
export function placePlayer(slots: HQSlot[], players: PlacementPlayer[], settings: HQSettings,
  values: Map<string, number>, id: string, target?: number, toBench = false): HQSlot[] {
  const next = slots.map(s => ({ ...s }));
  const player = (id?: string) => players.find(p => p.id === id);
  const score = (id?: string) => id ? values.get(id) : -Infinity;
  const fits = (id: string, i: number) => fitsSlot(player(id), next[i], settings);
  const weakest = (id: string, skip = -1) => next.reduce((best, s, i) => {
    if (i === skip || !fits(id, i)) return best;
    if (!s.id) return best >= 0 && !next[best].id ? best : i;
    if (score(s.id) === undefined) return best;
    return best < 0 || score(s.id)! < score(next[best].id)! ? i : best;
  }, -1);
  const from = next.findIndex(s => s.id === id);
  if (toBench) {
    if (from < 0) return slots;
    delete next[from].id;
    const candidate = players.filter(p => p.id !== id && !next.some(s => s.id === p.id) && fits(p.id, from)
      && (score(p.id) ?? 0) > 0).sort((a, b) => score(b.id)! - score(a.id)!)[0];
    if (candidate) next[from].id = candidate.id;
    return next;
  }
  const at = target ?? weakest(id);
  if (at < 0 || !next[at] || !fits(id, at)) return slots;
  const out = next[at].id;
  if (from >= 0) {
    if (out && !fits(out, from)) return slots;
    next[from].id = out; next[at].id = id;
    return next;
  }
  next[at].id = id;
  function cascade(displaced: string, skip: number, depth: number) {
    if (depth > 2) return;
    const i = weakest(displaced, skip);
    const incoming = score(displaced), incumbent = i < 0 ? undefined : score(next[i].id);
    if (i < 0 || incoming === undefined || incumbent === undefined || incumbent >= incoming) return;
    const out = next[i].id; next[i].id = displaced;
    if (out) cascade(out, i, depth + 1);
  }
  if (out) cascade(out, at, 1);
  return next;
}
