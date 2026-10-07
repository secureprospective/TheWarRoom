// What the league misprices: a buy list and a sell list, each kind of asset's cost per delivered
// win against the going rate, for the stretch of the year and the assets he picks above them.

import {GROUPS, GROUP_PLURAL, WINDOWS, WINDOW_LABEL, PHASES, AGES, kindLabel, fmt} from '../model.js';
import {leagueMarket, tradesByPhase, pickClock, contenderTrades, inScope, evidence, assetProfile, stretchPrices, assetOf, ANCHOR, MIN_TRADES as MIN} from '../engine.js';
import {h, s, howCounted, empty, segmented} from '../ui.js';

const ROUND = {1: '1sts', 2: '2nds', 3: '3rds and later'};
const AGE_NAME = Object.fromEntries(AGES.map(a => [a.id, a.label]));
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

// A kind of asset as the lists name it: "Linebackers", "Linebackers, 25 to 27", "Next-draft 1sts".
function kindName(kind) {
  if (kind.startsWith('pick')) {
    const [k, when] = kind.split(':');
    return `${when === 'next' ? 'Next' : 'Later'}-draft ${ROUND[k.slice(4)]}`;
  }
  const [group, age] = kind.split(':');
  const plural = GROUP_PLURAL[group];
  return plural[0].toUpperCase() + plural.slice(1) + (age ? `, ${AGE_NAME[age]}` : '');
}

const pct = v => `${fmt.n(v * 100)}%`;
const Z = 1.96;

// Cost against the going rate on a log scale: the dashed mark is the going rate, the line the
// likely range, the dot the estimate.
const SCALE = {lo: 0.2, hi: 5, ticks: [[0.25, '25%'], [0.5, '50%'], [1, 'going rate'], [2, '200%'], [4, '400%']]};
const BAR = {w: 420, pad: 22};
const barX = v => BAR.pad + (Math.log(Math.min(SCALE.hi, Math.max(SCALE.lo, v)) / SCALE.lo) / Math.log(SCALE.hi / SCALE.lo)) * (BAR.w - 2 * BAR.pad);
function costBar(c) {
  const svg = s('svg', {viewBox: `0 0 ${BAR.w} 24`, class: 'costbar', role: 'img',
    'aria-label': `${pct(c.value)} of the going rate, likely between ${pct(c.lo)} and ${pct(c.hi)}`});
  svg.append(s('line', {x1: BAR.pad, x2: BAR.w - BAR.pad, y1: 12, y2: 12, class: 'axis'}));
  for (const [v] of SCALE.ticks) svg.append(s('line', {x1: barX(v), x2: barX(v), y1: v === 1 ? 2 : 9, y2: v === 1 ? 22 : 15, class: v === 1 ? 'going' : 'tick'}));
  svg.append(s('line', {x1: barX(c.lo), x2: barX(c.hi), y1: 12, y2: 12, class: 'range'}));
  svg.append(s('circle', {cx: barX(c.value), cy: 12, r: 5, class: 'dot'}));
  return svg;
}

// The scale once, at the head of each list.
function scaleRow() {
  const svg = s('svg', {viewBox: `0 0 ${BAR.w} 16`, class: 'costbar scale', 'aria-hidden': 'true'});
  for (const [v, label] of SCALE.ticks) svg.append(s('text', {x: barX(v), y: 12, 'text-anchor': 'middle', class: v === 1 ? 'going-label' : 'tick-label'}, label));
  return h('div', {class: 'mk-scale'}, svg);
}

function halfLabel(m, i) {
  const [a, b] = i === 0 ? [m.seasons[0], m.mid] : [m.mid + 1, m.seasons.at(-1)];
  return a === b ? `${a}` : `${a}–${String(b).slice(2)}`;
}

// How the finding stands in each half on its own, the recent half first.
function trend(m, r) {
  const [a, b] = r.halves.map(x => x?.cost?.value);
  const span = t => h('span', {class: 'nowrap'}, t);
  const early = a != null ? span(`${pct(a)} in ${halfLabel(m, 0)}`) : null, late = b != null ? span(`${pct(b)} in ${halfLabel(m, 1)}`) : null;
  const join = (xs, sep) => xs.filter(Boolean).flatMap((x, k) => (k ? [sep, x] : [x]));
  if (r.trend === 'holds') return {text: ['Clear in both halves: ', ...join([late, early], ', ')], cls: 'mk-holds'};
  if (r.trend === 'new') return {text: ['Clear only lately: ', ...join([late, early], ', ')], cls: 'mk-once'};
  if (r.trend === 'fading') return {text: ['Fading: ', ...join([early, late], ', then ')], cls: 'mk-once'};
  if (a == null && b == null) return {text: ['Too few trades in view to check each half on its own'], cls: 'mk-once'};
  const weaker = a != null && b != null && Math.abs(Math.log(b)) < Math.abs(Math.log(a));
  return {text: [weaker ? 'Weaker lately: ' : 'Not clear in either half alone: ', ...join(weaker ? [early, late] : [late, early], weaker ? ', then ' : ', ')], cls: 'mk-once'};
}

