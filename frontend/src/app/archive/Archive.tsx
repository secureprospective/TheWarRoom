import { useEffect } from 'react';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { archiveState, type Archive as ArchiveBoard } from './state';

const franchiseReports = ['M-104', 'M-105', 'M-023', 'M-110', 'M-019'];
const seasonReports = ['M-111', 'M-112', 'M-113', 'M-174', 'M-200', 'M-201', 'M-079', 'M-089'];
export function ArchiveView({ archive, selected }: { archive: ArchiveBoard; selected: readonly string[] }) {
  const shown = archive.board.reports.find(r => selected.includes(r.id)) ?? archive.board.reports[0];
  const dispatch = commands.dispatch;
  const controls = <div className="market-controls race-choices" aria-label="Archive report choices">
    {seasonReports.includes(shown.id) && <label>Season{' '}
      <select value={String(archive.season)} disabled={!archive.seasons.length}
        onChange={e => dispatch('archive.reports', { action: 'season', value: e.target.value })}>
        {archive.seasons.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
      </select>
    </label>}
    {franchiseReports.includes(shown.id) && <label>Franchise{' '}
      <select value={archive.team}
        onChange={e => dispatch('archive.reports', { action: 'team', value: e.target.value })}>
        <option value="">Choose a franchise</option>
        {archive.teams.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
      </select>
    </label>}
    {shown.id === 'M-019' && <label>Opponent{' '}
      <select value={archive.other}
        onChange={e => dispatch('archive.reports', { action: 'other', value: e.target.value })}>
        <option value="">Choose another franchise</option>
        {archive.teams.filter(c => c.value !== archive.team)
          .map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
      </select>
    </label>}
  </div>;
  return <ReportsView board={archive.board} selected={[shown.id]} refreshVerb="archive.reports"
    controls={controls} />;
}
export default function Archive() {
  const state = archiveState.use();
  const selected = commands.use().endpointIds;
  useEffect(() => {
    void archiveState.load();
    let disposed = false;
    let stop: (() => void) | undefined;
    void import('../../../wailsjs/runtime/runtime').then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void archiveState.load(); });
    });
    return () => { disposed = true; stop?.(); };
  }, []);
  return <section aria-label="League archive and pools">
    {state.pending && <p role="status">{state.archive?.board.loading ?? 'Reading the league archive…'}</p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.archive && <ArchiveView archive={state.archive} selected={selected} />}
  </section>;
}
