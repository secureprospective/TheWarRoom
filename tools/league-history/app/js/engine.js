// The Lab's numbers, recomputed in the browser so every filter gives an honest answer.
// Mirrors compile/value.py (the market model) and adds the small-sample handling the screens
// use: shrinking rates toward the league and funnel limits for comparing teams.

export const ANCHOR = 'pick1:next';
const RIDGE = 0.25;
const Z = 1.96;

// ---------- linear algebra (tiny dense systems) ----------

function gauss(A, b) {
  const k = b.length;
  const M = A.map((r, i) => [...r, b[i]]);
  for (let c = 0; c < k; c++) {
    let p = c;
    for (let r = c + 1; r < k; r++) if (Math.abs(M[r][c]) > Math.abs(M[p][c])) p = r;
    [M[c], M[p]] = [M[p], M[c]];
    if (Math.abs(M[c][c]) < 1e-12) continue;
    for (let r = 0; r < k; r++) {
      if (r === c || !M[r][c]) continue;
      const f = M[r][c] / M[c][c];
      for (let j = c; j <= k; j++) M[r][j] -= f * M[c][j];
    }
  }
  return M.map((r, i) => (Math.abs(r[i]) > 1e-12 ? r[k] / r[i] : 0));
}

function inverse(A) {
  const k = A.length;
  const cols = [];
  for (let j = 0; j < k; j++) cols.push(gauss(A, A.map((_, i) => (i === j ? 1 : 0))));
  return A.map((_, i) => cols.map(c => c[i]));
}

// ---------- the market model ----------

// Trades in scope, each as two sides of {kind, value}: what side A gave, what side B gave.
export function valuedTrades(data, keep = () => true) {
  return data.trades.filter(t => t.valued && keep(t)).map(t => {
    const sides = [[], []];
    for (const l of t.legs) if (l.xValue != null) sides[l.side].push({kind: l.xClass, value: l.xValue});
    return {t, sides};
  });
}

// What the league pays per expected win for each kind of asset, against a next-draft 1st.
// Each trade says both sides were worth the same that day; an asset's price is its expected
// wins times (1 + m) for its kind, plus a fixed amount per asset. Ridge-shrunk, with standard
// errors robust to trades of different sizes. See compile/value.py fit_prices.
export function fitPrices(trades) {
  const kinds = [...new Set(trades.flatMap(v => v.sides.flat().map(a => a.kind)))].filter(k => k !== ANCHOR).sort();
  const idx = new Map(kinds.map((k, i) => [k, i]));
  const p = kinds.length + 1;
  const X = [], Y = [];
  for (const v of trades) {
    const x = new Array(p).fill(0);
    let y = 0;
    v.sides.forEach((side, s) => {
      const sign = s === 0 ? -1 : 1;
      for (const a of side) {
        if (idx.has(a.kind)) x[idx.get(a.kind)] += sign * a.value;
        x[p - 1] += sign;
        y += s === 0 ? a.value : -a.value;
      }
    });
    X.push(x);
    Y.push(y);
  }
  const A = Array.from({length: p}, () => new Array(p).fill(0));
  const b = new Array(p).fill(0);
  X.forEach((x, n) => {
    for (let i = 0; i < p; i++) {
      if (!x[i]) continue;
      b[i] += x[i] * Y[n];
      for (let j = 0; j < p; j++) A[i][j] += x[i] * x[j];
    }
  });
  for (let i = 0; i < p; i++) A[i][i] += RIDGE;
  const beta = gauss(A, b);
  const inv = inverse(A);
  const meat = Array.from({length: p}, () => new Array(p).fill(0));
  X.forEach((x, n) => {
    const e = Y[n] - x.reduce((s, xi, i) => s + xi * beta[i], 0);
    const nz = x.map((xi, i) => [i, xi]).filter(([, xi]) => xi);
    for (const [i, xi] of nz) for (const [j, xj] of nz) meat[i][j] += xi * xj * e * e;
  });
  // The full sandwich covariance, so ratios between two kinds carry their joint uncertainty.
  const half = inv.map(row => meat[0].map((_, c) => row.reduce((v, x, a) => (x ? v + x * meat[a][c] : v), 0)));
  const V = half.map(row => inv[0].map((_, j) => row.reduce((v, x, c) => v + x * inv[c][j], 0)));
  const prices = new Map(kinds.map((k, i) => [k, {price: 1 + beta[i], se: Math.sqrt(Math.max(0, V[i][i]))}]));
  prices.set(ANCHOR, {price: 1, se: 0});
  const cov = (a, b) => (idx.has(a) && idx.has(b) ? V[idx.get(a)][idx.get(b)] : 0);
  return {prices, cov, perAsset: beta[p - 1], trades: trades.length};
}

