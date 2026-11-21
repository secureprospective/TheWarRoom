import {
  useEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode,
} from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { readNativeHQ, type NativeHQ } from '../commissioner/hqReading';
import type { HostedPort } from '../commissioner/hostedRoom';
import { useContractRead } from '../contracts/Contracts';
import { FailureAlert } from '../data/FailureAlert';
import { parseDraft } from '../draftroom/draft';
import { parseHostedGames, type Game } from '../now/liveContract';
import { HQAct } from '../shell/HQAct';
import { useNativeLineup, NativeLineupReview } from '../shell/hqNativeLineup';
import { quickMove } from '../shell/hqPlacement';
import { addHQTrade, hqEvents } from '../shell/hqReads';
import { absent, unavailable } from '../shell/hqModel';
import { usePlayerCard } from '../shell/PlayerCard';
import { fixedPoints } from '../shell/playerCardParts';
import {
  benchStarter, canMoveTo, moveStarters, myPicks, teamView, type TeamPick, type TeamRow,
} from './teamModel';
import { parseHostedRoster } from '../commissioner/hostedReadings';
import { ordinal } from '../shell/hqPlayerData';
import { readPlayerStatus, statusWords, type PlayerStatus } from './playerStatus';
import { createRowDrag, rowKeyAtPoint, type DragState } from './rowDrag';
import './TeamTab.css';

const fixed = fixedPoints;
/** placeWords is the mock's "11th of 12"; a division standing or the unavailable words stand in. */
export const placeWords = (reading: Pick<NativeHQ, 'place' | 'standing'>) =>
  reading.place ? `${ordinal(reading.place.rank)} of ${reading.place.of}` : reading.standing || unavailable;

