import { expect, it } from 'vitest';
import { readHostedActData } from './actReadings';
import type { NativeRequest } from '../commands/native';
import type { HostedPort } from './hostedRoom';

const league = 'native.test';
const directory = { league, name: 'Test League', revision: 7, rosterSeason: 2026, source: 'held',
  franchises: [{ franchise: 'one', name: 'One', abbreviation: 'ONE', owner: 'owner',
    caretaker: '', division: 'North' }] };
const response = (data: unknown) => ({ data, label: 'League server', writable: true, football: 'held' });

it('holds draft and published schedule games as hosted form choices', async () => {
  const commands: string[] = [];
  const port: HostedPort = async (request: NativeRequest) => {
    commands.push(request.command);
    if (request.command === 'native.directory.read') return response(directory);
    if (request.command === 'commish.schedule.read') return response({ league, revision: 7,
      draft: [{ week: 1, games: [{ id: 'draft-1', home: 'one', away: 'two' }] }],
      published: [{ week: 1, games: [{ id: 'live-1', home: 'two', away: 'one' }] }] });
    throw new Error(`Unexpected read ${request.command}`);
  };
  const held = await readHostedActData(league, 7, ['scheduleGame', 'franchise'], 'schedule.edit', port);
  expect(held.options.scheduleGame).toEqual([
    {
      value: JSON.stringify({ week: 1, id: 'draft-1', published: false }),
      label: 'Draft week 1: One v Unnamed team',
    },
    {
      value: JSON.stringify({ week: 1, id: 'live-1', published: true }),
      label: 'Published week 1: Unnamed team v One',
    },
  ]);
  expect(commands).toEqual(['native.directory.read', 'commish.schedule.read']);
});

it('holds playoff presets, current shape, seeds and game routes for bracket tools', async () => {
  const commands: string[] = [];
  const bracket = {
    league, name: 'Test League', season: 2026, revision: 7, status: 'Ready', words: 'Ready to build',
    preset: 'league_rules', presets: ['saved-layout'],
    rules: { words: '', perConference: 1, divisionWinners: 1, wildCards: 0, byeSeeds: 0,
      seeding: 'rules', reseed: false, startWeek: 15, finalWeek: 16, homeBonus: '0', shape: [
        { id: 'final', week: 16, homeSeed: 1, awaySeed: 1 },
      ] },
    started: false, startKnown: false,
    fields: [{ conference: 'North', seeds: [{ seed: 1, franchise: 'one', name: 'One',
      division: 'North', rank: 1, kind: 'Division winner', bye: false, byHand: false }] }],
    projected: [], rounds: [{ round: 1, week: 16, name: 'Final', games: [{ id: 'final', node: 'final',
      conference: 'North', week: 16, homeSlot: 'final.home', awaySlot: 'final.away', status: 'Waiting',
      homeBonus: '0', byHand: false, words: 'Waiting for teams' }] }],
    champion: null, title: {}, history: [], changes: [],
  };
  const port: HostedPort = async (request: NativeRequest) => {
    commands.push(request.command);
    if (request.command === 'native.directory.read') return response(directory);
    if (request.command === 'bracket.read') return response(bracket);
    throw new Error(`Unexpected read ${request.command}`);
  };
  const held = await readHostedActData(league, 7, ['number', 'preset', 'bracketGame', 'bracketNode'],
    'bracket.build', port, 2026);
  expect(held.options.preset?.map(option => option.value)).toEqual([
    'league_rules', 'legacy_nfl', 'saved-layout',
  ]);
  expect(held.options.shape?.[0]?.label).toBe('Current playoff layout');
  expect(held.options.bracketSlot).toEqual([
    { value: '1', label: 'Seed 1 in the chosen team’s conference' },
  ]);
  expect(held.options.bracketGame?.[0]?.label).toBe(
    'Final, week 16: Team to be decided v Team to be decided',
  );
  expect(held.options.bracketNode).toHaveLength(2);
  expect(commands).toEqual(['native.directory.read', 'bracket.read']);
});
