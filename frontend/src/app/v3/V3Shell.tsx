// Desktop v3 frame: top bar, alert band, rail, work area (Home board or a league screen), status line.
// Markup and class names follow TheWarRoom-mock-v3-FINAL.html (shell.js, grid.js, pieces.js) one for one;
// the styling is v3.css, copied verbatim. Only the data differs: every row here comes from hosted reads.
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { gatherNeedsYou, type NeedsYouItem } from '../shell/homeNeedsYou';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import { left, nameOf } from '../shell/homeTiles';
import { leagueInitials, type LeagueChoice } from '../shell/leaguePicker';
import type { ThinHomeEntry } from '../shell/ThinHome';
import { Icon, type IconName } from './icons';
import './v3.css';
import './legacyHost.css';

// League identity colours, in the mock's order (IDC).
const IDC = ['hsl(172,30%,52%)', 'hsl(210,28%,66%)', 'hsl(14,48%,58%)', 'hsl(75,26%,54%)',
  'hsl(30,62%,56%)', 'hsl(195,32%,60%)', 'hsl(40,40%,66%)', 'hsl(36,22%,72%)'];
const COLS = 24, ROWS = 14, GAP = 4;

export type V3League = LeagueChoice & { bd: string; c: string; commish: boolean; team: boolean };
type Brief = { brief?: HomeBrief; error?: string };
type Lane = 'need' | 'wait' | 'land';
type DocketItem = { id: string; lg: string; lane: Lane; t: string; who: string; clock: string; btn: string;
  entry?: ThinHomeEntry; urgent?: boolean; bad?: boolean };
type PieceId = 'docket' | 'week' | 'today' | 'nflnews' | 'deadlines' | 'shortcuts';
type Inst = { iid: string; pid: PieceId; x: number; y: number; w: number; h: number };
type Board = { id: string; name: string; icon: IconName; insts: Inst[] };
type PieceDef = { title: string; d: string; pal: string; live?: boolean };

const PIECES: Record<PieceId, PieceDef> = {
  docket: { title: 'Docket', d: 'What is waiting, on whom, in every league.', pal: 'pal-slate' },
  week: { title: 'This week across my leagues', d: 'Every matchup you are in this week, on one plate.', pal: 'pal-slate', live: true },
  today: { title: 'My players today', d: 'Your starters across every league, grouped by kickoff.', pal: 'pal-slate', live: true },
  nflnews: { title: 'NFL news', d: 'Injuries, returns and moves for the players that matter to you.', pal: 'pal-slate' },
  deadlines: { title: 'Deadlines', d: 'The next clocks that matter, soonest first.', pal: 'pal-slate' },
  shortcuts: { title: 'Shortcuts', d: 'Your pinned places, one press away.', pal: 'pal-slate' },
};
const mk = (pid: PieceId, x: number, y: number, w: number, h: number): Inst => ({ iid: pid, pid, x, y, w, h });
const TODAY: Board = { id: 'today', name: 'Today', icon: 'home', insts: [mk('docket', 0, 0, 7, 9),
  mk('week', 7, 0, 10, 6), mk('today', 17, 0, 7, 9), mk('nflnews', 7, 6, 10, 8), mk('deadlines', 0, 9, 7, 5),
  mk('shortcuts', 17, 9, 7, 5)] };
const sizeOf = (w: number, h: number) => w >= 20 && h >= 12 ? 'XL'
  : (w >= 10 && h >= 6) || (w >= 7 && h >= 8) ? 'L' : w < 5 || h < 3 ? 'S' : 'M';
const isCommish = (role?: string) => role === 'Commissioner' || role === 'Co-commissioner';
const roleWords = (L: V3League) => L.commish && L.team ? 'Commissioner + owner' : L.commish ? 'Commissioner'
  : 'Owner';

function Badge({ L, big }: { L?: V3League | null; big?: boolean }) {
  if (!L) return <span className={`bdg all${big ? ' big' : ''}`} title="All leagues">≡</span>;
  return <span className={`bdg${big ? ' big' : ''}`} style={{ '--c': L.c } as CSSProperties} title={L.name}>{L.bd}</span>;
}
function Sh({ children }: { children: ReactNode }) { return <div className="sh">{children}</div>; }
function Ladder({ v, n, cls }: { v: number; n: number; cls?: string }) {
  return <span className="lad">{Array.from({ length: n }, (_, i) =>
    <i key={i} className={i < Math.round(v * n) ? `on ${cls ?? ''}` : ''} />)}</span>;
}
function St({ children }: { children: ReactNode }) {
  return <div className="st"><div className="eng">{children}</div></div>;
}

