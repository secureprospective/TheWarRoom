import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { all, button, click, flush, mount, unmount } from '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { WarRoomContext } from './warroomReads';

const reading = vi.hoisted(() => ({
  players: [
    { id: '0001', name: 'Need and affordable', position: 'WR', minimum: '$85M', minimumCents: 8500000000,
      age: '25', points: '12', value: '10' },
    { id: '0002', name: 'Affordable only', position: 'RB', minimum: '$50M', minimumCents: 5000000000,
      age: '24', points: '8', value: '7' },
    { id: '0003', name: 'Need but costly', position: 'WR', minimum: '$90M', minimumCents: 9000000000,
      age: '26', points: '4', value: '3' },
    { id: '0004', name: 'Kicker need', position: 'K', minimum: '$1M', minimumCents: 100000000,
      age: '23', points: '2', value: '2' },
  ], capRoom: 8500000000 as number | undefined, targets: [], wins: [] as {
    player: string; name: string; team: string; salaryCents: number; years: number; decidedAt: string;
  }[], errors: [], integrate: false,
}));
vi.mock('./warroomReads', async () => {
  const React = await import('react');
  return { useWarRoomRead: (key: string, load: () => Promise<unknown>) => {
    const loader = React.useRef(load);
    loader.current = load;
    const [state, setState] = React.useState<{ reading?: unknown; error?: string; pending: boolean }>({
      pending: true,
    });
    React.useEffect(() => {
      if (!reading.integrate) return;
      let active = true;
      setState(current => ({ ...current, pending: true, error: undefined }));
      void loader.current().then(value => {
        if (active) setState({ reading: value, pending: false });
      }, error => {
        if (active) setState({ error: String(error), pending: false });
      });
      return () => { active = false; };
    }, [key]);
    return reading.integrate ? state : { reading };
  } };
});
import { WarRoomMarket } from './WarRoomMarket';

beforeEach(() => { reading.capRoom = 8500000000; reading.integrate = false; });
afterEach(async () => { reading.integrate = false; await unmount(); vi.restoreAllMocks(); });

const context = { mine: 'franchise.mine', snapshot: {} as WarRoomContext['snapshot'], week: 1,
  native: { league: 'native.test', mine: 'franchise.mine', owner: true, leave: () => {} } } as WarRoomContext;

it('filters hosted players by the header needs and server cap room', async () => {
  const host = await mount(<WarRoomMarket context={context} needs={[
    { position: 'WR', state: 'hole', have: 0, starts: 2 },
    { position: 'PK', state: 'thin', have: 1, starts: 2 },
    { position: 'RB', state: 'ok', have: 2, starts: 2 },
  ]} select={() => {}} />);
  await flush();
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Affordable only');
  expect(host.textContent).toContain('Need but costly');
  expect(host.textContent).toContain('Kicker need');

  await click(button(host, 'My needs'));
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Need but costly');
  expect(host.textContent).toContain('Kicker need');
  expect(host.textContent).not.toContain('Affordable only');

  await click(button(host, 'Affordable'));
  expect(host.textContent).toContain('Need and affordable');
  expect(host.textContent).toContain('Affordable only');
  expect(host.textContent).toContain('Kicker need');
  expect(host.textContent).not.toContain('Need but costly');
});

it('renders recent wins and the empty state from the hosted wins read', async () => {
  reading.integrate = true;
  const port = vi.fn<HostedPort>(async request => {
    if (request.command === 'native.assets.read') return { label: '', writable: true, football: 'pin',
      data: { league: 'native.test', revision: 4, seeded: true, entry: 'entry.1', source: 'held',
        pool: [], identities: [], corrections: [], snapshot: { season: 2026, franchises: [],
          players: [], picks: [] } } };
    if (request.command === 'market.wins.read') return { label: '', writable: true, football: 'pin',
      data: { league: 'native.test', wins: [{ player: '0001', name: 'Player One', team: 'Team 2',
        salaryCents: 1234000000, years: 3, decidedAt: '2026-11-17T12:00:00Z' }] } };
    throw new Error(`Unexpected ${request.command}`);
  });
  const hosted: WarRoomContext = { ...context, native: { league: 'native.test', mine: 'franchise.mine',
    leave: () => {}, port, owner: false,
    reading: { settings: { maxYears: 4 }, runway: [{ room: 8500000000 }] } as
      NonNullable<WarRoomContext['native']>['reading'] } };
  const host = await mount(<WarRoomMarket context={hosted} select={() => {}} />);
  await flush();
  expect(host.textContent).toContain('Player One');
  expect(host.textContent).toContain('Team 2');
  expect(host.textContent).toContain('$12.34M');
  expect(host.textContent).toContain('3 years');
  expect(host.textContent).toContain(new Date('2026-11-17T12:00:00Z').toLocaleDateString());
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'market.wins.read', mode: 'read',
    arguments: { league: 'native.test' } }));

  await unmount();
  port.mockImplementation(async request => {
    if (request.command === 'native.assets.read') return { label: '', writable: true, football: 'pin',
      data: { league: 'native.test', revision: 4, seeded: true, entry: 'entry.1', source: 'held',
        pool: [], identities: [], corrections: [], snapshot: { season: 2026, franchises: [],
          players: [], picks: [] } } };
    if (request.command === 'market.wins.read') return { label: '', writable: true, football: 'pin',
      data: { league: 'native.test', wins: [] } };
    throw new Error(`Unexpected ${request.command}`);
  });
  const empty = await mount(<WarRoomMarket context={hosted} select={() => {}} />);
  await flush();
  expect(empty.textContent).toContain('No completed auction wins yet');
});

