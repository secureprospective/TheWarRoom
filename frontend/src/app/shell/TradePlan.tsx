import type { Receipt } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { stateLabels } from './moveLabels';

export function TradePlan({ receipt }: { receipt: Receipt }) {
  const mine = receipt.spec.franchiseId;
  const exit = ['trade.reject', 'trade.revoke'].includes(receipt.spec.intent);
  const label = receipt.spec.intent === 'trade.reject'
    ? "Open MFL's trade desk to reject" : "Open MFL's trade desk to withdraw it";
  const pending = commands.useMoves((s) => s.drafting[`handoff:${receipt.correlationId}`]);
  const error = commands.useMoves((s) => s.draftErrors[`handoff:${receipt.correlationId}`]);
  const note = receipt.audit.at(-1)?.note;
  const handed = !['landed', 'stale', 'failed'].includes(receipt.state) &&
    receipt.audit.some((entry) => entry.to === 'handed_off');
  return <div className="lineup-plan plan-actions"
    aria-label={exit ? 'Trade reject or withdraw plan' : 'Trade accept plan'} aria-live="polite">
    {error && <p role="alert">Trade hand-off · {error}</p>}
    {receipt.state === 'blocked' ? <p>Not ready · {note}</p> : <>
      {receipt.state === 'ready' && <>
        <p>{exit ? 'Complete this yourself on MFL; the app does not reject or withdraw offers.'
          : "Accept this offer on MFL's trade desk; this plan does not accept it."}</p>
        {receipt.spec.intent === 'trade.revoke' && <p>Warning: MFL's withdraw link acts at once.
          Do not withdraw a deal already accepted on ProBoards.</p>}
        {exit ? <Act verb="trade.builder" args={{ action: 'handoff', mine, id: receipt.correlationId }}
          label={label} disabled={Boolean(pending)} /> : <Act verb="move.handoff"
          args={{ correlationId: receipt.correlationId }}
          disabled={Boolean(pending)} label={pending ? 'Opening MFL…' : 'Open MFL trade desk'} />}
      </>}
      {handed && !exit && <p>
        Opened MFL's trade desk. Accept offer {receipt.spec.expected.trade?.tradeId} there.
        {' '}After you accept, the DOT votes and the commissioner approves; TheWarRoom shows Landed
        {' '}when the trade executes.
      </p>}
      {receipt.state !== 'ready' && <p>{stateLabels[receipt.state]} · {note}</p>}
    </>}
  </div>;
}
