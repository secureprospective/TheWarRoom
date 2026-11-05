import { describe, expect, it } from 'vitest';
import { alertsOf, seasonAlerts, tradeAlerts } from './HomeBrief';

const now = Date.parse('2026-11-05T12:00:00Z');
const day = 24 * 3600 * 1000;
const iso = (ms: number) => new Date(now + ms).toISOString();
const offer = (id: string, status: string, over: Partial<{ accepted: string[]; offeredAt: string }> = {}) => ({
  id, status, parties: ['f1', 'f2'], proposer: 'f2', accepted: ['f2'], offeredAt: iso(-day),
  expiresAt: iso(day), teams: [['f1', 'Syndicate'], ['f2', 'Iron Curtain']] as [string, string][],
  words: `words ${status}`, expires: '', details: [], reasons: [], ...over });
const trades = (offers: ReturnType<typeof offer>[]) =>
  ({ at: {}, errors: {}, trades: { league: 'L', franchise: 'f1', revision: 1, notes: [], offers } });

describe('home tray trade alerts', () => {
  it.each([
    ['offer waiting on me', offer('a', 'offered'), 'Trade offer from Iron Curtain'],
    ['already accepted by me', offer('b', 'offered', { accepted: ['f1', 'f2'] }), undefined],
    ['in review', offer('c', 'review'), 'Trade with Iron Curtain in review'],
    ['went through this week', offer('d', 'approved'), 'Trade with Iron Curtain went through'],
    ['declined this week', offer('e', 'declined'), 'Trade with Iron Curtain was declined'],
    ['ended over a week ago', offer('f', 'expired', { offeredAt: iso(-8 * day) }), undefined],
  ])('%s', (_, o, title) => {
    const got = tradeAlerts(trades([o]) as never, now);
    expect(got.map(a => a.title)).toEqual(title ? [title] : []);
  });
  it('ignores offers I am not party to', () => {
    expect(tradeAlerts(trades([{ ...offer('g', 'offered'), parties: ['f2', 'f3'] }]) as never, now)).toEqual([]);
  });
});

describe('home tray seasonal alerts', () => {
  it.each([
    ['deadline in 7h', { id: 'td', label: 'Trade deadline', at: iso(7 * 3600 * 1000) }, 'League deadline this week'],
    ['window closing in 6 days', { id: 'w', label: 'Free agency', at: iso(-day), closes: iso(6 * day) }, 'Window closes this week'],
    ['beyond a week', { id: 'x', label: 'Draft', at: iso(8 * day) }, undefined],
    ['past', { id: 'y', label: 'Old', at: iso(-day) }, undefined],
    ['undated', { id: 'z', label: 'Someday' }, undefined],
  ])('%s', (_, e, words) => {
    const got = seasonAlerts({ at: {}, errors: {}, calendar: [e] } as never, now);
    expect(got.map(a => a.words)).toEqual(words ? [words] : []);
  });
  it('joins the tray after cap and red jobs', () => {
    const b = { at: {}, errors: {}, calendar: [{ id: 'td', label: 'Trade deadline', at: iso(day) }],
      runway: [{ room: -100 }] };
    expect(alertsOf(b as never, now).map(a => a.id)).toEqual(['cap', 'season.td']);
  });
});
