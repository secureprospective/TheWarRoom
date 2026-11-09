// The Control Room's guide content, word for word from docs/ui/mocks/control-room-mock-v1.html (locked).
// It is the app's own help, not league data: each step names the command that does it. A step's mark says
// whether that command is built (B), ruled but not built (R), or a gap (G).

export type Step = readonly [words: string, command: string, mark: 'B' | 'R' | 'G'];
export type Situation = readonly [
  category: string,
  story: string,
  steps: readonly Step[],
  yours: string,
  check: string,
];
export const situations: readonly Situation[] = [
  [
    'GAMES',
    'An NFL game is cancelled or postponed',
    [
      ['Hold the stat queue for that game', 'stat.queue.hold game=…', 'G'],
      ['Adjust affected player scores', 'commish.score.player', 'B'],
      ['Or adjust whole franchise scores', 'commish.score.franchise', 'B'],
      ['Decide the result: replay, tie, or set it', 'commish.standings.record.adjust', 'B'],
      ['Override the rule that would apply', 'rule.exception', 'B'],
      ['Tell the league what you decided', 'commish.notice.publish', 'B'],
    ],
    'Which result counts is your call; the app does not guess.',
    'Standings, playoff seeds and the waiver order after the change.',
  ],
  [
    'GAMES',
    'An NFL stat was corrected',
    [
      ['Open the stat queue', 'stats.queue', 'B'],
      ['Approve or reject (shows if a result flips)', 'commish.stat.queue.approve', 'B'],
      ['Post a notice if a result flipped', 'commish.notice.publish', 'B'],
    ],
    'Nothing.',
    "The week's results and standings.",
  ],
  [
    'GAMES',
    'The stat feed is wrong or silent',
    [
      ['Enter the stat by hand with its source', 'commish.stat.correct', 'B'],
      ['Check feed health', 'Server › Feeds', 'B'],
    ],
    'Finding the true number.',
    'Live scores for that game.',
  ],
  [
    'PLAYERS',
    'The feed has a player at the wrong position',
    [
      ['Override the position', 'commish.player.position', 'B'],
      ['Clear it when the feed is fixed', 'commish.player.position.clear', 'B'],
    ],
    'Nothing.',
    'Lineup validity for teams that start him.',
  ],
  [
    'PLAYERS',
    'A player retires or dies',
    [
      ['Contract rules apply automatically (retirement 30%, death 0%)', 'Rules › Cap & contracts', 'B'],
      ['Adjust the cap by hand if needed', 'commish.cap.adjust', 'B'],
    ],
    'Deciding any exception.',
    "That team's cap.",
  ],
  [
    'PLAYERS',
    'Career-ending injury or suspension: cap relief',
    [['Grant cap relief (reasons from the rules)', 'commish.cap.relief', 'B']],
    'Judging the appeal.',
    "That team's cap.",
  ],
  [
    'OWNERS',
    'An owner vanished mid-season',
    [
      ['Seat yourself as caretaker', 'commish.seat.caretaker', 'B'],
      ["Set the team's lineup each week", 'commish.lineup.correct', 'B'],
      ['Invite a new owner', 'commish.seat.invite', 'B'],
      ['Hand the seat over', 'commish.seat.replace', 'B'],
    ],
    'Finding a new owner.',
    'Missed-lineup flags stop once the caretaker sets lineups.',
  ],
  [
    'OWNERS',
    'An owner wants out',
    [
      ['Remove the owner (team, contracts and picks stay)', 'commish.seat.remove', 'B'],
      ['Caretaker until filled', 'commish.seat.caretaker', 'B'],
      ['Invite a replacement', 'commish.seat.invite', 'B'],
    ],
    'Finding the replacement.',
    'Pending trades and bids from that team.',
  ],
  [
    'OWNERS',
    "A new owner can't redeem the invitation",
    [
      ['Revoke the old invite', 'commish.seat.invite.revoke', 'B'],
      ['Send a new one', 'commish.seat.invite', 'B'],
      ['Help with sign-in', 'Server › Sign-in help', 'B'],
    ],
    'Talking to the owner.',
    'Seat shows filled.',
  ],
  [
    'OWNERS',
    'An owner lost their authenticator',
    [
      ['Reset their authenticator (ruled 10-30)', 'account.totp.reset', 'R'],
      ['They sign in and can set it up again', '—', 'R'],
    ],
    'Confirming it is really them.',
    'Ledger shows the reset.',
  ],
  [
    'TRADES',
    'A trade looks like collusion',
    [
      ['Let the DOT vote', 'trade.review', 'B'],
      ['Overturn the DOT (typed reason, visible)', 'trade.override', 'B'],
      ['Party may appeal once', 'trade.appeal', 'B'],
    ],
    'The judgement.',
    'Both rosters and caps.',
  ],
  [
    'TRADES',
    'The commissioner is a party to a trade',
    [
      ['Step aside from the review', 'trade.recuse', 'B'],
      ['Co-commissioner rules on any appeal', 'trade.appeal.rule', 'B'],
    ],
    'Nothing.',
    '—',
  ],
  [
    'TRADES',
    'A trade went through by mistake',
    [
      ['Reverse it with compensating entries', 'correct.transaction', 'B'],
      ['Or undo the ledger entry', 'commish.correction.revert', 'B'],
    ],
    'Agreeing with both owners.',
    'Rosters, caps and picks of both teams.',
  ],
  [
    'ROSTERS',
    'A player was cut by mistake',
    [
      ['Put him back', 'correct.roster', 'B'],
      ['Restore the contract', 'commish.contract.edit', 'B'],
      ['Remove the dead cap', 'commish.cap.adjust', 'B'],
    ],
    'Nothing.',
    'Waiver claims already placed on him.',
  ],
  [
    'ROSTERS',
    'A team is over the roster limit',
    [
      ['See the must-cut job (no auto-cut by design)', 'Jobs › Must-cut', 'B'],
      ['Allow it for a time', 'commish.reserve.exception', 'B'],
    ],
    'Getting the owner to cut.',
    'Lineup validity.',
  ],
  [
    'ROSTERS',
    'Someone needs IR or taxi outside the rules',
    [['Allow it, time-boxed', 'commish.reserve.exception', 'B']],
    'Nothing.',
    'Roster count.',
  ],
  [
    'MONEY',
    'A team is over the cap',
    [
      ['Adjust the cap for that team', 'commish.cap.adjust', 'B'],
      ['Or allow it until a date', 'rule.exception', 'B'],
    ],
    'Nothing.',
    'Bids and trades for that team.',
  ],
  [
    'MONEY',
    'A contract was entered wrong',
    [['Edit the contract', 'commish.contract.edit', 'B']],
    'Nothing.',
    'Cap and pending trades.',
  ],
  [
    'MARKET',
    'A waiver or bid was awarded wrong',
    [
      ['Override the waiver award', 'waivers.award.override', 'B'],
      ['Override the bid award', 'bids.award.override', 'B'],
    ],
    'Nothing.',
    "The losing team's cap and roster.",
  ],
  [
    'MARKET',
    'A blind auction ends tied',
    [
      ['Process by the tie rule', 'bids.process', 'B'],
      ['Or award by hand', 'bids.award.override', 'B'],
    ],
    'Bids stay private, even to you.',
    '—',
  ],
  [
    'CALENDAR',
    'One action is needed outside its window',
    [['Allow one action for one team, time-boxed', 'commish.calendar.exception', 'B']],
    'Nothing; it is visible to all owners.',
    '—',
  ],
  [
    'CALENDAR',
    'A deadline needs to move',
    [['Drag the window on the calendar', 'calendar.window.move', 'B']],
    'Telling the league.',
    'Everything that window opens.',
  ],
  [
    'DRAFT',
    'A draft pick went wrong',
    [
      ['Change the pick', 'draft.change', 'B'],
      ['Skip, assign or insert', 'draft.skip / assign / insert', 'B'],
      ['Restore a checkpoint', 'draft.revert', 'B'],
    ],
    'Nothing.',
    'Rookie contracts.',
  ],
  [
    'DRAFT',
    'The startup draft stalls',
    [
      ['Pause or reset the clock', 'draft.timer', 'B'],
      ['Skip or assign the pick', 'draft.skip / draft.assign', 'B'],
    ],
    'Nudging the owner.',
    '—',
  ],
  [
    'PICKS',
    'A mini-game winner earns a pick',
    [
      ['Create and grant the pick', 'pick.grant', 'R'],
      ['Tell the league', 'commish.notice.publish', 'B'],
    ],
    'Running the mini-game (side games are later).',
    "That team's pick count against the draft maximum.",
  ],
  [
    'PICKS',
    'A team is penalised with a lost pick',
    [['Forfeit the pick (typed reason, visible)', 'pick.forfeit', 'R']],
    'The decision.',
    'Draft order for that round.',
  ],
  [
    'PICKS',
    'A pick belongs to the wrong team',
    [['Move the pick', 'correct.pick', 'B']],
    'Nothing.',
    "Both teams' pick lists.",
  ],
  [
    'STANDINGS',
    'Standings must be set by hand',
    [
      ["Adjust a team's record", 'commish.standings.record.adjust', 'B'],
      ['Or set the full order', 'commish.standings.override', 'B'],
    ],
    'Nothing.',
    'Seeds and draft order.',
  ],
  [
    'STANDINGS',
    'A playoff game must be decided by hand',
    [['Advance the team (reason)', 'bracket.advance', 'B']],
    'The decision.',
    'Next round.',
  ],
  [
    'RULES',
    'A rule loophole shows up mid-season',
    [
      ['Change the rule (next rollover by default)', 'commish.rules.set', 'B'],
      ['Close it now with a reason', 'commish.rules.set effective=now', 'B'],
      ['Override single outcomes meanwhile', 'rule.exception', 'B'],
    ],
    'Writing what the rule means.',
    'Everything the rule governs.',
  ],
  [
    'RULES',
    'A house rule exists only in words',
    [
      ['Add it to the rulebook text', 'rules.text.add', 'B'],
      ['Words become an enforced rule (later, chat)', '—', 'G'],
    ],
    'Enforcing it by hand.',
    '—',
  ],
  [
    'LEAGUE',
    'The league expands or shrinks',
    [
      ['Change teams, conferences, divisions', 'Rules › League & teams', 'B'],
      ['Re-align divisions', 'league.alignment', 'B'],
      ['Guided expansion flow', '—', 'G'],
    ],
    'Expansion draft, new owners.',
    'Schedule, playoffs, draft order.',
  ],
  [
    'LEAGUE',
    'The commissioner steps down or goes quiet',
    [
      ['Step down, naming a successor', 'commish.succession.stepdown', 'B'],
      ['Deputy takes over after inactivity', 'commish.succession.takeover', 'B'],
    ],
    'Nothing.',
    'Roles.',
  ],
  [
    'LEAGUE',
    'The league wants a new commissioner',
    [
      ['Open a vote-out or election (co-commish or DOT)', 'commish.succession.voteout.open', 'B'],
      ['Owners vote', 'commish.vote.cast', 'B'],
    ],
    'Campaigning.',
    'Roles.',
  ],
  [
    'LEAGUE',
    'Undo a change',
    [
      [
        'Revert any ledger entry (except the MFL cutover, which says so first)',
        'commish.correction.revert',
        'B',
      ],
    ],
    'Nothing.',
    'Whatever the entry touched.',
  ],
  [
    'LEAGUE',
    'Bring in an AI helper',
    [['Connect an AI agent: job, scope, expiry', 'People › AI agents', 'R']],
    "Watching what it does (ledger shows 'via AI').",
    '—',
  ],
];
/** Extra words each situation answers to in the find box. */
export const synonyms: Readonly<Record<number, string>> = {
  '0': 'weather hamlin emergency suspended',
  '6': 'left quit gone abandoned missing absent',
  '7': 'left quit leave sell',
  '9': '2fa totp phone lost locked',
  '10': 'cheat unfair lopsided veto',
  '13': 'dropped released accident',
  '16': 'cap space over money',
  '24': 'reward prize minigame pickem contest',
  '25': 'penalty punish take away',
  '30': 'loophole exploit',
  '31': 'expansion contraction add teams',
  '33': 'impeach remove commissioner',
};

