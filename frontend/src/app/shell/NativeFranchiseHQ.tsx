import { pointsLabel, type PulseResults } from '../data/nativePulse';
import { RailLeaguePicker } from './leaguePicker';
import type { Node } from './nodes';
import { useNativeLineup, NativeLineupReview } from './hqNativeLineup';
import { HQAct } from './HQAct';
import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { readNativeHQ, readNativeCut, type NativeHQ } from '../commissioner/hqReading';
import type { HostedPort } from '../commissioner/hostedRoom';
import { useContractRead } from '../contracts/Contracts';
import { hqEvents } from './hqReads';
import { HQShell, type HQTab } from './HQShell';
import { HQHeader } from './HQHeader';
import { HQBoard } from './HQBoard';
import { HQContracts, HQReserves } from './HQPanels';
import { HQMoves, HQQueue } from './HQMoves';
import { HQRelease } from './HQRelease';
import { HQInspector } from './HQInspector';
import { absent, unavailable } from './hqModel';
import { HQNativeWhatIf } from './HQNativeWhatIf';
import { countdown } from '../clock/urgency';
import { connectClock } from '../clock/ticker';
import './hq.css';

export default function NativeFranchiseHQ({ league, mine, name, port, leave, embedded,
  tab: activeTab, navigate, canManage = true }: {
  league: string; mine: string; name: string; port?: HostedPort; leave: (node?: Node) => void;
  embedded?: boolean; canManage?: boolean; tab?: HQTab; navigate?: (tab: HQTab) => void;
}) {
  const root = useRef<HTMLDivElement>(null);
  const [localTab, setLocalTab] = useState<HQTab>('lineup-and-roster');
  const tab = activeTab ?? localTab;
  const setTab = navigate ?? setLocalTab;
  const [drawer, setDrawer] = useState<'ir' | 'taxi' | 'whatif'>();
  const [release, setRelease] = useState<string>();
  const [selected, select] = useState<string>();
  const [revision, refresh] = useState(0);
  const provider = useMemo(() => ({ onChange: hqEvents.onChange }), [port]);
  const { reading, error, pending } = useContractRead<NativeHQ>(
    JSON.stringify([league, mine, revision]),
    publish => readNativeHQ(league, mine, port, publish), provider);
  const editor = useNativeLineup(league, mine, reading?.lineup, port, canManage);
  const chosen = selected ?? reading?.lineup?.starters[0]?.id;
  const quote = useContractRead(JSON.stringify([league, mine, chosen]), () => chosen
    ? readNativeCut(league, mine, chosen, port) : Promise.resolve(undefined), provider);
  useEffect(() => {
    if (!root.current || !reading?.lock) return;
    return connectClock(root.current, [{ id: 'native-lineup', label: 'Lineup', at: reading.lock, urgency:
      'U1', pinned: false, promoted: false }]);
  }, [reading?.lock]);
  useEffect(() => { const key = (e: KeyboardEvent) => { if (e.key === 'Escape') { setDrawer(undefined);
    setRelease(undefined); } };
    document.addEventListener('keydown', key); return () => document.removeEventListener('keydown', key);
      }, []);
  if (!reading) return <div className="hq-v10">{!embedded && <RailLeaguePicker />}
    <p className="empty" role="status"
    title={error}>{absent(pending)}</p></div>;
  const { snapshot, lineup, runway, moves = [] } = reading;
  const settings = { ...reading.settings };
  const positions = new Map(lineup?.starters.concat(lineup.bench).map(p => [p.id, p.position ||
    snapshot.players.value.find(v => v.id === p.id)?.position || '—']));
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  if (settings.reserveExcluded === false && settings.active !== undefined)
    settings.active -= roster.filter(p => p.rosterStatus !== 'ROSTER').length;
  const cut = quote.reading?.revision === reading.revision ? quote.reading : undefined;
  const view = <HQShell fullscreen={!embedded} embedded={embedded} root={root} tab={tab} navigate={setTab}
    week={reading.week}
    leagueName={reading.leagueName}
    navigateNode={node => { if (node !== 'hq') leave(node); }} lock={reading.lock ? { at: reading.lock } :
      undefined}
    queue={<HQQueue moves={moves} available={reading.moves !== undefined} />}
    inspector={<HQInspector native snapshot={snapshot} mine={mine} id={chosen} cut={cut}
      valuesPreparing={reading.values?.preparing}
      leagueValue={reading.values?.players.find(p => p.player === chosen)}
      projection={reading.projections?.get(chosen ?? '')}
      annual={reading.contracts?.get(chosen ?? '')}
      position={lineup?.starters.concat(lineup.bench).find(p => p.id === chosen)?.position ||
        snapshot.players.value.find(v => v.id === chosen)?.position}
      bench={editor.canEdit && chosen && editor.edited?.starters.some(p => p.id === chosen) ?
        id => editor.assign(editor.edited!.starters.filter(p => p.id !== id).map(p => p.id)) : undefined}
      toIR={editor.canEdit && roster.some(p => p.id === chosen && p.rosterStatus === 'ROSTER') ?
        id => { void editor.reserve(id); } : undefined} />}>
    {!mine && <p role="status">No franchise is assigned to you. This view is read-only.</p>}
    <div>{reading.weekNote && <p role="status" className="empty">{reading.weekNote}</p>}
    <HQHeader name={snapshot.franchises.value.find(f => f.id === mine)?.name ?? name}
      active={roster.filter(p => p.rosterStatus === 'ROSTER').length} settings={settings}
      cap={runway?.[0] ?? { season: snapshot.league.value.season }} check={reading.check ?? unavailable}
      record={reading.record} standing={reading.standing} standingsNote={reading.standingsNote}
      lock={reading.lock ? { id: 'native-lineup', at:
        reading.lock, text: countdown(reading.lock, Date.now()) } : undefined} /></div>
    {tab === 'lineup-and-roster' ? <HQBoard showContracts snapshot={snapshot} mine={mine} key={editor.version}
      reading={editor.edited} matchup={<NativeHQMatchup results={reading.thisWeek} mine={mine} />}
      error={reading.errors.join(' · ')}
      settings={settings} values={reading.projections ?? new Map()} select={select} canEdit={editor.canEdit}
      changed={editor.changed} assign={editor.assign} />
      : tab === 'contracts-and-cap' ? <HQContracts native snapshot={snapshot} mine={mine} runway={runway}
        positions={positions} annual={reading.contracts} cutCosts={cut?.deadNow === undefined ? undefined :
          new Map([[cut.player,
          cut.deadNow]])}
        select={id => { select(id); setRelease(id); }} />
        : <HQMoves native snapshot={snapshot} moves={moves} available={reading.moves !== undefined}
          reason={reading.moveReason} />}
    <div className="tray"><div className="pulls" style={{ visibility: tab === 'lineup-and-roster' ?
      'visible' : 'hidden' }}>
      {(['taxi', 'ir'] as const).map(kind => <HQAct key={kind} className="pull" expanded={drawer === kind}
        onAction={() => setDrawer(drawer === kind ? undefined : kind)}>
        {kind.toUpperCase()} <b>{roster.filter(p => p.rosterStatus === (kind === 'ir' ? 'IR' :
          'TAXI_SQUAD')).length} / {settings[kind] ?? unavailable}</b></HQAct>)}
      <HQAct className="pull" expanded={drawer === 'whatif'}
        onAction={() => setDrawer(drawer === 'whatif' ? undefined : 'whatif')}>WHAT-IF</HQAct></div>
      <span className="msg" title={[...reading.errors, quote.error ??
        ''].join(' · ')}>{editor.message}</span><HQAct
        className="btn sp" disabled={!editor.dirty || editor.state.pending}
          onAction={editor.discard}>DISCARD</HQAct><HQAct className="btn g2"
        disabled={!editor.canEdit || !editor.changed.length} onAction={() => { void editor.save(); }}
        >CHECK AND SAVE PLAN</HQAct>
      </div>
    <NativeLineupReview editor={editor} refresh={() => refresh(n => n + 1)} />
    {drawer === 'whatif' && <HQNativeWhatIf snapshot={snapshot} mine={mine} chosen={chosen}
      select={select} cut={cut} beforeRoom={runway?.[0]?.room} close={() => setDrawer(undefined)} />}
    {(drawer === 'ir' || drawer === 'taxi') && <HQReserves native kind={drawer} snapshot={snapshot}
      mine={mine} settings={settings}
      countedIRPay={runway?.[0]?.ir} reserveNotes={reading.reserveNotes}
      activate={editor.canEdit ? id => { setDrawer(undefined); void editor.reserve(id, 'active'); } :
        undefined}
      week={reading.week} close={() => setDrawer(undefined)} />}
    {release && <HQRelease native id={release} mine={mine} snapshot={snapshot} activeLimit={settings.active}
      cut={cut?.player === release ? cut : undefined} beforeRoom={runway?.[0]?.room}
        starter={lineup?.starters.some(p => p.id === release)}
      close={() => setRelease(undefined)} />}
  </HQShell>;
  return embedded || typeof document === 'undefined' ? view : createPortal(view, document.body);
}

