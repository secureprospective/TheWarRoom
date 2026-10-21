import { useEffect, useState } from 'react';
import { array } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import { ReportView } from '../reports/ReportView';
import { commands } from '../commands/registry';
import { readNFLDraft } from '../draftreports/api';
import { targetRosterReports } from '../contracts/api';

export default function RosterReports({ mine, draft = false }: { mine: string; draft?: boolean }) {
  const [reports, setReports] = useState<Report[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    const read = async () => {
      try {
        if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
          throw new Error('Roster reports need the desktop app; no sample data is substituted.');
        const reports = draft ? [await readNFLDraft()]
          : array(await targetRosterReports(mine), 'Roster reports', report);
        if (active) { setReports(reports); setError(''); }
      } catch (cause) { if (active) setError(String(cause)); }
    };
    void read();
    const stop = commands.alerts.onSeasonChange(read);
    return () => { active = false; stop(); };
  }, [mine, draft]);
  return <section aria-label={draft ? 'NFL draft' : 'Roster stats and byes'}>
    {error && <p role="alert">{error}</p>}
    {!reports && !error && <p role="status">
      {draft ? 'Reading NFL draft…' : 'Reading roster stats and bye weeks…'}</p>}
    {reports?.map(r => <ReportView key={r.id} report={r} howLabel="How this is counted" />)}
  </section>;
}
