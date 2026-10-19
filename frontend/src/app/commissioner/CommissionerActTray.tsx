import { useEffect, useRef, useState } from 'react';
import { Act } from '../commands/Act';
import { actsFor, type ActDefinition } from './actTable';
import { actChange, type ActValues } from './actBody';
import { readActData, type ActReadings } from './actReadings';
import { ActFields } from './ActFields';
import { bodyOf, useRequestKey } from './forms';
import { ChangeBar } from './ChangeBar';
import { PreviewReview } from './PreviewReview';
import { formOf, nativeRolloverDryRun, refusalText } from './proxy';
import SideTray from './SideTray';
import { surfaceActions } from './surfaceActions';
import { useChangePreview } from './useChangePreview';

export function CommissionerActForm({ definition, league }: { definition: ActDefinition; league: string }) {
  const [values, setValues] = useState<ActValues>({});
  const [reason, setReason] = useState('');
  const [held, setHeld] = useState<ActReadings>();
  const [error, setError] = useState('');
  const [dry, setDry] = useState('');
  const [dryError, setDryError] = useState('');
  const [readingDry, setReadingDry] = useState(false);
  const readingSeason = definition.root === 'bracket' ? values.season : undefined;
  useEffect(() => {
    let active = true;
    setHeld(undefined);
    setError('');
    const season = Number(readingSeason);
    const year = season >= 1920 && season <= 9999 ? season : undefined;
    void readActData(league, definition.fields.map(f => f.kind), year, '',
      definition.command).then(data => {
      if (active) setHeld(data);
    }).catch(cause => { if (active) setError(refusalText(cause)); });
    return () => { active = false; };
  }, [league, definition, readingSeason]);
  const change = held ? actChange(definition, values, held, reason) : null;
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  const rollover = definition.command === 'season.rollover';
  const signature = change ? bodyOf(change, request) : '';
  const dryReady = !rollover || dry === signature;
  const runDry = async () => {
    if (!change || readingDry) return;
    setReadingDry(true);
    setDryError('');
    try {
      await nativeRolloverDryRun(bodyOf({ ...change,
        args: { rollover: { to: Number(values.to), dry_run: true } } }, `${request}.dry`));
      setDry(signature);
    } catch (cause) { setDryError(refusalText(cause)); }
    finally { setReadingDry(false); }
  };
  const latestDry = useRef(runDry);
  latestDry.current = runDry;
  useEffect(() => surfaceActions.register('act-dry-run', () => void latestDry.current()), []);
  const blocked = error || (!change ? 'Complete each field to review this change.' : wait) ||
    (!dryReady ? 'Run the season check before recording the rollover.' : '');
  return <section className="league-form" aria-label={definition.label}>
    {error && <p role="alert">{error}</p>}
    {!held && !error && <p role="status">Reading held league data…</p>}
    <PreviewReview preview={preview} confirm={<>
      {rollover && <>
        <Act verb="commissioner.surface" args={{ screen: 'act-dry-run', action: 'check', value: '' }}
          variant="button" disabled={!change || readingDry || !!wait}>
          Check the next season without recording
        </Act>
        {dry === signature && dry && <p role="status">Season check passed. Nothing was recorded.</p>}
        {dryError && <p role="alert">{dryError}</p>}
      </>}
      <ChangeBar form={formOf(signature)} change={change} safeguard={safeguard}
        reason={reason} onReason={setReason} request={request} confirmLabel="Record this change"
        blocked={blocked} />
    </>}>
      {held && <>
        <ActFields fields={definition.fields} values={values} held={held}
          onChange={(path, value) => setValues(v => path === 'season' && definition.root === 'bracket'
            ? { season: value } : { ...v, [path]: value })} />
      </>}
    </PreviewReview>
  </section>;
}
export default function CommissionerActTray({ control, league, opener, background, onClose }: {
  control: string; league: string; opener: HTMLElement; background: HTMLElement | null; onClose(): void;
}) {
  const choices = actsFor(control);
  const [command, setCommand] = useState(choices[0]?.command ?? '');
  const definition = choices.find(a => a.command === command);
  return <SideTray title="League action" screen="landing" opener={opener} background={background}
    onClose={onClose} closeLabel="Close league action">
    <label>What to do<select value={command} onChange={e => setCommand(e.target.value)}>
      {choices.map(a => <option key={a.command} value={a.command}>{a.label}</option>)}
    </select></label>
    {definition && <CommissionerActForm key={`${league}:${command}`}
      definition={definition} league={league} />}
  </SideTray>;
}
