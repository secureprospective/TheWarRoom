import { useEffect, useState } from 'react';
import type { LeagueWindow } from '../data/contract';
import { Act } from '../commands/Act';
import { localDate, ruleDatedWindows, windowLabels } from './labels';
import { useWindowResults } from './windowState';

export function localInput(at?: string): string {
  if (!at) return '';
  const date = new Date(at);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

export function WindowRow({ window }: { window: LeagueWindow }) {
  const [date, setDate] = useState(localInput(window.at));
  useEffect(() => { setDate(localInput(window.at)); }, [window.at]);
  const result = useWindowResults()[window.kind];
  const label = windowLabels[window.kind];
  const hasRule = ruleDatedWindows.has(window.kind);
  const instant = date && !Number.isNaN(new Date(date).getTime()) ? new Date(date).toISOString() : '';
  return <li className="league-window" data-moved={window.moved || undefined}>
    <p><b>{label}</b> · {window.at ? localDate(window.at) : 'No date'}</p>
    <p className="league-window-basis">{window.reason}</p>
    <div className="league-window-edit">
      <label>{label} date{' '}
        <input type="datetime-local" value={date} onChange={event => setDate(event.target.value)} />
      </label>
      <Act variant="button" verb="window.set" args={{ kind: window.kind, at: instant }} label="Set date"
        disabled={!instant || result?.pending} />
      {window.moved || (!hasRule && window.at) ?
        <Act variant="button"
          verb="window.set" args={{ kind: window.kind, at: '' }} disabled={result?.pending}
          label={hasRule ? 'Back to the rulebook' : 'Clear date'} /> : null}
    </div>
    {result?.pending ? <p role="status">Saving league date…</p> : null}
    {result?.error ? <p role="alert">{result.error}</p> : null}
  </li>;
}
