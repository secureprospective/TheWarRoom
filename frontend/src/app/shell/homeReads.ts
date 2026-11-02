import type { HostedPort } from '../commissioner/hostedRoom';
import { parseHostedBids } from '../commissioner/hostedMarkets';
import { nativeHQCap } from '../commissioner/hqReading';
import { parseLedger, type Ledger } from '../commissioner/proxy';
import type { Snapshot } from '../data/contract';
import type { PulseResults } from '../data/nativePulse';
import { parseDraft, type Draft } from '../draftroom/draft';
import type { HQCap } from './hqModel';
import { readNativePulse, readPulseWeekResults, type NativePulse } from './pulseReads';
import { readWarRoomCalendar, type WarRoomEvent } from './warroomCalendar';

/** HomeBrief is every fact the Home brief shows. Each field is one existing read; a read that fails
 *  leaves its field empty and its words in errors, so one lost source never blanks the page. */
export type HomeBrief = {
  pulse?: NativePulse;
  results?: PulseResults;
  calendar?: WarRoomEvent[];
  runway?: HQCap[];
  ledger?: Ledger;
  bids?: ReturnType<typeof parseHostedBids>;
  draft?: Draft;
  errors: Partial<Record<HomeSource, string>>;
};
export type HomeSource = 'pulse' | 'results' | 'calendar' | 'cap' | 'ledger' | 'bids' | 'draft';

/** readHomeBrief calls the reads League Pulse, Franchise HQ, the War Room and the ledger already
 *  use. It adds no rule of its own: every value comes from the league's settings through them. */
export async function readHomeBrief(league: string, mine: string, port: HostedPort,
  publish?: (brief: HomeBrief) => void): Promise<HomeBrief> {
  const out: HomeBrief = { errors: {} };
  const send = () => publish?.({ ...out, errors: { ...out.errors } });
  const apply = async (source: HomeSource, read: () => Promise<void>) => {
    try { await read(); } catch (error) {
      out.errors[source] = error instanceof Error ? error.message : String(error);
    } finally { send(); }
  };
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  await apply('pulse', async () => { out.pulse = await readNativePulse(league, mine, port); });
  const season = out.pulse?.season;
  const week = out.pulse?.week;
  await Promise.all([
    apply('results', async () => {
      if (season === undefined || week === undefined) throw new Error('No league week is open');
      out.results = await readPulseWeekResults(league, season, week, port);
    }),
    apply('calendar', async () => {
      if (season === undefined) throw new Error('The league season is not known');
      const snapshot = { league: { value: { season } } } as unknown as Snapshot;
      out.calendar = await readWarRoomCalendar({ snapshot, mine, week: week ?? 0,
        native: { league, mine, port, leave: () => undefined } });
    }),
    apply('cap', async () => {
      const r = await port({ command: 'native.cap.check', mode: 'read',
        arguments: { league, viewerZone: zone } });
      out.runway = nativeHQCap(r.data, league, mine);
    }),
    apply('ledger', async () => {
      const r = await port({ command: 'commish.ledger.read', mode: 'read', arguments: { league } });
      const ledger = parseLedger(r.data);
      if (ledger.league !== league) throw new Error('Ledger belongs to another league');
      out.ledger = ledger;
    }),
    apply('bids', async () => {
      const r = await port({ command: 'bids.read', mode: 'read', arguments: { league, franchise: mine } });
      const bids = parseHostedBids(r.data);
      if (bids.league !== league || bids.franchise !== mine) throw new Error('Bids belong elsewhere');
      out.bids = bids;
    }),
    apply('draft', async () => {
      const r = await port({ command: 'draft.read', mode: 'read', arguments: { league, franchise: mine,
        season: 0, at: '0001-01-01T00:00:00Z' } });
      const draft = parseDraft(r.data);
      if (draft.league !== league) throw new Error('Draft belongs to another league');
      out.draft = draft;
    }),
  ]);
  return out;
}
