import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { parseLive } from '../now/liveContract';
import wire from '../now/live.testdata.json';
import { FollowedGame, InThisGame, NflList, Plays, espnTeam } from './PulseNow';
import { pulseRows } from './pulseModel';
import { parsePulseResults } from '../data/nativePulse';
import pulse from './pulse.wire.testdata.json';

it('shows the followed NFL game: score, clock, field, down and its plays newest first', () => {
  const live = parseLive(wire);
  const g = live.games[0];
  g.status = 'in'; g.period = 2; g.clock = '4:31';
  g.state = { possession: g.home.id, down: 3, distance: 7, yardLine: 40, toEndzone: 60 };
  const html = renderToStaticMarkup(<FollowedGame game={g} />);
  expect(html).toContain('Q2 4:31');
  expect(html).toContain(`${g.home.abbreviation} ball`);
  expect(html).toContain('3rd &amp; 7');
  expect(html).toContain('60 to go for six');
  expect(html).toContain('class="los"');
  if (g.plays.length > 1) {
    const plays = renderToStaticMarkup(<Plays game={g} />);
    const newest = [...g.plays].sort((a, b) => b.sequence - a.sequence)[0];
    expect(plays.indexOf(newest.text.replace(/&/g, '&amp;'))).toBeGreaterThan(-1);
    expect(plays.match(/sp-row/g)?.length).toBe(Math.min(60, g.plays.length));
  }
  g.status = 'pre';
  expect(renderToStaticMarkup(<FollowedGame game={g} />)).toContain('Kickoff');
});

it('lists NFL games live first and never calls a finished game live', () => {
  const g = parseLive(wire).games[0];
  const games = [{ ...g, id: 'a', status: 'post' as const }, { ...g, id: 'b', status: 'pre' as const },
    { ...g, id: 'c', status: 'in' as const }];
  const html = renderToStaticMarkup(<NflList games={games} onFollow={() => {}} words="none" />);
  expect(html.indexOf('LIVE')).toBeLessThan(html.indexOf('FINAL'));
  expect(html.match(/>LIVE</g)?.length).toBe(1);
  expect(renderToStaticMarkup(<NflList games={[]} onFollow={() => {}} words="none held" />))
    .toContain('none held');
});

it('tags a matchup LIVE only while an NFL game is being played', () => {
  const read = parsePulseResults(pulse.results, pulse.results.league, pulse.results.week);
  const pending = { ...read, games: read.games.map(g => ({ ...g, status: 'No result yet',
    homePoints: undefined, awayPoints: undefined })) };
  expect(pulseRows(pending, '', [], true, false).every(r => r.status === 'IN PLAY')).toBe(true);
  expect(pulseRows(pending, '', [], true, true).every(r => r.status === 'LIVE')).toBe(true);
  expect(pulseRows(pending, '', [], false, true).every(r => r.status === 'WAITING')).toBe(true);
});

it("lists the matchup starters playing in the followed game, across league and ESPN team codes", () => {
  const g = parseLive(wire).games[0];
  const game = { ...g, away: { ...g.away, abbreviation: "NO" }, home: { ...g.home, abbreviation: "MIA" } };
  const row = (player: string, name: string, team: string, points?: number) => ({ slot: "QB", player,
    name, team, points, projection: undefined, words: "" });
  const html = renderToStaticMarkup(<InThisGame followed={game} sides={{ private: false,
    left: [row("1", "Mine In", "MIA", 12.5), row("2", "Mine Out", "BUF")],
    right: [row("3", "Theirs In", "NOS")] }} />);
  expect(html).toContain("Mine In"); expect(html).toContain("12.5");
  expect(html).toContain("Theirs In"); expect(html).not.toContain("Mine Out");
  expect(espnTeam("JAC")).toBe("JAX"); expect(espnTeam("MIA")).toBe("MIA");
});

it("says the lineups are still being read instead of claiming no starter plays", () => {
  const game = parseLive(wire).games[0];
  const reading = renderToStaticMarkup(<InThisGame followed={game}
    sides={{ private: false, reading: true, left: [], right: [] }} />);
  expect(reading).toContain("Reading this matchup"); expect(reading).not.toContain("No starter");
  const read = renderToStaticMarkup(<InThisGame followed={game} sides={{ private: false, left: [], right: [] }} />);
  expect(read).toContain("No starter");
});
