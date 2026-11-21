import { useEffect, useRef, useState } from 'react';
import type { Game } from '../now/liveContract';
import { espnTeam } from '../shell/pulseModel';

export type ValueChange = { changed: false } | { changed: true; from: number; to: number };

const EPSILON = 1e-6;
const NON_OFFENSE_SLOTS = new Set([
  'DEF', 'DST', 'K', 'PK', 'DL', 'DT', 'DE', 'LB', 'CB', 'S', 'DB',
]);

function roundTo(value: number, decimals: number): number {
  const factor = 10 ** decimals;
  return Math.round(value * factor) / factor;
}

/** detectChange reports whether a held numeric value genuinely changed within the same context
 *  (same matchup, player, or game). Never fires on first read, missing values, or context switches. */
export function detectChange(
  prev: number | undefined,
  next: number | undefined,
  sameContext = true,
): ValueChange {
  if (!sameContext || prev === undefined || next === undefined) return { changed: false };
  if (!Number.isFinite(prev) || !Number.isFinite(next)) return { changed: false };
  if (Math.abs(next - prev) < EPSILON) return { changed: false };
  return { changed: true, from: prev, to: next };
}

/** countUpSteps builds eased intermediate values for score and win-chance transitions.
 *  Under reduced motion, returns only the final target value for an instant update. */
export function countUpSteps(
  from: number,
  to: number,
  opts: { reduced: boolean; decimals?: number; steps?: number },
): number[] {
  const decimals = opts.decimals ?? 1;
  const target = roundTo(to, decimals);
  if (opts.reduced || Math.abs(to - from) < EPSILON) return [target];
  const frames = Math.max(2, opts.steps ?? 10);
  const out: number[] = [];
  for (let i = 1; i <= frames; i++) {
    const t = i / frames;
    const eased = 1 - Math.pow(1 - t, 3);
    const val = i === frames ? target : roundTo(from + (to - from) * eased, decimals);
    if (val !== roundTo(from, decimals) && out[out.length - 1] !== val) out.push(val);
  }
  if (out[out.length - 1] !== target) out.push(target);
  return out;
}

/** scoreRollSteps builds the mock's point-by-point NFL score odometer sequence (130ms per point).
 *  Caps large reconnect jumps at 14 steps; snaps immediately under reduced motion or decreases. */
export function scoreRollSteps(from: number, to: number, reduced: boolean): number[] {
  const start = Math.round(from);
  const end = Math.round(to);
  if (reduced || end <= start) return [end];
  const diff = end - start;
  if (diff <= 14) return Array.from({ length: diff }, (_, i) => start + i + 1);
  return countUpSteps(start, end, { reduced: false, decimals: 0, steps: 14 });
}

/** onFieldFromPossession approximates "on field now" from NFL game possession only (not snap
 *  participation): true when the starter's NFL game is live and their club holds offensive
 *  possession, excluding kicking and defensive slots. */
export function onFieldFromPossession(
  starter: { slot: string; team?: string } | undefined,
  game?: Game,
): boolean {
  const clubCode = espnTeam(starter?.team);
  if (!starter || !clubCode || !game || game.status !== 'in' || !game.state?.possession) return false;
  const club = game.home.abbreviation === clubCode ? game.home
    : game.away.abbreviation === clubCode ? game.away : undefined;
  if (!club) return false;
  return game.state.possession === club.id && !NON_OFFENSE_SLOTS.has(starter.slot);
}

/** useAnimatedValue holds the previous reading in a ref and animates held numeric changes in steps
 *  with a brief highlight (.chg), snapping immediately under reduced motion or context changes. */
export function useAnimatedValue(
  value: number | undefined,
  contextKey: string,
  opts: { reduced: boolean; decimals?: number; stepMs?: number; flashMs?: number },
): { display: number | undefined; changing: boolean; flashKey: number } {
  const decimals = opts.decimals ?? 1;
  const stepMs = opts.stepMs ?? 40;
  const flashMs = opts.flashMs ?? 1200;
  const prev = useRef<{ key: string; value: number | undefined }>({ key: contextKey, value });
  const [display, setDisplay] = useState<number | undefined>(value);
  const [changing, setChanging] = useState(false);
  const [flashKey, setFlashKey] = useState(0);

  useEffect(() => {
    const sameContext = prev.current.key === contextKey;
    const diff = detectChange(prev.current.value, value, sameContext);
    prev.current = { key: contextKey, value };
    if (!diff.changed) {
      setDisplay(value);
      setChanging(false);
      return;
    }
    if (opts.reduced) {
      setDisplay(diff.to);
      setChanging(false);
      return;
    }
    const steps = countUpSteps(diff.from, diff.to, { reduced: false, decimals });
    setDisplay(steps[0]);
    setChanging(true);
    setFlashKey(k => k + 1);
    const timers: number[] = [];
    for (let i = 1; i < steps.length; i++) {
      timers.push(window.setTimeout(() => setDisplay(steps[i]), i * stepMs));
    }
    timers.push(window.setTimeout(() => setChanging(false), flashMs));
    return () => timers.forEach(id => window.clearTimeout(id));
  }, [value, contextKey, opts.reduced, decimals, stepMs, flashMs]);

  return { display, changing, flashKey };
}

/** useScoreRoll animates an NFL team's score rolling up point-by-point like the mock's odometer
 *  (130ms per step, team-coloured .scoring hold for 5.2s), snapping under reduced motion. */
export function useScoreRoll(
  score: number,
  gameId: string,
  reduced: boolean,
): { display: number; scoring: boolean; rollSeq: number } {
  const prev = useRef<{ id: string; score: number }>({ id: gameId, score });
  const [display, setDisplay] = useState(score);
  const [scoring, setScoring] = useState(false);
  const [rollSeq, setRollSeq] = useState(0);

  useEffect(() => {
    const sameGame = prev.current.id === gameId;
    const diff = detectChange(prev.current.score, score, sameGame);
    prev.current = { id: gameId, score };
    if (!diff.changed) {
      setDisplay(score);
      if (!sameGame) setScoring(false);
      return;
    }
    if (reduced || diff.to <= diff.from) {
      setDisplay(diff.to);
      setScoring(false);
      return;
    }
    const steps = scoreRollSteps(diff.from, diff.to, false);
    setDisplay(steps[0]);
    setScoring(true);
    setRollSeq(n => n + 1);
    const timers: number[] = [];
    for (let i = 1; i < steps.length; i++) {
      timers.push(window.setTimeout(() => {
        setDisplay(steps[i]);
        setRollSeq(n => n + 1);
      }, i * 130));
    }
    timers.push(window.setTimeout(() => setScoring(false), (steps.length - 1) * 130 + 5200));
    return () => timers.forEach(id => window.clearTimeout(id));
  }, [score, gameId, reduced]);

  return { display, scoring, rollSeq };
}