/** The phone Team tab: lineup, bench and IR from the hosted HQ reads; moves go through the lineup command. */
export function TeamTab({ league, mine, port = hostedWorkflow, onByes }: {
  league: string; mine: string; port?: HostedPort; onByes?: (count: number) => void;
}) {
  const [revision, refresh] = useState(0);
  const provider = useMemo(() => ({ onChange: hqEvents.onChange }), []);
  const { reading, error, pending } = useContractRead<NativeHQ>(JSON.stringify([league, mine, revision]),
    publish => readNativeHQ(league, mine, port, publish, true, true, false), provider);
  const [games, setGames] = useState<Game[]>();
  useEffect(() => {
    let live = true;
    port({ command: 'nfl.games.read', mode: 'read', arguments: { league } })
      .then(r => { if (live) setGames(parseHostedGames(r.data, league).games); })
      .catch(() => { if (live) setGames(undefined); });
    return () => { live = false; };
  }, [league, port, revision]);
  const [openBench, setOpenBench] = useState<number>();
  const [status, setStatus] = useState<PlayerStatus>();
  useEffect(() => {
    let live = true;
    setOpenBench(undefined);
    port({ command: 'native.assets.read', mode: 'read', arguments: { league } })
      .then(r => { if (live) setOpenBench(parseHostedRoster(r.data).openBenchSlots?.[mine]); })
      .catch(() => { if (live) setOpenBench(undefined); });
    readPlayerStatus(league, port).then(s => { if (live) setStatus(s); })
      .catch(() => { if (live) setStatus(undefined); });
    return () => { live = false; };
  }, [league, mine, port, revision]);
  const editor = useNativeLineup(league, mine, reading?.lineup, port, true);
  const [moving, setMoving] = useState<string>();
  const [drag, setDrag] = useState<DragState>();
  const [sheet, setSheet] = useState<{ kind: 'card'; row: TeamRow } | { kind: 'picks' }>();
  const tabRef = useRef<HTMLDivElement>(null);
  const dragCtrl = useMemo(() => createRowDrag(), []);
  useEffect(() => () => dragCtrl.cancel(), [dragCtrl]);
  useEffect(() => {
    if (!editor.canEdit) { dragCtrl.cancel(); setDrag(undefined); }
  }, [editor.canEdit, dragCtrl]);
  const view = reading && teamView(reading, editor.edited, mine, games);
  const byes = view?.starters.filter(r => r.state === 'bye').length ?? 0;
  useEffect(() => { onByes?.(byes); }, [byes, onByes]);
  if (!reading || !view) return <div className="team-tab">
    {error ? <FailureAlert
      message={error}
      prefix={`${unavailable} · `}
      className="team-tab__note"
      league={league}
      screen="Phone · Team"
      route="phone/team"
      port={port}
    /> : <p className="team-tab__note" role="status">{absent(pending)}</p>}
  </div>;
  const all = [...view.starters, ...view.bench, ...view.ir];
  const from = all.find(r => r.key === (drag?.key ?? moving));
  const team = reading.snapshot.franchises.value.find(f => f.id === mine)?.name ?? unavailable;
  const editable = editor.canEdit;
  const tap = (to: TeamRow) => {
    if (!from || !editor.edited) return;
    if (to.key === from.key) return setMoving(undefined);
    if (!canMoveTo(from, to, reading)) return;
    editor.assign(moveStarters(editor.edited, from, to));
    setMoving(undefined);
  };
  const quickAct = (r: TeamRow) => {
    if (!editable || !editor.edited || !r.id || r.section === 'ir' || r.locked) return;
    const next = quickMove(r.id, view.starters, slot => canMoveTo(r, slot, reading));
    if (next.kind === 'bench') editor.assign(benchStarter(editor.edited, r.id));
    else if (next.kind === 'fill') editor.assign(moveStarters(editor.edited, r, next.slot));
    else setMoving(r.key);
  };
  const startDrag = (r: TeamRow, e: PointerEvent) => {
    if (!editable || from || !r.id || r.section === 'ir' || r.locked) return;
    if (typeof e.button === 'number' && e.button !== 0) return;
    dragCtrl.start(r.key, e.pointerId, e.clientX, e.clientY, active => {
      try { tabRef.current?.setPointerCapture?.(active.pointerId); } catch { /* noop */ }
      setDrag(active);
    });
  };
  const finishDrag = (e: PointerEvent) => {
    const result = dragCtrl.finish(e.pointerId, e.clientX, e.clientY, rowKeyAtPoint);
    if (!result) return;
    setDrag(undefined);
    const source = all.find(r => r.key === result.from);
    const target = result.to ? all.find(r => r.key === result.to) : undefined;
    if (source && target && editor.edited && canMoveTo(source, target, reading)) {
      editor.assign(moveStarters(editor.edited, source, target));
    }
  };
  const row = (r: TeamRow) => {
    const can = from && canMoveTo(from, r, reading);
    const over = Boolean(drag && can && drag.over === r.key);
    const cls = [
      'team-row', r.id ? '' : 'empty', from?.key === r.key ? 'pick' : '',
      can ? 'can' : '', over ? 'over' : '',
    ].filter(Boolean).join(' ');
    const control = !r.id ? '—' : r.section === 'ir' ? 'IR' : r.locked ? 'LOCKED' : 'MOVE';
    const fact = !r.id || r.section === 'ir' || r.locked;
    const off = !editable || fact;
    return <div className={cls} key={r.key} data-row={r.key} onPointerDown={e => startDrag(r, e)}>
      <span className="team-row__slot">{r.slot}</span>
      {r.id ? <HQAct className="team-row__who" label={`${r.name} player card`}
        onAction={() => from ? tap(r) : setSheet({ kind: 'card', row: r })}>
        <b>{r.name}</b><em className={r.state}>{r.position !== r.slot && r.section === 'starters'
          ? `${r.position} · ` : ''}{r.words}{status?.facts.get(r.id)
          ? ` · ${statusWords(status.facts.get(r.id)!)}` : ''}</em></HQAct>
        : <span className="team-row__who"><b>{r.name}</b><em>{can ? 'Tap to place'
          : r.section === 'bench' ? 'Add a player from Moves' : ''}</em></span>}
      <span className={`team-row__pt ${r.state}`} style={meter(r)}
        title={r.range ? `Last 6 games: low ${r.range[0]} · high ${r.range[1]}` : undefined}>
        <b>{fixed(r.points)}</b><em>{r.range ? r.range.map(x => x.toFixed(0)).join('–') : ''}</em></span>
      {from ? <HQAct className="team-row__mv" disabled={!can && from.key !== r.key}
        label={can ? `Place ${from.name} here` : from.key === r.key ? 'Cancel move' : `${r.slot} not legal`}
        onAction={() => tap(r)}>{from.key === r.key ? 'CANCEL' : can ? 'HERE' : '—'}</HQAct>
        : <HQAct className={`team-row__mv${fact ? ' team-row__mv--fact' : ''}`} disabled={off}
          label={r.locked ? 'Locked, game started'
            : r.section === 'ir' ? 'On injured reserve' : `Move ${r.name}`}
          onAction={() => quickAct(r)}>{control}</HQAct>}
    </div>;
  };
  return <div ref={tabRef} className={`team-tab${from ? ' moving' : ''}${drag ? ' dragging' : ''}`}
    onPointerMove={e => dragCtrl.move(e.pointerId, e.clientX, e.clientY, rowKeyAtPoint, setDrag)}
    onPointerUp={finishDrag}
    onPointerCancel={() => { dragCtrl.cancel(); setDrag(undefined); }}
    onClickCapture={e => { if (dragCtrl.consumeClick()) { e.stopPropagation(); e.preventDefault(); } }}
    onContextMenu={e => { if (drag) e.preventDefault(); }}>
    <div className="team-head">
      <h2>{team} <HQAct className="team-key team-head__picks" onAction={() => setSheet({ kind: 'picks' })}>
        My picks</HQAct></h2>
      <span className="team-head__m">{reading.record ?? unavailable} · {placeWords(reading)}
        {reading.week ? ` · week ${reading.week}` : ''}</span>
      <span className="team-head__pts"><b>{fixed(view.points)}</b>
        <em>PROJ {fixed(view.projection)}</em></span>
    </div>
    {reading.weekNote && <p className="team-tab__note" role="status">{reading.weekNote}</p>}
    {from ? <div className="team-movebar" role="status">
      <span>{drag ? <>Dragging <b>{from.name}</b>. Drop on a lit row.</>
        : <>Moving <b>{from.name}</b>. Tap a lit row.</>}</span>
      {!drag && <HQAct className="team-key" onAction={() => setMoving(undefined)}>Cancel</HQAct>}</div>
      : view.bye && <div className="team-warn" role="alert">
        <span><b>STARTER ON BYE</b><br />{view.bye.name} scores 0 this week.</span>
        <HQAct className="team-key" disabled={!editable}
          onAction={() => setMoving(view.bye!.key)}>Fix</HQAct></div>}
    {editor.dirty && !from && <div className="team-save" role="status">
      <span>{editor.message}</span>
      <HQAct className="team-key" onAction={editor.discard}>Discard</HQAct>
      <HQAct className="team-key team-key--go" disabled={!editable || !editor.changed.length}
        onAction={() => { void editor.save(); }}>Check and save</HQAct></div>}
    {reading.errors.length > 0 && <FailureAlert
      message={reading.errors[0]}
      prefix="Some reads are unavailable: "
      className="team-tab__note"
      league={league}
      screen="Phone · Team"
      route="phone/team"
      port={port}
    />}
    <Group title="Starters" aside={`${view.locked} locked · games started`}>{view.starters.map(row)}</Group>
    <Group title="Bench" aside={openBench === undefined ? `${view.bench.length} players`
      : `${view.bench.length} of ${view.bench.length + openBench}`}>
      {view.bench.length || openBench ? [...view.bench, ...emptyBench(view.bench.length, openBench)].map(row)
        : <p className="team-tab__note">No bench players.</p>}</Group>
    <Group title="Injured reserve" aside={view.irOutsideBench
      ? "doesn't count toward the bench" : 'counts toward the roster'}>
      {view.ir.length ? view.ir.map(row)
        : <p className="team-tab__note">No players on injured reserve.</p>}</Group>
    {drag && from && <span className="team-drag-pill" style={{ left: drag.x + 8, top: drag.y - 18 }}>
      {from.position} · {from.name}</span>}
    <NativeLineupReview editor={editor} refresh={() => refresh(n => n + 1)} />
    {sheet?.kind === 'picks' && <TeamSheet title="My draft picks" onClose={() => setSheet(undefined)}>
      <PicksList league={league} mine={mine} port={port} /></TeamSheet>}
    {sheet?.kind === 'card' && <TeamSheet title="Player" onClose={() => setSheet(undefined)}>
      <PlayerSheet row={sheet.row} reading={reading} league={league} mine={mine} port={port} status={status}
        canMove={editable && sheet.row.section !== 'ir' && !sheet.row.locked}
        onMove={() => { setSheet(undefined); setMoving(sheet.row.key); }}
        onTrade={() => { setSheet(undefined);
          void addHQTrade(mine, sheet.row.id!, reading.snapshot, league).catch(() => {}); }} />
    </TeamSheet>}
  </div>;
}

