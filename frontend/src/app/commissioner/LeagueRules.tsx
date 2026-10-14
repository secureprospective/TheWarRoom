import { ReportView } from '../reports/ReportView';
import { leagueState } from './proxy';
import { pendingWhen } from './rules';
import { rulesReport, scoringReports } from './rulesReport';
import './commissioner.css';

// LeagueRules is the read screen for a league's rules: Set or Needs a ruling, its source, what waits.
export default function LeagueRules() {
  const { rules, pending } = leagueState.use();
  if (!rules) return pending ? <p role="status">Reading league rules…</p> : null;
  const scoring = rules.sections.flatMap(s => s.rows).find(r => r.field === 'scoring.catalogue');
  const reports = scoring && scoring.state === 'Set' ? scoringReports(scoring) : null;
  return (
    <section className="league-rules" aria-label="League rules">
      <p>Next rollover: {rules.rollover}</p>
      <ReportView report={rulesReport(rules)} howLabel="How to read this" heading="h4" />
      {scoring && (
        <section aria-label="Scoring">
          <h4>Scoring</h4>
          {!reports && <p>{scoring.citation}</p>}
          {scoring.pending.map(p => <p key={p.entry}>{pendingWhen(p)} (a scoring change)</p>)}
          {reports && <ReportView report={reports.used} howLabel="How to read this" />}
          {reports?.unused && (
            <details><summary>{reports.unused.title}</summary>
              <ReportView report={reports.unused} howLabel="How to read this" /></details>
          )}
        </section>
      )}
    </section>
  );
}
