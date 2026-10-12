import { array, choice, id, object, text } from '../data/parseValues';
import { parseBoard, report, type Board, type Report } from '../reports/contract';

export type Scope = {
  mine: string; search: string; position: string; team: string; group: string;
  trend: string; depth: string; injuries: string;
};
export type Player = { id: string; name: string; position: string; team: string; franchiseId: string };
export type Article = { id: string; article: string; name: string; headline: string };
export type Research = {
  board: Board; news: Report; articles: Article[]; players: Player[];
  teams: string[]; positions: string[]; scope: Scope;
};
export function parsePlayer(value: unknown, path: string): Player {
  const p = object(value, path, ['id', 'name', 'position', 'team', 'franchiseId']);
  const franchiseId = text(p.franchiseId, path);
  if (franchiseId && !/^\d{4}$/.test(franchiseId)) throw new Error(`${path}: invalid franchise`);
  return { id: id(p.id, path, true), name: text(p.name, path),
    position: text(p.position, path), team: text(p.team, path), franchiseId };
}
export function parseResearch(value: unknown): Research {
  const path = 'Player research';
  const r = object(value, path, ['board', 'news', 'articles', 'players', 'teams', 'positions', 'scope']);
  const s = object(r.scope, `${path}.scope`,
    ['mine', 'search', 'position', 'team', 'group', 'trend', 'depth', 'injuries']);
  const scope: Scope = {
    mine: text(s.mine, path), search: text(s.search, path), position: text(s.position, path),
    team: text(s.team, path), group: choice(s.group, path, ['QB', 'RB', 'WR/TE', 'K', 'IDP', 'DST']),
    trend: choice(s.trend, path, ['topAdds', 'topDrops', 'topStarters', 'topOwns', 'topTrades']),
    depth: choice(s.depth, path, ['team', 'grid']), injuries: choice(s.injuries, path, ['all', 'games']),
  };
  return {
    board: parseBoard(r.board), news: report(r.news, path), scope,
    teams: array(r.teams, path, text), positions: array(r.positions, path, text),
    players: array(r.players, path, parsePlayer),
    articles: array(r.articles, path, (value, path) => {
      const a = object(value, path, ['id', 'article', 'name', 'headline']);
      return { id: id(a.id, path, true), article: text(a.article, path),
        name: text(a.name, path), headline: text(a.headline, path) };
    }),
  };
}
