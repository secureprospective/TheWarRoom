import { HomeAct } from './homeTiles';
import { surfaceActions } from '../commissioner/surfaceActions';
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import './home.css';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { ResearchReader } from '../research/api';
import { readHomeBrief, type HomeBrief as Brief } from './homeReads';
import { bump, current, defaultLayout, layoutKey, pack, parseLayout, place, remove, SPANS, swap,
  type Layout, type Phase, type Span, type TileId } from './homeLayout';
import { clock, go, isFinal, left, myGame, myRow, nameOf, oldestAt, pts, SHORTCUT_KEYS, shortTime, TileBody,
  tileState,
  TILES, topFive } from './homeTiles';
import { browserStorage } from './franchise';
import { money, unavailable } from './hqModel';
import { pointsLabel } from '../data/nativePulse';

type Tab = 'brief' | 'depth' | 'stats';
type Alert = { id: string; level: 'red' | 'amber'; title: string; words: string; when?: string };
const TRAY: Record<Tab | 'edit', string> = {
  brief: 'Home reads and links only · nothing here changes the league',
  edit: 'CUSTOMIZE · drag ⠿ to move · size buttons resize · ✕ sends to overflow · nothing commits from Home',
  depth: 'Depth source: nflverse depth charts, newest snapshot · ' +
    'names coloured by who has them in your league',
  stats: 'NFL stats: no league-wide stats table read exists yet',
};

/** alertsOf is what the app already raises for this league and owner: the commissioner jobs board's
 *  blocked jobs and live jobs with a deadline, and my cap check when it is over. Nothing here is invented. */
export function alertsOf(b: Brief): Alert[] {
  // A blocked job needs someone; a live job is an alert only when it has a deadline to meet.
  const jobs = (b.jobs?.jobs ?? []).filter(j => j.state === 'blocked' || (j.state === 'live' && j.closes))
    .map(j => ({
    id: `job.${j.id}`, level: j.state === 'blocked' ? 'red' as const : 'amber' as const, title: j.name,
    words: j.words, when: j.closes || undefined }));
  const room = b.runway?.[0]?.room;
  const cap = room !== undefined && room < 0 ? [{ id: 'cap', level: 'red' as const,
    title: `Over the cap by ${money(-room)}`, words: "Your cap check is over this season's cap." }] : [];
  return [...cap, ...jobs.filter(j => j.level === 'red'), ...jobs.filter(j => j.level !== 'red')];
}

/** HomeBrief is Home as the locked mock draws it (docs/ui/mocks/home-mock-v5.html), fed by the reads
 *  in homeReads. Layout is saved per owner per league; nothing here invents a number. */
