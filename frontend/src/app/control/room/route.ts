import { actTable, type ActDefinition } from '../../commissioner/actTable';
import type { ActionKind } from '../../commissioner/landing';

/** Where one of the mock's command buttons goes in the real app. Every button runs an existing command
 *  through its existing form (preview, confirm, typed reason); a command with no form says so. */
export type Route =
  | { kind: 'act'; definition: ActDefinition }
  | { kind: 'action'; action: ActionKind }
  | { kind: 'backup' }
  /** The live draft's controls (revert, archive, timer) are in the Draft room's commissioner tray. */
  | { kind: 'draft' }
  /** App data: refresh a source or clear a cache (the Data and sources commands). */
  | { kind: 'data'; mode: 'refresh' | 'clear' }
  /** Move an MFL league into the app (the existing import). */
  | { kind: 'migrate' }
  | { kind: 'unbuilt'; command: string };

/** The forms that already exist outside the act table. */
const actions: Record<string, ActionKind> = {
  'commish.notice.publish': { kind: 'notice' },
  'commish.whatsnew.post': { kind: 'note' },
  'commish.role.grant': { kind: 'role' },
  'commish.role.revoke': { kind: 'role' },
  'commish.league.create': { kind: 'start' },
};

export function routeCommand(command: string): Route {
  const name = command.split(' ')[0]!;
  if (name === 'backup.now') return { kind: 'backup' };
  if (name === 'league.migrate.prepare') return { kind: 'migrate' };
  if (name === 'data.refresh') return { kind: 'data', mode: 'refresh' };
  if (name === 'cache.clear') return { kind: 'data', mode: 'clear' };
  if (['draft.revert', 'draft.archive', 'draft.timer', 'draft.skip', 'draft.assign'].includes(name))
    return { kind: 'draft' };
  const action = actions[name];
  if (action) return { kind: 'action', action };
  const definition = actTable.find(a => a.command === name);
  return definition ? { kind: 'act', definition } : { kind: 'unbuilt', command: name };
}

/** The mock's words for a command no form runs yet. */
export const unbuiltWords = (command: string) => `Not built yet: nothing would change. (${command})`;
