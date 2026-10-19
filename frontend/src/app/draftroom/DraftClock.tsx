import { Act } from '../commands/Act';
import type { Draft } from './draft';
import { contractWords, countdownWords, franchiseName, nanosToMs, when } from './draftWords';
import { trayValue, type TrayAction } from './trayAction';

// DraftLauncher is the only thing the draft room carries for a commissioner change: a control that opens the
// change in the side tray. The form, its preview and its confirm live in the tray.
export function DraftLauncher({ action, text, label = text }:
  { action: TrayAction; text: string; label?: string }) {
  return (
    <Act verb="commissioner.surface" variant="button" label={label}
      args={{ screen: 'draftroom', action: 'tray', value: trayValue(action) }}>{text}</Act>
  );
}

function Running({ draft, now }: { draft: Draft; now: number }) {
  const c = draft.onClock!;
  const left = c.paused ? nanosToMs(c.remaining) : Date.parse(c.deadline) - now;
  return (
    <>
      <p className="draft-clock-who">
        On the clock: <strong>{franchiseName(draft, c.owner)}</strong>, round {c.round}, pick {c.slot}
        {c.original !== c.owner && ` (originally ${franchiseName(draft, c.original)}’s)`}
      </p>
      <p className="draft-countdown">
        <span role="timer" aria-live="off">{countdownWords(left)}</span>
        {c.paused ? ' left, and the clock is paused'
          : <> left. The pick closes <time dateTime={c.deadline}>{when(c.deadline)}</time></>}
      </p>
      <p>This pick signs for {contractWords(c.contract)}.</p>
    </>
  );
}

// DraftClock says whose turn it is, how long is left and when the pick closes in the viewer's own time.
export function DraftClock({ draft, now, may }: { draft: Draft; now: number; may: boolean }) {
  const running = draft.onClock !== null && draft.closed === null;
  const waiting = !draft.started && draft.closed === null && Date.parse(draft.start) > 0;
  return (
    <section className="draft-clock" aria-label="The pick clock">
      <h5>The pick clock</h5>
      <p role="status">{draft.words}</p>
      {waiting && (
        <p className="draft-countdown">
          Starts in <span role="timer" aria-live="off">{countdownWords(Date.parse(draft.start) - now)}</span>,
          on <time dateTime={draft.start}>{when(draft.start)}</time>
        </p>
      )}
      {running && <Running draft={draft} now={now} />}
      {draft.closed && <p>Closed <time dateTime={draft.closed.at}>{when(draft.closed.at)}</time>
        {draft.closed.automatic ? ', automatically, at the last pick' : ' by the commissioner'}.</p>}
      {may && running && (
        <div className="draft-launchers" role="group" aria-label="Work the clock">
          <DraftLauncher action={{ kind: 'skip' }} text="Skip this pick" />
          <DraftLauncher action={{ kind: draft.onClock!.paused ? 'resume' : 'pause' }}
            text={draft.onClock!.paused ? 'Resume the clock' : 'Pause the clock'} />
          <DraftLauncher action={{ kind: 'set' }} text="Set the clock" />
        </div>
      )}
    </section>
  );
}
