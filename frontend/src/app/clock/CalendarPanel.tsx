import { memo } from 'react';
import type { ClockProps } from './labels';
import { ClockSignal } from './ClockSignal';
import { deadlineLabel, localDate, phaseLabels, windowLabels } from './labels';
import { assertShortList } from '../shell/shortList';

export const CalendarPanel = memo(function CalendarPanel(
  { reading }: ClockProps,
) {
  if (!reading) return <p>Clock loading…</p>;
  if (reading.provenance.freshness.state === 'fail') {
    return <p><ClockSignal provenance={reading.provenance} /> Clock unavailable</p>;
  }
  const { deadlines, windows, phase } = reading.value;
  assertShortList(deadlines.length);
  return (
    <section className="league-calendar" aria-label="League clock">
      <h4>{phaseLabels[phase]}</h4>
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
        {windows.map((window) => (
          <li key={window.kind} title={window.reason}>
            {windowLabels[window.kind]} · unknown · gap closure
          </li>
        ))}
      </ul>
      <ClockSignal provenance={reading.provenance} />
    </section>
  );
});