function strip(r) {
  // Only worth drawing when the stretches in view reach more than one time of year.
  if (r.times.length < 2) return null;
  // Buy lines mark when the league sells cheapest; sell lines when it pays most.
  const mark = r.side === 'cheap' ? r.cheapest : r.dearest;
  return h('span', {class: 'mk-strip'}, r.times.map(t => h('span', {class: `mk-cell${t.window === mark ? ' marked' : ''}`},
    h('span', {class: 'mk-cell-when'}, WINDOW_LABEL[t.window]),
    h('span', {class: 'mk-cell-cost num'}, t.cost != null ? pct(t.cost) : '—'))));
}

// The trades behind a row, in the seasons and stretches of the year in view.
function openRow(ctx, m, r) {
  const keep = inScope(m);
  const legs = ctx.data.legs.filter(l => l.xClass && (l.xClass === r.kind || l.xClass.split(':')[0] === r.kind) && l.t.valued && keep(l.t));
  ctx.openTrades(kindName(r.kind), legs.map(l => l.t), {legs, note: 'the asset of this kind is outlined'});
}

function line(ctx, m, r, whole) {
  const tr = trend(m, r);
  const year = whole?.rows?.find(x => x.kind === r.kind)?.cost;
  return h('button', {class: `mk-line ${r.side}${m.focus?.has(r.kind) ? ' focus' : ''}`, onclick: () => openRow(ctx, m, r)},
    h('span', {class: 'mk-name'}, kindName(r.kind)),
    h('span', {class: 'mk-cost num'}, pct(r.cost.value), h('span', {class: 'mk-unit'}, ' of the going rate')),
    h('span', {class: 'mk-bar'}, costBar(r.cost)),
    strip(r),
    h('span', {class: 'mk-meta'}, h('span', {class: tr.cls}, tr.text, year ? [' · ', h('span', {class: 'mk-year'}, `all year ${pct(year.value)}`)] : null),
      h('span', {class: 'mk-see'}, `See the ${fmt.n(r.n)} trades`)));
}

function list(ctx, m, rows, title, note, cls, whole) {
  const body = rows.length ? [scaleRow(), ...rows.map(r => line(ctx, m, r, whole))] : [empty('Nothing in view clears chance on this side.')];
  return h('section', {class: `mk-list ${cls}`, 'aria-label': title},
    h('h2', {}, title), h('p', {class: 'mk-note'}, note), h('div', {class: 'mk-lines'}, body));
}

function middle(ctx, m, rows) {
  const judged = rows.filter(r => r.side === 'fair' && evidence(r.range) !== 'thin').sort((a, b) => a.cost.value - b.cost.value);
  const thin = rows.filter(r => !r.cost || (r.side === 'fair' && evidence(r.range) === 'thin'))
    .sort((a, b) => (b.paid?.value ?? 0) - (a.paid?.value ?? 0));
  const item = (r, figure) => h('button', {class: 'mk-item', onclick: () => openRow(ctx, m, r), title: `See the ${fmt.n(r.n)} trades`},
    `${kindName(r.kind)} `, h('span', {class: 'num muted'}, figure));
  return h('div', {class: 'mk-middle'},
    judged.length ? h('div', {class: 'mk-fair'}, h('strong', {}, 'Priced about right ', h('span', {class: 'muted mk-why'}, 'each likely range includes the going rate')),
      judged.map(r => item(r, pct(r.cost.value)))) : null,
    thin.length ? h('div', {class: 'mk-fair'}, h('strong', {}, 'Too uncertain to judge ',
      h('span', {class: 'muted mk-why'}, 'too few have played out, or what they delivered varies too much; the figure is what GMs paid per forecast win, against the typical asset')),
      thin.map(r => item(r, r.paid ? `paid ${pct(r.paid.value)}` : ''))) : null);
}

