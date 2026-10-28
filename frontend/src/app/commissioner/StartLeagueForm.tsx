import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import type { StartingPreset } from './controls';
import { newKey, useRequestKey, type Change } from './forms';
import { PreviewReview } from './PreviewReview';
import { leagueState } from './proxy';
import { useChangePreview } from './useChangePreview';

const freshId = () => `native.${newKey().slice(0, 16)}`;
const sizeWords = (p: StartingPreset) => p.minTeams === p.maxTeams
  ? `The ${p.label} is for exactly ${p.minTeams} franchises.`
  : `Choose between ${p.minTeams} and ${p.maxTeams} franchises.`;

// StartLeagueForm creates an app league, in the side tray. Go expands the franchises and the preset; the
// form sends the name, the size, the preset, the NFL season the league starts in, and a typed reason. The
// presets are Go's catalog, read with the league controls; the first is the one offered first.
export function StartLeagueForm() {
  const [name, setName] = useState('');
  const [chosen, setPreset] = useState('');
  const [count, setCount] = useState('32');
  const [season, setSeason] = useState(() => String(new Date().getFullYear()));
  const [reason, setReason] = useState('');
  const [league, setLeague] = useState(freshId);
  const { outcome, controls } = leagueState.use();
  const presets = controls?.presets ?? [];
  const starting = presets.find(p => p.id === chosen) ?? presets[0];
  const preset = starting?.id ?? '';
  const landed = outcome.status === 'done' && outcome.form === 'create' ? outcome.receipt.entry : '';
  useEffect(() => {
    if (!landed) return;
    setName(''); setReason(''); setLeague(freshId());
  }, [landed]);
  const size = /^\d+$/.test(count.trim()) ? Number(count) : 0;
  const year = /^\d{4}$/.test(season.trim()) ? Number(season) : 0;
  const change: Change | null = name.trim() !== '' && size > 0 && year > 0 && preset !== '' ? {
    league, command: 'commish.league.create',
    args: { name: name.trim(), preset, franchiseCount: size, season: year },
    reason: reason.trim(), expected: 0, effective: 'now', visibility: 'league',
  } : null;
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  const blocked = !starting ? 'The starting presets could not be read.'
    : name.trim() === '' ? 'Name the league.'
    : size === 0 ? 'Enter the number of franchises.'
    : size < starting.minTeams || size > starting.maxTeams ? sizeWords(starting)
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
          <input type="number" min={starting?.minTeams ?? 2} max={starting?.maxTeams ?? 32} step={1}
            value={count} onChange={e => setCount(e.target.value)} />
        </label>
        {starting && <p className="change-note">{starting.description}</p>}
        <label>First NFL season
          <input type="number" min={1920} max={9999} step={1} value={season}
            onChange={e => setSeason(e.target.value)} />
        </label>
        <label>Starting rules
          <select value={preset} onChange={e => {
            const picked = presets.find(p => p.id === e.target.value);
            setPreset(e.target.value);
            if (picked && picked.minTeams === picked.maxTeams) setCount(String(picked.minTeams));
          }}>
            {presets.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
          </select>
        </label>
      </PreviewReview>
    </section>
  );
}