// ---- data: the same hosted reads the thin Home made, one brief per admitted league ----
function useHome(leagues: V3League[], port: HostedPort) {
  const [briefs, setBriefs] = useState<Record<string, Brief>>({});
  const [needs, setNeeds] = useState<NeedsYouItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [stamp, setStamp] = useState(Date.now());
  useEffect(() => { const t = window.setInterval(() => setStamp(Date.now()), 60_000); return () => clearInterval(t); }, []);
  useEffect(() => {
    let active = true;
    setLoading(true);
    void gatherNeedsYou(leagues, port, {
      now: Date.now(),
      readPeople: async id => {
        const L = leagues.find(l => l.id === id);
        if (!L) throw new Error('Admitted league is unavailable');
        return { league: id, yourFranchise: L.franchise ?? '', yourRole: L.role ?? '' };
      },
      readBrief: async (id, mine, hosted, _owner, commish) => {
        try {
          const brief = await readThinHomeBrief(id, mine, hosted, undefined, commish);
          if (active) setBriefs(cur => ({ ...cur, [id]: { brief } }));
          return brief;
        } catch (cause) {
          if (active) setBriefs(cur => ({ ...cur, [id]: { error: cause instanceof Error ? cause.message : String(cause) } }));
          throw cause;
        }
      },
    }).then(items => { if (active) setNeeds(items); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [leagues, port]);
  return { briefs, needs, loading, now: stamp };
}

function entryFor(item: NeedsYouItem): ThinHomeEntry | undefined {
  const s = item.screen;
  if (s.node === 'home') return undefined;
  if (s.node === 'hq') return s.workspace === 'lineup-and-roster' ? 'lineup' : 'hq';
  if (s.node === 'war') return s.workspace === 'trade-desk' ? 'trades'
    : s.workspace === 'valuations-and-pool' ? 'bids' : 'war';
  if (s.node === 'pulse') return 'pulse';
  return 'control';
}
const clockOf = (iso: string | undefined, now: number) => {
  if (!iso) return 'no clock';
  const at = new Date(iso);
  const time = at.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' });
  return Date.parse(iso) - now < 86_400_000 ? `${time} · ${left(iso, now)}` : left(iso, now);
};
function docketOf(needs: NeedsYouItem[], leagues: V3League[], now: number): DocketItem[] {
  return needs.filter(n => n.kind !== 'unavailable').map((n, i) => {
    const L = leagues.find(l => l.id === n.leagueId);
    const soon = n.deadline ? Date.parse(n.deadline) - now < 3_600_000 : false;
    return { id: n.id, lg: n.leagueId, lane: 'need', t: n.title && n.detail ? `${n.title}: ${n.detail}` : n.words,
      who: L?.commish && entryFor(n) === 'control' ? 'you (commissioner)' : 'you',
    clock: clockOf(n.deadline, now), btn: entryFor(n) === 'lineup' ? 'Open lineup' : entryFor(n) === 'trades' ? 'Open offer'
      : entryFor(n) === 'control' ? 'Open job' : 'Open', entry: entryFor(n), urgent: soon || i === 0 && Boolean(n.deadline) };
  });
}

// ---- pieces: R[id](c) in the mock -> { s, fb, b, f } ----
type Ctx = { leagues: V3League[]; briefs: Record<string, Brief>; docket: DocketItem[]; now: number;
  size: string; w: number; open: (L: V3League, e: ThinHomeEntry) => void; loading: boolean };
type Body = { fb?: ReactNode; b: ReactNode; f?: string };

function DkRow({ d, L, sm, open }: { d: DocketItem; L?: V3League; sm?: boolean; open: Ctx['open'] }) {
  return <div className={`dk${sm ? ' sm' : ''}${d.urgent ? ' u' : ''}${d.bad ? ' bad' : ''} unread`}>
    <Badge L={L} /><span className="t ell">{d.t}</span>
    {!sm && <span className="s ell">{d.lane === 'need' ? 'held by ' : ''}{d.who}</span>}
    <span className="ck">{d.clock}</span>
    {!sm && <button className="btn sm" disabled={!L || !d.entry}
      onClick={() => L && d.entry && open(L, d.entry)}>{d.btn}</button>}</div>;
}
function Docket(c: Ctx, lane: number, setLane: (n: number) => void): Body {
  const LN: [Lane, string][] = [['need', 'Needs you'], ['wait', 'Waiting on'], ['land', 'Landed']];
  const by = (k: Lane) => c.docket.filter(d => d.lane === k);
  const none = (k: Lane) => <St>{c.loading ? 'Reading your leagues…' : `Nothing ${k === 'need' ? 'needs you'
    : k === 'wait' ? 'is waiting on anyone' : 'has landed'} right now.`}</St>;
  const L = (d: DocketItem) => c.leagues.find(l => l.id === d.lg);
  const sel = LN[lane][0];
  const rest = c.size === 'L' ? LN.filter((x, i) => i !== lane && by(x[0]).length).map(([k, t]) =>
    <div key={k}><Sh>{t} · {by(k).length}</Sh>{by(k).map(d => <DkRow key={d.id} d={d} L={L(d)} sm open={c.open} />)}</div>) : null;
  return {
    fb: <div className="fbar">{LN.map(([k, t], i) => <button key={k} className="chip" aria-pressed={lane === i}
      onClick={() => setLane(i)}>{t} {by(k).length}</button>)}</div>,
    b: <>{by(sel).length ? by(sel).map(d => <DkRow key={d.id} d={d} L={L(d)} sm={c.size === 'M' && c.w < 7}
      open={c.open} />) : none(sel)}{rest}</>,
    f: 'nothing leaves the app · since you were last here',
  };
}
function Week(c: Ctx): Body {
  const big = c.size === 'L' || c.size === 'XL';
  const rows = c.leagues.map(L => {
    const b = c.briefs[L.id]?.brief, res = b?.pulse?.results, games = res?.games ?? [];
    const mine = L.franchise ? games.find(g => g.home === L.franchise || g.away === L.franchise) : undefined;
    const g = mine ?? games[0];
    if (!g) return <div key={L.id} className={`row${big ? ' two' : ''}`} style={{ gridTemplateColumns: 'auto minmax(0,1fr)' }}>
      <Badge L={L} /><span className="ell d2">{c.briefs[L.id]?.error ? `Unavailable · ${c.briefs[L.id]?.error}`
        : b ? (b.pulse?.weekNote || 'No matchup this week') : 'Reading…'}</span></div>;
    const flip = mine && g.away === L.franchise;
    const [hs, as] = [g.homePoints ?? g.homeProvisional ?? 0, g.awayPoints ?? g.awayProvisional ?? 0];
    const [ms, os, mn, on] = flip ? [as, hs, g.awayName, g.homeName] : [hs, as, g.homeName, g.awayName];
    const st = b?.pulse?.standings?.rows.find(r => r.franchise === L.franchise);
    const wp = b?.pulse?.odds ? undefined : undefined;
    const sub = mine ? `${L.name}${st ? ` · ${st.record} · seed ${st.rank}` : ''}${g.words ? ` · ${g.words}` : ''}`
      : `${L.name} · you run this league and hold no team · the closest game is shown`;
    return <button key={L.id} className={`row${big ? ' two' : ''}`} onClick={() => c.open(L, 'pulse')}
      style={{ gridTemplateColumns: 'auto minmax(0,1.2fr) 56px 14px 56px minmax(0,1fr) 70px 38px' }}>
      <Badge L={L} /><span className="ell">{mine ? <b>{mn}</b> : <span className="d2">{mn}</span>}</span>
      <span className={`v${ms >= os ? '' : ' d2'}`}>{ms.toFixed(1)}</span><span className="m">–</span>
      <span className={`v${os > ms ? '' : ' d2'}`} style={{ textAlign: 'left' }}>{os.toFixed(1)}</span>
      <span className="ell d2">{on}</span>
      {mine ? (wp === undefined ? <span className="m" style={{ gridColumn: 'span 2', textAlign: 'right' }}>{g.status.toUpperCase()}</span>
        : <><Ladder v={wp / 100} n={8} cls={wp >= 65 ? 'g' : wp <= 35 ? 'a' : ''} /><span className="m">{wp}%</span></>)
        : <span className="m" style={{ gridColumn: 'span 2', textAlign: 'right' }}>NO TEAM</span>}
      {big && <span className="sub ell">{sub}</span>}</button>;
  });
  const asOf = Object.values(c.briefs).map(b => b.brief?.pulse?.results?.evidence?.asOf).find(Boolean);
  return { b: <>{rows}</>, f: `${asOf ? `as of ${new Date(asOf).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' })}` : 'current week'} · sorted by league` };
}
function Today(c: Ctx): Body {
  type P = { id: string; name: string; tm: string; locked: boolean; pts?: number; proj?: number; lg: V3League[] };
  const by = new Map<string, P>();
  for (const L of c.leagues) {
    const b = c.briefs[L.id]?.brief; if (!b || !L.franchise) continue;
    const lineup = b.matchup?.lineups[L.franchise] ?? [];
    const scores = b.pulse?.results?.scores.find(s => s.franchise === L.franchise)?.starters ?? [];
    for (const p of lineup) {
      const cur = by.get(p.player) ?? { id: p.player, name: nameOf(b, p.player), tm: p.team, locked: p.locked, lg: [] };
      const sc = scores.find(s => s.player === p.player);
      cur.pts = sc?.points ?? cur.pts; cur.proj = b.matchup?.projections[p.player] ?? cur.proj; cur.lg.push(L);
      by.set(p.player, cur);
    }
  }
  const ps = [...by.values()], multi = c.leagues.filter(l => l.team).length > 1;
  const grp: [string, P[]][] = [['On the field now', ps.filter(p => p.locked)], ['Later today', ps.filter(p => !p.locked)]];
  if (!ps.length) return { b: <St>{c.loading ? 'Reading your lineups…' : 'No starters set in your leagues this week.'}</St>, f: 'starters from your saved lineups' };
  return { b: <>{grp.map(([k, v]) => <div key={k}><Sh>{k} · {v.length}</Sh>{v.map(p =>
    <div key={p.id} className="row" style={{ gridTemplateColumns: '30px minmax(0,1fr) auto 44px 44px' }}>
      <span className="pos" /><span className="ell">{p.name} <span className="m">{p.tm}</span></span>
      <span>{multi ? p.lg.map(l => <Badge key={l.id} L={l} />) : null}</span>
      <span className={`m${p.locked ? ' b' : ''}`}>{p.locked ? 'LIVE' : ''}</span>
      <span className="v">{(p.locked ? p.pts ?? 0 : p.proj ?? 0).toFixed(1)}</span></div>)}</div>)}</>,
  f: multi ? 'badges = leagues where you start him' : 'live · from the league server' };
}
function Deadlines(c: Ctx): Body {
  const d = c.leagues.flatMap(L => (c.briefs[L.id]?.brief?.calendar ?? []).flatMap(e => {
    const at = e.closes || e.at; return at && Date.parse(at) > c.now ? [{ L, e, at }] : [];
  })).sort((a, b) => Date.parse(a.at) - Date.parse(b.at)).slice(0, 8);
  if (!d.length) return { b: <St>{c.loading ? 'Reading your calendars…' : 'No dated deadlines in your leagues.'}</St>, f: 'soonest first' };
  return { b: <>{d.map(({ L, e, at }) => { const soon = Date.parse(at) - c.now < 3_600_000;
    return <div key={L.id + e.id} className="row dlr" style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto' }}>
      <Badge L={L} /><button className="ell dlgo" onClick={() => c.open(L, 'war')}>{e.label}</button>
      <span className={`v${soon ? ' a' : ''}`}>{left(at, c.now)}</span>
      <button className="rem" aria-pressed="false" aria-label="Remind me 15 minutes before">○</button></div>; })}</>,
  f: 'soonest first · ○ remind me' };
}
function Shortcuts(c: Ctx): Body {
  const L = c.leagues.find(l => l.team) ?? c.leagues[0], C = c.leagues.find(l => l.commish);
  const sc: [V3League | undefined, ThinHomeEntry, string][] = [[L, 'lineup', 'Lineup'], [L, 'trades', 'Trade desk'],
    [L, 'pulse', 'Live scoring'], [C, 'control', 'Commissioner jobs'], [L, 'pulse', 'Standings']];
  return { b: <div className="scg">{sc.filter(s => s[0] && (s[1] !== 'lineup' && s[1] !== 'trades' || s[0].team)).map(([l, e, t]) =>
    <button key={t} className="scb" onClick={() => l && c.open(l, e)}><Badge L={l} /><span className="ell">{t}</span></button>)}</div>,
  f: 'pin any tab with the star in the rail' };
}
function News(c: Ctx): Body {
  return { fb: <div className="fbar">{['All', 'My players', 'Injuries', 'On the wire'].map((t, i) =>
    <button key={t} className="chip" aria-pressed={i === 0}>{t} 0</button>)}</div>,
  b: <St>{c.loading ? 'Reading the news…' : 'Player news is not on this league server yet.'}</St>, f: 'every player in the news' };
}

function Piece({ inst, c, place, docketLane, setDocketLane }: { inst: Inst; c: Omit<Ctx, 'size' | 'w'>;
  place: (i: Inst) => CSSProperties; docketLane: number; setDocketLane: (n: number) => void }) {
  const p = PIECES[inst.pid], size = sizeOf(inst.w, inst.h), ctx: Ctx = { ...c, size, w: inst.w };
  const r = inst.pid === 'docket' ? Docket(ctx, docketLane, setDocketLane) : inst.pid === 'week' ? Week(ctx)
    : inst.pid === 'today' ? Today(ctx) : inst.pid === 'deadlines' ? Deadlines(ctx)
      : inst.pid === 'shortcuts' ? Shortcuts(ctx) : News(ctx);
  const live = p.live && c.leagues.some(L => c.briefs[L.id]?.brief?.pulse?.results?.evidence?.live === 'live');
  return <div className={`pc ${p.pal} sz-${size}`} data-iid={inst.iid} tabIndex={0} aria-label={p.title} style={place(inst)}>
    <div className="ph" data-tip={p.d}><span className="grip">⋮⋮</span><Badge /><b className="pt">{p.title}</b>
      <span className="sp" />{live && <span className="lv"><i className="lamp live" /><span>LIVE</span></span>}
      <button className="pm" aria-label="Piece menu"><Icon n="dots" /></button></div>
    <div className="pb">{r.fb}{r.b}</div>{r.f && <div className="pf"><span className="ell">{r.f}</span></div>}</div>;
}

function HomeBoard({ c }: { c: Omit<Ctx, 'size' | 'w'> }) {
  const grid = useRef<HTMLDivElement>(null);
  const [cell, setCell] = useState<{ cw: number; ch: number }>();
  const [lane, setLane] = useState(0);
  useLayoutEffect(() => {
    const g = grid.current; if (!g) return;
    const fit = () => setCell({ cw: g.clientWidth / COLS, ch: g.clientHeight / ROWS });
    fit(); const ro = new ResizeObserver(fit); ro.observe(g); return () => ro.disconnect();
  }, []);
  const place = (r: Inst): CSSProperties => !cell ? { visibility: 'hidden' } : {
    transform: `translate(${r.x * cell.cw + GAP}px,${r.y * cell.ch + GAP}px)`,
    width: r.w * cell.cw - GAP * 2, height: r.h * cell.ch - GAP * 2 };
  return <>
    <div className="ws-tabs"><span className="ttl"><Icon n="home" />HOME</span>
      <button className="tb btab" aria-selected="true"><Icon n={TODAY.icon} /><span>{TODAY.name}</span>
        <span className="ed" role="button" tabIndex={0} aria-label="Board menu">▾</span></button>
      <button className="tb" aria-label="Add a board"><Icon n="plus" /><span className="lb">Board</span></button>
      <div className="tools"><button className="key"><Icon n="plus" /><span className="lb">Add a piece</span></button>
        <button className="key"><Icon n="pencil" /><span className="lb">Edit</span></button></div></div>
    <div className="wbody"><div className="grid" id="grid" ref={grid}
      style={cell ? { '--cw': `${cell.cw}px`, '--ch': `${cell.ch}px` } as CSSProperties : undefined}>
      {TODAY.insts.map(i => <Piece key={i.iid} inst={i} c={c} place={place} docketLane={lane} setDocketLane={setLane} />)}
    </div></div></>;
}

const SCREENS: { k: ThinHomeEntry; name: string; icon: IconName; on: (L: V3League) => boolean; now: string }[] = [
  { k: 'hq', name: 'Franchise HQ', icon: 'shield', on: L => L.team, now: 'Lineup and roster' },
  { k: 'war', name: 'War Room', icon: 'swords', on: L => L.team, now: 'Trade desk and pool' },
  { k: 'pulse', name: 'League Pulse', icon: 'pulse', on: () => true, now: 'Scores and the race' },
  { k: 'control', name: 'Control Room', icon: 'gear', on: L => L.commish, now: 'Jobs waiting' },
];

export type V3ShellProps = { leagues: LeagueChoice[]; active?: { league: LeagueChoice; entry: ThinHomeEntry };
  onOpen: (league: LeagueChoice, entry: ThinHomeEntry) => void; onHome: () => void; onSignOut: () => void;
  accountName?: string; port?: HostedPort; children?: ReactNode };

export default function V3Shell({ leagues: admitted, active, onOpen, onHome, onSignOut, accountName,
  port = hostedWorkflow, children }: V3ShellProps) {
  const leagues = useMemo<V3League[]>(() => admitted.filter(l => l.app).map((l, i) => ({ ...l,
    bd: leagueInitials(l.name), c: IDC[i % IDC.length], commish: isCommish(l.role), team: Boolean(l.franchise) })), [admitted]);
  const { briefs, needs, loading, now } = useHome(leagues, port);
  const docket = useMemo(() => docketOf(needs, leagues, now), [needs, leagues, now]);
  const [selected, setSelected] = useState<string>();
  const [layer, setLayer] = useState<'switch' | 'cmdk' | 'account' | null>(null);
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && layer) { setLayer(null); return; }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); setLayer('cmdk'); return; }
      if (e.altKey && e.key.toLowerCase() === 'l') { e.preventDefault(); setLayer('switch'); return; }
      if (e.altKey && e.key.toLowerCase() === 'h') { e.preventDefault(); setLayer(null); setSelected(undefined); onHome(); }
    };
    window.addEventListener('keydown', key); return () => window.removeEventListener('keydown', key);
  }, [layer, onHome]);
  const L = leagues.find(l => l.id === (active?.league.id ?? selected)) ?? null;
  const open = (l: V3League, e: ThinHomeEntry) => onOpen(admitted.find(a => a.id === l.id) ?? l, e);
  const screen = active?.entry === 'control' ? 'pal-slate' : active ? ({ hq: 'pal-hq', lineup: 'pal-hq', war: 'pal-war',
    bids: 'pal-war', trades: 'pal-war', pulse: 'pal-pulse', league: 'pal-pulse', home: 'pal-slate', control: 'pal-slate' } as const)[active.entry] : 'pal-slate';
  useEffect(() => {
    const was = document.body.className; document.body.className = screen;
    document.documentElement.style.setProperty('--rail-w', '208px');
    document.documentElement.style.setProperty('--lg', L ? L.c : 'var(--edge)');
    return () => { document.body.className = was; };
  }, [screen, L]);
  const n = docket.filter(d => d.lane === 'need').length;
  const top = docket.filter(d => d.lane === 'need').sort((a, b) => Number(b.urgent ?? 0) - Number(a.urgent ?? 0))[0];
  const topL = top && leagues.find(l => l.id === top.lg);
  const week = Object.values(briefs).map(b => b.brief?.pulse?.week).find(w => w !== undefined);
  const liveOn = Object.values(briefs).some(b => b.brief?.pulse?.results?.evidence?.live === 'live');
  const initials = (accountName ?? 'You').slice(0, 2).toUpperCase();
  const c = { leagues, briefs, docket, now, open, loading };
  const stamp = new Date(now).toLocaleString('en-US', { weekday: 'short', hour: 'numeric', minute: '2-digit',
    timeZone: 'America/Chicago' }).toUpperCase().replace(',', '');

  return <div id="app">
    <header className="top">
      <button className="mark" onClick={() => { setSelected(undefined); onHome(); }} data-tip="Home: everything you play is here|Alt+H">
        <span className="gl">TWR</span><span><b>THEWARROOM</b><small>COMMAND CENTER</small></span></button>
      <button className="lchip" onClick={() => setLayer('switch')} style={{ '--lg': L ? L.c : 'var(--edge)' } as CSSProperties} data-tip="The league you are working in. Click to switch.|Alt+L">
        <span className="edge" /><Badge L={L} big /><span className="ell"><b className="ell">{L ? L.name : 'All leagues'}</b>
          <small>{L ? roleWords(L).toUpperCase() : `${leagues.length} ${leagues.length === 1 ? 'LEAGUE' : 'LEAGUES'} · PICK ONE`}</small></span>
        <span className="car">▾</span></button>
      <button className="search" onClick={() => setLayer('cmdk')} data-tip="Find a player, team, screen, piece or command|Ctrl+K"><Icon n="search" />
        <span>Search or type a command</span><kbd>Ctrl K</kbd></button>
      <div className="tkeys"><button className="key" data-tip="What is waiting on you, in every league|Alt+D"><Icon n="bell" />
        <span className="lb">Needs you</span>{n ? <span className="ct">{n}</span> : null}</button>
        <button className="key" data-tip="Ask in plain words: arrange screens, prepare a move|Alt+A"><Icon n="chat" /><span className="lb">Assistant</span></button>
        <button className="key" data-tip="Show where every number comes from and goes|Alt+F"><Icon n="plug" /><span className="lb">Feeds</span></button>
        <button className="acct" onClick={() => setLayer(l => l === 'account' ? null : 'account')} aria-expanded={layer === 'account'} data-tip="Account, devices, labels, your AI key, sign out">{initials}</button></div>
    </header>
    {top && topL ? <div className={`band${top.urgent ? ' crit' : ''}`} role="button" tabIndex={0}
      onClick={() => top.entry && open(topL, top.entry)}>
      <span className="ak">{top.urgent ? 'NOW' : 'NEXT'}</span><Badge L={topL} /><span className="at ell">{top.t}</span>
      <span className="clock">{top.clock}</span><span className="go">{top.btn.toUpperCase()} ▸</span>
      <span className="more"><b>{n}</b>NEED YOU · OPEN DOCKET</span></div> : <div className="band none" />}
    <nav className="rail">
      <div className="appz"><div className="gl"><span>APP</span></div>
        <button className={`nb${active ? '' : ' on'}`} onClick={() => { setSelected(undefined); onHome(); }}><Icon n="home" /><span className="lb">Home</span></button>
        <button className={`leaf${active ? '' : ' sel'}`}><span className="tr">{active ? '├' : '▮'}</span><span className="ell">Today</span></button>
        <div className="gl" style={{ paddingTop: 8 }}><Icon n="star" /><span>MY SHORTCUTS</span></div>
        <div className="leaf gone" data-tip="Open any league tab and press the star to pin it here">Pin any tab with ☆</div></div>
      {!L ? <div className="lz none"><button className="cart" onClick={() => setLayer('switch')} style={{ boxShadow: 'inset 0 -3px 0 var(--edge)' }}>
        <Badge big /><b>All leagues</b><span className="rl">▾</span><small>No league selected</small></button>
        <div className="lzb"><div className="gl"><span>PICK A LEAGUE</span></div>
          {leagues.map(l => { const w = docket.filter(d => d.lg === l.id && d.lane === 'need').length;
            return <button key={l.id} className="leaf" style={{ paddingLeft: 14 }} onClick={() => setSelected(l.id)}>
              <Badge L={l} /><span className="ell">{l.name}</span>{w ? <span className="sp tag am">{w}</span> : null}</button>; })}</div></div>
        : <div className="lz" style={{ '--lg': L.c } as CSSProperties}>
          <button className="cart" onClick={() => { setSelected(undefined); if (active) onHome(); }} data-tip="Selected league. Click to switch.">
            <Badge L={L} big /><b className="ell">{L.name}</b><span className="rl">▾</span>
            <small className="ell">{L.team ? `${L.franchiseName ?? ''}${(() => { const r = briefs[L.id]?.brief?.pulse?.standings?.rows.find(x => x.franchise === L.franchise); return r ? ` · ${r.record} · seed ${r.rank}` : ''; })()}` : `No team · ${roleWords(L)}`}</small></button>
          <div className="lzb">{SCREENS.filter(s => s.on(L)).map(s => { const on = active && (active.entry === s.k
            || (s.k === 'hq' && active.entry === 'lineup') || (s.k === 'war' && (active.entry === 'bids' || active.entry === 'trades')));
          return <div key={s.k}><button className={`nb${on ? ' on' : ''}`} onClick={() => open(L, s.k)}><Icon n={s.icon} />
            <span className="lb">{s.name}</span><span className="car">{on ? '▾' : '▸'}</span></button>
            {!on && <button className="leaf now" onClick={() => open(L, s.k)}><span className="tag">NOW</span><span className="ell">{s.now}</span></button>}</div>; })}
          {SCREENS.some(s => !s.on(L)) && <><div className="gl" style={{ paddingTop: 12 }}><span>SHAPED FOR THIS LEAGUE</span></div>
            {SCREENS.filter(s => !s.on(L)).map(s => <div key={s.k} className="leaf gone" style={{ paddingLeft: 14, whiteSpace: 'normal', lineHeight: 1.3 }}>
              No {s.name}: {s.k === 'control' ? 'only the commissioner has a Control Room' : 'you have no team in this league'}</div>)}</>}</div></div>}
      <div className="rail-foot"><button data-tip="Collapse the rail to icons">«</button><span>{L ? 'hosted league' : 'your app, above your leagues'}</span></div>
    </nav>
    <main className="work">{active ? <div className="legacy-host" style={{ flex: 1, minHeight: 0, overflow: 'auto' }}>{children}</div> : <HomeBoard c={c} />}</main>
    <aside className="insp closed" />
    <footer className="status">
      <button className="stb" data-tip="League servers"><i className="lamp g" />CONNECTED · {L ? `league server ${L.bd.toLowerCase()}` : `${leagues.length} league ${leagues.length === 1 ? 'server' : 'servers'}`}</button>
      <button className="stb" data-tip="Spoiler delay"><i className={`lamp${liveOn ? ' live' : ''}`} />{liveOn ? 'LIVE · NO DELAY' : 'CURRENT'}</button>
      <button className="stb" data-tip="Season calendar">{stamp} CT{week !== undefined ? ` · WEEK ${week}` : ''}</button>
      <span className="idp">TWR-MK3 · {accountName ? accountName.toUpperCase() : 'SIGNED IN'} · <button onClick={onSignOut} style={{ textDecoration: 'underline' }}>SIGN OUT</button></span>
    </footer>
    <div id="layer">{layer && <div className="ov" style={{ position: 'absolute', inset: 0 }}>
      {layer === 'cmdk' ? <div className="scrim" onClick={() => setLayer(null)} />
        : <div style={{ position: 'absolute', inset: 0 }} onClick={() => setLayer(null)} />}
      {layer === 'switch' && <Switcher leagues={leagues} docket={docket} briefs={briefs} current={L?.id}
        pick={id => { setLayer(null); setSelected(id); if (!id) onHome(); }} />}
      {layer === 'cmdk' && <Commands leagues={leagues} close={() => setLayer(null)} open={open}
        home={() => { setSelected(undefined); onHome(); }} pick={id => setSelected(id)} />}
      {layer === 'account' && <div className="pop menu" style={{ right: 8, top: 52, width: 300 }}>
        <div className="mh">YOU · {leagues.length} {leagues.length === 1 ? 'LEAGUE' : 'LEAGUES'}</div>
        <button disabled><Icon n="tv" /> Devices <span className="w">this desktop</span></button>
        <button disabled><Icon n="chat" /> Assistant: connect your AI provider</button><hr />
        <button onClick={() => { setLayer(null); onSignOut(); }}><Icon n="x" /> Sign out</button></div>}
    </div>}</div>
  </div>;
}

