// Desktop v3 Franchise HQ, after TheWarRoom-mock-v3-FINAL.html (screens.js hq.lineup): the plate,
// starters, bench with taxi and IR, and the This week, Win chance and Form panels. The reads, the lineup
// editor and the save review are Franchise HQ's own (readNativeHQ, useNativeLineup); only the look is v3.
import { Fragment, useRef, useState, type PointerEvent, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { boardSlots, type HQLineup, type HQSlot } from '../shell/HQBoard';
import { fitsSlot, placePlayer } from '../shell/hqPlacement';
import { totalProjection } from '../shell/hqPlayerData';
import type { HQSettings } from '../shell/hqModel';
import type { NativeHQ } from '../commissioner/hqReading';
import type { HostedPort } from '../commissioner/hostedRoom';
import { pointsLabel, type PulseResults } from '../data/nativePulse';
import { readPulseOdds, readRecentResults, usePulseRead } from '../shell/pulseReads';
import type { useNativeLineup } from '../shell/hqNativeLineup';
import type { HQTab } from '../shell/HQShell';
import { Icon } from './icons';
import { Seg } from './PulseV3';

type Editor = ReturnType<typeof useNativeLineup>;
const ORDER = ['QB', 'RB', 'WR', 'TE', 'FLEX', 'OL', 'DL', 'DT', 'DE', 'LB', 'DB', 'CB', 'S', 'D·FLX',
  'K', 'PK'];
const rank = (code: string) => { const i = ORDER.indexOf(code); return i < 0 ? ORDER.length : i; };
const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
const cash = (v?: number | null) => v == null ? '—' : `$${(v / 100_000_000).toFixed(1)}M`;
const TABS: [HQTab, string][] = [['lineup-and-roster', 'Lineup and roster'],
  ['contracts-and-cap', 'Contracts and cap'], ['my-moves', 'My moves']];

export function HQTabs({ tab, navigate, badge, kind, moves, pick }: { tab: HQTab;
  navigate: (t: HQTab) => void; badge: ReactNode; kind: string; moves: number; pick?: ReactNode }) {
  return <div className="ws-tabs"><span className="ttl"><Icon n="shield" />FRANCHISE HQ</span>
    {TABS.map(([t, words]) => <HQAct key={t} className="tb" selected={tab === t} onAction={() => navigate(t)}>
      {words}{t === 'my-moves' && moves > 0 && <span className="tag am">{moves}</span>}</HQAct>)}
    <div className="tools">{pick}<span className="d mono" style={{ fontSize: 10, letterSpacing: '.08em' }}>
      {badge} {kind}</span></div></div>;
}

/** The commissioner's franchise choice: every team in the league, theirs to run. */
export function TeamPick({ teams, mine, pick }: { teams: { id: string; name: string }[]; mine: string;
  pick: (id: string) => void }) {
  return <label className="d mono" style={{ fontSize: 10, letterSpacing: '.08em', display: 'flex', gap: 6,
    alignItems: 'center' }}>FRANCHISE<select value={mine} onChange={e => pick(e.target.value)}
      style={{ font: '600 11px var(--mono)', background: 'var(--sunken)', color: 'var(--ink)',
        border: '1px solid var(--edge)', padding: '3px 6px' }}>
      {!mine && <option value="">Choose a franchise…</option>}
      {[...teams].sort((a, b) => a.name.localeCompare(b.name)).map(t =>
        <option key={t.id} value={t.id}>{t.name}</option>)}</select></label>;
}

/** When a starter's game is, in the mock's words: LIVE, a kickoff time, TNF / SNF / MNF, or BYE. */
export function kickWords(kickoff: string | undefined, now: number): { w: string; c: string } {
  if (!kickoff) return { w: 'BYE', c: 'd' };
  const at = Date.parse(kickoff);
  if (Number.isNaN(at)) return { w: '—', c: 'd' };
  // An NFL game runs about three and a half hours from kickoff, with the scoring settled after it.
  if (at <= now) return now - at < 4 * 3600_000 ? { w: 'LIVE', c: 'b' } : { w: 'DONE', c: 'd' };
  const parts = new Intl.DateTimeFormat('en-US', { timeZone: 'America/Chicago', weekday: 'short',
    hour: 'numeric', minute: '2-digit', hour12: true }).formatToParts(new Date(at));
  const get = (t: string) => parts.find(p => p.type === t)?.value ?? '';
  const day = get('weekday'), hour = Number(get('hour')), pm = get('dayPeriod') === 'PM';
  if (day === 'Thu') return { w: 'TNF', c: 'd' };
  if (day === 'Mon') return { w: 'MNF', c: 'd' };
  if (day === 'Sun' && pm && hour >= 7 && hour < 12) return { w: 'SNF', c: 'd' };
  return { w: `${day === 'Sun' ? '' : `${day.toUpperCase()} `}${hour}:${get('minute')}`, c: 'd' };
}

function Form({ points, projection }: { points?: { week: number; points: number }[]; projection?: number }) {
  const shown = (points ?? []).slice(-5);
  const max = Math.max(1, projection ?? 0, ...shown.map(p => Math.abs(p.points)));
  const words = shown.length ? `last ${shown.length} started weeks` : 'no started weeks';
  return <span className="fm" title={words}>
    {shown.map(p => <i key={p.week} className={projection !== undefined && p.points >= projection ? 'up' : ''}
      style={{ height: `${Math.max(4, Math.abs(p.points) / max * 100)}%` }} />)}</span>;
}

export function HQLineupV3({ reading, mine, name, editor, select, badge, league, port, now, openGame }: {
  reading: NativeHQ; mine: string; name: string; editor: Editor; select: (id: string) => void;
  badge: ReactNode; league: string; port?: HostedPort; now: number; openGame?: () => void }) {
  const { snapshot } = reading;
  const settings: HQSettings = reading.settings;
  const lineup: HQLineup | undefined = editor.edited;
  const values = reading.projections ?? new Map<string, number>();
  const [picked, setPicked] = useState<string>();
  const [drag, setDrag] = useState<{ id: string; x: number; y: number; on: boolean }>();
  const skip = useRef(false);
  const slots = boardSlots(lineup, settings).sort((a, b) => rank(a.code) - rank(b.code));
  const players = lineup ? lineup.starters.concat(lineup.bench) : [];
  const directory = new Map(snapshot.players.value.map(p => [p.id, p]));
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const contract = new Map(roster.map(p => [p.id, p]));
  const posOf = (id: string) => players.find(p => p.id === id)?.position
    || directory.get(id)?.position || '—';
  const benched = players.filter(p => !slots.some(s => s.id === p.id));
  const moving = drag?.on ? drag.id : picked;
  const canPlace = (id: string, slot: HQSlot) => editor.canEdit && fitsSlot(players.find(p => p.id === id),
    slot, settings) && (!slot.id || !slots.some(s => s.id === id) || fitsSlot(players.find(p => p.id ===
      slot.id), slots.find(s => s.id === id)!, settings));
  const move = (id: string, slot?: HQSlot) => {
    if (!editor.canEdit || (slot && !canPlace(id, slot))) return;
    const next = placePlayer(slots, players, settings, values, id, slot ? slots.indexOf(slot) : undefined,
      !slot);
    if (next !== slots) editor.assign(next.flatMap(s => s.id ? [s.id] : []));
  };
  const drop = (e: PointerEvent) => {
    if (!drag) return;
    if (drag.on) {
      skip.current = true;
      const at = document.elementFromPoint(e.clientX, e.clientY);
      const slot = at?.closest<HTMLElement>('[data-slot]');
      if (slot) move(drag.id, slots[Number(slot.dataset.slot)]);
      else if (at?.closest('[data-benchzone]') && slots.some(s => s.id === drag.id)) move(drag.id);
    }
    setDrag(undefined);
  };
  const tag = (id: string) => <span style={{ display: 'contents' }} onPointerDown={e => {
    if (editor.canEdit) setDrag({ id, x: e.clientX, y: e.clientY, on: false }); }}>
    <HQAct className="pos drag" label={`Move ${directory.get(id)?.name ?? id}`}
      title="Drag to another slot or the bench. Or click it, then click the slot."
      onAction={() => { if (skip.current) { skip.current = false; return; }
        if (editor.canEdit) setPicked(picked === id ? undefined : id); }}>{posOf(id)}</HQAct></span>;
  const nm = (id: string) => <HQAct className="nm" onAction={() => {
    if (picked && picked !== id) { const s = slots.find(x => x.id === id); if (s) move(picked, s);
      else if (slots.some(x => x.id === picked)) move(picked); setPicked(undefined); } else select(id); }}>
    {directory.get(id)?.name ?? id}<small>{directory.get(id)?.team}</small></HQAct>;
  const state = (id: string, reserve?: string) => {
    if (reserve) return <span className="stt d">{reserve}</span>;
    const k = kickWords(reading.kickoffs?.get(id), now);
    return <span className={`stt ${k.c}`}>{k.w}</span>;
  };
  const cls = (id: string | undefined, slot?: HQSlot) => [
    picked === id && id ? 'picked' : '', id && editor.changed.includes(id) ? 'chg' : '',
    moving && slot ? canPlace(moving, slot) ? 'legal' : 'illegal' : ''].join(' ');
  const set = slots.filter(s => s.id).length;
  const total = lineup ? totalProjection(slots.flatMap(s => s.id ? [s.id] : []), values) : '—';
  const place = reading.place ? ` · seed ${reading.place.rank}` : '';
  const reserves = (kind: 'TAXI_SQUAD' | 'IR') => roster.filter(p => p.rosterStatus === kind);
  const room = reading.runway?.[0]?.room;
  return <div className="wbody scr scr-hq" onPointerMove={e => {
    if (drag && (drag.on || Math.hypot(e.clientX - drag.x, e.clientY - drag.y) > 5))
      setDrag({ ...drag, x: e.clientX, y: e.clientY, on: true }); }}
    onPointerUp={drop} onPointerCancel={() => setDrag(undefined)}>
    <div className="hq-plate">
      <div className="who"><b>{name}</b>
        <div>{reading.record ?? '—'}{place}{reading.week ? ` · week ${reading.week}` : ''}</div></div>
      <div className="chk"><span className="h">LINEUP</span><span className="v">{lineup
        ? `${set} of ${slots.length} set` : '—'} <i className={`lamp ${set === slots.length ? 'g' : 'a'}`} />
      </span></div>
      <div className="chk"><span className="h">PROJECTED</span><span className="v mono">{total}</span></div>
      <div className="chk"><span className="h">CAP ROOM</span>
        <span className="v mono">{cash(room)}</span></div>
      <div className="chk"><span className="h">LOCK</span><span className="v a">{reading.kickoffs?.size
        ? 'each player at his kickoff' : reading.lock ? new Date(reading.lock).toLocaleString('en-US', {
          timeZone: 'America/Chicago', weekday: 'short', hour: 'numeric', minute: '2-digit' }) : '—'}
      </span></div>
      <span className="sp" />
      {editor.dirty || editor.state.review ? <>
        <span className="d mono" style={{ fontSize: 10, maxWidth: 280, lineHeight: 1.5 }}>
          {editor.message}</span>
        <HQAct className="btn" disabled={editor.state.pending} onAction={editor.discard}>DISCARD</HQAct>
        <HQAct className="btn pri" disabled={!editor.canEdit || !editor.changed.length}
          onAction={() => { void editor.save(); }}>CHECK AND SAVE</HQAct></>
        : <span className="d mono" title={reading.errors.join(' · ')}
          style={{ fontSize: 10, maxWidth: 280, lineHeight: 1.5 }}>{editor.canEdit
          ? 'Drag a position tag to move a player, or click it and then click the slot. Illegal slots dim.'
          : editor.message}</span>}
    </div>
    <div className="hq-board">
      <div className="colm" id="starters">
        <div className="ch"><b>STARTERS</b><span>slot · player · last five · proj</span></div>
        {!lineup ? <St>{reading.errors.join(' · ') || 'Reading the lineup…'}</St>
          : slots.map((slot, i) => <div key={slot.id ?? `gap${i}`} data-slot={i}
            className={`slot ${slot.id ? '' : 'gap'} ${cls(slot.id, slot)}`}>
            <span className="sl">{slot.code}</span>
            {slot.id ? <>{tag(slot.id)}{nm(slot.id)}
              <Form points={reading.playerForm?.[slot.id]} projection={values.get(slot.id)} />
              <span className="pr">{values.get(slot.id)?.toFixed(1) ?? '—'}</span>{state(slot.id)}</>
              : <><span /><HQAct className="nm" onAction={() => { if (picked) { move(picked, slot);
                setPicked(undefined); } }}>Empty · any eligible player</HQAct><span /><span />
                <span className="stt a">OPEN</span></>}
          </div>)}
      </div>
      <div className="colm" id="bench">
        <div className="ch"><b>BENCH</b><span>drop here to sit a starter</span></div>
        <div className="benchzone" data-benchzone="">
          {benched.map(p => <div key={p.id} className={`slot bn ${cls(p.id)}`}>
            {tag(p.id)}{nm(p.id)}
            <Form points={reading.playerForm?.[p.id]} projection={values.get(p.id)} />
            <span className="pr">{cash(contract.get(p.id)?.salary)}</span>
            <span className="pr">{values.get(p.id)?.toFixed(1) ?? '—'}</span>{state(p.id)}</div>)}
          {lineup && !benched.length && <St>No one on the bench.</St>}
        </div>
        {([['TAXI', 'TAXI_SQUAD', settings.taxi], ['IR', 'IR', settings.ir]] as const)
          .map(([label, kind, cap]) => <Fragment key={kind}><div className="ch"><b>{label}</b>
            <span>{reserves(kind).length} of {cap ?? '—'}</span></div>
            {reserves(kind).map(p => <div key={p.id} className="slot bn">
              <span className="pos">{directory.get(p.id)?.position ?? '—'}</span>{nm(p.id)}
              <span className="fm" />
              <span className="pr">{cash(p.salary)}</span><span className="pr" />{state(p.id, label)}</div>)}
          </Fragment>)}
      </div>
      <Side reading={reading} mine={mine} name={name} badge={badge} league={league} port={port} now={now}
        openGame={openGame} />
    </div>
    {drag?.on && <span className="pos" style={{ position: 'fixed', left: drag.x + 6, top: drag.y + 6,
      pointerEvents: 'none', zIndex: 50, padding: '2px 6px', border: '2px solid var(--edge-hi)',
      background: 'var(--tile)' }}>{posOf(drag.id)}</span>}
  </div>;
}

function Panel({ title, badge, live, children, foot }: { title: string; badge: ReactNode; live?: boolean;
  children: ReactNode; foot?: ReactNode }) {
  return <section className="pc panel sz-L"><div className="ph">{badge}<b className="pt">{title}</b>
    <span className="sp" />{live && <span className="lv"><span>LIVE</span></span>}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}
  </section>;
}

