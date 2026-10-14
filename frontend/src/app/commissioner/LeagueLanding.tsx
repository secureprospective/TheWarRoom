import { useEffect, useMemo, useRef, useState } from 'react';
import { ControlGroupView } from './ControlGroupView';
import { ControlSearch } from './ControlSearch';
import { entriesOf } from './controls';
import { DensityScreen } from './DensityScreen';
import { JobsStrip } from './JobsStrip';
import { parseOpen, type LandingView } from './landing';
import { LeaguePicker } from './LeaguePicker';
import { leagueState } from './proxy';
import { surfaceActions } from './surfaceActions';
import { revealWorkSection } from '../shell/workScroll';
import './commissioner.css';

// LeagueLanding is the front of Control Room › League: the jobs strip, a search over every control, then
// the control groups. Go decides every job and every control; this only presents them.
export default function LeagueLanding() {
  const { choices, selected, jobs, controls, landingError, error, pending } = leagueState.use();
  const [view, setView] = useState<LandingView>({ open: null });
  const rows = useRef(new Map<string, HTMLLIElement>());
  const entries = useMemo(() => (controls ? entriesOf(controls) : []), [controls]);
  useEffect(() => { void leagueState.choose(); }, []);
  useEffect(() => surfaceActions.register('landing', args => {
    if (args.action === 'group') {
      setView(v => ({ ...v, open: v.open === args.value ? '' : args.value, reveal: undefined }));
      return;
    }
    const found = args.action === 'open' ? parseOpen(args.value, entries) : null;
    if (found) setView(v => ({ open: found.control.group,
      reveal: { control: found.control.id, field: found.field, n: (v.reveal?.n ?? 0) + 1 } }));
  }), [entries]);
  // Once the opened group is drawn, move to the control it was opened for.
  useEffect(() => {
    const row = view.reveal && rows.current.get(view.reveal.control);
    if (row) revealWorkSection(row);
  }, [view.reveal]);
  const noLeague = !pending && choices.length === 0;
  const open = view.open ?? (noLeague ? 'constitution' : '');
  const refer = (id: string, element: HTMLLIElement | null) => {
    if (element) rows.current.set(id, element); else rows.current.delete(id);
  };
  return (
    <DensityScreen screen="landing" label="League landing" pending={pending}>
      <div className="league-landing">
        {error && <p role="alert">{error}</p>}
        {landingError && <p role="alert">{landingError}</p>}
        <LeaguePicker />
        {noLeague && <p>No app league has been started yet. Start one under Constitution.</p>}
        {jobs && <JobsStrip jobs={jobs} />}
        {!jobs && selected && pending && <p role="status">Reading the league’s jobs…</p>}
        {entries.length > 0 && <ControlSearch entries={entries} />}
        {controls?.groups.map(group => (
          <ControlGroupView key={group.id} open={open === group.id} reveal={view.reveal} refer={refer}
            group={{ id: group.id, label: group.label,
              entries: entries.filter(e => e.group === group.id) }} />
        ))}
      </div>
    </DensityScreen>
  );
}
