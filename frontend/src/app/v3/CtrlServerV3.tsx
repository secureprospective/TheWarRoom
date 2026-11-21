// Desktop v3 Control Room, Server tab, after TheWarRoom-mock-v3-FINAL.html (screens.js control.server):
// the league ledger, commissioner backup via HostedBackupNow, and operator-only notices for restore, feeds
// and upgrades. Everything comes from the hosted room's own ledger query and backup command.
import { useEffect, useMemo, type ReactNode } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HostedBackupNow } from '../commissioner/HostedBackupNow';
import { createHostedLedgerQuery } from '../commissioner/hostedControlRoomData';
import { memberDisplayName, type People } from '../commissioner/people';
import type { LedgerEntry } from '../commissioner/proxy';
import { hostedWorkflow } from '../control/nativeApi';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** Formats a ledger entry's actor display name, reason and safeguard for display. */
export function ledgerDetail(entry: LedgerEntry, people?: Pick<People, 'seats'>): string {
  const actor = memberDisplayName(entry.who, people);
  return `By ${actor} · Reason: ${entry.why || 'No reason recorded'} · Safeguard: ${entry.gravity}`;
}

export function CtrlServerV3({ league, port = hostedWorkflow, badge, people }: ControlBodyProps) {
  const ledgerQuery = useMemo(() => createHostedLedgerQuery(league, port), [league, port]);
  const ledgerState = ledgerQuery.use();
  useEffect(() => {
    void ledgerQuery.load([league, '']);
    return () => ledgerQuery.dispose();
  }, [league, ledgerQuery]);
  const ledger = ledgerState.ledger;
  const error = ledgerState.error;
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.4fr) minmax(0,1fr)' }}>
    <Panel title="League ledger" badge={badge}
      tag={ledger ? <span className="tag o">{ledger.entries.length}</span> : null}
      foot={ledger ? `${ledger.entries.length} from the league ledger` : 'Reading server ledger…'}>
      {error && <St><FailureAlert message={error} code={ledgerState.errorCode} league={league}
        screen="Control Room · Server" route="control/server" port={port} /></St>}
      {!ledger && !error && <St><span role="status">Reading server ledger…</span></St>}
      {ledger && ledger.entries.length === 0 &&
        <St>No changes are visible to you in this league yet.</St>}
      {ledger && ledger.entries.map(entry => <div key={entry.id} className="row"
        style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto' }}>
        <span className="tag">{entry.gravity}</span>
        <span className="ell">
          <b>{entry.what}</b> <span className="d">{ledgerDetail(entry, people)}</span>
        </span>
        <span className="m">{new Date(entry.when).toLocaleString()}</span></div>)}
    </Panel>
    <Panel title="Backups, feeds and upgrades" badge={badge}
      foot="restores put rosters and contracts back first">
      <div className="sec">BACKUPS</div>
      <p className="d" style={{ padding: '6px 10px 0' }}>
        Commissioners can save and verify a fresh league backup. Restore, feed controls and upgrades
        are operator-only.
      </p>
      <div style={{ padding: '6px 10px' }}><HostedBackupNow league={league} port={port} /></div>
      <p className="d" style={{ padding: '0 10px 8px' }}>
        Restore is operator-only and unavailable here.
      </p>
      <div className="sec">FEEDS AND UPGRADES</div>
      <p className="d" style={{ padding: '6px 10px' }}>
        Feed controls and upgrades are operator-only and unavailable here.
      </p>
    </Panel>
  </div></div>;
}
