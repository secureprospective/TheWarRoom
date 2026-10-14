import type { ActionKind } from './landing';
import { leagueState } from './proxy';
import { RoleForm } from './RoleForm';
import SideTray from './SideTray';
import { StartLeagueForm } from './StartLeagueForm';
import { UndoForm } from './UndoForm';

const names = {
  start: { title: 'Start an app league', close: 'Close league start' },
  role: { title: 'Give or take a role', close: 'Close role change' },
  undo: { title: 'Undo this change', close: 'Close undo' },
} as const;

// ActionTray holds the other three commissioner changes in the same side tray as a rule change. What a
// form needs (the people, the ledger) is read from the one league state, so it is fresh after a receipt.
export default function ActionTray({ action, opener, background, onClose }: {
  action: ActionKind; opener: HTMLElement; background: HTMLElement | null; onClose(): void;
}) {
  const { people, ledger } = leagueState.use();
  const entry = action.kind === 'undo' ? ledger?.entries.find(e => e.id === action.entry) : undefined;
  return (
    <SideTray title={names[action.kind].title} screen="landing" opener={opener} background={background}
      onClose={onClose} closeLabel={names[action.kind].close}>
      {action.kind === 'start' && <StartLeagueForm />}
      {action.kind === 'role' && people && <RoleForm key={people.league} people={people} />}
      {action.kind === 'undo' && ledger && entry &&
        <UndoForm key={entry.id} reading={ledger} entry={entry} />}
    </SideTray>
  );
}
