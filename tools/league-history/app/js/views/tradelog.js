// How the teams trade: every team side of every valued trade, cut by where the team stood that day
// (contending, in the hunt, rebuilding), the time of year and what the side did (bought for now,
// built for later, like for like). The tiers, their trends, who trades with whom, the teams as a
// table and one team in depth, then the trades themselves.

import {h, tradeCard, empty, segmented, howCounted} from '../ui.js';
import {franchiseName, playerName, WINDOWS, PHASES, fmt} from '../model.js';
import {TIERS, POSTURES, sidesInScope, behaviour, tierTrends, tierPairs, teamRows, teamProfile} from '../teams.js';

const PAGE = 30;
const ORDER = [
  {id: 'new', label: 'Newest'},
  {id: 'edge', label: 'Most lopsided on the day', hint: 'Largest gap in expected wins between the two sides, on the day of the trade.'},
  {id: 'swing', label: 'Most lopsided since', hint: 'Largest gap in wins actually delivered since, where seasons have been played.'},
];
const TIER = Object.fromEntries(TIERS.map(t => [t.id, t]));
const POSTURE = Object.fromEntries(POSTURES.map(p => [p.id, p]));
const pct = v => (v == null ? '—' : `${fmt.n(v * 100)}%`);
const wins = v => `${fmt.signed(v, 2)}`;
const toneClass = tone => (tone === 'up' ? ' up' : tone === 'down' ? ' down' : '');

function toggle(sel, id, all) {
  if (!sel) return [id];
  const next = sel.includes(id) ? sel.filter(x => x !== id) : [...sel, id];
  return !next.length || next.length === all.length ? null : all.filter(x => next.includes(x));
}
const lit = (sel, id) => (!sel ? '' : sel.includes(id) ? ' on' : ' off');

// ---------- what to look at ----------

function tierPicker(ctx, st, counts, seasons) {
  const all = TIERS.map(t => t.id);
  const max = Math.max(...TIERS.map(t => counts[t.id] || 0), 1);
  return h('div', {class: 'yr'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Standing on the day'),
      h('span', {class: 'flt-note'}, `Trade sides a season from teams in each tier, ${seasons}. A team’s tier is its rank that day: live once three games are final, else last season’s finish.`),
      h('button', {class: `flt-all${st.tiers ? '' : ' on'}`, onclick: () => ctx.setLocal({tiers: null})}, 'Every team')),
    h('div', {class: 'yr-grid', style: {gridTemplateColumns: 'repeat(3, minmax(0, 1fr))'}}, TIERS.map(t => {
      const n = counts[t.id] || 0;
      return h('button', {class: `yr-phase${lit(st.tiers, t.id)}`, 'aria-pressed': String(!!st.tiers && st.tiers.includes(t.id)), title: t.hint,
        onclick: () => ctx.setLocal({tiers: toggle(st.tiers, t.id, all)})},
      h('span', {class: 'yr-bar-wrap'}, h('span', {class: 'yr-count num'}, fmt.n(n)), h('span', {class: 'yr-bar', style: {height: `${Math.max(2, Math.round((n / max) * 52))}px`, maxWidth: '120px'}})),
      h('span', {class: 'yr-label'}, t.label), h('span', {class: 'yr-months'}, t.hint.replace(' on the day of the trade', '')));
    })));
}

function sidePicker(ctx, st, data) {
  const chip = (key, all, id, label, hint) => h('button', {class: `chip${lit(st[key], id)}`, 'aria-pressed': String(!!st[key] && st[key].includes(id)), title: hint,
    onclick: () => ctx.setLocal({[key]: toggle(st[key], id, all)})}, label);
  const group = (title, ...xs) => h('div', {class: 'as-group', role: 'group', 'aria-label': title}, h('span', {class: 'as-title'}, title), xs);
  const teams = [...data.franchises.values()].sort((a, b) => a.name.localeCompare(b.name));
  return h('div', {class: 'as'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Trades'),
      h('span', {class: 'flt-presets'}, h('button', {class: `flt-all${st.windows || st.postures || st.team ? '' : ' on'}`, onclick: () => ctx.setLocal({windows: null, postures: null, team: ''})}, 'Every trade'))),
    h('div', {class: 'as-rows'},
      group('Time of year', WINDOWS.map(w => chip('windows', WINDOWS.map(x => x.id), w.id, w.label, w.hint))),
      group('What it did', POSTURES.map(p => chip('postures', POSTURES.map(x => x.id), p.id, p.label, p.hint))),
      group('Team', h('select', {'aria-label': 'Team', class: 'team-select', onchange: e => ctx.setLocal({team: e.target.value})},
        h('option', {value: ''}, 'Every team'), teams.map(f => h('option', {value: f.id, selected: st.team === f.id ? true : null}, f.name))))));
}