// Wins delivered per win forecast at the time, with a standard error across assets.
export function deliveredRatio(records) {
  const e = records.reduce((s, r) => s + r.expected, 0);
  if (records.length < 8 || e < 0.5) return null;
  const d = records.reduce((s, r) => s + r.delivered, 0);
  const ratio = d / e;
  const se = Math.sqrt(records.reduce((s, r) => s + (r.delivered - ratio * r.expected) ** 2, 0)) / e;
  return {ratio, se, n: records.length};
}

// Price per delivered win, with an interval that carries both uncertainties.
export function perDelivered(price, delivered) {
  if (!price || !delivered || delivered.ratio <= 0 || price.price <= 0) return null;
  const value = price.price / delivered.ratio;
  const rel = Math.sqrt((price.se / price.price) ** 2 + (delivered.se / delivered.ratio) ** 2);
  return {value, lo: value * Math.exp(-Z * rel), hi: value * Math.exp(Z * rel), spread: Math.exp(2 * Z * rel)};
}

// How much to trust a row: the ratio of the top of its range to the bottom.
export function evidence(range) {
  if (!range) return 'none';
  if (range.spread <= 2.5) return 'solid';
  if (range.spread <= 6) return 'rough';
  return 'thin';
}

// The full price list for a set of trades: prices, delivered share, price per delivered win,
// both halves of the period, and each time of year.
export function priceList(data, {from, to, windows = null}) {
  const keep = t => t.season >= from && t.season <= to && (!windows || windows.includes(t.window));
  const trades = valuedTrades(data, keep);
  if (trades.length < 40) return null;
  const fit = fitPrices(trades);
  const records = data.delivered.filter(r => keep(r.t));
  const seasons = [...new Set(trades.map(v => v.t.season))].sort((a, b) => a - b);
  const mid = seasons[Math.floor((seasons.length - 1) / 2)];
  const half = h => {
    const sub = trades.filter(v => (h === 0 ? v.t.season <= mid : v.t.season > mid));
    return sub.length >= 40 ? {fit: fitPrices(sub), records: records.filter(r => (h === 0 ? r.t.season <= mid : r.t.season > mid))} : null;
  };
  const halves = [half(0), half(1)];
  const byWindow = new Map();
  for (const w of ['spring', 'summer', 'season']) {
    if (windows && !windows.includes(w)) continue;
    const sub = trades.filter(v => v.t.window === w);
    if (sub.length >= 60) byWindow.set(w, fitPrices(sub));
  }
  const count = new Map();
  const value = new Map();
  for (const v of trades) for (const a of v.sides.flat()) {
    count.set(a.kind, (count.get(a.kind) || 0) + 1);
    value.set(a.kind, (value.get(a.kind) || 0) + a.value);
  }
  const rows = [...fit.prices.entries()].map(([kind, price]) => {
    const recs = records.filter(r => r.class === kind);
    const delivered = deliveredRatio(recs);
    const range = perDelivered(price, delivered);
    const halfRows = halves.map(hv => {
      if (!hv) return null;
      const p = hv.fit.prices.get(kind);
      const d = deliveredRatio(hv.records.filter(r => r.class === kind));
      return p ? {price: p, delivered: d, range: perDelivered(p, d)} : null;
    });
    const times = [...byWindow.entries()].map(([w, f]) => ({window: w, ...(f.prices.get(kind) || {})}))
      .filter(x => x.price != null && x.se < 0.5 * Math.max(x.price, 0.2));
    return {kind, n: count.get(kind) || 0, avgValue: (value.get(kind) || 0) / (count.get(kind) || 1),
      price, delivered, range, evidence: evidence(range), halves: halfRows, times};
  }).filter(r => r.kind !== 'player:unknown');
  return {rows, trades: trades.length, perAsset: fit.perAsset, from, to, mid, seasons,
    records: records.length};
}