it('leaves My needs and Affordable disabled when their held inputs are unavailable', async () => {
  reading.capRoom = undefined;
  const host = await mount(<WarRoomMarket context={context} select={() => {}} />);
  await flush();
  expect(button(host, 'My needs').disabled).toBe(true);
  expect(button(host, 'Affordable').disabled).toBe(true);
});

it('renders hosted watch order and rereads after starring, unstarring and moving targets', async () => {
  reading.integrate = true;
  const ids = ['0001', '0002'];
  const commands: string[] = [];
  let watchReads = 0;
  const port = vi.fn<HostedPort>(async request => {
    commands.push(request.command);
    const args = request.arguments as unknown as Record<string, unknown>;
    const response = (data: unknown) => ({ label: '', writable: true, football: 'pin', data });
    switch (request.command) {
      case 'native.assets.read': return response({ league: 'native.test', revision: 4, seeded: true,
        entry: 'entry-1', source: 'held', identities: [], corrections: [],
        pool: [
          { player: '0001', name: 'Alpha', position: 'WR', string: 'FA', points: '12', age: '24' },
          { player: '0002', name: 'Beta', position: 'WR', string: 'FA', points: '10', age: '25' },
          { player: '0003', name: 'Gamma', position: 'WR', string: 'FA', points: '8', age: '26' },
        ], snapshot: { season: 2026, franchises: [], players: [], picks: [] } });
      case 'bids.read': return response({ league: 'native.test', franchise: 'franchise.mine',
        revision: 4, season: 2026, mode: 'open', activeBids: 0, activeLimit: '3',
        players: [], awards: [], notes: [] });
      case 'waivers.read': return response({ league: 'native.test', franchise: 'franchise.mine',
        revision: 4, season: 2026, players: [], claims: [], order: [], orderSource: '', method: '' });
      case 'value.read': return response({ league: 'native.test', season: 2026, franchise: '',
        preparing: false, players: [], best: { franchise: '', starterN: 0, value: 0, age: 0,
          ageKnown: false, rank: 0, of: 0, available: false, words: '' }, age: [], notes: [] });
      case 'market.wins.read': return response({ league: 'native.test', wins: [] });
      case 'player.watch.read': watchReads++; return response({ league: 'native.test', players: [...ids] });
      case 'player.watch.set': {
        const player = String(args.player);
        if (args.watched) ids.push(player);
        else ids.splice(ids.indexOf(player), 1);
        return response({ league: 'native.test', player });
      }
      case 'player.watch.order': {
        const from = ids.indexOf(String(args.player));
        const to = ids.indexOf(String(args.with));
        [ids[from], ids[to]] = [ids[to], ids[from]];
        return response({ league: 'native.test', player: args.player });
      }
      default: throw new Error(`Unexpected hosted command ${request.command}`);
    }
  });
  const hostedContext = { ...context, snapshot: { players: { value: [] } } as never,
    native: { ...context.native!, port,
      reading: { settings: { maxYears: 4 }, runway: [{ room: 8500000000 }] } as never } };
  const host = await mount(<WarRoomMarket context={hostedContext} select={() => {}} />);
  await flush();
  const targetNames = () => all(host, 'div').filter(node => node.getAttribute('class') === 'tgt')
    .map(row => ['Alpha', 'Beta', 'Gamma'].find(name => row.textContent.includes(name)) ?? '');
  expect(targetNames()).toEqual(['Alpha', 'Beta']);
  expect(watchReads).toBe(1);

  await click(button(host, 'Watch Gamma'));
  await flush();
  expect(targetNames()).toEqual(['Alpha', 'Beta', 'Gamma']);
  expect(watchReads).toBe(2);
  await click(button(host, 'Unwatch Beta'));
  await flush();
  expect(targetNames()).toEqual(['Alpha', 'Gamma']);
  expect(watchReads).toBe(3);
  await click(button(host, 'Move Gamma up'));
  await flush();
  expect(targetNames()).toEqual(['Gamma', 'Alpha']);
  expect(watchReads).toBe(4);
  await click(button(host, 'Move Gamma down'));
  await flush();
  expect(targetNames()).toEqual(['Alpha', 'Gamma']);
  expect(watchReads).toBe(5);
  expect(commands.filter(command => command === 'player.watch.set')).toHaveLength(2);
  expect(commands.filter(command => command === 'player.watch.order')).toHaveLength(2);
});
