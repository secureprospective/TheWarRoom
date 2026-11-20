// Desktop v3 frame: top bar, alert band, rail, work area (Home board or a league screen), status line.
// Markup and class names follow TheWarRoom-mock-v3-FINAL.html (shell.js, grid.js, pieces.js) one for one;
// the styling is v3.css, copied verbatim. Only the data differs: every row here comes from hosted reads.
import { lazy, Suspense, useEffect, useMemo, useState, type CSSProperties, type ReactNode } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { LeagueChoice } from '../shell/leaguePicker';
import type { ThinHomeEntry } from '../shell/ThinHome';
import type { PulseTab } from '../shell/pulseReads';
import type { HQTab } from '../shell/HQShell';
import { HQAct } from '../shell/HQAct';
import { Icon } from './icons';
import { ctStamp, docketOf, dress, lanesOf, roleWords, SCREENS, useHome, type V3League } from './model';
import { AssistantDrawer, Commands, DocketDrawer, Switcher } from './overlays';
import { Badge, HomeBoard } from './pieces';
import './v3.css';
import './legacyHost.css';
import './fixes.css';

const NativeLeaguePulse = lazy(() => import('../shell/NativeLeaguePulse'));
const NativeFranchiseHQ = lazy(() => import('../shell/NativeFranchiseHQ'));
const HQ_TABS: [HQTab, string][] = [['lineup-and-roster', 'Lineup and roster'],
  ['contracts-and-cap', 'Contracts and cap'], ['my-moves', 'My moves']];

export type V3ShellProps = { leagues: LeagueChoice[]; active?: { league: LeagueChoice; entry: ThinHomeEntry };
  onOpen: (league: LeagueChoice, entry: ThinHomeEntry) => void; onHome: () => void; onSignOut: () => void;
  accountName?: string; port?: HostedPort; children?: ReactNode;
  /** An account screen (authenticator, recovery) shown in the work area over Home or the open league. */
  panel?: ReactNode;
  /** Account menu entries (the initials button). */
  account?: ReactNode;
  /** A failed account action's reason, shown in the status line. */
  status?: string };

const PALETTE: Record<ThinHomeEntry, string> = { hq: 'pal-hq', lineup: 'pal-hq', war: 'pal-war',
  bids: 'pal-war', trades: 'pal-war', pulse: 'pal-pulse', league: 'pal-pulse', home: 'pal-slate',
  control: 'pal-slate' };
const TIP = {
  mark: 'Home: everything you play is here|Alt+H',
  chip: 'The league you are working in. Click to switch.|Alt+L',
  search: 'Find a player, team, screen, piece or command|Ctrl+K',
  needs: 'What is waiting on you, in every league|Alt+D',
  assist: 'Ask in plain words: arrange screens, prepare a move|Alt+A',
  acct: 'Account, devices, labels, your AI key, sign out',
};
const HOST: CSSProperties = { flex: 1, minHeight: 0, overflow: 'auto' };
const tip = (t: string) => ({ 'data-tip': t });
const PULSE_TABS: [PulseTab, string][] = [['now', 'Now'], ['race', 'Race']];
const isOn = (k: ThinHomeEntry, e: ThinHomeEntry) => e === k || (k === 'hq' && e === 'lineup')
  || (k === 'war' && (e === 'bids' || e === 'trades'));