/** The mock's win-chance dial: 21 ticks over the half circle, amber low, green high. */
export function Gauge({ pct }: { pct?: number }) {
  const ticks = Array.from({ length: 21 }, (_, i) => {
    const a = Math.PI - i * Math.PI / 20, big = i % 5 === 0, r = big ? 70 : 78;
    return <line key={i} x1={100 + r * Math.cos(a)} y1={100 - r * Math.sin(a)} x2={100 + 86 * Math.cos(a)}
      y2={100 - 86 * Math.sin(a)} stroke={i <= 6 ? 'var(--amber)' : i >= 14 ? 'var(--green)' : 'var(--ink3)'}
      strokeWidth={big ? 3 : 1.5} />;
  });
  return <div className="gauge"><svg viewBox="0 0 200 124">
    <path d="M14 100A86 86 0 01186 100" fill="none" stroke="var(--edge)" strokeWidth="2" />{ticks}
    {pct !== undefined && <g className="nd" style={{ transform: `rotate(${(pct - 50) * 1.8}deg)` }}>
      <path d="M97 100L100 22L103 100z" fill="var(--ink)" /></g>}
    <rect x="92" y="92" width="16" height="16" fill="var(--edge-hi)" />
    <text x="100" y="121" textAnchor="middle" fontFamily="var(--mono)" fontWeight="700" fontSize="17"
      fill="var(--ink)">{pct === undefined ? '—' : `${Math.round(pct)}%`}</text>
    <text x="14" y="116" fontFamily="var(--mono)" fontSize="8" fill="var(--ink3)">0</text>
    <text x="178" y="116" fontFamily="var(--mono)" fontSize="8" fill="var(--ink3)">100</text></svg></div>;
}

