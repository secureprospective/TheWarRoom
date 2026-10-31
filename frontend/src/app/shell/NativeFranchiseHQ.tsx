import { useNativeLineup, NativeLineupReview } from './hqNativeLineup';
import { HQAct } from './HQAct';
import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { readNativeHQ, readNativeCut } from '../commissioner/hqReading';
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
import { unavailable } from './hqModel';
import { countdown } from '../clock/urgency';
import { connectClock } from '../clock/ticker';
import './hq.css';

export default function NativeFranchiseHQ({ league, mine, name, port, leave }: {
  league: string; mine: string; name: string; port?: HostedPort; leave: () => void;
}) {
  const root = useRef<HTMLDivElement>(null);
  const [tab, setTab] = useState<HQTab>('lineup-and-roster');
  const [drawer, setDrawer] = useState<'ir' | 'taxi'>();
  const [release, setRelease] = useState<string>();
  const [selected, select] = useState<string>();
  const [revision, refresh] = useState(0);
  const provider = useMemo(() => ({ onChange: hqEvents.onChange }), [port]);
  const { reading, error } = useContractRead(JSON.stringify([league, mine, revision]), () => readNativeHQ(league,
    mine, port), provider);
  const editor = useNativeLineup(league, mine, reading?.lineup, port);
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
  if (!reading) return <div className="hq-v10"><p className="empty" role="status"
    title={error}>{unavailable}</p></div>;
  const { snapshot, lineup, runway, moves = [] } = reading;
  const settings = { ...reading.settings };
  const positions = new Map(lineup?.starters.concat(lineup.bench).map(p => [p.id, p.position ?? '—']));
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  if (settings.reserveExcluded === false && settings.active !== undefined)
    settings.active -= roster.filter(p => p.rosterStatus !== 'ROSTER').length;
  const cut = quote.reading?.revision === reading.revision ? quote.reading : undefined;
  const view = <HQShell fullscreen root={root} tab={tab} navigate={setTab} week={reading.week}
    leagueName={reading.leagueName}
    navigateNode={node => { if (node !== 'hq') leave(); }} lock={reading.lock ? { at: reading.lock } :
      undefined}
    queue={<HQQueue moves={moves} available={reading.moves !== undefined} />}
    inspector={<HQInspector native snapshot={snapshot} mine={mine} id={chosen} cut={cut}
      position={lineup?.starters.concat(lineup.bench).find(p => p.id === chosen)?.position} />}>
    <HQHeader name={snapshot.franchises.value.find(f => f.id === mine)?.name ?? name}
      active={roster.filter(p => p.rosterStatus === 'ROSTER').length} settings={settings}
      cap={runway?.[0] ?? { season: snapshot.league.value.season }} check={reading.check ?? unavailable}
      record={reading.record} standing={reading.standing} lock={reading.lock ? { id: 'native-lineup', at:
        reading.lock, text: countdown(reading.lock, Date.now()) } : undefined} />
    {tab === 'lineup-and-roster' ? <HQBoard snapshot={snapshot} mine={mine} key={editor.version} reading={editor.edited}
      settings={settings} values={new Map()} select={select} canEdit={editor.canEdit}
      changed={editor.changed} assign={editor.assign} />
      : tab === 'contracts-and-cap' ? <HQContracts native snapshot={snapshot} mine={mine} runway={runway}
        positions={positions} cutCosts={cut?.deadNow === undefined ? undefined : new Map([[cut.player,
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
      <HQAct className="pull" disabled title="Native what-if read is not available">WHAT-IF</HQAct></div>
      <span className="msg" title={[...reading.errors, quote.error ??
        ''].join(' · ')}>{editor.message}</span><HQAct
        className="btn sp" disabled={!editor.dirty || editor.state.pending} onAction={editor.discard}>DISCARD</HQAct><HQAct className="btn g2"
        disabled={!editor.canEdit || !editor.changed.length} onAction={() => { void editor.save(); }}
        >CHECK AND SAVE PLAN</HQAct>
      <HQAct className="btn solid" disabled
        title="This league is native, not an MFL mirror">OPEN MFL ↗</HQAct></div>
    <NativeLineupReview editor={editor} refresh={() => refresh(n => n + 1)} />
    {drawer && <HQReserves native kind={drawer} snapshot={snapshot} mine={mine} settings={settings}
      countedIRPay={runway?.[0]?.ir}
      week={reading.week} close={() => setDrawer(undefined)} />}
    {release && <HQRelease native id={release} mine={mine} snapshot={snapshot} activeLimit={settings.active}
      cut={cut?.player === release ? cut : undefined} beforeRoom={runway?.[0]?.room}
        starter={lineup?.starters.some(p => p.id === release)}
      close={() => setRelease(undefined)} />}
  </HQShell>;
  return typeof document === 'undefined' ? view : createPortal(view, document.body);
}
