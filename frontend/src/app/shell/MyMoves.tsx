import { lazy, Suspense, useEffect, useRef } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { FranchisePicker } from './FranchisePicker';
import { useHQMovePlans, mergeHQMoves } from './hqMovePlans';
import { HQMoves } from './HQMoves';
import { EnvelopeRail } from './EnvelopeRail';
import { useWorkAnchor } from './workScroll';

const PersonalLists = lazy(() => import('../personal/PersonalLists'));
const ReleasePlans = lazy(() => import('../contracts/ReleasePlans'));
const BidPlans = lazy(() => import('../pool/BidPlans'));
const ContractActs = lazy(() => import('../contracts/ContractActs'));

export default function MyMoves({ snapshot, executor = commands, appearance }: {
  snapshot: Snapshot;
  executor?: typeof commands;
  appearance?: 'hq';
}) {
  const root = useRef<HTMLElement>(null);
  useWorkAnchor(root, '.bid-plan');
  const s = executor.use();
  const fixture = executor.providerKind === 'fixture';
  const franchiseId = s.franchiseId ?? '';
  useEffect(() => {
    const read = () => { if (fixture || franchiseId) void executor.loadMoves(franchiseId); };
    read();
    return executor.onMovesChange(read);
  }, [executor, fixture, franchiseId]);
  const moves = executor.useMoves((state) => state.moves);
  const loading = executor.useMoves((state) => state.movesLoading[franchiseId]);
  const error = executor.useMoves((state) => state.movesErrors[franchiseId]);
  const demo = executor.useMoves((state) => state.demo);
  const plans = useHQMovePlans(franchiseId, appearance === 'hq' && !fixture && Boolean(franchiseId));
  const receipts = moves.filter((r) => r.spec.franchiseId === franchiseId);
  const visible = receipts.filter((r, i) => i < 15 || r.spec.intent === 'trade.propose');
  if (!fixture && !franchiseId) return <FranchisePicker snapshot={snapshot} />;
  if (appearance === 'hq') return <HQMoves snapshot={snapshot} available={loading === false && !error &&
    plans.reading?.errors.length === 0}
    reason={error || plans.error || plans.reading?.errors.join(' · ')}
    moves={mergeHQMoves(error ? [] : receipts.map(r => ({ id: r.correlationId, intent: r.spec.intent,
      state: r.state, note: r.audit.at(-1)?.note ?? '', at: r.audit.at(-1)?.at ?? '',
      players: r.spec.subject.players, handoff: () => { void executor.dispatch('move.handoff', {
        correlationId: r.correlationId }); } })), plans.reading?.moves ?? [])} />;
  return (
    <section ref={root} className="my-moves">
      <h4>My moves</h4>
      <Suspense fallback={null}><PersonalLists snapshot={snapshot} /></Suspense>
      {fixture && <p>Fixture · demo envelope · landing simulated from a real roster snapshot</p>}
      {loading !== false && <p role="status">Moves loading…</p>}
      {error && <p role="alert">{error}</p>}
      {fixture && demo && <EnvelopeRail receipt={demo.receipt} snapshot={snapshot} />}
      {!fixture && visible.map((receipt) => (
        <EnvelopeRail key={receipt.correlationId} receipt={receipt} snapshot={snapshot} />
      ))}
      {!fixture && receipts.length > visible.length && (
        <p>{receipts.length - visible.length} older moves this session not shown</p>
      )}
      {!fixture && !receipts.length && loading === false &&
        !error && <p>No moves drafted this session</p>}
      {!fixture && <Suspense fallback={null}>
        <BidPlans /><ReleasePlans mine={franchiseId} /><ContractActs mine={franchiseId} revealing />
      </Suspense>}
    </section>
  );
}