/** meter feeds the row's 2px range bar: the lit band is his low–high, the fill is this week so far. */
function meter(r: TeamRow): CSSProperties | undefined {
  if (!r.range) return undefined;
  const high = r.range[1] || 1;
  return { '--pp': Math.min(1, Math.max(0, (r.points ?? 0) / high)).toFixed(3),
    '--lo': Math.max(0, r.range[0] / high).toFixed(3) } as CSSProperties;
}

/** emptyBench is the mock's open bench row. A league with many open spots gets one row that counts
 * them, so a 45-spot bench does not push the lineup off the screen. */
export function emptyBench(filled: number, open = 0): TeamRow[] {
  if (open <= 0) return [];
  return [{ key: 'bench-open', section: 'bench', slot: 'BN', index: filled,
    name: open > 1 ? `${open} empty bench spots` : 'Empty bench spot', position: '', state: '', words: '',
    locked: false }];
}

function Group({ title, aside, children }: { title: string; aside: string; children: ReactNode }) {
  return <section className="team-grp" aria-label={title}>
    <h3>{title} <span>{aside}</span></h3>{children}</section>;
}

function PicksList({ league, mine, port }: { league: string; mine: string; port: HostedPort }) {
  const [picks, setPicks] = useState<TeamPick[] | string>();
  useEffect(() => {
    let live = true;
    port({ command: 'draft.read', mode: 'read', arguments: { league, franchise: mine, season: 0,
      at: '0001-01-01T00:00:00Z' } })
      .then(r => { const draft = parseDraft(r.data);
        if (draft.league !== league) throw new Error('Draft belongs to another league');
        if (live) setPicks(myPicks(draft, mine)); })
      .catch(e => { if (live) setPicks(e instanceof Error ? e.message : unavailable); });
    return () => { live = false; };
  }, [league, mine, port]);
  return <>
    <section className="team-grp">
      {picks === undefined ? <p className="team-tab__note" role="status">Still reading…</p>
        : typeof picks === 'string' ? <p className="team-tab__note" role="status"
          title={picks}>{unavailable}</p>
          : picks.length === 0 ? <p className="team-tab__note">You hold no unused picks in this draft.</p>
            : picks.map(p => <div className="team-pick" key={p.key}>
              <b>{p.label}</b><span>{p.from}</span></div>)}
    </section>
    <p className="team-tab__note">Trade picks from Moves › Trades.</p>
  </>;
}

