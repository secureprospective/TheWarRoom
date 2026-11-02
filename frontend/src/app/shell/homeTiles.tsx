import { useState, type ReactNode } from 'react';
import { commands } from '../commands/registry';
import type { HomeBrief, HomeSource } from './homeReads';
import type { TileId } from './homeLayout';
import { money, unavailable } from './hqModel';
import type { Node } from './nodes';
import { pointsLabel, type PulseStarter } from '../data/nativePulse';

/** Each tile is the locked mock's tile: its title, source screen, the read that feeds it and where it
 *  opens. `missing` names the data the app does not hold yet; such a tile says so and shows nothing. */
export type TileDef = { t: string; src: string; prov: string; link: string; refresh: string;
  open?: { node: Node; workspace?: string }; needs: HomeSource[]; missing?: string };
export const TILES: Record<TileId, TileDef> = {
  due: { t: 'DECISIONS DUE', src: 'Clock + alerts', prov: 'calendar.read', link: 'HQ ›',
    refresh: 'on clock tick · 60s', open: { node: 'hq' }, needs: ['calendar'] },
  mu: { t: 'THIS WEEK', src: 'League Pulse · Live',
    prov: 'results.read(week) + standings.read + lineup.read + projections.read',
    link: 'LIVE ›', refresh: 'live push', open: { node: 'pulse', workspace: 'now' },
    needs: ['pulse', 'results'] },
  race: { t: 'RACE', src: 'League Pulse · Race', prov: 'standings.read + week.read + bracket.read',
    link: 'RACE ›', refresh: 'after each final', open: { node: 'pulse', workspace: 'race' },
    needs: ['pulse'] },
  head: { t: 'LEAGUE HEADLINES', src: 'League record', prov: '—', link: 'PULSE ›', refresh: 'on event',
    open: { node: 'pulse' }, needs: [], missing: 'No read writes league headlines from scores, trades and streaks' },
  news: { t: 'NFL NEWS', src: 'Player research · MFL player news', prov: 'research.TargetResearch (articles)',
    link: 'RESEARCH ›', refresh: 'on research refresh', open: { node: 'war', workspace: 'research' }, needs: ['research'] },
  wire: { t: 'TRANSACTIONS', src: 'League ledger', prov: 'commish.ledger.read', link: 'LEDGER ›',
    refresh: 'on event', open: { node: 'control', workspace: 'league' }, needs: ['ledger'] },
  sc: { t: 'MY SHORTCUTS', src: 'Owner', prov: 'owner pins (☆)', link: '', refresh: '—', needs: [] },
  pwr: { t: 'POWER', src: 'League Pulse · Race', prov: '—', link: 'POWER ›', refresh: 'weekly',
    open: { node: 'pulse', workspace: 'race' }, needs: [], missing: 'No power ranking read for an app league' },
  plan: { t: 'PLANS', src: 'War Room · Plans', prov: '—', link: 'PLANS ›', refresh: 'on edit',
    open: { node: 'war' }, needs: [], missing: 'No plans checklist is stored' },
  picks: { t: 'MY PICKS', src: 'War Room · Draft Room', prov: 'draft.read', link: 'DRAFT ›',
    refresh: 'on trade', open: { node: 'war' }, needs: ['draft'] },
  cap: { t: 'CAP RUNWAY', src: 'Franchise HQ', prov: 'native.cap.check + native.assets.read', link: 'CAP ›',
    refresh: 'on move',
    open: { node: 'hq', workspace: 'contracts-and-cap' }, needs: ['cap'] },
  bids: { t: 'MY BIDS', src: 'War Room · Market', prov: 'bids.read', link: 'MARKET ›', refresh: 'on bid',
    open: { node: 'war' }, needs: ['bids'] },
  form: { t: 'FORM · LAST 5', src: 'Franchise HQ', prov: 'research.TargetResearchPlayer (Form · Wk, 5)',
    link: 'HQ ›', refresh: 'after each final', open: { node: 'hq' }, needs: ['results', 'form'] },
};

