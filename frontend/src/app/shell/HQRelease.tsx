import { useEffect, useMemo, useRef, useState } from 'react';
import type { Snapshot } from '../data/contract';
import type { HQCap } from './hqModel';
import { capacity, money, unavailable } from './hqModel';
import type { RosterFacts } from './hqRosterFacts';
import { researchFact } from './hqPlayerData';
import { useContractRead } from '../contracts/Contracts';
import { whatIfProvider } from '../contracts/whatif';
import { releaseAction } from '../contracts/releases';
import { commands } from '../commands/registry';

export function HQRelease({ id, mine, snapshot, cap, activeLimit, facts, close }: {
  id: string; mine: string; snapshot: Snapshot; cap: HQCap; activeLimit?: number; facts?: RosterFacts; close: () => void;
}) {
  const provider = useMemo(whatIfProvider, []);
  const { reading, error } = useContractRead(mine, () => provider.query(mine), provider);
  const [confirm, setConfirm] = useState(false); const [holding, setHolding] = useState(false);
  const [message, setMessage] = useState(''); const [pending, setPending] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>();
  const player = snapshot.players.value.find(p => p.id === id);
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const contract = roster.find(p => p.id === id);
  const active = roster.filter(p => p.rosterStatus === 'ROSTER').length;
  const move = reading?.value?.moves.find(m => m.player === id && m.tag.toLowerCase().includes('cut'));
  const stop = () => { if (timer.current) clearTimeout(timer.current); timer.current = undefined; setHolding(false); };
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);
  const hold = () => {
    if (!move || pending || timer.current) return;
    setHolding(true);
    timer.current = setTimeout(() => {
      timer.current = undefined; setHolding(false); setPending(true);
      void releaseAction({ mine, action: 'draft', move: move.id, confirmed: true }).then(() => setMessage('Release plan saved. Nothing submitted to MFL.'), e => setMessage(String(e))).finally(() => setPending(false));
    }, 600);
  };
  const selectedStarter = commands.lineups.peek(mine).reading?.starters.some(p => p.id === id);
  return <section className="sdrawer open" aria-label="Release plan"><div className="bd-head"><h2>Release plan · {player?.name ?? id}</h2><button className="x" onClick={close}>ESC</button></div>
    <div className="sd-body"><div className="sd-sec">THE COST<span>{contract?.contractStatus}</span></div>
      <div className="kv2">{[['Salary this year', money(contract?.salary)], ['Years left after', contract?.yearsRemaining === undefined ? unavailable : String(Math.max(0, contract.yearsRemaining - 1))],
        ['Dead cap now', researchFact(facts?.get(id)?.research, 'Dead cap if cut today') ?? unavailable], ['Dead cap next yr', unavailable],
        ['Room after', reading?.value?.figures.find(f => /room/i.test(f.label))?.after ?? unavailable], ['Roster after', capacity(active - (contract?.rosterStatus === 'ROSTER' ? 1 : 0), activeLimit)]].map(([label, value]) => <div className="kv" key={label}><span>{label}</span><span>{value}</span></div>)}</div>
      <div className="sd-sec">CAP ROOM, BEFORE ┆ AFTER<span>dashed = today</span></div>
      {reading?.value?.figures.length && move ? reading.value.figures.map(f => <div className="rb" key={f.label}><span>{f.label}</span><div className="bar" /><b>{f.before} ┆ {f.after}</b></div>) : <p className="note">{unavailable}</p>}
      <div className="sd-sec">YOUR LINEUP<span>{selectedStarter === undefined ? unavailable : selectedStarter ? 'starter' : 'bench'}</span></div>
      <div className="alt" style={{ border: 0 }}><b>{selectedStarter === undefined ? unavailable : selectedStarter ? 'Slot goes empty' : 'Lineup unchanged'}</b><small>{unavailable}</small></div>
      <div className="sd-sec">BEFORE YOU CUT<span>other ways out</span></div><p className="note">{unavailable}</p>
      {(message || error) && <p className="note" role="status">{message || error}</p>}
    </div><div className="sd-act"><button className="btn" disabled={!reading?.value || reading.value.stale || Boolean(move)} onClick={() => { void commands.dispatch('contracts.set', { field: 'whatif', value: { mine, action: 'cut', player: id } }); }}>TRY IN WHAT-IF</button>
      <button className="btn g2" disabled={!move || pending} onClick={() => setConfirm(true)} style={{ marginLeft: 'auto' }}>PREPARE RELEASE</button></div>
    <div className={`nested ${confirm ? 'open' : ''}`} role="dialog" aria-label="Confirm release" aria-hidden={!confirm}>
      <div className="g3">IRREVERSIBLE</div><h3>Release {player?.name ?? id}</h3><p className="note">Hold the bar for about half a second. Let go early to cancel.</p>
      <button className={`hold ${holding ? 'go' : ''}`} disabled={pending || !move} onPointerDown={hold} onPointerUp={stop} onPointerCancel={stop} onPointerLeave={stop}
        onKeyDown={e => { if ((e.key === ' ' || e.key === 'Enter') && !e.repeat) { e.preventDefault(); hold(); } }} onKeyUp={stop} onBlur={stop}><span className="fill" /><span>HOLD TO RELEASE</span></button>
      <button className="btn" onClick={() => { stop(); setConfirm(false); }} style={{ marginTop: 10 }}>BACK</button>
    </div></section>;
}
