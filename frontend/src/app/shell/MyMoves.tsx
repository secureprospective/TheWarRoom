import { useEffect } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { FranchisePicker } from './FranchisePicker';
import { EnvelopeRail } from './EnvelopeRail';

export default function MyMoves({ snapshot, executor = commands }: {
  snapshot: Snapshot;
  executor?: typeof commands;
}) {
  const s = executor.use();
  const fixture = executor.providerKind === 'fixture';
  const franchiseId = s.franchiseId ?? '';
  useEffect(() => {
    if (fixture || franchiseId) void executor.loadMoves(franchiseId);
  }, [executor, fixture, franchiseId]);
  const moves = executor.useMoves((state) => state.moves);
  const loading = executor.useMoves((state) => state.movesLoading[franchiseId]);
  const error = executor.useMoves((state) => state.movesErrors[franchiseId]);
  const demo = executor.useMoves((state) => state.demo);
  const receipts = moves.filter((r) => r.spec.franchiseId === franchiseId);
  const visible = receipts.slice(0, 15);
  if (!fixture && !franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return (
    <section className="my-moves">
      <h4>My moves</h4>
      {fixture && <p>Fixture · demo envelope · landing simulated from a real roster snapshot</p>}
      {loading !== false && <p role="status">Moves loading…</p>}
      {error && <p role="alert">{error}</p>}
      {fixture && demo && <EnvelopeRail receipt={demo.receipt} snapshot={snapshot} />}
      {!fixture && visible.map((receipt) => (
        <EnvelopeRail key={receipt.correlationId} receipt={receipt} snapshot={snapshot} />
      ))}
      {!fixture && receipts.length > 15 && (
        <p>{receipts.length - 15} older moves this session not shown</p>
      )}
      {!fixture && !receipts.length && loading === false &&
        !error && <p>No moves drafted this session</p>}
    </section>
  );
}