// ---------- small samples ----------

// Shrink group means toward a common mean by how much each rests on (normal-normal empirical
// Bayes). The spread of true group means is estimated the meta-analysis way (DerSimonian and
// Laird), weighting each group by its precision, so a group of three can't swamp the estimate.
// cells: [{n, mean, var}] with var the spread of single observations.
export function shrink(cells, center) {
  const usable = cells.filter(c => c.n >= 2 && c.var > 0);
  if (usable.length < 2) return cells.map(c => ({...c, shrunk: null, lo: null, hi: null}));
  const w = usable.map(c => c.n / c.var);
  const sw = w.reduce((a, b) => a + b, 0);
  const pooled = usable.reduce((s, c, i) => s + w[i] * c.mean, 0) / sw;
  const q = usable.reduce((s, c, i) => s + w[i] * (c.mean - pooled) ** 2, 0);
  const tau2 = Math.max(0, (q - (usable.length - 1)) / (sw - w.reduce((s, x) => s + x * x, 0) / sw));
  return cells.map(c => {
    if (c.n < 2 || !(c.var > 0)) return {...c, shrunk: null, lo: null, hi: null};
    const s2 = c.var / c.n;
    const weight = tau2 / (tau2 + s2);
    const shrunk = center + weight * (c.mean - center);
    const sd = Math.sqrt(weight * s2);
    return {...c, weight, tau2, shrunk, lo: shrunk - Z * sd, hi: shrunk + Z * sd};
  });
}

export function meanVar(values) {
  const n = values.length;
  if (!n) return {n, mean: null, var: null};
  const mean = values.reduce((a, b) => a + b, 0) / n;
  const v = n > 1 ? values.reduce((s, x) => s + (x - mean) ** 2, 0) / (n - 1) : 0;
  return {n, mean, var: v};
}

// ---------- draft ----------

// Each judged pick: wins in its first `seasons` seasons against what its slot returned in the
// other classes over the same seasons.
export function judgedPicks(data, {seasons, from, to}) {
  return data.draft.filter(d => d.year >= from && d.year <= to && d.year + seasons - 1 <= data.lastFinished)
    .map(d => {
      const got = d.wins.slice(0, seasons), slot = d.slotWins.slice(0, seasons);
      if (got.some(v => v == null) || slot.some(v => v == null)) return null;
      const wins = got.reduce((a, b) => a + b, 0), expected = slot.reduce((a, b) => a + b, 0);
      return {...d, got: wins, expected, over: wins - expected};
    }).filter(Boolean);
}

// Position by draft band: wins over the slot, shrunk within each band, plus where the league
// spends its picks.
export function draftGrid(picks, groups, bands) {
  const out = new Map();
  for (const b of bands) {
    const inBand = picks.filter(p => p.band === b.id);
    const bandStats = meanVar(inBand.map(p => p.over));
    const cells = groups.map(g => {
      const mine = inBand.filter(p => p.group === g.id);
      return {group: g.id, band: b.id, picks: mine, share: inBand.length ? mine.length / inBand.length : 0,
        expected: mine.length ? mine.reduce((s, p) => s + p.expected, 0) / mine.length : null, ...meanVar(mine.map(p => p.over))};
    });
    const pooled = cells.filter(c => c.n >= 2);
    // Within-cell spread is pooled across positions so a cell of three picks doesn't claim certainty.
    const within = pooled.length ? pooled.reduce((s, c) => s + c.var * (c.n - 1), 0) / Math.max(1, pooled.reduce((s, c) => s + c.n - 1, 0)) : 0;
    const shrunk = shrink(cells.map(c => ({...c, var: within})), bandStats.mean ?? 0);
    out.set(b.id, {band: b, n: inBand.length, mean: bandStats.mean, expected: inBand.length ? inBand.reduce((s, p) => s + p.expected, 0) / inBand.length : null,
      cells: new Map(shrunk.map(c => [c.group, c]))});
  }
  return out;
}

