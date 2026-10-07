// The decoded league data and every label a GM reads, so wording changes happen in one place.

export const GROUPS = [
  {id: 'QB', label: 'Quarterback', plural: 'quarterbacks', short: 'QB'},
  {id: 'RB', label: 'Running back', plural: 'running backs', short: 'RB'},
  {id: 'WR', label: 'Wide receiver', plural: 'wide receivers', short: 'WR'},
  {id: 'TE', label: 'Tight end', plural: 'tight ends', short: 'TE'},
  {id: 'PK', label: 'Kicker', plural: 'kickers', short: 'K'},
  {id: 'DL', label: 'Defensive lineman', plural: 'defensive linemen', short: 'DL', hint: 'Defensive tackles and ends'},
  {id: 'LB', label: 'Linebacker', plural: 'linebackers', short: 'LB'},
  {id: 'DB', label: 'Defensive back', plural: 'defensive backs', short: 'DB', hint: 'Cornerbacks and safeties'},
];
export const GROUP_LABEL = Object.fromEntries(GROUPS.map(g => [g.id, g.label]));
export const GROUP_PLURAL = Object.fromEntries(GROUPS.map(g => [g.id, g.plural]));
export const OFFENSE = new Set(['QB', 'RB', 'WR', 'TE', 'PK']);

export const AGES = [
  {id: 'young', label: '24 or younger'},
  {id: 'prime', label: '25 to 27'},
  {id: 'older', label: '28 or older'},
];
const AGE_LABEL = Object.fromEntries(AGES.map(a => [a.id, a.label]));

// Times of the league year a trade can happen in, as the market model groups them.
export const WINDOWS = [
  {id: 'spring', label: 'Offseason', hint: 'From the start of the league year to the rookie draft (and winter trades in the years they were allowed).'},
  {id: 'summer', label: 'Draft to kickoff', hint: 'From the rookie draft to the first NFL game.'},
  {id: 'season', label: 'In season', hint: 'From kickoff to the trade deadline.'},
];
export const WINDOW_LABEL = Object.fromEntries(WINDOWS.map(w => [w.id, w.label]));

// The stretches of the league year, in calendar order, each set by that year's own dates
// (compile/build_lab.py phase). Each sits inside one of the three windows above.
export const PHASES = [
  {id: 'winter', window: 'spring', label: 'After the season', hint: 'From the end of the season to the new league year, in the years winter trades were allowed.'},
  {id: 'opening', window: 'spring', label: 'Before the NFL draft', hint: 'From the start of the league year to the NFL draft.'},
  {id: 'between', window: 'spring', label: 'NFL draft to rookie draft', hint: 'From the NFL draft to the start of the league’s rookie draft.'},
  {id: 'rookieDraft', window: 'summer', label: 'During the rookie draft', hint: 'While the league’s rookie draft is running.'},
  {id: 'summer', window: 'summer', label: 'Rookie draft to kickoff', hint: 'From the end of the rookie draft to the first NFL game.'},
  {id: 'early', window: 'season', label: 'Weeks 1–4', hint: 'The first four weeks of the NFL season.'},
  {id: 'midseason', window: 'season', label: 'Midseason', hint: 'From week 5 to two weeks before the trade deadline.'},
  {id: 'deadline', window: 'season', label: 'Deadline run‑in', hint: 'The last two weeks before the trade deadline.'},
];

// A kind of asset, as the market model prices it: "LB:prime", "pick1:next".
export function kindLabel(id) {
  const [kind, band] = id.split(':');
  if (kind.startsWith('pick')) {
    const r = {1: '1st', 2: '2nd', 3: '3rd or later'}[kind.slice(4)];
    return `${r}-round pick, ${band === 'next' ? 'next draft' : 'a later draft'}`;
  }
  if (kind === 'player') return 'Player with no record';
  return `${GROUP_LABEL[kind]}, ${AGE_LABEL[band]}`;
}
export const kindIsPick = id => id.startsWith('pick');
export const kindGroup = id => id.split(':')[0];

// Draft bands: where in the draft a pick sits.
export const BANDS = [
  {id: 'early1', label: 'Early 1st', hint: 'Picks 1.01 to 1.10', test: d => d.round === 1 && d.slot <= 10},
  {id: 'late1', label: 'Late 1st', hint: 'Picks 1.11 to 1.32', test: d => d.round === 1 && d.slot > 10},
  {id: 'r2', label: '2nd round', test: d => d.round === 2},
  {id: 'r3', label: '3rd round', test: d => d.round === 3},
  {id: 'r4', label: '4th round on', test: d => d.round >= 4},
];
export const bandOf = d => BANDS.find(b => b.test(d))?.id;