export const go = (open?: { node: Node; workspace?: string }) => {
  if (open) commands.dispatch('nav.open', open as never);
};
const row = (cls: string, children: ReactNode, key: string | number, extra: Record<string, unknown> = {}) =>
  <div className={`r ${cls}`} key={key} {...extra}>{children}</div>;
export const nameOf = (b: HomeBrief, id: string) =>
  b.pulse?.seats.find(s => s.franchise === id)?.name ?? id;
const pts = (s?: { points?: number; provisional?: number }) => s?.points ?? s?.provisional;
/** A read's stand-in label for a player it cannot name ("player 13448"). */
const STAND_IN = /^player \d+$/i;
/** playerName is HQ's resolver: the identity block's held name (nativeHQAssets), then the read's own
 *  name unless it is only a stand-in, then the id. */
export function playerName(b: HomeBrief, id: string, given = ''): string {
  const held = b.assets?.players.value.find(p => p.id === id)?.name;
  if (held && !STAND_IN.test(held)) return held;
  return given && !STAND_IN.test(given) ? given : held || given || id;
}

export function myGame(b: HomeBrief, mine: string) {
  return b.results?.games.find(g => g.home === mine || g.away === mine);
}
/** ranked says the standings hold at least one decided game (or a commissioner's ruling). Before then
 *  every row is 0-0 and rank is list order, which reads as a fact it is not: Home shows "—" instead,
 *  with the standings' own words saying why (e.g. "0 games have a result, 48 do not yet"). */
export function ranked(b: HomeBrief): boolean {
  const s = b.pulse?.standings;
  return !!s && (s.decided > 0 || s.overrideWords !== undefined);
}
export function myRow(b: HomeBrief, mine: string) {
  return ranked(b) ? b.pulse?.standings?.rows.find(r => r.franchise === mine) : undefined;
}
const recordOf = (b: HomeBrief, id: string) =>
  ranked(b) ? b.pulse?.standings?.rows.find(r => r.franchise === id)?.record ?? '—' : '—';

/** tileState is a tile's lifecycle from its sources: loading until each has answered or failed,
 *  stale once the oldest answer is past STALE_MS, else fresh. Unavailable and error show in the body. */
export const STALE_MS = 10 * 60_000;
export function tileState(id: TileId, b: HomeBrief, now = Date.now()): 'loading' | 'stale' | 'fresh' {
  const needs = TILES[id].needs;
  if (needs.some(n => !b.at[n] && !b.errors[n])) return 'loading';
  const oldest = oldestAt(id, b);
  return oldest && now - Date.parse(oldest) > STALE_MS ? 'stale' : 'fresh';
}
export function oldestAt(id: TileId, b: HomeBrief): string | undefined {
  return TILES[id].needs.map(n => b.at[n]).filter((x): x is string => Boolean(x)).sort()[0];
}
/** clock writes an instant as Central time, as the mock's footers do ("as of 14:05"). */
export function clock(iso: string, seconds = false): string {
  return new Date(iso).toLocaleTimeString('en-GB', { timeZone: 'America/Chicago', hour: '2-digit',
    minute: '2-digit', ...(seconds ? { second: '2-digit' } : {}), hour12: false });
}

