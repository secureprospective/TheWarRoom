import { HQAct } from '../shell/HQAct';
import { HoldToFire } from './HoldToFire';
import { hostedWords } from './hostedWords';
import { previewChange } from './preview';
import type { HostedLeagueRoom } from './hostedRoom';

export function Review({ room }: { room: HostedLeagueRoom }) {
  const { review, pending, reading } = room.use();
  const words = (value: string) => hostedWords(reading, value);
  if (!review) return null;
  return (
    <section aria-label="Confirm league change" className="hosted-review">
      <h3>{review.summary}</h3>
      <p>This changes your league server, not MFL or a local plan. Nothing is queued while offline.</p>
      {review.preview?.fields.map((field, i) => (
        <p key={i}>
          <strong>{words(field.label)}</strong>: {words(previewChange(field))}
        </p>
      ))}
      {review.preview?.blockers.map((b, i) => (
        <p role="alert" key={i}>
          {words(b)}
        </p>
      ))}
      {!review.preview?.blockers.length && (
        <HoldToFire
          label="Confirm league change"
          body={JSON.stringify(review.request)}
          disabled={pending || !room.canChange()}
          onConfirm={() => void room.confirm()}
        />
      )}
      <HQAct disabled={pending} onAction={() => room.cancel()}>Cancel change</HQAct>
    </section>
  );
}
