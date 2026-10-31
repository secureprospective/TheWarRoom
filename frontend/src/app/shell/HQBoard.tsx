import { useRef, useState, type PointerEvent } from 'react';
import type { LineupReading, Snapshot } from '../data/contract';
import type { HQSettings } from './hqModel';
import { unavailable } from './hqModel';
import { formPoints, totalProjection } from './hqPlayerData';
import type { Now } from '../now/contract';
import type { RosterFacts } from './hqRosterFacts';

export type HQBoardProps = { snapshot: Snapshot; mine: string; reading?: LineupReading;
  settings: HQSettings; values: Map<string, number>; facts?: RosterFacts;
  byes?: Map<string, number>; now?: Now; changed?: string[]; canEdit?: boolean; select: (id: string) => void;
  place?: (id: string, out?: string) => void; bench?: (id: string) => void };
const defensive = (position?: string) => ['DT', 'DE', 'LB', 'CB', 'S', 'DL', 'DB'].includes(position ?? '');
export type HQSlot = { code: string; side: 'off' | 'def'; id?: string };
export function boardSlots(reading: LineupReading | undefined, settings: HQSettings): HQSlot[] {
  if (!reading) return [];
  const remaining = [...reading.starters];
  const slots: HQSlot[] = [];
  for (const rule of settings.positions) for (let n = 0; n < rule.min; n++) {
    const index = remaining.findIndex(p => p.position === rule.code);
    const player = index < 0 ? undefined : remaining.splice(index, 1)[0];
    slots.push({ code: rule.code, side: defensive(rule.code) ? 'def' : 'off', id: player?.id });
  }
  for (const player of remaining) slots.push({ code: player.position ?? '—',
    side: defensive(player.position) ? 'def' : 'off', id: player.id });
  const fill = (side: HQSlot['side'], count?: number) => {
    if (count === undefined) return;
    while (slots.filter(s => s.side === side).length < count)
      slots.push({ code: side === 'off' ? 'FLEX' : 'D·FLX', side });
  };
  fill('off', settings.offence); fill('def', settings.defence);
  while (slots.length < reading.starterCount) slots.push({ code: '—', side: 'off' });
  return slots;
}
export function FormChart({ id, facts, projection }: { id: string; facts?: RosterFacts; projection?: number }) {
  const points = formPoints(facts?.get(id)?.research);
  if (!points.length) return <span className="form"><em>{unavailable}</em></span>;
  const max = Math.max(1, projection ?? 0, ...points.map(p => Math.abs(p.points))) * 1.08;
  return <span className="form" title={`${points.length} held finished weeks`}>
    {points.map(p => <i key={p.week} className={projection !== undefined && p.points >= projection ? 'up' : ''}
      style={{ height: `${Math.max(6, Math.abs(p.points) / max * 100)}%` }} title={`Wk ${p.week} · ${p.points.toFixed(1)}`} />)}
    {projection !== undefined && <span className="pj" style={{ bottom: `${Math.max(0, projection) / max * 20}px` }} />}
    <b>avg {(points.reduce((n, p) => n + p.points, 0) / points.length).toFixed(1)}</b>
  </span>;
}
export function HQBoard({ snapshot, mine, reading, settings, values, facts, changed = [], canEdit, byes, now,
  select, place, bench }: HQBoardProps) {
  const [filter, setFilter] = useState('ALL');
  const [picked, setPicked] = useState<string>();
  const [drag, setDrag] = useState<{ id: string; x: number; y: number }>();
  const moving = useRef(false);
  const slots = boardSlots(reading, settings);
  const directory = new Map(snapshot.players.value.map(p => [p.id, p]));
  const active = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.filter(p => p.rosterStatus === 'ROSTER') ?? [];
  const starters = reading?.starters ?? [];
  const benchPlayers = reading?.bench ?? active.map(p => ({ id: p.id, position: directory.get(p.id)?.position }));
  const positions = Array.from(new Set([...settings.positions.map(p => p.code), ...benchPlayers.flatMap(p => p.position ? [p.position] : [])]));
  function drop(event: PointerEvent) {
    if (!drag) return;
    const target = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>('[data-hq-slot], [data-hq-bench]');
    if (target?.hasAttribute('data-hq-bench')) bench?.(drag.id);
    else if (target) place?.(drag.id, target.dataset.player || undefined);
    setDrag(undefined); setPicked(undefined);
  }
  const row = (id: string, slot?: HQSlot) => {
    const player = directory.get(id);
    const usage = facts?.get(id)?.usage;
    const injury = snapshot.eligibility.players[id]?.ir;
    const injuryText = facts?.get(id)?.research?.report.rows.find(r => r.cells[0] === 'Injury')?.cells[1] ?? '';
    const bye = byes?.get(id) === reading?.week;
    const status = bye ? 'BYE' : /Questionable|\bQ\b/i.test(injuryText) ? 'Q' : /^Out\b/i.test(injuryText) ? 'OUT' : /^IR\b/.test(injuryText) ? 'IR' : '';
    return <div key={id} className={`slot ${picked === id ? 'picked' : ''} ${changed.includes(id) ? 'chg' : ''} ${picked && slot ? 'target' : ''}`}
      data-player={id} data-hq-slot={slot ? '' : undefined} role="button" tabIndex={0}
      aria-label={`${slot?.code ?? 'Bench'} · ${player?.name ?? id}`}
      onClick={() => { if (moving.current) { moving.current = false; return; } select(id);
        if (slot && picked && canEdit) { place?.(picked, id); setPicked(undefined); }
        else if (!slot) setPicked(id); }}
      onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); e.currentTarget.click(); } }}>
      {slot && <span className="sl">{slot.code}</span>}
      <span className={`pg ${canEdit ? 'drag' : ''}`} onPointerDown={e => {
        if (!canEdit) return;
        e.preventDefault(); moving.current = true;
        e.currentTarget.setPointerCapture(e.pointerId); setDrag({ id, x: e.clientX, y: e.clientY });
      }}>{player?.position ?? '—'}</span>
      <span className="nm">{player?.name ?? id}<small>{player?.team}</small></span>
      <FormChart id={id} facts={facts} projection={values.get(id)} />
      <span className="elig" title={usage?.reason}>{bye ? 'BYE' : usage?.opponent || unavailable}</span>
      <span className="pr">{values.get(id)?.toFixed(1) ?? unavailable}</span>
      <span className={`st ${status === 'Q' ? 'q' : status === 'BYE' ? 'bye' : 'out'}`} title={injury?.reason}>{status}</span>
    </div>;
  };
  return <section className="board ws on" id="ws-lineup" onPointerMove={e => { if (drag) setDrag({ ...drag, x: e.clientX, y: e.clientY }); }} onPointerUp={drop} onPointerCancel={() => setDrag(undefined)}>
    <div className="col"><div className="ch"><b>STARTING {reading?.starterCount ?? settings.starters ?? unavailable}</b><span>proj <b>{reading ? totalProjection(starters.map(p => p.id), values) : unavailable}</b></span></div>
      {(['off', 'def'] as const).map(side => <div key={side}>
        <div className="sec">{side === 'off' ? 'OFFENCE' : 'DEFENCE'} · {side === 'off' ? settings.offence ?? unavailable : settings.defence ?? unavailable}
          <span>{reading ? totalProjection(slots.filter(s => s.side === side && s.id).map(s => s.id!), values) : unavailable}</span></div>
        <div className="slots">{!reading ? <p className="empty">{unavailable}</p> : slots.filter(s => s.side === side).map((slot, i) => slot.id ? row(slot.id, slot)
          : <div key={`empty:${i}`} className={`slot gap ${picked ? 'target' : ''}`} data-hq-slot="" role="button" tabIndex={0}
            onClick={() => { if (picked && canEdit) { place?.(picked); setPicked(undefined); } }}
            onKeyDown={e => { if (e.key === 'Enter') e.currentTarget.click(); }}>
            <span className="sl">{slot.code}</span><span /><span className="nm">Empty</span><span /><span /><span className="pr">—</span><span className="st warn">OPEN</span></div>)}</div>
      </div>)}
      <HQMatchup now={now} />
    </div>
    <div className="col bench" data-hq-bench=""><div className="ch"><b>BENCH · <span>{benchPlayers.length}</span></b><span>drag a tag into the lineup</span></div>
      <div className="filters">{['ALL', ...positions].map(pos => <button key={pos} className="chip" aria-pressed={filter === pos} onClick={() => setFilter(pos)}>{pos}</button>)}</div>
      <div className="blist">{benchPlayers.filter(p => filter === 'ALL' || p.position === filter).map(p => row(p.id))}
        {!benchPlayers.length && <p className="empty">{unavailable}</p>}</div><div className="inl" />
    </div>
    {drag && <span className="pg ghost" style={{ transform: `translate(${drag.x}px,${drag.y}px)` }}>{directory.get(drag.id)?.position ?? '—'}</span>}
  </section>;
}

export function HQMatchup({ now }: { now?: Now }) {
  const report = now?.matchup;
  const opponent = report?.summary.find(r => r.cells[0] === 'Opponent')?.cells[1];
  return <div className="mu"><div className="ch"><b>THIS WEEK{opponent ? ` · VS ${opponent.toUpperCase()}` : ''}</b><span>closest matchups first</span></div>
    <div className="mu-h"><div className="side"><b>{unavailable}</b><span>you · proj</span></div>
      <div className="wp">{unavailable}</div><div className="side them"><b>{unavailable}</b><span>{opponent ?? unavailable} · proj</span></div></div>
    <div className="mu-rows">{report && ['ok', 'stale'].includes(report.status) ? report.rows.map((r, i) => <div key={i}>
      <span className="sl">{r.cells[0]}</span><span>{r.cells[1]} vs {r.cells[3]}</span><span className="n">{unavailable}</span>
    </div>) : <p className="empty">{unavailable}</p>}</div>
  </div>;
}