function headline(rows) {
  const sure = rows.filter(r => r.holds);   // clear in both halves on their own
  const cheap = sure.filter(r => r.side === 'cheap').sort((a, b) => a.cost.value - b.cost.value)[0];
  const dear = sure.filter(r => r.side === 'dear').sort((a, b) => b.cost.value - a.cost.value)[0];
  const one = (r, what) => `The clearest ${what} in view: ${kindName(r.kind).toLowerCase()}. A delivered win costs ${pct(r.cost.value)} of the going rate, in both halves of these seasons.`;
  if (cheap && !dear) return one(cheap, 'buy');
  if (dear && !cheap) return one(dear, 'sell');
  if (!cheap || !dear) return 'Nothing in view is clearly too cheap or too dear in both halves of these seasons.';
  const x = dear.cost.value / cheap.cost.value;
  return `A win bought with ${kindName(dear.kind).toLowerCase()} costs ${fmt.n(x, x < 10 ? 1 : 0)} times one bought with ${kindName(cheap.kind).toLowerCase()}.`;
}

// ---------- what to look at: the stretch of the year, the assets ----------

// A selection is null (everything) or a list. Clicking from everything picks just that one; after
// that each click adds or removes it, and emptying it or choosing everything returns to null.
function toggle(sel, id, all) {
  if (!sel) return [id];
  const next = sel.includes(id) ? sel.filter(x => x !== id) : [...sel, id];
  return !next.length || next.length === all.length ? null : all.filter(x => next.includes(x));
}
const same = (sel, ids) => !!sel && sel.length === ids.length && ids.every(x => sel.includes(x));

const POSITIONS = GROUPS.map(g => g.id);
const PICKS = ['pick1:next', 'pick2:next', 'pick3:next', 'pick1:later', 'pick2:later', 'pick3:later'];
const ASSETS = [...POSITIONS, ...PICKS];
const PICK_SHORT = {1: '1st', 2: '2nd', 3: '3rd+'};

function monthSpan([a, b]) {
  return a === b ? MONTHS[a] : `${MONTHS[a]}–${MONTHS[b]}`;
}

// The league year, stretch by stretch: how much the league trades in each, and which are in view.
function yearPicker(ctx, st, byPhase) {
  const all = PHASES.map(p => p.id);
  const set = phases => ctx.setLocal({phases});
  const max = Math.max(...PHASES.map(p => byPhase.phases.get(p.id)?.perSeason || 0));
  // Nothing narrowed: every stretch plain. Narrowed: the chosen ones lit, the rest dimmed.
  const state = id => (!st.phases ? '' : st.phases.includes(id) ? ' on' : ' off');
  const heads = WINDOWS.map(w => {
    const ids = PHASES.filter(p => p.window === w.id).map(p => p.id);
    const chosen = same(st.phases, ids);
    return h('button', {class: `yr-window${chosen ? ' on' : ''}`, style: {gridColumn: `span ${ids.length}`}, 'aria-pressed': String(chosen),
      title: w.hint, onclick: () => set(chosen ? null : ids)}, w.label);
  });
  const cols = PHASES.map(p => {
    const c = byPhase.phases.get(p.id);
    const per = c?.perSeason || 0;
    return h('button', {class: `yr-phase${state(p.id)}`, 'aria-pressed': String(!!st.phases && st.phases.includes(p.id)),
      title: `${p.hint} ${fmt.n(per, 1)} completed trades a season.`, onclick: () => set(toggle(st.phases, p.id, all))},
      h('span', {class: 'yr-bar-wrap'}, h('span', {class: 'yr-count num'}, fmt.n(per)), h('span', {class: 'yr-bar', style: {height: `${max ? Math.max(2, Math.round((per / max) * 52)) : 0}px`}})),
      h('span', {class: 'yr-label'}, p.label),
      h('span', {class: 'yr-months'}, c ? monthSpan(c.span) : '—'));
  });
  return h('div', {class: 'yr'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Time of year'),
      h('span', {class: 'flt-note'}, `${byPhase.assets ? `Trades a season that moved ${scopeWords({assets: byPhase.assets}).split('; ')[1]}` : 'Completed trades a season'} in each stretch, ${byPhase.from}–${byPhase.to}. Each stretch follows that year’s own draft, kickoff and deadline dates.`),
      h('button', {class: `flt-all${st.phases ? '' : ' on'}`, 'aria-pressed': String(!st.phases), onclick: () => set(null)}, 'All year')),
    h('div', {class: 'yr-grid'}, heads, cols));
}