export function TileBody({ id, b, mine, onTx, pins }: { id: TileId; b: HomeBrief;
  mine: string; onTx: (index: number) => void; pins: TileId[] }) {
  const def = TILES[id];
  if (def.missing) return <div className="empty">{unavailable} · {def.missing}</div>;
  const failed = def.needs.map(n => b.errors[n]).find(Boolean);
  if (failed) return <div className="empty">{unavailable} · {failed}</div>;
  if (tileState(id, b) === 'loading') return <div className="empty loading">Reading…</div>;
  switch (id) {
    case 'mu': return <Matchup b={b} mine={mine} />;
    case 'race': return <Race b={b} mine={mine} />;
    case 'wire': return <>{(b.ledger?.entries ?? []).slice().reverse().map((e, i) =>
      row('wire', <><span className="m">{shortTime(e.when)}</span>
        <span className="tx">{e.what.split(' ')[0].toUpperCase()}</span>
        <span className="nm">{e.what}<small>{e.who}{e.why ? ` · ${e.why}` : ''}</small></span></>, e.id,
      { 'data-tx': i, onClick: () => onTx(i) }))}
      {b.ledger && !b.ledger.entries.length && <div className="empty">No transactions in the ledger yet.</div>}</>;
    case 'sc': return <Shortcuts pins={pins} />;
    case 'form': return <Form b={b} />;
    case 'news': return <News b={b} mine={mine} />;
    case 'picks': {
      const picks = b.draft?.order.flatMap(o => o.picks).filter(p => p.owner === mine) ?? [];
      return <>{picks.map(p => row('pk', <><span className="v" style={{ textAlign: 'left' }}>
        {b.draft!.season} {p.round}.{String(p.slot).padStart(2, '0')}</span>
        <span className="nm m">{p.original === mine ? 'own' : `from ${nameOf(b, p.original)}`}</span>
        <span className="m" style={{ textAlign: 'right' }}>—</span></>, `${p.round}.${p.slot}`))}
        {b.draft && !picks.length && <div className="empty">You hold no picks in the {b.draft.season} draft.</div>}</>;
    }
    case 'cap': {
      const [now, ...later] = b.runway ?? [];
      if (!now) return <div className="empty">{unavailable} · no cap runway for this franchise</div>;
      const pct = (v?: number) => now.max ? `${Math.min(100, ((v ?? 0) / now.max) * 100)}%` : '0%';
      return <><div className="capr"><b>{money(now.room)}</b><span>ROOM {now.season} · of {money(now.max)}</span>
        <div className="capbar"><i style={{ width: pct(now.committed) }} /><u style={{ width: pct(now.dead) }} /><s /></div>
        <div className="m" style={{ font: '10px var(--mono)', color: 'var(--ink3)', marginTop: 5 }}>
          ■ committed {money(now.committed)} · ▤ dead {money(now.dead)}</div></div>
        {[now, ...later].map(c => row('yr', <><span className="v" style={{ textAlign: 'left' }}>{c.season}</span>
          <span className="nm">{money(c.room)} room</span><span className="m" style={{ textAlign: 'right' }}>
            {capNote(b, mine, c.season)}</span></>, c.season))}</>;
    }
    case 'bids': return <>{(b.bids?.players ?? []).filter(p => p.own).map(p => {
      const high = p.leader?.id === p.own!.id;
      return row('bd', <><span className="nm">{p.name}</span><span className="v">{p.own!.words}</span>
        <span className={`k ${high ? 'hi' : 'out'}`}>{high ? 'PLACED · HIGH' : 'OUTBID'}</span></>, p.player);
    })}{b.bids && !b.bids.players.some(p => p.own) &&
      <div className="empty">No open bids.</div>}</>;
    case 'due': {
      const now = Date.now();
      const due = (b.calendar ?? []).map(e => ({ ...e, when: e.live ? e.closes : e.at }))
        .filter(e => e.when && Date.parse(e.when) >= now)
        .sort((x, y) => Date.parse(x.when!) - Date.parse(y.when!));
      return <>{due.map(d => row('dq', <><span className="k clock">CLOCK</span><span className="nm">{d.label}</span>
        <span className="m">{shortTime(d.when!)}</span>
        <span className="clock" style={{ textAlign: 'right' }}>{left(d.when!, now)}</span></>, d.id))}</>;
    }
    default: return <div className="empty">{unavailable}</div>;
  }
}

