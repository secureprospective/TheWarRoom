export type FieldKind = 'text' | 'number' | 'money' | 'decimal' | 'instant' | 'boolean'
  | 'franchise' | 'player' | 'bidPlayer' | 'pick' | 'entry' | 'window' | 'claim' | 'bid' | 'trade' | 'item'
  | 'order' | 'starters' | 'choice' | 'scope' | 'franchiseOrAll';
export type Field = {
  path: string; label: string; kind: FieldKind; choices?: readonly string[]; optional?: boolean;
};
export type ActDefinition = {
  control: string; command: string; label: string; root: string; fields: Field[]; private?: boolean;
};
const f = (path: string, label: string, kind: FieldKind = 'text', choices?: readonly string[]): Field =>
  ({ path, label, kind, choices });
const franchise = f('franchise', 'Franchise', 'franchise');
const player = f('player', 'Player', 'player');
const season = f('season', 'Season', 'number');
const week = f('week', 'Week', 'number');
const order = f('order', 'Franchises, first to last', 'order');
const principal = f('principal', 'Sign-in');
const id = f('id', 'Invitation or vote name');
const running = f('running', 'The draft clock is already running', 'boolean');
const until = f('until', 'Expires at (your local time)', 'instant');
const optionalUntil = { ...until, label: 'Expires at (optional, your local time)', optional: true };
const choice = (path: string, label: string, values: readonly string[]) => f(path, label, 'choice', values);
const decision = choice('decision', 'Decision', ['approve', 'veto']);
const def = (control: string, command: string, label: string, root: string, fields: Field[],
  privateReading = false): ActDefinition =>
  ({ control, command, label, root, fields, private: privateReading });

