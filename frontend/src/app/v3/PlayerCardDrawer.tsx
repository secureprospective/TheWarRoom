// Desktop v3 player card drawer: mounts the shared full player card inside the v3 bottom drawer.
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { NativeCut, NativeHQ } from '../commissioner/hqReading';
import { usePlayerCard } from '../shell/PlayerCard';
import { BDrawer } from './pulseDrawers';

export function PlayerCardDrawer({ id, reading, league, mine, port = hostedWorkflow, cut, close }: {
  id: string; reading: NativeHQ; league: string; mine: string; port?: HostedPort;
  cut?: NativeCut; close: () => void;
}) {
  const card = usePlayerCard({ id, reading, league, mine, port, cut });
  return <BDrawer title={card.title} sub={card.sub} close={close} tall tabs={card.tabs} side={card.side}>
    {card.main}
  </BDrawer>;
}