function assetPicker(ctx, st) {
  const set = assets => ctx.setLocal({assets});
  const state = id => (!st.assets ? '' : st.assets.includes(id) ? ' on' : ' off');
  const chip = (id, label, hint) => h('button', {class: `chip${state(id)}`, 'aria-pressed': String(!!st.assets && st.assets.includes(id)), title: hint,
    onclick: () => set(toggle(st.assets, id, ASSETS))}, label);
  const preset = (label, ids) => {
    const chosen = ids ? same(st.assets, ids) : !st.assets;
    return h('button', {class: `flt-all${chosen ? ' on' : ''}`, 'aria-pressed': String(chosen), onclick: () => set(chosen || !ids ? null : ids)}, label);
  };
  const group = (title, ...chips) => h('div', {class: 'as-group', role: 'group', 'aria-label': title}, h('span', {class: 'as-title'}, title), chips);
  return h('div', {class: 'as'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Assets'),
      h('span', {class: 'flt-presets'}, preset('Everything', null), preset('Players', POSITIONS), preset('Picks', PICKS))),
    h('div', {class: 'as-rows'},
      group('Players', GROUPS.map(g => chip(g.id, g.short, g.hint || g.label))),
      group('Next draft', [1, 2, 3].map(r => chip(`pick${r}:next`, PICK_SHORT[r], kindLabel(`pick${r}:next`)))),
      group('Later drafts', [1, 2, 3].map(r => chip(`pick${r}:later`, PICK_SHORT[r], kindLabel(`pick${r}:later`)))),
      group('Ages', segmented([{id: false, label: 'Together'}, {id: true, label: 'Split by age', hint: '24 or younger, 25 to 27, 28 or older'}],
        st.byAge, v => ctx.setLocal({byAge: v}), 'Players by age'))));
}

// The scope in words: "in season, linebackers and next-draft 1sts".
function scopeWords(st) {
  const time = !st.phases ? 'all year'
    : WINDOWS.find(w => same(st.phases, PHASES.filter(p => p.window === w.id).map(p => p.id)))?.label.toLowerCase()
      ?? PHASES.filter(p => st.phases.includes(p.id)).map(p => p.label.toLowerCase()).join(', ');
  const assets = !st.assets ? 'every asset' : same(st.assets, POSITIONS) ? 'players' : same(st.assets, PICKS) ? 'picks'
    : st.assets.map(a => kindName(a).toLowerCase()).join(', ');
  return `${time}; ${assets}`;
}

// The pick clock in the trades in view: what a 1st a year before its draft fetches against the
// same round once its draft is next.
function clockNote(clock) {
  const known = clock?.halves.filter(hv => hv.clock && hv.clock.value > 0) ?? [];
  if (!known.length) return null;
  const each = known.map(hv => h('span', {class: 'nowrap'}, `${fmt.n(1 / hv.clock.value, 1)} times in ${hv.from}–${String(hv.to).slice(2)}${hv.clock.clear ? '' : ' (within chance)'}`));
  const clear = known.length === clock.halves.length && known.every(hv => hv.clock.clear);
  return h('div', {class: 'mk-clock'},
    h('h3', {}, 'The pick clock'),
    h('p', {class: 'say'}, 'In these trades a next-draft 1st sells for ', each.flatMap((e, k) => (k ? [' and ', e] : [e])),
      ' what the same pick fetches a year before its draft, per expected win, after allowing for the year’s wait.',
      clear ? ' Buy 1sts a year out; sell them once their draft is next.' : ''));
}

// ---------- one asset's own market ----------

const PHASE = Object.fromEntries(PHASES.map(p => [p.id, p]));
const times = v => `${fmt.n(v, v < 1 ? 2 : v < 10 ? 1 : 0)}×`;
const clearOf = r => (r.hi < 1 ? 'low' : r.lo > 1 ? 'high' : 'even');
const plural = kind => kindName(kind).toLowerCase();
// A stretch's name inside a sentence: lower case, except the NFL.
const inline = label => (label.startsWith('NFL') ? label : label[0].toLowerCase() + label.slice(1));

// The legs of one kind in the trades in view, optionally in one stretch of the year.
function openKind(ctx, scope, kind, title, phase = null) {
  const keep = inScope({...scope, phases: phase ? [phase] : scope.phases});
  const legs = ctx.data.legs.filter(l => l.xClass && (l.xClass === kind || l.xClass.split(':')[0] === kind) && l.t.valued && keep(l.t));
  ctx.openTrades(title, legs.map(l => l.t), {legs, note: 'the asset of this kind is outlined'});
}