// Each position against its slot across every band: more picks per position, so real
// differences can clear chance. Shrunk toward zero (the slot) the same way as the grid.
export function positionEffects(picks, groups, bands) {
  const cells = groups.map(g => {
    const mine = picks.filter(p => p.group === g.id);
    const expected = mine.length ? mine.reduce((s, p) => s + p.expected, 0) / mine.length : null;
    const share = new Map(bands.map(b => [b.id, mine.length ? mine.filter(p => p.band === b.id).length / mine.length : 0]));
    return {group: g.id, picks: mine, expected, share, ...meanVar(mine.map(p => p.over))};
  });
  return shrink(cells, 0);
}

// ---------- the league's market: buy list, sell list, calendar ----------

// The kind of asset a GM thinks in: a position at any age, or a pick by round and draft.
export function groupOf(kind) {
  const [k] = kind.split(':');
  return kind.startsWith('pick') || k === 'player' ? kind : k;
}

// The same, with players kept apart by age band: "LB:prime". The model's own kinds already are.
export const ageOf = kind => kind;

// The asset a GM picks in the filter: a position, or a pick by round and draft.
export const assetOf = kind => (kind.startsWith('pick') ? kind : kind.split(':')[0]);

function regroup(trades, of) {
  return trades.map(v => ({t: v.t, sides: v.sides.map(side => side.map(a => ({kind: of(a.kind), value: a.value})))}));
}

// The going rate: what a delivered win cost across everything traded, the median asset's price.
function goingRate(rows) {
  const items = rows.filter(r => r.range).sort((a, b) => a.range.value - b.range.value);
  const total = items.reduce((s, r) => s + r.n, 0);
  let seen = 0;
  for (const r of items) {
    seen += r.n;
    if (seen >= total / 2) return r.range.value;
  }
  return null;
}

// The typical price per forecast win: the median traded asset's.
function typicalPrice(rows) {
  const items = rows.filter(r => r.price.price > 0).sort((a, b) => a.price.price - b.price.price);
  const total = items.reduce((s, r) => s + r.n, 0);
  let seen = 0;
  for (const r of items) {
    seen += r.n;
    if (seen >= total / 2) return r.price.price;
  }
  return null;
}

// Cost per delivered win for each kind, against the going rate. The going rate is always taken
// over positions and picks, so splitting players by age changes the rows, never the yardstick.
function costRows(trades, records, of) {
  const rowsFor = by => {
    const fit = fitPrices(regroup(trades, by));
    const count = new Map();
    for (const v of trades) for (const a of v.sides.flat()) count.set(by(a.kind), (count.get(by(a.kind)) || 0) + 1);
    return [...fit.prices.entries()].filter(([k]) => k !== 'player:unknown').map(([kind, price]) => {
      const delivered = deliveredRatio(records.filter(r => by(r.class) === kind));
      return {kind, n: count.get(kind) || 0, price, delivered, range: perDelivered(price, delivered)};
    });
  };
  const base = rowsFor(groupOf);
  const going = goingRate(base);
  const paying = typicalPrice(base);
  const rows = of === groupOf ? base : rowsFor(of);
  for (const r of rows) {
    r.cost = r.range && going ? {value: r.range.value / going, lo: r.range.lo / going, hi: r.range.hi / going} : null;
    r.side = !r.cost ? null : r.cost.hi < 1 ? 'cheap' : r.cost.lo > 1 ? 'dear' : 'fair';
    // What GMs paid per forecast win against the typical asset: known even before a kind has
    // played out, so a row too young to judge on delivery still says what the league paid.
    r.paid = paying ? {value: r.price.price / paying, lo: (r.price.price - Z * r.price.se) / paying, hi: (r.price.price + Z * r.price.se) / paying} : null;
  }
  return {rows, going, trades: trades.length};
}

