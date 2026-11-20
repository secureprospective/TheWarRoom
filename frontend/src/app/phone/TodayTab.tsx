import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { myChance, percent, pointsLabel } from '../data/nativePulse';
import { unavailable } from '../shell/hqModel';
import { gatherNeedsYou, type NeedsYouItem } from '../shell/homeNeedsYou';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import { myGame, shortTime } from '../shell/homeTiles';
import { HQAct } from '../shell/HQAct';
import type { LeagueChoice } from '../shell/leaguePicker';
import type { ThinHomeEntry } from '../shell/ThinHome';
import './TodayTab.css';

const Boards = lazy(() => import('../shell/HomeBrief'));
const HOUR = 60 * 60 * 1000;
type BriefReader = (league: string, mine: string, port: HostedPort, owner: boolean,
  commissioner: boolean) => Promise<HomeBrief>;
export type TodayLane = 'NOW' | 'TODAY' | 'THIS WEEK' | 'NO CLOCK' | 'NOT AVAILABLE';
const LANES: TodayLane[] = ['NOW', 'TODAY', 'THIS WEEK', 'NO CLOCK', 'NOT AVAILABLE'];
const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/** laneOf files an item by its server deadline: within three hours (or past) is NOW. */
export function laneOf(item: NeedsYouItem, now: number): TodayLane {
  if (item.kind === 'unavailable') return 'NOT AVAILABLE';
  if (!item.deadline) return 'NO CLOCK';
  const due = Date.parse(item.deadline);
  if (due - now <= 3 * HOUR) return 'NOW';
  return new Date(due).toDateString() === new Date(now).toDateString() ? 'TODAY' : 'THIS WEEK';
}

function entryFor(screen: NeedsYouItem['screen']): ThinHomeEntry | undefined {
  if (screen.node === 'home') return undefined;
  if (screen.node === 'hq') return screen.workspace === 'lineup-and-roster' ? 'lineup' : 'hq';
  if (screen.node === 'war') return screen.workspace === 'trade-desk' ? 'trades'
    : screen.workspace === 'valuations-and-pool' ? 'bids' : 'war';
  if (screen.node === 'pulse') return 'pulse';
  return 'control';
}

/** keyWord is the row's action word, as the mock varies it: a lineup problem is FIX, else OPEN.
 *  DONE belongs to rows that are a simple confirm; no read today offers a confirm verb. */
export function keyWord(item: NeedsYouItem): 'Fix' | 'Open' {
  return item.kind === 'lineup' ? 'Fix' : 'Open';
}

/** tomorrowMorning is the mock's "Snooze until tomorrow": 9 AM local the next day. */
export function tomorrowMorning(now: number): string {
  const at = new Date(now);
  at.setDate(at.getDate() + 1);
  at.setHours(9, 0, 0, 0);
  return at.toISOString();
}

/** Tab bar marks Today can fill from its own reads, for the league the bar is showing. */
export type TodayBadges = { team: number; moves: number; league: number; live: boolean };

/** badgesFor counts open jobs per tab for one league, as the mock's bar does. */
export function badgesFor(jobs: { item: NeedsYouItem; lane: TodayLane }[], league: string | undefined,
  playing: boolean): TodayBadges {
  const mine = jobs.filter(job => job.lane !== 'NOT AVAILABLE' && job.item.leagueId === league);
  const count = (kinds: NeedsYouItem['kind'][]) => mine.filter(job => kinds.includes(job.item.kind)).length;
  return { team: count(['lineup']), moves: count(['trade', 'auction']), league: count(['commissioner']),
    live: playing };
}

