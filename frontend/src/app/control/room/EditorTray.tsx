import { Act } from '../../commands/Act';
import { CommissionerActForm } from '../../commissioner/CommissionerActTray';
import { NoteForm } from '../../commissioner/NoteForm';
import { RoleForm } from '../../commissioner/RoleForm';
import { RuleChangeForm } from '../../commissioner/RuleChangeForm';
import { StartLeagueForm } from '../../commissioner/StartLeagueForm';
import { UndoForm } from '../../commissioner/UndoForm';
import { BackupNow } from '../../commissioner/BackupNow';
import type { Route } from './route';
import { DataActs } from './DataActs';
import MoveMFL from '../MoveMFL';
import { useRoom } from './room';
import '../../commissioner/commissioner.css';

export type TrayContent =
  | { kind: 'route'; route: Exclude<Route, { kind: 'unbuilt' } | { kind: 'draft' }> }
  | { kind: 'rule'; field: string }
  | { kind: 'undo'; entry: string };

/** The mock's editor tray above the command line. It holds the app's existing forms, so every change
 *  keeps its preview, its safeguard (click, hold or typed reason) and its ledger receipt. */
export function EditorTray({ content }: { content: TrayContent }) {
  const room = useRoom();
  const { rules, people, ledger } = room.state;
  let body;
  let label = 'Change';
  if (content.kind === 'rule') {
    label = 'Change a rule';
    body = rules ? (
      <RuleChangeForm key={`${rules.league}:${content.field}`} rules={rules} initialField={content.field} />
    ) : (
      <p role="status">Reading the league rules…</p>
    );
  } else if (content.kind === 'undo') {
    label = 'Undo this change';
    const entry = ledger?.entries.find((e) => e.id === content.entry);
    body =
      ledger && entry ? (
        <UndoForm key={entry.id} reading={ledger} entry={entry} />
      ) : (
        <p role="status">Reading the ledger…</p>
      );
  } else if (content.route.kind === 'act') {
    label = content.route.definition.label;
    body = (
      <CommissionerActForm
        key={content.route.definition.command}
        definition={content.route.definition}
        league={room.league}
        initial={content.route.initial}
      />
    );
  } else if (content.route.kind === 'migrate') {
    label = 'Move my MFL league';
    body = <MoveMFL />;
  } else if (content.route.kind === 'data') {
    label = content.route.mode === 'clear' ? 'Clear a cache' : 'Refresh now';
    body = <DataActs mode={content.route.mode} />;
  } else if (content.route.kind === 'backup') {
    label = 'Back up now';
    body = <BackupNow />;
  } else {
    const a = content.route.action;
    label = {
      start: 'Start an app league',
      role: 'Give or take a role',
      notice: 'Post an official notice',
      note: 'Post a league note',
      undo: 'Undo this change',
    }[a.kind];
    body =
      a.kind === 'start' ? (
        <StartLeagueForm />
      ) : !people ? (
        <p role="status">Reading the people…</p>
      ) : a.kind === 'role' ? (
        <RoleForm key={people.league} people={people} />
      ) : a.kind === 'notice' || a.kind === 'note' ? (
        <NoteForm key={`${people.league}:${a.kind}`} people={people} kind={a.kind} />
      ) : null;
  }
  return (
    <div className="cr-ed open" role="region" aria-label={label}>
      <div className="f" style={{ flex: 1 }}>
        {body}
      </div>
      <div className="f" style={{ alignSelf: 'flex-end', gap: 6 }}>
        <Act
          className="cr-btn"
          verb="commissioner.surface"
          variant="surface"
          label={'CANCEL'}
          args={{
            screen: 'control-room',
            action: 'close',
            value: 'tray',
          }}
          attributes={{ 'data-x': '' }}
        >
          CANCEL
        </Act>
      </div>
    </div>
  );
}
