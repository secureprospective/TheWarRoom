import { commands } from '../commands/registry';
import { leagueState } from './proxy';

// The one native-league picker: the ledger, the rules, the people and every form follow it.
export function LeaguePicker() {
  const { choices, selected } = leagueState.use();
  if (choices.length === 0) return null;
  return (
    <label>App league{' '}
      <select value={selected || choices[0]?.id}
        onChange={e => commands.dispatch('commissioner.read', { league: e.target.value })}>
        {choices.map(choice => <option key={choice.id} value={choice.id}>{choice.name}</option>)}
      </select>
    </label>
  );
}