// ---------- the tiers ----------

function postureBar(p) {
  return h('div', {class: 'pbar', role: 'img', 'aria-label': POSTURES.map(x => `${x.label} ${pct(p[x.id].p)}`).join(', ')},
    POSTURES.map(x => h('span', {class: `pbar-${x.id}`, style: {width: `${(p[x.id].p || 0) * 100}%`}, title: `${x.label}: ${pct(p[x.id].p)}`})));
}
function postureKey(p) {
  return h('div', {class: 'pkey'}, POSTURES.map(x => h('span', {class: `pkey-${x.id}${toneClass(p[x.id].tone)}`}, h('i', {}), `${x.label} ${pct(p[x.id].p)}`)));
}
function edgeText(e, unit = 'expected wins a trade') {
  if (e.mean == null || e.lo == null) return '—';
  return e.tone === 'even' ? `${wins(e.mean)} ${unit}, within chance` : `${wins(e.mean)} ${unit}, likely ${wins(e.lo)} to ${wins(e.hi)}`;
}

function tierCards(sides, league) {
  return h('div', {class: 'tier-grid'}, TIERS.map(t => {
    const mine = sides.filter(s => s.tier === t.id);
    if (mine.length < 20) return h('section', {class: 'tier-card'}, h('h3', {}, t.label), empty('Too few trades in view.'));
    const b = behaviour(mine, league);
    const row = (label, value, tone = 'even') => h('div', {class: 'tier-row'}, h('span', {class: 'muted'}, label), h('span', {class: `num${toneClass(tone)}`}, value));
    return h('section', {class: 'tier-card', 'aria-label': t.label},
      h('h3', {}, t.label, h('span', {class: 'muted mk-why'}, t.hint.replace(' on the day of the trade', ''))),
      h('p', {class: 'tier-big num'}, fmt.n(b.perSeason), h('span', {class: 'mk-unit'}, ' trade sides a season')),
      postureBar(b.posture), postureKey(b.posture),
      row('On the day', edgeText(b.now), b.now.tone),
      row('Since (2+ seasons played)', edgeText(b.real, 'wins a trade'), b.real.tone),
      row('Picks in minus out', `${fmt.signed(b.picks.mean, 2)} a trade`, b.picks.tone),
      row('Age in minus out', b.age.n ? `${fmt.signed(b.age.mean, 1)} years` : '—', b.age.tone),
      row('Trades with', TIERS.map(o => `${o.label.toLowerCase()} ${pct(b.partners[o.id].p)}`).join(' · ')));
  }));
}

function trendTable(trends) {
  const head = h('tr', {}, h('th', {scope: 'col'}, 'Season'), TIERS.map(t => h('th', {scope: 'colgroup', colspan: 3, class: 'tt-group'}, t.label)));
  const sub = h('tr', {}, h('th', {}), TIERS.map(() => [h('th', {scope: 'col', class: 'num'}, 'Sides'), h('th', {scope: 'col', class: 'num'}, 'For now'), h('th', {scope: 'col', class: 'num'}, 'For later')]));
  const body = trends.seasons.map((y, i) => h('tr', {}, h('th', {scope: 'row'}, y), trends.tiers.map(t => {
    const r = t.rows[i];
    return [h('td', {class: 'num tt-first'}, r.n || '—'), h('td', {class: 'num'}, r.n >= 10 ? pct(r.now.p) : '—'), h('td', {class: 'num'}, r.n >= 10 ? pct(r.later.p) : '—')];
  })));
  return h('div', {class: 'table-wrap'}, h('table', {class: 'data compact tt'}, h('thead', {}, head, sub), h('tbody', {}, body)));
}