export function TodayTab({ leagues, onOpen, port = hostedWorkflow, readBrief, now, onBriefReady, onNow,
  current, onBadges }: {
  leagues: LeagueChoice[];
  onOpen: (league: LeagueChoice, entry: ThinHomeEntry) => void;
  port?: HostedPort;
  readBrief?: BriefReader;
  now?: number;
  onBriefReady?: (leagueId: string, mine: string, brief: HomeBrief) => void;
  /** onNow reports how many open jobs are in the NOW lane, for the Today tab badge. */
  onNow?: (count: number) => void;
  /** current is the league the tab bar is showing; onBadges reports its Team/Moves/League/Live marks. */
  current?: string;
  onBadges?: (badges: TodayBadges) => void;
}) {
  const appLeagues = useMemo(() => leagues.filter(league => league.app), [leagues]);
  const reader = useMemo<BriefReader>(() => readBrief ?? ((id, mine, hosted, _owner, isCommissioner) =>
    readThinHomeBrief(id, mine, hosted, undefined, isCommissioner)), [readBrief]);
  const readingTime = useMemo(() => now ?? Date.now(), [now]);
  const [briefs, setBriefs] = useState<Record<string, HomeBrief>>({});
  const [failures, setFailures] = useState<Record<string, string>>({});
  const [items, setItems] = useState<NeedsYouItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [snoozed, setSnoozed] = useState<Set<string>>(new Set());
  const [snoozeError, setSnoozeError] = useState('');

  useEffect(() => {
    let active = true;
    setBriefs({}); setFailures({}); setItems([]); setLoading(true);
    void gatherNeedsYou(appLeagues, port, {
      now: readingTime,
      readPeople: async id => {
        const league = appLeagues.find(choice => choice.id === id);
        if (!league) throw new Error('Admitted league is unavailable');
        return { league: id, yourFranchise: league.franchise ?? '', yourRole: league.role ?? '' };
      },
      readBrief: async (id, mine, hosted, owner, isCommissioner) => {
        try {
          const brief = await reader(id, mine, hosted, owner, isCommissioner);
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
    }).then(found => { if (active) setItems(found); })
      .catch(cause => { if (active) setFailures({ '*': String(cause) }); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [appLeagues, reader, onBriefReady, port, readingTime]);

  const jobs = items.filter(item => !snoozed.has(item.id))
    .map(item => ({ item, lane: laneOf(item, readingTime) }))
    .sort((a, b) => LANES.indexOf(a.lane) - LANES.indexOf(b.lane));
  const open = jobs.filter(job => job.lane !== 'NOT AVAILABLE').length;
  const nowCount = loading ? 0 : jobs.filter(job => job.lane === 'NOW').length;
  useEffect(() => { onNow?.(nowCount); }, [nowCount, onNow]);
  const snooze = (item: NeedsYouItem) => {
    const jobId = item.id.slice(`${item.leagueId}:commissioner:`.length);
    setSnoozeError('');
    void port({ command: 'job.snooze.set', mode: 'execute',
      arguments: { league: item.leagueId, jobId, until: tomorrowMorning(readingTime) } })
      .then(() => setSnoozed(current => new Set(current).add(item.id)))
      .catch(cause => setSnoozeError(cause instanceof Error ? cause.message : String(cause)));
  };
  const failed = Object.keys(failures).length > 0 || jobs.length > open;
  const first = appLeagues[0];
  const week = first && briefs[first.id]?.pulse?.week;
  const isPlaying = (league: LeagueChoice) => {
    const brief = briefs[league.id];
    const game = brief && league.franchise ? myGame(brief, league.franchise) : undefined;
    return Boolean(game && !game.winner
      && (game.homeProvisional !== undefined || game.awayProvisional !== undefined));
  };
  const playing = appLeagues.filter(isPlaying).length;
  const shown = current ?? first?.id;
  const shownPlaying = playing > 0 && appLeagues.some(league => league.id === shown && isPlaying(league));
  const marks = loading ? undefined : badgesFor(jobs, shown, shownPlaying);
  const markKey = marks && JSON.stringify(marks);
  useEffect(() => { if (marks) onBadges?.(marks); }, [markKey, onBadges]); // eslint-disable-line
  const go = (item: NeedsYouItem) => {
    const league = appLeagues.find(choice => choice.id === item.leagueId);
    const entry = entryFor(item.screen);
    if (league && entry) onOpen(league, entry);
  };
  return <div className="today6" data-testid="today-tab">
    <h2>{DAYS[new Date(readingTime).getDay()]} · {week !== undefined ? `week ${week}`
      : 'week unavailable'}</h2>
    <div className="today6__sub">{appLeagues.length} {appLeagues.length === 1 ? 'league' : 'leagues'}
      {' · '}{playing} playing now</div>
    <div className="today6__sec"><b>Needs you</b>
      <span>{loading ? 'reading' : open ? `${open} open` : ''}</span></div>
    <section className="today6__grp" aria-label="Needs you">
      {snoozeError && <p role="alert" className="today6__note">{snoozeError}</p>}
      {failures['*'] && <p role="alert" className="today6__note">{unavailable} · {failures['*']}</p>}
      {loading && <p role="status" className="today6__note">Reading your leagues…</p>}
      {!loading && !failed && jobs.length === 0 &&
        <div className="today6__clear"><i aria-hidden="true">✓</i>Nothing needs you right now.</div>}
      {!loading && failed && open === 0 && !failures['*'] &&
        <p role="alert" className="today6__note">
          {unavailable} · Some reads failed, so this list may be short.</p>}
      {jobs.map(({ item, lane }) => <div key={item.id}
        className={`today6__job${lane === 'NOW' ? ' is-now' : ''}`
        + `${lane === 'NOT AVAILABLE' ? ' is-off' : ''}`}>
        <span><small>{lane} · {item.leagueName.toUpperCase()}</small>
          <b>{lane === 'NOT AVAILABLE' ? unavailable : item.title ?? item.words}</b>
          <em>{lane === 'NOT AVAILABLE' ? item.words : item.detail
            ? item.detail : item.deadline ? `Due ${shortTime(item.deadline, readingTime)}`
              : 'No deadline from the league'}</em></span>
        {entryFor(item.screen) && <HQAct className="today6__key"
          label={`${keyWord(item)} ${item.words} in ${item.leagueName}`}
          onAction={() => go(item)}>{keyWord(item)}</HQAct>}
        {item.kind === 'commissioner' && <HQAct className="today6__key today6__clock"
          label="Snooze until tomorrow" onAction={() => snooze(item)}>
          <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true" fill="none" stroke="currentColor"
            strokeWidth="1.8"><circle cx="8" cy="8" r="6.2" /><path d="M8 4.5V8l2.5 1.6" /></svg></HQAct>}
      </div>)}
    </section>
    <div className="today6__sec"><b>Live now</b><span>your matchups</span></div>
    <section className="today6__grp" aria-label="Live now">
      {appLeagues.map(league => <LiveRow key={league.id} league={league} brief={briefs[league.id]}
        failure={failures[league.id]} onOpen={() => onOpen(league, 'pulse')} />)}
    </section>
    <div className="today6__sec"><b>My boards</b><span>your account layout</span></div>
    {!first && <p role="status" className="today6__note">
      {unavailable} · No admitted league for these boards.</p>}
    {first && !first.franchise &&
      <p role="status" className="today6__note">
        {unavailable} · No owner franchise is admitted for these boards.</p>}
    {first?.franchise && failures[first.id] &&
      <p role="alert" className="today6__note">{unavailable} · {failures[first.id]}</p>}
    {first?.franchise && !failures[first.id] && !briefs[first.id] &&
      <p role="status" className="today6__note">Reading your boards…</p>}
    {first?.franchise && briefs[first.id] && <div className="today6__boards">
      <Suspense fallback={<p role="status" className="today6__note">Opening your boards…</p>}>
        <Boards league={first.id} mine={first.franchise} port={port} initial={briefs[first.id]}
          owner accountSync />
      </Suspense></div>}
  </div>;
}

function LiveRow({ league, brief, failure, onOpen }: { league: LeagueChoice; brief?: HomeBrief;
  failure?: string; onOpen: () => void }) {
  const mine = league.franchise ?? '';
  const game = brief && mine ? myGame(brief, mine) : undefined;
  // A refused or failed read is never shown as error text: the row offers to switch to that league.
  if (failure) return <HQAct className="today6__live is-quiet" label={`Open ${league.name} to see scores`}
    onAction={onOpen}><span><b>{league.name}</b><em>Open to see scores</em></span><b></b><i>›</i></HQAct>;
  const words = !brief ? 'Reading…'
      : !mine ? 'No team of yours in this league'
        : brief.errors.results ? `${unavailable} · ${brief.errors.results}`
          : !game ? brief.results?.bye === mine ? 'Bye week' : 'No game this week'
            : '';
  if (!game || !brief) return <div className="today6__live is-quiet">
    <span><b>{league.name}</b><em>{words}</em></span><b></b><i>–</i></div>;
  const home = game.home === mine;
  const score = (side: 'home' | 'away') => game[`${side}Points`] ?? game[`${side}Provisional`];
  const chance = myChance(brief.pulse?.odds, mine);
  const shown = chance && chance.week === brief.results?.week ? percent(chance.mine) : '–';
  return <HQAct className="today6__live" label={`Open Live for ${league.name}`} onAction={onOpen}>
    <span><b>{league.name}</b>
      <em>{home ? game.homeName : game.awayName} vs {home ? game.awayName : game.homeName}</em></span>
    <b>{pointsLabel(score(home ? 'home' : 'away'))}–{pointsLabel(score(home ? 'away' : 'home'))}</b>
    <i>{shown}</i>
  </HQAct>;
}

export default TodayTab;
