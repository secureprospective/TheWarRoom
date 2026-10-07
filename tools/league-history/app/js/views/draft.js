// When to take each position: every rookie pick against what its slot returned, sliced by where
// in the draft it was taken, the season after the draft, NFL draft round and age, with the trade
// market's price for the same position once it plays.

import {GROUPS, GROUP_LABEL, GROUP_PLURAL, franchiseName, playerName, fmt} from '../model.js';
import {leagueMarket} from '../engine.js';
import {SEGMENTS, NFL_ROUNDS, AGE_BANDS, SEASONS_AFTER, draftScope, draftMap, positionDraft} from '../draftlab.js';
import {h, segmented, howCounted, empty, openDrawer, paged, download, toCsv} from '../ui.js';

const JUDGE = [2, 3, 4].map(n => ({id: n, label: `First ${n} seasons`}));
const SEG = Object.fromEntries(SEGMENTS.map(s => [s.id, s]));
const NFL = Object.fromEntries(NFL_ROUNDS.map(n => [n.id, n]));
const AGE = Object.fromEntries(AGE_BANDS.map(a => [a.id, a]));
const POSITIONS = GROUPS.map(g => g.id);
const pct = v => `${fmt.n(v * 100)}%`.replace('-', '−');
const picksWord = n => `${fmt.n(n)} pick${n === 1 ? '' : 's'}`;
const nflName = id => (id === 'nx' ? 'No NFL pick' : `NFL ${NFL[id].label}`);
const Name = g => GROUP_PLURAL[g][0].toUpperCase() + GROUP_PLURAL[g].slice(1);
const ordinal = n => `${n}${n % 100 >= 11 && n % 100 <= 13 ? 'th' : ['th', 'st', 'nd', 'rd'][n % 10] || 'th'}`;

// ---------- the records behind a number ----------

function showPicks(ctx, title, picks, basis) {
  const {data} = ctx;
  const list = [...picks].sort((a, b) => b.over - a.over);
  const row = p => h('div', {class: 'pick-row'},
    h('div', {}, h('strong', {}, playerName(p.player)), h('span', {class: 'muted small'}, ` ${p.pos} · ${p.year} pick ${fmt.pick(p.round, p.slot)} · ${franchiseName(data, p.fid, true)}`)),
    h('div', {class: 'num'}, `${fmt.n(p.got, 2)} wins`, h('span', {class: 'muted small'}, ` vs ${fmt.n(p.expected, 2)} for the slot`)),
    h('div', {class: `num ${p.over > 0 ? 'up' : 'down'}`}, fmt.signed(p.over, 2)));
  const csv = () => download(`${title}.csv`, toCsv(list.map(p => ({player: playerName(p.player), position: p.pos, year: p.year,
    pick: fmt.pick(p.round, p.slot), team: franchiseName(data, p.fid), wins: p.got.toFixed(3), slot_wins: p.expected.toFixed(3), over_slot: p.over.toFixed(3)}))));
  openDrawer(title, `${list.length} pick${list.length === 1 ? '' : 's'} · ${basis}`, paged(list, row, 40), csv);
}
const judgedOf = picks => picks.filter(p => p.judged).map(p => ({...p, over: p.got - p.expected}));
const seasonOf = (picks, i) => picks.filter(p => p.wins[i] != null && p.slotWins[i] != null)
  .map(p => ({...p, got: p.wins[i], expected: p.slotWins[i], over: p.wins[i] - p.slotWins[i]}));

// ---------- what to look at ----------

function toggle(sel, id, all) {
  if (!sel) return [id];
  const next = sel.includes(id) ? sel.filter(x => x !== id) : [...sel, id];
  return !next.length || next.length === all.length ? null : all.filter(x => next.includes(x));
}
const lit = (sel, id) => (!sel ? '' : sel.includes(id) ? ' on' : ' off');

