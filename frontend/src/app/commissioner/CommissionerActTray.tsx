import { useEffect, useRef, useState } from 'react';
import { errorCodeFor, errorMessage } from '../data/errorCodes';
import { FailureAlert } from '../data/FailureAlert';
import { Act } from '../commands/Act';
import { HQAct } from '../shell/HQAct';
import type { NativeRequest } from '../commands/native';
import { type ActDefinition } from './actTable';
import { actChange, type ActValues } from './actBody';
import { readHostedActData, type HostedActReadings } from './actReadings';
import { ActFields } from './ActFields';
import { newKey, useRequestKey } from './forms';
import { ChangeBar } from './ChangeBar';
import { refusalText } from './proxy';
import { Review } from './HostedReview';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';
import { surfaceActions } from './surfaceActions';

export function HostedCommissionerActTray({ definition, league, room, port, initial }: {
  definition: ActDefinition; league: string; room: HostedLeagueRoom; port?: HostedPort; initial?: ActValues;
}) {
  const state = room.use();
  return <section aria-label={definition.label} className="hosted-act-tray">
    <h3>{definition.label}</h3>
    <HostedCommissionerActForm key={`${league}:${definition.command}`} definition={definition}
      league={league} room={room} port={port} initial={initial} />
    {state.error && <p role="alert">{state.error}</p>}
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
