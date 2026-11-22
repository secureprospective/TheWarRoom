import { memo } from 'react';
import { WindowRow } from './WindowRow';
import type { ClockProps } from './labels';
import { ClockSignal } from './ClockSignal';
import { deadlineLabel, localDate, phaseLabels } from './labels';

export const CalendarPanel = memo(function CalendarPanel(
  { reading }: ClockProps,
) {
  if (!reading) return <p>Clock loading…</p>;
  if (reading.provenance.freshness.state === 'fail') {
    return (
      <p>
        <ClockSignal provenance={reading.provenance} /> Clock unavailable{' · '}
        {reading.provenance.freshness.note}
      </p>
    );
  }
  const { deadlines, windows, phase, week } = reading.value;

  return (
    <section className="league-calendar" aria-label="League clock" data-endpoint="M-096">
      <h4>{phaseLabels[phase]}</h4>
      {week === undefined ? null : <p>NFL week {week}</p>}
      {deadlines.length ? (
        <ul>
          {deadlines.map((deadline) => (
            <li key={deadline.id} data-deadline={deadline.id}>
              <b>{deadlineLabel(deadline.label)}</b>{' · '}
              <span>{localDate(deadline.at)}</span>{' · '}
              <span className="channel-countdown" data-countdown>—</span>
              {deadline.pinned ? <small> · pinned</small> : null}
              {!deadline.pinned && deadline.promoted ? <small> · promoted</small> : null}
            </li>
          ))}
        </ul>
      ) : <p>No deadlines on the commissioner calendar.</p>}
      <h4>League windows</h4>
      <ul>
        {windows.map(window => <WindowRow key={window.kind} window={window} />)}
      </ul>
      <p>
        <ClockSignal provenance={reading.provenance} />{' '}
        {reading.provenance.freshness.note}
      </p>
    </section>
  );
});
