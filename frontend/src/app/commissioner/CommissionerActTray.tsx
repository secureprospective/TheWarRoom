import { useEffect, useRef, useState } from 'react';
import { errorCodeFor, errorMessage } from '../data/errorCodes';
import { FailureAlert } from '../data/FailureAlert';
import { Act } from '../commands/Act';
import { HQAct } from '../shell/HQAct';
import type { NativeRequest } from '../commands/native';
import { actsFor, type ActDefinition } from './actTable';
import { actChange, type ActValues } from './actBody';
import { readActData, readHostedActData, type ActReadings, type HostedActReadings } from './actReadings';
import { ActFields } from './ActFields';
import { bodyOf, newKey, useRequestKey } from './forms';
import { ChangeBar } from './ChangeBar';
import { PreviewReview } from './PreviewReview';
import { formOf, nativeRolloverDryRun, refusalText } from './proxy';
import SideTray from './SideTray';
import { Review } from './HostedReview';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';
import { surfaceActions } from './surfaceActions';
import { useChangePreview } from './useChangePreview';

export function CommissionerActForm({
  definition,
  league,
  initial,
}: {
  definition: ActDefinition;
  league: string; /** Values a gesture already chose (a dragged
    window's new day). */
  initial?: ActValues;
}) {
  const [values, setValues] = useState<ActValues>(initial ?? {});
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
    void readActData(
      league,
      definition.fields.map((f) => f.kind),
      year,
      '',
      definition.command,
    )
      .then((data) => {
        if (active) setHeld(data);
      })
      .catch((cause) => {
        if (active) setError(refusalText(cause));
      });
    return () => {
      active = false;
    };
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
      await nativeRolloverDryRun(
        bodyOf({ ...change, args: { rollover: { to: Number(values.to), dry_run: true } } }, `${request}.dry`),
      );
      setDry(signature);
    } catch (cause) {
      setDryError(refusalText(cause));
      throw cause;
    } finally {
      setReadingDry(false);
    }
  };
  const latestDry = useRef(runDry);
  latestDry.current = runDry;
  useEffect(() => surfaceActions.register('act-dry-run', () => latestDry.current()), []);
  const blocked =
    error ||
    (!change ? 'Complete each field to review this change.' : wait) ||
    (!dryReady ? 'Run the season check before recording the rollover.' : '');
  return (
    <section className="league-form" aria-label={definition.label}>
      {error && <p role="alert">{error}</p>}
      {!held && !error && <p role="status">Reading held league data…</p>}
      <PreviewReview
        preview={preview}
        confirm={
          <>
            {rollover && (
              <>
                <Act
                  verb="commissioner.surface"
                  args={{ screen: 'act-dry-run', action: 'check', value: '' }}
                  variant="button"
                  disabled={!change || readingDry || !!wait}
                >
                  Check the next season without recording
                </Act>
                {dry === signature && dry && <p role="status">Season check passed. Nothing was recorded.</p>}
                {dryError && <p role="alert">{dryError}</p>}
              </>
            )}
            <ChangeBar
              form={formOf(signature)}
              change={change}
              safeguard={safeguard}
              reason={reason}
              onReason={setReason}
              request={request}
              confirmLabel="Record this change"
              blocked={blocked}
            />
          </>
        }
      >
        {held && (
          <>
            <ActFields
              fields={definition.fields}
              values={values}
              held={held}
              onChange={(path, value) =>
                setValues((v) =>
                  path === 'season' && definition.root === 'bracket'
                    ? { season: value }
                    : { ...v, [path]: value },
                )
              }
            />
          </>
        )}
      </PreviewReview>
    </section>
  );
}
export function HostedCommissionerActTray({ definition, league, room, port, initial }: {
  definition: ActDefinition; league: string; room: HostedLeagueRoom; port?: HostedPort; initial?: ActValues;
}) {
  const state = room.use();
  return <section aria-label={definition.label} className="hosted-act-tray">
    <h3>{definition.label}</h3>
    <HostedCommissionerActForm key={`${league}:${definition.command}`} definition={definition}
      league={league} room={room} port={port} initial={initial} />
    {state.review && <Review room={room} />}
  </section>;
}