/** Fix: one thing went wrong. The example fields are left out: the real form asks for the real values. */
export const fixes: readonly (readonly [thing: string, words: string, command: string])[] = [
  ['A stat', 'approve, reject or enter a stat by hand', 'commish.stat.correct'],
  ['A score', 'add or take points from a player or team', 'commish.score.player'],
  ['A lineup', 'set the lineup that counts, after the lock', 'commish.lineup.correct'],
  ['A player', 'position the feed got wrong', 'commish.player.position'],
  ['A roster', 'add or remove a player by correction', 'correct.roster'],
  ['A transaction', 'reverse a trade, claim or move', 'correct.transaction'],
  ['Money', 'contract, cap adjustment, cap relief', 'commish.contract.edit'],
  ['Standings', 'a record, or the full order', 'commish.standings.record.adjust'],
  ['A rule outcome', 'one exception to an automatic rule', 'rule.exception'],
  ['A window', 'one action outside its window', 'commish.calendar.exception'],
  ['Reserve', 'IR or taxi outside the rules', 'commish.reserve.exception'],
  ['Undo', 'revert any ledger entry', 'commish.correction.revert'],
];
export const pickActs: readonly (readonly [name: string, words: string, command: string])[] = [
  ['Create and grant a pick', 'from nothing, e.g. a mini-game reward', 'pick.grant'],
  ['Move a pick', 'to another team', 'correct.pick'],
  ['Swap two picks', 'between teams', 'pick.swap'],
  ['Forfeit a pick', 'penalty; it leaves the draft', 'pick.forfeit'],
  ['Add a condition', 'protection, or depends on a result', 'pick.condition'],
  ['Void or restore', 'a granted or forfeited pick', 'pick.void'],
];
export const launch: readonly (readonly [name: string, key: string])[] = [
  ['Approve stat changes', 'stat'],
  ['Review a trade', 'trade'],
  ['Run waivers now', 'waivers'],
  ['Send a league notice', 'notice'],
  ['Invite an owner', 'invite'],
  ['Grant or move a pick', 'pick'],
  ['Fix something', 'fix'],
  ['Change a rule', 'rule'],
];