// The draft in stretches: what an average pick in each returns, and which are in view.
function draftPicker(ctx, st, picksAll) {
  const all = SEGMENTS.map(s => s.id);
  const avg = SEGMENTS.map(s => {
    const xs = picksAll.filter(p => p.judged && p.segment === s.id);
    return xs.length ? xs.reduce((t, p) => t + p.expected, 0) / xs.length : 0;
  });
  const max = Math.max(...avg, 0.01);
  return h('div', {class: 'yr'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Where in the draft'),
      h('span', {class: 'flt-note'}, `What a pick in each stretch returns in its first ${st.seasons} seasons, every position together.`),
      h('button', {class: `flt-all${st.segments ? '' : ' on'}`, 'aria-pressed': String(!st.segments), onclick: () => ctx.setLocal({segments: null})}, 'Every pick')),
    h('div', {class: 'yr-grid', style: {gridTemplateColumns: `repeat(${SEGMENTS.length}, minmax(0, 1fr))`}},
      SEGMENTS.map((sg, i) => h('button', {class: `yr-phase${lit(st.segments, sg.id)}`, 'aria-pressed': String(!!st.segments && st.segments.includes(sg.id)),
        title: sg.hint, onclick: () => ctx.setLocal({segments: toggle(st.segments, sg.id, all)})},
      h('span', {class: 'yr-bar-wrap'}, h('span', {class: 'yr-count num'}, `${fmt.n(avg[i], 2)} wins`),
        h('span', {class: 'yr-bar', style: {height: `${Math.max(2, Math.round((avg[i] / max) * 52))}px`}})),
      h('span', {class: 'yr-label'}, sg.label)))));
}

function filterPicker(ctx, st) {
  const chip = (key, all, id, label, hint) => h('button', {class: `chip${lit(st[key], id)}`, 'aria-pressed': String(!!st[key] && st[key].includes(id)), title: hint,
    onclick: () => ctx.setLocal({[key]: toggle(st[key], id, all)})}, label);
  const group = (title, ...xs) => h('div', {class: 'as-group', role: 'group', 'aria-label': title}, h('span', {class: 'as-title'}, title), xs);
  return h('div', {class: 'as'},
    h('div', {class: 'flt-head'}, h('h2', {}, 'Picks'),
      h('span', {class: 'flt-presets'}, h('button', {class: `flt-all${st.groups || st.nfl ? '' : ' on'}`, onclick: () => ctx.setLocal({groups: null, nfl: null})}, 'Every pick'))),
    h('div', {class: 'as-rows'},
      group('Position', GROUPS.map(g => chip('groups', POSITIONS, g.id, g.short, g.hint || g.label))),
      group('NFL round', NFL_ROUNDS.map(n => chip('nfl', NFL_ROUNDS.map(x => x.id), n.id, n.label, n.hint))),
      group('Judge on', segmented(JUDGE, st.seasons, v => ctx.setLocal({seasons: v}), 'Judge picks on'))));
}

// ---------- the league map ----------

function tone(c) {
  return c?.ratio == null ? '' : c.tone === 'up' ? ' up' : c.tone === 'down' ? ' down' : '';
}
function ratioCell(c, open, title) {
  if (!c || c.ratio == null) return h('td', {class: 'dm-cell none'}, c?.n ? h('span', {class: 'faint small'}, picksWord(c.n)) : '');
  return h('td', {class: `dm-cell click${tone(c)}`, tabindex: '0', onclick: open, onkeydown: e => { if (e.key === 'Enter') open(); },
    title: `${title}: ${pct(c.ratio)} of the slot, likely ${pct(c.lo)} to ${pct(c.hi)}. Click for the picks.`},
  h('span', {class: 'dm-value num'}, pct(c.ratio)), c.n != null ? h('span', {class: 'dm-n'}, `${c.n}`) : null);
}

function whenMap(ctx, map, picks, st) {
  return h('section', {class: 'dm', 'aria-label': 'When each position pays'},
    h('h2', {}, 'When each position pays'),
    h('p', {class: 'mk-note'}, 'Wins in each season after the draft as a percent of what the same slots returned that season, from every class that has played it. Green beats the slot beyond chance, red falls short. The small figure is how many picks.'),
    h('div', {class: 'table-wrap'}, h('table', {class: 'data dm-table'},
      h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Position'),
        Array.from({length: SEASONS_AFTER}, (_, i) => h('th', {scope: 'col'}, `Year ${i + 1}`)),
        h('th', {scope: 'col'}, `First ${st.seasons} together`))),
      h('tbody', {}, map.map(r => h('tr', {},
        h('th', {scope: 'row'}, h('button', {class: 'link dm-pos', onclick: () => ctx.setLocal({groups: [r.group]})}, GROUP_LABEL[r.group])),
        r.when.map((c, i) => ratioCell(c, () => showPicks(ctx, `${Name(r.group)}, year ${i + 1}`, seasonOf(picks.filter(p => p.group === r.group), i), `wins in their ${ordinal(i + 1)} season against the slot`), `${GROUP_LABEL[r.group]}, year ${i + 1}`)),
        ratioCell({...r.overall, n: r.n}, () => showPicks(ctx, `${Name(r.group)}`, judgedOf(picks.filter(p => p.group === r.group)), `wins in the first ${st.seasons} seasons against the slot`), GROUP_LABEL[r.group])))))));
}

