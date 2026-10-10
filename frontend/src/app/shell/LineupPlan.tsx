import type { Receipt, Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { stateLabels } from './moveLabels';

export function lineupChanges(receipt: Receipt, snapshot: Snapshot) {
  const lineup = receipt.spec.expected.lineup;
  if (!lineup) return [];
  const directory = new Map(snapshot.players.value.map((p) => [p.id, p]));
  const changes = [
    ...lineup.starters.filter((id) => !lineup.baseline.includes(id)).map((id) => ({ id, verb: 'Start' })),
    ...lineup.baseline.filter((id) => !lineup.starters.includes(id)).map((id) => ({ id, verb: 'Bench' })),
  ];
  return changes.map(({ id, verb }) => ({
    id, verb, name: directory.get(id)?.name ?? id, position: directory.get(id)?.position ?? 'unknown',
  }));
}
export function LineupPlan({ receiptId, franchiseId, snapshot }: {
  receiptId: string | null; franchiseId: string; snapshot: Snapshot;
}) {
  const receipt = commands.useMoves((s) => s.moves.find((r) => r.correlationId === receiptId));
  const { draft } = commands.useLineupEdit();
  const pending = commands.useMoves((s) => s.drafting[`handoff:${receiptId}`]);
  const error = commands.useMoves((s) =>
    s.draftErrors[`lineup:${franchiseId}`] || s.draftErrors[`handoff:${receiptId}`]);
  const lineup = receipt?.spec.franchiseId === franchiseId ? receipt.spec.expected.lineup : undefined;
  if (draft?.franchiseId !== franchiseId && !lineup) return null;
  const note = receipt?.audit.at(-1)?.note;
  const handed = receipt?.audit.some((entry) => entry.to === 'handed_off');
  return <div className="lineup-plan" aria-label="Lineup plan" aria-live="polite">
    {error && <p role="alert">Lineup plan · {error}</p>}
    {!lineup && <p>
      Press Check and save plan when the lineup is right. TheWarRoom never changes MFL itself.
    </p>}
    {receipt && lineup && <>
      {receipt.state === 'blocked' ? <p>Not ready · {note}</p> : <>
        <h5>Saved plan · Week {lineup.week}</h5>
        <ul>{lineupChanges(receipt, snapshot).map((change) => (
          <li key={change.id}>{change.verb} {change.name} ({change.position})</li>
        ))}</ul>
        {receipt.state === 'ready' && <>
          <p>Make these changes on MFL; this plan does not submit them.</p>
          <Act verb="move.handoff" args={{ correlationId: receipt.correlationId }}
            disabled={Boolean(pending)} label={pending ? 'Opening MFL…' : 'Open MFL lineup page'} />
        </>}
        {handed && <p>
          Opened MFL's week {lineup.week} lineup page. Make these same changes there and press
          {' '}Submit Partial Lineup at the bottom.
          {' '}TheWarRoom will show Landed when MFL has them.
        </p>}
        {receipt.state !== 'ready' && <p>{stateLabels[receipt.state]} · {note}</p>}
      </>}
    </>}
  </div>;
}