function Matchup({ b, mine }: { b: HomeBrief; mine: string }) {
  const g = myGame(b, mine);
  if (!g) return <div className="empty">{b.results?.bye === mine ? 'Bye week.' : 'No game this week.'}</div>;
  const opp = g.home === mine ? g.away : g.home;
  const side = (id: string) => b.results?.scores.find(s => s.franchise === id);
  const rec = (id: string) => recordOf(b, id);
  const me = side(mine), op = side(opp);
  const proj = (id: string) => projected(b, id);
  const ups = b.matchup?.lineups[mine] ?? [];
  const final = isFinal(g.status);
  const locked = ups.filter(u => u.locked).length;
  const pair = Array.from({ length: Math.max(me?.starters.length ?? 0, op?.starters.length ?? 0) },
    (_, i) => [me?.starters[i], op?.starters[i]] as [PulseStarter | undefined, PulseStarter | undefined]);
  return <>
    <div className="mx"><div className="sd me"><b>{nameOf(b, mine)}</b><span>{rec(mine)} · proj {proj(mine)}</span>
      <div className="sc">{pointsLabel(pts(me))}</div></div><div className="vs">vs</div>
      <div className="sd op rt"><b>{nameOf(b, opp)}</b><span>{rec(opp)} · proj {proj(opp)}</span>
        <div className="sc">{pointsLabel(pts(op))}</div></div></div>
    <div className="wpl" title="Win probability is computed only for the MFL mirror's Now board (M-018)">
      <span className="g">—</span><span className="m" style={{ font: '10px var(--mono)',
      color: 'var(--ink3)' }}>WIN % · {unavailable}</span><span className="a">—</span></div>
    <div className="prog" title="Kicked-off starters are locked; the lineup read cannot split playing from done">
      <div><b>{final ? 0 : ups.length ? locked : '—'}</b><span>PLAYING</span></div>
      <div><b>{final ? 0 : ups.length ? ups.length - locked : '—'}</b><span>YET TO PLAY</span></div>
      <div><b>{final ? me?.starters.length ?? '—' : '—'}</b><span>DONE</span></div></div>
    {pair.map(([a, c], i) => row('duo', <><span className="nm">{a ? playerName(b, a.player, a.name) : '—'} <span className="m">
      {a?.position ?? ''}</span></span><span className="v me">{pointsLabel(pts(a))}</span>
      <span className="v op">{pointsLabel(pts(c))}</span><span className="nm rt">{c ? playerName(b, c.player, c.name) : '—'}</span></>, i))}
  </>;
}

function Race({ b, mine }: { b: HomeBrief; mine: string }) {
  const rows = b.pulse?.standings?.rows ?? [];
  const me = rows.find(r => r.franchise === mine);
  const cut = (b.pulse?.bracket?.fields ?? []).reduce((n, f) => n + f.seeds.length, 0);
  const next = (b.pulse?.fixtures ?? []).filter(f => f.week > (b.pulse?.week ?? 0)).slice(0, 5);
  const i = rows.findIndex(r => r.franchise === mine);
  const window = rows.slice(Math.max(0, i - 2));
  return <>
    <div className="seedl" title={ranked(b) ? undefined : b.pulse?.standings?.words}><div><b>{ranked(b) ? me?.rank ?? '—' : '—'}</b><span>SEED NOW</span></div>
      <div><b>—</b><span>PROJECTED</span></div><div><b>—</b><span>PLAYOFF ODDS</span></div>
      <div><b>—</b><span>POWER</span></div></div>
    <div className="runin">{next.map(f => <div key={f.week}>WK {f.week}<b>{f.venue === 'away' ? '@ ' : ''}
      {nameOf(b, f.opponent)}</b><i>—</i></div>)}</div>
    {window.map(r => row(`mini ${r.franchise === mine ? 'me' : r.rank === cut ? 'cut' : ''}`, <>
      <span className="m">{r.rank}</span><span className="nm">{r.name}</span><span className="v">{ranked(b) ? r.record : '—'}</span>
      <span className="m" style={{ textAlign: 'right' }}>{r.rank === cut ? 'CUT ─' : ''}</span></>, r.franchise))}
  </>;
}

/** projected is a side's projection: the sum of its starters' medians, only when every starter has one. */
export function projected(b: HomeBrief, franchise: string): string {
  const starters = b.results?.scores.find(s => s.franchise === franchise)?.starters ?? [];
  const values = starters.map(s => b.matchup?.projections[s.player]);
  if (!starters.length || values.some(v => v === undefined)) return '—';
  return (values as number[]).reduce((n, v) => n + v, 0).toFixed(1);
}

