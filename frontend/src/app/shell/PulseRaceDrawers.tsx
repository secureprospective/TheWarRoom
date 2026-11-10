import type { PulseBracket, PulseStandings } from '../data/nativePulse';
import { PulseDrawer } from './PulseDrawer';
import { PulseIdle } from './PulseNow';
import { bracketRounds, tieGroups, type PulseSeedRow } from './pulseModel';

/** TiebreakDrawer says how each tie on this side stands today, in the standings' own words. */
export function TiebreakDrawer({ open, onClose, standings, rows, side }: { open: boolean;
  onClose: () => void; standings?: PulseStandings; rows: PulseSeedRow[]; side: string }) {
  const ties = standings ? tieGroups(standings, rows) : [];
  return <PulseDrawer open={open} onClose={onClose} title="Tiebreakers"
    sub={`${side ? `${side} side · ` : ''}how ties resolve today`}>
    {!ties.length ? <PulseIdle words={standings ? 'No two franchises on this side are level on record.'
      : 'Reading the standings…'} />
      : <div className="scroll"><table className="ct ties"><thead><tr>
        <th>#</th><th>TIED TEAMS</th><th>STEP USED</th><th>DETAIL</th><th>ORDER</th>
      </tr></thead><tbody>
        {ties.map((tie, i) => <tr key={tie.detail}>
          <td>{i + 1}</td>
          <td>{tie.teams.map(t => t.name).join(' · ')} ({tie.teams[0].record})</td>
          <td>{tie.step}</td><td>{tie.detail}</td>
          <td>{tie.teams.map(t => t.seed ? `seed ${t.seed}` : 'out').join(' · ')}</td>
        </tr>)}
      </tbody></table></div>}
    {standings && <div className="drawer-note">Order: {standings.criteria.join(' → ')}.
      {standings.noData.length ? ` ${standings.noData.join(' ')}` : ''}
      {' '}League setting · Control Room › League.</div>}
  </PulseDrawer>;
}

/** BracketDrawer lays the playoffs out in the league's own bracket shape, from today's seeds. */
export function BracketDrawer({ open, onClose, bracket }: { open: boolean; onClose: () => void;
  bracket?: PulseBracket }) {
  const sides = bracket ? bracketRounds(bracket) : undefined;
  const columns = sides ? sides.fields.reduce((n, f) => n + f.rounds.length, sides.final ? 1 : 0) : 0;
  const field = (f: NonNullable<typeof sides>['fields'][number]) => f.rounds.map(round =>
    <div className="rd" key={`${f.conference}-${round.heading}`}>
      <span className="rh">{f.conference ? `${f.conference.toUpperCase()} · ` : ''}{round.heading}</span>
      {round.games.map(game => <div className="bm" key={game.id}>
        {[game.home, game.away].map((side, i) => <div key={i}>{side.words}
          {side.seed !== undefined && <b>{side.seed}</b>}</div>)}
      </div>)}
    </div>);
  return <PulseDrawer open={open} onClose={onClose} title="Bracket"
    sub={!bracket ? 'reading the playoffs' : !bracket.projected ? bracket.words
      : `projected from today's seeds · ${bracket.perConference} per side, ${bracket.byeSeeds} bye`}>
    {!sides || !columns ? <PulseIdle words={bracket?.words || 'Reading the playoffs…'} />
      : <div className="bk" style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
        {sides.fields.slice(0, 1).flatMap(field)}
        {sides.final && <div className="rd">
          <span className="rh">{sides.final.heading}</span>
          <div className="bm">{sides.final.sides.map(words => <div key={words}>{words}</div>)}</div>
        </div>}
        {sides.fields.slice(1).flatMap(field)}
      </div>}
  </PulseDrawer>;
}
