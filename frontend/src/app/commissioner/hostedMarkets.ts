import { array, id, integer, object, text, timestamp } from '../data/parseValues';

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
        closesAt: p.closesAt == null ? undefined : timestamp(p.closesAt, 'Bid close'),
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
    /** The same awards as facts: who won, when it was decided, and the viewer's own result. */
    results: array(r.awards, 'Awards', (v) => {
      const a = object(v, 'Award', ['player', 'name', 'winner', 'team', 'contract', 'points', 'decided',
        'words', 'yours']);
      return { player: text(a.player, 'Player'), name: text(a.name, 'Name'),
        winner: a.winner ? text(a.winner, 'Winner') : '',
        contract: a.contract ? text(a.contract, 'Contract') : '',
        decided: timestamp(a.decided, 'Decided'), yours: a.yours ? text(a.yours, 'Your result') : '' };
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
    order: array(r.order, 'Waiver order', text),
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
        status: text(p.status, 'Status'),
        closesAt: p.closesAt == null ? undefined : timestamp(p.closesAt, 'Bid close'),
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
        'holds',
        'recusals',
        'appeals',
        'overrides',
        'expires',
      ]);
      const assets = array(t.assets, 'Trade assets', (v) => {
        const a = object(v, 'Asset', ['from', 'to', 'player', 'season', 'round', 'original']);
        const from = text(a.from, 'From');
        const to = text(a.to, 'To');
        const player = a.player === undefined ? '' : id(a.player, 'Player', true);
        if (player) {
          if (a.season !== undefined || a.round !== undefined || a.original !== undefined)
            throw new Error('Trade asset mixes a player and a pick');
          return { from, to, token: player };
        }
        return { from, to, token: `pick:${integer(a.season, 'Pick season', 1)}:${
          integer(a.round, 'Pick round', 1)}:${text(a.original, 'Pick origin')}` };
      });
      const details = array(t.assetDetails, 'Offered assets', (v) => {
        const a = object(v, 'Asset', [
          'fromTeam',
          'toTeam',
          'label',
          'contract',
          'yearlySalaryCents',
          'contractEndsInSeason',
        ]);
        const contract = a.contract ? ` · ${text(a.contract, 'Contract')}` : '';
        return { from: text(a.fromTeam, 'From'), to: text(a.toTeam, 'To'), label: text(a.label, 'Asset'),
          words: `${text(a.label, 'Asset')}: ${text(a.fromTeam, 'From')} → ${
            text(a.toTeam, 'To')}${contract}` };
      });
      if (assets.length !== details.length) throw new Error('Trade assets and descriptions do not match');
      return {
        id: text(t.id, 'Trade'),
        status: text(t.status, 'Trade status'),
        parties: strings(t.parties),
        proposer: text(t.proposer, 'Proposer'),
        accepted: strings(t.accepted),
        offeredAt: text(t.offeredAt, 'Offered'),
        expiresAt: text(t.expiresAt, 'Expires'),
        assets,
        teams: assets.flatMap((asset, i) => [[asset.from, details[i].from],
          [asset.to, details[i].to]] as [string, string][]),
        words: text(t.words, 'Trade status'),
        expires: text(t.expires, 'Expiry'),
        details: details.map(asset => asset.words),
        /** legs pairs each asset's franchise ids with its plain label, for give/get columns. */
        legs: assets.map((asset, i) => ({ from: asset.from, to: asset.to, label: details[i].label })),
        reasons: [
          ...['votes', 'recusals', 'appeals', 'overrides'].flatMap((key) =>
            array(t[key] ?? [], key, (v) => {
              const a = object(v, 'Decision', ['who', 'decision', 'at', 'reason']);
              return `${text(a.decision, 'Decision')}${a.reason ? `: ${text(a.reason, 'Reason')}` : ''}`;
            }),
          ),
          ...array(t.holds ?? [], 'Review holds', (v) => {
            const hold = object(v, 'Review hold', ['who', 'at', 'reason']);
            return `Review held for 24 hours: ${text(hold.reason, 'Reason')}`;
          }),
        ],
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
    'evidence',
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
        'homeProvisional',
        'awayProvisional',
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
    'evidence',
  ]);
  const rows = (v: unknown) =>
    array(v, 'Standings rows', (v) => {
      const s = object(v, 'Standing', [
        'rank',
        'franchise',
        'name',
        'division',
        'conference',
        'wins',
        'losses',
        'ties',
        'record',
        'streak',
        'computedRecord',
        'pointsFor',
        'pointsWords',
        'why',
      ]);
      return {
        rank: integer(s.rank, 'Place'),
        franchise: text(s.franchise, 'Franchise'),
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