function trendSay(trends) {
  const bits = [];
  for (const t of trends.tiers) for (const [id, s] of Object.entries(t.shift)) {
    if (!s?.clear) continue;
    bits.push(`${TIER[t.tier].label} ${POSTURE[id].label.toLowerCase()} in ${pct(s.early)} of their trades in ${s.from[0]}–${String(s.from[1]).slice(2)} and ${pct(s.late)} in ${s.to[0]}–${String(s.to[1]).slice(2)}.`);
  }
  return bits.length ? `${bits.join(' ')} Every other habit held within chance between the two halves.` : 'No tier’s habit moved beyond chance between the earlier and later halves of these seasons.';
}

function pairTable(pairs) {
  return h('div', {class: 'table-wrap'}, h('table', {class: 'data dm-table pairs'},
    h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Trading with →'), TIERS.map(t => h('th', {scope: 'col'}, t.label)))),
    h('tbody', {}, pairs.map(r => h('tr', {}, h('th', {scope: 'row'}, TIER[r.tier].label), r.cells.map(c => h('td', {class: `dm-cell${toneClass(c.now.tone)}`,
      title: `${TIER[r.tier].label} trading with ${TIER[c.other].label.toLowerCase()}: ${edgeText(c.now)} on the day; ${edgeText(c.real, 'wins a trade')} since.`},
    h('span', {class: 'dm-value num'}, c.n ? wins(c.now.mean) : '—'), h('span', {class: 'dm-n'}, `${fmt.n(c.n)} trades`))))))));
}

// ---------- the teams ----------

const SORTS = {
  n: {label: 'Trades', key: r => r.n},
  perSeason: {label: 'A season', key: r => r.perSeason},
  now: {label: 'For now', key: r => r.posture.now.p},
  later: {label: 'For later', key: r => r.posture.later.p},
  edge: {label: 'On the day', key: r => r.now.total},
  real: {label: 'Since', key: r => r.real.total ?? -Infinity},
  picks: {label: 'Picks in − out', key: r => r.picks.mean},
  age: {label: 'Age in − out', key: r => r.age.mean ?? -Infinity},
};

function teamTable(ctx, data, rows, st) {
  const sort = SORTS[st.sort] ? st.sort : 'edge';
  const sorted = [...rows].sort((a, b) => SORTS[sort].key(b) - SORTS[sort].key(a));
  const th = id => h('th', {scope: 'col', class: `num sortable${sort === id ? ' sorted' : ''}`, 'aria-sort': sort === id ? 'descending' : null},
    h('button', {class: 'th-sort', onclick: () => ctx.setLocal({sort: id})}, SORTS[id].label));
  const tierBar = r => h('div', {class: 'pbar small', title: TIERS.map(t => `${t.label} ${pct(r.tiers[t.id])}`).join(', ')},
    TIERS.map(t => h('span', {class: `tbar-${t.id}`, style: {width: `${(r.tiers[t.id] || 0) * 100}%`}})));
  return h('div', {class: 'table-wrap'}, h('table', {class: 'data team-table'},
    h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Team'), th('n'), th('perSeason'),
      h('th', {scope: 'col', title: 'Share of its trade sides as a contender, in the hunt, rebuilding'}, 'Standing when trading'),
      th('now'), th('later'), th('edge'), th('real'), th('picks'), th('age'), h('th', {scope: 'col'}, 'Most trades with'))),
    h('tbody', {}, sorted.map(r => {
      const open = () => ctx.setLocal({team: r.fid});
      return h('tr', {class: `click${st.team === r.fid ? ' focus-row' : ''}`, tabindex: '0', onclick: open, onkeydown: e => { if (e.key === 'Enter') open(); }},
        h('th', {scope: 'row'}, franchiseName(data, r.fid)),
        h('td', {class: 'num'}, fmt.n(r.n)), h('td', {class: 'num'}, fmt.n(r.perSeason, 1)),
        h('td', {}, tierBar(r)),
        h('td', {class: `num${toneClass(r.posture.now.tone)}`}, pct(r.posture.now.p)),
        h('td', {class: `num${toneClass(r.posture.later.tone)}`}, pct(r.posture.later.p)),
        h('td', {class: `num${toneClass(r.now.tone)}`, title: edgeText(r.now)}, wins(r.now.total)),
        h('td', {class: `num${toneClass(r.real.tone)}`, title: edgeText(r.real, 'wins a trade')}, r.real.n ? wins(r.real.total) : '—'),
        h('td', {class: 'num'}, fmt.signed(r.picks.mean, 2)),
        h('td', {class: 'num'}, r.age.n ? fmt.signed(r.age.mean, 1) : '—'),
        h('td', {}, r.partner ? `${franchiseName(data, r.partner.fid, true)} (${r.partner.n})` : '—'));
    }))));
}

