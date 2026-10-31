import { HQAct } from './HQAct';
import { useRef, useState, type PointerEvent } from 'react';
import type { LineupReading, Snapshot } from '../data/contract';
import type { HQSettings } from './hqModel';
import { unavailable } from './hqModel';
import { formPoints, totalProjection } from './hqPlayerData';
import type { Injury } from './hqPlayerData';
import type { Now } from '../now/contract';
import type { RosterFacts } from './hqRosterFacts';

export type HQLineup = Omit<LineupReading, 'starters' | 'bench'> & { starters: { id: string; position?:
  string }[]; bench: { id: string; position?: string }[] };
export type HQBoardProps = { snapshot: Snapshot; mine: string; reading?: HQLineup;
  settings: HQSettings; values: Map<string, number>; facts?: RosterFacts;
  byes?: Map<string, number>; injuries?: Injury[]; now?: Now; changed?: string[]; canEdit?: boolean; select: (id: string) => void;
  place?: (id: string, out?: string) => void; bench?: (id: string) => void };
const defensive = (position?: string) => ['DT', 'DE', 'LB', 'CB', 'S', 'DL', 'DB'].includes(position ?? '');
export type HQSlot = { code: string; side: 'off' | 'def'; id?: string };
export function boardSlots(reading: HQLineup | undefined, settings: HQSettings): HQSlot[] {
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
export function FormChart({ id, facts, projection, week, bye }: { id: string; facts?: RosterFacts;
  projection?: number; week?: number; bye?: number }) {
  const points = formPoints(facts?.get(id)?.research);
  if (!points.length && !week) return <span className="form"><em>{unavailable}</em></span>;
  const max = Math.max(1, projection ?? 0, ...points.map(p => Math.abs(p.points))) * 1.08;
  const weeks = week ? Array.from({ length: Math.min(5, Math.max(0, week - 1)) },
    (_, i) => Math.max(1, week - 5) + i) : points.map(p => p.week);
  const real = points.filter(p => p.week !== bye);
  return <span className="form" title={`${points.length} held finished weeks · last 5 vs projection`}>
    {weeks.map(w => { const p = points.find(p => p.week === w);
      if (w === bye) return <i key={w} className="bye" title={`Wk ${w} · bye`} />;
      if (!p) return <i key={w} className="missing" title={`Wk ${w} · ${unavailable}`} />;
      return <i key={p.week} className={projection !== undefined && p.points >= projection ? 'up' :
      ''}
      style={{ height: `${Math.max(6, Math.abs(p.points) / max * 100)}%` }}
        title={`Wk ${p.week} · ${p.points.toFixed(1)}`} />; })}
    {projection !== undefined && <span className="pj" style={{ bottom: `${Math.max(0, projection) / max *
      20}px` }} />}
    <b>avg {real.length ? (real.reduce((n, p) => n + p.points, 0) / real.length).toFixed(1) : '—'}</b>
  </span>;
}
export function HQBoard({ snapshot, mine, reading, settings, values, facts, changed = [], canEdit, byes, injuries, now,
  select, place, bench }: HQBoardProps) {
  const [filter, setFilter] = useState('ALL');
  const [picked, setPicked] = useState<string>();
  const [drag, setDrag] = useState<{ id: string; x: number; y: number }>();
  const moving = useRef(false);
  const slots = boardSlots(reading, settings);
  const directory = new Map(snapshot.players.value.map(p => [p.id, p]));
  const starters = reading?.starters ?? [];
  const benchPlayers = reading?.bench ?? [];
  const positions = Array.from(new Set([...settings.positions.map(p => p.code), ...benchPlayers.flatMap(p =>
     p.position ? [p.position] : [])]));
  function drop(event: PointerEvent) {
    if (!drag) return;
    const target = document.elementFromPoint(event.clientX,
      event.clientY)?.closest<HTMLElement>('[data-hq-slot], [data-hq-bench]');
    if (target?.hasAttribute('data-hq-bench')) bench?.(drag.id);
    else if (target) place?.(drag.id, target.dataset.player || undefined);
    setDrag(undefined); setPicked(undefined);
  }
  const row = (id: string, slot?: HQSlot) => {
    const player = directory.get(id);
    const usage = facts?.get(id)?.usage;
    const injury = snapshot.eligibility.players[id]?.ir;
    const hasForm = formPoints(facts?.get(id)?.research).length > 0;
    const noContext = !hasForm && !usage?.opponent && byes?.get(id) !== reading?.week;
    const injuryText = facts?.get(id)?.research?.report.rows.find(r => r.cells[0] === 'Injury')?.cells[1]
      ?? '';
    const bye = byes?.get(id) === reading?.week;
    const status = bye ? 'BYE' : /Questionable|\bQ\b/i.test(injuryText) ? 'Q' : /^Out\b/i.test(injuryText)
      ? 'OUT' : /^IR\b/.test(injuryText) ? 'IR' : '';
    return <HQAct key={id} className={`slot ${picked === id ? 'picked' : ''} ${changed.includes(id) ? 'chg'
      : ''} ${picked && slot ? 'target' : ''}`}
      attributes={{ 'data-player': id, 'data-hq-slot': slot ? '' : undefined }}
      style={{ width: '100%', textAlign: 'left' }}
      label={`${slot?.code ?? 'Bench'} · ${player?.name ?? id}`}
      onAction={() => { if (moving.current) { moving.current = false; return; } select(id);
        if (slot && picked && canEdit) { place?.(picked, id); setPicked(undefined); }
        else if (!slot) setPicked(id); }}>
      {slot && <span className="sl">{slot.code}</span>}
      <span className={`pg ${canEdit ? 'drag' : ''}`} onPointerDown={e => {
        if (!canEdit) return;
        e.preventDefault(); moving.current = true;
        e.currentTarget.setPointerCapture(e.pointerId); setDrag({ id, x: e.clientX, y: e.clientY });
      }}>{reading?.starters.concat(reading.bench).find(p => p.id === id)?.position ?? player?.position ??
        '—'}</span>
      <span className="nm">{player?.name ?? id}<small>{player?.team}</small></span>
      {noContext ? <span className="form" style={{ gridColumn: slot ? '4 / 6' : '3 / 5' }}>
        <em>{unavailable}</em></span> : <>
        <FormChart id={id} facts={facts} projection={values.get(id)} week={reading?.week} bye={byes?.get(id)} />
        <span className="elig" title={usage?.reason || unavailable}>{bye ? 'BYE' : usage?.opponent ||
          '—'}</span>
      </>}
      <span className="pr" title={values.has(id) ? undefined : unavailable}>
        {values.get(id)?.toFixed(1) ?? '—'}</span>
      <span className={`st ${status === 'Q' ? 'q' : status === 'BYE' ? 'bye' : 'out'}`}
        title={injury?.reason}>{status}</span>
    </HQAct>;
  };
  return <section className="board ws on" id="ws-lineup" onPointerMove={e => { if (drag) setDrag({ ...drag,
    x: e.clientX, y: e.clientY }); }} onPointerUp={drop} onPointerCancel={() => setDrag(undefined)}>
    <div className="col"><div className="ch"><b>STARTING {reading?.starterCount ?? settings.starters ??
      unavailable}</b><span>proj <b>{reading ? totalProjection(starters.map(p => p.id), values) :
      unavailable}</b></span></div>
      {(['off', 'def'] as const).map(side => <div key={side}>
        <div className="sec">{side === 'off' ? 'OFFENCE' : 'DEFENCE'} · {side === 'off' ? settings.offence
          ?? unavailable : settings.defence ?? unavailable}
          <span>{reading ? totalProjection(slots.filter(s => s.side === side && s.id).map(s => s.id!),
            values) : unavailable}</span></div>
        <div className="slots">{!reading ? <p className="empty">{unavailable}</p> : slots.filter(s =>
          s.side === side).map((slot, i) => slot.id ? row(slot.id, slot)
          : <HQAct key={`empty:${i}`} className={`slot gap ${picked ? 'target' : ''}`} attributes={{
            'data-hq-slot': '' }} style={{ width: '100%', textAlign: 'left' }}
            onAction={() => { if (picked && canEdit) { place?.(picked); setPicked(undefined); } }}>
            <span className="sl">{slot.code}</span><span /><span className="nm">Empty</span><span /><span
              /><span className="pr">—</span><span className="st warn">OPEN</span></HQAct>)}</div>
      </div>)}
      <HQMatchup now={now} values={values}
        ownProjection={reading ? totalProjection(starters.map(p => p.id), values) : unavailable} />
    </div>
    <div className="col bench" data-hq-bench=""><div
      className="ch"><b>BENCH · <span>{reading ? benchPlayers.length : unavailable}</span></b>
      <span>{'drag a tag into the lineup'}</span></div>
      <div className="filters">{['ALL', ...positions].map(pos => <HQAct key={pos} className="chip"
        active={filter === pos} onAction={() => setFilter(pos)}>{pos}</HQAct>)}</div>
      <div className="blist">{benchPlayers.filter(p => filter === 'ALL' || p.position === filter).map(p =>
        row(p.id))}
        {!benchPlayers.length && <p className="empty">{unavailable}</p>}</div><InlineReserves snapshot={snapshot} mine={mine} settings={settings}
          injuries={injuries} select={select} />
    </div>
    {drag && <span className="pg ghost" style={{ transform: `translate(${drag.x}px,${drag.y}px)`
      }}>{directory.get(drag.id)?.position ?? '—'}</span>}
  </section>;
}

export function HQMatchup({ now, values = new Map(), ownProjection = unavailable }: {
  now?: Now; values?: Map<string, number>; ownProjection?: string;
}) {
  const report = now?.matchup;
  const opponent = report?.summary.find(r => r.cells[0] === 'Opponent')?.cells[1];
  const previews = now?.board.reports.find(r => r.id === 'M-018');
  const preview = previews?.rows.find(r => r.mine);
  const complete = previews?.summary.find(r => r.cells[0] === 'Starters MFL does not project')?.cells[1]
    === '0' && ['ok', 'stale'].includes(previews.status);
  const other = preview?.cells[0] === report?.columns[1] ? preview?.cells[3] : preview?.cells[1];
  const opponentProjection = complete && other && /^-?\d+(?:\.\d+)?$/.test(other) ? other : unavailable;
  const projected = (name: string) => {
    const identities = report?.entities?.filter(e => e.kind === 'player' && name.includes(e.text)) ?? [];
    return identities.length === 1 ? values.get(identities[0].id) : undefined;
  };
  const rows = report && ['ok', 'stale'].includes(report.status) ? report.rows.map(r => {
    const mine = projected(r.cells[1]), theirs = projected(r.cells[3]);
    return { row: r, mine, theirs, delta: mine === undefined || theirs === undefined ? undefined : mine -
      theirs };
  }).sort((a, b) => Math.abs(a.delta ?? Infinity) - Math.abs(b.delta ?? Infinity)) : [];
  return <div className="mu"><div className="ch"><b>THIS WEEK{opponent ? ` · VS ${opponent.toUpperCase()}`
    : ''}</b>
    <span>closest matchups first</span></div>
    <div className="mu-h"><div className="side"><b data-empty={ownProjection ===
      unavailable}>{ownProjection}</b>
      <span>you · proj</span></div><div className="wp">{unavailable}</div>
      <div className="side them"><b data-empty={opponentProjection === unavailable}>{opponentProjection}</b>
        <span>{opponent ?? unavailable} · proj</span></div></div>
    <div className="mu-rows">{rows.length ? rows.map(({ row, mine, theirs, delta }, i) => <div key={i}>
      <span className="sl">{row.cells[0]}</span><span>{row.cells[1]} {mine?.toFixed(1)} vs {row.cells[3]}
        {theirs?.toFixed(1)}</span>
      <span className="n" title={delta === undefined ? unavailable : undefined}>
        {delta === undefined ? '—' : `${delta > 0 ? '+' : ''}${delta.toFixed(1)}`}</span>
    </div>) : <p className="empty">{unavailable}</p>}</div>
  </div>;
}

export function InlineReserves({ snapshot, mine, settings, injuries, select }: {
  snapshot: Snapshot; mine: string; settings: HQSettings; injuries?: Injury[];
  select: (id: string) => void;
}) {
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  return <div className="inl">{(['taxi', 'ir'] as const).map(kind => {
    const players = roster.filter(p => p.rosterStatus === (kind === 'ir' ? 'IR' : 'TAXI_SQUAD'));
    return <div key={kind}><div className="sec">{kind === 'ir' ? 'INJURED RESERVE' : 'TAXI SQUAD'}
      {' · '}{players.length} of {settings[kind] ?? unavailable}<span>also in the pull tab</span></div>
      {players.map(p => {
        const player = snapshot.players.value.find(v => v.id === p.id);
        const injury = injuries?.find(i => i.id === p.id);
        const back = injury?.returnWeek && injury.returnSeason === snapshot.league.value.season
          ? `Wk ${injury.returnWeek}` : injury?.returnSeason &&
            injury.returnSeason > snapshot.league.value.season ? 'Season' : unavailable;
        return <HQAct key={p.id} className="slot" onAction={() => select(p.id)}>
          <span className="pg">{player?.position ?? '—'}</span>
          <span className="nm">{player?.name ?? p.id}</span>
          <span className="elig" title={injury?.returnReason}>{kind === 'ir' ? back :
            player?.isRookie ? 'yr 1' : unavailable}</span><span />
        </HQAct>;
      })}</div>;
  })}</div>;
}
