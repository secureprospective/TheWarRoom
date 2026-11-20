// Desktop v3 overlays: the league switcher (Alt+L) and the command palette (Ctrl K), after the mock's
// shell.js. Arrow keys and Enter in the palette go through a window listener, not an element handler.
import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import { HQAct } from '../shell/HQAct';
import { Icon } from './icons';
import { Badge, DkRow } from './pieces';
import { SCREENS, type Brief, type DocketItem, type Lane, type Open,
  type V3League } from './model';

export function Switcher({ leagues, docket, briefs, current, pick }: { leagues: V3League[];
  docket: DocketItem[]; briefs: Record<string, Brief>; current?: string; pick: (id?: string) => void }) {
  const [q, setQ] = useState('');
  const waiting = (l: V3League) => docket.filter(d => d.lg === l.id && d.lane === 'need').length;
  const grp: [string, V3League[]][] = [['NEEDS YOU', leagues.filter(l => waiting(l))],
    ['EVERYTHING ELSE', leagues.filter(l => !waiting(l))]];
  const rec = (l: V3League) =>
    briefs[l.id]?.brief?.pulse?.standings?.rows.find(r => r.franchise === l.franchise)?.record;
  const match = (l: V3League) => !q || l.name.toLowerCase().includes(q.toLowerCase());
  return <div className="pop sw">
    <input placeholder="Find a league…" autoFocus value={q} onChange={e => setQ(e.target.value)} />
    <HQAct className={`lrow${current ? '' : ' sel'}`} onAction={() => pick(undefined)}><Badge />
      <span><b>All leagues</b> <span className="d">back to Home</span></span><span /><span /><span /></HQAct>
    <div className="list">{grp.map(([g, all]) => {
      const ls = all.filter(match);
      return ls.length ? <div key={g}><div className="sh">{g} · {ls.length}</div>{ls.map(l =>
        <HQAct key={l.id} className={`lrow${l.id === current ? ' sel' : ''}`}
          style={{ '--c': l.c } as CSSProperties} onAction={() => pick(l.id)}>
          <Badge L={l} />
          <span className="ell"><b>{l.name}</b> <span className="d">{l.franchiseName ?? ''}</span></span>
          <span className="role">{l.commish ? <Icon n="crown" /> : null}</span>
          <span className="rc">{rec(l) ?? '—'}</span>
          <span className={`w8${waiting(l) ? '' : ' z'}`}>{waiting(l) || '·'}</span></HQAct>)}</div> : null;
    })}</div>
    <div className="sh" style={{ borderTop: '1px solid var(--edge)' }}>
      <Icon n="crown" /> commissioner · amber = matters waiting on you</div></div>;
}

type Command = [label: string, kind: string, run: () => void];

export function Commands({ leagues, close, open, home, pick }: { leagues: V3League[]; close: () => void;
  open: Open; home: () => void; pick: (id: string) => void }) {
  const [q, setQ] = useState(''), [sel, setSel] = useState(0);
  const items = useMemo(() => {
    const out: Command[] = [['Home', 'screen', home]];
    for (const L of leagues) {
      for (const s of SCREENS.filter(x => x.on(L))) {
        out.push([`${s.name} › ${L.name}`, 'screen', () => open(L, s.k)]);
      }
      out.push([`League › ${L.name}`, 'league', () => pick(L.id)]);
    }
    return out;
  }, [leagues, home, open, pick]);
  const shown = items.filter(i => !q || i[0].toLowerCase().includes(q.toLowerCase())).slice(0, 12);
  const run = (i: number) => { const f = shown[i]; close(); f?.[2](); };
  useEffect(() => {
    if (typeof window.addEventListener !== 'function') return;
    const key = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        const step = e.key === 'ArrowDown' ? 1 : -1;
        setSel(v => Math.max(0, Math.min(shown.length - 1, v + step)));
      } else if (e.key === 'Enter') { e.preventDefault(); run(sel); }
    };
    window.addEventListener('keydown', key);
    return () => window.removeEventListener('keydown', key);
  });
  return <div className="pop cmdk">
    <input placeholder="Search players, teams, screens, pieces and commands" autoFocus value={q}
      onChange={e => { setQ(e.target.value); setSel(0); }} />
    <div className="res">{shown.length ? shown.map((it, k) =>
      <HQAct key={it[0]} className={k === sel ? 'on' : ''} onAction={() => run(k)}>
        {it[0]}<small>{it[1].toUpperCase()}</small></HQAct>)
      : <div className="st">Nothing matches. Try a league or a screen.</div>}</div></div>;
}

const LANES: [Lane, string][] = [['need', 'NEEDS YOU'], ['wait', 'WAITING ON'], ['land', 'LANDED']];
const EMPTY: Record<Lane, string> = { need: 'Nothing needs you.', wait: 'Nothing is waiting on anyone else.',
  land: 'Nothing has landed since you were last here.' };

/** The Docket drawer (Needs you key, Alt+D, the band's OPEN DOCKET): who has the ball, in every league. */
export function DocketDrawer({ leagues, docket, current, open, close, home }: { leagues: V3League[];
  docket: DocketItem[]; current: V3League | null; open: Open; close: () => void; home: () => void }) {
  const [lane, setLane] = useState<Lane>('need'), [only, setOnly] = useState(false);
  const all = docket.filter(d => !only || !current || d.lg === current.id);
  const rows = all.filter(d => d.lane === lane);
  const go: Open = (L, e) => { close(); open(L, e); };
  return <aside className="sdrawer">
    <div className="dh rivets"><h2>Docket</h2><span className="sub">who has the ball, in every league</span>
      <HQAct className="x" onAction={close}>CLOSE ✕</HQAct></div>
    <div className="lanes">{LANES.map(([k, t]) => <HQAct key={k} className={k === 'need' ? 'need' : undefined}
      selected={lane === k} label={`${t} ${all.filter(d => d.lane === k).length}`}
      onAction={() => setLane(k)}>
      {t}<b>{all.filter(d => d.lane === k).length}</b></HQAct>)}</div>
    <div className="fbar" style={{ display: 'flex' }}>
      <HQAct className="chip" active={!only || !current} onAction={() => setOnly(false)}>
        {`All ${leagues.length} ${leagues.length === 1 ? 'league' : 'leagues'}`}</HQAct>
      {current && <HQAct className="chip" active={only} onAction={() => setOnly(true)}>
        {`${current.name} only`}</HQAct>}</div>
    <div className="db" style={{ padding: 0, gap: 0 }}>{rows.length ? rows.map(d =>
      <DkRow key={d.id} d={d} L={leagues.find(l => l.id === d.lg)} open={go} />)
      : <div className="st"><div className="eng">{EMPTY[lane]}</div></div>}</div>
    <div className="df"><span className="d mono" style={{ fontSize: 10, lineHeight: 1.5 }}>
      NOTHING LEAVES THE APP: no email, no push.<br />Read marks are private. Private offers stay between
      the parties.</span><span className="sp" />
      <HQAct className="btn" onAction={() => { close(); home(); }}>Add to Home</HQAct></div>
  </aside>;
}
