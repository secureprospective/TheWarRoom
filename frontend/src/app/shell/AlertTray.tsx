import { useEffect, useRef } from 'react';
import type { Alert, Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { Card } from '../cards/Card';
import { provenanceSlot } from '../cards/playerModel';
import { connectClock } from '../clock/ticker';
import { countdown } from '../clock/urgency';
import { localDate } from '../clock/labels';
import { routeFor, type Node } from './nodes';
import { FranchisePicker } from './FranchisePicker';
import { useReading } from './lineups';

const unavailableLabels: Record<string, string> = { ir: 'IR', lineup: 'Lineup', trades: 'Trades' };

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
    <Card gravity="G1" header={<h5>{alert.title}</h5>}
      provenance={provenanceSlot([['Alert', alert.provenance]])}
      actionTray={<Act verb="nav.open" args={route}>
        Open {alert.node === 'trade' ? 'Trade desk' : 'Franchise HQ'}
      </Act>}>
      <p>{alert.detail}</p>
      <p data-deadline="alert" data-urgency={alert.urgency}>
        <time dateTime={at}>{localDate(at)}</time>{' · '}
        {at && (alert.kind === 'lineup' ? 'lock in ' : 'expires in ')}
        <span className="channel-countdown" data-countdown>{countdown(at, Date.now())}</span>
      </p>
    </Card>
  </div>;
}
function FranchiseAlerts({ franchiseId, snapshot }: { franchiseId: string; snapshot: Snapshot }) {
  const { reading, error } = useReading(commands.alerts, franchiseId, snapshot);
  if (!reading) return <p role="status">{error || 'Reading alerts…'}</p>;
  return <>
    {error && <p role="alert">Alerts unavailable · {error}</p>}
    {!reading.alerts.length && <p>Nothing needs you right now</p>}
    {reading.alerts.map((alert, index) => <AlertCard key={`${alert.kind}:${index}`} alert={alert} />)}
    {reading.unavailable.map((item, index) => <p className="not-wired" key={`${item.kind}:${index}`}>
      {unavailableLabels[item.kind] ?? item.kind} · {item.note}
    </p>)}
  </>;
}
export default function AlertTray({ snapshot }: { snapshot: Snapshot }) {
  const franchiseId = commands.use().franchiseId;
  if (!franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return <section aria-label="Alert tray">
    <FranchiseAlerts key={franchiseId} franchiseId={franchiseId} snapshot={snapshot} />
  </section>;
}