export default function V3Shell({ leagues: admitted, active, onOpen, onHome, onSignOut, accountName,
  port = hostedWorkflow, children, panel, account, status }: V3ShellProps) {
  const leagues = useMemo(() => dress(admitted), [admitted]);
  const { briefs, needs, loading, now } = useHome(leagues, port);
  const docket = useMemo(() => [...docketOf(needs, leagues, now), ...lanesOf(leagues, briefs, now)],
    [needs, leagues, briefs, now]);
  const [selected, setSelected] = useState<string>();
  const [pulseTab, setPulseTab] = useState<PulseTab>('now');
  const [hqTab, setHqTab] = useState<HQTab>('lineup-and-roster');
  // The commissioner's franchise choice per league, kept in this browser so a reload reopens it.
  const [hqTeam, setHqTeamState] = useState<Record<string, string>>(() => {
    try { return JSON.parse(localStorage.getItem('twr.hqTeam') ?? '{}') as Record<string, string>; }
    catch { return {}; } });
  const setHqTeam = (f: (t: Record<string, string>) => Record<string, string>) => setHqTeamState(t => {
    const next = f(t); localStorage.setItem('twr.hqTeam', JSON.stringify(next)); return next; });
  const [layer, setLayer] = useState<'switch' | 'cmdk' | 'account' | 'docket' | 'assist' | null>(null);
  const home = () => { setSelected(undefined); onHome(); };
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      const k = e.key.toLowerCase();
      if (e.key === 'Escape' && layer) setLayer(null);
      else if ((e.ctrlKey || e.metaKey) && k === 'k') { e.preventDefault(); setLayer('cmdk'); }
      else if (e.altKey && k === 'a') { e.preventDefault(); setLayer('assist'); }
      else if (e.altKey && k === 'd') { e.preventDefault(); setLayer('docket'); }
      else if (e.altKey && k === 'l') { e.preventDefault(); setLayer('switch'); }
      else if (e.altKey && k === 'h') { e.preventDefault(); setLayer(null); home(); }
    };
    if (typeof window.addEventListener !== 'function') return;
    window.addEventListener('keydown', key);
    return () => window.removeEventListener('keydown', key);
  }, [layer, onHome]);
  const L = leagues.find(l => l.id === (active?.league.id ?? selected)) ?? null;
  const open = (l: V3League, e: ThinHomeEntry) => onOpen(admitted.find(a => a.id === l.id) ?? l, e);
  const screen = active ? PALETTE[active.entry] : 'pal-slate';
  useEffect(() => {
    const was = document.body.className;
    document.body.className = screen;
    document.documentElement.style.setProperty('--rail-w', '208px');
    document.documentElement.style.setProperty('--lg', L ? L.c : 'var(--edge)');
    return () => { document.body.className = was; };
  }, [screen, L]);
  const needing = docket.filter(d => d.lane === 'need');
  const n = needing.length;
  const top = [...needing].sort((a, b) => Number(b.urgent ?? 0) - Number(a.urgent ?? 0))[0];
  const topL = top && leagues.find(l => l.id === top.lg);
  const week = Object.values(briefs).map(b => b.brief?.pulse?.week).find(w => w !== undefined);
  const liveOn = Object.values(briefs).some(b => b.brief?.pulse?.results?.evidence?.live === 'live');
  const initials = (accountName ?? 'You').slice(0, 2).toUpperCase();
  const c = { leagues, briefs, docket, now, open, loading };
  const scrim = layer === 'cmdk' || layer === 'docket';
  const plural = (k: number, one: string, many: string) => `${k} ${k === 1 ? one : many}`;

  return <div id="app">
    <header className="top">
      <HQAct className="mark" onAction={home} attributes={tip(TIP.mark)}>
        <span className="gl">TWR</span><span><b>THEWARROOM</b><small>COMMAND CENTER</small></span></HQAct>
      <HQAct className="lchip" onAction={() => setLayer('switch')} attributes={tip(TIP.chip)}
        style={{ '--lg': L ? L.c : 'var(--edge)' } as CSSProperties}>
        <span className="edge" /><Badge L={L} big />
        <span className="ell"><b className="ell">{L ? L.name : 'All leagues'}</b>
          <small>{L ? roleWords(L).toUpperCase()
            : `${plural(leagues.length, 'LEAGUE', 'LEAGUES')} · PICK ONE`}</small>
        </span><span className="car">▾</span></HQAct>
      <HQAct className="search" onAction={() => setLayer('cmdk')} attributes={tip(TIP.search)}>
        <Icon n="search" />
        <span>Search or type a command</span><kbd>Ctrl K</kbd></HQAct>
      <div className="tkeys">
        <HQAct className="key" attributes={tip(TIP.needs)} expanded={layer === 'docket'}
          onAction={() => setLayer(l => l === 'docket' ? null : 'docket')}><Icon n="bell" />
          <span className="lb">Needs you</span>{n ? <span className="ct">{n}</span> : null}</HQAct>
        <HQAct className="key" attributes={tip(TIP.assist)} expanded={layer === 'assist'}
          onAction={() => setLayer(l => l === 'assist' ? null : 'assist')}>
          <Icon n="chat" /><span className="lb">Assistant</span></HQAct>
        <HQAct className="acct" label="Account" expanded={layer === 'account'} attributes={tip(TIP.acct)}
          onAction={() => setLayer(l => l === 'account' ? null : 'account')}>{initials}</HQAct></div>
    </header>
    {top && topL ? <div className={`band${top.urgent ? ' crit' : ''}`}>
      <HQAct label={`Open ${top.t}`} style={{ display: 'contents' }}
        onAction={() => { if (top.entry) open(topL, top.entry); }}>
      <span className="ak">{top.urgent ? 'NOW' : 'NEXT'}</span><Badge L={topL} />
      <span className="at ell">{top.t}</span>
      <span className="clock">{top.clock}</span><span className="go">{top.btn.toUpperCase()} ▸</span></HQAct>
      <HQAct className="more" label="Open docket" onAction={() => setLayer('docket')}>
        <b>{n}</b>NEED YOU · OPEN DOCKET</HQAct></div> : <div className="band none" />}
    <Rail leagues={leagues} L={L} active={active} c={c} home={home} pick={setSelected}
      pulseTab={pulseTab} setPulseTab={setPulseTab} hqTab={hqTab} setHqTab={setHqTab}
      switcher={() => setLayer('switch')} onHome={onHome} />
    <main className="work">
      {panel && <div className="legacy-host" style={HOST}>{panel}</div>}
      {active && L && (active.entry === 'pulse' || active.entry === 'league') ? !panel
        && <Suspense fallback={<div className="st"><div className="eng">Opening League Pulse…</div></div>}>
          <NativeLeaguePulse key={L.id} league={L.id} mine={L.franchise ?? ''} port={port} serverTime
            commissioner={L.commish} tab={pulseTab} navigate={t => setPulseTab(t)} leave={home}
            v3={{ badge: <Badge L={L} />, kind: roleWords(L).toUpperCase() }} /></Suspense>
      : active && L && (L.franchise || L.commish) && (active.entry === 'hq' || active.entry === 'lineup')
        ? !panel
        && <Suspense fallback={<div className="st"><div className="eng">Opening Franchise HQ…</div></div>}>
          <NativeFranchiseHQ key={`${L.id}:${hqTeam[L.id] || L.franchise || ''}`} league={L.id}
            mine={hqTeam[L.id] || L.franchise || ''} name={L.franchiseName ?? ''} port={port} embedded
            tab={hqTab} navigate={setHqTab} canManage={(hqTeam[L.id] || L.franchise) === L.franchise}
            leave={home} openGame={() => { setPulseTab('now'); open(L, 'pulse'); }}
            v3={{ badge: <Badge L={L} />, kind: roleWords(L).toUpperCase(), pickTeam: L.commish
              ? id => setHqTeam(t => ({ ...t, [L.id]: id })) : undefined }} /></Suspense>
      : active ? <div className="legacy-host" hidden={Boolean(panel)}
        style={HOST}>{children}</div>
        : !panel && <HomeBoard c={c} />}</main>
    <aside className="insp closed" />
    <footer className="status">
      <HQAct className="stb" attributes={tip('League servers')}><i className="lamp g" />CONNECTED · {L
        ? `league server ${L.bd.toLowerCase()}`
        : plural(leagues.length, 'league server', 'league servers')}</HQAct>
      <HQAct className="stb" attributes={tip('Spoiler delay')}>
        <i className={`lamp${liveOn ? ' live' : ''}`} />{liveOn ? 'LIVE · NO DELAY' : 'CURRENT'}</HQAct>
      <HQAct className="stb" attributes={tip('Season calendar')}>
        {ctStamp(now)} CT{week !== undefined ? ` · WEEK ${week}` : ''}</HQAct>
      {status && <span className="r" role="alert">{status}</span>}
      <span className="idp">TWR-MK3 · {accountName ? accountName.toUpperCase() : 'SIGNED IN'} · <HQAct
        onAction={onSignOut} style={{ textDecoration: 'underline' }}>SIGN OUT</HQAct></span>
    </footer>
    <div id="layer">{layer && <div className="ov" style={{ position: 'absolute', inset: 0 }}>
      <HQAct className={scrim ? 'scrim' : undefined} label="Close"
        style={scrim ? undefined : { position: 'absolute', inset: 0, cursor: 'default' }}
        onAction={() => setLayer(null)}><></></HQAct>
      {layer === 'switch' && <Switcher leagues={leagues} docket={docket} briefs={briefs} current={L?.id}
        pick={id => { setLayer(null); setSelected(id); if (!id) onHome(); }} />}
      {layer === 'cmdk' && <Commands leagues={leagues} close={() => setLayer(null)} open={open}
        home={home} pick={id => setSelected(id)} />}
      {layer === 'docket' && <DocketDrawer leagues={leagues} docket={docket} current={L} open={open}
        close={() => setLayer(null)} home={home} />}
      {layer === 'assist' && <AssistantDrawer close={() => setLayer(null)}
        commish={active?.entry === 'control' && L?.commish ? L : null}
        league={L ?? leagues.find(l => l.team)} />}
      {layer === 'account' && <div className="pop menu" style={{ right: 8, top: 52, width: 300 }}>
        <div className="mh">YOU · {plural(leagues.length, 'LEAGUE', 'LEAGUES')}</div>
        {account ?? <HQAct onAction={() => { setLayer(null); onSignOut(); }}>
          <Icon n="x" /> Sign out</HQAct>}</div>}
    </div>}</div>
  </div>;
}

