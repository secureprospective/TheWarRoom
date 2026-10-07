// How the teams trade. Every valued trade is two team sides: who traded, where they stood that
// day, what they gave and got (players or picks, forecast wins, ages), what it was worth on the
// day and what it has delivered since. Tiers, trends, partners and team profiles are all cuts of
// the same sides, so every number on the screen adds up to the same trades.

import {meanVar, groupOf} from './engine.js';

const Z = 1.96;
const EDGE = 0.05;     // forecast wins: below this a side's player or pick flow counts as even

// Standing at the time of the trade: live rank once three games are final, else last season's.
export const TIERS = [
  {id: 'contender', label: 'Contenders', one: 'Contender', hint: 'Ranked 1 to 10 on the day of the trade', test: r => r <= 10},
  {id: 'hunt', label: 'In the hunt', one: 'In the hunt', hint: 'Ranked 11 to 22 on the day of the trade', test: r => r > 10 && r < 23},
  {id: 'rebuild', label: 'Rebuilding', one: 'Rebuilding', hint: 'Ranked 23 to 32 on the day of the trade', test: r => r >= 23},
];
export const tierOf = rank => (rank == null ? null : TIERS.find(t => t.test(rank))?.id);

// What a side did: bought now (players in, picks out), built for later (the reverse), or swapped.
export const POSTURES = [
  {id: 'now', label: 'Bought for now', hint: 'Took in more player value than it sent, and sent more pick value than it took'},
  {id: 'later', label: 'Built for later', hint: 'Sent more player value than it took, and took in more pick value than it sent'},
  {id: 'swap', label: 'Like for like', hint: 'Players for players, picks for picks, or an even mix'},
];

export function tradeSides(data) {
  if (data._sides) return data._sides;
  const out = [];
  for (const t of data.trades) {
    if (!t.valued) continue;
    for (const [fid, other, rank, oRank, sign] of [[t.a, t.b, t.aRank, t.bRank, 1], [t.b, t.a, t.bRank, t.aRank, -1]]) {
      const got = t.legs.filter(l => l.to === fid && l.xClass), gave = t.legs.filter(l => l.from === fid && l.xClass);
      const value = (legs, pick) => legs.filter(l => l.xClass.startsWith('pick') === pick).reduce((s, l) => s + l.xValue, 0);
      const ages = legs => legs.filter(l => l.age != null).map(l => l.age);
      const players = value(got, false) - value(gave, false), picks = value(got, true) - value(gave, true);
      out.push({t, fid, other, season: t.season, window: t.window, phase: t.phase,
        tier: tierOf(rank), otherTier: tierOf(oRank), rank,
        now: sign * t.aNet, real: t.aRealNet != null && t.realSeasons >= 2 ? sign * t.aRealNet : null,
        players, picks, picksIn: got.filter(l => l.xClass.startsWith('pick')).length, picksOut: gave.filter(l => l.xClass.startsWith('pick')).length,
        ageIn: ages(got), ageOut: ages(gave), got, gave,
        posture: players > EDGE && picks < -EDGE ? 'now' : players < -EDGE && picks > EDGE ? 'later' : 'swap'});
    }
  }
  data._sides = out;
  return out;
}

export function sidesInScope(data, {from, to, tiers = null, windows = null, postures = null, team = null}) {
  return tradeSides(data).filter(s => s.season >= from && s.season <= to
    && (!tiers || tiers.includes(s.tier)) && (!windows || windows.includes(s.window))
    && (!postures || postures.includes(s.posture)) && (!team || s.fid === team));
}

// A mean with its range, and whether the range clears zero.
function edge(values) {
  const m = meanVar(values);
  if (m.n < 2) return {n: m.n, mean: m.mean, lo: null, hi: null, tone: 'even'};
  const half = Z * Math.sqrt(m.var / m.n);
  return {n: m.n, mean: m.mean, lo: m.mean - half, hi: m.mean + half, total: m.mean * m.n, tone: m.mean - half > 0 ? 'up' : m.mean + half < 0 ? 'down' : 'even'};
}
// A share with its range (normal approximation), against a baseline share.
function share(k, n, base = null) {
  if (!n) return {k, n, p: null};
  const p = k / n, half = Z * Math.sqrt(Math.max(p * (1 - p), 1e-9) / n);
  const tone = base == null ? 'even' : p - half > base ? 'up' : p + half < base ? 'down' : 'even';
  return {k, n, p, lo: p - half, hi: p + half, base, tone};
}

const seasonsOf = sides => new Set(sides.map(s => s.season)).size || 1;
const ageGap = sides => {
  const xs = sides.filter(s => s.ageIn.length && s.ageOut.length).map(s => s.ageIn.reduce((a, b) => a + b, 0) / s.ageIn.length - s.ageOut.reduce((a, b) => a + b, 0) / s.ageOut.length);
  return edge(xs);
};

// What a group of sides does: how often, what posture, what it wins, which way picks and ages move.
export function behaviour(sides, league = sides) {
  const posture = Object.fromEntries(POSTURES.map(p => {
    const base = league.length ? league.filter(s => s.posture === p.id).length / league.length : null;
    return [p.id, share(sides.filter(s => s.posture === p.id).length, sides.length, league === sides ? null : base)];
  }));
  const partners = Object.fromEntries(TIERS.map(t => [t.id, share(sides.filter(s => s.otherTier === t.id).length, sides.length)]));
  const windows = Object.fromEntries(['spring', 'summer', 'season'].map(w => {
    const base = league.length ? league.filter(s => s.window === w).length / league.length : null;
    return [w, share(sides.filter(s => s.window === w).length, sides.length, league === sides ? null : base)];
  }));
  return {n: sides.length, perSeason: sides.length / seasonsOf(sides), posture, partners, windows,
    now: edge(sides.map(s => s.now)), real: edge(sides.filter(s => s.real != null).map(s => s.real)),
    picks: edge(sides.map(s => s.picksIn - s.picksOut)), age: ageGap(sides)};
}

