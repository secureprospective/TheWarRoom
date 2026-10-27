import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { newKey, useRequestKey, type Change } from './forms';
import { PreviewReview } from './PreviewReview';
import { leagueState } from './proxy';
import { useChangePreview } from './useChangePreview';

// Legacy NFL rules stay Go's preset for MFL import and recovery; a new league starts from these two.
const presets = [
  { value: 'default_32', label: '32-Team Default' },
  { value: 'empty', label: 'Start empty: every rule Needs a ruling' },
] as const;
const freshId = () => `native.${newKey().slice(0, 16)}`;

// StartLeagueForm creates an app league, in the side tray. Go expands the franchises and the preset; the
// form sends the name, the size, the preset, the NFL season the league starts in, and a typed reason.
export function StartLeagueForm() {
  const [name, setName] = useState('');
  const [preset, setPreset] = useState<(typeof presets)[number]['value']>('default_32');
  const [count, setCount] = useState('32');
  const [season, setSeason] = useState(() => String(new Date().getFullYear()));
  const [reason, setReason] = useState('');
  const [league, setLeague] = useState(freshId);
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === 'create' ? outcome.receipt.entry : '';
  useEffect(() => {
    if (!landed) return;
    setName(''); setReason(''); setLeague(freshId());
  }, [landed]);
  const size = /^\d+$/.test(count.trim()) ? Number(count) : 0;
  const year = /^\d{4}$/.test(season.trim()) ? Number(season) : 0;
  const change: Change | null = name.trim() !== '' && size > 0 && year > 0 ? {
    league, command: 'commish.league.create',
    args: { name: name.trim(), preset, franchiseCount: size, season: year },
    reason: reason.trim(), expected: 0, effective: 'now', visibility: 'league',
  } : null;
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  const blocked = name.trim() === '' ? 'Name the league.'
    : size === 0 ? 'Enter the number of franchises.'
    : preset === 'default_32' && size !== 32 ? 'The 32-Team Default is for exactly 32 franchises.'
    : year === 0 ? 'Enter the NFL season the league starts in.' : wait;
  return (
    <section className="league-form" aria-label="Start an app league">
      <PreviewReview preview={preview} confirm={
        <ChangeBar form="create" change={change} request={request} reason={reason} onReason={setReason}
          safeguard={safeguard} confirmLabel="Start this league" blocked={blocked} />
      }>
        <p>A league run in this app. You become its commissioner.</p>
        <label>League name
          <input value={name} onChange={e => setName(e.target.value)} />
        </label>
        <label>Number of franchises
          <input type="number" min={2} max={32} step={1} value={count}
            onChange={e => setCount(e.target.value)} />
        </label>
        <p className="change-note">The 32-Team Default is for 32 franchises; an empty league works with 2 to 32.</p>
        <label>First NFL season
          <input type="number" min={1920} max={9999} step={1} value={season}
            onChange={e => setSeason(e.target.value)} />
        </label>
        <label>Starting rules
          <select value={preset} onChange={e => setPreset(e.target.value as typeof preset)}>
            {presets.map(p => <option key={p.value} value={p.value}>{p.label}</option>)}
          </select>
        </label>
      </PreviewReview>
    </section>
  );
}
