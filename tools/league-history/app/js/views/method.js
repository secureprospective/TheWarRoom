// How it's measured: the currency, the forecast, the market model, the backtest and the ledger,
// each with the check that says how far to trust it.

import {GROUPS, fmt} from '../model.js';
import {h, panel} from '../ui.js';

const defs = items => h('dl', {class: 'defs'}, items.flatMap(([t, d]) => [h('dt', {}, t), h('dd', {}, d)]));

function forecastTable(checks) {
  const cols = [1, 2, 3, 5];
  return h('table', {class: 'data compact'},
    h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Position'), cols.map(k => h('th', {scope: 'col', class: 'num'}, k === 1 ? 'Next season' : `${k} seasons on`)))),
    h('tbody', {}, GROUPS.map(g => h('tr', {}, h('th', {scope: 'row'}, g.label),
      cols.map(k => h('td', {class: 'num'}, checks[`${g.id}+${k}`] ? fmt.pct(checks[`${g.id}+${k}`].explained) : '—'))))));
}

function backtestTable(bt) {
  return h('table', {class: 'data compact'},
    h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Trades, by how lopsided they looked'), h('th', {scope: 'col', class: 'num'}, 'Trades'),
      h('th', {scope: 'col', class: 'num'}, 'Expected gain'), h('th', {scope: 'col', class: 'num'}, 'Delivered gain'), h('th', {scope: 'col', class: 'num'}, 'Favoured side ahead'))),
    h('tbody', {}, bt.bands.map((b, i) => h('tr', {},
      h('th', {scope: 'row'}, ['Closest fifth', 'Second fifth', 'Middle fifth', 'Fourth fifth', 'Most lopsided fifth'][i]),
      h('td', {class: 'num'}, b.n), h('td', {class: 'num'}, fmt.n(b.expected, 2)), h('td', {class: 'num'}, fmt.n(b.realized, 2)), h('td', {class: 'num'}, fmt.pct(b.won))))));
}

function ledgerTable(ledger) {
  return h('table', {class: 'data compact'},
    h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Season'), h('th', {scope: 'col', class: 'num'}, 'Players who changed teams week to week'),
      h('th', {scope: 'col', class: 'num'}, 'Explained by recorded moves'))),
    h('tbody', {}, Object.entries(ledger.byYear).map(([y, v]) => h('tr', {}, h('th', {scope: 'row'}, y),
      h('td', {class: 'num'}, fmt.n(v.changed)), h('td', {class: 'num'}, v.changed ? fmt.pct(v.changeRate, 1) : '—')))));
}

