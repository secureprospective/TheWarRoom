// The draft's own numbers: what each pick returned against its slot, sliced by where in the
// draft it was taken, the season after the draft, the player's NFL draft round and his age.
// Every comparison is a ratio to the slot (wins returned over what the same overall pick returned
// in the other classes), with a range, so a position's edge reads as a percent, not ±0.05 wins.

import {deliveredRatio} from './engine.js';

const Z = 1.96;

// Stretches of the rookie draft, in draft order.
export const SEGMENTS = [
  {id: 'r1a', label: '1.01–1.10', hint: 'Round 1, picks 1 to 10', test: p => p.round === 1 && p.slot <= 10},
  {id: 'r1b', label: '1.11–1.32', hint: 'Round 1, picks 11 to 32', test: p => p.round === 1 && p.slot > 10},
  {id: 'r2a', label: '2.01–2.16', hint: 'Round 2, picks 1 to 16', test: p => p.round === 2 && p.slot <= 16},
  {id: 'r2b', label: '2.17–2.32', hint: 'Round 2, picks 17 to 32', test: p => p.round === 2 && p.slot > 16},
  {id: 'r3', label: 'Round 3', hint: 'Round 3', test: p => p.round === 3},
  {id: 'r4', label: 'Round 4', hint: 'Round 4', test: p => p.round === 4},
  {id: 'r5', label: 'Round 5 on', hint: 'Round 5 and any later round', test: p => p.round >= 5},
];
export const segmentOf = p => SEGMENTS.find(s => s.test(p))?.id;

// The player's NFL draft capital, as of the spring he entered this league's draft.
export const NFL_ROUNDS = [
  {id: 'n1', label: '1st', hint: 'Taken in round 1 of the NFL draft', test: p => p.nflRound === 1},
  {id: 'n2', label: '2nd', hint: 'Taken in round 2 of the NFL draft', test: p => p.nflRound === 2},
  {id: 'n3', label: '3rd', hint: 'Taken in round 3 of the NFL draft', test: p => p.nflRound === 3},
  {id: 'n4', label: '4th–7th', hint: 'Taken in rounds 4 to 7 of the NFL draft', test: p => p.nflRound >= 4},
  {id: 'nx', label: 'None that spring', hint: 'Not an NFL pick that spring: undrafted, or drafted in an earlier year', test: p => p.nflRound == null},
];
export const nflOf = p => NFL_ROUNDS.find(n => n.test(p))?.id;

export const AGE_BANDS = [
  {id: 'a21', label: '21 or younger', test: p => p.age != null && p.age < 22},
  {id: 'a22', label: '22', test: p => p.age != null && p.age >= 22 && p.age < 23},
  {id: 'a23', label: '23 or older', test: p => p.age != null && p.age >= 23},
];

export const SEASONS_AFTER = 5;     // seasons after the draft shown in "when it pays"
const HIT = new Set(['top', 'starter']);

// The picks in view: classes in the seasons range, narrowed by the filters. judged picks have
// `seasons` finished seasons and carry their totals; the rest still count toward the seasons
// they have played.
export function draftScope(data, {from, to, seasons, segments = null, nfl = null}) {
  return data.draft.filter(p => p.year >= from && p.year <= to && p.group
    && (!segments || segments.includes(segmentOf(p))) && (!nfl || nfl.includes(nflOf(p))))
    .map(p => {
      const got = p.wins.slice(0, seasons), slot = p.slotWins.slice(0, seasons);
      const judged = p.year + seasons - 1 <= data.lastFinished && got.length === seasons && got.every(v => v != null) && slot.every(v => v != null);
      return {...p, segment: segmentOf(p), nfl: nflOf(p), judged,
        got: judged ? got.reduce((a, b) => a + b, 0) : null, expected: judged ? slot.reduce((a, b) => a + b, 0) : null};
    });
}

// Wins returned over what the slots returned, with a range: the ratio estimator the market uses
// for delivered wins, so a few big hits can't hide behind an average.
export function vsSlot(picks) {
  const r = deliveredRatio(picks.filter(p => p.judged).map(p => ({expected: p.expected, delivered: p.got})));
  return r ? {...r, lo: r.ratio - Z * r.se, hi: r.ratio + Z * r.se, tone: r.ratio - Z * r.se > 1 ? 'up' : r.ratio + Z * r.se < 1 ? 'down' : 'even'} : null;
}

// The same, season by season after the draft, from every class that has played that season.
export function bySeason(picks) {
  return Array.from({length: SEASONS_AFTER}, (_, i) => {
    const recs = picks.filter(p => p.wins[i] != null && p.slotWins[i] != null).map(p => ({expected: p.slotWins[i], delivered: p.wins[i]}));
    const r = deliveredRatio(recs);
    return {season: i + 1, n: recs.length, ...(r ? {ratio: r.ratio, lo: r.ratio - Z * r.se, hi: r.ratio + Z * r.se,
      tone: r.ratio - Z * r.se > 1 ? 'up' : r.ratio + Z * r.se < 1 ? 'down' : 'even'} : {ratio: null})};
  });
}

