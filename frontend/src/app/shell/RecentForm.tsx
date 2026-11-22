import type { PulseFormResult } from '../data/nativePulse';

const labels: Record<PulseFormResult['result'], string> = { W: 'Win', L: 'Loss', T: 'Tie', B: 'Bye' };

/** RecentForm renders the server-results outcomes; it does not decide any outcome itself. */
export function RecentForm({ form, className = '' }: { form: PulseFormResult[]; className?: string }) {
  const description = form.map(item => `Week ${item.week}: ${labels[item.result]}`).join(', ');
  return <span className={`recent-form-marks ${className}`.trim()} aria-label={description || 'No results'}>
    {form.map(item => <i key={item.week} className={item.result.toLowerCase()}
      title={`Week ${item.week} · ${labels[item.result]}`}>{item.result}</i>)}
  </span>;
}
