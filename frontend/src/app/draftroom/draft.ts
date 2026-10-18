import { array, choice, integer, object, text, timestamp } from '../data/parseValues';

// The rookie draft as Go reads it (TargetNativeDraft). Go orders the rounds, replays every pick and the
// clock, and words every refusal; this only parses what it sent and writes it for a GM.
export type Terms = { salaryCents: number; years: number; rfaAfter: boolean };
export const hows = ['', 'owner', 'commissioner', 'inserted', 'expired', 'skipped', 'overrode'] as const;
export type How = (typeof hows)[number];
export type Slot = { round: number; original: string; owner: string; slot: number };
export type Pick = Slot & {
  extra: number; player: string; skipped: boolean; forfeited: boolean; madeAt: string;
  contract: Terms | null; how: How; changed: boolean;
};
// remaining is nanoseconds, as Go counts time.
export type Clock = Slot & {
  openedAt: string; deadline: string; contract: Terms; paused: boolean; remaining: number;
};
export type Player = { id: string; name: string; position: string; team: string };
export type Closed = {
  at: string; automatic: boolean; pool: Player[];
  issued: { season: number; rounds: number; picks: number; tradable: boolean }[];
};
export type Checkpoint = { entry: string; words: string; at: string };
export type Draft = {
  league: string; season: number; revision: number; rules: { label: string; words: string }[];
  owners: { id: string; name: string }[]; order: { round: number; words: string; picks: Slot[] }[];
  words: string; start: string; started: boolean; exhausted: boolean; picks: Pick[]; onClock: Clock | null;
  next: Slot | null; closed: Closed | null; players: Record<string, Player>; available: Player[];
  checkpoints: Checkpoint[];
};

const slotKeys = ['round', 'original', 'owner', 'slot'];
const slot = (r: Record<string, unknown>, path: string): Slot => ({
  round: integer(r.round, path, 1), original: text(r.original, path), owner: text(r.owner, path),
  slot: integer(r.slot, path, 1),
});
function terms(value: unknown, path: string): Terms {
  const t = object(value, path, ['salaryCents', 'years', 'rfaAfter']);
  if (typeof t.rfaAfter !== 'boolean') throw new Error(`${path}.rfaAfter: expected true or false`);
  return { salaryCents: integer(t.salaryCents, path), years: integer(t.years, path), rfaAfter: t.rfaAfter };
}
function flag(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected true or false`);
  return value;
}
function player(value: unknown, path: string): Player {
  const p = object(value, path, ['id', 'name', 'position', 'team']);
  return { id: text(p.id, path), name: text(p.name, path), position: text(p.position, path),
    team: text(p.team, path) };
}
function parsePick(value: unknown, path: string): Pick {
  const r = object(value, path, [...slotKeys, 'extra', 'player', 'skipped', 'forfeited', 'madeAt',
    'contract', 'how', 'changed']);
  return {
    ...slot(r, path), extra: r.extra === undefined ? 0 : integer(r.extra, `${path}.extra`),
    player: r.player === undefined ? '' : text(r.player, `${path}.player`),
    skipped: flag(r.skipped, `${path}.skipped`), forfeited: flag(r.forfeited, `${path}.forfeited`),
    madeAt: r.madeAt === undefined ? '' : timestamp(r.madeAt, `${path}.madeAt`),
    contract: r.contract === undefined ? null : terms(r.contract, `${path}.contract`),
    how: r.how === undefined ? '' : choice(r.how, `${path}.how`, hows),
    changed: r.changed === undefined ? false : flag(r.changed, `${path}.changed`),
  };
}
function parseClock(value: unknown): Clock | null {
  if (value === null) return null;
  const r = object(value, 'Clock', [...slotKeys, 'openedAt', 'deadline', 'contract', 'paused', 'remaining']);
  return { ...slot(r, 'Clock'), openedAt: timestamp(r.openedAt, 'Clock.openedAt'),
    deadline: timestamp(r.deadline, 'Clock.deadline'), contract: terms(r.contract, 'Clock.contract'),
    paused: flag(r.paused, 'Clock.paused'), remaining: integer(r.remaining, 'Clock.remaining') };
}
function parseClosed(value: unknown): Closed | null {
  if (value === null) return null;
  const r = object(value, 'Closed draft', ['at', 'automatic', 'pool', 'issued']);
  return {
    at: timestamp(r.at, 'Closed.at'), automatic: flag(r.automatic, 'Closed.automatic'),
    pool: array(r.pool, 'Closed.pool', player),
    issued: array(r.issued, 'Closed.issued', (v, p) => {
      const i = object(v, p, ['season', 'rounds', 'picks', 'tradable']);
      return { season: integer(i.season, p, 1), rounds: integer(i.rounds, p), picks: integer(i.picks, p),
        tradable: flag(i.tradable, p) };
    }),
  };
}

export function parseDraft(value: unknown): Draft {
  const r = object(value, 'Rookie draft', ['league', 'season', 'revision', 'rules', 'owners', 'order',
    'words', 'draft', 'nextOwnedPick', 'closed', 'players', 'available', 'checkpoints']);
  const d = object(r.draft, 'Draft', ['start', 'started', 'exhausted', 'picks', 'onClock']);
  const players = object(r.players, 'Players', Object.keys(Object(r.players) as object));
  return {
    league: text(r.league, 'League'), season: integer(r.season, 'Season', 1),
    revision: integer(r.revision, 'League version', 1),
    rules: array(r.rules, 'Rules', (v, p) => {
      const l = object(v, p, ['label', 'words']);
      return { label: text(l.label, p), words: text(l.words, p) };
    }),
    owners: array(r.owners, 'Owners', (v, p) => {
      const o = object(v, p, ['id', 'name']);
      return { id: text(o.id, p), name: text(o.name, p) };
    }),
    order: array(r.order, 'Order', (v, p) => {
      const o = object(v, p, ['round', 'words', 'picks']);
      return { round: integer(o.round, p, 1), words: text(o.words, p),
        picks: array(o.picks, `${p}.picks`, (s, q) => slot(object(s, q, slotKeys), q)) };
    }),
    words: text(r.words, 'Draft words'), start: timestamp(d.start, 'Draft.start'),
    started: flag(d.started, 'Draft.started'), exhausted: flag(d.exhausted, 'Draft.exhausted'),
    picks: array(d.picks, 'Picks', parsePick), onClock: parseClock(d.onClock),
    next: r.nextOwnedPick === null ? null : slot(object(r.nextOwnedPick, 'Next pick', slotKeys), 'Next pick'),
    closed: parseClosed(r.closed),
    players: Object.fromEntries(Object.entries(players).map(([id, v]) => [id, player(v, `Players.${id}`)])),
    available: array(r.available, 'Available rookies', player),
    checkpoints: array(r.checkpoints, 'Checkpoints', (v, p) => {
      const c = object(v, p, ['entry', 'words', 'at']);
      return { entry: text(c.entry, p), words: text(c.words, p), at: timestamp(c.at, p) };
    }),
  };
}
