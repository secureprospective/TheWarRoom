import { HQAct } from './HQAct';
import { useMemo, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { parseLedger } from '../commissioner/proxy';
import { useContractRead } from '../contracts/Contracts';
import { onHostedTopics } from '../data/hostedReads';

// History is a season page of the audience-filtered native ledger, not pending move payloads.
export function HQMoveHistory({ league, franchise, season, port = hostedWorkflow }: {
  league: string; franchise: string; season: number; port?: HostedPort;
}) {
  const [selected, select] = useState(season);
  const [retry, refresh] = useState(0);
  const provider = useMemo(() => ({ onChange: (load: () => void) => onHostedTopics(topics => {
    if (topics.some(t => t.league === league && (t.kind === 'league'
      || (t.kind === 'team' && t.franchise === franchise)))) load();
  }) }), [league, franchise, port]);
  const { reading, pending, error } = useContractRead(
    JSON.stringify([league, franchise, selected, retry]), async () => {
      const result = await port({ command: 'franchise.moves.history.read', mode: 'read',
        arguments: { league, franchise, season: selected } });
      const ledger = parseLedger(result.data);
      if (ledger.league !== league) throw new Error('History belongs to another league.');
      return ledger;
    }, provider);
  return <div className="mlog" aria-label="Season ledger">
    <div className="ch"><b>SEASON LEDGER · {selected}</b>
      <span>recorded franchise and league changes</span></div>
    <div className="ch" aria-label="History seasons">
      <HQAct className="btn" disabled={selected <= 1920} onAction={() => select(y => y - 1)}>
        Previous season</HQAct>
      <span className="mono">{selected}</span>
      <HQAct className="btn" disabled={selected >= season} onAction={() => select(y => y + 1)}>
        Next season</HQAct>
      <HQAct className="btn" disabled={pending} onAction={() => refresh(n => n + 1)}>Refresh history</HQAct>
    </div>
    {pending && <p className="empty" role="status">Reading season history…</p>}
    {error && <p className="empty" role="alert">
      History unavailable. Refresh history to try again. {error}</p>}
    {reading && <div className="ct-wrap"><table className="ct"><thead><tr>
      {['WHEN', 'MOVE', 'BY', 'REASON'].map(h => <th scope="col" key={h}>{h}</th>)}
    </tr></thead><tbody>{reading.entries.map(entry => <tr key={entry.id}>
      <td><time dateTime={entry.when}>{new Date(entry.when).toLocaleString()}</time></td>
      <td>{entry.what}{entry.replacedAt && <p>Replaced on <time dateTime={entry.replacedAt}>
        {new Date(entry.replacedAt).toLocaleString()}</time></p>}</td>
      <td>{entry.who}</td><td>{entry.why || 'No reason recorded'}</td>
    </tr>)}</tbody></table>{!reading.entries.length && <p className="empty">
      No changes visible to you for this franchise in {selected}.</p>}</div>}
  </div>;
}
