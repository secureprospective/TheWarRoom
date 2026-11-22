import { commands } from '../commands/registry';
import { targetWhatIf } from './api';
import { array, object, text, choice } from '../data/parseValues';
import { selectProvider } from '../data/provider';
import { parseReading } from './provider';
import type { Reading, Amount } from './contract';

export type WhatIfAction = {
  mine: string; action: 'read' | 'cut' | 'buyout' | 'undo' | 'reset'; player?: string; move?: string;
};
export type WhatIf = {
  stale: boolean; banner: string; empty: string; notice: string;
  moves: { id: string; player: string; tag: string; status: string; label: string; charge: Amount;
    explanation: string; plan?: string }[];
  figures: { label: string; before: string; after: string; change: string }[];
  choices: {
    player: string; cutLabel: string; buyoutLabel: string; buyoutAllowed: boolean; reason: string;
  }[];
};
export interface WhatIfProvider {
  query(mine: string): Promise<Reading<WhatIf>>;
  change(action: WhatIfAction): Promise<void>;
  onChange(listener: () => void): () => void;
}
function bool(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected boolean`);
  return value;
}
export function parseWhatIf(value: unknown): Reading<WhatIf> {
  return parseReading(value, (v, p) => {
    const r = object(v, p, ['stale', 'banner', 'empty', 'notice', 'moves', 'figures', 'choices']);
    return {
      stale: bool(r.stale, p), banner: text(r.banner, p), empty: text(r.empty, p), notice: text(r.notice, p),
      moves: array(r.moves, p, (v, p) => {
        const r = object(v, p, ['id', 'player', 'tag', 'status', 'label', 'charge', 'explanation', 'plan']);
        return { id: text(r.id, p), player: text(r.player, p), tag: text(r.tag, p),
          status: text(r.status, p), label: text(r.label, p),
          charge: parseAmount(r.charge, p), explanation: text(r.explanation, p),
          plan: r.plan === undefined ? '' : text(r.plan, p) };
      }),
      figures: array(r.figures, p, (v, p) => {
        const r = object(v, p, ['label', 'before', 'after', 'change']);
        return { label: text(r.label, p), before: text(r.before, p), after: text(r.after, p),
          change: text(r.change, p) };
      }),
      choices: array(r.choices, p, (v, p) => {
        const r = object(v, p, ['player', 'cutLabel', 'buyoutLabel', 'buyoutAllowed', 'reason']);
        return { player: text(r.player, p), cutLabel: text(r.cutLabel, p),
          buyoutLabel: text(r.buyoutLabel, p), buyoutAllowed: bool(r.buyoutAllowed, p),
          reason: text(r.reason, p) };
      }),
    };
  });
}
import { integer } from '../data/parseValues';
function parseAmount(v: unknown, p: string): Amount {
  const r = object(v, p, ['cents', 'text']);
  return { cents: integer(r.cents, p, -Number.MAX_SAFE_INTEGER), text: text(r.text, p) };
}
const listeners = new Set<() => void>();
let pending = false;
export function whatIfProvider(): WhatIfProvider {
  const selected = selectProvider();
  const call = async (action: WhatIfAction) => {
    if (selected.kind !== 'live') throw new Error('What-if needs the desktop app; no figures invented');
    choice(action.action, 'What-if action', ['read', 'cut', 'buyout', 'undo', 'reset']);
    return parseWhatIf(await targetWhatIf({
      ...action, player: action.player ?? '', move: action.move ?? '',
    }));
  };
  return {
    query: (mine) => call({ mine, action: 'read' }),
    change: async (action) => {
      if (pending) return;
      pending = true;
      try {
        const r = await call(action);
        if (r.status !== 'ok') throw new Error(r.absenceReason ?? 'What-if failed');
        if (r.value?.notice) throw new Error(r.value.notice);
      } finally {
        pending = false;
        for (const listener of listeners) listener();
      }
    },
    onChange: (listener) => {
      listeners.add(listener);
      const off = selected.onSeasonChange(listener);
      const movesOff = commands.onMovesChange(listener);
      return () => { listeners.delete(listener); off(); movesOff(); };
    },
  };
}
export async function changeWhatIf(action: WhatIfAction) {
  return whatIfProvider().change(action);
}

export function notifyWhatIf() { for (const listener of listeners) listener(); }
