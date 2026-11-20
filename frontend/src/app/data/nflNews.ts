import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';

/** NFLNewsItem is one public news line for a rostered player (nfl.news.read). */
export type NFLNewsItem = {
  at: string; player: string; name: string; position: string; nflTeam: string;
  tag: 'INJ' | 'OUT' | 'IR' | 'ACT' | 'BYE' | 'NOTE' | string; text: string; source: 'espn' | 'sleeper' | string;
};
export type NFLNews = { asOf: string | null; stale: boolean; items: NFLNewsItem[] };

function fail(what: string): never { throw new Error(`NFL news: ${what}`); }
function record(v: unknown, what: string): Record<string, unknown> {
  if (typeof v !== 'object' || v === null || Array.isArray(v)) fail(`${what} is not an object`);
  return v as Record<string, unknown>;
}
function text(v: unknown, what: string): string {
  if (typeof v !== 'string') fail(`${what} is not text`);
  return v;
}

/** parseNFLNews checks the nfl.news.read reply's shape; it trusts the server's labels and order. */
export function parseNFLNews(v: unknown): NFLNews {
  const r = record(v, 'reading');
  if (typeof r.stale !== 'boolean') fail('stale is not a flag');
  if (!Array.isArray(r.items)) fail('items is not a list');
  const asOf = r.asOf === null || r.asOf === undefined ? null : text(r.asOf, 'asOf');
  const items = r.items.map((raw, i) => {
    const it = record(raw, `item ${i}`);
    return { at: text(it.at, 'at'), player: text(it.player, 'player'), name: text(it.name, 'name'),
      position: text(it.position, 'position'), nflTeam: text(it.nflTeam, 'nflTeam'), tag: text(it.tag, 'tag'),
      text: text(it.text, 'text'), source: text(it.source, 'source') };
  });
  return { asOf, stale: r.stale, items };
}

/** readNFLNews asks the league's server for public news about its rostered players (optionally just these ids). */
export async function readNFLNews(league: string, players: string[] = [],
  port: HostedPort = hostedWorkflow): Promise<NFLNews> {
  const reply = await port({ command: 'nfl.news.read', mode: 'read',
    arguments: { league, players: players.join(',') } });
  return parseNFLNews(reply.data);
}
