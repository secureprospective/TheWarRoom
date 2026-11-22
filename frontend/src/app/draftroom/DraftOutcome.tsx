import type { Closed } from './draft';
import { when } from './draftWords';

const picksOf = (n: number) => `${n} ${n === 1 ? 'pick' : 'picks'}`;

// DraftOutcome is what closing the draft did: the rookies nobody took, now in the free-agent pool, and the
// picks issued to every franchise for the seasons ahead.
export function DraftOutcome({ closed }: { closed: Closed }) {
  return (
    <section className="draft-closed" aria-label="What closing the draft did">
      <h5>What closing the draft did</h5>
      <p>Closed <time dateTime={closed.at}>{when(closed.at)}</time>.</p>
      <h6>Undrafted rookies, now free agents</h6>
      {closed.pool.length === 0 ? <p>Every rookie was drafted.</p> : (
        <ul>{closed.pool.map(p => <li key={p.id}>{p.name}, {p.position} · {p.team}</li>)}</ul>
      )}
      <h6>New picks issued</h6>
      {closed.issued.length === 0 ? <p>No new picks were issued.</p> : (
        <ul>
          {closed.issued.map(i => (
            <li key={i.season}>{i.season}: {picksOf(i.picks)} over {i.rounds} rounds,{' '}
              {i.tradable ? 'tradable now' : 'not tradable yet'}</li>
          ))}
        </ul>
      )}
    </section>
  );
}