// The trades a market view prices: valued trades in the seasons, in the chosen stretches of the year.
export function inScope({from, to, phases = null}) {
  return t => t.season >= from && t.season <= to && (!phases || phases.includes(t.phase));
}

// Fewest valued trades a price list is fitted on, for the whole view and for each half or window.
export const MIN_TRADES = {whole: 80, part: 60};

// Buy list and sell list: each kind's cost per delivered win against the going rate, whether it
// sat on the same side in both halves of the seasons, and how its price moved by time of year.
// phases narrows the trades to stretches of the league year; byAge keeps players apart by age.
export function leagueMarket(data, {from, to, phases = null, byAge = false}) {
  const keep = inScope({from, to, phases});
  const of = byAge ? ageOf : groupOf;
  const trades = valuedTrades(data, keep);
  if (trades.length < MIN_TRADES.whole) return {trades: trades.length, rows: null, from, to};
  const records = data.delivered.filter(r => keep(r.t));
  const all = costRows(trades, records, of);
  const seasons = [...new Set(trades.map(v => v.t.season))].sort((a, b) => a - b);
  const mid = seasons[Math.floor((seasons.length - 1) / 2)];
  const halfOf = t => (t.season <= mid ? 0 : 1);
  const halves = [0, 1].map(h => {
    const sub = trades.filter(v => halfOf(v.t) === h);
    return sub.length >= MIN_TRADES.part ? costRows(sub, records.filter(r => halfOf(r.t) === h), of) : null;
  });
  // By time of year: price per expected win against that window's going rate. Delivered wins are
  // too thin to split three ways, so this is what GMs paid, not what they got. Only the windows
  // the chosen stretches reach, and only those with enough trades of their own.
  const reached = ['spring', 'summer', 'season'].filter(w => trades.some(v => v.t.window === w));
  const windows = reached.map(w => {
    const sub = trades.filter(v => v.t.window === w);
    if (sub.length < MIN_TRADES.part) return {window: w, prices: null};
    const fit = fitPrices(regroup(sub, of));
    const base = fitPrices(regroup(sub, groupOf));
    const n = new Map(), nBase = new Map();
    for (const v of sub) for (const a of v.sides.flat()) {
      n.set(of(a.kind), (n.get(of(a.kind)) || 0) + 1);
      nBase.set(groupOf(a.kind), (nBase.get(groupOf(a.kind)) || 0) + 1);
    }
    const items = [...base.prices.entries()].filter(([k]) => k !== 'player:unknown').sort((a, b) => a[1].price - b[1].price);
    const total = items.reduce((s, [k]) => s + (nBase.get(k) || 0), 0);
    let seen = 0, median = 1;
    for (const [k, p] of items) { seen += nBase.get(k) || 0; if (seen >= total / 2) { median = p.price; break; } }
    return {window: w, prices: fit.prices, median, n};
  });
  for (const r of all.rows) {
    r.halves = halves.map(hv => hv?.rows.find(x => x.kind === r.kind) || null);
    // Each half on its own: does its whole likely range clear the going rate on this kind's side?
    const clears = x => !!x?.cost && (r.side === 'cheap' ? x.cost.hi < 1 : r.side === 'dear' ? x.cost.lo > 1 : false);
    const [early, late] = r.halves.map(clears);
    r.trend = !r.side || r.side === 'fair' ? null : early && late ? 'holds' : late ? 'new' : early ? 'fading' : 'whole';
    r.holds = r.trend === 'holds';
    r.times = windows.map(w => {
      const p = w.prices?.get(r.kind);
      const n = w.n?.get(r.kind) || 0;
      // Only where the kind traded often enough in that window for its price to mean something.
      return p && n >= 25 && p.se < 0.5 * Math.max(p.price, 0.2) ? {window: w.window, cost: p.price / w.median, se: p.se / w.median, n} : {window: w.window, cost: null, n};
    });
    const known = r.times.filter(x => x.cost != null);
    r.cheapest = r.dearest = null;
    if (known.length >= 2) {
      const lo = known.reduce((a, b) => (b.cost < a.cost ? b : a)), hi = known.reduce((a, b) => (b.cost > a.cost ? b : a));
      // A window is named only when it beats every other window beyond chance.
      const beats = (w, sign) => known.every(o => o === w || sign * (o.cost - w.cost) > Z * Math.hypot(o.se, w.se));
      r.cheapest = beats(lo, 1) ? lo.window : null;     // when to buy it
      r.dearest = beats(hi, -1) ? hi.window : null;     // when to sell it
    }
  }
  return {...all, seasons, mid, from, to, phases};
}

