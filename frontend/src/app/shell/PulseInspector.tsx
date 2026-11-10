import { HQAct as PulseAct } from './HQAct';
import { myChance, percent, pointsLabel, type PulseResults } from '../data/nativePulse';
import type { NativePulse } from './pulseReads';
import { scoreOf } from './pulseModel';
import type { Node } from './nodes';

/** PulseInspector is the franchise card beside League Pulse (league-pulse-mock-v14 `.insp`). Every
 *  line comes from a league read already on the page; a line the league does not hold is left out. */
export function PulseInspector({ reading, results, franchise, mine, week, leave }: {
  reading: NativePulse; results?: PulseResults; franchise: string; mine: string; week?: number;
  leave: (node?: Node) => void;
}) {
  const seat = reading.seats.find(s => s.franchise === franchise);
  const row = reading.standings?.rows.find(r => r.franchise === franchise);
  const odds = reading.odds?.rows.find(r => r.franchise === franchise);
  const seed = reading.bracket?.fields.flatMap(f => f.seeds).find(s => s.franchise === franchise)?.seed;
  const game = results?.games.find(g => g.home === franchise || g.away === franchise);
  const opponent = game && (game.home === franchise ? game.away : game.home);
  const points = (id?: string) => { const s = scoreOf(results, id ?? '');
    return pointsLabel(s?.points ?? s?.provisional); };
  const chance = myChance(reading.odds, franchise);
  const sub = [row?.division ?? seat?.division, row?.record, seed ? `seed ${seed}` : '']
    .filter(Boolean).join(' · ');
  return <aside className="insp" aria-label="Franchise">
    <div className="ihead">
      <div className="top"><span className="on">FRANCHISE</span>
        {franchise === mine && <span className="mine">YOU</span>}</div>
      <div className="nm">{seat?.name ?? franchise}</div>
      <div className="sub">{sub || 'No standings are held yet'}</div>
    </div>
    <div className="nums">
      <div title={reading.odds?.powerWords}><b>{odds?.power ?? '—'}</b><span>POWER</span></div>
      <div title={reading.odds?.oddsWords}><b>{percent(odds?.playoffOdds)}</b>
        <span>PLAYOFF ODDS</span></div>
    </div>
    <div className="idrawers">
      <div className="idr open"><PulseAct>THIS WEEK<span className="c">{week ? `wk ${week}` : ''}</span>
      </PulseAct><div className="body">
        {game && opponent ? <>
          <div className="kv"><span>vs {game.home === franchise ? game.awayName : game.homeName}</span>
            <span>{points(franchise)} – {points(opponent)}</span></div>
          {chance && chance.week === week && !game.winner && <div className="kv"><span>win prob</span>
            <span>{percent(chance.mine)}</span></div>}
          {game.winner && <div className="kv"><span>result</span>
            <span>{game.winner === franchise ? 'won' : 'lost'}</span></div>}
        </> : <div className="kv"><span>{results?.bye === franchise ? 'bye week' : 'no game read'}</span>
          <span /></div>}
      </div></div>
      <div className="idr open"><PulseAct>SEASON<span className="c">
        {row ? `${row.wins + row.losses + row.ties} games` : ''}</span></PulseAct><div className="body">
        {row ? <>
          <div className="kv"><span>record</span><span>{row.record}</span></div>
          <div className="kv"><span>points for</span><span>{row.pointsFor.toFixed(1)}</span></div>
          <div className="kv"><span>standing</span><span>{row.rank} of {reading.standings?.rows.length}</span>
          </div>
        </> : <div className="kv"><span>{reading.standings?.words ?? 'standings not read'}</span><span /></div>}
      </div></div>
    </div>
    <div className="iact"><span className="h">GO TO · links, nothing executes here</span>
      <div className="ia">
        <PulseAct className="btn" onAction={() => leave('hq')}>ROSTER ▸</PulseAct>
        <PulseAct className="btn" onAction={() => leave('war')}>TRADE DESK ▸</PulseAct>
      </div></div>
    <div className="ifoot"><span>{reading.name}</span><span>{reading.season}</span></div>
  </aside>;
}
