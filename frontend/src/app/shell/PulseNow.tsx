import type { PulseMatchup, PulseSide, Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { SignalChip } from '../look/Slots';
import { provenanceSlot } from '../cards/playerModel';
import { useReading } from './lineups';
import './pulse.css';

function SideSummary({ side }: { side: PulseSide }) {
  return <div className="pulse-summary">
    <h5>{side.name || side.franchiseId}</h5>
    <b className="pulse-score">{side.score.toFixed(1)}</b>
    <small>{side.playing} playing · {side.yetToPlay} yet to play</small>
  </div>;
}
function MatchupCard({ matchup, own, week }: { matchup: PulseMatchup; own: boolean; week: number }) {
  const key = `${week}:${matchup.home.franchiseId}:${matchup.away.franchiseId}`;
  const expanded = commands.usePulse()[key] ?? own;
  const controls = `pulse-${key}`;
  return <Card gravity="G0" header={<div className="pulse-sides">
    <SideSummary side={matchup.home} />
    <SideSummary side={matchup.away} />
  </div>} actionTray={<Act verb="pulse.toggle" args={{ matchup: key, expanded }}
    expanded={expanded} controls={controls}>
    {expanded ? 'Hide starters' : 'Show starters'}
  </Act>}>
    {expanded && <div id={controls} className="pulse-sides">
      {[matchup.home, matchup.away].map((side) => <ul key={side.franchiseId} aria-label={side.name}>
        {side.starters.map((player) => <li key={player.id}>
          <span>{player.name || player.id} ({player.position || '—'}, {player.team || '—'})</span>
          <b>{player.score.toFixed(1)}</b>
        </li>)}
      </ul>)}
    </div>}
  </Card>;
}
export default function PulseNow({ snapshot }: { snapshot: Snapshot }) {
  const franchiseId = commands.use().franchiseId;
  const { reading, error } = useReading(commands.pulse, 'now', snapshot);
  const freshness = reading?.provenance.freshness;
  const failed = freshness?.state === 'fail' || (freshness?.state === 'stale' && freshness.note);
  const reason = error || (failed && (freshness?.note || 'feed failed'));
  if (reason) return <p role="alert">Live scoring unavailable · {reason}</p>;
  if (!reading) return <p role="status">Reading live scoring…</p>;
  const own = (m: PulseMatchup) => m.home.franchiseId === franchiseId || m.away.franchiseId === franchiseId;
  const matchups = [...reading.matchups.filter(own), ...reading.matchups.filter((m) => !own(m))];
  return <section className="pulse-now" aria-label="League Pulse Now">
    <h4>{reading.week ? `Week ${reading.week}` : 'Week unknown'}</h4>
    <SignalChip {...provenanceSlot([['Live scoring', reading.provenance]])} />
    {!matchups.length && <p>No matchups in the held feed</p>}
    {matchups.map((matchup) => <MatchupCard
      key={`${matchup.home.franchiseId}:${matchup.away.franchiseId}`}
      matchup={matchup} own={own(matchup)} week={reading.week} />)}
  </section>;
}
