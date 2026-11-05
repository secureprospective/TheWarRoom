import type { HostedPort } from '../commissioner/hostedRoom';
import { parseHostedBids } from '../commissioner/hostedMarkets';
import { nativeHQAssets, nativeHQCap } from '../commissioner/hqReading';
import { parseJobs, type Jobs } from '../commissioner/jobs';
import { researchAPI, researchAvailable, type ResearchReader } from '../research/api';
import type { Research } from '../research/contract';
import { formPoints } from './hqPlayerData';
import { parseLedger, type Ledger } from '../commissioner/proxy';
import type { Snapshot } from '../data/contract';
import type { PulseResults } from '../data/nativePulse';
import { parseDraft, type Draft } from '../draftroom/draft';
import type { HQCap } from './hqModel';
import { readNativePulse, readPulseMatchup, readPulseWeekResults, type NativePulse, type PulseMatchup }
  from './pulseReads';
import { readWarRoomCalendar, type WarRoomEvent } from './warroomCalendar';

/** Newest ledger rows Home reads and draws. */
export const HOME_LEDGER_ROWS = 12;

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
  /** My game's lineups (who is locked) and starter projections: League Pulse's matchup read. */
  matchup?: PulseMatchup;
  /** League assets: contract ends for the cap notes and ownership for the news tags. */
  assets?: Snapshot;
  /** The commissioner jobs board: live and blocked jobs are the league's alerts. */
  jobs?: Jobs;
  /** Player research (NFL news, depth charts): the same read the Research screen calls. */
  research?: Research;
  /** Each of my starters' last five finished weeks, from the research player read HQ's form chart uses. */
  form?: { player: string; name: string; position: string; weeks: { week: number; points: number }[] }[];
  /** When each source last answered (ISO); a tile's footer shows the oldest of its sources. */
  at: Partial<Record<HomeSource, string>>;
  errors: Partial<Record<HomeSource, string>>;
};
export type HomeSource = 'pulse' | 'results' | 'calendar' | 'cap' | 'ledger' | 'bids' | 'draft'
  | 'matchup' | 'assets' | 'jobs' | 'research' | 'form';

export const defaultResearch = (): ResearchReader | undefined => researchAvailable() ? researchAPI : undefined;
export const OWNER_ONLY_BIDS = 'Bids are visible only to this franchise\'s owner';

/** readHomeBrief calls the reads League Pulse, Franchise HQ, the War Room and the ledger already
 *  use. It adds no rule of its own: every value comes from the league's settings through them. */
