// Shared interface pieces: element helper, page furniture, the records drawer and trade cards.

import {franchiseName, playerName, pickLabel, kindLabel, WINDOW_LABEL, fmt} from './model.js';

export function h(tag, attrs = {}, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k === 'class') el.className = v;
    else if (k === 'style' && typeof v === 'object') Object.assign(el.style, v);
    else if (k.startsWith('on') && typeof v === 'function') el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? '' : v);
  }
  for (const c of children.flat(Infinity)) {
    if (c == null || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}

const SVG = 'http://www.w3.org/2000/svg';
export function s(tag, attrs = {}, ...children) {
  const el = document.createElementNS(SVG, tag);
  for (const [k, v] of Object.entries(attrs)) if (v != null) el.setAttribute(k, v);
  for (const c of children.flat()) if (c != null) el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  return el;
}

// ---------- page furniture ----------

export function panel(title, {note, controls, id} = {}, ...body) {
  return h('section', {class: 'panel', id},
    h('header', {class: 'panel-head'}, h('div', {}, h('h2', {}, title), note ? h('p', {class: 'panel-note'}, note) : null),
      controls ? h('div', {class: 'panel-controls'}, controls) : null),
    ...body);
}

export function segmented(options, value, onChange, label) {
  return h('div', {class: 'segmented', role: 'group', 'aria-label': label}, options.map(o =>
    h('button', {class: value === o.id ? 'on' : '', 'aria-pressed': String(value === o.id), title: o.hint || null,
      onclick: () => onChange(o.id)}, o.label)));
}

export function howCounted(...lines) {
  return h('details', {class: 'how'}, h('summary', {}, 'How this is counted'), ...lines.map(l => h('p', {}, l)));
}

export function empty(text) {
  return h('p', {class: 'empty'}, text);
}

// The few conclusions a screen leads with. Each is a sentence and the evidence it rests on.
export function pins(items) {
  return h('div', {class: 'pinned'}, items.filter(Boolean).map(p =>
    h('div', {class: `pin${p.tone ? ` pin-${p.tone}` : ''}`}, h('p', {class: 'say'}, p.say), p.basis ? h('p', {class: 'small muted'}, p.basis) : null,
      p.action ? h('button', {class: 'link', onclick: p.action.run}, p.action.label) : null)));
}

// An estimate and its range on a log scale where 1 is the reference (a next-draft 1st).
export function rangeBar({value, lo, hi}, {min = 0.08, max = 3, width = 200, height = 22} = {}) {
  const x = v => Math.max(0, Math.min(width, (Math.log(v / min) / Math.log(max / min)) * width));
  const svg = s('svg', {width, height, viewBox: `0 0 ${width} ${height}`, class: 'rangebar', role: 'img',
    'aria-label': `${value.toFixed(2)}, likely between ${lo.toFixed(2)} and ${hi.toFixed(2)}`});
  svg.append(s('line', {x1: x(1), x2: x(1), y1: 2, y2: height - 2, class: 'ref'}));
  svg.append(s('line', {x1: x(lo), x2: x(hi), y1: height / 2, y2: height / 2, class: 'range'}));
  svg.append(s('circle', {cx: x(value), cy: height / 2, r: 4.5, class: 'dot'}));
  return svg;
}

// ---------- records drawer ----------

const drawer = () => document.getElementById('drawer');

export function closeDrawer() {
  const d = drawer();
  if (d && !d.hidden) { d.hidden = true; document.body.classList.remove('drawer-open'); d._return?.focus(); }
}
document.addEventListener('keydown', e => { if (e.key === 'Escape') closeDrawer(); });

export function openDrawer(title, subtitle, body, csv) {
  const d = drawer();
  d._return = document.activeElement;
  d.replaceChildren(
    h('div', {class: 'drawer-head'},
      h('div', {}, h('h2', {tabindex: '-1'}, title), subtitle ? h('p', {class: 'muted small'}, subtitle) : null),
      h('div', {class: 'row'}, csv ? h('button', {class: 'ghost', onclick: csv}, 'Download CSV') : null,
        h('button', {class: 'ghost', 'aria-label': 'Close', onclick: closeDrawer}, 'Close'))),
    h('div', {class: 'drawer-body'}, body));
  d.hidden = false;
  document.body.classList.add('drawer-open');
  d.querySelector('h2').focus();
  d.scrollTop = 0;
}

export function paged(items, render, size = 30) {
  const box = h('div', {});
  let shown = 0;
  const more = h('button', {class: 'ghost more'}, '');
  const step = () => {
    const next = items.slice(shown, shown + size);
    shown += next.length;
    for (const it of next) box.insertBefore(render(it), more);
    more.textContent = `Show ${Math.min(size, items.length - shown)} more of ${fmt.n(items.length - shown)}`;
    more.hidden = shown >= items.length;
  };
  more.onclick = step;
  box.append(more);
  step();
  return box;
}

// ---------- trades ----------

export function assetName(data, l) {
  if (l.kind === 'player') return playerName(l.player);
  if (l.kind === 'pick') return pickLabel(l, data);
  return 'Other consideration';
}

function assetLine(data, l, highlight) {
  const became = l.kind === 'pick' && l.pickPlayer ? `became ${playerName(l.pickPlayer)}` : null;
  return h('li', {class: highlight?.has(l) ? 'hit' : null},
    h('span', {class: 'asset-name'}, assetName(data, l)),
    h('span', {class: 'asset-meta'}, [l.xClass ? kindLabel(l.xClass) : null, became].filter(Boolean).join(' · ')),
    l.xValue != null ? h('span', {class: 'asset-value'},
      h('span', {title: 'Wins above replacement expected over the next five seasons, as known on the day (later seasons count less).'}, `${fmt.n(l.xValue, 2)} expected`),
      l.xReal != null ? h('span', {class: 'after', title: 'Wins it actually produced over the seasons played since, counted the same way.'}, ` · ${fmt.n(l.xReal, 2)} delivered`) : null) : null);
}

export function tradeCard(data, t, {highlight} = {}) {
  const side = (fid, other, gaveSide) => h('div', {class: 'trade-side'},
    h('div', {class: 'trade-team'}, h('strong', {}, franchiseName(data, fid)), h('span', {class: 'small muted'}, `sent to ${franchiseName(data, other, true)}`)),
    h('ul', {class: 'assets'}, t.legs.filter(l => l.side === gaveSide).map(l => assetLine(data, l, highlight))));
  const verdict = t.valued ? h('p', {class: 'trade-verdict small'},
    `At the time: ${franchiseName(data, t.a, true)} ${t.aNet >= 0 ? 'gained' : 'gave up'} ${fmt.n(Math.abs(t.aNet), 2)} expected wins.`,
    t.aRealNet != null ? ` Since then, over ${t.realSeasons} season${t.realSeasons === 1 ? '' : 's'}: ${franchiseName(data, t.a, true)} came out ${t.aRealNet >= 0 ? 'ahead' : 'behind'} by ${fmt.n(Math.abs(t.aRealNet), 2)} wins.` : '') : null;
  return h('article', {class: 'trade'},
    h('div', {class: 'trade-when'}, h('strong', {}, fmt.date(t.date)), t.window ? h('span', {}, WINDOW_LABEL[t.window]) : null,
      !t.valued ? h('span', {class: 'muted'}, t.year < 2017 ? 'before picks were recorded: not valued' : t.oneSided ? 'one-sided entry: not valued' : 'not valued') : null),
    h('div', {class: 'trade-sides'}, side(t.a, t.b, 0), side(t.b, t.a, 1)), verdict);
}

export function showTrades(ctx, title, trades, {legs, note} = {}) {
  const list = [...new Set(trades)].sort((a, b) => b.ts - a.ts);
  const highlight = legs ? new Set(legs) : null;
  const csv = () => download(`${title}.csv`, tradesCsv(ctx.data, list));
  openDrawer(title, `${fmt.n(list.length)} trade${list.length === 1 ? '' : 's'}${note ? ` · ${note}` : ''}`,
    paged(list, t => tradeCard(ctx.data, t, {highlight})), csv);
}

function tradesCsv(data, trades) {
  const rows = [];
  for (const t of trades) for (const l of t.legs) {
    rows.push({date: fmt.isoDay(t.date), season: t.season ?? t.year, time_of_year: WINDOW_LABEL[t.window] ?? '',
      from: franchiseName(data, l.from), to: franchiseName(data, l.to), asset: assetName(data, l), kind: l.xClass ? kindLabel(l.xClass) : '',
      expected_wins: l.xValue ?? '', delivered_wins: l.xReal ?? '', seasons_since: l.xRealSeasons ?? ''});
  }
  return toCsv(rows);
}

export function toCsv(rows) {
  if (!rows.length) return '';
  const cols = Object.keys(rows[0]);
  const esc = v => { const t = String(v ?? ''); const safe = /^[=+\-@]/.test(t) ? `'${t}` : t; return /[",\n]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe; };
  return [cols.join(','), ...rows.map(r => cols.map(c => esc(r[c])).join(','))].join('\n');
}

export function download(name, text) {
  const a = h('a', {href: URL.createObjectURL(new Blob([text], {type: 'text/csv'})), download: name.replace(/[^\w\- .]+/g, '_')});
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
}
