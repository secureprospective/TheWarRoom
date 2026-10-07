import { memo } from 'react';
import type { ClockProps } from './labels';
import { Act } from '../commands/Act';
import { ClockSignal } from './ClockSignal';
import { deadlineLabel, phaseLabels } from './labels';
import './clock.css';

// The strip is glanceable; the full sentence rides in the title and the calendar panel.
const EMPTY = 'no deadlines on the commissioner calendar';

function weekSegment(week: number | undefined): string {
  return week === undefined ? '' : `Wk ${week} · `;
}

export const ClockStrip = memo(function ClockStrip(
  { reading }: ClockProps,
) {
  if (!reading) return <span>Clock loading…</span>;
  if (reading.provenance.freshness.state === 'fail') {
    return <span><ClockSignal provenance={reading.provenance} /> Clock unavailable</span>;
  }
  const next = reading.value.deadlines[0];
  const lead = `${phaseLabels[reading.value.phase]} · ${weekSegment(reading.value.week)}`;
  return (
    <>
      {reading.provenance.freshness.state === 'stale' ? (
        <ClockSignal provenance={reading.provenance} />
      ) : null}
      <Act
        verb="calendar.summon"
        args={{}}
        controls="target-calendar"
        label={next ? undefined : `${lead}${EMPTY}`}
      >
        {lead}
        {next ? (
          <span data-deadline={next.id}>
            {deadlineLabel(next.label)} ·{' '}
            <span className="channel-countdown" data-countdown>—</span>
          </span>
        ) : 'no deadlines'}
      </Act>
    </>
  );
});