function Switcher({ leagues, docket, briefs, current, pick }: { leagues: V3League[]; docket: DocketItem[];
  briefs: Record<string, Brief>; current?: string; pick: (id?: string) => void }) {
  const [q, setQ] = useState('');
  const waiting = (l: V3League) => docket.filter(d => d.lg === l.id && d.lane === 'need').length;
  const grp: [string, V3League[]][] = [['NEEDS YOU', leagues.filter(l => waiting(l))],
    ['EVERYTHING ELSE', leagues.filter(l => !waiting(l))]];
  const rec = (l: V3League) => briefs[l.id]?.brief?.pulse?.standings?.rows.find(r => r.franchise === l.franchise)?.record;
  return <div className="pop sw"><input placeholder="Find a league…" autoFocus value={q} onChange={e => setQ(e.target.value)} />
    <button className={`lrow${current ? '' : ' sel'}`} onClick={() => pick(undefined)}><Badge />
      <span><b>All leagues</b> <span className="d">back to Home</span></span><span /><span /><span /></button>
    <div className="list">{grp.map(([g, ls]) => { ls = ls.filter(l => !q || l.name.toLowerCase().includes(q.toLowerCase()));
      return ls.length ? <div key={g}><div className="sh">{g} · {ls.length}</div>{ls.map(l =>
        <button key={l.id} className={`lrow${l.id === current ? ' sel' : ''}`} style={{ '--c': l.c } as CSSProperties} onClick={() => pick(l.id)}>
          <Badge L={l} /><span className="ell"><b>{l.name}</b> <span className="d">{l.franchiseName ?? ''}</span></span>
          <span className="role">{l.commish ? <Icon n="crown" /> : null}</span><span className="rc">{rec(l) ?? '—'}</span>
          <span className={`w8${waiting(l) ? '' : ' z'}`}>{waiting(l) || '·'}</span></button>)}</div> : null; })}</div>
    <div className="sh" style={{ borderTop: '1px solid var(--edge)' }}><Icon n="crown" /> commissioner · amber = matters waiting on you</div></div>;
}

