import type { PlayerSubject } from './state';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import './lineup.css';
import ReceiptStatus from './ReceiptStatus';
import { rosterHandoffLabel } from './ReceiptStatus';
import { eligibilityLine, ineligibleLabel } from './eligibilityLine';

export default function PlayerAct({ subject, executor = commands }: {
  subject: PlayerSubject;
  executor?: typeof commands;
}) {
  executor.use();
  const key = `${subject.franchiseId}:${subject.id}`;
  const pending = executor.useMoves((s) => s.drafting[key]);
  const error = executor.useMoves((s) => s.draftErrors[key]);
  const receipt = executor.useMoves((s) => s.moves.find((r) =>
    r.spec.franchiseId === subject.franchiseId && r.spec.subject.players.includes(subject.id) &&
    Boolean(rosterHandoffLabel(r.spec.intent))));
  const reason = executor.draftReason(subject);
  const status = executor.rosterStatus(subject);
  const eligibility = executor.eligibility(subject);
  const feeds = executor.eligibilityFeeds();
  const handoffKey = `handoff:${receipt?.correlationId}`;
  const opening = executor.useMoves((s) => s.drafting[handoffKey]);
  const handoffError = executor.useMoves((s) => s.draftErrors[handoffKey]);
  const taxiLabel = status === 'TAXI_SQUAD' ? 'Draft promotion from taxi' : 'Draft move to taxi squad';
  return (
    <section className="zone player-act">
      <h5>Act</h5>
      {status !== 'TAXI_SQUAD' && <Act
        verb="roster.ir"
        args={{ subject }}
        disabled={Boolean(reason) || eligibility?.ir.verdict === 'ineligible'}
        label={ineligibleLabel('Draft IR placement', eligibility?.ir)}
      >
        {pending ? 'Drafting…' : 'Draft IR placement'}
      </Act>}
      {(status === 'ROSTER' || status === 'TAXI_SQUAD') && <Act
        verb="roster.taxi"
        args={{ subject }}
        disabled={Boolean(reason) || eligibility?.taxi.verdict === 'ineligible'}
        label={ineligibleLabel(taxiLabel, eligibility?.taxi)}
      >
        {pending ? 'Drafting…' : taxiLabel}
      </Act>}
      {reason && <p>{reason}</p>}
      {eligibility && (
        <ul className="eligibility-lines">
          <li>{eligibilityLine('IR', eligibility.ir, feeds)}</li>
          <li>{eligibilityLine('Taxi', eligibility.taxi, feeds)}</li>
        </ul>
      )}
      {error && <p role="alert">{error}</p>}
      {receipt && <ReceiptStatus receipt={receipt} />}
      {handoffError && <p role="alert">{handoffError}</p>}
      {receipt?.state === 'ready' && <Act
        verb="move.handoff"
        args={{ correlationId: receipt.correlationId }}
        disabled={Boolean(opening)}
        label={opening ? 'Opening MFL…' : rosterHandoffLabel(receipt.spec.intent)}
      />}
      <Act verb="nav.open" args={{ node: 'hq', workspace: 'my-moves' }}>
        My moves
      </Act>
    </section>
  );
}
