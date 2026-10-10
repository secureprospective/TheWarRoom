import type { Receipt } from '../data/contract';
import { SignalChip } from '../look/Slots';
import { stateLabels, stateSignals } from './moveLabels';

export default function ReceiptStatus({ receipt }: { receipt: Receipt }) {
  return (
    <div role="status">
      <SignalChip signal={stateSignals[receipt.state]} label={stateLabels[receipt.state]} />
      <p>{receipt.audit.at(-1)?.note}</p>
    </div>
  );
}

export function rosterHandoffLabel(intent: string, status?: string): string | undefined {
  if (intent === 'roster.ir') return status === 'ROSTER'
    ? 'Open MFL IR page for activation' : 'Open MFL IR page';
  if (intent === 'roster.taxi') return 'Open MFL taxi page';
  return undefined;
}