// ---------- one team ----------

function teamPanel(ctx, data, p, st) {
  const b = p.behaviour;
  const name = franchiseName(data, p.fid);
  const open = (title, sides) => ctx.openTrades(title, [...new Set(sides.map(s => s.t))]);
  const seasonCells = h('div', {class: 'ap-when-grid', style: {gridTemplateColumns: `repeat(${p.bySeason.length}, minmax(0, 1fr))`}}, p.bySeason.map(y => {
    // Flagged when at least two trades, and nearly a third of the season's, went against the standing.
    const flag = y.against >= 2 && y.against / y.n >= 0.3;
    return h('button', {class: `ap-when-cell season-cell${flag ? ' against' : ''}`, disabled: !y.n || null,
      title: `${y.season}: ${y.n} trade sides; ${y.now} for now, ${y.later} for later, ${y.swap} like for like${y.finish ? `; finished ${y.finish}` : ''}.${flag ? ` ${y.against} went against its standing that day.` : ''}`,
      onclick: () => open(`${name}, ${y.season}`, y.sides)},
    h('span', {class: 'ap-when-label'}, `${y.season}${y.finish ? ` · finished ${y.finish}` : ''}`),
    h('span', {class: 'ap-when-value num'}, y.n ? fmt.n(y.n) : '—'),
    y.n ? h('div', {class: 'pbar small'}, ['now', 'later', 'swap'].map(k => h('span', {class: `pbar-${k}`, style: {width: `${(y[k] / y.n) * 100}%`}}))) : null,
    h('span', {class: 'ap-when-n'}, y.n ? `${y.mostly ? TIER[y.mostly].one.toLowerCase() : ''}${y.net != null ? ` · ${wins(y.net)}` : ''}` : 'no trades'));
  }));
  const kindName = k => (k.startsWith('pick') ? `${k.split(':')[1] === 'next' ? 'Next' : 'Later'}-draft ${{1: '1sts', 2: '2nds', 3: '3rds+'}[k[4]]}` : {QB: 'Quarterbacks', RB: 'Running backs', WR: 'Wide receivers', TE: 'Tight ends', PK: 'Kickers', DL: 'Defensive linemen', LB: 'Linebackers', DB: 'Defensive backs'}[k] || k);
  const maxW = Math.max(...p.kinds.map(k => Math.max(k.inW, k.outW)), 0.01);
  const flowRow = k => h('div', {class: 'flow-row'},
    h('span', {class: 'ap-row-name'}, kindName(k.kind)),
    h('span', {class: 'flow-out'}, h('span', {style: {width: `${(k.outW / maxW) * 100}%`}})),
    h('span', {class: 'flow-in'}, h('span', {style: {width: `${(k.inW / maxW) * 100}%`}})),
    h('span', {class: 'ap-row-value num'}, `${k.outN} out · ${k.inN} in`));
  return h('section', {class: 'ap', 'aria-label': name},
    h('header', {class: 'ap-head'}, h('h2', {}, name),
      h('span', {class: `ap-cost num${toneClass(b.now.tone).replace('up', 'cheap').replace('down', 'dear')}`}, wins(b.now.total), h('span', {class: 'mk-unit'}, ' expected wins on the day, all trades')),
      b.real.n ? h('span', {class: `ap-hits ${b.real.tone}`}, `${wins(b.real.total)} wins since, over ${fmt.n(b.real.n)} trades with 2+ seasons played`) : null,
      h('button', {class: 'link ap-see', onclick: () => ctx.setLocal({team: ''})}, 'Back to every team')),
    h('div', {class: 'ap-cell ap-gap'}, h('h3', {}, 'Season by season'),
      h('p', {class: 'ap-say'}, `${fmt.n(b.n)} trade sides, ${fmt.n(b.perSeason, 1)} a season. Bars: bought for now, built for later, like for like. An outlined season is one where at least a third of its trades went against the team’s standing that day: buying for now while rebuilding, or building while contending. Click a season for its trades.`),
      seasonCells),
    h('div', {class: 'ap-grid even'},
      h('div', {class: 'ap-cell'}, h('h3', {}, 'How it trades, against the league'),
        postureBar(b.posture), postureKey(b.posture),
        h('div', {class: 'tier-rows'},
          p.asTier.map(t => h('div', {class: 'tier-row'}, h('span', {class: 'muted'}, `As ${TIER[t.tier].one.toLowerCase()} (${fmt.n(t.n)})`), h('span', {class: `num${toneClass(t.tone)}`}, t.n ? edgeText(t) : '—'))),
          h('div', {class: 'tier-row'}, h('span', {class: 'muted'}, 'Picks in minus out'), h('span', {class: 'num'}, `${fmt.signed(b.picks.mean, 2)} a trade`)),
          h('div', {class: 'tier-row'}, h('span', {class: 'muted'}, 'Age in minus out'), h('span', {class: 'num'}, b.age.n ? `${fmt.signed(b.age.mean, 1)} years` : '—')),
          h('div', {class: 'tier-row'}, h('span', {class: 'muted'}, 'Time of year'), h('span', {class: 'num'}, WINDOWS.map(w => `${w.label.toLowerCase()} ${pct(b.windows[w.id].p)}`).join(' · ')))),
        h('p', {class: 'mk-note'}, 'Coloured shares differ from the league’s beyond chance. On the day is what the trade was worth in expected wins at the league’s own prices.')),
      h('div', {class: 'ap-cell'}, h('h3', {}, 'What it sends and takes'),
        h('div', {class: 'flow-head'}, h('span', {}), h('span', {class: 'muted small'}, 'Sent'), h('span', {class: 'muted small'}, 'Taken in'), h('span', {})),
        h('div', {class: 'ap-rows'}, p.kinds.map(flowRow)),
        h('p', {class: 'mk-note'}, 'Bars are forecast wins sent and taken in for each kind of asset; the figures count the assets.'))),
    h('div', {class: 'ap-cell ap-gap'}, h('h3', {}, 'Who it trades with'),
      h('div', {class: 'partner-grid'}, p.partners.slice(0, 12).map(x => h('button', {class: 'partner', onclick: () => open(`${name} with ${franchiseName(data, x.fid)}`, p.sides.filter(s => s.other === x.fid))},
        h('span', {}, franchiseName(data, x.fid)), h('span', {class: 'num muted'}, `${x.n} trade${x.n === 1 ? '' : 's'}`),
        h('span', {class: `num${toneClass(x.net.tone)}`}, `${wins(x.net.total ?? x.net.mean)} on the day`)))),
      p.partners.length > 12 ? h('p', {class: 'mk-note'}, `And ${p.partners.length - 12} more partners.`) : null));
}