type RailProps = { leagues: V3League[]; L: V3League | null; active?: V3ShellProps['active'];
  c: Parameters<typeof HomeBoard>[0]['c']; home: () => void; pick: (id?: string) => void;
  switcher: () => void; onHome: () => void; pulseTab: PulseTab; setPulseTab: (t: PulseTab) => void;
  hqTab: HQTab; setHqTab: (t: HQTab) => void };

function Rail({ leagues, L, active, c, home, pick, switcher, onHome, pulseTab, setPulseTab, hqTab,
  setHqTab }: RailProps) {
  const waiting = (id: string) => c.docket.filter(d => d.lg === id && d.lane === 'need').length;
  const standing = (l: V3League) =>
    c.briefs[l.id]?.brief?.pulse?.standings?.rows.find(x => x.franchise === l.franchise);
  const teamLine = (l: V3League) => {
    const r = standing(l);
    return `${l.franchiseName ?? ''}${r ? ` · ${r.record} · seed ${r.rank}` : ''}`;
  };
  return <nav className="rail">
    <div className="appz"><div className="gl"><span>APP</span></div>
      <HQAct className={`nb${active ? '' : ' on'}`} onAction={home}>
        <Icon n="home" /><span className="lb">Home</span></HQAct>
      <HQAct className={`leaf${active ? '' : ' sel'}`}>
        <span className="tr">{active ? '├' : '▮'}</span><span className="ell">Today</span></HQAct>
      <div className="gl" style={{ paddingTop: 8 }}><Icon n="star" /><span>MY SHORTCUTS</span></div>
      <div className="leaf gone" data-tip="Open any league tab and press the star to pin it here">
        Pin any tab with ☆</div>
    </div>
    {!L ? <div className="lz none">
      <HQAct className="cart" onAction={switcher} style={{ boxShadow: 'inset 0 -3px 0 var(--edge)' }}>
        <Badge big /><b>All leagues</b><span className="rl">▾</span><small>No league selected</small></HQAct>
      <div className="lzb"><div className="gl"><span>PICK A LEAGUE</span></div>
        {leagues.map(l => <HQAct key={l.id} className="leaf" label={`Pick ${l.name}`}
          style={{ paddingLeft: 14 }} onAction={() => pick(l.id)}>
          <Badge L={l} /><span className="ell">{l.name}</span>
          {waiting(l.id) ? <span className="sp tag am">{waiting(l.id)}</span> : null}</HQAct>)}</div></div>
      : <div className="lz" style={{ '--lg': L.c } as CSSProperties}>
        <HQAct className="cart" attributes={tip('Selected league. Click to switch.')}
          onAction={() => { pick(undefined); if (active) onHome(); }}>
          <Badge L={L} big /><b className="ell">{L.name}</b><span className="rl">▾</span>
          <small className="ell">{L.team ? teamLine(L) : `No team · ${roleWords(L)}`}</small></HQAct>
        <div className="lzb">{SCREENS.filter(s => s.on(L)).map(s => {
          const on = Boolean(active && isOn(s.k, active.entry));
          return <div key={s.k}>
            <HQAct className={`nb${on ? ' on' : ''}`} label={`Open ${s.name}`}
              onAction={() => c.open(L, s.k)}>
              <Icon n={s.icon} /><span className="lb">{s.name}</span>
              <span className="car">{on ? '▾' : '▸'}</span></HQAct>
            {on && s.k === 'pulse' && PULSE_TABS.map(([t, name]) => <HQAct key={t}
              className={`leaf${pulseTab === t ? ' sel' : ''}`} onAction={() => setPulseTab(t)}>
              <span className="tr">{pulseTab === t ? '▮' : '├'}</span>
              <span className="ell">{name}</span></HQAct>)}
            {on && s.k === 'hq' && HQ_TABS.map(([t, name]) => <HQAct key={t}
              className={`leaf${hqTab === t ? ' sel' : ''}`} onAction={() => setHqTab(t)}>
              <span className="tr">{hqTab === t ? '▮' : '├'}</span>
              <span className="ell">{name}</span></HQAct>)}
            {!on && <HQAct className="leaf now" onAction={() => c.open(L, s.k)}>
              <span className="tag">NOW</span><span className="ell">{s.now}</span></HQAct>}</div>;
        })}
        {SCREENS.some(s => !s.on(L)) && <>
          <div className="gl" style={{ paddingTop: 12 }}><span>SHAPED FOR THIS LEAGUE</span></div>
          {SCREENS.filter(s => !s.on(L)).map(s => <div key={s.k} className="leaf gone"
            style={{ paddingLeft: 14, whiteSpace: 'normal', lineHeight: 1.3 }}>
            No {s.name}: {s.off}</div>)}</>}
        </div></div>}
    <div className="rail-foot"><HQAct attributes={tip('Collapse the rail to icons')}>«</HQAct>
      <span>{L ? 'hosted league' : 'your app, above your leagues'}</span></div>
  </nav>;
}
