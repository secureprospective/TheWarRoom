import type { Provenance } from '../data/contract';
import { freshnessSignals } from '../cards/playerModel';
import { SignalChip } from '../look/Slots';

export function ClockSignal({ provenance }: { provenance: Provenance }) {
  const state = provenance.freshness.state;
  // A failed fixture still reads as fail: the failure is the news.
  const label = state !== 'fail' && provenance.kind === 'fixture' ? 'fixture' : state;
  const detail = `${provenance.source} · ${provenance.freshness.note}`;
  return <SignalChip signal={freshnessSignals[state]} label={label} detail={detail} />;
}
