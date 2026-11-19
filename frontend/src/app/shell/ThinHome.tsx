import { useEffect, useMemo, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { unavailable } from './hqModel';
import { gatherNeedsYou, type NeedsYouItem } from './homeNeedsYou';
import { readThinHomeBrief, type HomeBrief } from './homeReads';
import { shortTime } from './homeTiles';
import { homeRoute, type Route } from './nodes';
import { HQAct } from './HQAct';
import { leagueInitials, type LeagueChoice } from './leaguePicker';
import './thinHome.css';

export const thinHomeRoute: Route = homeRoute;
type BriefReader = (league: string, mine: string, port: HostedPort, owner: boolean,
  commissioner: boolean) => Promise<HomeBrief>;
export type ThinHomeEntry = 'home' | 'hq' | 'war' | 'pulse' | 'league' | 'control'
  | 'lineup' | 'bids' | 'trades';
type ThinHomeProps = {
  leagues: LeagueChoice[];
  onOpen: (league: LeagueChoice, entry: ThinHomeEntry) => void;
  port?: HostedPort;
  readBrief?: BriefReader;
  now?: number;
  onBriefReady?: (leagueId: string, mine: string, brief: HomeBrief) => void;
};
type OpenShortcut = { label: string; entry: ThinHomeEntry; commissionerOnly?: boolean };
const shortcuts: OpenShortcut[] = [
  { label: 'Franchise HQ', entry: 'hq' }, { label: 'War Room', entry: 'war' },
  { label: 'Pulse', entry: 'pulse' }, { label: 'Control Room', entry: 'control', commissionerOnly: true },
];
const commissioner = (role?: string) => role === 'Commissioner' || role === 'Co-commissioner';
function needsYouEntry(screen: NeedsYouItem['screen']): ThinHomeEntry | undefined {
  if (screen.node === 'home') return undefined;
  if (screen.node === 'hq') return screen.workspace === 'lineup-and-roster' ? 'lineup' : 'hq';
  if (screen.node === 'war') return screen.workspace === 'trade-desk' ? 'trades'
    : screen.workspace === 'valuations-and-pool' ? 'bids' : 'war';
  if (screen.node === 'pulse') return 'pulse';
  return 'control';
}
export function ThinHome({ leagues, onOpen, port = hostedWorkflow, readBrief, now,
  onBriefReady }: ThinHomeProps) {
  const appLeagues = useMemo(() => leagues.filter(league => league.app), [leagues]);
  const briefReader = useMemo<BriefReader>(() => readBrief ?? ((_id, mine, hosted, _owner, isCommissioner) =>
    readThinHomeBrief(_id, mine, hosted, undefined, isCommissioner)),
  [appLeagues, readBrief]);
  const readingTime = useMemo(() => now ?? Date.now(), [now]);
  const [briefs, setBriefs] = useState<Record<string, HomeBrief>>({});
  const [failures, setFailures] = useState<Record<string, string>>({});
  const [needsYou, setNeedsYou] = useState<NeedsYouItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    setBriefs({});
    setFailures({});
    setNeedsYou([]);
    setLoading(true);
    void gatherNeedsYou(appLeagues, port, {
      now: readingTime,
      readPeople: async id => {
        const league = appLeagues.find(choice => choice.id === id);
        if (!league) throw new Error('Admitted league is unavailable');
        return { league: id, yourFranchise: league.franchise ?? '', yourRole: league.role ?? '' };
      },
      readBrief: async (id, mine, hosted, owner, isCommissioner) => {
        try {
          const brief = await briefReader(id, mine, hosted, owner, isCommissioner);
          if (active) {
            setBriefs(current => ({ ...current, [id]: brief }));
            onBriefReady?.(id, mine, brief);
          }
          return brief;
        } catch (cause) {
          if (active) setFailures(current => ({ ...current,
            [id]: cause instanceof Error ? cause.message : String(cause) }));
          throw cause;
        }
      },
    }).then(items => { if (active) setNeedsYou(items); })
      .catch(cause => { if (active) setFailures({ '*': String(cause) }); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [appLeagues, briefReader, onBriefReady, port, readingTime]);

  const openFrom = (item: NeedsYouItem) => {
    const league = appLeagues.find(choice => choice.id === item.leagueId);
    const entry = needsYouEntry(item.screen);
    if (league && entry) onOpen(league, entry);
  };
  return <section className="thin-home" aria-label="Home" data-testid="thin-home">
    <header className="thin-home__header">
      <div><span className="thin-home__eyebrow">THE WAR ROOM · THIN MODE</span>
        <h1>Home</h1></div>
      <span className="thin-home__count">{appLeagues.length}{' '}
        {appLeagues.length === 1 ? 'LEAGUE' : 'LEAGUES'}</span>
    </header>
    <div className="thin-home__content">
      <section className="thin-home__needs" aria-labelledby="thin-home-needs-title">
        <div className="thin-home__section-head"><h2 id="thin-home-needs-title">NEEDS YOU</h2>
          <span>{loading ? 'READING' : `${needsYou.length} ITEMS`}</span></div>
        {failures['*'] && <p role="alert">{unavailable} · {failures['*']}</p>}
        {loading && <p role="status">Reading your league…</p>}
        {!loading && needsYou.length === 0 &&
          <p className="thin-home__empty">Nothing needs you right now.</p>}
        <ol className="thin-home__items" aria-label="Needs-you items">
          {needsYou.map(item => <li key={item.id}>
            <HQAct className={`thin-home__need${item.kind === 'unavailable' ? ' is-unavailable' : ''}`}
              disabled={!needsYouEntry(item.screen)} label={`Open ${item.words} in ${item.leagueName}`}
              onAction={() => openFrom(item)}>
              <span className="thin-home__need-league">{item.leagueName}</span>
              <span className="thin-home__need-words">{item.kind === 'unavailable'
                ? `${unavailable} · ${item.words}` : item.words}</span>
              {item.deadline && <time dateTime={item.deadline}>{shortTime(item.deadline, readingTime)}</time>}
            </HQAct>
          </li>)}
        </ol>
      </section>
      <section className="thin-home__leagues" aria-labelledby="thin-home-leagues-title">
        <div className="thin-home__section-head"><h2 id="thin-home-leagues-title">YOUR LEAGUES</h2>
          <span>ADMITTED MEMBERSHIPS</span></div>
        {!appLeagues.length && <p className="thin-home__empty">No admitted leagues.</p>}
        <div className="thin-home__cards">
          {appLeagues.map(league => <LeagueCard key={league.id} league={league}
            brief={briefs[league.id]} failure={failures[league.id]} now={readingTime} onOpen={onOpen} />)}
        </div>
      </section>
    </div>
  </section>;
}

function LeagueCard({ league, brief, failure, now, onOpen }: { league: LeagueChoice; brief?: HomeBrief;
  failure?: string; now: number; onOpen: ThinHomeProps['onOpen'] }) {
  const record = brief?.pulse?.standings?.rows.find(row => row.franchise === league.franchise)?.record;
  const week = brief?.pulse?.week;
  const pulseFailure = brief?.errors.pulse || brief?.pulse?.errors[0];
  const calendarFailure = brief?.errors.calendar;
  const next = (brief?.calendar ?? []).flatMap(event => {
    const at = event.closes || event.at;
    return at && Date.parse(at) > now ? [{ event, at }] : [];
  }).sort((a, b) => Date.parse(a.at) - Date.parse(b.at))[0];
  const showControl = commissioner(league.role);
  const actions = shortcuts.filter(shortcut => !shortcut.commissionerOnly || showControl);
  return <article className="thin-home__card">
    <div className="thin-home__card-head">
      <HQAct className="thin-home__league-open" label={`Open ${league.name}`}
        onAction={() => onOpen(league, 'hq')}>
        <span className="thin-home__crest" aria-hidden="true">{leagueInitials(league.name)}</span>
        <span className="thin-home__league-name">{league.name}</span>
        <span className="thin-home__open-mark">OPEN ›</span>
      </HQAct>
    </div>
    <div className="thin-home__facts">
      {league.franchiseName && <p><span>MY TEAM</span><b>{league.franchiseName}</b></p>}
      {record && <p><span>RECORD</span><b>{record}</b></p>}
      {!record && week !== undefined && <p><span>WEEK</span><b>{week}</b></p>}
      {pulseFailure && <p className="thin-home__unavailable"><span>LEAGUE READ</span>
        <b>{unavailable} · {pulseFailure}</b></p>}
      {calendarFailure
        ? <p className="thin-home__unavailable"><span>NEXT DEADLINE</span>
          <b>{unavailable} · {calendarFailure}</b></p>
        : next
          ? <p><span>NEXT DEADLINE</span><b>{next.event.label} · {shortTime(next.at, now)}</b></p>
          : brief?.calendar && <p><span>NEXT DEADLINE</span><b>None scheduled</b></p>}
      {failure && <p role="alert" className="thin-home__unavailable"><span>LEAGUE READ</span>
        <b>{unavailable} · {failure}</b></p>}
      {!brief && !failure && <p role="status" className="thin-home__reading">Reading league details…</p>}
    </div>
    <nav className="thin-home__shortcuts" aria-label={`${league.name} shortcuts`}>
      {actions.map(shortcut => <HQAct key={shortcut.label} className="thin-home__shortcut"
        label={`Open ${shortcut.label} for ${league.name}`}
        onAction={() => onOpen(league, shortcut.entry)}>
        {shortcut.label}<span aria-hidden="true">›</span>
      </HQAct>)}
    </nav>
  </article>;
}

export default ThinHome;
