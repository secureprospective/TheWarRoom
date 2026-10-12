import type { Snapshot } from '../data/contract';
import type { EntitySubject } from './state';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { NoteEditor } from '../personal/PersonalLists';
import { useReading } from './lineups';
import { OfferCard } from './TradeDesk';

function OfferInspector({ entity, snapshot }: {
  entity: Extract<EntitySubject, { kind: 'offer' }>; snapshot: Snapshot;
}) {
  const { reading, error } = useReading(commands.trades, entity.franchiseId, snapshot);
  const offer = reading?.offers.find(o => o.tradeId === entity.id);
  return <section aria-label="Offer inspector">
    {error && <p role="alert">{error}</p>}
    {offer ? <OfferCard offer={offer} franchiseId={entity.franchiseId} />
      : <p>Offer not present in the held pending trades.</p>}
  </section>;
}
export default function EntityInspector({ entity, snapshot }: { entity: EntitySubject; snapshot: Snapshot }) {
  if (entity.kind === 'offer') return <OfferInspector entity={entity} snapshot={snapshot} />;
  if (entity.kind === 'pick') return <section aria-label="Pick inspector">
    <h5>{entity.name}</h5>
    <p>Held draft asset. Open Draft room for draft results and pick ownership.</p>
    <Act verb="nav.open" args={{ node: 'trade', workspace: 'draft-room' }}>Open Draft room</Act>
  </section>;
  const franchise = snapshot.franchises.value.find(f => f.id === entity.id);
  const count = snapshot.rosters.value.find(r => r.franchiseId === entity.id)?.players.length;
  return <section aria-label="Franchise inspector">
    <h5>{franchise?.name ?? 'Franchise name unavailable'}</h5>
    <p>{count ?? 'Unavailable'} rostered players</p>
    <Act verb="market" args={{ field: 'inspect', value: `team:${entity.id}` }}>Historical trade profile</Act>
    <NoteEditor key={entity.id} id={entity.id} entity="franchise" />
  </section>;
}
