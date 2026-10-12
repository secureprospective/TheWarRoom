import { array, object, text } from '../data/parseValues';
import { report, type Report } from '../reports/contract';
import { choice, type Choice } from '../race/contract';

export type Comms = {
  feed: Report; transactions: Report; events: { sentence: string; date: string }[];
  boardNote: string; leagueTalk: string; boardPosts: Report[];
  team: string; kind: string; teams: Choice[]; kinds: Choice[];
};
export function parseComms(value: unknown): Comms {
  const p = 'Comms';
  const r = object(value, p, ['feed', 'transactions', 'events', 'boardNote', 'leagueTalk',
    'boardPosts', 'team', 'kind', 'teams', 'kinds']);
  const result = {
    feed: report(r.feed, p), transactions: report(r.transactions, p),
    events: array(r.events, p, (value, p) => {
      const row = object(value, p, ['sentence', 'date']);
      return { sentence: text(row.sentence, p), date: text(row.date, p) };
    }),
    boardNote: text(r.boardNote, p), leagueTalk: text(r.leagueTalk, p),
    boardPosts: array(r.boardPosts, p, report), team: text(r.team, p), kind: text(r.kind, p),
    teams: array(r.teams, p, choice), kinds: array(r.kinds, p, choice),
  };
  if (result.feed.id !== 'C-05' || result.transactions.id !== 'M-032' ||
    result.boardPosts.map(r => r.id).join(',') !== 'M-120,M-123')
    throw new Error('Comms: unexpected reports');
  if (!result.teams.some(t => t.value === result.team) || !result.kinds.some(k => k.value === result.kind))
    throw new Error('Comms: filter outside the choices');
  return result;
}
