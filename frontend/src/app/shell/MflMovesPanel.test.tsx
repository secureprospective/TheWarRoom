import { afterEach, expect, it, vi } from 'vitest';
import { all, button, check, click, control, flush, mount, present, render, type, unmount } from
  '../commissioner/commissionerDom';
import type { HostedPort } from '../commissioner/hostedRoom';
import MflMovesPanel, { ageWords } from './MflMovesPanel';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

const reply = (data: unknown) => ({ label: '', writable: true, football: 'pin', data });
const teams = {
  mflLeague: '14432',
  teams: [{ id: '0001', name: 'Alpha' }, { id: '0002', name: 'Beta' }],
  yours: '',
};
const pool = { rows: [{ id: '100', name: 'Runner One', position: 'RB', team: 'BUF', age: '24', minimum: '$1',
  value: 'High' }], positions: ['RB'], sorts: ['value'], ageSeconds: 180 };
const plan = (over: object) => ({ id: 'p1', kind: 'bid', state: 'ready', words: 'Bid $5 on Runner One.',
  canDone: true, canWithdraw: true, ...over });

const mflRoster = { players: [
  { id: '1001', name: 'Quinn QB', position: 'QB', team: 'BUF', status: 'active' },
  { id: '1005', name: 'Ira Injured', position: 'WR', team: 'PHI', status: 'IR' }] };

function port(over: Record<string, (args: Record<string, unknown>) => unknown> = {}) {
  return vi.fn<HostedPort>(async request => {
    const handler = over[request.command];
    if (handler) return reply(handler(request.arguments as Record<string, unknown>));
    switch (request.command) {
      case 'mflmoves.teams.read': return reply({ ...teams, yours: '0001' });
      case 'mflmoves.pool.read': return reply(pool);
      case 'mflmoves.plans.read': return reply({ plans: [] });
      case 'mflmoves.roster.read': return reply(mflRoster);
      default: throw new Error(`unexpected ${request.command}`);
    }
  });
}

it('says one line when the league has no MFL number', async () => {
  const p = port({
    'mflmoves.teams.read': () => ({
      mflLeague: '',
      teams: [],
      words: 'This league follows no MFL league yet; the commissioner sets it in league settings.',
    }),
  });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'follows no MFL league yet')).toBe(true);
  expect(all(host, 'button')).toHaveLength(0);
});

it('lets the owner refresh from MFL on the pick screen when no teams are held yet', async () => {
  let held = false;
  const p = port({
    'mflmoves.teams.read': () => (held ? teams : {
      mflLeague: '14432', teams: [], words: 'Refresh from MFL to load the teams.',
    }),
    'mflmoves.refresh': () => { held = true; return { askedMfl: true, words: 'Fresh from MFL just now.' }; },
  });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'Refresh from MFL to load the teams.')).toBe(true);
  await click(button(host, 'Refresh from MFL'));
  await flush();
  expect(present(host, 'Alpha')).toBe(true);
  expect(present(host, 'Beta')).toBe(true);
});

it('lets the owner pick their team from the public list', async () => {
  const p = port({ 'mflmoves.teams.read': () => teams, 'mflmoves.team.set': () => ({}) });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  const picks = all(host, 'button').filter(b => b.textContent === 'This is my team');
  expect(picks).toHaveLength(2);
  await click(picks[1]);
  await flush();
  expect(p).toHaveBeenCalledWith(expect.objectContaining({ command: 'mflmoves.team.set',
    arguments: { league: 'L', team: '0002' } }));
  expect(present(host, 'Refresh from MFL')).toBe(true);
});

it('says the held copy age and that MFL was not asked again', async () => {
  expect(ageWords(180)).toBe('held 3 minutes ago');
  const p = port({ 'mflmoves.refresh': () => ({ askedMfl: false, words: 'x' }) });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'held 3 minutes ago')).toBe(true);
  await click(button(host, 'Refresh from MFL'));
  await flush();
  expect(present(host, 'MFL was not asked again')).toBe(true);
});

