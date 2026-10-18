import type { Change } from '../commissioner/forms';
import type { Draft } from './draft';

// What the tray can be opened for. A pick is named by its round, the franchise it began with, and for an
// extra pick added after it, which one.
export type PickRef = { round: number; original: string; extra: number };
export type TrayAction =
  | { kind: 'skip' | 'pause' | 'resume' | 'set' | 'restore' | 'close' }
  | { kind: 'assign' | 'insert' | 'change'; ref: PickRef };

const plain = ['skip', 'pause', 'resume', 'set', 'restore', 'close'] as const;
const onPick = ['assign', 'insert', 'change'] as const;

// The value a launcher sends: "skip", or "assign 1 franchise.03 0".
export const trayValue = (a: TrayAction) =>
  'ref' in a ? `${a.kind} ${a.ref.round} ${a.ref.original} ${a.ref.extra}` : a.kind;
export function parseTrayAction(value: string): TrayAction | null {
  const [kind, round, original, extra] = value.split(' ');
  const plainKind = plain.find(k => k === kind);
  if (plainKind) return round === undefined ? { kind: plainKind } : null;
  const pickKind = onPick.find(k => k === kind);
  const ref = { round: Number(round), original, extra: Number(extra) };
  return pickKind && Number.isInteger(ref.round) && ref.round > 0 && original &&
    Number.isInteger(ref.extra) && ref.extra >= 0 ? { kind: pickKind, ref } : null;
}

// What each kind says on screen. Go's preview words everything else.
export const trayWords = {
  skip: { title: 'Skip this pick', confirm: 'Skip this pick', close: 'Close skip' },
  assign: { title: 'Choose a rookie for a pick', confirm: 'Give this rookie to the pick',
    close: 'Close choose a rookie' },
  insert: { title: 'Add an extra pick', confirm: 'Add this extra pick', close: 'Close extra pick' },
  change: { title: 'Change a pick', confirm: 'Change this pick', close: 'Close change a pick' },
  pause: { title: 'Pause the pick clock', confirm: 'Pause the clock', close: 'Close pause' },
  resume: { title: 'Resume the pick clock', confirm: 'Resume the clock', close: 'Close resume' },
  set: { title: 'Set the pick clock', confirm: 'Set the clock', close: 'Close set the clock' },
  restore: { title: 'Restore the draft to an earlier point', confirm: 'Restore the draft',
    close: 'Close restore' },
  close: { title: 'Close the draft', confirm: 'Close the draft', close: 'Close the draft form' },
} as const;

export type Inputs = { player: string; franchise: string; minutes: string; checkpoint: string };
export const noInputs: Inputs = { player: '', franchise: '', minutes: '', checkpoint: '' };

const refOf = (r: PickRef) => ({ round: r.round, original: r.original,
  ...(r.extra > 0 ? { extra: r.extra } : {}) });
const commands = { skip: 'draft.skip', assign: 'draft.assign', insert: 'draft.insert',
  change: 'draft.change', pause: 'draft.timer', resume: 'draft.timer', set: 'draft.timer',
  restore: 'draft.revert', close: 'draft.archive' } as const;

// what is still missing before the change can be sent, in words; empty once it is whole.
export function missing(a: TrayAction, draft: Draft, i: Inputs): string {
  if (a.kind === 'skip' && !draft.onClock) return 'No pick is on the clock.';
  if (['assign', 'insert', 'change'].includes(a.kind) && i.player === '') return 'Choose a rookie.';
  if (a.kind === 'insert' && i.franchise === '') return 'Choose the franchise that gets the extra pick.';
  if (a.kind === 'set' && !(Number(i.minutes) >= 1 && Number.isInteger(Number(i.minutes))))
    return 'Enter the minutes the pick on the clock has left.';
  if (a.kind === 'restore' && i.checkpoint === '') return 'Choose the point to restore the draft to.';
  return '';
}

function actOf(a: TrayAction, draft: Draft, i: Inputs): object | undefined {
  switch (a.kind) {
    case 'skip': return draft.onClock ? { round: draft.onClock.round, original: draft.onClock.original } : {};
    case 'assign': return { ...refOf(a.ref), player: i.player };
    case 'insert': return { ...refOf(a.ref), player: i.player, franchise: i.franchise };
    case 'change': {
      const held = draft.picks.find(p => p.round === a.ref.round && p.original === a.ref.original &&
        p.extra === a.ref.extra);
      return { ...refOf(a.ref), player: i.player, replaces: held?.player ?? '' };
    }
    case 'pause': case 'resume': return { timer: a.kind };
    case 'set': return { timer: 'set', seconds: Number(i.minutes) * 60 };
    case 'restore': return { to: i.checkpoint };
    case 'close': return undefined;
  }
}

// changeOf is the exact change the tray sends through commissioner.execute, or null while it is not whole.
export function changeOf(a: TrayAction, draft: Draft, i: Inputs, reason: string): Change | null {
  if (missing(a, draft, i) !== '') return null;
  const act = actOf(a, draft, i);
  return { league: draft.league, command: commands[a.kind], reason: reason.trim(), expected: draft.revision,
    args: { draft: { season: draft.season, ...(act ? { act } : {}) } }, effective: 'now',
    visibility: 'league' };
}