// ---------- offers against acceptance ----------

const OUTCOMES = [
  {id: 'accepted', label: 'Accepted'}, {id: 'rejected', label: 'Rejected'}, {id: 'withdrawn', label: 'Withdrawn'},
  {id: 'expired', label: 'Expired'}, {id: 'unknown', label: 'Ending not on record'},
];
const DIRECTIONS = [{id: 'all', label: 'Every offer'}, {id: 'sent', label: 'Sent'}, {id: 'received', label: 'Received'}];

// The offers in view and their acceptance in each stretch of the year. Acceptance counts only
// offers whose ending is on record; a stretch is called out only when it beats every other beyond chance.
function offerStretches(data, local, from, to) {
  const owner = (() => {
    const c = new Map();
    for (const o of data.offers) for (const f of [o.by, o.to]) c.set(f, (c.get(f) || 0) + 1);
    return [...c.entries()].sort((a, b) => b[1] - a[1])[0]?.[0];
  })();
  const windowOf = Object.fromEntries(PHASES.map(p => [p.id, p.window]));
  const dir = local.offerDir || 'all';
  const offers = data.offers.filter(o => o.season >= from && o.season <= to && o.phase in windowOf
    && (!local.windows || local.windows.includes(windowOf[o.phase]))
    && (dir === 'all' || (dir === 'sent' ? o.by === owner : o.to === owner))
    && (!local.team || local.team === owner || o.by === local.team || o.to === local.team));
  const seasons = new Set(data.offers.filter(o => o.season >= from && o.season <= to).map(o => o.season)).size || 1;
  const stretches = PHASES.filter(p => !local.windows || local.windows.includes(p.window)).map(p => {
    const xs = offers.filter(o => o.phase === p.id);
    const counts = Object.fromEntries(OUTCOMES.map(x => [x.id, xs.filter(o => o.outcome === x.id).length]));
    const known = xs.length - counts.unknown;
    const rate = known ? counts.accepted / known : null;
    return {phase: p, n: xs.length, perSeason: xs.length / seasons, counts, known, rate,
      se: known ? Math.sqrt(Math.max(rate * (1 - rate), 1e-9) / known) : null};
  });
  const usable = stretches.filter(s => s.known >= 10);
  if (usable.length >= 2) {
    const hi = usable.reduce((a, b) => (b.rate > a.rate ? b : a)), lo = usable.reduce((a, b) => (b.rate < a.rate ? b : a));
    if (usable.every(o => o === hi || hi.rate - o.rate > 1.96 * Math.hypot(hi.se, o.se))) hi.best = true;
    if (usable.every(o => o === lo || o.rate - lo.rate > 1.96 * Math.hypot(lo.se, o.se))) lo.worst = true;
  }
  const known = offers.filter(o => o.outcome !== 'unknown');
  return {owner, dir, offers, stretches, seasons,
    rate: known.length ? known.filter(o => o.outcome === 'accepted').length / known.length : null};
}

