import { array, integer, object, text } from '../data/parseValues';

const strings = (v: unknown) => array(v, 'Explanations', text);
export function parseHostedBids(v: unknown) {
  const r = object(v, 'Bids', [
    'league',
    'franchise',
    'revision',
    'season',
    'mode',
    'activeBids',
    'activeLimit',
    'players',
    'awards',
    'notes',
  ]);
  const bid = (v: unknown) => {
    if (v === undefined) return undefined;
    const b = object(v, 'Bid', [
      'id',
      'franchise',
      'team',
      'contract',
      'points',
      'placedAt',
      'placed',
    ]);
    return {
      id: text(b.id, 'Bid'),
      words: `${text(b.team, 'Team')}: ${text(b.contract, 'Contract')}`,
    };
  };
  return {
    league: text(r.league, 'League'),
    franchise: text(r.franchise, 'Franchise'),
    revision: integer(r.revision, 'Version'),
    notes: strings(r.notes),
    limit: text(r.activeLimit, 'Bid limit'),
    players: array(r.players, 'Bids', (v) => {
      const p = object(v, 'Bid player', [
        'player',
        'name',
        'mode',
        'modeWords',
        'leader',
        'bids',
        'yourBid',
        'youLead',
        'lateAt',
        'lateWords',
        'lateNow',
        'closesAt',
        'closes',
      ]);
      return {
        player: text(p.player, 'Player'),
        name: text(p.name, 'Name'),
        words: text(p.modeWords, 'Privacy'),
        closes: text(p.closes, 'Closes'),
        own: bid(p.yourBid),
        leader: bid(p.leader),
      };
    }),
    awards: array(r.awards, 'Awards', (v) => {
      const a = object(v, 'Award', [
        'player',
        'name',
        'winner',
        'team',
        'contract',
        'points',
        'decided',
        'words',
        'yours',
      ]);
      const own = a.yours ? ` — ${text(a.yours, 'Your result')}` : '';
      return `${text(a.name, 'Name')}: ${text(a.words, 'Award')}${own}`;
    }),
  };
}
export function parseHostedWaivers(v: unknown) {
  const r = object(v, 'Waivers', [
    'corrections',
    'league',
    'franchise',
    'revision',
    'season',
    'players',
    'claims',
    'order',
    'orderSource',
    'method',
  ]);
  return {
    league: text(r.league, 'League'),
    franchise: text(r.franchise, 'Franchise'),
    revision: integer(r.revision, 'Version'),
    words: text(r.orderSource, 'Priority'),
    method: text(r.method, 'Method'),
    players: array(r.players, 'Waiver players', (v) => {
      const p = object(v, 'Waiver player', [
        'window',
        'player',
        'name',
        'closesAt',
        'closes',
        'status',
        'winner',
      ]);
      return {
        window: text(p.window, 'Window'),
        player: text(p.player, 'Player'),
        name: text(p.name, 'Name'),
        words: `${text(p.status, 'Status')} · ${text(p.closes, 'Closes')}`,
      };
    }),
    claims: array(r.claims, 'Your claims', (v) => {
      const c = object(v, 'Claim', ['id', 'window', 'player', 'status']);
      return {
        id: text(c.id, 'Claim'),
        player: text(c.player, 'Player'),
        words: text(c.status, 'Status'),
      };
    }),
  };
}
export function parseHostedTrades(v: unknown) {
  const r = object(v, 'Trades', ['league', 'viewer', 'revision', 'rules', 'offers', 'notes']);
  return {
    league: text(r.league, 'League'),
    franchise: text(r.viewer, 'Viewer'),
    revision: integer(r.revision, 'Version'),
    notes: strings(r.notes),
    offers: array(r.offers, 'Trades', (v) => {
      const t = object(v, 'Trade', [
        'vetoNumber',
        'appealedVetoNumber',
        'assetDetails',
        'previous',
        'id',
        'proposer',
        'parties',
        'assets',
        'accepted',
        'policy',
        'offeredAt',
        'expiresAt',
        'status',
        'words',
        'votes',
        'recusals',
        'appeals',
        'overrides',
        'expires',
      ]);
      return {
        id: text(t.id, 'Trade'),
        words: text(t.words, 'Trade status'),
        expires: text(t.expires, 'Expiry'),
        details: array(t.assetDetails, 'Offered assets', (v) => {
          const a = object(v, 'Asset', [
            'fromTeam',
            'toTeam',
            'label',
            'contract',
            'yearlySalaryCents',
            'contractEndsInSeason',
          ]);
          const contract = a.contract ? ` · ${text(a.contract, 'Contract')}` : '';
          return (
            `${text(a.label, 'Asset')}: ${text(a.fromTeam, 'From')} → ` +
            `${text(a.toTeam, 'To')}${contract}`
          );
        }),
        reasons: ['votes', 'recusals', 'appeals', 'overrides'].flatMap((key) =>
          array(t[key] ?? [], key, (v) => {
            const a = object(v, 'Decision', ['who', 'decision', 'at', 'reason']);
            return `${text(a.decision, 'Decision')}${a.reason ? `: ${text(a.reason, 'Reason')}` : ''}`;
          }),
        ),
      };
    }),
  };
}
export function parseHostedResults(v: unknown) {
  const r = object(v, 'Results', [
    'league',
    'season',
    'week',
    'revision',
    'published',
    'words',
    'games',
    'bye',
    'byeName',
    'scores',
    'missed',
  ]);
  return {
    league: text(r.league, 'League'),
    revision: integer(r.revision, 'Version'),
    words: text(r.words, 'Results'),
    games: array(r.games, 'Games', (v) => {
      const g = object(v, 'Game', [
        'week',
        'home',
        'away',
        'homeName',
        'awayName',
        'status',
        'homePoints',
        'awayPoints',
        'winner',
        'words',
        'why',
      ]);
      return {
        names: `${text(g.homeName, 'Home')} v ${text(g.awayName, 'Away')}`,
        words: text(g.words, 'Result'),
        why: strings(g.why),
      };
    }),
  };
}
export function parseHostedStandings(v: unknown) {
  const r = object(v, 'Standings', [
    'league',
    'name',
    'season',
    'revision',
    'status',
    'words',
    'through',
    'decided',
    'pending',
    'criteria',
    'noData',
    'computed',
    'override',
    'changes',
  ]);
  const rows = (v: unknown) =>
    array(v, 'Standings rows', (v) => {
      const s = object(v, 'Standing', [
        'rank',
        'franchise',
        'name',
        'division',
        'wins',
        'losses',
        'ties',
        'record',
        'computedRecord',
        'pointsFor',
        'pointsWords',
        'why',
      ]);
      return {
        rank: integer(s.rank, 'Place'),
        name: text(s.name, 'Team'),
        record: text(s.record, 'Record'),
        points: text(s.pointsWords, 'Points'),
        why: strings(s.why),
      };
    });
  const override = r.override
    ? object(r.override, 'Commissioner standings', ['words', 'rows', 'order', 'records'])
    : undefined;
  return {
    league: text(r.league, 'League'),
    revision: integer(r.revision, 'Version'),
    words: text(r.words, 'Standings'),
    criteria: strings(r.criteria),
    missing: strings(r.noData),
    computed: rows(r.computed),
    override: override
      ? { words: text(override.words, 'Ruling'), rows: rows(override.rows) }
      : undefined,
  };
}
export type HostedMarkets = {
  bids: ReturnType<typeof parseHostedBids>;
  waivers: ReturnType<typeof parseHostedWaivers>;
  trades: ReturnType<typeof parseHostedTrades>;
  results: ReturnType<typeof parseHostedResults>;
  standings: ReturnType<typeof parseHostedStandings>;
};

export function parseHostedMarket(v: unknown) {
  const r = object(v, 'Commissioner choices', [
    'league',
    'revision',
    'windows',
    'claims',
    'players',
    'bids',
    'trades',
  ]);
  const choices = (v: unknown) =>
    array(v, 'Choices', (v) => {
      const c = object(v, 'Choice', ['id', 'words']);
      return {
        value: text(c.id, 'Choice'),
        label: text(c.words, 'Description'),
      };
    });
  return {
    league: text(r.league, 'League'),
    revision: integer(r.revision, 'Version'),
    options: {
      window: choices(r.windows),
      claim: choices(r.claims),
      bidPlayer: choices(r.players),
      bid: choices(r.bids),
      trade: choices(r.trades),
    },
  };
}
