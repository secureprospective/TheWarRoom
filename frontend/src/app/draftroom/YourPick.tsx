import { useState } from 'react';
import { Act } from '../commands/Act';
import type { Draft } from './draft';
import { contractWords, playerLine } from './draftWords';
import type { PickOutcome } from './state';

// YourPick is the owner's own pick: choose a rookie for the pick on your clock and make it. Go checks
// whose turn it is, the roster and the cap, and says so in words if the pick cannot be made.
export function YourPick({ draft, outcome }: { draft: Draft; outcome: PickOutcome }) {
  const [choice, setChoice] = useState('');
  const [search, setSearch] = useState('');
  const [position, setPosition] = useState('');
  const clock = draft.onClock;
  if (!clock || clock.paused) return null;
  const positions = [...new Set(draft.available.map(p => p.position))].sort();
  const needle = search.trim().toLowerCase();
  const shown = draft.available
    .filter(p => p.name.toLowerCase().includes(needle) && (position === '' || p.position === position));
  const chosen = draft.available.find(p => p.id === choice);
  return (
    <section className="draft-your-pick" aria-label="Your pick">
      <h5>It is your pick</h5>
      <label>Search by name
        <input value={search} onChange={e => setSearch(e.target.value)} />
      </label>
      <label>Position
        <select value={position} onChange={e => setPosition(e.target.value)}>
          <option value="">All positions</option>
          {positions.map(pos => <option key={pos} value={pos}>{pos}</option>)}
        </select>
      </label>
      <label>{draft.startup ? 'Player' : 'Rookie'}
        <select value={choice} onChange={e => setChoice(e.target.value)}>
          <option value="">Choose…</option>
          {shown.map(p => <option key={p.id} value={p.id}>{playerLine(p)}</option>)}
        </select>
      </label>
      {shown.length === 0 && <p>No player matches.</p>}
      <p>This pick signs for {contractWords(clock.contract)}.</p>
      <Act verb="commissioner.surface" variant="button" disabled={!chosen || outcome.status === 'sending'}
        label={chosen ? `Pick ${chosen.name}` : 'Make this pick'}
        args={{ screen: 'draftroom', action: 'pick', value: choice }}>
        {chosen ? `Pick ${chosen.name}` : 'Make this pick'}
      </Act>
      {outcome.status === 'sending' && <p role="status">Making your pick…</p>}
      {outcome.status === 'done' && <p role="status">{outcome.message}</p>}
      {outcome.status === 'refused' && <p role="alert">{outcome.message}</p>}
    </section>
  );
}