function offersChart(ctx, data, local, from, to) {
  if (!data.offers.length) return null;
  const v = offerStretches(data, local, from, to);
  const ownerName = franchiseName(data, v.owner);
  const max = Math.max(...v.stretches.map(s => s.perSeason), 0.01);
  const who = v.dir === 'sent' ? `sent by the ${ownerName}` : v.dir === 'received' ? `sent to the ${ownerName}` : `to and from the ${ownerName}`;
  const partner = local.team && local.team !== v.owner ? ` with the ${franchiseName(data, local.team)}` : '';
  const best = v.stretches.find(s => s.best), worst = v.stretches.find(s => s.worst);
  const say = !v.offers.length ? 'No offers in view.'
    : [`${fmt.n(v.offers.length)} offers ${who}${partner}; ${pct(v.rate)} of those with a known ending were accepted.`,
      best ? `Most often accepted: ${best.phase.label} (${pct(best.rate)}), beyond chance against every other stretch.` : null,
      worst ? `Least often accepted: ${worst.phase.label} (${pct(worst.rate)}), beyond chance against every other stretch.` : null,
      !best && !worst ? 'No stretch’s acceptance differs from the others beyond chance.' : null].filter(Boolean).join(' ');
  const col = s => h('div', {class: 'of-col', title: `${s.phase.label}: ${s.n} offers, ${OUTCOMES.map(x => `${s.counts[x.id]} ${x.label.toLowerCase()}`).join(', ')}.${s.rate != null ? ` Accepted ${pct(s.rate)} of ${s.known} with a known ending.` : ''}`},
    h('span', {class: `of-rate num${s.best ? ' up' : s.worst ? ' down' : ''}`}, s.known >= 5 ? pct(s.rate) : '—'),
    h('span', {class: 'of-rate-label'}, s.known >= 5 ? 'accepted' : 'too few'),
    h('span', {class: 'of-bar-wrap'}, h('span', {class: 'of-bar', style: {height: `${Math.round((s.perSeason / max) * 110)}px`}},
      OUTCOMES.map(x => (s.counts[x.id] ? h('span', {class: `of-${x.id}`, style: {flexGrow: s.counts[x.id]}}) : null)))),
    h('span', {class: 'of-n num'}, `${fmt.n(s.perSeason, 1)} a season`),
    h('span', {class: 'yr-label'}, s.phase.label));
  return h('section', {class: 'dm', 'aria-label': 'Offers against acceptance'},
    h('div', {class: 'of-head'}, h('h2', {}, 'Offers against acceptance, through the year'),
      segmented(DIRECTIONS, v.dir, d => ctx.setLocal({offerDir: d}), 'Which offers')),
    h('p', {class: 'mk-note'}, `MFL shows a team only its own offers, so these are the ${ownerName}${ownerName.endsWith('s') ? '’' : '’s'} offers, sent and received: the one window on how the league’s GMs answer an offer. The seasons, time of year and team filters above apply.`),
    h('p', {class: 'ap-say'}, say),
    h('div', {class: 'of-grid', style: {gridTemplateColumns: `repeat(${v.stretches.length}, minmax(0, 1fr))`}}, v.stretches.map(col)),
    h('div', {class: 'pkey'}, OUTCOMES.map(x => h('span', {class: `okey-${x.id}`}, h('i', {}), x.label))));
}