// Why it costs what it costs: what GMs pay for a forecast win, and what share of the forecast it
// delivers. Cost per delivered win is the first over the second.
function whyBlock(a, row) {
  const name = kindName(a.kind);
  const lines = [];
  if (a.paid) {
    const p = a.paid, range = p.hi > p.lo ? ` (likely ${pct(p.lo)} to ${pct(p.hi)})` : '';
    const c = a.kind === ANCHOR ? 'priced' : clearOf(p);
    lines.push(h('p', {class: `ap-factor ${c === 'low' ? 'good' : c === 'high' ? 'bad' : ''}`},
      c === 'low' ? `GMs pay less for its forecast wins than for the typical asset’s: ${pct(p.value)} of the typical price${range}.`
        : c === 'high' ? `GMs pay a premium for its forecast wins: ${pct(p.value)} of the typical asset’s price${range}.`
          : c === 'priced' ? `GMs pay ${pct(p.value)} of the typical asset’s price for its forecast wins. Every other price is measured against this one.`
            : `GMs pay about the typical price for its forecast wins: ${pct(p.value)}${range}.`));
  }
  const d = a.delivered;
  if (d) {
    const r = {lo: d.ratio - Z * d.se, hi: d.ratio + Z * d.se}, c = clearOf(r);
    lines.push(h('p', {class: `ap-factor ${c === 'high' ? 'good' : c === 'low' ? 'bad' : ''}`},
      c === 'high' ? `${name} deliver more than forecast: ${pct(d.ratio)} of the wins expected when traded (${fmt.n(d.n)} played out).`
        : c === 'low' ? `${name} fall short of forecast: ${pct(d.ratio)} of the wins expected when traded (${fmt.n(d.n)} played out).`
          : `${name} deliver about what was forecast: ${pct(d.ratio)} (${fmt.n(d.n)} played out).`));
  } else {
    lines.push(h('p', {class: 'ap-factor muted'}, `Too few ${plural(a.kind)} traded in view have played out to say what they deliver.`));
  }
  if (row?.cost) lines.push(h('p', {class: 'ap-sum'}, `So a delivered win through ${plural(a.kind)} costs ${pct(row.cost.value)} of the going rate`,
    row.side === 'cheap' ? ': the league sells them cheap.' : row.side === 'dear' ? ': the league overpays.' : ', within chance of it.'));
  // Who takes it on, against who takes on anything: named only when the gap clears chance.
  const b = a.buyers;
  const gap = ['rebuilder', 'contender'].map(k => ({k, mine: b.mine[k], all: b.all[k],
    z: (b.mine[k] - b.all[k]) / Math.sqrt(Math.max(1e-9, b.all[k] * (1 - b.all[k]) / Math.max(1, b.n)))}))
    .sort((x, y) => Math.abs(y.z) - Math.abs(x.z))[0];
  const who = {rebuilder: 'Rebuilders (bottom 10 at the time)', contender: 'Contenders (top 10 at the time)'}[gap.k];
  lines.push(h('p', {class: 'ap-factor'}, Math.abs(gap.z) > Z
    ? `${who} take ${pct(gap.mine)} of them, against ${pct(gap.all)} of all assets traded.`
    : `Contenders, middle teams and rebuilders take them in about their usual shares.`));
  return h('div', {class: 'ap-cell ap-why'}, h('h3', {}, 'Why'), lines);
}

