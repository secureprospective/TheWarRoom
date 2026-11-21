import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';

export type PlayerBio = {
  height?: string;
  weight?: string;
  born?: string;
  age?: number;
  college?: string;
  draft?: string;
  headshot?: string;
  jersey?: string;
  team?: string;
  position?: string;
  status?: string;
  injury?: string;
};

export type NewsItem = {
  id: string;
  headline: string;
  description?: string;
  published: string;
  url?: string;
  source: string;
  type?: string;
  injury?: boolean;
};

export type PlayerNews = {
  league: string;
  player: string;
  espnId: string;
  bio: PlayerBio;
  items: NewsItem[];
  asOf: string;
  stale: boolean;
  reason?: string;
};

export type FeedNewsItem = NewsItem & { players: string[] };

export type NewsFeed = {
  league: string;
  items: FeedNewsItem[];
  asOf: string;
  stale: boolean;
  reason?: string;
};

const CT = 'America/Chicago';

function asObj(v: unknown): Record<string, unknown> | undefined {
  return v && typeof v === 'object' && !Array.isArray(v) ? v as Record<string, unknown> : undefined;
}

function asStr(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

function asNum(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined;
}

function parseBio(raw: unknown): PlayerBio {
  const o = asObj(raw);
  if (!o) return {};
  const age = asNum(o.age);
  return {
    height: asStr(o.height) || undefined,
    weight: asStr(o.weight) || undefined,
    born: asStr(o.born) || undefined,
    age: age && age > 0 ? age : undefined,
    college: asStr(o.college) || undefined,
    draft: asStr(o.draft) || undefined,
    headshot: asStr(o.headshot) || undefined,
    jersey: /^\d{1,2}$/.test(asStr(o.jersey)) ? asStr(o.jersey) : undefined,
    team: asStr(o.team) || undefined,
    position: asStr(o.position) || undefined,
    status: asStr(o.status) || undefined,
    injury: asStr(o.injury) || undefined,
  };
}

function parseItem(raw: unknown): NewsItem | undefined {
  const o = asObj(raw);
  if (!o) return undefined;
  const headline = asStr(o.headline).trim();
  if (!headline) return undefined;
  return {
    id: asStr(o.id) || headline,
    headline,
    description: asStr(o.description).trim() || undefined,
    published: asStr(o.published),
    url: safeURL(asStr(o.url)),
    source: asStr(o.source) || 'ESPN',
    type: asStr(o.type) || undefined,
    injury: o.injury === true || undefined,
  };
}

function safeURL(raw: string): string | undefined {
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  return /^https?:\/\//i.test(trimmed) ? trimmed : undefined;
}

function sortNewest<T extends { published: string; id: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const ta = Date.parse(a.published) || 0;
    const tb = Date.parse(b.published) || 0;
    return tb !== ta ? tb - ta : b.id.localeCompare(a.id);
  });
}

export function parsePlayerNews(raw: unknown): PlayerNews | undefined {
  const o = asObj(raw);
  if (!o || !Array.isArray(o.items)) return undefined;
  const items = o.items.map(parseItem).filter((x): x is NewsItem => Boolean(x));
  return {
    league: asStr(o.league),
    player: asStr(o.player),
    espnId: asStr(o.espnId),
    bio: parseBio(o.bio),
    items: sortNewest(items),
    asOf: asStr(o.asOf),
    stale: o.stale === true,
    reason: asStr(o.reason) || undefined,
  };
}

export function parseNewsFeed(raw: unknown): NewsFeed | undefined {
  const o = asObj(raw);
  if (!o || !Array.isArray(o.items)) return undefined;
  const items: FeedNewsItem[] = [];
  for (const entry of o.items) {
    const item = parseItem(entry);
    const eo = asObj(entry);
    if (!item || !eo) continue;
    const players = Array.isArray(eo.players)
      ? eo.players.filter((p): p is string => typeof p === 'string' && p.length > 0) : [];
    items.push({ ...item, players });
  }
  return {
    league: asStr(o.league),
    items: sortNewest(items),
    asOf: asStr(o.asOf),
    stale: o.stale === true,
    reason: asStr(o.reason) || undefined,
  };
}

export async function readPlayerNews(
  league: string, player: string, port: HostedPort = hostedWorkflow,
): Promise<PlayerNews | undefined> {
  const { data } = await port({
    command: 'player.news.read', mode: 'read', arguments: { league, player },
  });
  return parsePlayerNews(data);
}

export async function readNewsFeed(
  league: string, port: HostedPort = hostedWorkflow,
): Promise<NewsFeed | undefined> {
  const { data } = await port({
    command: 'news.feed.read', mode: 'read', arguments: { league },
  });
  return parseNewsFeed(data);
}

export function formatBioBorn(born?: string, age?: number): string | undefined {
  if (!born) return undefined;
  const iso = /^(\d{4})-(\d{2})-(\d{2})$/.exec(born.trim());
  const formatted = iso
    ? new Date(Date.UTC(Number(iso[1]), Number(iso[2]) - 1, Number(iso[3])))
      .toLocaleDateString('en-US', {
        month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC',
      })
    : born.trim();
  return age !== undefined ? `${formatted} (${age})` : formatted;
}

export function ctDateTime(iso: string): string {
  const ms = Date.parse(iso);
  if (!Number.isFinite(ms)) return '';
  return `${new Date(ms).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZone: CT,
  })} CT`;
}

export function newsAge(iso: string, now = Date.now()): string {
  const ms = now - Date.parse(iso);
  if (!Number.isFinite(ms)) return '';
  if (ms < 60_000) return 'just now';
  if (ms < 3_600_000) return `${Math.max(1, Math.round(ms / 60_000))}m ago`;
  if (ms < 86_400_000) return `${Math.round(ms / 3_600_000)}h ago`;
  const days = Math.floor(ms / 86_400_000);
  return days === 1 ? 'yesterday' : `${days} days ago`;
}

export function shortNewsAge(iso: string, now = Date.now()): string {
  const ms = now - Date.parse(iso);
  if (!Number.isFinite(ms)) return '';
  if (ms < 60_000) return 'now';
  if (ms < 3_600_000) return `${Math.max(1, Math.round(ms / 60_000))}m`;
  if (ms < 86_400_000) return `${Math.round(ms / 3_600_000)}h`;
  return `${Math.floor(ms / 86_400_000)}d`;
}

const INJURY_RE = new RegExp(
  '\\b(' + [
    'injur\\w*', 'questionable', 'doubtful', 'ruled out', 'inactives?', 'ir', 'pup',
    'concussion', 'hamstring', 'ankle', 'knee', 'acl', 'mcl', 'achilles', 'groin',
    'quad\\w*', 'shoulder', 'calf', 'surgery', 'sprain\\w*', 'strain\\w*', 'fractur\\w*',
    'did not practice', 'limited practice', 'limited participant',
  ].join('|') + ')\\b',
  'i',
);

export function isInjuryNews(item: Pick<NewsItem, 'injury' | 'type' | 'headline' | 'description'>): boolean {
  if (item.injury) return true;
  if (item.type && /injur/i.test(item.type)) return true;
  return INJURY_RE.test(`${item.headline} ${item.description ?? ''}`);
}

export function openExternalArticle(url?: string): void {
  if (!url || !/^https?:\/\//i.test(url)) return;
  if (typeof window !== 'undefined' && typeof window.open === 'function') {
    window.open(url, '_blank', 'noopener,noreferrer');
  }
}