// ---------- the log ----------

function log(ctx, data, trades, local) {
  const search = h('input', {type: 'search', class: 'search', placeholder: 'Player or team name', value: local.q, 'aria-label': 'Search trades'});
  const countEl = h('span', {class: 'muted small'});
  const list = h('div', {});
  const box = h('section', {class: 'dm', 'aria-label': 'The trades'}, h('h2', {}, 'The trades'),
    h('div', {class: 'log-tools'}, search, segmented(ORDER, local.order, v => ctx.setLocal({order: v}), 'Order'), countEl), list);
  const text = t => {
    t._text ??= [franchiseName(data, t.a), franchiseName(data, t.b),
      ...t.legs.map(l => (l.player ? playerName(l.player) : '')), ...t.legs.map(l => (l.pickPlayer ? playerName(l.pickPlayer) : ''))].join(' ').toLowerCase();
    return t._text;
  };
  const draw = () => {
    const q = local.q.trim().toLowerCase();
    let rows = trades.filter(t => !q || q.split(/\s+/).every(part => text(t).includes(part)));
    if (local.order === 'edge') rows = rows.filter(t => t.valued).sort((a, b) => Math.abs(b.aNet) - Math.abs(a.aNet));
    else if (local.order === 'swing') rows = rows.filter(t => t.aRealNet != null).sort((a, b) => Math.abs(b.aRealNet) - Math.abs(a.aRealNet));
    else rows.sort((a, b) => b.ts - a.ts);
    countEl.textContent = `${fmt.n(rows.length)} trade${rows.length === 1 ? '' : 's'}`;
    list.replaceChildren();
    if (!rows.length) { list.append(empty(q ? `No trades in view mention “${local.q}”.` : 'No trades in view.')); return; }
    let shown = 0;
    const more = h('button', {class: 'ghost more'});
    const step = () => {
      for (const t of rows.slice(shown, shown + PAGE)) list.insertBefore(tradeCard(data, t), more);
      shown = Math.min(rows.length, shown + PAGE);
      more.hidden = shown >= rows.length;
      more.textContent = `Show more (${fmt.n(rows.length - shown)} left)`;
    };
    more.onclick = step;
    list.append(more);
    step();
  };
  let timer;
  search.addEventListener('input', () => {
    clearTimeout(timer);
    timer = setTimeout(() => { local.q = search.value; ctx.saveLocal(); draw(); }, 150);
  });
  draw();
  return box;
}

function headline(sides, league) {
  const bits = [];
  const by = Object.fromEntries(TIERS.map(t => [t.id, behaviour(sides.filter(s => s.tier === t.id), league)]));
  const c = by.contender, r = by.rebuild;
  if (c.n >= 20 && r.n >= 20) bits.push(`Contenders buy for now in ${pct(c.posture.now.p)} of their trades and rebuilders build for later in ${pct(r.posture.later.p)}; most trades on both sides are like for like.`);
  const winners = TIERS.filter(t => by[t.id].n >= 20 && by[t.id].now.tone !== 'even');
  bits.push(winners.length ? winners.map(t => `${TIER[t.id].label} ${by[t.id].now.tone === 'up' ? 'come out ahead' : 'come out behind'} on the day (${wins(by[t.id].now.mean)} a trade).`).join(' ')
    : 'No tier wins its trades on the day beyond chance: the edge is in the teams, not the standings.');
  return bits.join(' ');
}