export default function HomeBrief({ league, mine, port, initial, research }: { league: string; mine: string;
  port: HostedPort; initial?: Brief; research?: ResearchReader }) {
  const [b, setB] = useState<Brief>(initial ?? { at: {}, errors: {} });
  const grid = useRef<HTMLElement>(null);
  const [tab, setTab] = useState<Tab>('brief');
  const key = layoutKey(league, mine);
  const [lay, setLay] = useState<Layout>(() => parseLayout(browserStorage()?.getItem(key)));
  const [edit, setEdit] = useState(false);
  const [pick, setPick] = useState<{ id: TileId; span: Span }>();
  const [sel, setSel] = useState<TileId>();
  const [drawer, setDrawer] = useState<'al' | 'add' | 'ov'>();
  const [card, setCard] = useState<number>();
  const [msg, setMsg] = useState('');
  const [drag, setDrag] = useState<TileId>();
  useEffect(() => {
    if (initial) return;
    let live = true;
    void readHomeBrief(league, mine, port, r => { if (live) setB(r); }, ...(research ? [research] : []));
    return () => { live = false; };
  }, [league, mine, port, initial, research]);
  const save = (next: Layout) => {
    setLay(next);
    try { browserStorage()?.setItem(key, JSON.stringify(next)); } catch (e) {
      setMsg(`Layout not saved · ${String(e)}`);
    }
  };
  const g = myGame(b, mine);
  const phase: Phase = g || b.results?.bye === mine ? 'in' : 'off';
  const tiles = current(lay, phase);
  const placed = useMemo(() => pack(tiles) ?? [], [tiles]);
  const onGrid = new Set(placed.map(p => p.id));
  // Rows that don't fit are hidden whole; the footer says how many more live in the source screen.
  useLayoutEffect(() => {
    const trim = () => grid.current?.querySelectorAll<HTMLElement>('article.t').forEach(t => {
      const body = t.querySelector<HTMLElement>('.tb'); const more =
        t.querySelector<HTMLElement>('.tf .more');
      if (!body || !more) return;
      const rows = [...body.querySelectorAll<HTMLElement>('.r')];
      rows.forEach(r => { r.style.display = ''; });
      const limit = body.getBoundingClientRect().bottom + 0.5;
      const cut = rows.filter(r => r.getBoundingClientRect().bottom > limit);
      cut.forEach(r => { r.style.display = 'none'; });
      const where = TILES[t.dataset.id as TileId]?.link.replace(' ›', '') || 'source';
      more.textContent = cut.length ? `+${cut.length} more in ${where}` : '';
    });
    trim();
    window.addEventListener('resize', trim);
    return () => window.removeEventListener('resize', trim);
  });
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setCard(undefined); setPick(undefined); setDrawer(undefined); return; }
      if (e.metaKey || e.ctrlKey || e.altKey || edit || tab !== 'brief') return;
      const el = e.target as HTMLElement | null;
      if (el && (/^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName) || el.isContentEditable)) return;
      const open = SHORTCUT_KEYS[e.key.toLowerCase()];
      if (open && e.key.length === 1) { e.preventDefault(); go(open); }
    };
    document.addEventListener('keydown', key);
    return () => document.removeEventListener('keydown', key);
  }, [edit, tab]);
  const alerts = alertsOf(b);
  const alertsLost = b.errors.jobs;
  const pins = (Object.keys(lay.pins) as TileId[]).filter(id => lay.pins[id]);
  const row = myRow(b, mine);
  const room = b.runway?.[0];
  const week = b.pulse?.week;
  const opp = g ? (g.home === mine ? g.away : g.home) : '';
  const myPts = pointsLabel(pts(b.results?.scores.find(s => s.franchise === mine)));
  const opPts = pointsLabel(pts(b.results?.scores.find(s => s.franchise === opp)));
  const next = b.calendar?.filter(e => e.at && Date.parse(e.at) > Date.now())
    .sort((x, y) => Date.parse(x.at!) - Date.parse(y.at!))[0];
  const nowLine = phase === 'in' && g
    ? `Week ${week} · ${standing(pts(b.results?.scores.find(s => s.franchise === mine)),
      pts(b.results?.scores.find(s => s.franchise === opp)), isFinal(g.status))} ${nameOf(b,
        opp)} ${myPts} – ${opPts}${
      next ? ` · ${next.label} ${shortTime(next.at!)}` : ''}`
    : next ? `${next.label} ${shortTime(next.at!)}` : b.pulse?.weekNote || 'Reading the league…';
  const stats: [string, string][] = phase === 'in'
    ? [[row?.record ?? '—', 'RECORD'], [row ? String(row.rank) : '—', 'SEED'], ['—', 'POWER'],
      [room?.room === undefined ? '—' : money(room.room), 'CAP ROOM']]
    : [['—', 'POWER · DYN'], [firstPick(b, mine), 'FIRST PICK'],
      [b.runway?.[1]?.room === undefined ? '—' : money(b.runway[1].room), `${b.runway?.[1]?.season ??
        ''} ROOM`.trim()], ['—', 'ROSTER']];
  const ticker = topFive(b);
  const tickerHtml = ticker.map(([p, xs]) => <span className="tp" key={p}><b>{p}</b>{xs.map((x, i) =>
    <span className={`tpi${i ? '' : ' one'}`} key={x.player}><u>{i + 1}.</u>{x.name}{' '}
      <i>{pointsLabel(pts(x))}</i>
    </span>)}</span>);
  const clickTile = (id: TileId) => {
    if (pick) {
      const n = bump(lay, phase, pick, id);
      if (!n) { setMsg(`Still no room after moving ${TILES[id].t} · pick another`); return; }
      setPick(undefined); save(n); return;
    }
    setSel(id);
  };
  const add = (id: TileId) => {
    const n = place(lay, phase, id, SPANS[id][0]);
    if (n) save(n); else { setPick({ id, span: SPANS[id][0] }); setDrawer(undefined); }
  };
  const resize = (id: TileId, span: Span) => {
    const n = place(lay, phase, id, span);
    if (n) save(n); else setPick({ id, span });
  };
  const cls = ['hb', edit ? 'editing' : '', pick ? 'picking' : '', tab !== 'brief' ? `tab-${tab}` : '']
    .filter(Boolean).join(' ');
  const entries = (b.ledger?.entries ?? []).slice().reverse();
  const tx = card === undefined ? undefined : entries[card];
  useEffect(() => surfaceActions.register('home-brief', ({ value }) => {
    const [action, id, size] = value.split(':');
    switch (action) {
      case 'tab': setTab(id as Tab); break;
      case 'customize': setEdit(!edit); setPick(undefined); setDrawer(undefined); break;
      case 'drawer': setDrawer(drawer === id ? undefined : id as 'al' | 'add' | 'ov'); break;
      case 'close-drawer': setDrawer(undefined); break;
      case 'select': clickTile(id as TileId); break;
      case 'resize': resize(id as TileId, size as Span); break;
      case 'pin': {
        const pins = { ...lay.pins };
        if (pins[id as TileId]) delete pins[id as TileId]; else pins[id as TileId] = true;
        save({ ...lay, pins }); break;
      }
      case 'overflow': save(remove(lay, phase, id as TileId)); break;
      case 'cancel': setPick(undefined); break;
      case 'phase-default': save({ ...lay, mode: 'default', tiles: [] }); break;
      case 'reset': save(defaultLayout()); break;
      case 'done': setEdit(false); setDrawer(undefined); break;
      case 'add': if (!onGrid.has(id as TileId)) add(id as TileId); break;
      case 'restore': add(id as TileId); break;
      case 'transaction': if (!edit) setCard(Number(id)); clickTile('wire'); break;
      case 'close-card': setCard(undefined); break;
    }
  }));
  return <div className={cls} data-testid="home-brief">
    <div className="shell hb-shell">
      <main className="work">
        <div className="ws-tabs" role="tablist">
          {(['brief', 'depth', 'stats'] as const).map(t => <HomeAct role="tab" key={t} selected={tab === t}
            action="tab" value={t}>{{ brief: 'BRIEF', depth: 'DEPTH CHARTS', stats: 'NFL STATS'
            }[t]}</HomeAct>)}
          <div className="lay">LAYOUT <b>{lay.mode === 'default'
            ? `DEFAULT · ${phase === 'in' ? 'IN-SEASON' : 'OFFSEASON'}` : 'MINE (custom)'}</b></div>
          <HomeAct className={`queue${edit ? ' on' : ''}`} style={{ marginLeft: 'auto' }}
            action="customize">
            <span className="ql">CUSTOMIZE</span></HomeAct>
        </div>
        <section className="now" aria-label="Now">
          <div className="nl"><span className="tag">{phase === 'off' ? 'OFFSEASON' : g && isFinal(g.status) ?
            'FINAL' : 'LIVE'}</span>{nowLine}</div>
          {stats.map(([v, l]) => <div className="st" key={l}><b>{v}</b><span>{l}</span></div>)}
        </section>
        <HomeAct className={`aband${alertsLost ? ' unavail' : !alerts.length ? ' clear' : alerts[0].level ===
          'red' ? ' crit' : ''}`} 
          action="drawer" value="al" label="Open all alerts">
          {alertsLost ? <><span className="ak">ALERTS UNAVAILABLE</span>
            <span className="at">{alertsLost} · this is not an all-clear</span></>
            : !b.at.jobs ? <><span className="ak">ALERTS</span><span className="at">Reading…</span></>
            : alerts.length ? <><span className="ak">▲ {alerts[0].title.toUpperCase()}</span>
              <span className="at">{alerts[0].words}</span>
              {alerts[0].when && <span className="clock">{left(alerts[0].when, Date.now())}</span>}</>
            : <><span className="ak">NO ALERTS</span><span className="at">Nothing needs you right now</span>
              </>}
          <span className="more">{alerts.length > 1 && <b>+{alerts.length - 1}</b>}ALL ALERTS ›</span>
            </HomeAct>
        <div className="tick" title="Hover to pause"><span className="tl0">TOP 5 · WK {week ?? '—'}
          {g && !isFinal(g.status) ? ' LIVE' : ' FINAL'}</span>
          <div className="tkw"><div className="tkr">{ticker.length ? <>{tickerHtml}{tickerHtml}</>
            : <span className="tp">{b.errors.results ? `${unavailable} · ${b.errors.results}` :
              'No scored starters yet'}</span>}
          </div></div></div>
        <section className="grid" aria-label="Tiles" ref={grid}>{placed.map(p => {
          const d = TILES[p.id];
          const state = tileState(p.id, b);
          return <article key={p.id} className={`t ${state}${sel === p.id ? ' sel' : ''}`}
            data-id={p.id} draggable={edit}
            onDragStart={() => setDrag(p.id)} onDragEnd={() => setDrag(undefined)}
            onDragOver={e => { if (drag) e.preventDefault(); }}
            onDrop={e => { e.preventDefault(); if (!drag || drag === p.id) return;
              const n = swap(lay, phase, drag, p.id);
              if (n) save(n); else setMsg("Those two sizes don't fit swapped · resize one first");
              setDrag(undefined); }}
            style={{ gridRow: `${p.row + 1}/span ${p.h}`, gridColumn: `${p.col + 1}/span ${p.w}` }}>
            <HomeAct className="tile-select" action="select" value={p.id} label={`Select ${d.t}`} />
            <div className="th"><span className="ed"><span className="grip" title="Drag to move">⠿</span>
              </span>
              <b>{d.t}</b><span className="src">{d.src}</span>
              {d.link && <HomeAct className="lnk" action="navigate" open={d.open}>{d.link}</HomeAct>}
              <span className="ed" style={{ marginLeft: 'auto' }}>
                {(['1x1', '2x1', '1x2'] as const).map(s => <HomeAct key={s} className="sp" active={s ===
                  p.span}
                  disabled={!SPANS[p.id].includes(s)}
                  action="resize" value={`${p.id}:${s}`}>{s.replace('x', '×')}</HomeAct>)}
                <HomeAct className={`pin${lay.pins[p.id] ? ' on' : ''}`}
                  label="Pin this screen to MY SHORTCUTS"
                  action="pin" value={p.id}>
                  {lay.pins[p.id] ? '★' : '☆'}</HomeAct>
                <HomeAct className="off" label="Move to overflow"
                  action="overflow" value={p.id}>✕</HomeAct></span></div>
            <div className="tb"><TileBody id={p.id} b={b} mine={mine} pins={pins} /></div>
            <div className="tf"><span>{asOf(p.id, b)}</span><span className="more" /></div>
          </article>;
        })}</section>
        <section className={`wsx${tab === 'depth' ? ' on' : ''}`} aria-label="Depth charts">
          <Depth b={b} mine={mine} /></section>
        <section className={`wsx${tab === 'stats' ? ' on' : ''}`} aria-label="NFL stats">
          <div className="empty">{unavailable} · No league-wide NFL stats table read for an app league</div>
            </section>
        <div className="tray">
          <div className="pulls">
            <HomeAct className="pull" expanded={drawer === 'al'}
              action="drawer" value="al">ALERTS <b>{alertsLost ? '?' : alerts.length}</b></HomeAct>
            {edit && <HomeAct className="pull" action="drawer" value="add">
              ADD TILE <b>+</b></HomeAct>}
            <HomeAct className="pull" action="drawer" value="ov">
              OVERFLOW <b>{lay.over.length}</b></HomeAct>
          </div>
          <span className={`msg${pick ? ' ask' : ''}`}>{pick
            ?
              `Grid full · click the tile to move to OVERFLOW to make room for ${TILES[pick.id].t} ` +
              pick.span.replace('x', '×')
            : msg || TRAY[edit ? 'edit' : tab]}</span>
          <span className="sp" />
          <span style={{ display: 'flex', gap: 8 }}>{pick
            ? <HomeAct className="btn" action="cancel">CANCEL</HomeAct>
            : edit && <>
              <HomeAct className="btn" action="phase-default">USE THIS PHASE'S DEFAULT</HomeAct>
              <HomeAct className="btn" action="reset">RESET TO DEFAULT</HomeAct>
              <HomeAct className="btn solid" action="done">DONE</HomeAct></>}
          </span>
        </div>
        <div className={`bdrawer${drawer === 'add' || drawer === 'ov' ? ' open' : ''}`}>
          <div className="bd-head"><h2>{drawer === 'add' ? 'ADD TILE' : 'OVERFLOW'}</h2>
            <span className="sub">{drawer === 'add' ? 'registered summary tiles · each opens its home screen'
              : 'tiles you moved off the grid · kept, never deleted'}</span>
            <HomeAct className="x" action="close-drawer">ESC ✕</HomeAct></div>
          <div style={{ overflow: 'hidden' }}>{drawer === 'add' ? <div className="addg">
            {(Object.keys(TILES) as TileId[]).map(id => <HomeAct key={id} attributes={{ 'data-add': id }}
              className={`at2${onGrid.has(id) ? ' on' : ''}${lay.over.includes(id) ? ' ov' : ''}`}
              action="add" value={id}><b>{TILES[id].t}</b>
              <span className="s">{SPANS[id].map(s => s.replace('x', '×')).join(' ')}</span>
              <span className="d">{TILES[id].src}</span></HomeAct>)}</div>
            : lay.over.length ? <div className="addg">{lay.over.map(id => <HomeAct key={id} className="at2 ov"
              action="restore" value={id}><b>{TILES[id].t}</b><span className="s">RESTORE</span>
              <span className="d">{TILES[id].src}</span></HomeAct>)}</div>
              : <div className="empty" style={{ padding: 26 }}>Overflow is empty. When the grid is full, a
                tile you
                bump lands here instead of being lost.</div>}</div>
        </div>
        <HomeAct className={`scrim${tx ? ' open' : ''}`} action="close-card" label="Close transaction" />
        <div className={`card small${tx ? ' open' : ''}`} role="dialog" aria-modal="true">{tx && <>
          <div className="ch2"><span className="tx">{tx.what.split(' ')[0].toUpperCase()}</span>
            <span className="m">{shortTime(tx.when)} · league ledger</span>
            <HomeAct className="cardx" action="close-card">ESC ✕</HomeAct></div>
          <h2 className="hd">{tx.who}</h2>
          <div className="kvc"><div className="kv"><span>What</span><b>{tx.what}</b></div>
            <div className="kv"><span>Detail</span><b>{tx.why || '—'}</b></div></div>
          <div className="cfoot"><span className="m">recorded {shortTime(tx.when)}</span>
            <span style={{ display: 'flex', gap: 8 }}><HomeAct className="btn g2"
              action="navigate" open={TILES.wire.open}>OPEN IN LEDGER ›</HomeAct></span></div></>}</div>
        <div className={`sdrawer${drawer === 'al' ? ' open' : ''}`}>
          <div className="bd-head"><h2>ALERTS</h2><span className="sub">{alertsLost ? 'check unavailable'
            : 'commissioner jobs board · your cap check'}</span>
            <HomeAct className="x" action="close-drawer">ESC ✕</HomeAct></div>
          <div style={{ overflow: 'hidden' }}>{alertsLost ? <div className="al"><span className="s"
            style={{ background: 'var(--edge)' }} /><div className="b"><div className="t1">{unavailable}</div>
            <div className="t2">{alertsLost}. Nothing here means all-clear.</div></div></div>
            : alerts.length ? alerts.map(a => <div className="al" key={a.id}><span className="s"
              style={{ background: a.level === 'red' ? 'var(--red)' : 'var(--amber)' }} />
              <div className="b"><div className="t1">{a.title}</div><div className="t2">{a.words}
                {a.when ? ` · closes ${shortTime(a.when)}` : ''}</div></div></div>)
            : <div className="empty" style={{ padding: 26 }}>No blocked jobs, no deadlines open, and your cap
              is in line.</div>}</div>
        </div>
      </main>
      <aside className="insp" aria-label="Inspector">
        <div className="ihead"><div className="top"><span className="on">MY FRANCHISE</span>
          <span className="m" style={{ font: '11px var(--mono)', color: 'var(--ink3)' }}>{row?.division ??
            ''}</span></div>
          <div className="nm">{nameOf(b, mine)}</div><div className="sub">{b.pulse?.name ?? ''}</div></div>
        <div className="nums"><div><b>{row?.record ?? '—'}</b><span>RECORD · SEED {row?.rank ?? '—'}</span>
          </div>
          <div><b>—</b><span>POWER · {unavailable}</span></div></div>
        <div className="idrawers" style={{ overflow: 'hidden' }}>{sel ? <>
          <div className="ch"><b>SELECTED TILE</b><span>{TILES[sel].t}</span></div><div className="kv3">
            <span>ID</span><span className="mono">{sel}</span><span>SOURCE</span><span>{TILES[sel].src}</span>
            <span>PROVIDER</span><span className="mono" style={{ fontSize: 11 }}>{TILES[sel].prov}</span>
            <span>REFRESH</span><span>{TILES[sel].refresh}</span><span>AS OF</span><span>{asOf(sel, b)}</span>
            <span>OPENS</span><span>{TILES[sel].link || '—'}</span>
            <span>SIZES</span><span>{SPANS[sel].map(s => s.replace('x', '×')).join(' · ')}</span>
            <span>STATES</span><span>loading · empty · stale · unavailable · error</span></div></>
          : <><div className="ch"><b>SELECTED TILE</b><span>none</span></div>
            <div className="empty">Click a tile to see where its data comes from, how fresh it is and where it
              opens.</div></>}
          <div className="ch" style={{ borderTop: '2px solid var(--black)' }}><b>LAYOUT</b>
            <span>saved per owner · per league</span></div><div className="kv3">
            <span>MODE</span><span>{lay.mode === 'default' ? 'Default · follows season phase' :
              'Custom · never moves on its own'}</span>
            <span>TILES</span><span>{placed.length} on grid · {lay.over.length} in overflow</span>
            <span>ALERTS</span><span>always on top · not part of the layout</span></div></div>
        <div /><div className="ifoot"><span>{edit ? 'CUSTOMIZE ON' : 'read only'}</span></div>
      </aside>
    </div>
  </div>;
}

