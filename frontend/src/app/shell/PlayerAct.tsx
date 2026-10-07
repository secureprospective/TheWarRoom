import type { PlayerSubject } from './state';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import ReceiptStatus from './ReceiptStatus';

export default function PlayerAct({ subject, executor = commands }: {
  subject: PlayerSubject;
  executor?: typeof commands;
}) {
  executor.use();
  const key = `${subject.franchiseId}:${subject.id}`;
  const pending = executor.useMoves((s) => s.drafting[key]);
  const error = executor.useMoves((s) => s.draftErrors[key]);
  const receipt = executor.useMoves((s) => s.moves.find((r) =>
    r.spec.franchiseId === subject.franchiseId && r.spec.subject.players.includes(subject.id)));
  const reason = executor.draftReason(subject);
  return (
    <section className="zone">
      <h5>Act</h5>
      <Act
        verb="roster.ir"
        args={{ subject }}
        disabled={Boolean(reason)}
      >
        {pending ? 'Drafting…' : 'Draft IR placement'}
      </Act>
      {reason && <p>{reason}</p>}
      {error && <p role="alert">{error}</p>}
      {receipt && <ReceiptStatus receipt={receipt} />}
      <Act verb="nav.open" args={{ node: 'hq', workspace: 'my-moves' }}>
        My moves
      </Act>
    </section>
  );
}
