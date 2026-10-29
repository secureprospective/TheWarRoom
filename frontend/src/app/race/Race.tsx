import { useEffect } from 'react';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { ReportView } from '../reports/ReportView';
import { raceState } from './state';
import { subscribeRace } from './api';
import type { Race as RaceBoard } from './contract';

const powerReports = ['M-008', 'C-02'];
const weekReports = ['M-009', 'M-010'];
const teamReports = ['M-010', 'M-012', 'M-077'];

export function RaceView({ race, selected, aggMode = 'lineup' }: {
  race: RaceBoard; selected: readonly string[]; aggMode?: string;
}) {
  const shown = race.board.reports.find(r => selected.includes(r.id)) ?? race.board.reports[0];
  const dispatch = commands.dispatch;
  const hasChoices = [powerReports, weekReports, teamReports].some(ids => ids.includes(shown.id));
  const controls = hasChoices &&
    <div className="market-controls race-choices" aria-label="Report choices">
      {powerReports.includes(shown.id) && <label>Roster count{' '}
        <select value={aggMode}
          onChange={e => dispatch('race.reports', { action: 'aggMode', value: e.target.value })}>
          <option value="sum">Whole roster</option>
          <option value="lineup">Starting lineup</option>
          <option value="topn">Top N</option>
        </select>
      </label>}
      {weekReports.includes(shown.id) && <label>Week{' '}
        <select value={String(race.week)} disabled={race.weeks.length === 0}
          onChange={e => dispatch('race.reports', { action: 'week', value: e.target.value })}>
          {race.weeks.map(w => <option key={w.value} value={w.value}>{w.label}</option>)}
        </select>
      </label>}
      {teamReports.includes(shown.id) && <label>Franchise{' '}
        <select value={race.team}
          onChange={e => dispatch('race.reports', { action: 'team', value: e.target.value })}>
          {!race.team && <option value="">Choose a franchise</option>}
          {race.teams.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
      </label>}
    </div>;
  return <>
    <div className="draft-reports race-standings">
      <ReportView report={race.standings} howLabel={race.board.howLabel} heading="h4" />
    </div>
    <ReportsView board={race.board} selected={[shown.id]} refreshVerb="race.reports" controls={controls} />
  </>;
}

export default function Race() {
  const state = raceState.use();
  const selected = commands.use().endpointIds;
  const mine = commands.use().franchiseId;
  useEffect(() => {
    if (!raceState.peek().pending) void raceState.load(true);
    return subscribeRace(() => { void raceState.load(false); });
  }, []);
  useEffect(() => {
    if (raceState.peek().race) void raceState.load(false);
  }, [mine]);
  return <section className="race" aria-label="League race">
    {state.pending && <p role="status">{state.race?.board.loading ?? 'Reading the league race…'}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.race && <RaceView race={state.race} selected={selected} aggMode={state.aggMode} />}
  </section>;
}
