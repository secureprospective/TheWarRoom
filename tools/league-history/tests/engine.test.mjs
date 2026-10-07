// The browser engine, run against the real compiled data. Run: node --test tests/
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {decode, GROUPS, BANDS, fmt, kindLabel} from '../app/js/model.js';
import {ANCHOR, valuedTrades, fitPrices, priceList, deliveredRatio, perDelivered, evidence, shrink, meanVar,
  judgedPicks, draftGrid, positionEffects, leagueMarket, tradesByPhase, pickClock, contenderTrades, assetProfile, stretchPrices} from '../app/js/engine.js';
import {PHASES} from '../app/js/model.js';
import {tradeSides, sidesInScope, behaviour, tierTrends, teamRows, teamProfile, TIERS} from '../app/js/teams.js';
import {draftScope, draftMap, positionDraft, slotCurve, playsLike} from '../app/js/draftlab.js';

const raw = JSON.parse(readFileSync(new URL('../data/lab.json', import.meta.url)));
const data = decode(raw);

test('the browser price model matches the compiler’s', () => {
  const fit = fitPrices(valuedTrades(data));
  for (const c of raw.market.classes) {
    const mine = fit.prices.get(c.id);
    assert.ok(Math.abs(mine.price - c.price) < 0.01, `${c.id}: ${mine.price} vs ${c.price}`);
  }
  assert.equal(fit.prices.get(ANCHOR).price, 1);
});

test('the seasons and time-of-year filters narrow the trades priced', () => {
  const all = priceList(data, {from: 2017, to: 2026});
  const late = priceList(data, {from: 2022, to: 2026});
  const season = priceList(data, {from: 2017, to: 2026, windows: ['season']});
  assert.ok(late.trades < all.trades && season.trades < all.trades);
  assert.ok(valuedTrades(data, t => t.season >= 2022).every(v => v.t.season >= 2022));
  assert.equal(priceList(data, {from: 2030, to: 2030}), null);
});

test('price per delivered win carries both uncertainties, and thin rows say so', () => {
  assert.equal(deliveredRatio(Array.from({length: 7}, () => ({expected: 1, delivered: 1}))), null);
  const d = deliveredRatio(Array.from({length: 20}, (_, i) => ({expected: 1, delivered: i % 2 ? 1.5 : 0.5})));
  assert.equal(d.ratio, 1);
  const r = perDelivered({price: 0.5, se: 0.05}, d);
  assert.ok(r.lo < 0.5 && r.hi > 0.5);
  assert.equal(evidence(null), 'none');
  assert.equal(evidence({spread: 2}), 'solid');
  assert.equal(evidence({spread: 20}), 'thin');
});

test('shrinking pulls small groups harder and leaves identical groups alone', () => {
  const out = shrink([{n: 4, mean: 1, var: 1}, {n: 400, mean: 1, var: 1}, {n: 50, mean: 0, var: 1}, {n: 50, mean: -1, var: 1}], 0);
  assert.ok(Math.abs(out[0].shrunk) < Math.abs(out[1].shrunk));
  const flat = shrink([{n: 30, mean: 0.2, var: 1}, {n: 30, mean: 0.2, var: 1}, {n: 30, mean: 0.2, var: 1}], 0.2);
  assert.ok(flat.every(c => Math.abs(c.shrunk - 0.2) < 1e-9));
});

test('the buy and sell lists only hold kinds whose whole range clears the going rate', () => {
  const m = leagueMarket(data, {from: 2017, to: 2026});
  assert.ok(m.going > 0);
  for (const r of m.rows.filter(r => r.cost)) {
    if (r.side === 'cheap') assert.ok(r.cost.hi < 1, r.kind);
    if (r.side === 'dear') assert.ok(r.cost.lo > 1, r.kind);
    if (r.holds) assert.ok(r.halves.every(h => h?.cost && (h.cost.value < 1) === (r.side === 'cheap')), r.kind);
  }
  // Measured 2026-10-06: linebackers the cheapest wins, next-draft 1sts the dearest, both in each half.
  const row = k => m.rows.find(r => r.kind === k);
  assert.equal(row('LB').side, 'cheap');
  assert.ok(row('LB').holds);
  assert.equal(row('pick1:next').side, 'dear');
  assert.ok(row('pick1:next').holds);
});