function HostedCommissionerActForm({ definition, league, room, port, initial }: {
  definition: ActDefinition; league: string; room: HostedLeagueRoom; port?: HostedPort; initial?: ActValues;
}) {
  const state = room.use();
  const [values, setValues] = useState<ActValues>(initial ?? {});
  const [reason, setReason] = useState('');
  const [held, setHeld] = useState<HostedActReadings>();
  const [error, setError] = useState('');
  const [errorCode, setErrorCode] = useState<string>();
  const rules = state.reading?.rules;
  const selectedSeason = definition.root === 'bracket' ? Number(values.season) : undefined;
  const bracketSeason = selectedSeason !== undefined && selectedSeason >= 1920 && selectedSeason <= 9999
    ? selectedSeason : undefined;
  useEffect(() => {
    if (!rules || rules.league !== league) return;
    let active = true;
    setHeld(undefined);
    setError('');
    setErrorCode(undefined);
    void readHostedActData(league, rules.revision, definition.fields.map(field => field.kind),
      definition.command, port, bracketSeason).then(data => {
        if (active) setHeld(data);
      }).catch(cause => {
        if (active) { setError(errorMessage(cause)); setErrorCode(errorCodeFor(cause)); }
      });
    return () => { active = false; };
  }, [bracketSeason, definition, league, port, rules]);
  const change = held ? actChange(definition, values, held, reason) : null;
  const disabled = !room.canChange() || state.pending || !held || Boolean(error) || Boolean(state.review);
  const preview = async () => {
    if (!change) return;
    const request = { command: definition.command, mode: 'preview', arguments: {
      league, command: definition.command, args: change.args, reason: change.reason,
      expected: change.expected, request: newKey(), effective: change.effective,
      visibility: change.visibility,
    } } as NativeRequest;
    await room.review(request, definition.label);
  };
  return <div className="league-form">
    {error && <FailureAlert
      message={error}
      code={errorCode}
      league={league}
      screen={`Control Room · ${definition.label}`}
      route="control/jobs"
      port={port}
    />}
    {!held && !error && <p role="status">Reading held league choices…</p>}
    {held && <fieldset disabled={disabled}>
      <ActFields fields={definition.fields} values={values} held={held}
        onChange={(path, value) => setValues(current => ({ ...current, [path]: value }))} />
      <label>Reason for this change<textarea value={reason}
        onChange={event => setReason(event.target.value)} /></label>
    </fieldset>}
    <HQAct disabled={disabled || !change} onAction={preview}>Preview {definition.label.toLowerCase()}</HQAct>
    <HQAct disabled={!state.review} onAction={() => room.cancel()}>Cancel preview</HQAct>
  </div>;
}

export default function CommissionerActTray({
  control,
  league,
  opener,
  background,
  onClose,
}: {
  control: string;
  league: string;
  opener: HTMLElement;
  background: HTMLElement | null;
  onClose(): void;
}) {
  const choices = actsFor(control);
  const [command, setCommand] = useState(choices[0]?.command ?? '');
  const definition = choices.find((a) => a.command === command);
  return (
    <SideTray
      title="League action"
      screen="landing"
      opener={opener}
      background={background}
      onClose={onClose}
      closeLabel="Close league action"
    >
      <label>
        What to do
        <select value={command} onChange={(e) => setCommand(e.target.value)}>
          {choices.map((a) => (
            <option key={a.command} value={a.command}>
              {a.label}
            </option>
          ))}
        </select>
      </label>
      {definition && (
        <CommissionerActForm key={`${league}:${command}`} definition={definition} league={league} />
      )}
    </SideTray>
  );
}