function PlayerSheet({ row, reading, league, mine, port, status, canMove, onMove, onTrade }: {
  row: TeamRow; reading: NativeHQ; league: string; mine: string; port: HostedPort; status?: PlayerStatus;
  canMove: boolean; onMove: () => void; onTrade: () => void;
}) {
  const card = usePlayerCard({
    id: row.id!, reading, league, mine, port, status, fetchStatus: false, statusEmpty: unavailable,
    noteInputId: 'team-note',
  });
  return <div className="team-card">
    <div className="team-card__title"><b>{card.title}</b><span>{card.sub}</span></div>
    <div className="team-card__tabs">{card.tabs}</div>
    {card.main}
    {card.side}
    <div className="team-card__a">
      <HQAct className="team-key" disabled={!canMove} onAction={onMove}>
        {row.locked ? 'Locked' : 'Move'}</HQAct>
      <HQAct className="team-key" onAction={onTrade}>Trade</HQAct>
      <HQAct className="team-key" disabled label="Drop is not on the phone yet">Drop</HQAct>
    </div>
  </div>;
}

function TeamSheet({ title, onClose, children }: {
  title: string; onClose: () => void; children: ReactNode;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal?.();
    return () => element?.close?.();
  }, []);
  return <dialog ref={dialog} className="phone-app__sheet team-sheet" aria-modal="true" aria-label={title}
    onCancel={event => { event.preventDefault(); onClose(); }}
    onPointerUp={event => { if (event.target === event.currentTarget) onClose(); }}>
    <header className="phone-app__sheet-head"><h2>{title}</h2>
      <HQAct className="phone-app__close" label={`Close ${title}`} onAction={onClose}>Close</HQAct></header>
    <div className="phone-app__sheet-body">{children}</div>
  </dialog>;
}
