import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from './surfaceActions';
import { readLedgerFiltered, refusalText, type Ledger, type LedgerFilter } from './proxy';

type Props = {
  reading: Ledger;
  children(reading: Ledger, filtered: boolean): ReactNode;
  read?: typeof readLedgerFiltered;
};
const empty = (): LedgerFilter => ({ person: '', kind: '', from: '', before: '' });
const iso = (local: string) => local ? new Date(local).toISOString() : '';

export function LedgerFilters({ reading, children, read = readLedgerFiltered }: Props) {
  const [draft, setDraft] = useState(empty);
  const [applied, setApplied] = useState(empty);
  const [inputError, setInputError] = useState('');
  const [answer, setAnswer] = useState<{ key: string; reading?: Ledger; error?: string }>();
  const request = useRef<{ key: string; work: Promise<Ledger> }>();
  const filtered = Object.values(applied).some(value => value !== '');
  const key = JSON.stringify([reading.league, reading.revision, applied]);
  useEffect(() => {
    if (!filtered) return;
    let current = true;
    const work = request.current?.key === key
      ? request.current.work : read(reading.league, applied);
    request.current = undefined;
    void work.then(result => {
      if (current) setAnswer({ key, reading: result });
    }).catch(cause => {
      if (current) setAnswer({ key, error: refusalText(cause) });
    });
    return () => { current = false; };
  }, [reading, applied, filtered, key, read]);
  const apply = useCallback(() => {
    let next: LedgerFilter;
    try {
      next = { ...draft, from: iso(draft.from), before: iso(draft.before) };
    } catch {
      const message = 'Choose valid local dates and times.';
      setInputError(message);
      throw new Error(message);
    }
    setInputError('');
    setApplied(next);
    if (!Object.values(next).some(value => value !== '')) return;
    const work = read(reading.league, next);
    request.current = { key: JSON.stringify([reading.league, reading.revision, next]), work };
    return work.then(() => {}, cause => { throw new Error(refusalText(cause)); });
  }, [draft, reading, read]);
  const clear = useCallback(() => { setDraft(empty()); setApplied(empty()); setInputError(''); }, []);
  const screen = `ledger.${reading.league}`;
  useEffect(() => surfaceActions.register(screen, args => {
    if (args.action === 'find') return apply();
    if (args.action === 'clear') clear();
  }), [screen, apply, clear]);
  const update = (field: keyof LedgerFilter, value: string) => setDraft({ ...draft, [field]: value });
  const visible = filtered ? answer?.key === key ? answer.reading : undefined : reading;
  const error = inputError || (filtered && answer?.key === key ? answer.error : '');
  return (
    <>
      <fieldset className="ledger-filters">
        <legend>Find league changes</legend>
        <label>Person <input value={draft.person} onChange={e => update('person', e.target.value)} /></label>
        <label>Kind of change
          <select value={draft.kind} onChange={e => update('kind', e.target.value)}>
            <option value="">All changes</option><option value="setup">League setup</option>
            <option value="rules">Rules</option><option value="people">People</option>
            <option value="notices">Notices and notes</option>
            <option value="corrections">Corrections</option>
          </select>
        </label>
        <label>From (local time, included)
          <input type="datetime-local" value={draft.from} onChange={e => update('from', e.target.value)} />
        </label>
        <label>Before (local time, excluded)
          <input type="datetime-local" value={draft.before}
            onChange={e => update('before', e.target.value)} />
        </label>
        <Act verb="commissioner.surface" args={{ screen, action: 'find', value: '' }}
          variant="button" label="Find changes" />
        <Act verb="commissioner.surface" args={{ screen, action: 'clear', value: '' }}
          variant="button" label="Clear filters" />
      </fieldset>
      {error && <p role="alert">{error}</p>}
      {filtered && !visible && !error && <p role="status">Finding league changes…</p>}
      {visible && children(visible, filtered)}
    </>
  );
}
