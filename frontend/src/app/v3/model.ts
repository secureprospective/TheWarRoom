// Desktop v3 data: the admitted leagues dressed for the frame, and the Home reads behind the pieces.
// The reads are the thin Home's (gatherNeedsYou, readThinHomeBrief); only their presentation is v3.
import { useEffect, useState } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { gatherNeedsYou, type NeedsYouItem } from '../shell/homeNeedsYou';
import { readThinHomeBrief, type HomeBrief } from '../shell/homeReads';
import { left } from '../shell/homeTiles';
import { leagueInitials, type LeagueChoice } from '../shell/leaguePicker';
import type { ThinHomeEntry } from '../shell/ThinHome';
import type { IconName } from './icons';

// League identity colours, in the mock's order (IDC).
const IDC = ['hsl(172,30%,52%)', 'hsl(210,28%,66%)', 'hsl(14,48%,58%)', 'hsl(75,26%,54%)',
  'hsl(30,62%,56%)', 'hsl(195,32%,60%)', 'hsl(40,40%,66%)', 'hsl(36,22%,72%)'];

export type V3League = LeagueChoice & { bd: string; c: string; commish: boolean; team: boolean };
export type Brief = { brief?: HomeBrief; error?: string };
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
  { k: 'hq', name: 'Franchise HQ', icon: 'shield', on: L => L.team, now: 'Lineup and roster',
    off: 'you have no team in this league' },
  { k: 'war', name: 'War Room', icon: 'swords', on: L => L.team, now: 'Trade desk and pool',
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
          const brief = await readThinHomeBrief(id, mine, hosted, undefined, commish);
          keep(id, { brief });
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
