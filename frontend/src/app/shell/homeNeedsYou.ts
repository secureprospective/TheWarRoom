import type { HostedPort } from '../commissioner/hostedRoom';
import type { People } from '../commissioner/people';
import { readAppLeaguePeople, type LeagueChoice } from './leaguePicker';
import { readHomeBrief, type HomeBrief } from './homeReads';
import type { Node } from './nodes';

const WEEK = 7 * 24 * 60 * 60 * 1000;

type NeedsYouKind = 'trade' | 'auction' | 'lineup' | 'deadline' | 'commissioner' | 'unavailable';
export type NeedsYouItem = {
  id: string;
  leagueId: string;
  leagueName: string;
  kind: NeedsYouKind;
  words: string;
  deadline?: string;
  screen: { node: Node; workspace?: string };
};
type NeedsYouPerson = Pick<People, 'league' | 'yourFranchise' | 'mayChangeRules'>;
type NeedsYouReaders = {
  readPeople?: (league: string) => Promise<NeedsYouPerson>;
  readBrief?: (league: string, mine: string, port: HostedPort, owner: boolean) => Promise<HomeBrief>;
  now?: number;
};

const briefReader = (league: string, mine: string, port: HostedPort, owner: boolean) =>
  readHomeBrief(league, mine, port, undefined, undefined, owner);

const RFC3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;
const oneLine = (words: string) => words.replace(/\s+/g, ' ').trim();
const instant = (value?: string) => value && RFC3339.test(value) && Number.isFinite(Date.parse(value))
  ? value : undefined;
const isExpectedEmpty = (source: string, words: string) =>
  (source === 'results' && words === 'No league week is open') ||
  (source === 'calendar' && words === 'The league season is not known');

function failedSources(brief: HomeBrief, mine: string, commissioner: boolean): string[] {
  const sources = [...(mine ? ['trades', 'bids'] : []), 'results', 'calendar',
    ...(commissioner ? ['jobs'] : [])];
  return [...new Set(sources.flatMap(source => {
    const words = brief.errors[source as keyof HomeBrief['errors']];
    return words && !isExpectedEmpty(source, words) ? [words] : [];
  }))];
}

function item(league: LeagueChoice, kind: NeedsYouKind, id: string, words: string,
  screen: NeedsYouItem['screen'], deadline?: string): NeedsYouItem {
  return { id: `${league.id}:${kind}:${id}`, leagueId: league.id, leagueName: league.name, kind,
    words: oneLine(words), screen, deadline: instant(deadline) };
}

function needsYouFor(league: LeagueChoice, mine: string, commissioner: boolean, brief: HomeBrief,
  now: number): NeedsYouItem[] {
  const out: NeedsYouItem[] = [];
  if (mine && brief.trades) {
    const trades = brief.trades;
    if (trades.league !== league.id || trades.franchise !== mine)
      throw new Error('Trades belong to another viewer');
    for (const offer of trades.offers) {
      if (offer.status !== 'offered' || !offer.parties.includes(mine) || offer.accepted.includes(mine))
        continue;
      const from = offer.parties.filter(party => party !== mine).map(party =>
        offer.teams.find(([franchise]) => franchise === party)?.[1] ?? party).join(' + ');
      out.push(item(league, 'trade', offer.id, `Trade offer from ${from} is waiting on your answer`,
        { node: 'war', workspace: 'trade-desk' }, offer.expiresAt));
    }
  }
  if (mine && brief.bids) {
    const bids = brief.bids;
    if (bids.league !== league.id || bids.franchise !== mine)
      throw new Error('Bids belong to another franchise');
    for (const player of bids.players) {
      const closes = instant(player.closesAt);
      if (!player.own || !closes || Date.parse(closes) <= now) continue;
      out.push(item(league, 'auction', player.player, `Your bid on ${player.name} closes ${player.closes}`,
        { node: 'war', workspace: 'valuations-and-pool' }, closes));
    }
  }
  if (mine) {
    const score = brief.results?.scores.find(row => row.franchise === mine);
    if (score?.lineupStatus === 'missed') {
      const week = brief.pulse?.week;
      const kickoff = brief.pulse?.fixtures.find(fixture => fixture.week === week)?.kickoff;
      out.push(item(league, 'lineup', String(week ?? 'current'), score.lineupWords,
        { node: 'hq', workspace: 'lineup-and-roster' }, kickoff));
    }
  }
  for (const event of brief.calendar ?? []) {
    const deadline = instant(event.closes || event.at);
    if (!deadline || Date.parse(deadline) <= now || Date.parse(deadline) - now > WEEK) continue;
    const timing = event.closes ? 'Window closes this week' : 'League deadline this week';
    const screen = event.id === 'lineups'
      ? { node: 'hq' as const, workspace: 'lineup-and-roster' }
      : { node: 'war' as const, workspace: 'trade-desk' };
    out.push(item(league, 'deadline', event.id, `${event.label} · ${timing}`, screen, deadline));
  }
  if (commissioner && brief.jobs) {
    if (brief.jobs.league !== league.id) throw new Error('Jobs belong to another league');
    for (const job of brief.jobs.jobs) {
      if (job.state !== 'blocked' && !(job.state === 'live' && job.closes)) continue;
      out.push(item(league, 'commissioner', job.id, `${job.name} · ${job.words}`,
        { node: 'control', workspace: 'league' }, job.closes));
    }
  }
  return out;
}

function compareNeedsYou(a: NeedsYouItem, b: NeedsYouItem): number {
  const dueA = a.deadline ? Date.parse(a.deadline) : Infinity;
  const dueB = b.deadline ? Date.parse(b.deadline) : Infinity;
  if (dueA !== dueB) return dueA < dueB ? -1 : 1;
  return a.leagueName.localeCompare(b.leagueName) || a.id.localeCompare(b.id);
}

/** Reads every native app league once in parallel and returns only the signed-in account's actions. */
export async function gatherNeedsYou(leagues: readonly LeagueChoice[], port: HostedPort,
  readers: NeedsYouReaders = {}): Promise<NeedsYouItem[]> {
  const readPeople = readers.readPeople ?? readAppLeaguePeople;
  const readBrief = readers.readBrief ?? briefReader;
  const now = readers.now ?? Date.now();
  const appLeagues = leagues.filter(league => league.app);
  const perLeague = await Promise.all(appLeagues.map(async league => {
    try {
      const people = await readPeople(league.id);
      if (people.league !== league.id) throw new Error('People belong to another league');
      const mine = people.yourFranchise;
      const commissioner = people.mayChangeRules;
      const brief = await readBrief(league.id, mine, port, Boolean(mine));
      const items = needsYouFor(league, mine, commissioner, brief, now);
      for (const failure of failedSources(brief, mine, commissioner))
        items.push(item(league, 'unavailable', failure, failure, { node: 'home' }));
      return items;
    } catch (error) {
      const words = error instanceof Error ? error.message : String(error);
      return [item(league, 'unavailable', league.id, words || 'League unavailable', { node: 'home' })];
    }
  }));
  return perLeague.flat().sort(compareNeedsYou);
}
