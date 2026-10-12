import { useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import type { Snapshot } from '../data/contract';
import { AssetNumber } from './AssetNumber';
import { assetState } from './state';

type Row = { id: string; name: string; position: string; team: string; franchiseId: string;
  franchise: string };

// rosterRows lists every rostered player with the names the board shows.
export function rosterRows(snapshot: Snapshot): Row[] {
  const players = new Map(snapshot.players.value.map(p => [p.id, p]));
  const names = new Map(snapshot.franchises.value.map(f => [f.id, f.name ?? `Franchise ${f.id}`]));
  return snapshot.rosters.value.flatMap(roster => roster.players.map(({ id }) => {
    const p = players.get(id);
    return { id, name: p?.name ?? id, position: p?.position ?? '—', team: p?.team ?? '—',
      franchiseId: roster.franchiseId, franchise: names.get(roster.franchiseId) ?? roster.franchiseId };
  }));
}

// C-01: what every rostered player is worth on the field now and as a dynasty asset. Values only.
export function AssetBoardView({ snapshot }: { snapshot: Snapshot }) {
  const mine = commands.use().franchiseId ?? '';
  const [franchise, setFranchise] = useState<string>();
  const [position, setPosition] = useState('');
  const board = assetState.useBoard();
  const shown = franchise ?? mine;
  const all = rosterRows(snapshot);
  const value = (id: string) => board.byId.get(id);
  const rows = all
    .filter(r => (!shown || r.franchiseId === shown) && (!position || r.position === position))
    .sort((a, b) => (value(b.id)?.nowValue ?? -Infinity) - (value(a.id)?.nowValue ?? -Infinity) ||
      a.name.localeCompare(b.name));
  const positions = [...new Set(all.map(r => r.position))].sort();
  return <section className="asset-board" aria-label="Asset board" data-endpoint="C-01">
    <h4>Asset board</h4>
    <p>What each rostered player is worth on the field now and as a dynasty asset.
      Values only; nothing moves.</p>
    <div className="market-controls race-choices">
      <label>Franchise{' '}
        <select value={shown} onChange={e => setFranchise(e.target.value)}>
          <option value="">All franchises</option>
          {snapshot.franchises.value.map(f => <option key={f.id} value={f.id}>{f.name ?? f.id}</option>)}
        </select>
      </label>
      <label>Position{' '}
        <select value={position} onChange={e => setPosition(e.target.value)}>
          <option value="">All positions</option>
          {positions.map(p => <option key={p} value={p}>{p}</option>)}
        </select>
      </label>
    </div>
    {board.board && <p className="report-source">{board.board.nowSource} · {board.board.dynastySource}</p>}
    {board.error && <p role="alert">Player values unavailable · {board.error}</p>}
    <table className="report-table">
      <thead><tr>
        <th scope="col">Player</th><th scope="col">Pos</th><th scope="col">NFL</th>
        {!shown && <th scope="col">Franchise</th>}
        <th scope="col">On-field-now</th><th scope="col">Dynasty value</th><th scope="col">Market</th>
      </tr></thead>
      <tbody>{rows.map(r => <tr key={`${r.franchiseId}:${r.id}`}>
        <td><Act verb="inspector.open"
          args={{ subject: { kind: 'player', id: r.id, franchiseId: r.franchiseId } }}
          label={`Inspect ${r.name}`}>{r.name}</Act></td>
        <td>{r.position}</td><td>{r.team}</td>
        {!shown && <td>{r.franchise}</td>}
        <td><AssetNumber playerId={r.id} which="now" inline /></td>
        <td><AssetNumber playerId={r.id} which="dynasty" inline /></td>
        <td><AssetNumber playerId={r.id} which="market" inline /></td>
      </tr>)}</tbody>
    </table>
    {board.board && <details className="market-how"><summary>How this is counted</summary>
      <p>{board.board.how}</p></details>}
  </section>;
}
