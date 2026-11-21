// Desktop v3 data: the admitted leagues dressed for the frame, and the Home reads behind the pieces.
// The reads are the thin Home's (gatherNeedsYou, readThinHomeBrief); only their presentation is v3.
import { useEffect, useState } from 'react';
import { parseHostedWaivers } from '../commissioner/hostedMarkets';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { NativeHQ } from '../commissioner/hqReading';
import type { Provenance, Snapshot } from '../data/contract';
import { readNewsFeed, type NewsFeed } from '../research/espnNews';
import { gatherNeedsYou, type NeedsYouItem } from '../shell/homeNeedsYou';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import { left } from '../shell/homeTiles';
import { leagueInitials, type LeagueChoice } from '../shell/leaguePicker';
import type { ThinHomeEntry } from '../shell/ThinHome';
import { isEndedTrade } from '../shell/warroomPlansReads';
import type { IconName } from './icons';

// League identity colours, in the mock's order (IDC).
const IDC = ['hsl(172,30%,52%)', 'hsl(210,28%,66%)', 'hsl(14,48%,58%)', 'hsl(75,26%,54%)',
  'hsl(30,62%,56%)', 'hsl(195,32%,60%)', 'hsl(40,40%,66%)', 'hsl(36,22%,72%)'];

export type V3League = LeagueChoice & { bd: string; c: string; commish: boolean; team: boolean };
export type Waivers = ReturnType<typeof parseHostedWaivers>;
export type Brief = { brief?: HomeBrief; error?: string; waivers?: Waivers; news?: NewsFeed };
export type Lane = 'need' | 'wait' | 'land';
export type DocketItem = { id: string; lg: string; lane: Lane; t: string; who: string; clock: string;
  btn: string; entry?: ThinHomeEntry; urgent?: boolean; bad?: boolean };
export type Open = (L: V3League, e: ThinHomeEntry) => void;

const isCommish = (role?: string) => role === 'Commissioner' || role === 'Co-commissioner';
export const roleWords = (L: V3League) => L.commish && L.team ? 'Commissioner + owner'
  : L.commish ? 'Commissioner' : 'Owner';

export function dress(admitted: LeagueChoice[]): V3League[] {
  return admitted.filter(l => l.app).map((l, i) => ({ ...l, bd: leagueInitials(l.name),
    c: IDC[i % IDC.length], commish: isCommish(l.role), team: Boolean(l.franchise) }));
}

// The league screens a league can open, in the rail's order, and why one is shaped away.
export const SCREENS: { k: ThinHomeEntry; name: string; icon: IconName; on: (L: V3League) => boolean;
  now: string; off: string }[] = [
  { k: 'hq', name: 'Franchise HQ', icon: 'shield', on: L => L.team || L.commish, now: 'Lineup and roster',
    off: 'you have no team in this league' },
  { k: 'war', name: 'War Room', icon: 'swords', on: L => L.team || L.commish, now: 'Trade desk and pool',
    off: 'you have no team in this league' },
  { k: 'pulse', name: 'League Pulse', icon: 'pulse', on: () => true, now: 'Scores and the race', off: '' },
  { k: 'control', name: 'Control Room', icon: 'gear', on: L => L.commish, now: 'Jobs waiting',
    off: 'only the commissioner has a Control Room' },
];