function Commands({ leagues, close, open, home, pick }: { leagues: V3League[]; close: () => void;
  open: (L: V3League, e: ThinHomeEntry) => void; home: () => void; pick: (id: string) => void }) {
  const [q, setQ] = useState(''), [sel, setSel] = useState(0);
  const items = useMemo(() => {
    const out: [string, string, () => void][] = [['Home', 'screen', home]];
    for (const L of leagues) {
      for (const s of SCREENS.filter(x => x.on(L))) out.push([`${s.name} › ${L.name}`, 'screen', () => open(L, s.k)]);
      out.push([`League › ${L.name}`, 'league', () => pick(L.id)]);
    }
    return out;
  }, [leagues, home, open, pick]);
  const shown = items.filter(i => !q || i[0].toLowerCase().includes(q.toLowerCase())).slice(0, 12);
  const run = (i: number) => { const f = shown[i]; close(); f?.[2](); };
  return <div className="pop cmdk"><input placeholder="Search players, teams, screens, pieces and commands" autoFocus value={q}
    onChange={e => { setQ(e.target.value); setSel(0); }} onKeyDown={e => {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); setSel(v => Math.max(0, Math.min(shown.length - 1, v + (e.key === 'ArrowDown' ? 1 : -1)))); }
      if (e.key === 'Enter') run(sel);
    }} />
    <div className="res">{shown.length ? shown.map((it, k) => <button key={it[0]} className={k === sel ? 'on' : ''} onClick={() => run(k)}>
      {it[0]}<small>{it[1].toUpperCase()}</small></button>) : <div className="st">Nothing matches. Try a league or a screen.</div>}</div></div>;
}
