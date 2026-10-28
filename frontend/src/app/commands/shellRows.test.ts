import { it, expect } from 'vitest';
import { createCommands, type CommandId, type CommandArgs } from './registry';
import { FixtureProvider } from '../data/provider';
import { endpointById, type EndpointId } from '../registry';
import { searchCandidates, runResult } from './search';

const snapshot = await new FixtureProvider().snapshot();
type Invocation = { [K in CommandId]: { verb: K; args: CommandArgs<K> } }[CommandId];

it('dispatches the verb each phase registry row promises, using one table', async () => {
  const player = snapshot.rosters.value[0].players[0].id;
  const franchiseId = snapshot.franchises.value[0].id;
  const cases: { row: EndpointId; invocation: Invocation }[] = [
    { row: 'S-01', invocation: { verb: 'nav.open',
      args: { node: 'war', workspace: 'valuations-and-pool' } } },
    { row: 'S-02', invocation: { verb: 'inspector.pin',
      args: { entity: { kind: 'player', id: player, franchiseId } } } },
    { row: 'S-04', invocation: { verb: 'calendar.summon', args: {} } },
    { row: 'S-05', invocation: { verb: 'preset.apply', args: { preset: 'draft' } } },
    { row: 'S-08', invocation: { verb: 'commandbar.open', args: {} } },
    { row: 'S-11', invocation: { verb: 'watch.add',
      args: { id: player, entity: 'player', action: 'add' } } },
    { row: 'S-12', invocation: { verb: 'endpoint.open', args: { id: 'S-12' } } },
    { row: 'S-13', invocation: { verb: 'phase.goto', args: { phase: 'DRAFT' } } },
    { row: 'M-116', invocation: { verb: 'bait.set',
      args: { id: player, entity: 'player', action: 'add' } } },
    { row: 'M-117', invocation: { verb: 'watch.add',
      args: { id: player, entity: 'player', action: 'remove' } } },
    { row: 'M-118', invocation: { verb: 'note.add',
      args: { id: player, entity: 'player', action: 'add', text: 'My note' } } },
    ...(['M-096', 'M-149', 'M-103', 'M-101', 'M-154', 'M-087',
      'M-151', 'M-159', 'M-167', 'M-169'] as const).map(id => ({ row: id,
      invocation: { verb: 'endpoint.open', args: { id } } as Invocation })),
  ];
  for (const { row, invocation } of cases) {
    const c = createCommands(() => undefined);
    c.loadSnapshot(snapshot);
    expect(endpointById(row).verb.split(/[ =]/)[0]).toBe(invocation.verb);
    if (['watch.add', 'bait.set', 'note.add'].includes(invocation.verb)) {
      expect(() => c.dispatch(invocation.verb, invocation.args)).toThrow('Choose my franchise');
      expect(c.history().at(-1)).toMatchObject({ id: invocation.verb, status: 'failed' });
      continue;
    }
    await c.dispatch(invocation.verb, invocation.args);
    expect(c.history().at(-1)).toEqual({ id: invocation.verb, args: invocation.args, status: 'done' });
    if (invocation.verb === 'calendar.summon') expect(c.read().summoned).toBe('calendar');
    if (invocation.verb === 'commandbar.open') expect(c.read().commandbar).toBe(true);
    if (invocation.verb === 'inspector.pin') expect(c.read().subject?.id).toBe(player);
    if (invocation.verb === 'phase.goto' || invocation.verb === 'preset.apply')
      expect(c.read()).toMatchObject({ node: 'trade', workspace: { trade: 'draft-room' } });
    if (row === 'S-12') expect(c.read().endpointSurface).toBe('Status strip');
    if (row === 'M-151') expect(c.read().endpointIds).toEqual(['M-011', 'M-012']);
  }
});

it('pins all four entity kinds from any workspace without changing the page, then closes', () => {
  const c = createCommands(() => undefined);
  c.loadSnapshot(snapshot);
  const roster = snapshot.rosters.value[0];
  c.dispatch('nav.open', { node: 'control', workspace: 'app' });
  for (const entity of [
    { kind: 'player', id: roster.players[0].id, franchiseId: roster.franchiseId },
    { kind: 'franchise', id: snapshot.franchises.value[0].id },
    { kind: 'offer', id: '77', franchiseId: roster.franchiseId },
    { kind: 'pick', id: 'FP_0001_2027_1', name: '2027 round 1' },
  ] as const) {
    c.dispatch('commandbar.open', {});
    c.dispatch('inspector.pin', { entity });
    expect(c.read()).toMatchObject({ node: 'control', inspector: 'rest', commandbar: false,
      workspace: { control: 'app' } });
    c.dispatch('inspector.close', {});
    expect(c.read().inspector).toBe('closed');
  }
});

it('searches franchises and unrostered players beside commands and endpoints in one candidate set', () => {
  const c = createCommands(() => undefined);
  const expanded = { ...snapshot, players: { ...snapshot.players,
    value: [...snapshot.players.value,
      { id: '900001', name: 'Available rookie', position: 'WR' as const }] } };
  c.loadSnapshot(expanded);
  const candidates = searchCandidates(c, expanded);
  for (const kind of ['franchise', 'player', 'command', 'endpoint'])
    expect(candidates.some(r => r.kind === kind)).toBe(true);
  runResult(c, candidates.find(r => r.label.startsWith('Available rookie'))!);
  expect(c.read()).toMatchObject({ node: 'home', asset: 'pool:900001', inspector: 'rest' });
  runResult(c, candidates.find(r => r.kind === 'franchise')!);
  expect(c.read()).toMatchObject({ node: 'home', entity: { kind: 'franchise' }, inspector: 'rest' });
});
