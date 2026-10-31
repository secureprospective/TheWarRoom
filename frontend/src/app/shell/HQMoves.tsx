import type { Snapshot } from '../data/contract';
import { intentLabels, stateLabels, type RailState } from './moveLabels';
import { unavailable } from './hqModel';
export type HQMove = { id: string; intent: string; state: RailState; note: string; at: string;
  players: string[]; native?: boolean; handoff?: () => void };
export function moveLane(state: RailState): number {
  if (['draft', 'ready'].includes(state)) return 0;
  if (['handed_off', 'not_yet_done', 'dot_review', 'bid_pending', 'waiver_pending'].includes(state)) return 1;
  if (['landed', 'awarded_off_app'].includes(state)) return 2;
  return 3;
}
export function HQQueue({ moves, available }: { moves: HQMove[]; available: boolean }) {
  if (!available) return <span className="qn dim">{unavailable}</span>;
  return <><span className="qn need">{moves.filter(m => moveLane(m.state) === 3).length} NEED YOU</span>
    <span className="qn">{moves.filter(m => moveLane(m.state) === 1).length} WAITING</span>
    <span className="qn dim">{moves.filter(m => moveLane(m.state) === 2).length} LANDED</span></>;
}
export function HQMoves({ moves, snapshot, available = true, reason }: { moves: HQMove[]; snapshot: Snapshot; available?: boolean; reason?: string }) {
  return <section className="ws movews on" id="ws-moves">
    {['PLANNED', 'HANDED OFF', 'LANDED', 'NEEDS YOU'].map((label, lane) => {
      const items = moves.filter(m => moveLane(m.state) === lane);
      return <div className="lane" key={label}><div className="ch"><b>{label} <span className="mono">{available ? items.length : ''}</span></b>
        <span>{['app checked it', moves.some(m => m.native) ? 'waiting on league' : 'waiting on MFL', moves.some(m => m.native) ? 'recorded in league' : 'seen in MFL data', 'failed · stale · not verified'][lane]}</span></div>
        <div className="cards">{items.map(m => <article className={`mv ${lane === 3 ? 'blocked' : ''}`} key={m.id}>
          <div className="top"><span>{intentLabels[m.intent] ?? m.intent}</span><span>{stateLabels[m.state]}</span></div>
          <div className="what">{m.players.map(id => snapshot.players.value.find(p => p.id === id)?.name ?? id).join(' · ') || intentLabels[m.intent] || m.intent}</div>
          <div className="chk">{m.note || unavailable}</div><div className="bot"><time dateTime={m.at}>{m.at ? new Date(m.at).toLocaleString() : unavailable}</time>
            {m.handoff && m.state === 'ready' && <button className="btn" onClick={m.handoff}>OPEN MFL ↗</button>}</div>
        </article>)}{!items.length && <p className="empty" title={reason}>{unavailable}</p>}</div></div>;
    })}
    <div className="mlog"><div className="ch"><b>SEASON LEDGER</b><span>every move you made, as recorded</span></div>
      <div className="ct-wrap"><table className="ct"><thead><tr>{['KIND', 'WHEN', 'MOVE', 'CAP', 'RESULT'].map(h => <th key={h}>{h}</th>)}</tr></thead>
        <tbody>{moves.filter(m => m.native && m.state === 'landed').map(m => <tr key={m.id}><td>{m.intent}</td><td>{new Date(m.at).toLocaleString()}</td><td>{m.note}</td><td>{unavailable}</td><td>{stateLabels[m.state]}</td></tr>)}</tbody></table>
        {!moves.some(m => m.native && m.state === 'landed') && <p className="empty">{unavailable}</p>}</div></div>
  </section>;
}