export default {
  id: 'tradelog', nav: 'Trade log', hint: 'How every team trades, and every trade',
  title: 'How the teams trade',
  lede: null,
  render(root, ctx) {
    const {data} = ctx;
    const local = ctx.local({q: '', order: 'new', team: '', tiers: null, windows: null, postures: null, sort: 'edge', offerDir: 'all'});
    if (!DIRECTIONS.some(d => d.id === local.offerDir)) local.offerDir = 'all';
    const valid = (sel, ids) => (Array.isArray(sel) && sel.some(x => ids.includes(x)) ? ids.filter(x => sel.includes(x)) : null);
    local.tiers = valid(local.tiers, TIERS.map(t => t.id));
    local.windows = valid(local.windows, WINDOWS.map(w => w.id));
    local.postures = valid(local.postures, POSTURES.map(p => p.id));
    if (local.team && !data.franchises.has(local.team)) local.team = '';
    const from = Math.max(ctx.from, 2017), to = ctx.to;
    const scope = {from, to, windows: local.windows, postures: local.postures};
    const league = sidesInScope(data, {from, to});
    const tierless = sidesInScope(data, {...scope, team: local.team || null});
    const sides = tierless.filter(s => !local.tiers || local.tiers.includes(s.tier));
    const seasonsN = new Set(league.map(s => s.season)).size || 1;
    const counts = Object.fromEntries(TIERS.map(t => [t.id, tierless.filter(s => s.tier === t.id).length / seasonsN]));
    root.append(h('section', {class: 'flt', 'aria-label': 'What to look at'}, tierPicker(ctx, local, counts, `${from}–${to}`), sidePicker(ctx, local, data)));

    if (local.team) root.append(h('div', {class: 'ap-list'}, teamPanel(ctx, data, teamProfile(data, sides, league, local.team), local)));
    const all = sidesInScope(data, scope).filter(s => !local.tiers || local.tiers.includes(s.tier));
    if (all.length < 40) {
      root.append(empty('Too few trades in view to compare. Widen the standing, time of year or what the trades did.'));
    } else {
      root.append(h('h2', {class: 'mk-league'}, local.team ? 'Every team, in the same trades' : 'The tiers'));
      root.append(h('p', {class: 'mk-headline'}, headline(all, league)));
      root.append(h('p', {class: 'mk-unit-line'}, `Each trade has two sides, one per team. ${fmt.n(all.length)} sides of valued trades, ${from}–${to}. On the day is expected wins gained at the league’s own prices; since is wins actually delivered, for trades with two or more seasons played.`));
      root.append(tierCards(all, league));
      const trends = tierTrends(all);
      root.append(h('section', {class: 'dm', 'aria-label': 'How the tiers’ habits moved'}, h('h2', {}, 'How the tiers’ habits moved'),
        h('p', {class: 'ap-say'}, trendSay(trends)), trendTable(trends)));
      root.append(h('section', {class: 'dm', 'aria-label': 'Who trades with whom'}, h('h2', {}, 'Who trades with whom'),
        h('p', {class: 'mk-note'}, 'Each cell: trades between the row’s tier and the column’s, and what the row’s team came away with on the day, per trade. Coloured only when it clears chance.'),
        pairTable(tierPairs(all))));
      const offers = offersChart(ctx, data, local, from, to);
      if (offers) root.append(offers);
      root.append(h('section', {class: 'dm', 'aria-label': 'The teams'}, h('h2', {}, 'The teams'),
        h('p', {class: 'mk-note'}, 'Every franchise’s trading in view. Click a column to sort, a row for the team in depth. Shares are coloured where they differ from the league’s beyond chance; on the day and since are totals, coloured where the per-trade average clears chance.'),
        h('div', {class: 'pkey'}, h('span', {class: 'muted'}, 'Standing when trading:'), TIERS.map(t => h('span', {class: `tkey-${t.id}`}, h('i', {}), t.label))),
        teamTable(ctx, data, teamRows(all, league), local)));
    }
    // The trades behind everything above: valued trades with a side in view, or every trade when nothing is narrowed.
    const narrowed = local.tiers || local.windows || local.postures;
    const inView = new Set((local.team ? sides : all).map(s => s.t));
    const trades = narrowed || local.team
      ? [...inView]
      : data.trades.filter(t => (t.season ?? t.year) >= ctx.from && (t.season ?? t.year) <= ctx.to);
    root.append(log(ctx, data, trades, local));
    root.append(h('div', {class: 'mk-how'}, howCounted(
      'A team’s tier is its rank on the day of the trade: its live standing once three games are final, otherwise last season’s finish. Contenders are ranked 1 to 10, teams in the hunt 11 to 22, rebuilding teams 23 to 32. Rank is about where a team stood, not what it intended; what it intended shows in what it did.',
      'What a trade did is read from each side’s flow: bought for now when it took in more player value than it sent and sent more pick value than it took, built for later the reverse, like for like otherwise. Value is forecast wins on the day; a flow under 0.05 counts as even.',
      'On the day is the expected wins a side gained at the league’s own prices; since is the wins its assets actually delivered against the other side’s, for trades with two or more seasons played. Every range is the 95% likely range of the average per trade.',
    )));
  },
};
