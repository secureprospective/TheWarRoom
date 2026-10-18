import { useState } from 'react';
import { Act } from '../commands/Act';
import type { Draft } from './draft';
import { contractWords, playerLine } from './draftWords';
import type { PickOutcome } from './state';

// YourPick is the owner's own pick: choose a rookie for the pick on your clock and make it. Go checks
// whose turn it is, the roster and the cap, and says so in words if the pick cannot be made.
export function YourPick({ draft, outcome }: { draft: Draft; outcome: PickOutcome }) {
  const [choice, setChoice] = useState('');
  const clock = draft.onClock;
  if (!clock || clock.paused) return null;
  const chosen = draft.available.find(p => p.id === choice);
  return (
    <section className="draft-your-pick" aria-label="Your pick">
      <h5>It is your pick</h5>
      <label>Rookie
        <select value={choice} onChange={e => setChoice(e.target.value)}>
          <option value="">Choose…</option>
          {draft.available.map(p => <option key={p.id} value={p.id}>{playerLine(p)}</option>)}
        </select>
      </label>
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
