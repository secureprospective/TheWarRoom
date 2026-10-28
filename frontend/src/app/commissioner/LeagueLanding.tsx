import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import { ControlGroupView } from './ControlGroupView';
import { ControlSearch } from './ControlSearch';
import { entriesOf } from './controls';
import { DensityScreen } from './DensityScreen';
import { actsFor, liveDraftControls } from './actTable';
import { commands } from '../commands/registry';
import { draftRoom } from '../draftroom/state';
import { JobsStrip } from './JobsStrip';
import { startingJob, startupDraftJob } from './jobs';
import { parseOpen, parseTray, scopedRules, type ActionKind, type LandingView } from './landing';
import { LeaguePicker } from './LeaguePicker';
import type { People } from './people';
import { leagueState } from './proxy';
import { surfaceActions } from './surfaceActions';
import { revealWorkSection } from '../shell/workScroll';
import './commissioner.css';

const RuleTray = lazy(() => import('./RuleTray'));
const CommissionerActTray = lazy(() => import('./CommissionerActTray'));
const SeedTray = lazy(() => import('./SeedTray'));
const ActionTray = lazy(() => import('./ActionTray'));

// Which change is open in the tray, and the control that opened it (focus goes back there).
type Tray = { opener: HTMLElement } & (
  { kind: 'rule'; field: string; group: string } | { kind: 'act'; control: string }
  | { kind: 'seed' } | ActionKind);

// Go says what this person may start; a launcher the person cannot use opens nothing.
function mayOpen(action: ActionKind, people?: People): boolean {
  if (action.kind === 'role') return people?.mayGrantRoles === true;
  return action.kind === 'notice' || action.kind === 'note' ? people?.mayPostNotices === true : true;
}

// LeagueLanding is the front of Control Room › League: the jobs strip, a search over every control, then
// the control groups. Go decides every job and every control; this only presents them.
export default function LeagueLanding() {
  const { choices, selected, jobs, controls, rules, people, landingError, error, pending }
    = leagueState.use();
  const [tray, setTray] = useState<Tray>();
  const background = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<LandingView>({ open: null });
  const rows = useRef(new Map<string, HTMLLIElement>());
  const entries = useMemo(() => (controls ? entriesOf(controls) : []), [controls]);
  useEffect(() => { void leagueState.choose(); }, []);
  useEffect(() => surfaceActions.register('landing', args => {
    if (args.action === 'close') {
      setTray(undefined);
      return;
    }
    if (args.action === 'group') {
      setTray(undefined);
      setView(v => ({ ...v, open: v.open === args.value ? '' : args.value, reveal: undefined }));
      return;
    }
    if (args.action === 'open' && args.value === startupDraftJob) {
      draftRoom.choose(true);
      commands.dispatch('nav.open', { node: 'trade', workspace: 'draft-room' });
      return;
    }
    const opener = document.activeElement as HTMLElement;
    if (args.action === 'tray' && args.value === 'seed' && selected && startingJob(jobs)) {
      setTray({ kind: 'seed', opener });
      return;
    }
    const action = args.action === 'tray' ? parseTray(args.value) : null;
    if (action && mayOpen(action, people)) {
      setTray({ ...action, opener });
      return;
    }
    const found = args.action === 'open' ? parseOpen(args.value, entries) : null;
    if (found?.field && rules && people?.mayChangeRules) {
      setTray({ kind: 'rule', field: found.field, group: found.control.group, opener });
      return;
    }
    if (found?.control.built && !found.field) {
      if (found.control.id === 'CM-040' && rules && people?.mayChangeRules) {
        setTray({ kind: 'rule', field: found.control.fields[0]?.field ?? '',
          group: found.control.group, opener });
        return;
      }
      if (liveDraftControls.has(found.control.id)) {
        draftRoom.choose(false);
        commands.dispatch('nav.open', { node: 'trade', workspace: 'draft-room' });
        return;
      }
      if (found.control.id === 'CM-021' && people?.mayGrantRoles) {
        setTray({ kind: 'role', opener });
        return;
      }
      if (actsFor(found.control.id).length && selected) {
        setTray({ kind: 'act', control: found.control.id, opener });
        return;
      }
    }
    if (found) setView(v => ({ open: found.control.group,
      reveal: { control: found.control.id, field: found.field, n: (v.reveal?.n ?? 0) + 1 } }));
  }), [entries, rules, people, selected, jobs]);
  // Starting a league moves the picker to the new league; the tray stays, to show what was recorded.
  useEffect(() => { setTray(t => (t?.kind === 'start' ? t : undefined)); }, [selected]);
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
      <div className="league-landing" ref={background} aria-hidden={tray ? true : undefined}>
        {error && <p role="alert">{error}</p>}
        {landingError && <p role="alert">{landingError}</p>}
        <LeaguePicker />
        {noLeague && <p>No app league has been started yet. Start one under Constitution.</p>}
        {jobs && <JobsStrip jobs={jobs} />}
        {!jobs && selected && pending && <p role="status">Reading the league’s jobs…</p>}
        {entries.length > 0 && <ControlSearch entries={entries} />}
        {controls?.groups.map(group => (
          <ControlGroupView key={group.id} open={open === group.id} refer={refer}
            group={{ id: group.id, label: group.label,
              entries: entries.filter(e => e.group === group.id) }} />
        ))}
      </div>
      {tray?.kind === 'rule' && rules && <Suspense fallback={<p role="status">Opening the rule change…</p>}>
        <RuleTray key={`${rules.league}:${tray.field}`} field={tray.field} opener={tray.opener}
          background={background.current} onClose={() => setTray(undefined)}
          rules={scopedRules(rules, new Set(entries
            .filter(e => e.group === tray.group).flatMap(e => e.fields.map(f => f.field))))} />
      </Suspense>}
      {tray?.kind === 'act' && <Suspense fallback={<p role="status">Opening the league action…</p>}>
        <CommissionerActTray key={`${selected}:${tray.control}`} control={tray.control} league={selected}
          opener={tray.opener} background={background.current} onClose={() => setTray(undefined)} />
      </Suspense>}
      {tray?.kind === 'seed' && startingJob(jobs) && <Suspense
        fallback={<p role="status">Opening starting assets…</p>}>
        <SeedTray key={selected} league={selected} opener={tray.opener}
          background={background.current} onClose={() => setTray(undefined)} />
      </Suspense>}
      {tray && tray.kind !== 'rule' && tray.kind !== 'act' && tray.kind !== 'seed' &&
        <Suspense fallback={<p role="status">Opening the change…</p>}>
        <ActionTray action={tray} opener={tray.opener} background={background.current}
          onClose={() => setTray(undefined)} />
      </Suspense>}
    </DensityScreen>
  );
}
