import { useEffect, useMemo, useState } from 'react';
import { useHomeStore } from '../../store/home';
import { useHarnessStore } from '../../store/harness';
import { EngraveState } from '../board/primitives';
import { SeasonalCard } from './SeasonalCard';
import { selectSeasonal } from './seasonal';

// HomeBoard — the league landing (UI_Direction_Document §10.4, Session-A §6).
//
// Geometry: a symmetric 2×2 quadrant grid that FORCES narrative density regardless of the global
// setting (§10.4: "the landing is always approachable"). Density is scoped by re-declaring
// data-density on this container, so the shell's global tier is untouched when the operator leaves.
//
// Card slots are the locked four: League activity · Seasonal · Trade block · League chat. Only the
// slots with a real backend source render live data at alpha — the other three engrave what will
// live there, per the §8 honest-empty rule and the B-4a ruling that a surface never invents a value
// it was not given:
//   - League activity  → NO SOURCE. There is no transaction-feed reader; the only `ledger` in
//                        internal/store/state is the per-player contract ledger, not an event log.
//   - Trade block      → NO SOURCE. The backend carries no trade-block concept.
//   - League chat      → NO SOURCE. The comms layer does not exist (its summon is still a
//                        placeholder in App.tsx).
// Engraving them is the point: a first run should read as oriented, not broken.

export function HomeBoard() {
  const load = useHomeStore((s) => s.load);
  const phase = useHomeStore((s) => s.phase);
  const events = useHomeStore((s) => s.events);
  const phaseError = useHomeStore((s) => s.phaseError);
  const calendarError = useHomeStore((s) => s.calendarError);

  // Home READS the persisted board but never triggers its load. GetRankings resolves display names
  // through the players directory, which is a (cached, degrade-not-fail) MFL fetch — so calling it
  // from here would put a network touch on the landing, against the B-4b ruling that Home is
  // local-only. M1 is the default module and loads it on mount; if the operator lands on Home
  // first with nothing scored yet, the pulse engraves instead. (GLM review lead H1.)
  const rankings = useHarnessStore((s) => s.rankings);

  // `now` is captured once per mount and ticked each minute — the countdown must not be recomputed
  // on every render (it would make selectSeasonal's output depend on render timing) and must not
  // sit frozen while the operator watches a deadline approach.
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 60000);
    return () => window.clearInterval(id);
  }, []);

  const card = useMemo(() => selectSeasonal(phase, events, now), [phase, events, now]);

  return (
    <div className="twr-home" data-density="narrative">
      <HomeCard title="LEAGUE ACTIVITY">
        <EngraveState
          lines={['League activity', '— no transaction feed —', 'bids · votes · waivers · trades']}
        />
      </HomeCard>

      <HomeCard title={card.title} note={card.note}>
        {/* BOTH banners render when both reads fail. Showing only one would report a corrupt or
            locked local DB as a single failing card, hiding that the whole state engine is down
            — the store already tracks the two errors separately for exactly this reason.
            (GLM review lead H3.) */}
        {phaseError && <div className="twr-banner twr-banner--warn">{phaseError}</div>}
        {calendarError && <div className="twr-banner twr-banner--warn">{calendarError}</div>}
        <SeasonalCard card={card} now={now} rankings={rankings} />
      </HomeCard>

      <HomeCard title="TRADE BLOCK">
        <EngraveState
          lines={['Trade block', '— no listings source —', 'players listed across 32 teams']}
        />
      </HomeCard>

      <HomeCard title="LEAGUE CHAT">
        <EngraveState lines={['League chat', '— comms layer pending —', 'last 3 messages']} />
      </HomeCard>
    </div>
  );
}

// HomeCard — one quadrant. A titled tile on the sunken surface; the grid lines vanish and negative
// space carries the layout (Session-A §6: "a deliberate slow-down break").
function HomeCard({
  title,
  note,
  children,
}: {
  title: string;
  note?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="twr-home__card">
      <header className="twr-home__head">
        <h2 className="twr-home__title">{title}</h2>
        {note && <p className="twr-home__note">{note}</p>}
      </header>
      <div className="twr-home__body">{children}</div>
    </section>
  );
}
