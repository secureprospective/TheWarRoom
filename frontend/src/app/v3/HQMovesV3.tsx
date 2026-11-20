// Desktop v3 Franchise HQ, My moves, after TheWarRoom-mock-v3-FINAL.html (screens.js hq.moves): this
// franchise's moves in three lanes, Needs you, Waiting on and Landed. Every row is the server's own
// moves.read; the season ledger (franchise.moves.history.read) opens below on request.
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { moveLane, type HQMove } from '../shell/HQMoves';
import { intentLabels, stateLabels } from '../shell/moveLabels';
import { nativeCommands } from '../commands/native.gen';
import type { NativeCommandId } from '../commands/native';
import type { Snapshot } from '../data/contract';

const title = (intent: string) =>
  intentLabels[intent] ?? nativeCommands[intent as NativeCommandId]?.phrase ?? 'Move';
const when = (at: string) => at ? new Date(at).toLocaleString('en-US', { month: 'short', day: 'numeric',
  hour: 'numeric', minute: '2-digit', timeZone: 'America/Chicago' }) : '';

/** Needs you holds what is still in the owner's hands (planned or failed); then waiting; then landed. */
export function movesLanes(moves: HQMove[]): [HQMove[], HQMove[], HQMove[]] {
  const lane = (n: number[]) => moves.filter(m => n.includes(moveLane(m.state)));
  return [lane([3, 0]), lane([1]), lane([2])];
}

export function HQMovesV3({ moves, snapshot, available, reason, badge, history }: {
  moves: HQMove[]; snapshot: Snapshot; available: boolean; reason?: string; badge: ReactNode;
  history: ReactNode }) {
  const [ledger, setLedger] = useState(false);
  const names = new Map(snapshot.players.value.map(p => [p.id, p.name]));
  const lanes = movesLanes(moves);
  const empty = ['Nothing needs you.', 'Nothing is waiting on the league.', 'No moves landed yet.'];
  return <div className="wbody scr scr-hq"><div className="lay" style={{
    gridTemplateColumns: 'repeat(3,minmax(0,1fr))',
    gridTemplateRows: ledger ? 'minmax(0,1fr) minmax(0,1fr)' : 'minmax(0,1fr)' }}>
    {['Needs you', 'Waiting on', 'Landed'].map((name, i) => <section key={name} className="pc panel sz-L">
      <div className="ph">{badge}<b className="pt">{name}</b><span className="sp" />
        {available && <span className={`tag${i === 0 && lanes[0].length ? ' o' : ''}`}>
          {lanes[i].length}</span>}</div>
      <div className="pb">{!available
        ? <div className="st"><div className="eng">{reason || 'Moves are not available.'}</div></div>
        : !lanes[i].length ? <div className="st"><div className="eng">{empty[i]}</div></div>
          : lanes[i].map(m => <div key={m.id} className="dk">{badge}
            <span className="t ell">{m.players.map(id => names.get(id) ?? id).join(' · ') || title(m.intent)}
              {m.players.length ? ` · ${title(m.intent)}` : ''}</span>
            <span className="s ell">{m.note || (m.state === 'needs_you' ? 'Needs you' : stateLabels[m.state])}
            </span>
            <span className="ck">{when(m.at)}</span>
            {m.handoff && m.state === 'ready' && <HQAct className="btn sm" onAction={m.handoff}>
              Open MFL ↗</HQAct>}</div>)}</div>
      <div className="pf"><span className="ell">{i === 2
        ? <HQAct className="lk" onAction={() => setLedger(v => !v)}>
          {ledger ? 'Hide season ledger' : 'Season ledger ▾'}</HQAct>
        : "this franchise's moves, as the league recorded them"}</span></div>
    </section>)}
    {ledger && <section className="pc panel sz-L" style={{ gridColumn: '1/4', overflow: 'auto' }}>
      <div className="ph">{badge}<b className="pt">Season ledger</b><span className="sp" /></div>
      <div className="pb hq-v10">{history}</div></section>}
  </div></div>;
}
