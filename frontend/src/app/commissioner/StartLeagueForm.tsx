import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { newKey, type Change } from './forms';
import { leagueState } from './proxy';

const presets = [
  { value: 'legacy_nfl', label: 'Legacy NFL rules' },
  { value: 'empty', label: 'Start empty: every rule Needs a ruling' },
] as const;
const freshId = () => `native.${newKey().slice(0, 16)}`;

// StartLeagueForm creates an app league. Go expands the franchises and the preset; the form sends the
// name, the size and the preset, and a typed reason.
export function StartLeagueForm() {
  const [name, setName] = useState('');
  const [preset, setPreset] = useState<(typeof presets)[number]['value']>('legacy_nfl');
  const [count, setCount] = useState('32');
  const [reason, setReason] = useState('');
  const [league, setLeague] = useState(freshId);
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === 'create' ? outcome.receipt.entry : '';
  useEffect(() => {
    if (!landed) return;
    setName(''); setReason(''); setLeague(freshId());
  }, [landed]);
  const size = /^\d+$/.test(count.trim()) ? Number(count) : 0;
  const change: Change | null = name.trim() !== '' && size > 0 ? {
    league, command: 'commish.league.create', args: { name: name.trim(), preset, franchiseCount: size },
    reason, expected: 0, effective: 'now', visibility: 'league',
  } : null;
  return (
    <section className="league-form" aria-label="Start an app league">
      <h5>Start an app league</h5>
      <p>A league run in this app. You become its commissioner.</p>
      <label>League name
        <input value={name} onChange={e => setName(e.target.value)} />
      </label>
      <label>Number of franchises
        <input type="number" min={2} max={32} step={1} value={count}
          onChange={e => setCount(e.target.value)} />
      </label>
      <p className="change-note">Both starting choices work with 2 to 32 franchises.</p>
      <label>Starting rules
        <select value={preset} onChange={e => setPreset(e.target.value as typeof preset)}>
          {presets.map(p => <option key={p.value} value={p.value}>{p.label}</option>)}
        </select>
      </label>
      <ChangeBar form="create" change={change} safeguard="reason" reason={reason} onReason={setReason}
        confirmLabel="Start this league"
        blocked={name.trim() === '' ? 'Name the league.'
          : size === 0 ? 'Enter the number of franchises.' : ''} />
    </section>
  );
}
