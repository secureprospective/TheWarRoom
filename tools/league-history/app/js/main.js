// Boot, routing and the shared context every screen receives.

import {decode, fmt} from './model.js';
import {h, closeDrawer, showTrades} from './ui.js';
import market from './views/market.js';
import draft from './views/draft.js';
import tradelog from './views/tradelog.js';
import method from './views/method.js';

const VIEWS = [market, draft, tradelog, method];

const state = {data: null, view: VIEWS[0], from: 2017, to: null, local: {}};

function readHash() {
  const [path, query = ''] = location.hash.replace(/^#\/?/, '').split('?');
  const params = new URLSearchParams(query);
  let local = {};
  try { local = JSON.parse(params.get('s') || '{}'); } catch { local = {}; }
  if (!local || typeof local !== 'object' || Array.isArray(local)) local = {};
  const years = state.data.years;
  const year = v => (years.includes(+v) ? +v : null);
  const from = year(params.get('from')) ?? 2017, to = year(params.get('to')) ?? years.at(-1);
  return {view: VIEWS.find(v => v.id === path) || VIEWS[0], from: Math.min(from, to), to: Math.max(from, to), local};
}

function writeHash(replace = false) {
  const params = new URLSearchParams();
  if (state.from !== 2017) params.set('from', state.from);
  if (state.to !== state.data.years.at(-1)) params.set('to', state.to);
  const local = state.local[state.view.id];
  if (local && Object.keys(local).length) params.set('s', JSON.stringify(local));
  const q = params.toString();
  const url = `#/${state.view.id}${q ? `?${q}` : ''}`;
  if (url !== location.hash) (replace ? history.replaceState : history.pushState).call(history, null, '', url);
}

const ctx = {
  get data() { return state.data; },
  get from() { return state.from; },
  get to() { return state.to; },
  local(defaults) {
    const id = state.view.id;
    state.local[id] = {...defaults, ...(state.local[id] || {})};
    return state.local[id];
  },
  setLocal(patch) {
    state.local[state.view.id] = {...state.local[state.view.id], ...patch};
    writeHash(true);
    render();
  },
  saveLocal() { writeHash(true); },
  go(viewId, local) {
    state.view = VIEWS.find(v => v.id === viewId) || state.view;
    if (local) state.local[state.view.id] = {...(state.local[state.view.id] || {}), ...local};
    writeHash();
    render();
    window.scrollTo({top: 0});
  },
  openTrades: (title, trades, opts) => showTrades(ctx, title, trades, opts),
};

function nav() {
  const el = document.getElementById('nav');
  el.replaceChildren(...VIEWS.map(v => h('a', {href: `#/${v.id}`, class: v === state.view ? 'on' : null, 'aria-current': v === state.view ? 'page' : null,
    onclick: e => { e.preventDefault(); ctx.go(v.id); }}, h('span', {class: 'nav-label'}, v.nav), h('span', {class: 'nav-hint'}, v.hint))));
}

function topBar() {
  const {data} = state;
  const yearSelect = (value, label, onChange) => h('select', {'aria-label': label, onchange: e => onChange(+e.target.value)},
    data.years.map(y => h('option', {value: y, selected: y === value ? true : null}, y)));
  const setYears = (from, to) => { state.from = Math.min(from, to); state.to = Math.max(from, to); writeHash(); render(); };
  document.getElementById('levers').replaceChildren(h('div', {class: 'bar'},
    h('label', {class: 'bar-field'}, h('span', {}, 'Seasons'), yearSelect(state.from, 'First season', v => setYears(v, state.to)),
      h('span', {class: 'muted'}, 'to'), yearSelect(state.to, 'Last season', v => setYears(state.from, v))),
    h('span', {class: 'bar-note muted small'}, 'The whole league, every team')));
}

function render() {
  if (!state.data) return;
  nav();
  topBar();
  const main = document.getElementById('view');
  const scrollY = window.scrollY;
  main.replaceChildren();
  document.title = `${state.view.title} · Legacy NFL Lab`;
  main.append(h('header', {class: 'view-head'}, h('h1', {}, state.view.title), state.view.lede ? h('p', {class: 'lede'}, state.view.lede) : null));
  try {
    state.view.render(main, ctx);
  } catch (err) {
    console.error(err);
    main.append(h('p', {class: 'error'}, `This screen hit a problem: ${err.message}`));
  }
  window.scrollTo({top: scrollY});
}

function follow() {
  const r = readHash();
  state.view = r.view; state.from = r.from; state.to = r.to; state.local[r.view.id] = r.local;
  closeDrawer();
  render();
}
window.addEventListener('popstate', follow);
window.addEventListener('hashchange', follow);

async function boot() {
  const status = document.getElementById('status');
  try {
    const res = await fetch('data/lab.json', {cache: 'no-cache'});
    if (!res.ok) throw new Error(`the data file returned ${res.status}`);
    const raw = await res.json();
    if (raw.schema !== 'league-lab-2') throw new Error('the data file is from a different version of the Lab');
    state.data = decode(raw);
  } catch (err) {
    status.textContent = `Could not load the league data: ${err.message}. Rebuild it with python3 compile/build_lab.py.`;
    status.classList.add('error');
    return;
  }
  const r = readHash();
  state.view = r.view; state.from = r.from; state.to = r.to; state.local[r.view.id] = r.local;
  document.getElementById('through').textContent = `Archive through ${fmt.date(new Date(state.data.archiveThrough))}`;
  document.body.classList.remove('loading');
  status.remove();
  writeHash(true);
  render();
  registerOffline();
}

function registerOffline() {
  const badge = document.getElementById('offline');
  if (!('serviceWorker' in navigator)) { badge.textContent = 'Online only'; return; }
  // A worker that never becomes ready (blocked, or a private window) must not leave the line hanging.
  const settle = setTimeout(() => { badge.textContent = 'Online only'; }, 10000);
  navigator.serviceWorker.register('sw.js').then(async reg => {
    await navigator.serviceWorker.ready;
    clearTimeout(settle);
    badge.textContent = 'Works offline';
    badge.title = 'The app and the league data are saved on this computer.';
    reg.update?.();
  }).catch(() => { badge.textContent = 'Online only'; });
}

window.__lab = {ctx};
boot();
