import { useEffect } from 'react';
import { LeaguePicker } from './LeaguePicker';
import { leagueState } from './proxy';
import { RoleForm } from './RoleForm';
import { RuleChangeForm } from './RuleChangeForm';
import { StartLeagueForm } from './StartLeagueForm';
import './commissioner.css';

// LeagueDesk is where a commissioner starts a change: the league picker and the three first forms.
// A form shows only when the people reading says this person's role may use it; Go still decides.
export default function LeagueDesk() {
  const { choices, rules, people, pending } = leagueState.use();
  useEffect(() => { void leagueState.choose(); }, []);
  return (
    <section className="league-desk" aria-label="Commissioner tools" aria-busy={pending}>
      <h4>Commissioner tools</h4>
      <LeaguePicker />
      {choices.length === 0 ? <StartLeagueForm /> : (
        <details><summary>Start another app league</summary><StartLeagueForm /></details>
      )}
      {rules && people?.mayChangeRules && <RuleChangeForm key={rules.league} rules={rules} />}
      {people?.mayGrantRoles && <RoleForm key={people.league} people={people} />}
    </section>
  );
}
