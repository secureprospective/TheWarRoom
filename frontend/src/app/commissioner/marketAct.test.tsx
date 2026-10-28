import { afterEach, expect, it } from 'vitest';
import { actTable } from './actTable';
import { actChange } from './actBody';
import { readActData } from './actReadings';
import { CommissionerActForm } from './CommissionerActTray';
import { all, button, click, control, disabled, mount, render, settle, type, unmount }
  from './commissionerDom';
import { stub, type Sent } from './commissionerStub';
import { leagueState } from './proxy';
import wire from './market.wire.testdata.json';
import previewWire from './preview.wire.testdata.json';

const team = 'franchise.01';
afterEach(async () => { await unmount(); leagueState.forget(); });
const cases = [
  ['waivers.process', 'pending', 'window', 'Waiver window'],
  ['waivers.award.override', 'claims', 'claim', 'Waiver claim'],
  ['bids.process', 'waitingBids', 'bidPlayer', 'Player'],
  ['bids.award.override', 'bids', 'bid', 'Bid'],
  ['trade.review', 'trades', 'trade', 'Trade'],
  ['trade.appeal.rule', 'trades', 'trade', 'Trade'],
  ['trade.override', 'trades', 'trade', 'Trade'],
] as const;
const rows = { window: 'windows', claim: 'claims', bidPlayer: 'players',
  bid: 'bids', trade: 'trades' } as const;

for (const [command, key, kind, label] of cases) {
  it(`${command}: a commissioner without a franchise chooses Go's held market target`, async () => {
    const reading = wire[key];
    const previews: Sent[] = [];
    const requests: string[][] = [];
    const { app, calls } = stub({
      ...{
        TargetNativeAssets: async () => ({ league: reading.league, revision: reading.revision }),
        TargetNativeDirectory: async () => ({ league: reading.league, revision: reading.revision,
          rosterSeason: 2026, franchises: [{ franchise: team, name: 'Held team 1' }] }),
        TargetCommissionerMarket: async (...args: string[]) => { requests.push(args); return reading; },
        TargetNativeWaivers: async () => { throw new Error('Owner door used'); },
        TargetNativeBids: async () => { throw new Error('Owner door used'); },
        TargetNativeTrades: async () => { throw new Error('Owner door used'); },
      },
      TargetCommissionerPreview: async body => {
        const sent = JSON.parse(body) as Sent;
        previews.push(sent);
        return { ...previewWire.reason,
          target: { league: sent.league, name: 'Market league', version: sent.expected },
          fields: [{ label: 'Market decision', before: 'Waiting', after: 'Recorded',
            beforeSource: 'League record', afterSource: 'Your change', when: 'Now' }] };
      },
    });
    const definition = actTable.find(d => d.command === command)!;
    const host = await mount(render(CommissionerActForm, { league: reading.league, definition }), app);
    await settle();
    expect(requests).toEqual([[reading.league, command]]);
    const options = reading[rows[kind]];
    expect(options.length).toBeGreaterThan(0);
    for (const row of options) expect(host.textContent).toContain(row.words);
    await type(control(host, label), options[0].id);
    if (definition.fields.some(f => f.path === 'franchise')) await type(control(host, 'Franchise'), team);
    if (definition.fields.some(f => f.path === 'decision')) await type(control(host, 'Decision'), 'approve');
    await type(control(host, 'Reason'), 'Correct the recorded award');
    await settle();
    expect(host.textContent).toContain('Waiting → Recorded');
    expect(host.textContent).not.toContain('League version:');
    expect(previews.at(-1)?.expected).toBe(reading.revision);
    const target = definition.fields[0].path;
    expect(previews.at(-1)?.args).toEqual({ [definition.root]: { [target]: options[0].id,
      ...(definition.fields.some(f => f.path === 'franchise') ? { franchise: team } : {}),
      ...(definition.fields.some(f => f.path === 'decision') ? { decision: 'approve' } : {}),
    } });
    expect(disabled(button(host, 'Record this change'))).toBe(false);
    await click(button(host, 'Record this change'));
    await settle();
    expect(calls.sent).toEqual([previews.at(-1)]);
  });
}

it('shows a refused market reading in Go’s words without falling back to an owner reading', async () => {
  const refusal = 'commissioner market: only an eligible trade reviewer may decide this trade';
  const { app } = stub({ ...{
    TargetNativeAssets: async () => ({ league: 'native.seed', revision: 1 }),
    TargetNativeDirectory: async () => ({ league: 'native.seed', revision: 1,
      rosterSeason: 2026, franchises: [] }),
    TargetCommissionerMarket: async () => { throw new Error(refusal); },
  } });
  const definition = actTable.find(d => d.command === 'trade.review')!;
  const host = await mount(render(CommissionerActForm, { league: 'native.seed', definition }), app);
  await settle();
  expect(all(host, 'p').find(p => p.getAttribute('role') === 'alert')?.textContent).toBe(refusal);
  expect(disabled(button(host, 'Record this change'))).toBe(true);
});

const selfCommands = ['commish.seat.invite.redeem', 'commish.succession.stepdown',
  'commish.succession.accept', 'commish.succession.decline', 'commish.succession.activity',
  'commish.succession.takeover', 'commish.succession.voteout.open', 'commish.succession.election.open',
  'commish.vote.cast', 'commish.vote.tiebreak'];
for (const command of selfCommands) {
  it(`${command}: hides the acting account and sends the local session identity`, async () => {
    const { app } = stub({ ...{
      TargetNativeAssets: async () => ({ league: 'native.seed', revision: 1 }),
      TargetNativeDirectory: async () => ({ league: 'native.seed', revision: 1,
        rosterSeason: 2026, franchises: [] }),
      TargetCommissionerActor: async () => 'signed.in.person',
    } });
    const definition = actTable.find(d => d.command === command)!;
    const host = await mount(render(CommissionerActForm, { league: 'native.seed', definition }), app);
    await settle();
    expect(all(host, 'label').some(l => /Sign-in|You$/.test(l.textContent))).toBe(false);
    expect(definition.fields.find(f => f.path === 'principal')?.kind).toBe('self');
    const held = await readActData('native.seed', ['self']);
    const values = Object.fromEntries(definition.fields.map(f => [f.path, 'another.person']));
    const change = actChange(definition, values, held, 'Record my decision');
    expect(change?.args).toMatchObject({ people: { principal: 'signed.in.person' } });
  });
}

it('keeps owner trade actions on the seated owner reading', async () => {
  const calls: string[][] = [];
  const { app } = stub({ ...{
    TargetNativeAssets: async () => ({ league: 'native.seed', revision: 1 }),
    TargetNativeDirectory: async () => ({ league: 'native.seed', revision: 1,
      rosterSeason: 2026, franchises: [] }),
    TargetNativeTrades: async (...args: string[]) => { calls.push(args); return { offers: [] }; },
    TargetCommissionerMarket: async () => { throw new Error('Wrong market door'); },
  } });
  await mount(render('div'), app);
  await readActData('native.seed', ['trade'], 2026, team, 'trade.appeal');
  expect(calls).toEqual([['native.seed', team]]);
});

it('sends the chosen side with a per-side bid decision, but not a single-pool decision', () => {
  const def = actTable.find(d => d.command === 'bids.process')!;
  const held = { league: 'test', revision: 1 };
  expect(actChange(def, { player: 'side?side=AFC&player=player.1' }, held, 'decide')?.args)
    .toEqual({ bid: { player: 'player.1', side: 'AFC' } });
  expect(actChange(def, { player: 'player.1' }, held, 'decide')?.args)
    .toEqual({ bid: { player: 'player.1' } });
});