// The draft's value curve: what each overall pick returns, smoothed and never rising, from
// every judged pick in view. playsLike turns a return back into the pick that usually gives it.
export function slotCurve(picks) {
  const judged = picks.filter(p => p.judged);
  const max = Math.max(1, ...judged.map(p => p.overall));
  const at = Array.from({length: max + 1}, () => ({s: 0, n: 0}));
  for (const p of judged) { at[p.overall].s += p.expected; at[p.overall].n += 1; }
  const curve = new Array(max + 1).fill(null);
  for (let o = 1; o <= max; o++) {
    let s = 0, n = 0;
    for (let k = Math.max(1, o - 4); k <= Math.min(max, o + 4); k++) { s += at[k].s; n += at[k].n; }
    curve[o] = n ? s / n : null;
  }
  for (let o = 2; o <= max; o++) if (curve[o] == null || (curve[o - 1] != null && curve[o] > curve[o - 1])) curve[o] = curve[o - 1];
  return curve;
}
export function playsLike(curve, value) {
  if (value == null) return null;
  for (let o = 1; o < curve.length; o++) if (curve[o] != null && curve[o] <= value) return o;
  return curve.length - 1;
}

// How often a pick becomes a starter or better, against picks from the same stretches of the draft.
export function hitRate(picks, all) {
  const base = new Map(SEGMENTS.map(s => {
    const xs = all.filter(p => p.judged && p.segment === s.id);
    return [s.id, xs.length ? xs.filter(p => HIT.has(p.best)).length / xs.length : null];
  }));
  const mine = picks.filter(p => p.judged && base.get(p.segment) != null);
  if (mine.length < 8) return null;
  const hits = mine.filter(p => HIT.has(p.best)).length;
  const expected = mine.reduce((s, p) => s + base.get(p.segment), 0);
  const v = mine.reduce((s, p) => s + base.get(p.segment) * (1 - base.get(p.segment)), 0);
  const sd = Math.sqrt(v);
  return {n: mine.length, rate: hits / mine.length, baseline: expected / mine.length,
    tone: hits - expected > Z * sd ? 'up' : expected - hits > Z * sd ? 'down' : 'even'};
}

// Mark the best and worst of a row of cells, only where it beats every other beyond chance.
function markExtremes(cells) {
  const known = cells.filter(c => c.ratio != null && c.hi != null);
  if (known.length < 2) return cells;
  const se = c => (c.hi - c.lo) / (2 * Z);
  const top = known.reduce((a, b) => (b.ratio > a.ratio ? b : a)), bottom = known.reduce((a, b) => (b.ratio < a.ratio ? b : a));
  if (known.every(o => o === top || top.ratio - o.ratio > Z * Math.hypot(se(top), se(o)))) top.best = true;
  if (known.every(o => o === bottom || o.ratio - bottom.ratio > Z * Math.hypot(se(bottom), se(o)))) bottom.worst = true;
  return cells;
}

// Everything the Lab can say about drafting one position, in the picks in view.
export function positionDraft(picks, group) {
  const mine = picks.filter(p => p.group === group);
  const curve = slotCurve(picks);
  const cell = (xs, extra) => {
    const r = vsSlot(xs);
    const judged = xs.filter(p => p.judged);
    const at = judged.length ? judged.reduce((s, p) => s + p.overall, 0) / judged.length : null;
    const like = r && at ? playsLike(curve, curve[Math.round(at)] * r.ratio) : null;
    return {...extra, n: judged.length, picks: xs, ...(r ? {ratio: r.ratio, lo: r.lo, hi: r.hi, tone: r.tone} : {ratio: null}),
      at: at && Math.round(at), like};
  };
  return {group, n: mine.filter(p => p.judged).length, all: mine, overall: vsSlot(mine), hits: hitRate(mine, picks),
    kept: (() => { const k = mine.filter(p => p.keptYear2 != null); return k.length ? k.filter(p => p.keptYear2).length / k.length : null; })(),
    where: markExtremes(SEGMENTS.map(s => cell(mine.filter(p => p.segment === s.id), {segment: s.id}))),
    when: markExtremes(bySeason(mine).map(c => ({...c, picks: mine.filter(p => p.wins[c.season - 1] != null)}))),
    nfl: markExtremes(NFL_ROUNDS.map(n => cell(mine.filter(p => p.nfl === n.id), {nfl: n.id}))),
    age: markExtremes(AGE_BANDS.map(a => cell(mine.filter(p => a.test(p)), {age: a.id}))),
    curve};
}

// The league map: every position against its slots, by season after the draft and by stretch.
export function draftMap(picks, groups) {
  return groups.map(g => {
    const mine = picks.filter(p => p.group === g.id);
    return {group: g.id, overall: vsSlot(mine), n: mine.filter(p => p.judged).length,
      when: bySeason(mine), where: SEGMENTS.map(s => ({segment: s.id, ...(vsSlot(mine.filter(p => p.segment === s.id)) || {ratio: null}), n: mine.filter(p => p.judged && p.segment === s.id).length}))};
  });
}