// How much the league trades in each stretch of the year: completed trades per finished season,
// and the months each stretch mostly falls in (US Eastern).
const MONTH = new Intl.DateTimeFormat('en-US', {month: 'numeric', timeZone: 'America/New_York'});
export function tradesByPhase(data, {from, to, assets = null}) {
  const out = new Map();
  const years = new Set();
  // With assets chosen, only valued trades that moved one of them: the legs carry the kind.
  const moves = t => !assets || t.legs.some(l => l.xClass && assets.includes(assetOf(l.xClass)));
  for (const t of data.trades) {
    if (t.oneSided || t.year < from || t.year > to || !t.phase || !moves(t)) continue;
    const p = out.get(t.phase) || {n: 0, months: new Array(12).fill(0)};
    p.n += 1;
    p.months[+MONTH.format(t.date) - 1] += 1;
    out.set(t.phase, p);
    years.add(t.year);
  }
  for (const p of out.values()) {
    p.perSeason = p.n / Math.max(1, years.size);
    // The months holding at least a tenth of the stretch's trades.
    const held = p.months.map((c, i) => (c >= 0.1 * p.n ? i : -1)).filter(i => i >= 0);
    p.span = [held[0], held.at(-1)];
  }
  return {phases: out, seasons: years.size, total: [...out.values()].reduce((s, p) => s + p.n, 0)};
}

// The pick clock, in each half of the seasons: a 1st a year before its draft against the same
// round once its draft is next, per expected win, in the trades in view.
export function pickClock(data, {from, to, phases = null}) {
  const trades = valuedTrades(data, inScope({from, to, phases}));
  if (trades.length < MIN_TRADES.whole) return null;
  const seasons = [...new Set(trades.map(v => v.t.season))].sort((a, b) => a - b);
  const mid = seasons[Math.floor((seasons.length - 1) / 2)];
  const halves = [[seasons[0], mid], [mid + 1, seasons.at(-1)]].filter(([a, b]) => a <= b).map(([a, b]) => {
    const sub = trades.filter(v => v.t.season >= a && v.t.season <= b);
    const p = sub.length >= MIN_TRADES.part ? fitPrices(sub).prices.get('pick1:later') : null;
    // Clear only when the whole likely range sits below the next-draft 1st.
    return {from: a, to: b, trades: sub.length,
      clock: p ? {value: p.price, lo: p.price - Z * p.se, hi: p.price + Z * p.se, clear: p.price + Z * p.se < 1 && p.price > 0} : null};
  });
  return {halves};
}

