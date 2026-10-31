import type { Receipt } from './contract';
import { array, choice, object, requiredText, utc } from './parseValues';

export const NATIVE_MOVE_STATES = ['handed_off', 'needs_you', 'landed'] as const;
export type NativeMoveState = (typeof NATIVE_MOVE_STATES)[number];
export type NativeMove = Omit<Receipt, 'spec' | 'state' | 'audit'> & {
  spec: Omit<Receipt['spec'], 'target'> & { target: { kind: 'native' } };
  state: NativeMoveState;
  audit: { at: string; from: 'draft'; event: 'recorded'; to: NativeMoveState; note: string }[];
};
export type NativeMoves = {
  league: string; franchise: string; planned: NativeMove[]; reason: string; moves: NativeMove[];
};

// Native receipts prove committed league records, not an MFL browser hand-off.
function nativeMove(value: unknown, path: string, league: string, franchise: string): NativeMove {
  const r = object(value, path, ['correlationId', 'spec', 'state', 'audit']);
  const s = object(r.spec, `${path}.spec`, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected', 'gravity', 'undo', 'target', 'deadline',
  ]);
  if (s.leagueId !== league || s.franchiseId !== franchise) throw new Error(`${path}: wrong move scope`);
  const subject = object(s.subject, `${path}.spec.subject`, ['players', 'picks']);
  object(s.expected, `${path}.spec.expected`, []);
  const target = object(s.target, `${path}.spec.target`, ['kind']);
  const state = choice(r.state, `${path}.state`, NATIVE_MOVE_STATES);
  const audit = array(r.audit, `${path}.audit`, (v, p) => {
    const a = object(v, p, ['at', 'from', 'event', 'to', 'note']);
    return {
      at: utc(a.at, `${p}.at`), from: choice(a.from, `${p}.from`, ['draft'] as const),
      event: choice(a.event, `${p}.event`, ['recorded'] as const),
      to: choice(a.to, `${p}.to`, NATIVE_MOVE_STATES), note: requiredText(a.note, `${p}.note`),
    };
  });
  if (audit.length !== 1 || audit[0].to !== state) throw new Error(`${path}: expected recorded state`);
  return {
    correlationId: requiredText(r.correlationId, `${path}.correlationId`), state, audit,
    spec: {
      intent: requiredText(s.intent, `${path}.spec.intent`), leagueId: league, franchiseId: franchise,
      subject: {
        players: array(subject.players, `${path}.spec.subject.players`, requiredText),
        picks: array(subject.picks, `${path}.spec.subject.picks`, requiredText),
      },
      expected: {}, gravity: choice(s.gravity, `${path}.spec.gravity`, ['G0'] as const),
      undo: choice(s.undo, `${path}.spec.undo`, ['irreversible'] as const),
      target: { kind: choice(target.kind, `${path}.spec.target.kind`, ['native'] as const) },
      ...(s.deadline === undefined ? {} : { deadline: utc(s.deadline, `${path}.spec.deadline`) }),
    },
  };
}

export function parseNativeMoves(value: unknown): NativeMoves {
  const r = object(value, 'moves', ['league', 'franchise', 'planned', 'reason', 'moves']);
  const league = requiredText(r.league, 'moves.league');
  if (!league.startsWith('native.')) throw new Error('moves.league: expected native league');
  const franchise = requiredText(r.franchise, 'moves.franchise');
  const parseMove = (v: unknown, p: string) => nativeMove(v, p, league, franchise);
  const planned = array(r.planned, 'moves.planned', parseMove);
  if (planned.length) throw new Error('moves.planned: native plans not yet stored');
  return {
    league, franchise, planned, reason: requiredText(r.reason, 'moves.reason'),
    moves: array(r.moves, 'moves.moves', parseMove),
  };
}