function whereMap(ctx, map, picks, st) {
  return h('section', {class: 'dm', 'aria-label': 'Where in the draft each position pays'},
    h('h2', {}, 'Where in the draft each position pays'),
    h('p', {class: 'mk-note'}, `Wins in the first ${st.seasons} seasons as a percent of what the same slots returned, by where the pick was taken.`),
    h('div', {class: 'table-wrap'}, h('table', {class: 'data dm-table'},
      h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Position'), SEGMENTS.map(s => h('th', {scope: 'col', title: s.hint}, s.label)))),
      h('tbody', {}, map.map(r => h('tr', {},
        h('th', {scope: 'row'}, h('button', {class: 'link dm-pos', onclick: () => ctx.setLocal({groups: [r.group]})}, GROUP_LABEL[r.group])),
        r.where.map(c => ratioCell(c, () => showPicks(ctx, `${Name(r.group)}, ${SEG[c.segment].label}`, judgedOf(picks.filter(p => p.group === r.group && p.segment === c.segment)), `wins in the first ${st.seasons} seasons against the slot`), `${GROUP_LABEL[r.group]}, ${SEG[c.segment].label}`))))))));
}

// The clearest findings in the map, in words.
function headline(map) {
  const late = map.filter(r => r.when[0].tone === 'down' && r.when.slice(3).some(c => c.ratio != null && c.ratio >= 0.95))
    .sort((a, b) => a.when[0].ratio - b.when[0].ratio)[0];
  const ups = map.filter(r => r.overall?.tone === 'up' && r.group !== 'PK'), downs = map.filter(r => r.overall?.tone === 'down');
  const parts = [];
  if (late) {
    const back = late.when.findIndex((c, i) => i >= 3 && c.ratio >= 0.95);
    parts.push(`${Name(late.group)} pay late: ${pct(late.when[0].ratio)} of their slot in year 1, ${pct(late.when[back].ratio)} by year ${back + 1}.`);
  }
  if (downs.length) parts.push(`Short of their slots overall: ${downs.map(r => `${GROUP_PLURAL[r.group]} (${pct(r.overall.ratio)})`).join(', ')}.`);
  if (ups.length) parts.push(`Beyond their slots: ${ups.map(r => `${GROUP_PLURAL[r.group]} (${pct(r.overall.ratio)})`).join(', ')}.`);
  return parts.join(' ') || 'No position clearly beats or falls short of its slots in the picks in view.';
}

// ---------- one position ----------