// Each tier, season by season: how much it trades and how often it buys for now or builds.
export function tierTrends(sides) {
  const seasons = [...new Set(sides.map(s => s.season))].sort((a, b) => a - b);
  return {seasons, tiers: TIERS.map(t => {
    const mine = sides.filter(s => s.tier === t.id);
    const rows = seasons.map(y => {
      const ys = mine.filter(s => s.season === y);
      return {season: y, n: ys.length, now: share(ys.filter(s => s.posture === 'now').length, ys.length), later: share(ys.filter(s => s.posture === 'later').length, ys.length),
        net: ys.length ? ys.reduce((a, s) => a + s.now, 0) / ys.length : null};
    });
    // Has the tier's habit moved? Earlier half of the seasons against the later half.
    const mid = seasons[Math.floor((seasons.length - 1) / 2)];
    const shift = id => {
      const a = mine.filter(s => s.season <= mid), b = mine.filter(s => s.season > mid);
      if (a.length < 20 || b.length < 20) return null;
      const pa = a.filter(s => s.posture === id).length / a.length, pb = b.filter(s => s.posture === id).length / b.length;
      const se = Math.sqrt(pa * (1 - pa) / a.length + pb * (1 - pb) / b.length);
      return {early: pa, late: pb, from: [seasons[0], mid], to: [mid + 1, seasons.at(-1)], clear: Math.abs(pb - pa) > Z * se};
    };
    return {tier: t.id, rows, shift: {now: shift('now'), later: shift('later')}};
  })};
}

// Who trades with whom: trades between each pair of tiers, and what the row tier came away with.
export function tierPairs(sides) {
  return TIERS.map(a => ({tier: a.id, cells: TIERS.map(b => {
    const xs = sides.filter(s => s.tier === a.id && s.otherTier === b.id);
    return {other: b.id, n: xs.length, now: edge(xs.map(s => s.now)), real: edge(xs.filter(s => s.real != null).map(s => s.real))};
  })}));
}

// The teams as a table: one row per franchise.
export function teamRows(sides, league) {
  const by = new Map();
  for (const s of sides) { if (!by.has(s.fid)) by.set(s.fid, []); by.get(s.fid).push(s); }
  return [...by.entries()].map(([fid, mine]) => {
    const b = behaviour(mine, league);
    const partners = new Map();
    for (const s of mine) partners.set(s.other, (partners.get(s.other) || 0) + 1);
    const top = [...partners.entries()].sort((x, y) => y[1] - x[1])[0];
    return {fid, ...b, sides: mine, tiers: Object.fromEntries(TIERS.map(t => [t.id, mine.filter(s => s.tier === t.id).length / mine.length])),
      partner: top ? {fid: top[0], n: top[1]} : null, partnersCount: partners.size};
  });
}

// One team in depth.
export function teamProfile(data, sides, league, fid) {
  const mine = sides.filter(s => s.fid === fid);
  const seasons = [...new Set(league.map(s => s.season))].sort((a, b) => a - b);
  const standing = new Map(data.teamSeasons.filter(r => r.fid === fid).map(r => [r.year, r]));
  const bySeason = seasons.map(y => {
    const ys = mine.filter(s => s.season === y);
    const now = ys.filter(s => s.posture === 'now').length, later = ys.filter(s => s.posture === 'later').length;
    const tiers = ys.map(s => s.tier).filter(Boolean);
    const mostly = tiers.length ? TIERS.map(t => [t.id, tiers.filter(x => x === t.id).length]).sort((a, b) => b[1] - a[1])[0][0] : null;
    // Acting against the standing: buying now while rebuilding, or building while contending.
    const against = ys.filter(s => (s.tier === 'rebuild' && s.posture === 'now') || (s.tier === 'contender' && s.posture === 'later')).length;
    return {season: y, n: ys.length, now, later, swap: ys.length - now - later, mostly, against,
      net: ys.length ? ys.reduce((a, s) => a + s.now, 0) : null, finish: standing.get(y)?.rank ?? null, sides: ys};
  });
  // What they buy and sell, by kind of asset: count and forecast wins each way.
  const kinds = new Map();
  for (const s of mine) {
    for (const l of s.got) { const k = groupOf(l.xClass); const x = kinds.get(k) || {kind: k, inN: 0, outN: 0, inW: 0, outW: 0}; x.inN++; x.inW += l.xValue; kinds.set(k, x); }
    for (const l of s.gave) { const k = groupOf(l.xClass); const x = kinds.get(k) || {kind: k, inN: 0, outN: 0, inW: 0, outW: 0}; x.outN++; x.outW += l.xValue; kinds.set(k, x); }
  }
  // Who they trade with: each partner's count and what this team came away with.
  const partners = new Map();
  for (const s of mine) { if (!partners.has(s.other)) partners.set(s.other, []); partners.get(s.other).push(s); }
  return {fid, sides: mine, behaviour: behaviour(mine, league), bySeason,
    kinds: [...kinds.values()].filter(k => k.kind !== 'player:unknown').sort((a, b) => (b.inW - b.outW) - (a.inW - a.outW)),
    partners: [...partners.entries()].map(([other, xs]) => ({fid: other, n: xs.length, net: edge(xs.map(s => s.now))})).sort((a, b) => b.n - a.n),
    asTier: TIERS.map(t => ({tier: t.id, ...edge(mine.filter(s => s.tier === t.id).map(s => s.now)), posture: behaviour(mine.filter(s => s.tier === t.id), league).posture}))};
}
