// Home board and its pieces, after the mock's grid.js (24x14 grid, piece frame) and pieces.js (R[id]).
// Each body returns the mock's { fb, b, f }: filter bar, body and footer line.
import { useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import {
  ctDateTime, isInjuryNews, openExternalArticle, shortNewsAge, type FeedNewsItem,
} from '../research/espnNews';
import { left, nameOf } from '../shell/homeTiles';
import { HQAct } from '../shell/HQAct';
import type { ThinHomeEntry } from '../shell/ThinHome';
import { Icon, type IconName } from './icons';
import { ctTime, type Brief, type DocketItem, type Lane, type Open, type V3League } from './model';

const COLS = 24, ROWS = 14, GAP = 4;
type PieceId = 'docket' | 'week' | 'today' | 'nflnews' | 'deadlines' | 'shortcuts';
type Inst = { iid: string; pid: PieceId; x: number; y: number; w: number; h: number };
type Board = { id: string; name: string; icon: IconName; insts: Inst[] };
type PieceDef = { title: string; d: string; pal: string; live?: boolean };

const PIECES: Record<PieceId, PieceDef> = {
  docket: { title: 'Docket', d: 'What is waiting, on whom, in every league.', pal: 'pal-slate' },
  week: { title: 'This week across my leagues', d: 'Every matchup you are in this week, on one plate.',
    pal: 'pal-slate', live: true },
  today: { title: 'My players today', d: 'Your starters across every league, grouped by kickoff.',
    pal: 'pal-slate', live: true },
  nflnews: { title: 'NFL news', d: 'Injuries, returns and moves for the players that matter to you.',
    pal: 'pal-slate' },
  deadlines: { title: 'Deadlines', d: 'The next clocks that matter, soonest first.', pal: 'pal-slate' },
  shortcuts: { title: 'Shortcuts', d: 'Your pinned places, one press away.', pal: 'pal-slate' },
};
const mk = (pid: PieceId, x: number, y: number, w: number, h: number): Inst => ({
  iid: pid, pid, x, y, w, h });
// The mock's "Today" starter board (Home.starters.today).
export const TODAY: Board = { id: 'today', name: 'Today', icon: 'home', insts: [mk('docket', 0, 0, 7, 9),
  mk('week', 7, 0, 10, 6), mk('today', 17, 0, 7, 9), mk('nflnews', 7, 6, 10, 8), mk('deadlines', 0, 9, 7, 5),
  mk('shortcuts', 17, 9, 7, 5)] };
const sizeOf = (w: number, h: number) => w >= 20 && h >= 12 ? 'XL'
  : (w >= 10 && h >= 6) || (w >= 7 && h >= 8) ? 'L' : w < 5 || h < 3 ? 'S' : 'M';

export function Badge({ L, big }: { L?: V3League | null; big?: boolean }) {
  if (!L) return <span className={`bdg all${big ? ' big' : ''}`} title="All leagues">≡</span>;
  return <span className={`bdg${big ? ' big' : ''}`} style={{ '--c': L.c } as CSSProperties}
    title={L.name}>{L.bd}</span>;
}
function Sh({ children }: { children: ReactNode }) { return <div className="sh">{children}</div>; }
function St({ children }: { children: ReactNode }) {
  return <div className="st"><div className="eng">{children}</div></div>;
}

export type Ctx = { leagues: V3League[]; briefs: Record<string, Brief>; docket: DocketItem[]; now: number;
  open: Open; openPlayer?: (league: V3League, playerId: string) => void; loading: boolean };
type PieceCtx = Ctx & { size: string; w: number };
type Body = { fb?: ReactNode; b: ReactNode; f?: string };

export function DkRow({ d, L, sm, open }: { d: DocketItem; L?: V3League; sm?: boolean; open: Open }) {
  const cls = `dk${sm ? ' sm' : ''}${d.urgent ? ' u' : ''}${d.bad ? ' bad' : ''} unread`;
  return <div className={cls}>
    <Badge L={L} /><span className="t ell">{d.t}</span>
    {!sm && <span className="s ell">{d.lane === 'need' ? 'held by ' : ''}{d.who}</span>}
    <span className="ck">{d.clock}</span>
    {!sm && <HQAct className="btn sm" disabled={!L || !d.entry} label={`${d.btn}: ${d.t}`}
      onAction={() => { if (L && d.entry) open(L, d.entry); }}>{d.btn}</HQAct>}</div>;
}

const LANES: [Lane, string][] = [['need', 'Needs you'], ['wait', 'Waiting on'], ['land', 'Landed']];
function docket(c: PieceCtx, lane: number, setLane: (n: number) => void): Body {
  const by = (k: Lane) => c.docket.filter(d => d.lane === k);
  const L = (d: DocketItem) => c.leagues.find(l => l.id === d.lg);
  const none = (k: Lane) => <St>{c.loading ? 'Reading your leagues…' : `Nothing ${k === 'need' ? 'needs you'
    : k === 'wait' ? 'is waiting on anyone' : 'has landed'} right now.`}</St>;
  const sel = LANES[lane][0];
  const rest = c.size !== 'L' ? null : LANES.filter((x, i) => i !== lane && by(x[0]).length).map(([k, t]) =>
    <div key={k}><Sh>{t} · {by(k).length}</Sh>
      {by(k).map(d => <DkRow key={d.id} d={d} L={L(d)} sm open={c.open} />)}</div>);
  const rows = by(sel).map(d =>
    <DkRow key={d.id} d={d} L={L(d)} sm={c.size === 'M' && c.w < 7} open={c.open} />);
  return {
    fb: <div className="fbar">{LANES.map(([k, t], i) => <HQAct key={k} className="chip" active={lane === i}
      onAction={() => setLane(i)}>{`${t} ${by(k).length}`}</HQAct>)}</div>,
    b: <>{rows.length ? rows : none(sel)}{rest}</>,
    f: 'nothing leaves the app · since you were last here',
  };
}

const span2: CSSProperties = { gridColumn: 'span 2', textAlign: 'right' };
const WEEK_COLS = 'auto minmax(0,1.2fr) 56px 14px 56px minmax(0,1fr) 70px 38px';
function week(c: PieceCtx): Body {
  const big = c.size === 'L' || c.size === 'XL';
  const rows = c.leagues.map(L => {
    const b = c.briefs[L.id]?.brief, games = b?.pulse?.results?.games ?? [];
    const mine = L.franchise ? games.find(g => g.home === L.franchise || g.away === L.franchise) : undefined;
    const g = mine ?? games[0];
    if (!g) {
      const err = c.briefs[L.id]?.error;
      return <div key={L.id} className={`row${big ? ' two' : ''}`}
        style={{ gridTemplateColumns: 'auto minmax(0,1fr)' }}>
        <Badge L={L} /><span className="ell d2">{err ? `Unavailable · ${err}`
          : b ? (b.pulse?.weekNote || 'No matchup this week') : 'Reading…'}</span></div>;
    }
    const flip = mine && g.away === L.franchise;
    const hs = g.homePoints ?? g.homeProvisional ?? 0, as = g.awayPoints ?? g.awayProvisional ?? 0;
    const [ms, os, mn, on] = flip ? [as, hs, g.awayName, g.homeName] : [hs, as, g.homeName, g.awayName];
    const st = b?.pulse?.standings?.rows.find(r => r.franchise === L.franchise);
    const sub = mine
      ? `${L.name}${st ? ` · ${st.record} · seed ${st.rank}` : ''}${g.words ? ` · ${g.words}` : ''}`
      : `${L.name} · you run this league and hold no team · the closest game is shown`;
    return <HQAct key={L.id} className={`row${big ? ' two' : ''}`} label={`Open ${L.name} scoring`}
      onAction={() => c.open(L, 'pulse')} style={{ gridTemplateColumns: WEEK_COLS }}>
      <Badge L={L} /><span className="ell">{mine ? <b>{mn}</b> : <span className="d2">{mn}</span>}</span>
      <span className={`v${ms >= os ? '' : ' d2'}`}>{ms.toFixed(1)}</span><span className="m">–</span>
      <span className={`v${os > ms ? '' : ' d2'}`} style={{ textAlign: 'left' }}>{os.toFixed(1)}</span>
      <span className="ell d2">{on}</span>
      <span className="m" style={span2}>{mine ? g.status.toUpperCase() : 'NO TEAM'}</span>
      {big && <span className="sub ell">{sub}</span>}</HQAct>;
  });
  const asOf = Object.values(c.briefs).map(b => b.brief?.pulse?.results?.evidence?.asOf).find(Boolean);
  return { b: <>{rows}</>, f: `${asOf ? `as of ${ctTime(asOf)}` : 'current week'} · sorted by league` };
}

type Starter = { id: string; name: string; tm: string; locked: boolean; pts?: number; proj?: number;
  lg: V3League[] };
function today(c: PieceCtx): Body {
  const by = new Map<string, Starter>();
  for (const L of c.leagues) {
    const b = c.briefs[L.id]?.brief;
    if (!b || !L.franchise) continue;
    const scores = b.pulse?.results?.scores.find(s => s.franchise === L.franchise)?.starters ?? [];
    for (const p of b.matchup?.lineups[L.franchise] ?? []) {
      const cur = by.get(p.player)
        ?? { id: p.player, name: nameOf(b, p.player), tm: p.team, locked: p.locked, lg: [] };
      cur.pts = scores.find(s => s.player === p.player)?.points ?? cur.pts;
      cur.proj = b.matchup?.projections[p.player] ?? cur.proj;
      cur.lg.push(L);
      by.set(p.player, cur);
    }
  }
  const ps = [...by.values()], multi = c.leagues.filter(l => l.team).length > 1;
  if (!ps.length) return { b: <St>{c.loading ? 'Reading your lineups…'
    : 'No starters set in your leagues this week.'}</St>, f: 'starters from your saved lineups' };
  const grp: [string, Starter[]][] = [['On the field now', ps.filter(p => p.locked)],
    ['Later today', ps.filter(p => !p.locked)]];
  return { b: <>{grp.map(([k, v]) => <div key={k}><Sh>{k} · {v.length}</Sh>{v.map(p =>
    <div key={p.id} className="row" style={{ gridTemplateColumns: '30px minmax(0,1fr) auto 44px 44px' }}>
      <span className="pos" /><span className="ell">{p.name} <span className="m">{p.tm}</span></span>
      <span>{multi ? p.lg.map(l => <Badge key={l.id} L={l} />) : null}</span>
      <span className={`m${p.locked ? ' b' : ''}`}>{p.locked ? 'LIVE' : ''}</span>
      <span className="v">{(p.locked ? p.pts ?? 0 : p.proj ?? 0).toFixed(1)}</span></div>)}</div>)}</>,
  f: multi ? 'badges = leagues where you start him' : 'live · from the league server' };
}

function deadlines(c: PieceCtx): Body {
  const d = c.leagues.flatMap(L => (c.briefs[L.id]?.brief?.calendar ?? []).flatMap(e => {
    const at = e.closes || e.at;
    return at && Date.parse(at) > c.now ? [{ L, e, at }] : [];
  })).sort((a, b) => Date.parse(a.at) - Date.parse(b.at)).slice(0, 8);
  if (!d.length) return { b: <St>{c.loading ? 'Reading your calendars…'
    : 'No dated deadlines in your leagues.'}</St>, f: 'soonest first' };
  return { b: <>{d.map(({ L, e, at }) =>
    <div key={L.id + e.id} className="row dlr"
      style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto' }}>
      <Badge L={L} /><HQAct className="ell dlgo" onAction={() => c.open(L, 'war')}>{e.label}</HQAct>
      <span className={`v${Date.parse(at) - c.now < 3_600_000 ? ' a' : ''}`}>{left(at, c.now)}</span>
      <HQAct className="rem" active={false} label="Remind me 15 minutes before">○</HQAct></div>)}</>,
  f: 'soonest first · ○ remind me' };
}

function shortcuts(c: PieceCtx): Body {
  const L = c.leagues.find(l => l.team) ?? c.leagues[0], C = c.leagues.find(l => l.commish);
  const sc: [V3League | undefined, ThinHomeEntry, string][] = [[L, 'lineup', 'Lineup'],
    [L, 'trades', 'Trade desk'], [L, 'pulse', 'Live scoring'], [C, 'control', 'Commissioner jobs'],
    [L, 'pulse', 'Standings']];
  const shown = sc.filter(([l, e]) => l && ((e !== 'lineup' && e !== 'trades') || l.team));
  return { b: <div className="scg">{shown.map(([l, e, t]) =>
    <HQAct key={t} className="scb" label={t} onAction={() => { if (l) c.open(l, e); }}>
      <Badge L={l} /><span className="ell">{t}</span></HQAct>)}</div>,
  f: 'pin any tab with the star in the rail' };
}

type NewsFilter = 'all' | 'mine' | 'injuries' | 'wire';
type NewsMention = {
  league: V3League; playerId: string; name: string; onMyRoster: boolean; onAnyRoster: boolean;
  knownRosters: boolean;
};
type HomeNewsRow = { item: FeedNewsItem; mentions: NewsMention[]; injury: boolean };

const NEWS_CHIPS: [NewsFilter, string][] = [
  ['all', 'All'], ['mine', 'My players'], ['injuries', 'Injuries'], ['wire', 'On the wire'],
];

function collectHomeNews(c: PieceCtx): { rows: HomeNewsRow[]; asOf: string; stale: boolean; reason: string } {
  const byId = new Map<string, HomeNewsRow>();
  let asOf = '', stale = false, reason = '';
  for (const L of c.leagues) {
    const entry = c.briefs[L.id];
    const feed = entry?.news;
    if (!feed) continue;
    if (feed.asOf && (!asOf || feed.asOf > asOf)) asOf = feed.asOf;
    if (feed.stale) stale = true;
    if (!reason && feed.reason) reason = feed.reason;
    const snap = entry?.brief?.assets;
    const playersById = new Map((snap?.players.value ?? []).map(p => [p.id, p.name]));
    const rostered = new Set<string>();
    const mine = new Set<string>();
    const knownRosters = Boolean(snap?.rosters.value.length);
    for (const r of snap?.rosters.value ?? []) {
      for (const p of r.players) {
        rostered.add(p.id);
        if (L.franchise && r.franchiseId === L.franchise) mine.add(p.id);
      }
    }
    for (const item of feed.items) {
      let row = byId.get(item.id);
      if (!row) {
        row = { item, mentions: [], injury: isInjuryNews(item) };
        byId.set(item.id, row);
      }
      for (const pid of item.players) {
        if (row.mentions.some(m => m.league.id === L.id && m.playerId === pid)) continue;
        row.mentions.push({
          league: L, playerId: pid, name: playersById.get(pid) || `Player ${pid}`,
          onMyRoster: mine.has(pid), onAnyRoster: rostered.has(pid), knownRosters,
        });
      }
    }
  }
  const rows = [...byId.values()].sort((a, b) =>
    (Date.parse(b.item.published) || 0) - (Date.parse(a.item.published) || 0));
  return { rows, asOf, stale, reason };
}

function matchesNewsFilter(row: HomeNewsRow, filter: NewsFilter): boolean {
  if (filter === 'all') return true;
  if (filter === 'mine') return row.mentions.some(m => m.onMyRoster);
  if (filter === 'injuries') return row.injury;
  return row.mentions.some(m => m.knownRosters && !m.onAnyRoster);
}

function emptyNewsWords(filter: NewsFilter, loading: boolean, reason: string): string {
  if (loading) return 'Reading the news…';
  if (filter === 'mine') return 'No recent ESPN news mentions players on your roster.';
  if (filter === 'injuries') return 'No injury updates in the current ESPN news feed.';
  if (filter === 'wire') return 'No recent ESPN news mentions an unrostered player.';
  return reason || 'No ESPN news is available from this league server yet.';
}

function news(c: PieceCtx, filter: NewsFilter, setFilter: (f: NewsFilter) => void): Body {
  const { rows, asOf, stale, reason } = collectHomeNews(c);
  const count = (f: NewsFilter) => rows.filter(r => matchesNewsFilter(r, f)).length;
  const shown = rows.filter(r => matchesNewsFilter(r, filter));
  const fb = <div className="fbar">{NEWS_CHIPS.map(([k, label]) =>
    <HQAct key={k} className="chip" active={filter === k}
      onAction={() => setFilter(k)}>{`${label} ${count(k)}`}</HQAct>)}</div>;
  const b = !shown.length ? <St>{emptyNewsWords(filter, c.loading, reason)}</St>
    : <>{shown.map(({ item, mentions, injury }) => {
      const isMine = mentions.some(m => m.onMyRoster);
      const uniquePlayers = mentions.filter((m, i) =>
        mentions.findIndex(x => x.playerId === m.playerId) === i);
      return <div key={item.id} className={`news${isMine ? ' mine' : ''}`}>
        <span className="m" title={ctDateTime(item.published)}>
          {shortNewsAge(item.published, c.now)}</span>
        <b>
          {item.url
            ? <HQAct className="espn-news-link" attributes={{ 'data-url': item.url }}
              label={`Open article on ${item.source}: ${item.headline}`}
              onAction={() => openExternalArticle(item.url)}>{item.headline}</HQAct>
            : item.headline}
          <small>{item.source}{injury ? ' · INJURY' : ''}</small>
        </b>
        {uniquePlayers.length > 0 && <div className="news-players">
          {uniquePlayers.map(m => <HQAct key={`${m.league.id}:${m.playerId}`}
            className="tag" attributes={{ 'data-a': 'player' }}
            label={`Open ${m.name} player card`}
            onAction={() => c.openPlayer?.(m.league, m.playerId)}>{m.name}</HQAct>)}
        </div>}
        {item.description && <p>{item.description}</p>}
      </div>;
    })}</>;
  const foot = asOf
    ? `as of ${ctTime(asOf)} CT · ESPN${stale ? ' · cached copy' : ''}`
    : 'ESPN · every player in the news';
  return { fb, b, f: foot };
}

function Piece({ inst, c, place, lane, setLane, newsFilter, setNewsFilter }: {
  inst: Inst; c: Ctx; place: (i: Inst) => CSSProperties;
  lane: number; setLane: (n: number) => void;
  newsFilter: NewsFilter; setNewsFilter: (f: NewsFilter) => void;
}) {
  const p = PIECES[inst.pid], size = sizeOf(inst.w, inst.h), ctx: PieceCtx = { ...c, size, w: inst.w };
  const r = inst.pid === 'docket' ? docket(ctx, lane, setLane) : inst.pid === 'week' ? week(ctx)
    : inst.pid === 'today' ? today(ctx) : inst.pid === 'deadlines' ? deadlines(ctx)
      : inst.pid === 'shortcuts' ? shortcuts(ctx) : news(ctx, newsFilter, setNewsFilter);
  const live = p.live
    && c.leagues.some(L => c.briefs[L.id]?.brief?.pulse?.results?.evidence?.live === 'live');
  return <div className={`pc ${p.pal} sz-${size}`} data-iid={inst.iid} tabIndex={0} aria-label={p.title}
    style={place(inst)}>
    <div className="ph" data-tip={p.d}><span className="grip">⋮⋮</span><Badge />
      <b className="pt">{p.title}</b>
      <span className="sp" />
      {live && <span className="lv"><i className="lamp live" /><span>LIVE</span></span>}
      <HQAct className="pm" label="Piece menu"><Icon n="dots" /></HQAct></div>
    <div className="pb">{r.fb}{r.b}</div>
    {r.f && <div className="pf"><span className="ell">{r.f}</span></div>}</div>;
}

export function HomeBoard({ c }: { c: Ctx }) {
  const grid = useRef<HTMLDivElement>(null);
  const [cell, setCell] = useState<{ cw: number; ch: number }>();
  const [lane, setLane] = useState(0);
  const [newsFilter, setNewsFilter] = useState<NewsFilter>('all');
  useLayoutEffect(() => {
    const g = grid.current;
    if (!g) return;
    const fit = () => setCell({ cw: g.clientWidth / COLS, ch: g.clientHeight / ROWS });
    fit();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(fit);
    ro.observe(g);
    return () => ro.disconnect();
  }, []);
  const place = (r: Inst): CSSProperties => !cell ? { visibility: 'hidden' } : {
    transform: `translate(${r.x * cell.cw + GAP}px,${r.y * cell.ch + GAP}px)`,
    width: r.w * cell.cw - GAP * 2, height: r.h * cell.ch - GAP * 2 };
  const vars = cell ? { '--cw': `${cell.cw}px`, '--ch': `${cell.ch}px` } as CSSProperties : undefined;
  return <>
    <div className="ws-tabs"><span className="ttl"><Icon n="home" />HOME</span>
      <HQAct className="tb btab" selected label={`Board: ${TODAY.name}`}><Icon n={TODAY.icon} />
        <span>{TODAY.name}</span><span className="ed">▾</span></HQAct>
      <HQAct className="tb" label="Add a board"><Icon n="plus" /><span className="lb">Board</span></HQAct>
      <div className="tools"><HQAct className="key" label="Add a piece"><Icon n="plus" />
        <span className="lb">Add a piece</span></HQAct>
      <HQAct className="key" label="Edit">
        <Icon n="pencil" /><span className="lb">Edit</span></HQAct></div></div>
    <div className="wbody"><div className="grid" id="grid" ref={grid} style={vars}>
      {TODAY.insts.map(i => <Piece key={i.iid} inst={i} c={c} place={place} lane={lane} setLane={setLane}
        newsFilter={newsFilter} setNewsFilter={setNewsFilter} />)}
    </div></div></>;
}