// When it is cheapest: what GMs paid in each stretch, against that stretch's typical asset.
function whenBlock(ctx, scope, a, st) {
  const known = a.when.filter(w => w.paid);
  const lo = known.length ? known.reduce((x, y) => (y.paid.value < x.paid.value ? y : x)) : null;
  const hi = known.length ? known.reduce((x, y) => (y.paid.value > x.paid.value ? y : x)) : null;
  const named = w => `${inline(PHASE[w.phase].label)} (${pct(w.paid.value)})`;
  const marked = a.when.filter(w => w.cheapest || w.dearest);
  const say = known.length < 2 ? 'Too few trades in most stretches to compare.'
    : marked.length ? [lo.cheapest ? `Cheapest: ${named(lo)}.` : null, hi.dearest ? `Dearest: ${named(hi)}.` : null].filter(Boolean).join(' ') + ' Each beats every other stretch beyond chance.'
      : `Lowest: ${named(lo)}. Highest: ${named(hi)}. No stretch beats the others beyond chance.`;
  const cells = a.when.map(w => {
    const out = st.phases && !st.phases.includes(w.phase);
    return h('button', {class: `ap-when-cell${w.cheapest ? ' cheapest' : ''}${w.dearest ? ' dearest' : ''}${out ? ' out' : ''}`,
      disabled: !w.n || null, title: w.paid ? `${PHASE[w.phase].hint} Likely ${pct(w.paid.lo)} to ${pct(w.paid.hi)}. See the trades.` : PHASE[w.phase].hint,
      onclick: () => openKind(ctx, {...scope, phases: null}, a.kind, `${kindName(a.kind)}, ${inline(PHASE[w.phase].label)}`, w.phase)},
      h('span', {class: 'ap-when-label'}, PHASE[w.phase].label),
      h('span', {class: 'ap-when-value num'}, w.paid ? pct(w.paid.value) : '—'),
      h('span', {class: 'ap-when-n'}, w.paid ? `${fmt.n(w.n)} traded` : w.n ? `only ${fmt.n(w.n)} traded` : 'none traded'));
  });
  return h('div', {class: 'ap-cell ap-when'}, h('h3', {}, 'When it is cheapest'),
    h('p', {class: 'ap-say'}, say),
    h('div', {class: 'ap-when-grid'}, cells),
    h('p', {class: 'mk-note'}, 'What GMs paid per forecast win in each stretch, against the typical asset traded in the same stretch, every stretch of the seasons in view. A marked stretch beats every other beyond chance. Click a stretch for its trades.'));
}

// Against what: a delivered win through this kind against one through every other kind.
const RATIO = {lo: 0.1, hi: 10, w: 260, pad: 8};
const ratioX = v => RATIO.pad + (Math.log(Math.min(RATIO.hi, Math.max(RATIO.lo, v)) / RATIO.lo) / Math.log(RATIO.hi / RATIO.lo)) * (RATIO.w - 2 * RATIO.pad);
function ratioBar(x) {
  const svg = s('svg', {viewBox: `0 0 ${RATIO.w} 18`, class: 'ratiobar', role: 'img', 'aria-label': `${times(x.ratio)}, likely ${times(x.lo)} to ${times(x.hi)}`});
  svg.append(s('line', {x1: RATIO.pad, x2: RATIO.w - RATIO.pad, y1: 9, y2: 9, class: 'axis'}));
  svg.append(s('line', {x1: ratioX(1), x2: ratioX(1), y1: 1, y2: 17, class: 'going'}));
  svg.append(s('line', {x1: ratioX(x.lo), x2: ratioX(x.hi), y1: 9, y2: 9, class: 'range'}));
  svg.append(s('circle', {cx: ratioX(x.ratio), cy: 9, r: 4, class: 'dot'}));
  return svg;
}
function againstBlock(a) {
  // Long lists (players split by age) name only the widest gaps and fold the even rows away.
  const say1 = (xs, word) => {
    if (!xs.length) return null;
    if (xs.length <= 5) return `${word} than ${xs.map(x => kindName(x.kind).toLowerCase()).join(', ')}.`;
    const widest = [...xs].sort((x, y) => Math.abs(Math.log(y.ratio)) - Math.abs(Math.log(x.ratio))).slice(0, 3);
    return `${word} than ${xs.length} of the ${a.against.length} other kinds, most of all ${widest.map(x => kindName(x.kind).toLowerCase()).join(', ')}.`;
  };
  const cheaper = a.against.filter(x => x.side === 'cheaper'), dearer = a.against.filter(x => x.side === 'dearer');
  const say = [say1(cheaper, 'Cheaper'), say1(dearer, 'Dearer')].filter(Boolean).join(' ') || 'Within chance of every other kind.';
  const row = x => h('div', {class: `ap-row ${x.side}`},
    h('span', {class: 'ap-row-name'}, kindName(x.kind)),
    h('span', {class: 'ap-row-bar'}, ratioBar(x)),
    h('span', {class: 'ap-row-value num'}, times(x.ratio), x.byDelivered ? null : h('span', {class: 'faint'}, ' forecast')));
  const fold = a.against.length > 14;
  const shown = fold ? a.against.filter(x => x.side !== 'even') : a.against;
  const even = fold ? a.against.filter(x => x.side === 'even') : [];
  return h('div', {class: 'ap-cell ap-against'}, h('h3', {}, 'Against everything else'),
    h('p', {class: 'ap-say'}, say),
    h('div', {class: 'ap-rows'}, shown.map(row)),
    even.length ? h('details', {class: 'ap-more'}, h('summary', {}, `${even.length} more within chance of the same price`), h('div', {class: 'ap-rows'}, even.map(row))) : null,
    h('p', {class: 'mk-note'}, `What a delivered win through ${plural(a.kind)} costs, as a multiple of one through each other kind; the mark is “the same”, the line the likely range. “Forecast” compares forecast wins where too few have played out.`));
}

