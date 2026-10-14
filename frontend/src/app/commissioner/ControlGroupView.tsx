import { lazy, Suspense } from 'react';
import { Act } from '../commands/Act';
import { ControlRow } from './ControlRow';
import type { Entry } from './controls';
import { leagueState } from './proxy';
import { RoleForm } from './RoleForm';
import { RuleChangeForm } from './RuleChangeForm';
import type { RuleRow } from './rules';
import { StartLeagueForm } from './StartLeagueForm';
import { scopedRules, type Reveal } from './landing';

const LeagueRules = lazy(() => import('./LeagueRules'));
const LeaguePeople = lazy(() => import('./LeaguePeople'));
const LeagueLedger = lazy(() => import('./LeagueLedger'));

export type GroupEntries = { id: string; label: string; entries: Entry[] };
type Props = {
  group: GroupEntries; open: boolean; reveal?: Reveal;
  refer: (id: string, element: HTMLLIElement | null) => void;
};

// What a group shows besides its controls: the screens that already exist, placed with the controls they
// answer for.
function Existing({ id }: { id: string }) {
  const { choices, people, pending } = leagueState.use();
  if (id === 'constitution') return (
    <>
      {choices.length === 0 && !pending ? <StartLeagueForm /> : (
        <details><summary>Start another app league</summary><StartLeagueForm /></details>
      )}
      <LeagueRules />
    </>
  );
  if (id === 'people') return (
    <>
      <LeaguePeople />
      {people?.mayGrantRoles && <RoleForm key={people.league} people={people} />}
    </>
  );
  return id === 'ledger' ? <LeagueLedger /> : null;
}

// ControlGroupView is one collapsible group of the census. Only the open group is drawn, so there is
// only ever one rule form on the page.
export function ControlGroupView({ group, open, reveal, refer }: Props) {
  const { rules, people } = leagueState.use();
  const available = group.entries.filter(e => e.built).length;
  const owned = new Set(group.entries.flatMap(e => e.fields.map(f => f.field)));
  const rows = new Map(rules?.sections.flatMap(s => s.rows).map((r): [string, RuleRow] => [r.field, r]));
  const mayChange = people?.mayChangeRules === true;
  const mine = reveal && group.entries.some(e => e.id === reveal.control) ? reveal : undefined;
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
          {rules && mayChange && owned.size > 0 && (
            <RuleChangeForm key={`${group.id}:${mine?.n ?? 0}`} rules={scopedRules(rules, owned)}
              initialField={mine?.field ?? ''} />
          )}
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