function Side({ reading, mine, name, badge, league, port, now, openGame }: { reading: NativeHQ; mine: string;
  name: string; badge: ReactNode; league: string; port?: HostedPort; now: number; openGame?: () => void }) {
  const season = reading.snapshot.league.value.season;
  const week = reading.week ?? 0;
  const odds = usePulseRead(JSON.stringify(['hq-odds', league, season]),
    () => readPulseOdds(league, season, port), 'odds');
  const past = Array.from({ length: Math.max(0, week - 1) }, (_, i) => i + 1);
  const recent = usePulseRead<PulseResults[]>(JSON.stringify(['hq-form', league, season, week]),
    () => readRecentResults(league, season, past, week - 1, port));
  const res = reading.thisWeek;
  const game = res?.games.find(g => g.home === mine || g.away === mine);
  const them = game && (game.home === mine ? game.away : game.home);
  const own = res?.scores.find(s => s.franchise === mine);
  const other = res?.scores.find(s => s.franchise === them);
  const values = reading.projections ?? new Map<string, number>();
  const proj = (ids: string[]) => ids.some(id => values.has(id))
    ? ids.reduce((n, id) => n + (values.get(id) ?? 0), 0).toFixed(1) : '—';
  const mineIds = reading.lineup?.starters.map(p => p.id) ?? [];
  const og = odds.reading?.week === week ? odds.reading.games.find(g => (g.home === mine && g.away === them)
    || (g.away === mine && g.home === them)) : undefined;
  const win = og ? (og.home === mine ? og.homeWin : 1 - og.homeWin) * 100 : undefined;
  const left = mineIds.filter(id => { const k = reading.kickoffs?.get(id); return k && Date.parse(k) > now; })
    .length;
  const live = mineIds.some(id => kickWords(reading.kickoffs?.get(id), now).w === 'LIVE');
  const otherName = other?.name ?? (game ? (game.home === mine ? game.awayName : game.homeName) : undefined);
  const pts = (s?: { points?: number; provisional?: number }) => pointsLabel(s?.points ?? s?.provisional);
  const form = (recent.reading ?? []).map(r => {
    const g = r.games.find(x => x.home === mine || x.away === mine);
    const home = g?.home === mine;
    const p = g && (home ? g.homePoints : g.awayPoints);
    const o = g && (home ? g.awayPoints : g.homePoints);
    return { week: r.week, p, o, won: g?.winner === mine, vs: g && (home ? g.awayName : g.homeName) };
  }).filter(f => f.p !== undefined);
  const top = Math.max(1, ...form.map(f => f.p!)) * 1.07;
  return <div className="colm side">
    <Panel title="This week" badge={badge} live={live}>
      {!game ? <St>{res?.bye === mine ? 'Bye week.' : 'No matchup this week.'}</St> : <>
        <div className="mu"><div className="side"><span className="nm">{name}</span>
          <Seg v={pts(own)} size={34} /><span className="pj">proj {proj(mineIds)}</span></div>
          <span className="vs">VS</span>
          <div className="side rt"><span className="nm">{otherName}</span>
            <Seg v={pts(other)} size={34} /><span className="pj">proj {proj(other?.starters.map(s => s.player)
              ?? [])}</span></div></div>
        <div className="wbar"><i style={{ width: `${win ?? 0}%` }} /></div>
        <div className="wl"><span>{win === undefined ? '— win' : `${Math.round(win)}% win`}</span>
          <span>{left} left to play</span><span>{win === undefined ? '—' : `${100 - Math.round(win)}%`}</span>
        </div></>}
    </Panel>
    <Panel title="Win chance" badge={badge} live={live}
      foot={game ? `${name} vs ${otherName}` : odds.error ? `Win chance unavailable · ${odds.error}` : ''}>
      <HQAct className="clkgo"
        label={`Win chance ${win === undefined ? 'unknown' : `${Math.round(win)} percent`}. Open the game`}
        disabled={!openGame} onAction={openGame}><Gauge pct={win} /></HQAct>
    </Panel>
    <Panel title="Form (last five)" badge={badge} foot="bright = won the week">
      {!form.length ? <St>{recent.error ? `Form unavailable · ${recent.error}`
        : recent.reading ? 'No finished weeks yet.' : 'Reading recent weeks…'}</St>
        : <div className="fmbig">{form.map(f => <HQAct key={f.week} className={f.won ? 'up' : ''}
          style={{ height: `${f.p! / top * 100}%` }}
          title={`Week ${f.week}: ${f.p!.toFixed(1)} vs ${f.vs ?? '—'} ${f.o?.toFixed(1) ?? ''}`}
          label={`Week ${f.week}: ${f.p!.toFixed(1)} points`}><b>{Math.round(f.p!)}</b></HQAct>)}</div>}
    </Panel>
  </div>;
}