export async function readHomeBrief(league: string, mine: string, port: HostedPort,
  publish?: (brief: HomeBrief) => void,
  research: ResearchReader | undefined = defaultResearch(), owner = true): Promise<HomeBrief> {
  const out: HomeBrief = { at: {}, errors: {} };
  const send = () => publish?.({ ...out, at: { ...out.at }, errors: { ...out.errors } });
  const apply = async (source: HomeSource, read: () => Promise<void>) => {
    try { await read(); out.at[source] = new Date().toISOString(); delete out.errors[source]; } catch (error)
      {
      out.errors[source] = error instanceof Error ? error.message : String(error);
    } finally { send(); }
  };
  const noResearch = 'Player research requires the desktop app';
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  // Standings and odds can wait on season scoring. They must not hold unrelated Home tiles hostage.
  let ready!: () => void;
  const weekReady = new Promise<void>(resolve => { ready = resolve; });
  const pulse = apply('pulse', async () => {
    out.pulse = await readNativePulse(league, mine, port, reading => {
      out.pulse = reading;
      out.at.pulse = new Date().toISOString();
      send();
      if (reading.week !== undefined) ready();
    });
  }).finally(ready);
  await Promise.all([
    pulse,
    apply('results', async () => {
      await weekReady;
      const season = out.pulse?.season, week = out.pulse?.week;
      if (season === undefined || week === undefined) throw new Error('No league week is open');
      out.results = await readPulseWeekResults(league, season, week, port);
    }).then(() => apply('matchup', async () => {
      const season = out.pulse?.season, week = out.pulse?.week;
      const g = out.results?.games.find(x => x.home === mine || x.away === mine);
      if (!g || season === undefined || week === undefined) throw new Error('No game this week');
      const starters = (f: string) => out.results?.scores.find(s => s.franchise === f)?.starters
        .map(s => s.player) ?? [];
      out.matchup = await readPulseMatchup(league, season, week, [g.home, g.away],
        { [g.home]: starters(g.home), [g.away]: starters(g.away) }, port);
      if (out.matchup.errors.length) throw new Error(out.matchup.errors[0]);
    })).then(() => apply('form', async () => {
      if (!research?.player) throw new Error(noResearch);
      const mineStarters = out.results?.scores.find(s => s.franchise === mine)?.starters ?? [];
      if (!mineStarters.length) throw new Error('No starters this week to read form for');
      out.form = await Promise.all(mineStarters.map(async s => ({ player: s.player, name: s.name,
        position: s.position ?? '', weeks: formPoints(await research.player!(s.player)).slice(-5) })));
    })),
    apply('assets', async () => {
      const r = await port({ command: 'native.assets.read', mode: 'read', arguments: { league } });
      out.assets = nativeHQAssets(r.data, league);
    }),
    apply('jobs', async () => {
      const r = await port({ command: 'commish.jobs.read', mode: 'read', arguments: { league } });
      const jobs = parseJobs(r.data);
      if (jobs.league !== league) throw new Error('Jobs belong to another league');
      out.jobs = jobs;
    }),
    apply('research', async () => {
      if (!research) throw new Error(noResearch);
      // The Research screen's own opening scope: every player, depth by team.
      out.research = await research.read({ mine: '', search: '', team: '', position: '', group: 'QB',
        trend: 'topAdds', depth: 'team', injuries: 'all' });
    }),
    apply('calendar', async () => {
      await weekReady;
      const season = out.pulse?.season, week = out.pulse?.week;
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
      // Home shows a page of the newest entries, never the whole book (F12, run-1104-1415).
      const r = await port({ command: 'commish.ledger.filtered.read', mode: 'read', arguments: { league,
        filter: { person: '', kind: '', from: '', before: '', last: HOME_LEDGER_ROWS } } });
      const ledger = parseLedger(r.data);
      if (ledger.league !== league) throw new Error('Ledger belongs to another league');
      out.ledger = { ...ledger, entries: ledger.entries.slice().reverse() }; // newest first, once
    }),
    apply('bids', async () => {
      // Bids are owner-only on the server; viewing another franchise must not ask for them.
      if (!owner) throw new Error(OWNER_ONLY_BIDS);
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

/** A read that failed because the desktop was still starting (a timed-out or not-yet-ready store)
 *  will answer once startup work lands; Home re-reads for these instead of keeping the error. */
const TRANSIENT = /deadline exceeded|timed out|timeout|startup|not ready|not open yet/i;
export const transientErrors = (b: HomeBrief): boolean =>
  Object.values(b.errors).some(e => e !== undefined && TRANSIENT.test(e));

/** retryDelay backs off Home's re-reads: 5 s, 10 s, 20 s, then every 60 s while still failing. */
export const retryDelay = (attempt: number): number => Math.min(60_000, 5_000 * 2 ** Math.max(0, attempt));

/** mergeBrief lays a re-read over the brief on screen: a source keeps its last answer until the
 *  re-read answers it, and an error clears once its source answers. */
export function mergeBrief(prev: HomeBrief, next: HomeBrief): HomeBrief {
  const out: HomeBrief = { ...prev, at: { ...prev.at, ...next.at }, errors: { ...prev.errors, ...next.errors } };
  for (const [k, v] of Object.entries(next) as [keyof HomeBrief, unknown][])
    if (k !== 'at' && k !== 'errors' && v !== undefined) (out as Record<string, unknown>)[k] = v;
  for (const source of Object.keys(next.at) as HomeSource[]) delete out.errors[source];
  return out;
}
