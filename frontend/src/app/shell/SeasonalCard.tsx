import { useEffect, useRef } from 'react';
import type { ClockProps } from '../clock/labels';
import { phaseLabels, deadlineLabel, localDate } from '../clock/labels';
import { connectClock } from '../clock/ticker';
import { countdown } from '../clock/urgency';
import { ClockSignal } from '../clock/ClockSignal';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { deadlineRoute } from '../clock/deadlineRoute';

export default function SeasonalCard({ reading }: ClockProps) {
  const root = useRef<HTMLElement>(null);
  useEffect(() => {
    if (reading && root.current) return connectClock(root.current, reading.value.deadlines);
  }, [reading]);
  if (!reading) return <p role="status">Clock loading…</p>;
  if (reading.provenance.freshness.state === 'fail') {
    return <p role="alert">Clock unavailable · {reading.provenance.freshness.note}</p>;
  }
  const { season, phase, week, deadlines } = reading.value;
  return <section ref={root} aria-label="Seasonal card">
    <Card gravity="G2" header={<h4>Season {season} · {phaseLabels[phase]}</h4>}>
      <p>{week === undefined ? 'NFL week unknown' : `NFL week ${week}`}</p>
      {deadlines.length ? <ul className="season-deadlines">
        {deadlines.map((deadline) => <li key={deadline.id} data-deadline={deadline.id}
          data-urgency={deadline.urgency}>
          <Act {...deadlineRoute(deadline.id)} label={deadlineLabel(deadline.label)} />{' · '}
          <time dateTime={deadline.at}>{localDate(deadline.at)}</time>{' · '}
          <span className="channel-countdown" data-countdown>{countdown(deadline.at, Date.now())}</span>
        </li>)}
      </ul> : <p>No deadlines on the commissioner calendar.</p>}
      <Act verb="calendar.summon" args={{}} label="League windows and their sources" />
      <div className="market-controls" aria-label="Season phase workspaces" data-endpoint="S-13">
        {(['OFFSEASON', 'DRAFT', 'REGULAR_SEASON', 'PLAYOFFS'] as const).map(destination =>
          <Act key={destination} verb="phase.goto" args={{ phase: destination }}
            active={destination === phase} current={destination === phase}
            label={phaseDestination[destination]} />)}
      </div>
      <ClockSignal provenance={reading.provenance} />
    </Card>
  </section>;
}

const phaseDestination = {
  OFFSEASON: 'Offseason · Contracts and cap',
  DRAFT: 'Draft · Draft room',
  REGULAR_SEASON: 'Regular season · Lineup and roster',
  PLAYOFFS: 'Playoffs · Race',
};
