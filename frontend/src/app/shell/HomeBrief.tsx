import { useEffect, useMemo, useState } from 'react';
import './home.css';
import type { HostedPort } from '../commissioner/hostedRoom';
import { readHomeBrief, type HomeBrief as Brief } from './homeReads';
import { bump, current, defaultLayout, layoutKey, pack, parseLayout, place, remove, SPANS, swap,
  type Layout, type Phase, type Span, type TileId } from './homeLayout';
import { go, isFinal, myGame, myRow, nameOf, pts, shortTime, TileBody, TILES, topFive } from './homeTiles';
import { browserStorage } from './franchise';
import { money, unavailable } from './hqModel';
import { pointsLabel } from '../data/nativePulse';

type Tab = 'brief' | 'depth' | 'stats';
const ALERTS_MISSING = 'No alert read covers an app league yet';

/** HomeBrief is Home as the locked mock draws it (docs/ui/mocks/home-mock-v5.html), fed by the reads
 *  in homeReads. Layout is saved per owner per league; nothing here invents a number. */
export default function HomeBrief({ league, mine, port, initial }: { league: string; mine: string;
  port: HostedPort; initial?: Brief }) {
  const [b, setB] = useState<Brief>(initial ?? { errors: {} });
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
    void readHomeBrief(league, mine, port, r => { if (live) setB(r); });
    return () => { live = false; };
  }, [league, mine, port, initial]);
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
      pts(b.results?.scores.find(s => s.franchise === opp)), isFinal(g.status))} ${nameOf(b, opp)} ${myPts} – ${opPts}${
      next ? ` · ${next.label} ${shortTime(next.at!)}` : ''}`
    : next ? `${next.label} ${shortTime(next.at!)}` : b.pulse?.weekNote || 'Reading the league…';
  const stats: [string, string][] = phase === 'in'
    ? [[row?.record ?? '—', 'RECORD'], [row ? String(row.rank) : '—', 'SEED'], ['—', 'POWER'],
      [room?.room === undefined ? '—' : money(room.room), 'CAP ROOM']]
    : [['—', 'POWER · DYN'], [firstPick(b, mine), 'FIRST PICK'],
      [b.runway?.[1]?.room === undefined ? '—' : money(b.runway[1].room), `${b.runway?.[1]?.season ?? ''} ROOM`.trim()], ['—', 'ROSTER']];
  const ticker = topFive(b);
  const tickerHtml = ticker.map(([p, xs]) => <span className="tp" key={p}><b>{p}</b>{xs.map((x, i) =>
    <span className={`tpi${i ? '' : ' one'}`} key={x.player}><u>{i + 1}.</u>{x.name} <i>{pointsLabel(pts(x))}</i>
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
  return <div className={cls} data-testid="home-brief"
    onKeyDown={e => { if (e.key === 'Escape') { setCard(undefined); setPick(undefined); setDrawer(undefined); } }}>
    <div className="shell hb-shell">
      <main className="work">
        <div className="ws-tabs" role="tablist">
          {(['brief', 'depth', 'stats'] as const).map(t => <button role="tab" key={t} aria-selected={tab === t}
            onClick={() => setTab(t)}>{{ brief: 'BRIEF', depth: 'DEPTH CHARTS', stats: 'NFL STATS' }[t]}</button>)}
          <div className="lay">LAYOUT <b>{lay.mode === 'default'
            ? `DEFAULT · ${phase === 'in' ? 'IN-SEASON' : 'OFFSEASON'}` : 'MINE (custom)'}</b></div>
          <button className={`queue${edit ? ' on' : ''}`} style={{ marginLeft: 'auto' }}
            onClick={() => { setEdit(!edit); setPick(undefined); setDrawer(undefined); }}>
            <span className="ql">CUSTOMIZE</span></button>
        </div>
        <section className="now" aria-label="Now">
          <div className="nl"><span className="tag">{phase === 'off' ? 'OFFSEASON' : g && isFinal(g.status) ? 'FINAL' : 'LIVE'}</span>{nowLine}</div>
          {stats.map(([v, l]) => <div className="st" key={l}><b>{v}</b><span>{l}</span></div>)}
        </section>
        <div className="aband unavail" title="Open all alerts" role="button" tabIndex={0}
          onClick={() => setDrawer(drawer === 'al' ? undefined : 'al')}>
          <span className="ak">ALERTS UNAVAILABLE</span>
          <span className="at">{ALERTS_MISSING} · this is not an all-clear</span>
          <span className="more">DETAILS ▸</span></div>
        <div className="tick" title="Hover to pause"><span className="tl0">TOP 5 · WK {week ?? '—'}
          {g && !isFinal(g.status) ? ' LIVE' : ' FINAL'}</span>
          <div className="tkw"><div className="tkr">{ticker.length ? <>{tickerHtml}{tickerHtml}</>
            : <span className="tp">{b.errors.results ? `${unavailable} · ${b.errors.results}` : 'No scored starters yet'}</span>}
          </div></div></div>
        <section className="grid" aria-label="Tiles">{placed.map(p => {
          const d = TILES[p.id];
          return <article key={p.id} className={`t${sel === p.id ? ' sel' : ''}${drag && drag !== p.id ? '' : ''}`}
            data-id={p.id} draggable={edit}
            onDragStart={() => setDrag(p.id)} onDragEnd={() => setDrag(undefined)}
            onDragOver={e => { if (drag) e.preventDefault(); }}
            onDrop={e => { e.preventDefault(); if (!drag || drag === p.id) return;
              const n = swap(lay, phase, drag, p.id);
              if (n) save(n); else setMsg("Those two sizes don't fit swapped · resize one first");
              setDrag(undefined); }}
            onClick={() => clickTile(p.id)}
            style={{ gridRow: `${p.row + 1}/span ${p.h}`, gridColumn: `${p.col + 1}/span ${p.w}` }}>
            <div className="th"><span className="ed"><span className="grip" title="Drag to move">⠿</span></span>
              <b>{d.t}</b><span className="src">{d.src}</span>
              {d.link && <button className="lnk" onClick={e => { e.stopPropagation(); go(d.open); }}>{d.link}</button>}
              <span className="ed" style={{ marginLeft: 'auto' }}>
                {(['1x1', '2x1', '1x2'] as const).map(s => <button key={s} className="sp" aria-pressed={s === p.span}
                  disabled={!SPANS[p.id].includes(s)}
                  onClick={e => { e.stopPropagation(); resize(p.id, s); }}>{s.replace('x', '×')}</button>)}
                <button className={`pin${lay.pins[p.id] ? ' on' : ''}`} title="Pin this screen to MY SHORTCUTS"
                  onClick={e => { e.stopPropagation(); const pins = { ...lay.pins };
                    if (pins[p.id]) delete pins[p.id]; else pins[p.id] = true; save({ ...lay, pins }); }}>
                  {lay.pins[p.id] ? '★' : '☆'}</button>
                <button className="off" title="Move to overflow"
                  onClick={e => { e.stopPropagation(); save(remove(lay, phase, p.id)); }}>✕</button></span></div>
            <div className="tb"><TileBody id={p.id} b={b} mine={mine}
              onTx={i => { if (!edit) setCard(i); }} /></div>
            <div className="tf"><span>{asOf(b)}</span><span className="more" /></div>
          </article>;
        })}</section>
        <section className={`wsx${tab === 'depth' ? ' on' : ''}`} aria-label="Depth charts">
          <div className="empty">{unavailable} · No depth-chart read exists (Sleeper depth order / ESPN source TBD)</div></section>
        <section className={`wsx${tab === 'stats' ? ' on' : ''}`} aria-label="NFL stats">
          <div className="empty">{unavailable} · No league-wide NFL stats table read for an app league</div></section>
        <div className="tray">
          <div className="pulls">
            <button className="pull" aria-expanded={drawer === 'al'}
              onClick={() => setDrawer(drawer === 'al' ? undefined : 'al')}>ALERTS <b>?</b></button>
            {edit && <button className="pull" onClick={() => setDrawer(drawer === 'add' ? undefined : 'add')}>
              ADD TILE <b>+</b></button>}
            <button className="pull" onClick={() => setDrawer(drawer === 'ov' ? undefined : 'ov')}>
              OVERFLOW <b>{lay.over.length}</b></button>
          </div>
          <span className={`msg${pick ? ' ask' : ''}`}>{pick
            ? `Grid full · click the tile to move to OVERFLOW to make room for ${TILES[pick.id].t} ${pick.span.replace('x', '×')}`
            : msg}</span>
          <span className="sp" />
          <span style={{ display: 'flex', gap: 8 }}>{pick
            ? <button className="btn" onClick={() => setPick(undefined)}>CANCEL</button>
            : edit && <>
              <button className="btn" onClick={() => save({ ...lay, mode: 'default', tiles: [] })}>USE THIS PHASE'S DEFAULT</button>
              <button className="btn" onClick={() => save(defaultLayout())}>RESET TO DEFAULT</button>
              <button className="btn solid" onClick={() => { setEdit(false); setDrawer(undefined); }}>DONE</button></>}
          </span>
        </div>
        <div className={`bdrawer${drawer === 'add' || drawer === 'ov' ? ' open' : ''}`}>
          <div className="bd-head"><h2>{drawer === 'add' ? 'ADD TILE' : 'OVERFLOW'}</h2>
            <span className="sub">{drawer === 'add' ? 'registered summary tiles · each opens its home screen'
              : 'tiles you moved off the grid · kept, never deleted'}</span>
            <button className="x" onClick={() => setDrawer(undefined)}>ESC ✕</button></div>
          <div style={{ overflow: 'hidden' }}>{drawer === 'add' ? <div className="addg">
            {(Object.keys(TILES) as TileId[]).map(id => <button key={id} data-add={id}
              className={`at2${onGrid.has(id) ? ' on' : ''}${lay.over.includes(id) ? ' ov' : ''}`}
              onClick={() => { if (!onGrid.has(id)) add(id); }}><b>{TILES[id].t}</b>
              <span className="s">{SPANS[id].map(s => s.replace('x', '×')).join(' ')}</span>
              <span className="d">{TILES[id].src}</span></button>)}</div>
            : lay.over.length ? <div className="addg">{lay.over.map(id => <button key={id} className="at2 ov"
              onClick={() => add(id)}><b>{TILES[id].t}</b><span className="s">RESTORE</span>
              <span className="d">{TILES[id].src}</span></button>)}</div>
              : <div className="empty" style={{ padding: 26 }}>Overflow is empty. When the grid is full, a tile you
                bump lands here instead of being lost.</div>}</div>
        </div>
        <div className={`scrim${tx ? ' open' : ''}`} onClick={() => setCard(undefined)} />
        <div className={`card small${tx ? ' open' : ''}`} role="dialog" aria-modal="true">{tx && <>
          <div className="ch2"><span className="tx">{tx.what.split(' ')[0].toUpperCase()}</span>
            <span className="m">{shortTime(tx.when)} · league ledger</span>
            <button className="cardx" onClick={() => setCard(undefined)}>ESC ✕</button></div>
          <h2 className="hd">{tx.who}</h2>
          <div className="kvc"><div className="kv"><span>What</span><b>{tx.what}</b></div>
            <div className="kv"><span>Detail</span><b>{tx.why || '—'}</b></div></div>
          <div className="cfoot"><span className="m">recorded {shortTime(tx.when)}</span>
            <span style={{ display: 'flex', gap: 8 }}><button className="btn g2"
              onClick={() => go(TILES.wire.open)}>OPEN IN LEDGER ›</button></span></div></>}</div>
        <div className={`sdrawer${drawer === 'al' ? ' open' : ''}`}>
          <div className="bd-head"><h2>ALERTS</h2><span className="sub">check unavailable</span>
            <button className="x" onClick={() => setDrawer(undefined)}>ESC ✕</button></div>
          <div style={{ overflow: 'hidden' }}><div className="al"><span className="s" style={{ background: 'var(--edge)' }} />
            <div className="b"><div className="t1">{unavailable}</div>
              <div className="t2">{ALERTS_MISSING}. Lineup, trade, IR and bye checks read the MFL mirror only. Nothing
                here means all-clear.</div></div></div></div>
        </div>
      </main>
      <aside className="insp" aria-label="Inspector">
        <div className="ihead"><div className="top"><span className="on">MY FRANCHISE</span>
          <span className="m" style={{ font: '11px var(--mono)', color: 'var(--ink3)' }}>{row?.division ?? ''}</span></div>
          <div className="nm">{nameOf(b, mine)}</div><div className="sub">{b.pulse?.name ?? ''}</div></div>
        <div className="nums"><div><b>{row?.record ?? '—'}</b><span>RECORD · SEED {row?.rank ?? '—'}</span></div>
          <div><b>—</b><span>POWER · {unavailable}</span></div></div>
        <div className="idrawers" style={{ overflow: 'hidden' }}>{sel ? <>
          <div className="ch"><b>SELECTED TILE</b><span>{TILES[sel].t}</span></div><div className="kv3">
            <span>ID</span><span className="mono">{sel}</span><span>SOURCE</span><span>{TILES[sel].src}</span>
            <span>PROVIDER</span><span className="mono" style={{ fontSize: 11 }}>{TILES[sel].prov}</span>
            <span>REFRESH</span><span>{TILES[sel].refresh}</span><span>AS OF</span><span>{asOf(b)}</span>
            <span>OPENS</span><span>{TILES[sel].link || '—'}</span>
            <span>SIZES</span><span>{SPANS[sel].map(s => s.replace('x', '×')).join(' · ')}</span>
            <span>STATES</span><span>loading · empty · stale · unavailable · error</span></div></>
          : <><div className="ch"><b>SELECTED TILE</b><span>none</span></div>
            <div className="empty">Click a tile to see where its data comes from, how fresh it is and where it opens.</div></>}
          <div className="ch" style={{ borderTop: '2px solid var(--black)' }}><b>LAYOUT</b>
            <span>saved per owner · per league</span></div><div className="kv3">
            <span>MODE</span><span>{lay.mode === 'default' ? 'Default · follows season phase' : 'Custom · never moves on its own'}</span>
            <span>TILES</span><span>{placed.length} on grid · {lay.over.length} in overflow</span>
            <span>ALERTS</span><span>always on top · not part of the layout</span></div></div>
        <div /><div className="ifoot"><span>{edit ? 'CUSTOMIZE ON' : 'read only'}</span></div>
      </aside>
    </div>
  </div>;
}

function asOf(b: Brief): string {
  return b.pulse ? `revision ${b.pulse.revision}` : 'reading…';
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