export default {
  id: 'method', nav: 'How it’s measured', hint: 'The checks behind every number',
  title: 'How it’s measured',
  lede: 'Every number in the Lab comes from MFL alone and rests on four things: one currency, a forecast, the league’s own trade prices, and a complete record of who held whom. Each is checked here against what actually happened.',
  render(main, ctx) {
    const {data} = ctx;
    const c = data.checks, m = data.market, bt = c.backtest, ledger = data.ledger;
    const src = data.source;
    main.append(panel('Where the numbers come from', {note: 'The league’s MFL archive, loaded into one database of facts and checked against MFL’s own totals before anything is counted. Days are US Eastern, as MFL shows them.'},
      h('p', {class: 'say'}, `${fmt.n(src.pages.matched)} of ${fmt.n(src.pages.asked)} answers — records, points, every draft pick’s team, player and day, birthdates — match what MFL’s own web pages show. The rest are on the review list, unexplained.`),
      h('table', {class: 'data compact'},
        h('thead', {}, h('tr', {}, h('th', {scope: 'col'}, 'Check against MFL'), h('th', {scope: 'col', class: 'num'}, 'Compared'),
          h('th', {scope: 'col', class: 'num'}, 'Agree'), h('th', {scope: 'col', class: 'num'}, 'Explained'), h('th', {scope: 'col', class: 'num'}, 'Open'))),
        h('tbody', {}, src.checks.map(k => h('tr', {title: k.detail}, h('th', {scope: 'row'}, k.name),
          h('td', {class: 'num'}, fmt.n(k.checked)), h('td', {class: 'num'}, fmt.n(k.agreed)),
          h('td', {class: 'num'}, fmt.n(k.explained)), h('td', {class: 'num'}, fmt.n(k.open))))))));
    main.append(panel('One currency: wins above replacement', {},
      defs([
        ['What it is', 'What a player adds over the best player a team could get for free at his position, in this league’s scoring and lineups, turned into extra matchups won. Each week, his points above that free player change the chance of beating an average team, using that season’s spread of weekly team scores.'],
        ['The free player', 'For each position and season, the players ranked just below the number this league actually starts at that position each week.'],
        ['Check', `Across ${fmt.n(c.currency.teamSeasons)} team-seasons, the wins above replacement a team started line up with its all-play record (its record if it had played everyone every week) at ${c.currency.correlation.toFixed(2)}, where 1 would be perfect. The currency describes winning.`],
      ])));
    main.append(panel('The forecast', {note: 'Share of the season-to-season variation in a player’s wins the forecast explains, tested on players it never saw.'},
      defs([
        ['What it is', 'What comparable players did next: same position group, age, last two seasons and how many games they played. Players who dropped out count as zero, so survivors don’t flatter it. Its straight-line fit is then corrected for how sharply older players fall away.'],
        ['Point in time', 'Anything judged "at the time" uses a forecast built only from the seasons before that moment.'],
      ]), forecastTable(c.forecast),
      h('p', {class: 'small muted'}, 'Quarterbacks and receivers are the most predictable; defensive players and kickers the least. Weak forecasts are why the price list judges every kind of asset by what it delivered, not only by what was forecast.')));
    main.append(panel('The league’s own prices', {},
      defs([
        ['What it is', `Each of ${fmt.n(m.trades)} trades since 2017 says the two sides were worth the same to the two GMs that day. Across all of them, that gives what the league pays per expected win for each kind of asset, against a next-draft 1st.`],
        ['Next season against later', `The league’s trades balance best when each later season counts ${m.discount} of the one before: next season’s win is worth a third more than the one after.`],
        ['Bigger packages', `Each extra asset in a deal is worth ${fmt.n(Math.abs(m.perAsset), 3)} wins less than its expected wins say: two smaller pieces don’t add up to one bigger one.`],
        ['Contracts', `Across ${fmt.n(m.capTrades)} trades with every contract known, $1M of salary a season cut a player’s trade price by only ${fmt.n(Math.abs(m.perCapMillion), 3)} wins. This league barely prices contracts.`],
      ])));
    main.append(panel('Would it have worked?', {note: `${fmt.n(bt.trades)} trades from ${bt.seasons[0]} to ${bt.seasons[1]}, each valued with only the seasons before it, then followed for every season played since.`},
      backtestTable(bt),
      h('p', {class: 'say'}, `On single trades the numbers are a weak guide: trades that looked lopsided on the day delivered about ${fmt.pct(bt.slope)} of the gap they showed, and the favoured side came out ahead ${fmt.pct(bt.won)} of the time. GMs know things a box score doesn’t. The edges this Lab shows are in kinds of asset, measured across many trades and checked against what those assets delivered.`)));
    main.append(panel('A complete record', {note: 'Start from MFL’s roster on one week, replay every recorded trade, add and drop to the next week, and compare.'},
      h('p', {class: 'say'}, `Of ${fmt.n(ledger.changed)} times a player changed teams between weekly rosters, the recorded moves explain ${fmt.pct(ledger.changeRate, 1)}, and every one from 2018 on. Before 2018, MFL’s stored weekly rosters barely change through a season, so those years can’t be tested this way.`),
      ledgerTable(ledger)));
    main.append(panel('What the archive can’t tell', {},
      h('ul', {class: 'notes'},
        h('li', {}, h('strong', {}, 'Who ran a franchise. '), 'The archive has no owner names, so a franchise’s habits are the habits of whoever ran it at the time.'),
        h('li', {}, h('strong', {}, 'Free-agent bids. '), 'Adds carry no bid amounts, so the free-agent market can’t be priced.'),
        h('li', {}, h('strong', {}, 'Picks before 2017. '), 'The rookie draft ran off MFL and traded picks weren’t recorded, so earlier trades aren’t valued.'),
        h('li', {}, h('strong', {}, 'The 2013 season. '), 'The league joined MFL mid-season; 2013 is not used for values.'))));
  },
};