// Contenders (ranked 1-10 at the time) trading with rebuilders (23-32): who comes out ahead.
export function contenderTrades(data, {from, to, phases = null}) {
  const keep = inScope({from, to, phases});
  const nets = [], real = [];
  for (const t of data.trades) {
    if (!t.valued || !keep(t) || t.aRank == null || t.bRank == null) continue;
    const sign = t.aRank <= 10 && t.bRank >= 23 ? 1 : t.bRank <= 10 && t.aRank >= 23 ? -1 : 0;
    if (!sign) continue;
    nets.push(sign * t.aNet);
    if (t.aRealNet != null && t.realSeasons >= 2) real.push(sign * t.aRealNet);
  }
  const m = meanVar(nets), r = meanVar(real);
  return {n: m.n, mean: m.mean, half: m.n > 1 ? Z * Math.sqrt(m.var / m.n) : null,
    realN: r.n, realMean: r.mean, realHalf: r.n > 1 ? Z * Math.sqrt(r.var / r.n) : null};
}

// ---------- one asset's own market: when, why, against what ----------

// The typical price per forecast win in a fit: the median traded asset's, weighted by how often
// each kind traded.
function typicalOf(fit, n) {
  const items = [...fit.prices.entries()].filter(([k, p]) => k !== 'player:unknown' && p.price > 0).sort((a, b) => a[1].price - b[1].price);
  const total = items.reduce((s, [k]) => s + (n.get(k) || 0), 0);
  let seen = 0;
  for (const [k, p] of items) { seen += n.get(k) || 0; if (seen >= total / 2) return p.price; }
  return null;
}

function legCounts(trades, of) {
  const n = new Map();
  for (const v of trades) for (const a of v.sides.flat()) n.set(of(a.kind), (n.get(of(a.kind)) || 0) + 1);
  return n;
}

// Price per forecast win for every kind in each stretch of the year, against that stretch's
// typical asset. Fitted once per view and shared by every asset shown.
export function stretchPrices(data, {from, to, byAge = false}, phases) {
  const of = byAge ? ageOf : groupOf;
  return new Map(phases.map(ph => {
    const trades = valuedTrades(data, inScope({from, to, phases: [ph]}));
    if (trades.length < MIN_TRADES.part) return [ph, {trades: trades.length, fit: null}];
    const fit = fitPrices(regroup(trades, of));
    const n = legCounts(trades, of);
    return [ph, {trades: trades.length, fit, n, typical: typicalOf(fitPrices(regroup(trades, groupOf)), legCounts(trades, groupOf))}];
  }));
}

