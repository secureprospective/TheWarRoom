import { useMemo, useState } from 'react';
import { HQAct } from './HQAct';
import type { Node } from './nodes';

type Leaf = { slug: string; label: string };
export type PlayerCandidate = { id: string; name: string; position?: string };

export function findPlayers(players: readonly PlayerCandidate[], query: string): PlayerCandidate[] {
  const needles = query.toLocaleLowerCase().match(/[\p{L}\p{N}]+/gu) ?? [];
  if (!needles.length) return [];
  const byId = new Map(players.filter(player => player.id && player.name.trim())
    .map(player => [player.id, player]));
  return [...byId.values()].filter(player => {
    const nameParts = player.name.toLocaleLowerCase().match(/[\p{L}\p{N}]+/gu) ?? [];
    return needles.every(needle => nameParts.some(part => part.includes(needle)));
  }).sort((a, b) => a.name.localeCompare(b.name)).slice(0, 8);
}

export function ThinPlayerSearch({ players, onOpen }: {
  players: readonly PlayerCandidate[]; onOpen: (player: PlayerCandidate) => void;
}) {
  const [query, setQuery] = useState('');
  const matches = useMemo(() => findPlayers(players, query), [players, query]);
  return <div className="player-search">
    <label htmlFor="thin-player-search">PLAYER SEARCH</label>
    <input id="thin-player-search" type="search" autoComplete="off" value={query}
      placeholder="Find a player" onChange={event => setQuery(event.target.value)} />
    {query.trim() && <ul aria-label="Player search results">
      {matches.length ? matches.map(player => <li key={player.id}>
        <HQAct className="leaf search-result" label={`Open ${player.name} player card`}
          onAction={() => onOpen(player)}>{player.name}{player.position &&
            <small>{player.position}</small>}</HQAct>
      </li>) : <li className="search-empty">No player found in loaded league reads.</li>}
    </ul>}
  </div>;
}

export type ThinNode = Extract<Node, 'home' | 'hq' | 'war' | 'pulse' | 'control'>;
export type ThinScreenRoute = { node: ThinNode; workspace: string };
export const THIN_LEAVES: Partial<Record<ThinNode, readonly Leaf[]>> = {
  home: [{ slug: 'overview', label: 'Overview' }],
  hq: [{ slug: 'lineup-and-roster', label: 'Lineup and roster' },
    { slug: 'contracts-and-cap', label: 'Contracts and cap' }, { slug: 'my-moves', label: 'My moves' }],
  war: [{ slug: 'trade-desk', label: 'Trade desk' }, { slug: 'valuations-and-pool', label: 'Market' },
    { slug: 'draft-room', label: 'Draft room' }, { slug: 'plans', label: 'Plans' }],
  pulse: [{ slug: 'now', label: 'Now' }, { slug: 'race', label: 'Race' }],
  control: [{ slug: 'league', label: 'League' }],
};
