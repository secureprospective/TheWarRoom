import { create } from 'zustand';
import { TargetAcquisition } from '../../../wailsjs/go/main/App';
import type { main } from '../../../wailsjs/go/models';
import { array, object, text, integer } from '../data/parseValues';
import { selectProvider } from '../data/provider';
import { parseReading } from './provider';
import { parseAssets, parseFigures, type Figure } from '../shell/tradeBuilder';
import type { Reading, Amount } from './contract';
import type { TradeAsset } from '../data/contract';

export type AcquisitionRequest = { mine: string; player?: string; bid?: string; cut?: string };
type Acquisition = {
  incoming: { player: string; bid: string; name: string }[]; cuts: TradeAsset[];
  player: string; seasons: Figure[]; team: Figure[]; deadCap: Amount; cutName: string;
  spot: string; reasons: string[];
};
export function parseAcquisition(v: unknown): Reading<Acquisition> {
  return parseReading(v, (v, p) => {
    const r = object(v, p, ['incoming', 'cuts', 'player', 'seasons', 'team', 'deadCap',
      'cutName', 'spot', 'reasons']);
    const money = object(r.deadCap, p, ['cents', 'text']);
    return { incoming: array(r.incoming, p, (v, p) => {
      const c = object(v, p, ['player', 'bid', 'name']);
      return { player: text(c.player, p), bid: text(c.bid, p), name: text(c.name, p) };
    }), cuts: parseAssets(r.cuts, p), player: text(r.player, p), seasons: parseFigures(r.seasons, p),
    team: parseFigures(r.team, p), cutName: text(r.cutName, p), spot: text(r.spot, p),
    reasons: array(r.reasons, p, text), deadCap: {
      cents: integer(money.cents, p, -Number.MAX_SAFE_INTEGER), text: text(money.text, p) } };
  });
}
export const useAcquisition = create<{
  request?: AcquisitionRequest; reading?: Reading<Acquisition>; error: string; busy: boolean;
}>(() => ({ error: '', busy: false }));
let sequence = 0;
export async function openAcquisition(request: AcquisitionRequest) {
  const ticket = ++sequence;
  useAcquisition.setState({ request, busy: true, error: '' });
  try {
    if (selectProvider().kind !== 'live') throw new Error('Acquisition preview needs the desktop app');
    const reading = parseAcquisition(await TargetAcquisition({ mine: request.mine,
      player: request.player ?? '', bid: request.bid ?? '', cut: request.cut ?? '' } as main.RawAcquisition));
    if (ticket !== sequence) return;
    if (reading.status !== 'ok') throw new Error(reading.absenceReason ?? 'Acquisition preview unavailable');
    useAcquisition.setState({ reading });
  } catch (cause) { if (ticket === sequence) useAcquisition.setState({ error: String(cause) }); }
  finally { if (ticket === sequence) useAcquisition.setState({ busy: false }); }
}

export function refreshAcquisition(mine: string) {
  return openAcquisition(useAcquisition.getState().request ?? { mine });
}