it('shows bid problems, cap room, and notes the server did not check', async () => {
  const p = port({ 'mflmoves.bid.check': () => ({
    ok: false, words: 'Not ready.', capRoomAfterCents: 304750000,
    notChecked: ['MFL names no roster minimum, so the minimum roster size was not checked.'],
    problems: ['This leaves you $2 over the cap.'] }) });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  await type(control(host, 'Player'), '100');
  await type(control(host, 'Amount'), '1250000');
  await click(button(host, 'Check'));
  await flush();
  expect(p).toHaveBeenCalledWith(expect.objectContaining({ command: 'mflmoves.bid.check',
    arguments: expect.objectContaining({ amountCents: 125000000 }) }));
  expect(present(host, 'This leaves you $2 over the cap.')).toBe(true);
  expect(present(host, 'Cap room after bid: $3,047,500.')).toBe(true);
  expect(present(host, 'MFL names no roster minimum, so the minimum roster size was not checked.'))
    .toBe(true);
});

it('opens only the server URL and hides buttons that are not allowed', async () => {
  const open = vi.fn();
  const p = port({
    'mflmoves.plans.read': () => ({
      plans: [plan({ id: 'p1' }), plan({ id: 'p2', canDone: false, canWithdraw: false })] }),
    'mflmoves.plan.link': () => ({ url: 'https://www.myfantasyleague.com/x' }),
  });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  vi.stubGlobal('window', { ...(globalThis.window as object), open });
  await flush();
  const buttons = (label: string) => all(host, 'button').filter(b => b.textContent === label);
  expect(buttons('Open on MFL')).toHaveLength(2);
  expect(buttons('I did it')).toHaveLength(1);
  expect(buttons('Withdraw')).toHaveLength(1);
  await click(buttons('Open on MFL')[0]);
  await flush();
  expect(open).toHaveBeenCalledTimes(1);
  expect(open.mock.calls[0][0]).toBe('https://www.myfantasyleague.com/x');
});

it('shows plan identity and details for saved bid and lineup replies', async () => {
  const p = port({ 'mflmoves.plans.read': () => ({ plans: [
    plan({ playerName: 'Aguilar, Joey', salaryCents: 33000000, years: 1, dropName: 'Quinn QB' }),
    plan({ id: 'p2', kind: 'lineup', week: 12, starters: Array(16).fill('id') }),
  ] }) });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'Bid: Aguilar, Joey, $330,000 per year for 1 yr, drop Quinn QB.')).toBe(true);
  expect(present(host, 'Lineup, week 12: 16 starters.')).toBe(true);
});

it('uses owner-facing sort labels and falls back for unknown keys', async () => {
  const p = port({ 'mflmoves.pool.read': () => ({ ...pool,
    sorts: ['name', 'position', 'team', 'ppg', 'lab', 'minimum', 'mystery'] }) });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  for (const label of ['Name', 'Position', 'NFL team', 'Points per game', 'Model value',
    'MFL minimum salary', 'mystery']) expect(present(host, label)).toBe(true);
});

it('uses one radio choice for bid drops and sends at most one', async () => {
  const p = port();
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'No drop')).toBe(true);
  const radios = all(host, 'input').filter(input => input.props().type === 'radio');
  expect(radios).toHaveLength(3);
  await check(radios[1], true);
  await type(control(host, 'Player'), '100');
  await type(control(host, 'Amount'), '5');
  await click(button(host, 'Check'));
  await flush();
  expect(p).toHaveBeenCalledWith(expect.objectContaining({ command: 'mflmoves.bid.check',
    arguments: expect.objectContaining({ drop: ['1001'] }) }));
});

it('refresh reloads the MFL roster used by Drops and Starters', async () => {
  let refreshed = false;
  const p = port({
    'mflmoves.refresh': () => { refreshed = true; return { askedMfl: true, words: 'Fresh.' }; },
    'mflmoves.roster.read': () => ({ players: refreshed
      ? [{ id: '2001', name: 'New Player', position: 'WR', status: 'active' }] : mflRoster.players }),
  });
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(present(host, 'Quinn QB')).toBe(true);
  await click(button(host, 'Refresh from MFL'));
  await flush();
  expect(present(host, 'New Player')).toBe(true);
});

it('drop list shows the MFL roster from roster.read', async () => {
  const p = port();
  const host = await mount(render(MflMovesPanel, { league: 'L', port: p }));
  await flush();
  expect(p).toHaveBeenCalledWith(expect.objectContaining({ command: 'mflmoves.roster.read',
    arguments: { league: 'L' } }));
  const drops = (label: string) => all(host, 'fieldset').filter(f => f.textContent?.startsWith(label))[0];
  expect(drops('Drops').textContent).toContain('Quinn QB');
  expect(drops('Drops').textContent).toContain('Ira Injured');
  expect(drops('Starters').textContent).toContain('Quinn QB');
});
