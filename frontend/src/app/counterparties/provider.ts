import { TargetCounterparties, TargetCounterpartTeam } from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { freshness } from '../data/parse';
import { object, text, integer, choice, array } from '../data/parseValues';
import type {
  CounterpartiesReading,
  CounterScope,
  TeamCounterpart,
  TeamEdge,
  TeamPanelReading,
  TeamShare,
  TierShift,
} from './contract';

function list<T>(value: unknown, path: string, parse: (v: unknown, p: string) => T): T[] {
  return value === null ? [] : array(value, path, parse);
}
const toneOf = (value: unknown, path: string) => choice(value, path, ['', 'up', 'down', 'even']);
function finite(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value))
    throw new Error(`${path}: expected finite number`);
  return value;
}
function flag(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path}: expected boolean`);
  return value;
}
function years(value: unknown, path: string): [number, number] {
  const xs = array(value, path, (v) => integer(v, path));
  if (xs.length !== 2) throw new Error(`${path}: expected two years`);
  return [xs[0], xs[1]];
}
const numberOrNull = (value: unknown, path: string) =>
  value === null ? null : finite(value, path);

function edge(value: unknown, path: string): TeamEdge {
  const r = object(value, path, ['n', 'mean', 'lo', 'hi', 'tone', 'total']);
  const out: TeamEdge = {
    n: integer(r.n, path),
    mean: numberOrNull(r.mean, path),
    lo: numberOrNull(r.lo, path),
    hi: numberOrNull(r.hi, path),
    tone: toneOf(r.tone, path),
  };
  if (r.total !== undefined) out.total = finite(r.total, `${path}.total`);
  return out;
}
function share(value: unknown, path: string): TeamShare {
  const r = object(value, path, ['k', 'n', 'p', 'lo', 'hi', 'base', 'tone', 'countText']);
  const out: TeamShare = {
    k: integer(r.k, path),
    n: integer(r.n, path),
    p: numberOrNull(r.p, path),
  };
  if (r.countText !== undefined) out.countText = text(r.countText, path);
  if (r.lo !== undefined) out.lo = finite(r.lo, `${path}.lo`);
  if (r.hi !== undefined) out.hi = finite(r.hi, `${path}.hi`);
  if (r.base != null) out.base = finite(r.base, `${path}.base`);
  if (r.tone !== undefined) out.tone = toneOf(r.tone, `${path}.tone`);
  return out;
}
function words(value: unknown, path: string) {
  const w = object(value, path, [
    'team', 'sides', 'perSeason', 'now', 'later', 'picks', 'age', 'partners',
    'partner', 'day', 'since', 'legend', 'up', 'down',
  ]);
  return {
    team: text(w.team, path),
    sides: text(w.sides, path),
    perSeason: text(w.perSeason, path),
    now: text(w.now, path),
    later: text(w.later, path),
    picks: text(w.picks, path),
    age: text(w.age, path),
    partners: text(w.partners, path),
    partner: text(w.partner, path),
    day: text(w.day, path),
    since: text(w.since, path),
    legend: text(w.legend, path),
    up: text(w.up, path),
    down: text(w.down, path),
  };
}
function shell(r: Record<string, unknown>, path: string) {
  const status = choice(r.status, path, ['ok', 'thin', 'failed', 'not_wired']);
  const parsed = {
    status,
    absenceReason: r.absenceReason == null ? null : text(r.absenceReason, path),
    count: integer(r.count, path),
    threshold: integer(r.threshold, path),
    baseline: text(r.baseline, path),
    cutoff: text(r.cutoff, path),
    method: text(r.method, path),
    inputSHA: text(r.inputSHA, path),
    buildStamp: text(r.buildStamp, path),
    howCounted: text(r.howCounted, path),
  };
  if (status !== 'ok' && !parsed.absenceReason)
    throw new Error('Counterparties: absence reason required');
  return parsed;
}
function provenance(value: unknown, path: string) {
  const source = object(value, path, ['source', 'kind', 'freshness']);
  return {
    source: text(source.source, path),
    kind: choice(source.kind, path, ['archive']),
    freshness: freshness(source.freshness, path),
  };
}
export function parseCounterparties(value: unknown): CounterpartiesReading {
  const path = 'Counterparties';
  const r = object(value, path, [
    'value',
    'provenance',
    'status',
    'absenceReason',
    'count',
    'threshold',
    'baseline',
    'cutoff',
    'method',
    'inputSHA',
    'buildStamp',
    'howCounted',
  ]);
  const parsed: CounterpartiesReading = shell(r, path);
  if (parsed.status === 'ok' || parsed.status === 'thin') {
    parsed.provenance = provenance(r.provenance, path);
    const body = object(r.value, path, [
      'words',
      'countText',
      'summary',
      'asOf',
      'headline',
      'note',
      'picks',
      'teams',
      'cards',
      'trendSay',
      'trends',
      'pairs',
      'rows',
    ]);
    parsed.value = {
      words: words(body.words, path),
      countText: text(body.countText, path),
      summary: text(body.summary, path),
      asOf: text(body.asOf, path),
      headline: text(body.headline, path),
      note: text(body.note, path),
      picks: array(body.picks, path, (pick) => {
        const p = object(pick, `${path}.picks`, ['id', 'label', 'hint', 'count', 'with', 'tradingAs']);
        return {
          with: text(p.with, path),
          tradingAs: text(p.tradingAs, path),
          id: text(p.id, path),
          label: text(p.label, path),
          hint: text(p.hint, path),
          count: finite(p.count, path),
        };
      }),
      teams: array(body.teams, path, (team) => {
        const t = object(team, `${path}.teams`, ['fid', 'name']);
        return { fid: text(t.fid, path), name: text(t.name, path) };
      }),
      cards: list(body.cards, path, (card) => {
        const c = object(card, `${path}.cards`, [
          'id',
          'label',
          'hint',
          'thin',
          'perSeason',
          'perSeasonText',
          'posture',
          'rows',
        ]);
        return {
          id: text(c.id, path),
          label: text(c.label, path),
          hint: text(c.hint, path),
          thin: flag(c.thin, path),
          perSeason: finite(c.perSeason, path),
          perSeasonText: text(c.perSeasonText, path),
          posture: list(c.posture, path, (cell) => {
            const p = object(cell, `${path}.posture`, ['id', 'label', 'p', 'tone', 'title']);
            return {
              id: text(p.id, path),
              label: text(p.label, path),
              p: numberOrNull(p.p, path),
              tone: toneOf(p.tone, path),
              title: text(p.title, path),
            };
          }),
          rows: list(c.rows, path, (row) => {
            const x = object(row, `${path}.rows`, ['label', 'text', 'tone', 'range']);
            return {
              label: text(x.label, path),
              text: text(x.text, path),
              tone: toneOf(x.tone, path),
              range: x.range === undefined ? undefined : edge(x.range, path),
            };
          }),
        };
      }),
      trendSay: list(body.trendSay, path, (say) => text(say, path)),
      trends: body.trends == null ? { seasons: [], tiers: [] } : parseTrends(body.trends, path),
      pairs: list(body.pairs, path, (cell) => {
        const p = object(cell, `${path}.pairs`, ['other', 'n', 'now', 'real', 'trades', 'title']);
        return {
          other: text(p.other, path),
          n: integer(p.n, path),
          now: edge(p.now, `${path}.pairs.now`),
          real: edge(p.real, `${path}.pairs.real`),
          trades: text(p.trades, path),
          title: text(p.title, path),
        };
      }),
      rows: list(body.rows, path, (row) => {
        const x = object(row, `${path}.rows`, [
          'fid',
          'name',
          'n',
          'perSeason',
          'tiers',
          'forNow',
          'forLater',
          'day',
          'since',
          'picks',
          'age',
          'partner',
          'partnerName',
          'partnersCount',
          'nText',
          'partnersText',
          'standingTitle',
          'partnerTrades',
          'dayText',
          'sinceText',
        ]);
        const tiers = object(x.tiers, `${path}.tiers`, ['contender', 'hunt', 'rebuild']);
        return {
          dayText: text(x.dayText, path),
          sinceText: text(x.sinceText, path),
          fid: text(x.fid, path),
          name: text(x.name, path),
          n: integer(x.n, path),
          perSeason: finite(x.perSeason, path),
          tiers: {
            contender: finite(tiers.contender, path),
            hunt: finite(tiers.hunt, path),
            rebuild: finite(tiers.rebuild, path),
          },
          forNow: share(x.forNow, `${path}.forNow`),
          forLater: share(x.forLater, `${path}.forLater`),
          day: edge(x.day, `${path}.day`),
          since: edge(x.since, `${path}.since`),
          picks: edge(x.picks, `${path}.picks`),
          age: edge(x.age, `${path}.age`),
          partner:
            x.partner == null
              ? null
              : (() => {
                  const p = object(x.partner, `${path}.partner`, ['fid', 'n']);
                  return { fid: text(p.fid, path), n: integer(p.n, path) };
                })(),
          partnerName: text(x.partnerName, path),
          partnersCount: integer(x.partnersCount, path),
          nText: text(x.nText, path),
          partnersText: text(x.partnersText, path),
          standingTitle: text(x.standingTitle, path),
          partnerTrades: text(x.partnerTrades, path),
        };
      }),
    };
    if (!parsed.cutoff || !parsed.howCounted || !/^[a-f0-9]{64}$/.test(parsed.inputSHA)) {
      throw new Error('Counterparties: archive disclosure missing');
    }
  }
  return parsed;
}
function parseTrends(value: unknown, path: string) {
  const t = object(value, `${path}.trends`, ['seasons', 'tiers']);
  return {
    seasons: list(t.seasons, path, (y) => integer(y, path)),
    tiers: list(t.tiers, path, (tier) => {
      const x = object(tier, `${path}.trends.tiers`, ['tier', 'rows', 'shift']);
      const shift = object(x.shift, `${path}.shift`, ['now', 'later']);
      const parseShift = (s: unknown): TierShift =>
        s == null
          ? null
          : (() => {
              const o = object(s, `${path}.shift`, ['early', 'late', 'from', 'to', 'clear']);
              return {
                early: finite(o.early, path),
                late: finite(o.late, path),
                from: years(o.from, path),
                to: years(o.to, path),
                clear: flag(o.clear, path),
              };
            })();
      return {
        tier: text(x.tier, path),
        rows: array(x.rows, path, (row) => {
          const r = object(row, `${path}.rows`, [
            'season',
            'n',
            'nText',
            'now',
            'later',
            'net',
            'showShares',
          ]);
          return {
            nText: text(r.nText, path),
            showShares: flag(r.showShares, path),
            season: integer(r.season, path),
            n: integer(r.n, path),
            now: share(r.now, `${path}.now`),
            later: share(r.later, `${path}.later`),
            net: numberOrNull(r.net, path),
          };
        }),
        shift: { now: parseShift(shift.now), later: parseShift(shift.later) },
      };
    }),
  };
}
export function parseTeamPanel(value: unknown): TeamPanelReading {
  const path = 'CounterpartTeam';
  const r = object(value, path, [
    'value',
    'provenance',
    'status',
    'absenceReason',
    'count',
    'threshold',
    'baseline',
    'cutoff',
    'method',
    'inputSHA',
    'buildStamp',
    'howCounted',
  ]);
  const parsed: TeamPanelReading = shell(r, path);
  if (parsed.status === 'ok' || parsed.status === 'thin') {
    parsed.provenance = provenance(r.provenance, path);
    const body = object(r.value, path, [
      'words',
      'countText',
      'flowScale',
      'fid',
      'name',
      'asOf',
      'sides',
      'empty',
      'note',
      'seasons',
      'seasonSay',
      'posture',
      'kinds',
      'kindSay',
      'partners',
      'partnersSay',
      'partnersText',
      'behaviour',
      'asTier',
      'day',
      'since',
      'dayText',
      'sinceText',
    ]);
    const behaviour = object(body.behaviour, `${path}.behaviour`, [
      'n',
      'perSeason',
      'posture',
      'partners',
      'windows',
      'now',
      'real',
      'picks',
      'age',
    ]);
    const postureOf = (m: unknown) => {
      const p = object(m, `${path}.posture`, ['now', 'later', 'swap']);
      return { now: share(p.now, path), later: share(p.later, path), swap: share(p.swap, path) };
    };
    const windowsOf = (m: unknown) => {
      const p = object(m, `${path}.windows`, ['spring', 'summer', 'season']);
      return {
        spring: share(p.spring, path),
        summer: share(p.summer, path),
        season: share(p.season, path),
      };
    };
    const partners = object(behaviour.partners, `${path}.partners`, [
      'contender',
      'hunt',
      'rebuild',
    ]);
    for (const id of ['contender', 'hunt', 'rebuild'])
      share(partners[id], `${path}.partners.${id}`);
    const value: TeamCounterpart = {
      words: words(body.words, path),
      countText: text(body.countText, path),
      flowScale: finite(body.flowScale, path),
      fid: text(body.fid, path),
      name: text(body.name, path),
      asOf: text(body.asOf, path),
      sides: integer(body.sides, path),
      empty: flag(body.empty, path),
      note: text(body.note, path),
      seasons: array(body.seasons, path, (cell) => {
        const c = object(cell, `${path}.seasons`, [
          'season',
          'n',
          'now',
          'later',
          'swap',
          'against',
          'flagged',
          'nowText',
          'laterText',
          'swapText',
          'againstText',
          'mostlyText',
          'net',
          'finish',
          'label',
          'title',
        ]);
        return {
          season: integer(c.season, path),
          n: integer(c.n, path),
          now: integer(c.now, path),
          later: integer(c.later, path),
          swap: integer(c.swap, path),
          against: integer(c.against, path),
          flagged: flag(c.flagged, path),
          nowText: text(c.nowText, path),
          laterText: text(c.laterText, path),
          swapText: text(c.swapText, path),
          againstText: text(c.againstText, path),
          mostlyText: text(c.mostlyText, path),
          net: numberOrNull(c.net, path),
          finish: c.finish == null ? null : integer(c.finish, path),
          label: text(c.label, path),
          title: text(c.title, path),
        };
      }),
      seasonSay: text(body.seasonSay, path),
      posture: array(body.posture, path, (cell) => {
        const p = object(cell, `${path}.posture`, ['id', 'label', 'p', 'tone', 'title']);
        return {
          id: text(p.id, path),
          label: text(p.label, path),
          p: numberOrNull(p.p, path),
          tone: toneOf(p.tone, path),
          title: text(p.title, path),
        };
      }),
      kinds: array(body.kinds, path, (kind) => {
        const k = object(kind, `${path}.kinds`, [
          'kind',
          'name',
          'inN',
          'outN',
          'inW',
          'outW',
          'text',
        ]);
        return {
          kind: text(k.kind, path),
          name: text(k.name, path),
          inN: integer(k.inN, path),
          outN: integer(k.outN, path),
          inW: finite(k.inW, path),
          outW: finite(k.outW, path),
          text: text(k.text, path),
        };
      }),
      kindSay: text(body.kindSay, path),
      partners: array(body.partners, path, (partner) => {
        const p = object(partner, `${path}.partners`, [
          'fid',
          'name',
          'n',
          'net',
          'tradesText',
          'netText',
        ]);
        return {
          fid: text(p.fid, path),
          name: text(p.name, path),
          n: integer(p.n, path),
          net: edge(p.net, `${path}.net`),
          tradesText: text(p.tradesText, path),
          netText: text(p.netText, path),
        };
      }),
      partnersSay: text(body.partnersSay, path),
      partnersText: text(body.partnersText, path),
      behaviour: {
        n: integer(behaviour.n, path),
        perSeason: finite(behaviour.perSeason, path),
        posture: postureOf(behaviour.posture),
        windows: windowsOf(behaviour.windows),
        now: edge(behaviour.now, path),
        real: edge(behaviour.real, path),
        picks: edge(behaviour.picks, path),
        age: edge(behaviour.age, path),
      },
      asTier: array(body.asTier, path, (tier) => {
        const t = object(tier, `${path}.asTier`, ['tier', 'n', 'edge', 'label', 'text']);
        return {
          tier: text(t.tier, path),
          n: integer(t.n, path),
          edge: edge(t.edge, path),
          label: text(t.label, path),
          text: text(t.text, path),
        };
      }),
      day: edge(body.day, path),
      since: edge(body.since, path),
      dayText: text(body.dayText, path),
      sinceText: text(body.sinceText, path),
    };
    parsed.value = value;
    if (!parsed.cutoff || !parsed.howCounted || !/^[a-f0-9]{64}$/.test(parsed.inputSHA)) {
      throw new Error('CounterpartTeam: archive disclosure missing');
    }
  }
  return parsed;
}
function unavailable(reason: string, status: 'failed' | 'not_wired'): CounterpartiesReading {
  return {
    status,
    absenceReason: reason,
    count: 0,
    threshold: 40,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
function teamUnavailable(reason: string, status: 'failed' | 'not_wired'): TeamPanelReading {
  return {
    status,
    absenceReason: reason,
    count: 0,
    threshold: 1,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
async function read<T>(
  call: () => Promise<T>,
  fail: (reason: string, status: 'failed' | 'not_wired') => T,
): Promise<T> {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go) {
    return fail(
      'Counterparties needs the desktop app; no browser fixtures are substituted',
      'not_wired',
    );
  }
  try {
    return await call();
  } catch (cause) {
    return fail(
      `Counterparties read failed: ${String(cause)}. Retry the held archive reading.`,
      'failed',
    );
  }
}
export const counterpartProvider = {
  surface(scope: CounterScope) {
    return read(async () => parseCounterparties(await TargetCounterparties(scope)), unavailable);
  },
  team(scope: CounterScope, fid: string) {
    return read(
      async () =>
        parseTeamPanel(await TargetCounterpartTeam(new main.RawCounterpartTeam({ scope, fid }))),
      teamUnavailable,
    );
  },
};
