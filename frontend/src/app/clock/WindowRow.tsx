import { useEffect, useState } from 'react';
import type { LeagueWindow } from '../data/contract';
import { Act } from '../commands/Act';
import { localDate, windowLabels } from './labels';
import { useWindowResults } from './windowState';

export function localInput(at?: string): string {
  if (!at) return '';
  const date = new Date(at);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
const ruleDated = new Set(['trade_deadline', 'playoff_bids_only', 'multiyear_bids_close']);

export function WindowRow({ window }: { window: LeagueWindow }) {
  const [date, setDate] = useState(localInput(window.at));
  useEffect(() => { setDate(localInput(window.at)); }, [window.at]);
  const result = useWindowResults()[window.kind];
  const label = windowLabels[window.kind];
  const hasRule = ruleDated.has(window.kind);
  const instant = date && !Number.isNaN(new Date(date).getTime()) ? new Date(date).toISOString() : '';
  return <li>
    <p>{label} · {window.at ? localDate(window.at) : 'No date'} · {window.reason}
      {window.moved ? ' · set by the commissioner' : ''}</p>
    <label>{label} date{' '}
      <input type="datetime-local" value={date} onChange={event => setDate(event.target.value)} />
    </label>{' '}
    <Act verb="window.set" args={{ kind: window.kind, at: instant }} label="Set date"
      disabled={!instant || result?.pending} />{' '}
    {window.moved || (!hasRule && window.at) ?
      <Act verb="window.set" args={{ kind: window.kind, at: '' }} disabled={result?.pending}
        label={hasRule ? 'Back to the rulebook' : 'Clear date'} /> : null}
    {result?.pending ? <p role="status">Saving league date…</p> : null}
    {result?.error ? <p role="alert">{result.error}</p> : null}
  </li>;
}