function table(cols) {
  const names = Object.keys(cols);
  const n = names.length ? cols[names[0]].length : 0;
  const out = new Array(n);
  for (let i = 0; i < n; i++) {
    const row = {};
    for (const k of names) row[k] = cols[k][i];
    out[i] = row;
  }
  return out;
}

export function decode(raw) {
  const seasons = new Map(raw.seasons.map(s => [s.year, s]));
  const franchises = new Map(raw.franchises.map(f => [f.id, f]));
  const players = new Map(table(raw.players).map(p => [p.id, p]));
  const lastFinished = Math.max(...raw.seasons.filter(s => s.weeksDone.length >= s.regularWeeks).map(s => s.year));

  const trades = table(raw.trades);
  trades.forEach((t, i) => {
    t.id = i;
    t.date = new Date(t.ts * 1000);
    t.legs = [];
    t.valued = t.aNet != null;
  });
  const legs = table(raw.legs);
  for (const l of legs) {
    const t = trades[l.trade];
    l.t = t;
    l.side = l.from === t.a ? 0 : 1;          // 0: side A gave it, 1: side B gave it
    l.player = l.pid ? players.get(l.pid) : null;
    l.pickPlayer = l.pickPid ? players.get(l.pickPid) : null;
    t.legs.push(l);
  }
  const delivered = table(raw.delivered);
  for (const r of delivered) r.t = trades[r.trade];

  const draft = table(raw.draft);
  draft.forEach((d, i) => {
    d.id = i;
    d.player = players.get(d.pid);
    d.band = bandOf(d);
  });

  const teamSeasons = table(raw.teamSeasons);
  // Trade offers and how each ended. MFL shows a franchise only its own offers, so all involve one team.
  const offers = raw.offers ? table(raw.offers) : [];

  return {
    raw, seasons, franchises, players, trades, legs, delivered, draft, teamSeasons, offers, lastFinished,
    years: raw.seasons.map(s => s.year), market: raw.market, checks: {...raw.checks, ...raw.checks2}, ledger: raw.ledger, source: raw.source,
    archiveThrough: raw.archiveThrough,
  };
}

export function franchiseName(data, fid, short = false) {
  const f = data.franchises.get(fid);
  if (!f) return fid ?? 'Unknown team';
  return short ? f.short : f.name;
}

export function playerName(p) {
  if (!p) return 'Unknown player';
  const [last, first] = (p.name || '').split(', ');
  return first ? `${first} ${last}` : p.name;
}

export function pickLabel(l, data) {
  const r = ['1st', '2nd', '3rd', '4th', '5th', '6th'][l.round - 1] || `Round ${l.round}`;
  const from = l.orig ? ` (${franchiseName(data, l.orig, true)})` : '';
  const slot = l.slot ? ` · pick ${l.round}.${String(l.slot).padStart(2, '0')}` : '';
  return `${l.pickYear} ${r}${from}${slot}`;
}

export const fmt = {
  n: (v, d = 0) => v == null || Number.isNaN(v) ? '—' : Number(v).toLocaleString('en-US', {maximumFractionDigits: d, minimumFractionDigits: d}),
  pct: (v, d = 0) => v == null || Number.isNaN(v) ? '—' : `${(v * 100).toFixed(d)}%`,
  // MFL's clock is US Eastern: every day shown is the day MFL shows.
  date: d => d.toLocaleDateString('en-US', {month: 'short', day: 'numeric', year: 'numeric', timeZone: 'America/New_York'}),
  isoDay: d => d.toLocaleDateString('en-CA', {timeZone: 'America/New_York'}),
  signed: (v, d = 1) => {
    if (v == null || Number.isNaN(v)) return '—';
    const r = Number(v.toFixed(d));
    return `${r > 0 ? '+' : r < 0 ? '−' : ''}${Math.abs(r).toFixed(d)}`;
  },
  times: v => v == null ? '—' : `${v.toFixed(2)}×`,
  pick: (round, slot) => `${round}.${String(slot).padStart(2, '0')}`,
};
