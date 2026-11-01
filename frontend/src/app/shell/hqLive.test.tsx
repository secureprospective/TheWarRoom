import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { parseLive } from '../now/liveContract';
import wire from '../now/live.testdata.json';
import { liveTeam } from './hqLive';
import { HQBoard } from './HQBoard';
import { heldSnapshot, heldLineup } from '../data/lineupTestData';

it('renders LIVE and the live starter count from held NFL game states, not a weekday', () => {
  const live = parseLive(wire);
  live.games[0].status = 'in';
  const team = live.games[0].home.abbreviation;
  expect(liveTeam(team, live)).toBe(true);
  expect(liveTeam('XXX', live)).toBe(false);
  const snapshot = heldSnapshot(), reading = heldLineup();
  snapshot.players.value.find(p => p.id === reading.starters[0].id)!.team = team;
  const html = renderToStaticMarkup(<HQBoard snapshot={snapshot} mine="0025" reading={reading}
    settings={{ positions: [] }} values={new Map()} live={live} select={() => {}} />);
  expect(html).toContain('>LIVE<'); expect(html).toContain('1 of yours live now');
  live.games[0].status = 'post';
  expect(liveTeam(team, live)).toBe(false);
});
