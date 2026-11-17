import type { Snapshot } from '../data/contract';
import type { NativeCut } from '../commissioner/hqReading';
import { HQAct } from './HQAct';
import { money, unavailable } from './hqModel';
import { HQWhatIfDrawer } from './HQWhatIfDrawer';

export function HQNativeWhatIf({ snapshot, mine, chosen, select, cut, beforeRoom, close }: {
  snapshot: Snapshot; mine: string; chosen?: string; select: (id: string) => void;
  cut?: NativeCut; beforeRoom?: number; close: () => void;
}) {
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const quote = cut?.player === chosen ? cut : undefined;
  const input = roster.map(p => <HQAct key={p.id} className={`wi ${chosen === p.id ? 'on' : ''}`}
    onAction={() => select(p.id)}><span className="box" /><span className="k">CUT</span>
    <span>{snapshot.players.value.find(v => v.id === p.id)?.name ?? p.id}</span>
    <span className="d">{chosen === p.id ? money(quote?.deadNow) : ''}</span></HQAct>);
  const preview = <><div className="sum"><div><b>{money(quote?.roomAfter)}</b><span>CAP ROOM</span>
    <span className="delta">BEFORE {money(beforeRoom)} ┆ AFTER {money(quote?.roomAfter)}</span></div>
    <div><b>{money(quote?.deadNow)}</b><span>DEAD CAP NOW</span></div>
    <div><b>{money(quote?.deadNext)}</b><span>DEAD CAP NEXT YR</span></div></div>
    <p className="note">{quote?.reason}</p>
    <p className="note" title="The hosted workflow has no stacked simulation or saved local plan command">
      Stacked moves · {unavailable}</p></>;
  return <HQWhatIfDrawer close={close} detail="Nothing is saved or sent." port={{ input, preview }} />;
}
