import { array, object, text, timestamp } from '../data/parseValues';
import { selectProvider } from '../data/provider';
import { hostedWorkflow } from '../commissioner/api';
import type { WarRoomContext } from './warroomReads';
import { deadlineLabel } from '../clock/labels';
export type WarRoomEvent = { id: string; label: string; at?: string; closes?: string; live?: boolean };
export async function readWarRoomCalendar(c: WarRoomContext): Promise<WarRoomEvent[]> {
  if (!c.native) {
    const reading = await selectProvider().clock();
    if (reading.provenance.freshness.state === 'fail')
      throw new Error(reading.provenance.freshness.note || 'Calendar unavailable');
    return reading.value.deadlines;
  }
  const native = c.native;
  const { data } = await (native.port ?? hostedWorkflow)({ command: 'calendar.read', mode: 'read',
    arguments: { league: native.league, season: c.snapshot.league.value.season, franchise: c.mine,
      action: 'trade', localZone: Intl.DateTimeFormat().resolvedOptions().timeZone } });
  const r = object(data, 'Calendar', ['league', 'revision', 'clock', 'decision']);
  if (r.league !== native.league) throw new Error('Calendar belongs to another league');
  const clock = object(r.clock, 'Action calendar', ['windows', 'exceptions', 'tradeDeadline',
    'secondsToTradeDeadline']);
  const events: WarRoomEvent[] = array(clock.windows, 'Windows', (v, p) => {
    const w = object(v, p, ['id', 'opens', 'start', 'close', 'live', 'secondsToClose']);
    if (typeof w.live !== 'boolean') throw new Error('Unreadable window state');
    return { id: text(w.id, p), label: array(w.opens, p, text).map(deadlineLabel).join(' · '),
      at: timestamp(w.start, p),
      closes: w.close == null ? undefined : timestamp(w.close, p), live: w.live };
  });
  if (clock.tradeDeadline) events.push({ id: 'trade-deadline', label: 'Trade deadline', at:
    timestamp(clock.tradeDeadline, 'Trade deadline') });
  return events;
}