// Everything the Lab can say about one kind of asset in the trades in view.
//  why:     what GMs paid per forecast win against the typical asset, and what share of its
//           forecast it delivered; cost per delivered win is the one over the other.
//  when:    what it was paid in each stretch of the year, against that stretch's typical asset.
//  against: what a delivered win through it costs against one through every other kind.
//  back:    what actually came back for it: the other side's forecast wins, by kind.
//  buyers:  who took it on, by standing at the time, against who took on anything.
export function assetProfile(data, {from, to, phases = null, byAge = false}, kind, stretches) {
  const of = byAge ? ageOf : groupOf;
  const keep = inScope({from, to, phases});
  const trades = valuedTrades(data, keep);
  if (trades.length < MIN_TRADES.whole) return null;
  const fit = fitPrices(regroup(trades, of));
  const n = legCounts(trades, of);
  const price = fit.prices.get(kind);
  if (!price) return {kind, n: 0};
  const typical = typicalOf(fitPrices(regroup(trades, groupOf)), legCounts(trades, groupOf));
  const rel = (p, base) => ({value: p.price / base, lo: (p.price - Z * p.se) / base, hi: (p.price + Z * p.se) / base});
  const delivered = deliveredRatio(data.delivered.filter(r => keep(r.t) && of(r.class) === kind));

  const when = [...stretches.entries()].map(([ph, sx]) => {
    const p = sx.fit?.prices.get(kind);
    const k = sx.n?.get(kind) || 0;
    const usable = p && sx.typical && k >= 12 && p.se < 0.6 * Math.max(p.price, 0.2);
    return {phase: ph, n: k, trades: sx.trades, paid: usable ? {...rel(p, sx.typical), se: p.se / sx.typical} : null};
  });
  const known = when.filter(w => w.paid);
  if (known.length >= 2) {
    const lo = known.reduce((a, b) => (b.paid.value < a.paid.value ? b : a));
    const hi = known.reduce((a, b) => (b.paid.value > a.paid.value ? b : a));
    // Named only when it beats every other stretch beyond chance.
    const beats = (w, sign) => known.every(o => o === w || sign * (o.paid.value - w.paid.value) > Z * Math.hypot(o.paid.se, w.paid.se));
    if (beats(lo, 1)) lo.cheapest = true;
    if (beats(hi, -1)) hi.dearest = true;
  }

  // Against every other kind: what a delivered win through this kind costs, as a multiple of a
  // delivered win through the other. Prices carry their joint covariance; delivered shares are
  // independent samples. Where either has too little played out, the ratio stays per forecast win.
  const deliveredOf = new Map();
  for (const r of data.delivered) if (keep(r.t)) {
    const k = of(r.class);
    if (!deliveredOf.has(k)) deliveredOf.set(k, []);
    deliveredOf.get(k).push(r);
  }
  const against = [...fit.prices.entries()].filter(([k]) => k !== kind && k !== 'player:unknown' && (n.get(k) || 0) >= 25).map(([k, q]) => {
    if (price.price <= 0 || q.price <= 0) return null;
    let ratio = price.price / q.price;
    let v = (price.se / price.price) ** 2 + (q.se / q.price) ** 2 - 2 * fit.cov(kind, k) / (price.price * q.price);
    const dq = deliveredRatio(deliveredOf.get(k) || []);
    const byDelivered = !!(delivered && dq && delivered.ratio > 0 && dq.ratio > 0);
    if (byDelivered) {
      ratio *= dq.ratio / delivered.ratio;
      v += (delivered.se / delivered.ratio) ** 2 + (dq.se / dq.ratio) ** 2;
    }
    const sd = Math.sqrt(Math.max(0, v));
    const lo = ratio * Math.exp(-Z * sd), hi = ratio * Math.exp(Z * sd);
    return {kind: k, n: n.get(k), ratio, lo, hi, byDelivered, side: hi < 1 ? 'cheaper' : lo > 1 ? 'dearer' : 'even'};
  }).filter(Boolean).sort((a, b) => a.ratio - b.ratio);

  // What came back: in each trade that moved it, the other side's forecast wins by kind.
  const back = new Map();
  let backTotal = 0, moved = 0;
  const buyers = {contender: 0, middle: 0, rebuilder: 0}, everyone = {contender: 0, middle: 0, rebuilder: 0};
  const standing = r => (r == null ? null : r <= 10 ? 'contender' : r >= 23 ? 'rebuilder' : 'middle');
  for (const v of trades) {
    for (const l of v.t.legs) {
      const st = l.xClass ? standing(l.toRank) : null;
      if (st) everyone[st] += 1;
    }
    const mine = v.t.legs.filter(l => l.xClass && of(l.xClass) === kind);
    if (!mine.length) continue;
    moved += 1;
    for (const l of mine) { const st = standing(l.toRank); if (st) buyers[st] += 1; }
    const sides = new Set(mine.map(l => l.side));
    for (const l of v.t.legs) {
      if (!l.xClass || sides.has(l.side) || l.xValue == null) continue;
      const k = groupOf(l.xClass);
      back.set(k, (back.get(k) || 0) + l.xValue);
      backTotal += l.xValue;
    }
  }
  const share = o => { const t = o.contender + o.middle + o.rebuilder; return Object.fromEntries(Object.entries(o).map(([k, c]) => [k, t ? c / t : 0])); };
  return {kind, n: n.get(kind) || 0, moved, trades: trades.length,
    paid: typical ? rel(price, typical) : null, delivered,
    when, against,
    back: [...back.entries()].map(([k, w]) => ({kind: k, wins: w, share: backTotal ? w / backTotal : 0})).sort((a, b) => b.wins - a.wins),
    buyers: {mine: share(buyers), all: share(everyone), n: buyers.contender + buyers.middle + buyers.rebuilder}};
}
