import { useEffect, useRef } from 'react';
import type { Alert, Snapshot, Deadline } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { provenanceSlot } from '../cards/playerModel';
import { connectClock } from '../clock/ticker';
import { countdown } from '../clock/urgency';
import type { ClockProps } from '../clock/labels';
import { deadlineRoute } from '../clock/deadlineRoute';
import { deadlineLabel, localDate } from '../clock/labels';
import { routeFor, type Node } from './nodes';
import { FranchisePicker } from './FranchisePicker';
import { useReading } from './lineups';

const unavailableLabels: Record<string, string> = {
  ir: 'IR', lineup: 'Lineup', trades: 'Trades', bye: 'Bye weeks',
};

function AlertCard({ alert }: { alert: Alert }) {
  const root = useRef<HTMLDivElement>(null);
  const at = alert.at.startsWith('0001-') ? undefined : alert.at;
  useEffect(() => {
    if (root.current) return connectClock(root.current, [{
      id: 'alert', label: alert.title, at, urgency: alert.urgency, pinned: false, promoted: false,
    }]);
  }, [alert, at]);
  const route = routeFor(alert.node as Node, alert.workspace)!;
  return <div ref={root}>
    <Card gravity={alert.kind === 'ir' ? 'G2' : 'G1'} header={<h5>{alert.title}</h5>}
      provenance={provenanceSlot([['Alert', alert.provenance]])}
      actionTray={<><Act variant="text" verb={alert.kind === 'bye' ? 'endpoint.open' : 'nav.open'}
        args={alert.kind === 'bye' ? { id: 'M-154' } : route}>
        Open {alert.node === 'trade' ? 'Trade desk' : 'Franchise HQ'}
      </Act>{alert.kind === 'ir' && <Act variant="button" verb="personal.handoff" args={{ kind: 'ir' }}
          disabled={!alert.mflUrl}>Open IR page on MFL</Act>}</>}>
      <p>{alert.detail}</p>
      <p data-deadline="alert" data-urgency={alert.urgency}>
        <time dateTime={at}>{localDate(at)}</time>{' · '}
        {at && (['lineup', 'bye'].includes(alert.kind) ? 'lock in ' : 'expires in ')}
        <span className="channel-countdown" data-countdown>{countdown(at, Date.now())}</span>
      </p>
    </Card>
  </div>;
}
function PinnedDeadlines({ deadlines }: { deadlines: Deadline[] }) {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (root.current) return connectClock(root.current, deadlines);
  }, [deadlines]);
  return <div ref={root}>{deadlines.map(deadline =>
    <Card key={deadline.id} gravity="G2" header={<h5>{deadlineLabel(deadline.label)}</h5>}
      actionTray={<Act variant="text" {...deadlineRoute(deadline.id)}
        label={`Act · ${deadlineLabel(deadline.label)}`} />}>
      <p data-deadline={deadline.id} data-urgency={deadline.urgency}>
        <time dateTime={deadline.at}>{localDate(deadline.at)}</time>{' · '}
        <span className="channel-countdown" data-countdown>{countdown(deadline.at, Date.now())}</span>
      </p>
    </Card>)}</div>;
}
function FranchiseAlerts({ franchiseId, snapshot, pinned }: {
  franchiseId: string; snapshot: Snapshot; pinned: boolean;
}) {
  const { reading, error } = useReading(commands.alerts, franchiseId, snapshot);
  if (!reading) return <p role="status">{error || 'Reading alerts…'}</p>;
  return <>
    {error && <p role="alert">Alerts unavailable · {error}</p>}
    {!reading.alerts.length && !reading.unavailable.length && !pinned && <p>Nothing needs you right now</p>}
    {reading.alerts.map((alert, index) => <AlertCard key={`${alert.kind}:${index}`} alert={alert} />)}
    {reading.unavailable.map((item, index) => <p className="not-wired" key={`${item.kind}:${index}`}>
      {unavailableLabels[item.kind] ?? item.kind} · {item.note}
    </p>)}
  </>;
}
export default function AlertTray({ snapshot, reading }: ClockProps & { snapshot: Snapshot }) {
  const franchiseId = commands.use().franchiseId;
  const pinned = (reading?.value.deadlines ?? []).filter(d => d.pinned)
    .sort((a, b) => b.urgency.localeCompare(a.urgency) || a.id.localeCompare(b.id));
  return <section aria-label="Alert tray" data-endpoint="M-149">
    <PinnedDeadlines deadlines={pinned} />
    {reading?.provenance.freshness.state === 'fail' && <p role="alert">Clock unavailable</p>}
    {franchiseId ? <FranchiseAlerts key={franchiseId} franchiseId={franchiseId}
      snapshot={snapshot} pinned={pinned.length > 0} /> : <FranchisePicker snapshot={snapshot} />}
  </section>;
}