function asOf(id: TileId, b: Brief): string {
  const state = tileState(id, b);
  if (TILES[id].missing || !TILES[id].needs.length) return '—';
  if (state === 'loading') return 'reading…';
  const at = oldestAt(id, b);
  if (!at) return 'unavailable';
  return `as of ${clock(at, id === 'mu')}${id === 'mu' ? ' · live' : ''}${state === 'stale' ? ' · stale' :
    ''}`;
}
/** Depth is the research read's nflverse depth chart report (M-048), names tagged by who holds them here. */
function Depth({ b, mine }: { b: Brief; mine: string }) {
  if (b.errors.research) return <div className="empty">{unavailable} · {b.errors.research}</div>;
  if (!b.research) return <div className="empty loading">Reading…</div>;
  const r = b.research.board.reports.find(x => x.id === 'M-048');
  if (!r || !r.rows.length) return <div className="empty">{unavailable} · {r?.note ||
    'No depth chart is held'}</div>;
  // The report names players, not IDs: only a unique exact name joins to who holds him here.
  const names = new Map<string, string[]>();
  for (const x of b.assets?.rosters.value ?? []) for (const p of x.players) {
    const n = b.assets!.players.value.find(q => q.id === p.id)?.name;
    if (n) names.set(n, [...(names.get(n) ?? []), x.franchiseId]);
  }
  return <div className="ct-wrap"><table className="ct"><thead><tr>{r.columns.map(c => <th key={c}>{c}</th>)}
    <th>HELD BY</th></tr></thead><tbody>{r.rows.map((x, i) => {
      const held = names.get(x.cells[0]); const who = held?.length === 1 ? held[0] : undefined;
      return <tr key={i} className={who === mine ? 'me' : ''}>{x.cells.map((c, j) => <td key={j}>{c}</td>)}
        <td>{who === mine ? 'Yours' : who ? nameOf(b, who) : ''}</td></tr>;
    })}</tbody></table></div>;
}
function firstPick(b: Brief, mine: string): string {
  const p = b.draft?.order.flatMap(o => o.picks).find(x => x.owner === mine);
  return p ? `${p.round}.${String(p.slot).padStart(2, '0')}` : '—';
}
function standing(a: number | undefined, c: number | undefined, final: boolean): string {
  if (a === undefined || c === undefined) return 'you play';
  const verb = a > c ? (final ? 'you beat' : 'you lead') : a < c ? (final ? 'you lost to' : 'you trail')
    : (final ? 'you tied' : 'level with');
  return verb;
}
