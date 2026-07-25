import { main } from '../../../wailsjs/go/models';
import { EngraveState } from '../board/primitives';
import { countdown, kindLabel, parseAt, type SeasonalCard as Card } from './seasonal';

// SeasonalCard — renders the single active seasonal card chosen by selectSeasonal. Presentation
// only: every branch here is driven by the pure model, so "which card is showing" is decided in one
// testable place and never re-derived from phase strings mid-render.

export function SeasonalCard({
  card,
  now,
  rankings,
}: {
  card: Card;
  now: number;
  rankings: main.RankingsResult | null;
}) {
  return (
    <>
      {card.imminent && <Imminent event={card.imminent} now={now} />}
      {card.kind === 'inseason' && <AssetPulse rankings={rankings} />}
      {card.kind === 'playoffs' && (
        <p className="twr-home__line">
          Advance to the next league year from CONTROL → League Controls (season rollover).
        </p>
      )}
      <PlannedQueue
        events={card.planned}
        now={now}
        // The imminent event already has its own hero treatment above; listing it again in the queue
        // would double-count the same blob. Skipped by OBJECT IDENTITY, not by eventID: `imminent`
        // is always an element of `planned` (selectSeasonal finds it there), and an id comparison
        // would drop EVERY row sharing a blank id if the backend ever emitted one. (GLM lead H2.)
        skip={card.imminent}
      />
    </>
  );
}

// Imminent — the countdown hero for a commissioner op inside the deadline window.
function Imminent({ event, now }: { event: main.CalendarEventDTO; now: number }) {
  const at = parseAt(event.scheduledAt);
  return (
    <div className="twr-home__hero">
      <div className="twr-home__herofig">{at === null ? '—' : countdown(at, now)}</div>
      <div className="twr-home__herolab">
        {kindLabel(event.kind)}
        {event.note ? ` · ${event.note}` : ''}
      </div>
    </div>
  );
}

// AssetPulse — the in-season read Home can actually back: the top of the persisted M1 board. It
// carries the board's own honesty label verbatim rather than presenting proxy scores as final.
function AssetPulse({ rankings }: { rankings: main.RankingsResult | null }) {
  const rows = (rankings?.rows ?? []).slice(0, 5);
  if (rows.length === 0) {
    return (
      <EngraveState lines={['Asset pulse', '— awaiting scored data —', 'run “Score League” in M1']} />
    );
  }
  return (
    <>
      <ul className="twr-home__list">
        {rows.map((r) => (
          <li key={r.mflID} className="twr-home__item">
            <span className="twr-home__rank">{r.rank}</span>
            <span className="twr-home__name">{r.name}</span>
            <span className="twr-home__pos">{r.position}</span>
            <span className="twr-home__val">{r.adjustedScore.toFixed(2)}</span>
          </li>
        ))}
      </ul>
      {rankings?.label && <p className="twr-home__foot">{rankings.label}</p>}
    </>
  );
}

// PlannedQueue — the scheduled commissioner ops, soonest first. An empty queue is a real state, not
// a failure: it renders nothing here because the card's `note` already says so.
function PlannedQueue({
  events,
  now,
  skip,
}: {
  events: main.CalendarEventDTO[];
  now: number;
  skip?: main.CalendarEventDTO;
}) {
  const shown = events.filter((e) => e !== skip).slice(0, 4);
  if (shown.length === 0) return null;
  return (
    <ul className="twr-home__list">
      {shown.map((e, i) => {
        const at = parseAt(e.scheduledAt);
        return (
          // Keyed on eventID plus index: the id is the real identity, but a blank id from the
          // backend would collide across rows, and a React key collision drops siblings silently.
          <li key={`${e.eventID}:${i}`} className="twr-home__item">
            <span className="twr-home__name">{kindLabel(e.kind)}</span>
            <span className="twr-home__val">{at === null ? '—' : countdown(at, now)}</span>
          </li>
        );
      })}
    </ul>
  );
}