/** capNote names what ends in a season on my roster: "RFA: name" / "UFA: name", else "N expiring". */
export function capNote(b: HomeBrief, mine: string, season: number): string {
  const s = b.assets;
  if (!s) return '';
  const base = s.league.value.season;
  const ending = (s.rosters.value.find(r => r.franchiseId === mine)?.players ?? [])
    .filter(p => p.yearsRemaining !== undefined && base + p.yearsRemaining - 1 === season);
  if (!ending.length) return '';
  const name = (id: string) => s.players.value.find(p => p.id === id)?.name ?? id;
  const tagged = ending.filter(p => p.contractStatus === 'RFA' || p.contractStatus === 'UFA');
  const head = tagged.slice(0, 2).map(p => `${p.contractStatus}: ${name(p.id)}`).join(' · ');
  return season === base || !head ? `${ending.length} expiring` : head;
}

function Form({ b }: { b: HomeBrief }) {
  const rows = (b.form ?? []).filter(f => f.weeks.length);
  if (!rows.length) return <div className="empty">No finished weeks held for your starters yet.</div>;
  return <>{rows.map(f => {
    const avg = f.weeks.reduce((n, w) => n + w.points, 0) / f.weeks.length;
    return row('pw', <><span className="m">{f.position}</span><span className="nm">{f.name}</span>
      <span className="v">{Math.round(avg)}</span><span className="form" title={f.weeks.map(w =>
        `W${w.week}: ${w.points}`).join(' · ')} style={{ display: 'flex', gap: 2, alignItems: 'end', height: 18 }}>
        {f.weeks.map(w => <i key={w.week} style={{ width: 8, height: Math.max(0, Math.min(18, w.points / 1.8)),
          background: 'var(--edge-hi)', display: 'block' }} />)}</span></>, f.player);
  })}</>;
}

/** newsLine is an article's second line: the player's held name, else the research name when research
 *  knows him; a player research cannot place (position "Unknown", the directory-miss words) shows nothing. */
export function newsLine(b: HomeBrief, a: { id: string; name: string }): string {
  const held = b.assets?.players.value.find(p => p.id === a.id)?.name;
  if (held && !STAND_IN.test(held)) return held;
  const known = b.research?.players.find(p => p.id === a.id);
  return known && known.position !== 'Unknown' && known.name === a.name ? a.name : '';
}
const NF = { all: 'ALL', mine: 'MY PLAYERS', inj: 'INJURIES', wire: 'ON THE WIRE' } as const;
const INJ = /\b(injur|out|IR\b|questionable|doubtful|susp|hamstring|ankle|knee|concussion)/i;
function News({ b, mine }: { b: HomeBrief; mine: string }) {
  const [f, setF] = useState<keyof typeof NF>('all');
  const owner = new Map((b.assets?.rosters.value ?? []).flatMap(r => r.players.map(p => [p.id, r.franchiseId])));
  const items = (b.research?.articles ?? []).map(a => ({ ...a, who: owner.get(a.id) }))
    .filter(a => f === 'all' || (f === 'inj' ? INJ.test(a.headline)
      : f === 'mine' ? a.who === mine : a.who === undefined));
  return <><div className="nfb">{(Object.keys(NF) as (keyof typeof NF)[]).map(k => <button key={k}
    aria-pressed={k === f} onClick={e => { e.stopPropagation(); setF(k); }}>{NF[k]}</button>)}
    <span>{items.length} items</span></div>
    {items.map(a => row(`nw o-${a.who === mine ? 'mine' : a.who ? 'x' : 'wire'}`, <><span className="m" />
      <span className={`k${INJ.test(a.headline) ? ' n-inj' : ''}`}>{INJ.test(a.headline) ? 'INJ' : 'NEWS'}</span>
      <span className="nm"><b>{a.headline}</b><small>{newsLine(b, a)}</small></span>
      <span className="own">{a.who === mine ? 'Yours' : a.who ? nameOf(b, a.who) : 'Free agent'}</span></>,
    `${a.id}.${a.article}`))}
    {!items.length && <div className="empty">No held news{f === 'all' ? '' : ` for ${NF[f]}`}.</div>}</>;
}

