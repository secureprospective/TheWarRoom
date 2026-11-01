import type { NativeCut } from '../commissioner/hqReading';
import { HQAct } from './HQAct';
import { useMemo, useState } from 'react';
import type { Snapshot } from '../data/contract';
import { capacity, money, unavailable } from './hqModel';
import type { RosterFacts } from './hqRosterFacts';
import { researchFact } from './hqPlayerData';
import { useContractRead } from '../contracts/Contracts';
import { whatIfProvider, type WhatIf } from '../contracts/whatif';
import type { Reading } from '../contracts/contract';
import { releaseAction } from '../contracts/releases';
import { HoldToFire } from '../commissioner/HoldToFire';
import { useCachedLineup } from './lineups';
import { commands } from '../commands/registry';

export function HQRelease({ id, mine, snapshot, activeLimit, facts, close, native = false, cut, starter,
  beforeRoom }: {
  id: string; mine: string; snapshot: Snapshot; activeLimit?: number; facts?: RosterFacts;
    close: () => void; native?: boolean; cut?: NativeCut; starter?: boolean; beforeRoom?: number;
}) {
  const refusal = commands.useTeamActionReason('lineup.edit', { franchiseId: mine });
  const provider = useMemo(whatIfProvider, []);
  const { reading, error } = useContractRead<Reading<WhatIf> | undefined>(`${native}:${mine}`,
    () => native ? Promise.resolve(undefined) : provider.query(mine), provider);
  const [confirm, setConfirm] = useState(false);
  const [message, setMessage] = useState(''); const [pending, setPending] = useState(false);
  const player = snapshot.players.value.find(p => p.id === id);
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const contract = roster.find(p => p.id === id);
  const active = roster.filter(p => p.rosterStatus === 'ROSTER').length;
  const move = reading?.value?.moves.find(m => m.player === id && m.tag.toLowerCase().includes('cut'));
  const hold = () => {
    if (!move || pending || refusal) return;
    setPending(true);
    void releaseAction({ mine, action: 'draft', move: move.id, confirmed: true }).then(
      () => setMessage('Release plan saved. Nothing submitted to MFL.'), e => setMessage(String(e)),
    ).finally(() => setPending(false));
  };
  const cached = useCachedLineup(commands.lineups, mine);
  const selectedStarter = starter ?? cached.reading?.starters.some(p => p.id === id);
  return <section className="sdrawer open" aria-label="Release plan"><div
    className="bd-head"><h2>Release plan · {player?.name ?? id}</h2><HQAct className="x"
    onAction={close}>ESC</HQAct></div>
    <div className="sd-body"><div className="sd-sec">THE COST<span>{contract?.contractStatus}</span></div>
      <div className="kv2">{[['Salary this year', money(contract?.salary)], ['Years left after',
        contract?.yearsRemaining === undefined ? unavailable : String(Math.max(0, contract.yearsRemaining -
        1))],
        ['Dead cap now', native ? money(cut?.deadNow) : researchFact(facts?.get(id)?.research,
          'Dead cap if cut today') ?? unavailable],
          ['Dead cap next yr', money(cut?.deadNext)],
        ['Room after', native ? money(cut?.roomAfter) : move && reading?.value?.moves.length === 1
          ? reading.value.figures.find(f => /room/i.test(f.label))?.after ?? unavailable : unavailable],
          ['Roster after', capacity(active - (contract?.rosterStatus === 'ROSTER' ? 1 : 0),
          activeLimit)]].map(([label, value]) => <div className="kv"
          key={label}><span>{label}</span><span>{value}</span></div>)}</div>
      <div className="sd-sec">CAP ROOM, BEFORE ┆ AFTER<span>dashed = today</span></div>
      {native ? <div className="rb"><span>{snapshot.league.value.season}</span>
        <div className="bar" /><b>{money(beforeRoom)} ┆ {money(cut?.roomAfter)}</b></div> :
      reading?.value?.figures.length && move && reading.value.moves.length === 1 ?
        reading.value.figures.map(f => <div className="rb"
        key={f.label}><span>{f.label}</span><div className="bar" /><b>{f.before} ┆ {f.after}</b></div>) :
        <p className="note">{unavailable}</p>}
      <div className="sd-sec">YOUR LINEUP<span>{selectedStarter === undefined ? unavailable :
        selectedStarter ? 'starter' : 'bench'}</span></div>
      <div className="alt" style={{ border: 0 }}><b>{selectedStarter === undefined ? unavailable :
        selectedStarter ? 'Slot goes empty' : 'Lineup unchanged'}</b><small>{unavailable}</small></div>
      <div className="sd-sec">BEFORE YOU CUT<span>other ways out</span></div><p
        className="note">{unavailable}</p>
      {(message || error) && <p className="note" role="status">{message || unavailable}</p>}
    </div><div className="sd-act"><HQAct className="btn" disabled={native || !reading?.value ||
      reading.value.stale
      || Boolean(move)} onAction={() => { void commands.dispatch('contracts.set', { field: 'whatif', value:
      { mine, action: 'cut', player: id } }); }}>TRY IN WHAT-IF</HQAct>
      <HQAct className="btn g2" disabled={native || !move || pending} onAction={() => setConfirm(true)}
        style={{
        marginLeft: 'auto' }}>PREPARE RELEASE</HQAct></div>
    <div className={`nested ${confirm ? 'open' : ''}`} role="dialog" aria-label="Confirm release"
      aria-hidden={!confirm}>
      <div className="g3">IRREVERSIBLE</div><h3>Release {player?.name ?? id}</h3><p
        className="note">Hold the bar for about half a second. Let go early to cancel.</p>
      <HoldToFire appearance="hq" label="HOLD TO RELEASE" body={move?.id ?? ''}
        disabled={pending || !move || Boolean(refusal)} onConfirm={hold} />
      <HQAct className="btn" onAction={() => { setConfirm(false); }} style={{ marginTop: 10 }}>BACK</HQAct>
    </div></section>;
}
