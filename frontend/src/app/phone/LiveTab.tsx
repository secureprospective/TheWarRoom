import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import { myChance, percent, pointsLabel } from '../data/nativePulse';
import { unavailable } from '../shell/hqModel';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import { myGame } from '../shell/homeTiles';
import type { LeagueChoice } from '../shell/leaguePicker';
import './LiveTab.css';

type BriefReader = (league: string, mine: string, port: HostedPort) => Promise<HomeBrief>;
const readLive: BriefReader = (league, mine, port) => readThinHomeBrief(league, mine, port);

/** LiveTab is the phone Live page: the mock's "My matchup" card on top (the same results and odds
 *  reads Today's "Live now" row uses), then League Pulse's Now view for games, starters and plays. */
export function LiveTab({ league, port = hostedWorkflow, readBrief = readLive, children }: {
  league?: LeagueChoice; port?: HostedPort; readBrief?: BriefReader; children?: ReactNode;
}) {
  const mine = league?.franchise ?? '';
  const [brief, setBrief] = useState<HomeBrief>();
  const [failure, setFailure] = useState('');
  useEffect(() => {
    let active = true;
    setBrief(undefined); setFailure('');
    if (!league || !mine) return;
    readBrief(league.id, mine, port)
      .then(found => { if (active) setBrief(found); })
      .catch(cause => { if (active) setFailure(cause instanceof Error ? cause.message : String(cause)); });
    return () => { active = false; };
  }, [league, mine, port, readBrief]);
  return <div className="live6" data-testid="live-tab">
    <section className="live6__mym" aria-label="My matchup">
      <div className="live6__head">
        <b>My matchup{brief?.results ? ` · wk ${brief.results.week}` : ''}</b></div>
      <Matchup league={league} mine={mine} brief={brief} failure={failure} />
    </section>
    {children}
  </div>;
}

function Matchup({ league, mine, brief, failure }: { league?: LeagueChoice; mine: string;
  brief?: HomeBrief; failure: string }) {
  const game = useMemo(() => brief && mine ? myGame(brief, mine) : undefined, [brief, mine]);
  const words = !league ? 'No admitted league'
    : !mine ? 'No team of yours in this league'
      : failure ? `${unavailable} · ${failure}`
        : !brief ? 'Reading…'
          : brief.errors.results ? `${unavailable} · ${brief.errors.results}`
            : !game ? brief.results?.bye === mine ? 'Bye week' : 'No game this week' : '';
  if (!game || !brief) return <p role={failure || brief?.errors.results ? 'alert' : 'status'}
    className="live6__note">{words}</p>;
  const home = game.home === mine;
  const score = (side: 'home' | 'away') => game[`${side}Points`] ?? game[`${side}Provisional`];
  const chance = myChance(brief.pulse?.odds, mine);
  const held = chance && chance.week === brief.results?.week ? chance.mine : undefined;
  return <div className="live6__mys">
    <div className="live6__t is-me"><span>{home ? game.homeName : game.awayName} · YOU</span>
      <b>{pointsLabel(score(home ? 'home' : 'away'))}</b></div>
    <div className="live6__wp">
      <div className="live6__bar">
        <i style={{ width: held === undefined ? 0 : `${Math.round(held * 100)}%` }} /></div>
      <div className="live6__wpl"><b>{held === undefined ? 'WIN —' : `${percent(held)} WIN`}</b>
        {held === undefined && <span>{brief.errors.pulse ?? 'No odds for this week'}</span>}
      </div>
    </div>
    <div className="live6__t is-r"><span>{home ? game.awayName : game.homeName}</span>
      <b>{pointsLabel(score(home ? 'away' : 'home'))}</b></div>
  </div>;
}

export default LiveTab;
