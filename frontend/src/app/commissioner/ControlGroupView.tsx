import { lazy, Suspense } from 'react';
import { Act } from '../commands/Act';
import { ControlRow } from './ControlRow';
import type { Entry } from './controls';
import { leagueState } from './proxy';
import type { RuleRow } from './rules';
import { TrayLauncher } from './TrayLauncher';

const LeagueRules = lazy(() => import('./LeagueRules'));
const LeaguePeople = lazy(() => import('./LeaguePeople'));
const LeagueLedger = lazy(() => import('./LeagueLedger'));
const LeagueOps = lazy(() => import('./LeagueOps'));

export type GroupEntries = { id: string; label: string; entries: Entry[] };
type Props = {
  group: GroupEntries; open: boolean;
  refer: (id: string, element: HTMLLIElement | null) => void;
};

// What a group shows besides its controls: the screens that already exist, placed with the controls they
// answer for, and the way into each change. A change itself is made in the side tray, never in the group.
function Existing({ id }: { id: string }) {
  const { choices, people, pending } = leagueState.use();
  if (id === 'constitution') return (
    <>
      <p>A league run in this app. You become its commissioner.</p>
      <TrayLauncher tray="start" label={choices.length === 0 && !pending ? 'Start an app league'
        : 'Start another app league'} />
      <TrayLauncher tray="seed" label="Record starting assets" />
      <LeagueRules />
    </>
  );
  if (id === 'people') return (
    <>
      <LeaguePeople />
      {people?.mayGrantRoles && <TrayLauncher tray="role" label="Give or take a role" />}
    </>
  );
  if (id === 'ops') return <LeagueOps />;
  return id === 'ledger' ? <LeagueLedger /> : null;
}

// ControlGroupView draws only the open group of the census.
export function ControlGroupView({ group, open, refer }: Props) {
  const { rules, people } = leagueState.use();
  const available = group.entries.filter(e => e.built).length;
  const rows = new Map(rules?.sections.flatMap(s => s.rows).map((r): [string, RuleRow] => [r.field, r]));
  const mayChange = people?.mayChangeRules === true;
  return (
    <section className="control-group" aria-label={group.label} data-open={open}>
      <h4>
        <Act verb="commissioner.surface" variant="text" expanded={open}
          controls={`group-${group.id}`} args={{ screen: 'landing', action: 'group', value: group.id }}
          label={`${group.label}, ${available} of ${group.entries.length} available now`}>
          {group.label}{' '}
          <span className="control-count">{available} of {group.entries.length} available now</span>
        </Act>
      </h4>
      {open && (
        <div id={`group-${group.id}`} className="control-group-body">
          <Suspense fallback={<p role="status">Reading {group.label}…</p>}>
            <Existing id={group.id} />
          </Suspense>
          <ul className="control-rows">
            {group.entries.map(entry => (
              <ControlRow key={entry.id} entry={entry} rules={rows} mayChange={mayChange} refer={refer} />
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}
