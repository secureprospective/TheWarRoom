import type { PulseBracket, PulseStandings } from '../data/nativePulse';
import { PulseDrawer } from './PulseDrawer';
import { PulseIdle } from './PulseNow';
import { bracketRounds, tieGroups, tieStake, type PulseSeedRow } from './pulseModel';

/** TiebreakDrawer says how each tie on this side stands today at a glance: who is level, who came
 *  out ahead, the step that split them and what the tie decides. The full reason is the row's title. */
export function TiebreakDrawer({ open, onClose, standings, rows, side }: { open: boolean;
  onClose: () => void; standings?: PulseStandings; rows: PulseSeedRow[]; side: string }) {
  const ties = standings ? tieGroups(standings, rows) : [];
  return <PulseDrawer open={open} onClose={onClose} title="Tiebreakers"
    sub={`${side ? `${side} side · ` : ''}how ties resolve today`}>
    {!ties.length ? <PulseIdle words={standings ? 'No two franchises on this side are level on record.'
      : 'Reading the standings…'} />
      : <div className="scroll">{ties.map(tie => <div className="tie" key={tie.detail} title={tie.detail}>
        <span className="lv">{tie.teams[0].record} · {tie.teams.length} LEVEL</span>
        <span className="tm">{tie.teams.map((t, i) => <span key={t.franchise}>
          {i > 0 && <i>{tie.split ? '▸' : '='}</i>}
          {i === 0 && tie.split ? <b>{t.name}</b> : t.name}<small>{t.seed ? `#${t.seed}` : 'out'}</small>
        </span>)}</span>
        <span className="by">{tie.split ? `WON ON ${tie.step.toUpperCase()}` : 'STILL LEVEL'}
          <span>{tie.split ? tie.edge : 'league order holds'}</span></span>
        {tieStake(tie) ? <span className="stk">{tieStake(tie)}</span> : <span />}
      </div>)}</div>}
    {standings && <div className="drawer-note">Tie-break order: {standings.criteria.join(' → ')}.
      {standings.noData.length ? ` ${standings.noData.join(' ')}` : ''}</div>}
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
        {sides.fields.slice(1).flatMap(f => field(f).reverse())}
      </div>}
  </PulseDrawer>;
}