function cells(items, label, open) {
  return h('div', {class: 'ap-when-grid', style: {gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))`}}, items.map(c => {
    const known = c.ratio != null;
    return h('button', {class: `ap-when-cell${c.best ? ' cheapest' : ''}${c.worst ? ' dearest' : ''}${known && c.tone !== 'even' ? ` t-${c.tone}` : ''}`,
      disabled: !c.n || null, title: known ? `Likely ${pct(c.lo)} to ${pct(c.hi)} of the slot. See the picks.` : 'Too few picks to judge',
      onclick: () => open(c)},
    h('span', {class: 'ap-when-label'}, label(c)),
    h('span', {class: 'ap-when-value num'}, known ? pct(c.ratio) : '—'),
    h('span', {class: 'ap-when-n'}, c.like && known ? `like pick ${c.like}` : c.n ? picksWord(c.n) : 'none'),
    c.like && known ? h('span', {class: 'ap-when-n'}, `taken ~${c.at} · ${fmt.n(c.n)}`) : null);
  }));
}

// What a row of cells says, in words: the clear ones only.
function sayRow(items, label) {
  const up = items.filter(c => c.tone === 'up'), down = items.filter(c => c.tone === 'down');
  const best = items.find(c => c.best), worst = items.find(c => c.worst);
  const bits = [];
  const list = xs => xs.map(c => `${label(c)} (${pct(c.ratio)})`).join(', ');
  if (best) bits.push(`Best: ${label(best)} (${pct(best.ratio)}), beyond chance against every other.`);
  if (worst) bits.push(`Worst: ${label(worst)} (${pct(worst.ratio)}), beyond chance against every other.`);
  const ups = up.filter(c => c !== best), downs = down.filter(c => c !== worst);
  if (ups.length) bits.push(`Beats the slot: ${list(ups)}.`);
  if (downs.length) bits.push(`Falls short: ${list(downs)}.`);
  return bits.length ? bits.join(' ') : 'Nothing here clears chance: every cell is within reach of the slot.';
}

function buyBlock(g, market, picksOf) {
  if (!market?.rows) return null;
  const row = k => market.rows.find(r => r.kind === k);
  const ages = [['young', '24 or younger'], ['prime', '25 to 27'], ['older', '28 or older']].map(([a, l]) => ({l, r: row(`${g}:${a}`)})).filter(x => x.r?.cost);
  const picks = ['pick1:next', 'pick2:next', 'pick3:next'].map(k => row(k)).filter(r => r?.cost);
  const name = {'pick1:next': 'a next-draft 1st', 'pick2:next': 'a next-draft 2nd', 'pick3:next': 'a next-draft 3rd or later'};
  const line = (label, r) => h('div', {class: `ap-row ap-buy-row ${r.side === 'cheap' ? 'cheaper' : r.side === 'dear' ? 'dearer' : 'even'}`},
    h('span', {class: 'ap-row-name'}, label), h('span', {class: 'ap-row-value num'}, pct(r.cost.value)));
  const young = ages.find(x => x.l === '24 or younger')?.r;
  const cheapest = [...ages].sort((a, b) => a.r.cost.value - b.r.cost.value)[0];
  // Compare with the round these picks were actually spent in: the most common one in view.
  const rounds = new Map();
  for (const p of picksOf) { const k = `pick${Math.min(3, p.round)}:next`; rounds.set(k, (rounds.get(k) || 0) + 1); }
  const usual = [...rounds.entries()].sort((a, b) => b[1] - a[1])[0]?.[0];
  const via = picks.find(r => r.kind === usual) || picks[0];
  const say = cheapest && via
    ? `A delivered win from ${GROUP_PLURAL[g]} aged ${cheapest.l} costs ${pct(cheapest.r.cost.value)} of the going rate in trades; through ${name[via.kind]}, the round most of these were taken in, it costs ${pct(via.cost.value)}.`
    : null;
  return h('div', {class: 'ap-cell'}, h('h3', {}, 'Draft it, or buy it'),
    say ? h('p', {class: 'ap-say'}, say) : null,
    h('div', {class: 'ap-rows'}, ages.map(x => line(`${Name(g)}, ${x.l}`, x.r)), picks.map(r => line(name[r.kind].replace(/^a /, 'Pick: '), r))),
    h('p', {class: 'mk-note'}, `Cost per delivered win against the going rate, from the trade market in the same seasons: what the league charges for proven ${GROUP_PLURAL[g]} at each age, against what it charges through the picks you would draft them with.${young ? '' : ' Too few young ones traded to price.'}`));
}

function profile(ctx, p, st, market) {
  const g = p.group;
  const open = (title, xs, basis) => showPicks(ctx, `${Name(g)}, ${title}`, xs, basis);
  const basis = `wins in the first ${st.seasons} seasons against the slot`;
  const o = p.overall;
  return h('section', {class: 'ap', 'aria-label': GROUP_LABEL[g]},
    h('header', {class: 'ap-head'},
      h('h2', {}, Name(g)),
      o ? h('span', {class: `ap-cost num ${o.tone === 'up' ? 'cheap' : o.tone === 'down' ? 'dear' : ''}`}, pct(o.ratio), h('span', {class: 'mk-unit'}, ' of the slot')) : null,
      p.hits ? h('span', {class: `ap-hits ${p.hits.tone}`}, `${pct(p.hits.rate)} became starters, against ${pct(p.hits.baseline)} for the same picks overall`) : null,
      h('button', {class: 'link ap-see', onclick: () => open('every pick', judgedOf(p.all), basis)}, `See the ${fmt.n(p.n)} picks`)),
    h('div', {class: 'ap-grid even'},
      h('div', {class: 'ap-cell'}, h('h3', {}, 'Where to take it'),
        h('p', {class: 'ap-say'}, sayRow(p.where, c => SEG[c.segment].label.replace(/^R/, 'r'))),
        cells(p.where.filter(c => !st.segments || st.segments.includes(c.segment)), c => SEG[c.segment].label, c => open(SEG[c.segment].label, judgedOf(c.picks), basis)),
        h('p', {class: 'mk-note'}, '“Plays like pick N”: the overall pick whose usual return matches what these picks returned.')),
      h('div', {class: 'ap-cell'}, h('h3', {}, 'When it pays'),
        h('p', {class: 'ap-say'}, sayRow(p.when, c => `year ${c.season}`)),
        cells(p.when, c => `Year ${c.season}`, c => open(`year ${c.season}`, seasonOf(c.picks, c.season - 1), `wins in their ${ordinal(c.season)} season against the slot`)),
        h('p', {class: 'mk-note'}, 'Each season after the draft against what the same slots returned that season.'))),
    h('div', {class: 'ap-grid even'},
      h('div', {class: 'ap-cell'}, h('h3', {}, 'What decides it'),
        h('p', {class: 'ap-say'}, sayRow(p.nfl, c => (c.nfl === 'nx' ? 'no NFL pick that spring' : `NFL round ${NFL[c.nfl].label}`))),
        cells(p.nfl, c => nflName(c.nfl), c => open(c.nfl === 'nx' ? 'no NFL pick that spring' : `NFL round ${NFL[c.nfl].label}`, judgedOf(c.picks), basis)),
        h('p', {class: 'ap-say ap-gap'}, sayRow(p.age, c => `aged ${AGE[c.age].label}`)),
        cells(p.age, c => `Aged ${AGE[c.age].label}`, c => open(`aged ${AGE[c.age].label}`, judgedOf(c.picks), basis))),
      buyBlock(g, market, p.all.filter(x => x.judged))));
}

export default {
  id: 'draft', nav: 'Draft timing', hint: 'When each position is worth taking',
  title: 'When to take each position',
  lede: null,
  render(main, ctx) {
    const {data} = ctx;
    const st = ctx.local({seasons: 3, segments: null, groups: null, nfl: null});
    const valid = (sel, ids) => (Array.isArray(sel) && sel.some(x => ids.includes(x)) ? ids.filter(x => sel.includes(x)) : null);
    st.segments = valid(st.segments, SEGMENTS.map(s => s.id));
    st.groups = valid(st.groups, POSITIONS);
    st.nfl = valid(st.nfl, NFL_ROUNDS.map(n => n.id));
    if (![2, 3, 4].includes(st.seasons)) st.seasons = 3;
    const from = Math.max(ctx.from, 2017), to = ctx.to;
    const everyPick = draftScope(data, {from, to, seasons: st.seasons});
    main.append(h('section', {class: 'flt', 'aria-label': 'What to look at'}, draftPicker(ctx, st, everyPick), filterPicker(ctx, st)));

    const picks = draftScope(data, {from, to, seasons: st.seasons, segments: st.segments, nfl: st.nfl});
    const judged = picks.filter(p => p.judged);
    if (judged.length < 40) {
      main.append(empty(`Only ${picksWord(judged.length)} in view have ${st.seasons} finished seasons, too few to judge. Widen the draft stretches or NFL rounds, judge on fewer seasons, or widen the seasons above.`));
      return;
    }
    const classes = [...new Set(judged.map(p => p.year))].sort();
    if (st.groups) {
      const market = leagueMarket(data, {from, to, byAge: true});
      main.append(h('div', {class: 'ap-list'}, st.groups.map(g => profile(ctx, positionDraft(picks, g), st, market))));
      main.append(h('h2', {class: 'mk-league'}, 'Every position, in the same picks'));
    }
    const map = draftMap(picks, GROUPS);
    main.append(h('p', {class: 'mk-headline'}, headline(map)));
    main.append(h('p', {class: 'mk-unit-line'}, `100% is what the same slots returned in the other classes. `,
      h('span', {class: 'nowrap'}, `${fmt.n(judged.length)} picks from the ${classes[0]}–${classes.at(-1)} classes, judged on their first ${st.seasons} seasons.`)));
    main.append(whenMap(ctx, map, picks, st));
    main.append(whereMap(ctx, map, picks, st));
    main.append(h('div', {class: 'mk-how'}, howCounted(
      'A pick’s return is the wins above replacement its player produced, in this league’s scoring. Its slot is what players taken at the same overall pick returned in every other class, so a pick never sets its own bar. A percent is the picks’ total wins over their slots’ total, so a few big hits count as what they are and a short list gets a wide range.',
      'Green and red only when the likely range clears 100%. A stretch, season, NFL round or age is called best or worst only when it beats every other beyond chance. With many cells on the page, expect one or two of the colours to be chance; trust the ones that repeat across years and stretches.',
      '“Plays like pick N” reads a group’s return back onto the draft’s own value curve: the overall pick whose usual return matches it. A position taken around pick 82 that plays like pick 68 was worth reaching for. “Became starters” counts players whose best season was a starter’s or better, against the share for picks from the same stretches of the draft.',
      'Draft it, or buy it compares the trade market’s cost per delivered win for proven players of the position with the cost through the picks you would draft them with, in the same seasons.',
    )));
  },
};