test('the time of year narrows the trades priced; splitting by age keeps the same going rate', () => {
  const all = leagueMarket(data, {from: 2017, to: 2026});
  const season = leagueMarket(data, {from: 2017, to: 2026, phases: ['early', 'midseason', 'deadline']});
  assert.ok(season.trades < all.trades && season.trades > 400);
  assert.ok(season.rows.every(r => r.times.every(t => t.window === 'season')));
  const aged = leagueMarket(data, {from: 2017, to: 2026, byAge: true});
  assert.equal(aged.going, all.going);
  assert.ok(aged.rows.some(r => r.kind === 'LB:prime') && !aged.rows.some(r => r.kind === 'LB'));
  // Too few trades to price: the view says how many, and prices nothing.
  const winter = leagueMarket(data, {from: 2017, to: 2026, phases: ['winter']});
  assert.equal(winter.rows, null);
  assert.ok(winter.trades > 0);
});

test('the league year counts every completed trade once and keeps the halves apart', () => {
  const byPhase = tradesByPhase(data, {from: 2017, to: 2025});
  const clean = data.trades.filter(t => !t.oneSided && t.phase && t.year >= 2017 && t.year <= 2025).length;
  assert.equal(byPhase.total, clean);
  for (const p of byPhase.phases.values()) assert.ok(p.span[0] <= p.span[1]);
  const clock = pickClock(data, {from: 2017, to: 2026});
  assert.equal(clock.halves.length, 2);
  assert.ok(clock.halves[0].to < clock.halves[1].from);
  // Measured 2026-10-06: a 1st a year out trades at under half a next-draft 1st, in both halves.
  assert.ok(clock.halves.every(hv => hv.clock.clear && hv.clock.value < 0.5));
  const deals = contenderTrades(data, {from: 2017, to: 2026});
  assert.ok(deals.n > 300 && deals.half > 0);
});

test('one asset’s own market: why, when, against everything else, what came back', () => {
  const scope = {from: 2017, to: 2026};
  const stretches = stretchPrices(data, scope, PHASES.map(p => p.id));
  const lb = assetProfile(data, scope, 'LB', stretches);
  // Measured 2026-10-06: linebackers are cheap for both reasons, low price and over-delivery.
  assert.ok(lb.paid.hi < 1 && lb.delivered.ratio - 1.96 * lb.delivered.se > 1);
  assert.equal(lb.when.length, 8);
  const vsFirst = lb.against.find(x => x.kind === 'pick1:next');
  assert.ok(vsFirst.byDelivered && vsFirst.hi < 1 && vsFirst.side === 'cheaper');
  assert.ok(Math.abs(lb.back.reduce((t, x) => t + x.share, 0) - 1) < 1e-9);
  // The counts on the league year follow the asset filter.
  const all = tradesByPhase(data, {from: 2017, to: 2025}), mine = tradesByPhase(data, {from: 2017, to: 2025, assets: ['LB']});
  assert.ok(mine.total > 0 && mine.total < all.total);
});

test('picks are judged only on seasons already played, against their slot', () => {
  const picks = judgedPicks(data, {seasons: 3, from: 2017, to: 2026});
  assert.ok(picks.length > 1000);
  assert.ok(picks.every(p => p.year + 2 <= data.lastFinished && Math.abs(p.over - (p.got - p.expected)) < 1e-12));
  const g = draftGrid(picks, GROUPS, BANDS);
  assert.equal([...g.values()].reduce((s, b) => s + b.n, 0), picks.length);
  const eff = positionEffects(picks, GROUPS, BANDS);
  assert.equal(eff.reduce((s, e) => s + e.n, 0), picks.filter(p => p.group).length);
});