export function useHome(leagues: V3League[], port: HostedPort) {
  const [briefs, setBriefs] = useState<Record<string, Brief>>({});
  const [needs, setNeeds] = useState<NeedsYouItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(t);
  }, []);
  useEffect(() => {
    let active = true;
    setLoading(true);
    const keep = (id: string, value: Brief) => { if (active) setBriefs(cur => ({ ...cur, [id]: value })); };
    void gatherNeedsYou(leagues, port, {
      now: Date.now(),
      readPeople: async id => {
        const L = leagues.find(l => l.id === id);
        if (!L) throw new Error('Admitted league is unavailable');
        return { league: id, yourFranchise: L.franchise ?? '', yourRole: L.role ?? '' };
      },
      readBrief: async (id, mine, hosted, _owner, commish) => {
        try {
          const [brief, waivers, news] = await Promise.all([
            readThinHomeBrief(id, mine, hosted, undefined, commish),
            mine ? readWaivers(id, mine, hosted) : Promise.resolve(undefined),
            readLeagueNews(id, hosted)]);
          keep(id, { brief, waivers, news });
          return brief;
        } catch (cause) {
          keep(id, { error: cause instanceof Error ? cause.message : String(cause) });
          throw cause;
        }
      },
    }).then(items => { if (active) setNeeds(items); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [leagues, port]);
  return { briefs, needs, loading, now };
}

// Your own waiver claims: the Waiting on and Landed lanes. A failed read leaves those rows out.
async function readWaivers(league: string, franchise: string, port: HostedPort) {
  try {
    const r = await port({ command: 'waivers.read', mode: 'read',
      arguments: { league, franchise, at: new Date().toISOString() } });
    const w = parseHostedWaivers(r.data);
    return w.league === league && w.franchise === franchise ? w : undefined;
  } catch { return undefined; }
}

async function readLeagueNews(league: string, port: HostedPort): Promise<NewsFeed | undefined> {
  try {
    return await readNewsFeed(league, port);
  } catch { return undefined; }
}

const HOME_PROV: Provenance = {
  source: 'League ledger', kind: 'live',
  freshness: { state: 'live', fetchedAt: '', note: '' },
};

/** Build a NativeHQ view from a Home brief so clicking a player on Home opens the shared card drawer. */
export function homeCardReading(brief?: HomeBrief): NativeHQ {
  const season = brief?.pulse?.season ?? new Date().getFullYear();
  const emptySnap: Snapshot = {
    league: { value: { season, franchiseCount: 0 }, provenance: HOME_PROV },
    franchises: { value: [], provenance: HOME_PROV },
    rosters: { value: [], provenance: HOME_PROV },
    players: { value: [], provenance: HOME_PROV },
    eligibility: {
      players: {}, feeds: { injuries: HOME_PROV, players: HOME_PROV, rosters: HOME_PROV },
    },
  };
  return {
    revision: 0,
    errors: [],
    week: brief?.pulse?.week,
    searchPlayers: [],
    settings: { positions: [] },
    snapshot: brief?.assets ?? emptySnap,
    thisWeek: brief?.pulse?.results,
  } as unknown as NativeHQ;
}

function entryFor(item: NeedsYouItem): ThinHomeEntry | undefined {
  const s = item.screen;
  if (s.node === 'home') return undefined;
  if (s.node === 'hq') return s.workspace === 'lineup-and-roster' ? 'lineup' : 'hq';
  if (s.node === 'war') return s.workspace === 'trade-desk' ? 'trades'
    : s.workspace === 'valuations-and-pool' ? 'bids' : 'war';
  if (s.node === 'pulse') return 'pulse';
  return 'control';
}

const CT = 'America/Chicago';
export const ctTime = (iso: string) =>
  new Date(iso).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', timeZone: CT });
export const ctStamp = (ms: number) => new Date(ms).toLocaleString('en-US', { weekday: 'short',
  hour: 'numeric', minute: '2-digit', timeZone: CT }).toUpperCase().replace(',', '');

const clockOf = (iso: string | undefined, now: number) => {
  if (!iso) return 'no clock';
  return Date.parse(iso) - now < 86_400_000 ? `${ctTime(iso)} · ${left(iso, now)}` : left(iso, now);
};
const BUTTON: Partial<Record<ThinHomeEntry, string>> = { lineup: 'Open lineup', trades: 'Open offer',
  bids: 'Open pool', control: 'Open job', pulse: 'Open Pulse', hq: 'Open HQ', war: 'Open War Room' };

export function docketOf(needs: NeedsYouItem[], leagues: V3League[], now: number): DocketItem[] {
  return needs.filter(n => n.kind !== 'unavailable').map((n, i) => {
    const L = leagues.find(l => l.id === n.leagueId), entry = entryFor(n);
    const soon = n.deadline ? Date.parse(n.deadline) - now < 3_600_000 : false;
    return { id: n.id, lg: n.leagueId, lane: 'need', entry,
      t: n.title && n.detail ? `${n.title}: ${n.detail}` : n.words,
      who: L?.commish && entry === 'control' ? 'you (commissioner)' : 'you',
      clock: clockOf(n.deadline, now), btn: (entry && BUTTON[entry]) ?? 'Open',
      urgent: soon || (i === 0 && Boolean(n.deadline)) };
  });
}

// Waiting on and Landed: the moves you started that others now hold, and what has finished.
// Landed keeps the last week. Rows come only from trades.read, bids.read and waivers.read.
const LANDED_MS = 7 * 86_400_000;
const DAY = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', timeZone: CT });
export function ago(iso: string, now: number) {
  const ms = now - Date.parse(iso);
  if (ms < 3_600_000) return `${Math.max(1, Math.round(ms / 60_000))}m ago`;
  if (ms < 86_400_000) return `${Math.round(ms / 3_600_000)}h ago`;
  return ms < 2 * 86_400_000 ? 'yesterday' : `${Math.floor(ms / 86_400_000)} days ago`;
}
const TRADE_BAD = ['declined', 'expired', 'failed', 'vetoed', 'withdrawn'];

export function lanesOf(leagues: V3League[], briefs: Record<string, Brief>, now: number): DocketItem[] {
  const out: DocketItem[] = [];
  const recent = (iso?: string) => {
    const at = iso ? Date.parse(iso) : NaN;
    return at <= now && now - at < LANDED_MS;
  };
  for (const L of leagues) {
    const mine = L.franchise, b = briefs[L.id];
    if (!mine || !b) continue;
    const add = (lane: Lane, id: string, t: string, who: string, clock: string, btn: string,
      entry: ThinHomeEntry, bad?: boolean) => out.push({ id: `${L.id}:${lane}:${id}`, lg: L.id, lane, t, who,
      clock, btn, entry, bad });
    for (const o of b.brief?.trades?.offers ?? []) {
      if (!o.parties.includes(mine)) continue;
      const names = new Map(o.teams);
      const other = o.parties.filter(p => p !== mine).map(p => names.get(p) ?? p).join(' + ');
      if (isEndedTrade(o.status)) {
        if (!recent(o.offeredAt)) continue;
        const day = `offered ${DAY.format(new Date(o.offeredAt))}`;
        add('land', o.id, `Trade with ${other}: ${o.status}`, 'done', day, 'View', 'trades',
          TRADE_BAD.includes(o.status));
      } else if (!(o.status === 'offered' && !o.accepted.includes(mine))) {
        add('wait', o.id, o.proposer === mine ? `Your offer to ${other}` : `Trade with ${other}`, o.words,
          o.expiresAt ? left(o.expiresAt, now) : 'no clock', 'View', 'trades');
      }
    }
    for (const a of b.brief?.bids?.results ?? []) {
      if (!recent(a.decided)) continue;
      const won = `Won ${a.name}${a.contract ? ` for ${a.contract}` : ''}`;
      if (a.winner === mine) add('land', a.player, won, 'done', ago(a.decided, now), 'See pool', 'bids');
      else if (a.yours) {
        add('land', a.player, `${a.name}: ${a.yours}`, 'done', ago(a.decided, now), 'See pool', 'bids', true);
      }
    }
    const players = new Map((b.waivers?.players ?? []).map(p => [p.player, p]));
    for (const c of b.waivers?.claims ?? []) {
      const p = players.get(c.player), name = p?.name ?? c.player;
      if (c.words === 'Claim recorded privately') {
        add('wait', c.id, `Waiver claim: ${name}`, p ? `processing · ${p.words}` : 'processing',
          p?.closesAt ? left(p.closesAt, now) : 'no clock', 'View', 'bids');
      } else if (!p?.closesAt || recent(p.closesAt)) {
        const when = p?.closesAt ? ago(p.closesAt, now) : '';
        add('land', c.id, `Waiver claim: ${name}. ${c.words}`, 'done', when, 'See pool', 'bids',
          c.words !== 'Awarded to your franchise');
      }
    }
  }
  return out;
}
