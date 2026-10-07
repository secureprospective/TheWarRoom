import type { EnvelopeState, Receipt, Snapshot } from '../data/contract';
import { Card } from '../cards/Card';
import { signalClasses } from '../look/channels';
import { SignalChip } from '../look/Slots';
import { intentLabels, stateLabels, stateSignals } from './moveLabels';
import './moves.css';
import { Act } from '../commands/Act';
import { lineupChanges } from './LineupPlan';
import { tradeRailLabels } from './tradeLabels';

const time = new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' });
const mainPath: EnvelopeState[] = ['draft', 'ready', 'handed_off', 'not_yet_done', 'landed'];
const pending = new Set<EnvelopeState>(['dot_review', 'bid_pending', 'waiver_pending']);
const terminal = new Set<EnvelopeState>(['blocked', 'failed', 'stale', 'not_verified']);

export function railStages(receipt: Receipt): EnvelopeState[] {
  const path = [...mainPath];
  for (const entry of receipt.audit) {
    if (pending.has(entry.to) && !path.includes(entry.to)) {
      path.splice(path.indexOf(entry.from) + 1, 0, entry.to);
    }
  }
  if (!terminal.has(receipt.state)) return path;
  const reached = new Set(['draft', ...receipt.audit.map((entry) => entry.to)]);
  const last = path.reduce((index, stage, next) => reached.has(stage) ? next : index, 0);
  return [...path.slice(0, last + 1), receipt.state];
}

export function EnvelopeRail({ receipt, snapshot }: { receipt: Receipt; snapshot: Snapshot }) {
  const player = snapshot.players.value.find((p) => p.id === receipt.spec.expected.player);
  const lineup = receipt.spec.expected.lineup;
  const trade = tradeRailLabels(receipt, snapshot);
  const header = trade?.header || (lineup ? `Week ${lineup.week} lineup` :
    player?.name ?? receipt.spec.expected.player);
  const handoffLabel = trade ? 'Open MFL trade desk' : 'Open MFL page';
  const summary = lineupChanges(receipt, snapshot).map((change) =>
    `${change.verb === 'Start' ? '+' : '−'}${change.name.split(',')[0]}`).join(' ');
  const trail = new Map(receipt.audit.map((entry) => [entry.to, entry]));
  return (
    <Card
      gravity={receipt.spec.gravity}
      header={
        <div>
          <h4>{header}</h4>
          {(trade || lineup) && <p>{trade?.summary || summary}</p>}
          <span>{intentLabels[receipt.spec.intent] ?? receipt.spec.intent}</span>
          <span className="move-classification">{receipt.spec.gravity} · {receipt.spec.undo}</span>
        </div>
      }
    >
      <ol className="envelope-rail" aria-label="Move stages">
        {railStages(receipt).map((stage) => {
          const entry = trail.get(stage);
          const reached = stage === 'draft' || Boolean(entry);
          return (
            <li
              key={stage}
              className={terminal.has(stage) ? signalClasses[stateSignals[stage]] : undefined}
              data-reached={reached}
              aria-current={stage === receipt.state ? 'step' : undefined}
            >
              <SignalChip
                signal={reached ? stateSignals[stage] : 'grey'}
                label={stateLabels[stage]}
              />
              {entry && <time dateTime={entry.at}>{time.format(new Date(entry.at))}</time>}
              {entry && <p>{entry.note}</p>}
            </li>
          );
        })}
      </ol>
      {receipt.state === 'ready' && <p>Ready is not MFL-accepted</p>}
      {receipt.state === 'ready' && <Act
        verb="move.handoff" args={{ correlationId: receipt.correlationId }}
        label={lineup ? 'Open MFL lineup page' : handoffLabel}
      />}
    </Card>
  );
}
