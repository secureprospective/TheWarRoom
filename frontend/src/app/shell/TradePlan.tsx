import type { Receipt } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { stateLabels } from './moveLabels';

export function TradePlan({ receipt }: { receipt: Receipt }) {
  const pending = commands.useMoves((s) => s.drafting[`handoff:${receipt.correlationId}`]);
  const error = commands.useMoves((s) => s.draftErrors[`handoff:${receipt.correlationId}`]);
  const note = receipt.audit.at(-1)?.note;
  const handed = !['landed', 'stale', 'failed'].includes(receipt.state) &&
    receipt.audit.some((entry) => entry.to === 'handed_off');
  return <div className="lineup-plan plan-actions" aria-label="Trade accept plan" aria-live="polite">
    {error && <p role="alert">Trade hand-off · {error}</p>}
    {receipt.state === 'blocked' ? <p>Not ready · {note}</p> : <>
      {receipt.state === 'ready' && <>
        <p>Accept this offer on MFL's trade desk; this plan does not accept it.</p>
        <Act verb="move.handoff" args={{ correlationId: receipt.correlationId }}
          disabled={Boolean(pending)} label={pending ? 'Opening MFL…' : 'Open MFL trade desk'} />
      </>}
      {handed && <p>
        Opened MFL's trade desk. Accept offer {receipt.spec.expected.trade?.tradeId} there.
        {' '}After you accept, the DOT votes and the commissioner approves; TheWarRoom shows Landed
        {' '}when the trade executes.
      </p>}
      {receipt.state !== 'ready' && <p>{stateLabels[receipt.state]} · {note}</p>}
    </>}
  </div>;
}
