import { useEffect, useState } from 'react';
import { Act } from '../../commands/Act';
import { newKey, noReasonWords, safeguardWords, type Change } from '../../commissioner/forms';
import { HoldToFire } from '../../commissioner/HoldToFire';
import { previewChange, type Preview } from '../../commissioner/preview';
import { leagueState, readPreview, refusalText } from '../../commissioner/proxy';
import { applyPreset, presetChanges, readPresetComparison, type PresetComparison }
  from '../../commissioner/presetCompare';
import type { Safeguard } from '../../commissioner/rules';
import { useRoomSurface } from './useRoomSurface';

const screen = 'room-presets';
const firmness: Safeguard[] = ['confirm', 'hold', 'reason'];
// The firmest safeguard Go names among the changes is the one the whole preset change takes.
const strictest = (previews: Preview[]): Safeguard | null =>
  previews.some(p => p.safeguard === null) ? null
    : previews.reduce<Safeguard>((most, p) =>
      firmness.indexOf(p.safeguard!) > firmness.indexOf(most) ? p.safeguard! : most, 'confirm');

type Review = { previews: Preview[]; request: string } | { error: string };

// PresetCompare shows every setting where the league differs from a preset (setting, league value, preset
// value), lets the commissioner choose all or some, previews each through the ordinary rules preview, and
// records the chosen ones as one rules change. Go decides what differs, what is refused and how firmly to
// confirm; nothing here is a rule of its own.
export function PresetCompare({ league, initial = 'default_32' }: { league: string; initial?: string }) {
  const [preset, setPreset] = useState(initial);
  const [comparison, setComparison] = useState<PresetComparison>();
  const [error, setError] = useState('');
  const [chosen, setChosen] = useState<string[]>([]);
  const [effective, setEffective] = useState<Change['effective']>('next_rollover');
  const [reason, setReason] = useState('');
  const [review, setReview] = useState<Review>();
  const [done, setDone] = useState('');
  const load = (id: string) => {
    setPreset(id);
    setReview(undefined);
    readPresetComparison(league, id).then(c => {
      setComparison(c);
      setChosen(c.differences.map(d => d.field));
      setError('');
    }).catch(cause => { setComparison(undefined); setError(refusalText(cause)); });
  };
  useEffect(() => load(preset), [league]);
  const picked = comparison?.differences.filter(d => chosen.includes(d.field)) ?? [];
  const nowOnly = picked.some(d => d.nowOnly);
  const timing = nowOnly ? 'now' : effective;
  const apply = (request: string) => ({ league, preset, fields: chosen, reason: reason.trim(),
    expected: comparison?.revision ?? 0, request, effective: timing });
  const previewAll = async () => {
    setDone('');
    try {
      const request = newKey();
      const bodies = await presetChanges(apply(request));
      setReview({ previews: await Promise.all(bodies.map(readPreview)), request });
    } catch (cause) { setReview({ error: refusalText(cause) }); }
  };
  const confirm = async (request: string) => {
    try {
      const count = await applyPreset(apply(request));
      setDone(`Recorded ${count} rule ${count === 1 ? 'change' : 'changes'} from the ${comparison?.presetLabel}.`);
      setReview(undefined);
      await leagueState.choose(league);
      load(preset);
    } catch (cause) { setReview({ error: refusalText(cause) }); }
  };
  useRoomSurface(screen, ({ action, value }) => {
    if (action === 'open') load(value);
    if (action === 'check') {
      setReview(undefined);
      setChosen(c => c.includes(value) ? c.filter(f => f !== value) : [...c, value]);
    }
    if (action === 'pick') { setReview(undefined); setEffective(value as Change['effective']); }
    if (action === 'view') return previewAll();
    if (action === 'tray' && review && 'request' in review) return confirm(review.request);
    return undefined;
  });
  const previews = review && 'previews' in review ? review.previews : [];
  const safeguard = review && 'previews' in review ? strictest(previews) : null;
  const blockers = previews.flatMap(p => p.blockers);
  const need = safeguard === 'reason' && reason.trim() === '' ? noReasonWords : '';
  const ready = safeguard !== null && blockers.length === 0 && need === '';
  return (
    <section className="league-form" aria-label="Compare with a preset">
      <h3>Compare with a preset</h3>
      <div className="cr-chips">
        {(comparison?.presets ?? [{ id: preset, label: preset }]).map(p => (
          <Act key={p.id} className="cr-chip" active={p.id === preset} verb="commissioner.surface" variant="surface"
            label={p.label} args={{ screen, action: 'open', value: p.id }}>{p.label}</Act>
        ))}
      </div>
      {error && <p role="alert">{error}</p>}
      {comparison && <p role="status">{comparison.summary}</p>}
      {comparison && comparison.differences.length > 0 && (
        <table className="cr-t">
          <thead><tr><th>APPLY</th><th>SETTING</th><th>THIS LEAGUE</th><th>{comparison.presetLabel}</th></tr></thead>
          <tbody>
            {comparison.differences.map(d => (
              <tr key={d.field} data-preset-field={d.field}>
                <td>
                  <Act className="cr-btn sm" selected={chosen.includes(d.field)} verb="commissioner.surface"
                    variant="surface" label={`${chosen.includes(d.field) ? 'Skip' : 'Apply'} ${d.label}`}
                    args={{ screen, action: 'check', value: d.field }}>
                    {chosen.includes(d.field) ? '✓' : '—'}
                  </Act>
                </td>
                <td>{d.label}</td><td className="v">{d.league}</td><td className="v">{d.preset}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {picked.length > 0 && (
        <>
          <div className="cr-chips" aria-label="When it takes effect">
            {(['next_rollover', 'now'] as const).filter(t => !nowOnly || t === 'now').map(t => (
              <Act key={t} className="cr-chip" active={timing === t} verb="commissioner.surface" variant="surface"
                label={t === 'now' ? 'Now' : 'From the next rollover'} args={{ screen, action: 'pick', value: t }}>
                {t === 'now' ? 'Now' : 'From the next rollover'}
              </Act>
            ))}
          </div>
          {nowOnly && <p className="change-note">A chosen setting changes only now, so every chosen setting does.</p>}
          <label className="change-reason">Reason
            <textarea value={reason} rows={2} onChange={e => { setReason(e.target.value); setReview(undefined); }} />
          </label>
          <Act className="cr-btn" verb="commissioner.surface" variant="surface"
            label={`Preview ${picked.length} rule ${picked.length === 1 ? 'change' : 'changes'}`}
            args={{ screen, action: 'view', value: preset }}>
            Preview {picked.length} rule {picked.length === 1 ? 'change' : 'changes'}
          </Act>
        </>
      )}
      {review && 'error' in review && <p role="alert" className="change-refused">{review.error}</p>}
      {previews.length > 0 && (
        <section aria-label="Before and after">
          <h5>Before → after</h5>
          {previews.flatMap(p => p.fields).map((f, i) => <p key={i}>{f.label}: {previewChange(f)}</p>)}
          <h5>Before you confirm</h5>
          {blockers.length === 0 ? <p>No blockers.</p> : <ul role="alert">{blockers.map((b, i) => <li key={i}>{b}</li>)}</ul>}
          {safeguard && <p className="change-safeguard">{safeguardWords[safeguard]}</p>}
          {need && <p className="change-why" role="status">{need}</p>}
          {safeguard === 'hold'
            ? <HoldToFire label="Record these rule changes" body={preset} disabled={!ready}
              onConfirm={() => { if ('request' in review!) void confirm(review.request); }} />
            : <Act className="cr-btn pri" disabled={!ready} verb="commissioner.surface" variant="surface"
              label="Record these rule changes" args={{ screen, action: 'tray', value: preset }}>
              Record these rule changes
            </Act>}
        </section>
      )}
      {done && <p role="status" className="change-done">{done}</p>}
    </section>
  );
}