function NativeHQMatchup({ results, mine }: { results?: PulseResults; mine: string }) {
  const game = results?.games.find(g => g.home === mine || g.away === mine);
  const own = results?.scores.find(s => s.franchise === mine);
  const other = game && results?.scores.find(s => s.franchise === (game.home === mine ? game.away : game.home));
  const ownPoints = own?.points ?? own?.provisional;
  const otherPoints = other?.points ?? other?.provisional;
  return <div className="mu"><div className="ch"><b>THIS WEEK{results ? ` · WK ${results.week}` : ''}</b>
    <span>{game?.status ?? unavailable}</span></div>
    <div className="mu-h"><div className="side"><b>{pointsLabel(ownPoints)}</b>
      <span>{own?.name ?? unavailable}</span></div>
      <div className="wp">{game?.words ?? results?.words ?? unavailable}</div>
      <div className="side them"><b>{pointsLabel(otherPoints)}</b>
        <span>{other?.name ?? unavailable}</span></div></div>
    <div className="mu-rows">{own?.starters.length ? own.starters.map(p => <div key={p.player}>
      <span className="sl">{p.position ?? '—'}</span><span>{p.name}</span>
      <span className="n" title={p.words}>{pointsLabel(p.points ?? p.provisional)}</span>
    </div>) : <p className="empty">{results?.bye === mine ? results.byeName : unavailable}</p>}</div>
  </div>;
}