type Short = [string, string, { node: Node; workspace?: string } | undefined];
const SHORTCUTS: [string, string, Short[]][] = [
  ['LINEUP · ROSTER', 'HQ', [['Set lineup', 'L', { node: 'hq', workspace: 'lineup-and-roster' }],
    ['Roster', 'O', { node: 'hq', workspace: 'lineup-and-roster' }],
    ['Cap', 'C', { node: 'hq', workspace: 'contracts-and-cap' }],
    ['Taxi · IR', 'I', { node: 'hq', workspace: 'lineup-and-roster' }]]],
  ['DEALS', 'WAR ROOM', [['Trade Desk', 'T', { node: 'war' }], ['Market', 'M', { node: 'war' }],
    ['Draft Room', 'D', { node: 'war' }], ['Plans', 'P', undefined]]],
  ['LEAGUE', 'PULSE', [['Live', 'G', { node: 'pulse', workspace: 'now' }],
    ['Race', 'R', { node: 'pulse', workspace: 'race' }], ['Archive', 'A', undefined],
    ['Rules', 'U', { node: 'control', workspace: 'league' }]]],
];
/** SHORTCUT_KEYS maps each bound key to where it opens; a disabled shortcut has no key binding. */
export const SHORTCUT_KEYS: Record<string, { node: Node; workspace?: string }> = Object.fromEntries(
  SHORTCUTS.flatMap(([, , xs]) => xs.filter(x => x[2]).map(([, k, open]) => [k.toLowerCase(), open!])));
function Shortcuts({ pins }: { pins: TileId[] }) {
  const pinned: [string, string, Short[]][] = pins.length ? [['PINNED', '☆', pins.map(id =>
    [TILES[id].t, '', TILES[id].open] as Short)]] : [];
  return <><div className="scx">{[...pinned, ...SHORTCUTS].map(([g, home, xs]) => <div className="scgp" key={g}>
    <div className="sch"><b>{g}</b><span>{home}</span></div>
    {xs.map(([label, key, open]) => <button className="scb" key={label} disabled={!open}
      title={open ? undefined : unavailable} onClick={e => { e.stopPropagation(); go(open); }}>{label}
      {key && <kbd>{key}</kbd>}</button>)}
  </div>)}</div>
  <div className="scf">☆ any screen to add it here · drag ⠿ to reorder in CUSTOMIZE</div></>;
}

/** shortTime writes a time the way the mock does: clock time today, weekday this week, else the date. */
export function shortTime(iso: string, now = Date.now()): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  const days = Math.abs(now - d.getTime()) / 86_400_000;
  if (new Date(now).toDateString() === d.toDateString())
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: false });
  if (days < 7) return d.toLocaleDateString(undefined, { weekday: 'short' });
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}
export const isFinal = (status?: string) => status?.toLowerCase() === 'final';
export function left(iso: string, now: number): string {
  const s = Math.max(0, (Date.parse(iso) - now) / 1000);
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  return d > 3 ? `${d}d` : d ? `${d}d ${h}h` : `${h}h ${m}m`;
}
/** topFive is the ticker: each position's five best starters this week in this league. */
export function topFive(b: HomeBrief): [string, PulseStarter[]][] {
  const by = new Map<string, PulseStarter[]>();
  for (const s of b.results?.scores.flatMap(x => x.starters) ?? []) {
    if (!s.position || pts(s) === undefined) continue;
    by.set(s.position, [...(by.get(s.position) ?? []), { ...s, name: playerName(b, s.player, s.name) }]);
  }
  return Array.from(by.entries()).map(([p, xs]): [string, PulseStarter[]] =>
    [p, xs.sort((a, c) => pts(c)! - pts(a)!).slice(0, 5)]);
}
export { pts };