// Paths mirror the hand-written Raw* boundary types; no server-computed outcome is sent by a form.
export const actTable: readonly ActDefinition[] = [
  def('CM-018', 'commish.franchise.edit', 'Rename a franchise', 'people',
    [franchise, f('name', 'Franchise name'), f('abbreviation', 'Abbreviation')]),
  def('CM-019', 'commish.seat.invite', 'Invite an owner', 'people', [id, franchise, principal], true),
  def('CM-019', 'commish.seat.invite.revoke', 'Revoke an invitation', 'people', [id], true),
  def('CM-019', 'commish.seat.invite.redeem', 'Accept an invitation', 'people', [id, principal], true),
  def('CM-019', 'commish.seat.remove', 'Remove an owner', 'people', [franchise]),
  def('CM-019', 'commish.seat.replace', 'Replace an owner', 'people', [franchise, principal]),
  def('CM-020', 'commish.abilities.set', 'Change an owner ability', 'people',
    [f('franchise', 'Franchise', 'franchiseOrAll'), choice('ability', 'Ability',
      ['lineups', 'add/drop', 'drop-only', 'trades', 'pick trades',
      'IR', 'taxi', 'articles', 'board', 'chat', 'polls', 'customize franchise', 'home page']),
    f('on', 'Allow this ability', 'boolean')]),
  def('CM-020', 'commish.visibility.set', 'Change who can read a report', 'people',
    [f('report', 'Report name'), choice('audience', 'Who can read it', ['all', 'commish'])]),
  def('CM-022', 'commish.succession.stepdown', 'Step down as commissioner', 'people',
    [principal, f('successor', 'Successor sign-in')]),
  ...['accept', 'decline', 'activity'].map(action =>
    def('CM-022', `commish.succession.${action}`, {
      accept: 'Accept the commissioner seat', decline: 'Decline the commissioner seat',
      activity: 'Record commissioner activity',
    }[action]!, 'people', [principal])),
  def('CM-022', 'commish.succession.takeover', 'Take over after inactivity', 'people',
    [principal, f('commissioner', 'Current commissioner sign-in')]),
  def('CM-022', 'commish.succession.voteout.open', 'Open a removal vote', 'people',
    [id, principal, f('commissioner', 'Current commissioner sign-in')]),
  def('CM-022', 'commish.succession.election.open', 'Open an election', 'people', [id, principal]),
  def('CM-022', 'commish.succession.inactivity.set', 'Set inactivity periods', 'people',
    [f('inSeasonDays', 'In-season days', 'number'), f('offseasonDays', 'Offseason days', 'number')]),
  def('CM-022', 'commish.succession.periods.set', 'Set voting periods', 'people',
    [f('electionDays', 'Election days', 'number'), f('runoffDays', 'Runoff days', 'number')]),
  def('CM-022', 'commish.vote.cast', 'Cast a ballot', 'people',
    [id, principal, f('choice', 'Your choice')], true),
  def('CM-022', 'commish.vote.close', 'Close a vote', 'people', [id]),
  def('CM-022', 'commish.vote.tiebreak', 'Decide a tied runoff', 'people',
    [id, principal, f('choice', 'Winning choice')]),
  def('CM-026', 'commish.calendar.exception', 'Allow an action outside its window', 'exception',
    [choice('action', 'Action', ['lineup', 'trade', 'ufa_bid', 'rfa_bid', 'draft_pick',
      'cut', 'waiver_claim', 'ir', 'taxi', 'extension', 'restructure', 'buyout', 'tag',
      'rfa_tender', 'resign']),
    franchise, until]),
  def('CM-027', 'season.rollover', 'Move to the next season', 'rollover',
    [f('to', 'Next season', 'number')]),
  def('CM-028', 'correct.roster', 'Correct a roster', 'correction',
    [franchise, player, choice('action', 'Roster change', ['add', 'remove'])]),
  def('CM-028', 'correct.transaction', 'Reverse a recorded transaction', 'correction',
    [f('entry', 'Recorded transaction', 'entry')]),
  def('CM-028', 'correct.provenance', 'Correct how a player was acquired', 'correction',
    [player, choice('via', 'Acquired through', ['held', 'draft', 'waiver', 'trade', 'free_agent',
      'commissioner'])]),
  def('CM-028', 'correct.pick', 'Correct a pick owner', 'correction',
    [f('pick', 'Pick', 'pick'), f('owner', 'New owner', 'franchise')]),
  def('CM-029', 'commish.lineup.correct', 'Correct a lineup', 'lineup',
    [franchise, season, week, f('starters', 'Starting players', 'starters')]),
  def('CM-030', 'commish.player.position', 'Correct a player position', 'score',
    [player, choice('position', 'Scoring position',
      ['QB', 'RB', 'WR', 'TE', 'PK', 'DT', 'DE', 'LB', 'CB', 'S'])]),
  def('CM-030', 'commish.player.position.clear', 'Clear a position correction', 'score', [player]),
  def('CM-031', 'commish.stat.queue.approve', 'Accept a statistics change', 'stat',
    [f('item', 'Statistics change', 'item')]),
  def('CM-031', 'commish.stat.queue.reject', 'Reject a statistics change', 'stat',
    [f('item', 'Statistics change', 'item')]),
  def('CM-031', 'commish.stat.correct', 'Correct a statistic', 'stat',
    [player, season, week, f('stat', 'Statistic name'), f('value', 'Statistic value', 'decimal'),
    f('source', 'Source for this statistic')]),
  ...['player', 'franchise'].map(target =>
    def('CM-032', `commish.score.${target}`, `Adjust a ${target} score`, 'score',
      [target === 'player' ? player : franchise, season, week,
      f('delta', 'Points to add (negative to subtract)', 'decimal')])),
  def('CM-033', 'commish.standings.record.adjust', 'Set a franchise record', 'standings',
    [season, franchise, f('wins', 'Wins', 'number'), f('losses', 'Losses', 'number'),
    f('ties', 'Ties', 'number')]),
  def('CM-033', 'commish.standings.override', 'Set the standings order', 'standings', [season, order]),
  def('CM-033', 'commish.standings.override.clear', 'Clear standings corrections', 'standings', [season]),
  def('CM-034', 'commish.contract.edit', 'Correct a contract', 'override',
    [player, f('years', 'Contract years', 'number'), f('salaryCents', 'Annual salary (dollars)', 'money')]),
  def('CM-034', 'commish.cap.adjust', 'Adjust a franchise cap charge', 'override',
    [franchise, season, f('amountCents', 'Cap charge (dollars, signed)', 'money')]),
  def('CM-034', 'commish.cap.relief', 'Grant player cap relief', 'override',
    [player, f('amountCents', 'Cap relief (dollars)', 'money')]),
  def('CM-035', 'waivers.process', 'Process a waiver window', 'waiver',
    [f('window', 'Waiver window', 'window')]),
  def('CM-035', 'waivers.order.set', 'Set waiver priority', 'waiver', [order]),
  def('CM-035', 'waivers.order.auto', 'Use automatic waiver priority', 'waiver', []),
  def('CM-035', 'waivers.award.override', 'Correct a waiver award', 'waiver',
    [f('claim', 'Waiver claim', 'claim'), franchise]),
  def('CM-036', 'bids.process', 'Process bids for a player', 'bid', [f('player', 'Player', 'bidPlayer')]),
  def('CM-036', 'bids.award.override', 'Correct a bid award', 'bid', [f('bid', 'Bid', 'bid'), franchise]),
  ...['review', 'appeal.rule', 'override', 'recuse', 'appeal'].map(action =>
    def('CM-037', `trade.${action}`, {
      review: 'Review a trade', 'appeal.rule': 'Rule on a trade appeal',
      override: 'Override a trade decision',
      recuse: 'Step aside from a trade review', appeal: 'Appeal a trade decision',
    }[action]!, 'trade', [f('id', 'Trade', 'trade'),
    ...(action === 'appeal' ? [franchise] : action === 'recuse' ? [] : [decision])])),
  def('CM-038', 'commish.reserve.exception', 'Allow a reserve placement', 'override',
    [player, choice('bucket', 'Reserve list', ['ir', 'taxi']), optionalUntil]),
  def('CM-039', 'rule.exception', 'Grant a rule exception', 'correction',
    [choice('rule', 'Rule', ['roster.limits', 'roster.positionBounds', 'cap.enforcement', 'ir.eligible',
      'taxi.maximumYears', 'calendar.lineup', 'calendar.trade', 'calendar.ufa_bid']),
    f('scope', 'Applies to', 'scope'), optionalUntil]),
  def('CM-039', 'commish.correction.revert', 'Undo a recorded change', '', [f('entry', 'Change', 'entry')]),
  def('CM-041', 'draft.order.auto', 'Use the final standings for draft order', 'draft',
    [season, choice('source', 'Order source', ['standings']), running]),
  def('CM-041', 'draft.order.set', 'Set draft order', 'draft',
    [season, f('round', 'Round (zero means all rounds)', 'number'), order, running]),
  def('CM-041', 'draft.order.pattern', 'Set later-round draft order', 'draft',
    [choice('pattern', 'Later rounds', ['linear', 'snake']), running]),
  def('CM-041', 'draft.order.tiebreak', 'Set the draft tie breaker', 'draft',
    [choice('tie', 'Tie breaker', ['League franchise order', 'Reverse league franchise order']), running]),
  def('CM-042', 'draft.start', 'Set the draft start', 'draft',
    [season, f('at', 'Starts at (your local time)', 'instant')]),
];
export const actsFor = (control: string) => actTable.filter(a => a.control === control);
export const liveDraftControls = new Set(['CM-043', 'CM-044', 'CM-045']);
