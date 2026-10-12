import type { Receipt, Snapshot } from '../data/contract';
import { Card } from '../cards/Card';
import { signalClasses } from '../look/channels';
import { SignalChip } from '../look/Slots';
import { intentLabels, stateLabel, stateSignals, type RailState } from './moveLabels';
import './moves.css';
import { Act } from '../commands/Act';
import { lineupChanges } from './LineupPlan';
import { tradeRailLabels } from './tradeLabels';
import { rosterHandoffLabel } from './ReceiptStatus';

const time = new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' });
const mainPath: RailState[] = ['draft', 'ready', 'handed_off', 'not_yet_done', 'landed'];
const bidPath: RailState[] = ['draft', 'ready', 'handed_off', 'bid_pending', 'landed'];
const pending = new Set<RailState>(['dot_review', 'bid_pending', 'waiver_pending']);
const terminal = new Set<RailState>([
  'blocked', 'failed', 'stale', 'not_verified', 'awarded_off_app', 'withdrawn',
]);
const waiting = new Set<RailState>(['not_verified', 'not_yet_done']);

// What the strip reads of a plan: the move rail's receipts and bid plans both fit it.
export type RailReceipt = {
  state: RailState;
  spec: { intent: string };
  audit: { at: string; from: RailState; to: RailState; note: string }[];
};

export function railStages(receipt: RailReceipt): RailState[] {
  const path = [...(receipt.spec.intent === 'bid.submit' ? bidPath : mainPath)];
  for (const entry of receipt.audit) {
    if ((pending.has(entry.to) || waiting.has(entry.to)) && !path.includes(entry.to)) {
      path.splice(path.indexOf(entry.from) + 1, 0, entry.to);
    }
  }
  if (!terminal.has(receipt.state)) return path;
  const reached = new Set(['draft', ...receipt.audit.map((entry) => entry.to)]);
  const last = path.reduce((index, stage, next) => reached.has(stage) ? next : index, 0);
  const visited = path.slice(0, last + 1);
  return visited.at(-1) === receipt.state ? visited : [...visited, receipt.state];
}

export function RailStrip({ receipt }: { receipt: RailReceipt }) {
  const trail = new Map(receipt.audit.map((entry) => [entry.to, entry]));
  return (
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
              label={stateLabel(stage, receipt.spec.intent)}
            />
            {entry && <time dateTime={entry.at}>{time.format(new Date(entry.at))}</time>}
            {entry && <p>{/^superseded by \S+$/.test(entry.note)
              ? 'Replaced by a newer plan' : entry.note}</p>}
          </li>
        );
      })}
    </ol>
  );
}

export function EnvelopeRail({ receipt, snapshot }: { receipt: Receipt; snapshot: Snapshot }) {
  const player = snapshot.players.value.find((p) => p.id === receipt.spec.expected.player);
  const lineup = receipt.spec.expected.lineup;
  const trade = tradeRailLabels(receipt, snapshot);
  const header = trade?.header || (lineup ? `Week ${lineup.week} lineup` :
    player?.name ?? 'Player unavailable');
  const handoffLabel = rosterHandoffLabel(receipt.spec.intent, receipt.spec.expected.rosterStatus) ||
    (trade ? 'Open MFL trade desk' : 'Open MFL page');
  const lineupSummary = lineupChanges(receipt, snapshot).map((change) =>
    `${change.verb === 'Start' ? '+' : '−'}${change.name.split(',')[0]}`).join(' ');
  let summary = trade?.summary || lineupSummary;
  const shortName = player?.name?.split(',')[0];
  if (receipt.spec.intent === 'roster.ir') {
    const direction = receipt.spec.expected.rosterStatus === 'ROSTER' ? 'Activate' : 'IR';
    summary = shortName ? `${direction} ${shortName}` : '';
  }
  if (receipt.spec.intent === 'roster.taxi') {
    const direction = receipt.spec.expected.rosterStatus === 'TAXI_SQUAD' ? '+' : '−';
    summary = shortName ? `${direction}Taxi ${shortName}` : '';
  }
  return (
    <Card
      gravity={receipt.spec.gravity}
      header={
        <div>
          <h4>{header}</h4>
          {summary && <p>{summary}</p>}
          <span>{receipt.spec.intent === 'roster.ir' && receipt.spec.expected.rosterStatus === 'ROSTER'
            ? 'IR activation' : intentLabels[receipt.spec.intent] ?? receipt.spec.intent}</span>
          <span className="move-classification">{receipt.spec.gravity} · {receipt.spec.undo}</span>
        </div>
      }
    >
      <RailStrip receipt={receipt} />
      {['roster.ir', 'roster.taxi'].includes(receipt.spec.intent) &&
        !['landed', 'failed', 'withdrawn'].includes(receipt.state) && <Act verb="move.withdraw"
          args={{ correlationId: receipt.correlationId }}
          label={receipt.audit.some(entry => entry.to === 'handed_off')
            ? "I didn't submit this move" : 'Remove plan'} />}
      {receipt.state === 'ready' && <p>Ready is not MFL-accepted</p>}
      {receipt.state === 'ready' && receipt.spec.intent.startsWith('trade.') &&
        receipt.spec.intent !== 'trade.accept' ? <Act verb="trade.builder"
          args={{ action: 'handoff', mine: receipt.spec.franchiseId, id: receipt.correlationId }}
          label={receipt.spec.intent === 'trade.propose' ? "Copy the deal and open MFL's trade desk"
            : receipt.spec.intent === 'trade.reject' ? "Open MFL's trade desk to reject"
              : "Open MFL's trade desk to withdraw it"} />
        : receipt.state === 'ready' && <Act
        verb="move.handoff" args={{ correlationId: receipt.correlationId }}
        label={lineup ? 'Open MFL lineup page' : handoffLabel}
      />}
    </Card>
  );
}