test('the draft against its slots: by season, by stretch, by NFL round, read back onto the curve', () => {
  const picks = draftScope(data, {from: 2017, to: 2026, seasons: 3});
  const judged = picks.filter(p => p.judged);
  assert.ok(judged.length > 1000 && judged.every(p => p.year + 2 <= data.lastFinished));
  const map = draftMap(picks, GROUPS);
  const dl = map.find(r => r.group === 'DL');
  // Measured 2026-10-06: defensive linemen pay late, well short of the slot in year 1, level by year 4.
  assert.equal(dl.when[0].tone, 'down');
  assert.ok(dl.when[3].ratio > 0.9);
  const curve = slotCurve(picks);
  for (let o = 2; o < curve.length; o++) assert.ok(curve[o] <= curve[o - 1]);
  assert.equal(playsLike(curve, curve[40]), playsLike(curve, curve[40]));
  assert.ok(playsLike(curve, curve[1] * 2) === 1);
  const p = positionDraft(picks, 'DB');
  assert.equal(p.where.length, 7);
  assert.equal(p.n, judged.filter(x => x.group === 'DB').length);
  // The filters narrow the picks.
  assert.ok(draftScope(data, {from: 2017, to: 2026, seasons: 3, segments: ['r3']}).every(x => x.round === 3));
  assert.ok(draftScope(data, {from: 2017, to: 2026, seasons: 3, nfl: ['n1']}).every(x => x.nflRound === 1));
});

test('every valued trade is two team sides that add up', () => {
  const sides = tradeSides(data);
  assert.equal(sides.length, 2 * data.trades.filter(t => t.valued).length);
  // On the day is zero-sum: the two sides of a trade cancel.
  assert.ok(Math.abs(sides.reduce((s, x) => s + x.now, 0)) < 1e-6);
  assert.ok(sides.every(s => ['now', 'later', 'swap'].includes(s.posture)));
  const scope = sidesInScope(data, {from: 2017, to: 2026, tiers: ['rebuild'], windows: ['season']});
  assert.ok(scope.length > 0 && scope.every(s => s.tier === 'rebuild' && s.window === 'season'));
  const league = sidesInScope(data, {from: 2017, to: 2026});
  const rows = teamRows(league, league);
  assert.equal(rows.length, 32);
  assert.equal(rows.reduce((t, r) => t + r.n, 0), league.length);
  const trends = tierTrends(league);
  assert.equal(trends.tiers.length, TIERS.length);
  const p = teamProfile(data, league, league, rows[0].fid);
  assert.equal(p.sides.length, rows[0].n);
  assert.equal(p.bySeason.reduce((t, y) => t + y.n, 0), rows[0].n);
  assert.ok(behaviour(league).posture.swap.p > 0.4);
});

test('every offer is on file once, with its ending, its stretch of the year and its season', () => {
  assert.equal(data.offers.length, 470);
  // MFL shows a franchise only its own offers: every one involves the same team.
  const owner = data.offers[0].by === data.offers[1].by || data.offers[0].by === data.offers[1].to ? data.offers[0].by : data.offers[0].to;
  assert.ok(data.offers.every(o => o.by === owner || o.to === owner));
  const endings = new Set(['accepted', 'rejected', 'withdrawn', 'expired', 'unknown']);
  assert.ok(data.offers.every(o => endings.has(o.outcome) && PHASES.some(p => p.id === o.phase)));
  assert.ok(data.offers.filter(o => o.outcome === 'unknown').length <= 30);
  assert.ok(data.offers.every(o => o.decided == null || o.decided >= o.ts));
});

test('labels and numbers read cleanly', () => {
  assert.equal(fmt.signed(-0.001, 2), '0.00');
  assert.equal(fmt.signed(0.126, 2), '+0.13');
  assert.equal(kindLabel('LB:prime'), 'Linebacker, 25 to 27');
  assert.equal(kindLabel('pick3:later'), '3rd or later-round pick, a later draft');
});

test('mean and spread of a list', () => {
  assert.deepEqual(meanVar([1, 2, 3]), {n: 3, mean: 2, var: 1});
});
