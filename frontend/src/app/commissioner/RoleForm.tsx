import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { personLabel, roleChoices, useRequestKey, type Change } from './forms';
import type { People } from './people';
import { PreviewReview } from './PreviewReview';
import { leagueState } from './proxy';
import { useChangePreview } from './useChangePreview';

// RoleForm gives or takes a league role, in the side tray. Role changes are private; Go decides who may
// make them.
export function RoleForm({ people }: { people: People }) {
  const [franchise, setFranchise] = useState('');
  const [action, setAction] = useState<'grant' | 'revoke'>('grant');
  const [role, setRole] = useState<(typeof roleChoices)[number]['value']>('co-commissioner');
  const [reason, setReason] = useState('');
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === 'role' ? outcome.receipt.entry : '';
  useEffect(() => { if (landed) setReason(''); }, [landed]);
  const seated = people.seats.filter(s => s.owner !== 'Vacant');
  const seat = seated.find(s => s.franchise === franchise);
  const change: Change | null = seat ? {
    league: people.league, command: `commish.role.${action}`,
    args: { people: { principal: seat.owner, role } }, reason: reason.trim(), expected: people.revision,
    effective: 'now', visibility: 'private',
  } : null;
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  return (
    <section className="league-form" aria-label="Give or take a role">
      <PreviewReview preview={preview} confirm={
        <ChangeBar form="role" change={change} request={request} reason={reason} onReason={setReason}
          safeguard={safeguard}
          confirmLabel={action === 'grant' ? 'Give this role' : 'Take this role away'}
          note="Role changes are private: owners do not see them in the ledger."
          blocked={seat ? wait : 'Choose a franchise.'} />
      }>
        {seated.length === 0 ? (
          <p>Nobody else has joined this league yet. Roles go to owners who hold a franchise.</p>
        ) : <label>Franchise
          <select value={franchise} onChange={e => setFranchise(e.target.value)}>
            <option value="">Choose a franchise…</option>
            {seated.map(s => (
              <option key={s.franchise} value={s.franchise}>{s.name} ({personLabel(s.owner)})</option>
            ))}
          </select>
        </label>}
        <label>What to do
          <select value={action} onChange={e => setAction(e.target.value as typeof action)}>
            <option value="grant">Give the role</option>
            <option value="revoke">Take the role away</option>
          </select>
        </label>
        <label>Role
          <select value={role} onChange={e => setRole(e.target.value as typeof role)}>
            {roleChoices.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}
          </select>
        </label>
      </PreviewReview>
    </section>
  );
}
