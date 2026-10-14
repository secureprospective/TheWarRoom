import { lazy, Suspense, useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportView } from '../reports/ReportView';
import { controlState, matching } from './state';
import './control.css';

const LeagueDesk = lazy(() => import('../commissioner/LeagueDesk'));
const LeagueLedger = lazy(() => import('../commissioner/LeagueLedger'));
const LeagueRules = lazy(() => import('../commissioner/LeagueRules'));
const LeaguePeople = lazy(() => import('../commissioner/LeaguePeople'));

// LeagueRoom is Control Room › League: the league's settings, its own by-laws and accounting.
export function LeagueRoom() {
  const s = controlState.use();
  useEffect(() => { void controlState.choose('league'); }, []);
  const sections = s.bylaws ? matching(s.bylaws, s.search) : [];
  return (
    <section className="control-room" aria-busy={s.pending}>
      {s.error && <p role="alert">{s.error}</p>}
      {s.settings && <ReportView report={s.settings} howLabel="Where this comes from" />}
      <section className="bylaws" data-endpoint="M-185" aria-label="By-laws">
        <h4>By-laws</h4>
        <p className="report-source">Source: the league's own rulebook, as written by the league</p>
        <label htmlFor="bylaw-search">Search the by-laws</label>
        <input className="search-input" id="bylaw-search" type="search" value={s.search}
          onChange={e => commands.dispatch('control.read', { action: 'search', value: e.target.value })} />
        {s.bylaws && sections.length === 0 && <p>No section mentions “{s.search}”.</p>}
        {sections.map(section => (
          <details key={section.title} open={s.search !== ''}>
            <summary>{section.title}</summary>
            {section.lines.map((line, i) => <p key={i}>{line}</p>)}
          </details>
        ))}
      </section>
      <Suspense fallback={<p role="status">Reading the commissioner side…</p>}>
        <LeagueDesk />
        <LeagueLedger />
        <LeagueRules />
        <LeaguePeople />
      </Suspense>
      {s.accounting && <ReportView report={s.accounting} howLabel="How this was checked" />}
      <Act variant="button"
        verb="control.read" args={{ action: 'open-accounting', value: '' }}>Open MFL accounting</Act>
    </section>
  );
}