// What actually came back: the other side of every trade that moved it, by forecast wins.
function backBlock(a) {
  const players = a.back.filter(x => !x.kind.startsWith('pick')), picks = a.back.filter(x => x.kind.startsWith('pick'));
  const sum = xs => xs.reduce((t, x) => t + x.share, 0);
  const max = Math.max(...a.back.map(x => x.share), 0.01);
  const rows = xs => xs.filter(x => x.share >= 0.02).map(x => h('div', {class: 'ap-back-row'},
    h('span', {class: 'ap-row-name'}, kindName(x.kind)),
    h('span', {class: 'ap-back-bar'}, h('span', {style: {width: `${(x.share / max) * 100}%`}})),
    h('span', {class: 'ap-row-value num'}, pct(x.share))));
  return h('div', {class: 'ap-cell ap-back'}, h('h3', {}, 'What came back for it'),
    h('p', {class: 'ap-say'}, `In ${fmt.n(a.moved)} trades that moved ${plural(a.kind)}, the other side sent ${pct(sum(players))} of its forecast wins as players and ${pct(sum(picks))} as picks.`),
    h('div', {class: 'ap-back-cols'},
      h('div', {}, h('h4', {}, 'Players'), rows(players)),
      h('div', {}, h('h4', {}, 'Picks'), rows(picks))));
}

function profile(ctx, scope, a, row, st) {
  if (!a?.n) return h('section', {class: 'ap'}, h('header', {class: 'ap-head'}, h('h2', {}, kindName(a?.kind ?? ''))),
    empty('Not traded in the seasons and stretches in view.'));
  return h('section', {class: 'ap', 'aria-label': kindName(a.kind)},
    h('header', {class: 'ap-head'},
      h('h2', {}, kindName(a.kind)),
      row?.cost ? h('span', {class: `ap-cost num ${row.side}`}, pct(row.cost.value), h('span', {class: 'mk-unit'}, ' of the going rate')) : null,
      h('button', {class: 'link ap-see', onclick: () => openKind(ctx, scope, a.kind, kindName(a.kind))}, `See the ${fmt.n(a.n)} traded`)),
    h('div', {class: 'ap-grid top'}, whyBlock(a, row), whenBlock(ctx, scope, a, st)),
    h('div', {class: 'ap-grid'}, againstBlock(a), backBlock(a)));
}

export default {
  id: 'market', nav: 'The market', hint: 'What the league sells too cheap, and pays too much for',
  title: 'What the league misprices',
  lede: null,
  render(main, ctx) {
    const {data} = ctx;
    const st = ctx.local({phases: null, assets: null, byAge: false});
    // A hand-edited address must not narrow to stretches or assets that don't exist.
    const valid = (sel, ids) => (Array.isArray(sel) && sel.some(x => ids.includes(x)) ? ids.filter(x => sel.includes(x)) : null);
    st.phases = valid(st.phases, PHASES.map(p => p.id));
    st.assets = valid(st.assets, ASSETS);
    st.byAge = st.byAge === true;
    const from = Math.max(ctx.from, 2017), to = ctx.to;
    const done = Math.min(to, data.lastFinished);
    const byPhase = {...tradesByPhase(data, {from, to: done, assets: st.assets}), from, to: done, assets: st.assets};
    main.append(h('section', {class: 'flt', 'aria-label': 'What to look at'}, yearPicker(ctx, st, byPhase), assetPicker(ctx, st)));

    const m = leagueMarket(data, {from, to, phases: st.phases, byAge: st.byAge});
    if (!m.rows) {
      main.append(empty(`Only ${fmt.n(m.trades)} valued trades fall in the chosen stretch of the year (${scopeWords({...st, assets: null}).split(';')[0]}) from ${from} to ${to}: too few to price. Add a neighbouring stretch of the year or widen the seasons; trades are valued from 2017, when picks started being recorded.`));
      return;
    }
    // Chosen assets: each one's own market first, then the whole league for context.
    if (st.assets) {
      const scope = {from, to, phases: st.phases, byAge: st.byAge};
      const stretches = stretchPrices(data, scope, PHASES.map(p => p.id));
      const kinds = m.rows.map(r => r.kind).filter(k => st.assets.includes(assetOf(k)));
      main.append(h('div', {class: 'ap-list'}, kinds.map(k => profile(ctx, scope, assetProfile(data, scope, k, stretches), m.rows.find(r => r.kind === k), st))));
      main.append(h('h2', {class: 'mk-league'}, st.phases ? 'The whole league in the same stretch of the year' : 'The whole league, all year'));
    }
    // Narrowed to part of the year, each line also carries its all-year figure.
    const whole = st.phases ? leagueMarket(data, {from, to, byAge: st.byAge}) : null;
    const rows = m.rows;
    m.focus = st.assets ? new Set(rows.map(r => r.kind).filter(k => st.assets.includes(assetOf(k)))) : null;
    const buy = rows.filter(r => r.side === 'cheap').sort((a, b) => a.cost.value - b.cost.value);
    const sell = rows.filter(r => r.side === 'dear').sort((a, b) => b.cost.value - a.cost.value);
    const marks = new Set(m.rows.flatMap(r => r.times.map(t => t.window))).size > 1;
    const boxes = marks ? ' The boxes show what GMs paid at each time of year against everything else traded then; a marked box beats the other times of year beyond chance.' : '';
    main.append(h('p', {class: 'mk-headline'}, headline(rows)));
    main.append(h('p', {class: 'mk-unit-line'}, `100% is the going rate: what one delivered win costs through the typical asset traded in this part of the year. Below 100%, the league sells wins cheap; above it, the league overpays. `,
      h('span', {class: 'nowrap'}, `${fmt.n(m.trades)} trades, ${from}–${to}, `), h('span', {class: 'mk-scope'}, scopeWords({...st, assets: null}).split(';')[0]), '.'));
    main.append(h('div', {class: 'mk-columns'},
      list(ctx, m, buy, 'The league sells these too cheap', `A delivered win costs clearly less than the going rate. Buy these.${boxes}`, 'buy', whole),
      list(ctx, m, sell, 'The league pays too much for these', `A delivered win costs clearly more than the going rate. Sell these.${boxes}`, 'sell', whole)));
    main.append(middle(ctx, m, rows));
    {
      const note = clockNote(pickClock(data, {from, to, phases: st.phases}));
      if (note) main.append(note);
    }
    const deals = contenderTrades(data, {from, to, phases: st.phases});
    if (deals.n >= 30) {
      main.append(h('p', {class: 'mk-aside'}, Math.abs(deals.mean) <= deals.half
        ? `No edge either way when contenders (top 10 at the time) trade with rebuilders (bottom 10): ${fmt.signed(deals.mean, 2)} expected wins per trade to the contender, within chance, across ${fmt.n(deals.n)} trades in view.`
        : `When contenders (top 10) trade with rebuilders (bottom 10), the ${deals.mean > 0 ? 'contender' : 'rebuilder'} comes out ahead by ${fmt.n(Math.abs(deals.mean), 2)} expected wins per trade, across ${fmt.n(deals.n)} trades in view.`));
    }
    main.append(h('div', {class: 'mk-how'},
      howCounted(
        `Wins are wins above replacement: what a player adds over what a team could get for free at his position, in this league’s scoring and lineups. An asset’s cost per delivered win is the price the league paid for it in trades (what each side gave up, at the league’s own prices) over the wins it went on to produce, against a forecast made with only the seasons before the trade.`,
        `The going rate is the cost per delivered win of the median traded asset, priced on the trades in view: ${fmt.n(m.trades)} from ${from} to ${to}, ${scopeWords({...st, assets: null}).split(';')[0]}. Picking assets changes which lines show, never the going rate; splitting players by age prices each age band on its own against the same going rate. A kind is on a list only when its whole likely range sits on one side of the going rate; the bar shows that range on a scale where the dashed mark is the going rate.`,
        `The stretches of the year follow each season’s own dates: the NFL draft, the league’s rookie draft, kickoff, week 4 and the trade deadline. A narrow stretch has fewer trades, so its ranges are wider and fewer kinds clear chance; under ${fmt.n(MIN.whole)} valued trades nothing is priced.`,
        `"Clear in both halves" repeats everything for the earlier and later halves of the seasons in view. A time-of-year box is marked only where the kind traded often enough in each and the difference clears chance; it compares what GMs paid then, against everything else traded at the same time of year.`,
      )));
  },
};
