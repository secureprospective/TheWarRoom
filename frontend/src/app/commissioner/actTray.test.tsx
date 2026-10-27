import { afterEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import { readFileSync } from 'node:fs';
import { actTable, type ActDefinition } from './actTable';
import { actChange } from './actBody';
import { CommissionerActForm } from './CommissionerActTray';
import { SeedForm, seedPreviewPause } from './SeedTray';
import { all, button, check, click, control, disabled, holder, mount, render, settle, type, unmount }
  from './commissionerDom';
import { open, receipt, stub, type Sent } from './commissionerStub';
import previewWire from './preview.wire.testdata.json';
import { leagueState } from './proxy';

const league = 'native.preview';
const franchise = 'franchise.01';
const player = '12345';
const item = 'stat.2026.1.12345.2.0123456789abcdef';
const previewBodies: Sent[] = [];
const assets = { league, revision: 3, seeded: true, snapshot: { season: 2026,
  players: [{ id: player, name: 'Player One', franchise }], pool: [], picks: [
    { season: 2027, round: 1, original: franchise, owner: franchise }], franchises: [{ id: franchise }] } };
const native = {
  TargetNativeAssets: async () => assets,
  TargetCommissionerMarket: async () => ({ league, revision: 3,
    windows: [{ id: 'window.one', words: 'Player One — Tomorrow' }],
    claims: [{ id: 'claim.one', words: 'Player One — Awarded' }],
    players: [{ id: player, words: 'Player One' }],
    bids: [{ id: 'bid.one', words: 'Player One — First franchise' }],
    trades: [{ id: 'trade.one', words: 'First trade' }],
  }),
  TargetNativeDirectory: async () => ({ league, revision: 3, rosterSeason: 2026,
    franchises: [{ franchise, name: 'First franchise' },
      { franchise: 'franchise.02', name: 'Second franchise' }] }),
  TargetNativeTrades: async () => ({ offers: [{ id: 'trade.one', words: 'First trade' }] }),
  TargetNativeBids: async () => ({ players: [{ name: 'Player One', player,
    leader: { id: 'bid.one', team: 'First franchise', contract: '$10 for one year' } }] }),
  TargetNativeWaivers: async () => ({ players: [{ window: 'window.one', player, name: 'Player One',
    closes: 'Tomorrow' }], claims: [{ id: 'claim.one', player, status: 'Waiting' }] }),
  TargetStatQueue: async () => ({ open: [
    { id: item, name: 'Player One', week: 1, words: 'Tackle changed' }] }),
};
afterEach(async () => { await unmount(); leagueState.forget(); previewBodies.length = 0; });
const definition = (command: string): ActDefinition => actTable.find(d => d.command === command)!;
const answer = (sent: Sent, blockers: string[] = [], safeguard = 'reason') => ({
  ...previewWire.reason, question: 'Record this league action?',
  target: { league: sent.league, name: 'Test league', version: sent.expected },
  fields: [{ label: 'Recorded value', before: 'Before value', after: 'After value',
    beforeSource: 'League ledger', afterSource: 'Your change', when: 'Now' }], blockers, safeguard,
});
async function show(command: string, blockers: string[] = [], safeguard = 'reason') {
  const { app, calls } = stub({
    ...native,
    TargetCommissionerPreview: async body => {
      const sent = JSON.parse(body) as Sent;
      previewBodies.push(sent);
      return answer(sent, blockers, safeguard);
    },
  });
  await mount(render('div'), app);
  await open(league);
  await unmount();
  const host = await mount(render(CommissionerActForm, { definition: definition(command), league }), app);
  await settle();
  return { host, calls, app };
}
async function fill(host: Parameters<typeof control>[0], values: Record<string, string | boolean>) {
  for (const [label, value] of Object.entries(values)) {
    if (typeof value === 'boolean') await check(control(host, label), value);
    else await type(control(host, label), value);
  }
  await type(control(host, 'Reason'), 'The recorded facts were wrong');
  await settle();
}
async function fire(host: Parameters<typeof control>[0], safeguard = 'reason') {
  expect(disabled(button(host, 'Record this change'))).toBe(false);
  if (safeguard === 'hold') {
    const wrap = holder(host, 'Record this change');
    await act(async () => {
      (wrap.props().onKeyDown as (event: object) => void)({ key: 'Enter', preventDefault() {} });
      await new Promise(resolve => setTimeout(resolve, 730));
    });
  } else await click(button(host, 'Record this change'));
  await settle();
}
const cases: { family: string; command: string; fields: Record<string, string | boolean>; args: object }[] = [
  { family: 'corrections', command: 'correct.roster',
    fields: { Franchise: franchise, Player: player, 'Roster change': 'remove' },
    args: { correction: { franchise, player, action: 'remove' } } },
  { family: 'lineups', command: 'commish.lineup.correct',
    fields: { Franchise: franchise, Season: '2026', Week: '1', 'Player One': true },
    args: { lineup: { franchise, season: 2026, week: 1, starters: [player] } } },
  { family: 'stats', command: 'commish.stat.queue.approve',
    fields: { 'Statistics change': item }, args: { stat: { item } } },
  { family: 'scores', command: 'commish.score.player',
    fields: { Player: player, Season: '2026', Week: '1', 'Points to add': '-1.125' },
    args: { score: { player, season: 2026, week: 1, delta: -11250 } } },
  { family: 'standings', command: 'commish.standings.record.adjust',
    fields: { Season: '2026', Franchise: franchise, Wins: '3', Losses: '2', Ties: '0' },
    args: { standings: { season: 2026, franchise, wins: 3, losses: 2, ties: 0 } } },
  { family: 'contracts', command: 'commish.contract.edit',
    fields: { Player: player, 'Contract years': '2', 'Annual salary': '120.25' },
    args: { override: { player, years: 2, salaryCents: 12025 } } },
  { family: 'cap', command: 'commish.cap.adjust',
    fields: { Franchise: franchise, Season: '2026', 'Cap charge': '-15.25' },
    args: { override: { franchise, season: 2026, amountCents: -1525 } } },
  { family: 'reserve exceptions', command: 'commish.reserve.exception',
    fields: { Player: player, 'Reserve list': 'ir' }, args: { override: { player, bucket: 'ir' } } },
  { family: 'rule exceptions', command: 'rule.exception',
    fields: { Rule: 'roster.limits', 'Applies to': JSON.stringify({ franchise }) },
    args: { correction: { rule: 'roster.limits', scope: { franchise } } } },
  { family: 'waivers', command: 'waivers.process', fields: { 'Waiver window': 'window.one' },
    args: { waiver: { window: 'window.one' } } },
  { family: 'bids', command: 'bids.award.override', fields: { Bid: 'bid.one', Franchise: franchise },
    args: { bid: { bid: 'bid.one', franchise } } },
  { family: 'trades', command: 'trade.override', fields: { Trade: 'trade.one', Decision: 'approve' },
    args: { trade: { id: 'trade.one', decision: 'approve' } } },
  { family: 'people', command: 'commish.franchise.edit',
    fields: { Franchise: franchise, 'Franchise name': 'New team', Abbreviation: 'NEW' },
    args: { people: { franchise, name: 'New team', abbreviation: 'NEW' } } },
  { family: 'calendar exception', command: 'commish.calendar.exception',
    fields: { Action: 'trade', Franchise: franchise, 'Expires at': '2026-12-01T12:00' },
    args: { exception: { action: 'trade', franchise, until: new Date('2026-12-01T12:00').toISOString() } } },
  { family: 'draft start', command: 'draft.start',
    fields: { Season: '2027', 'Starts at': '2027-07-14T12:00' },
    args: { draft: { season: 2027, at: new Date('2027-07-14T12:00').toISOString() } } },
  { family: 'owner seats', command: 'commish.seat.replace',
    fields: { Franchise: franchise, 'Owner account': 'new-owner' },
    args: { people: { franchise, principal: 'new-owner' } } },
  { family: 'owner abilities', command: 'commish.abilities.set',
    fields: { Franchise: '*', Ability: 'lineups', 'Allow this ability': true },
    args: { people: { all: true, ability: 'lineups', on: true } } },
  { family: 'succession', command: 'commish.succession.inactivity.set',
    fields: { 'In-season days': '14', 'Offseason days': '30' },
    args: { people: { inSeasonDays: 14, offseasonDays: 30 } } },
  { family: 'draft order', command: 'draft.order.set',
    fields: { Season: '2027', Round: '0', 'Place 1': franchise, 'Place 2': 'franchise.02' },
    args: { draft: { season: 2027, round: 0, order: [franchise, 'franchise.02'], running: false } } },
];
describe('Commissioner act families', () => {
  for (const c of cases) it(`${c.family}: exact preview and execute body, review and receipt`, async () => {
    const safeguard = c.family === 'people' ? 'hold' : 'reason';
    const { host, calls } = await show(c.command, [], safeguard);
    await fill(host, c.fields);
    expect(host.textContent).toContain('Before value → After value');
    expect(host.textContent).toContain('No blockers.');
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b|commish\./);
    const preview = previewBodies[previewBodies.length - 1];
    expect(preview.args).toEqual(c.args);
    expect(preview).toMatchObject({ league, command: c.command, expected: 3, effective: 'now',
      visibility: 'league', reason: 'The recorded facts were wrong' });
    expect(preview.request).toMatch(/^[a-f0-9]{32}$/);
    await fire(host, safeguard);
    expect(calls.sent).toEqual([preview]);
    expect(host.textContent).toContain('Recorded in the league ledger.');
  });
  for (const c of cases) it(`${c.family}: Go blockers prevent recording`, async () => {
    const refusal = 'The recorded facts no longer match; reopen this change.';
    const { host, calls } = await show(c.command, [refusal]);
    await fill(host, c.fields);
    expect(host.textContent).toContain(refusal);
    expect(disabled(button(host, 'Record this change'))).toBe(true);
    expect(calls.sent).toEqual([]);
  });
  it('shows a Go blocker verbatim and sends no execute', async () => {
    const refusal = 'This player has no recorded contract; choose a recorded player.';
    const { host, calls } = await show('correct.roster', [refusal]);
    await fill(host, cases[0].fields);
    expect(host.textContent).toContain(refusal);
    expect(disabled(button(host, 'Record this change'))).toBe(true);
    expect(calls.sent).toEqual([]);
  });
  it('shows an execute refusal verbatim, without a receipt or league refresh', async () => {
    const { host, app, calls } = await show('correct.roster');
    const refusal = 'choose one active recorded cut, waiver award or draft pick';
    expect(readFileSync(new URL('../../../../internal/commissioner/correction_command.go', import.meta.url),
      'utf8')).toContain(refusal);
    app.TargetCommissionerExecute = async () => { throw new Error(refusal); };
    await fill(host, cases[0].fields);
    const reads = calls.ledger.length;
    await fire(host);
    expect(host.textContent).toContain(refusal);
    expect(host.textContent).not.toContain('Recorded in the league ledger.');
    expect(calls.ledger).toHaveLength(reads);
    expect(calls.sent).toEqual([]);
  });
  it('rollover requires a dry run of this exact draft before the real execute', async () => {
    const { host, calls } = await show('season.rollover');
    await fill(host, { 'Next season': '2027' });
    expect(disabled(button(host, 'Record this change'))).toBe(true);
    await click(button(host, 'Check the next season without recording'));
    await settle();
    expect(calls.sent[0].args).toEqual({ rollover: { to: 2027, dry_run: true } });
    expect(host.textContent).toContain('Season check passed. Nothing was recorded.');
    await fire(host);
    expect(calls.sent[1].args).toEqual({ rollover: { to: 2027, dry_run: false } });
    expect(calls.sent[0].request).not.toBe(calls.sent[1].request);
  });
});

it('starting assets previews the request, confirms by keyboard and uses only the seed binding', async () => {
  const previews: object[] = [];
  const seeds: object[] = [];
  const { app, calls } = stub({ ...native,
    TargetCommissionerPreview: async body => {
      const sent = JSON.parse(body);
      previews.push(sent);
      return answer({ ...sent, expected: sent.expected }, [], 'hold');
    },
  });
  Object.assign(app, { TargetNativeSeed: async (body: object) => { seeds.push(body); return receipt(); } });
  const host = await mount(render(SeedForm, { league }), app);
  await type(control(host, 'Reason'), 'Held rosters are the agreed starting point');
  await pause();
  expect(host.textContent).toContain('Before value → After value');
  const sent = previews[previews.length - 1];
  expect(sent).toEqual({ league, expected: leagueState.peek().ledger?.revision ?? 0,
    request: expect.stringMatching(/^[a-f0-9]{32}$/), reason: 'Held rosters are the agreed starting point' });
  const wrap = holder(host, 'Record starting assets');
  await act(async () => {
    (wrap.props().onKeyDown as (e: object) => void)({ key: ' ', preventDefault() {} });
    await new Promise(resolve => setTimeout(resolve, 730));
  });
  await settle();
  expect(seeds).toEqual([sent]);
  expect(calls.sent).toEqual([]);
  expect(host.textContent).toContain('Recorded in the league ledger.');
});

const pause = async () => {
  await act(async () => { await new Promise(resolve => setTimeout(resolve, seedPreviewPause + 50)); });
  await settle();
};

it('starting assets keeps its preview while the reason is typed, asking Go again after a pause', async () => {
  const previews: { reason: string }[] = [];
  const { app } = stub({ TargetCommissionerPreview: async body => {
    const sent = JSON.parse(body);
    previews.push(sent);
    return answer(sent, sent.reason ? [] : ['Give a reason for this change'], 'hold');
  } });
  Object.assign(app, { TargetNativeSeed: async () => receipt() });
  const host = await mount(render(SeedForm, { league }), app);
  await settle();
  expect(previews.map(p => p.reason)).toEqual(['']);
  for (const reason of ['H', 'He', 'Held']) await type(control(host, 'Reason'), reason);
  expect(host.textContent).toContain('Before value → After value');
  expect(host.textContent).not.toContain('Reading the held starting assets preview');
  expect(host.textContent).toContain('Checking the change again');
  expect(disabled(button(host, 'Record starting assets'))).toBe(true);
  await pause();
  expect(previews.map(p => p.reason)).toEqual(['', 'Held']);
  expect(disabled(button(host, 'Record starting assets'))).toBe(false);
});

it('every table field is a field on its Go Raw argument, including nested pick and scope', () => {
  const root = new URL('../../../../internal/commissioner/', import.meta.url);
  const structs: Record<string, [string, string]> = {
    people: ['people.go', 'RawPeople'], exception: ['calendar.go', 'RawWindowException'],
    rollover: ['rollover_command.go', 'RawRolloverArgs'],
    correction: ['correction_command.go', 'RawCorrection'],
    lineup: ['lineup_commands.go', 'RawLineupChange'], score: ['score_commands.go', 'RawScoreChange'],
    stat: ['stat_commands.go', 'RawStatChange'], standings: ['standings_commands.go', 'RawStandingsChange'],
    override: ['native_override.go', 'RawNativeOverride'], waiver: ['market_wire.go', 'RawWaiverArgs'],
    bid: ['market_wire.go', 'RawBidArgs'], trade: ['market_wire.go', 'RawTradeArgs'],
    schedule: ['schedule.go', 'RawScheduleArgs'], bracket: ['bracket_commands.go', 'RawBracketChange'],
    draft: ['draft_commands.go', 'RawDraftArgs'], '': ['command.go', 'RawArgs'],
  };
  for (const def of actTable) {
    const [file, name] = structs[def.root];
    const source = readFileSync(new URL(file, root), 'utf8');
    const struct = source.split(`type ${name} struct {`)[1]?.split('\n}')[0];
    expect(struct, `${def.command}: Go struct`).toBeDefined();
    for (const field of def.fields) {
      const key = def.command === 'schedule.edit' && ['game', 'home', 'away'].includes(field.path)
        ? 'games' : field.path.split('.')[0];
      expect(struct, `${def.command}: ${field.path}`).toMatch(new RegExp(`json:"${key}(?:,|")`));
    }
  }
});

it('keeps large cents exact, and refuses fractional cents instead of silently rounding', async () => {
  const { host } = await show('commish.contract.edit');
  await fill(host, { Player: player, 'Contract years': '2', 'Annual salary': '70368744177664.01' });
  expect(previewBodies[previewBodies.length - 1].args)
    .toEqual({ override: { player, years: 2, salaryCents: 7036874417766401 } });
  await type(control(host, 'Annual salary'), '1.005');
  await settle();
  expect(disabled(button(host, 'Record this change'))).toBe(true);
});

it('the seed preview blocker is word for word and prevents any starting assets write', async () => {
  const refusal = 'Hold the league rosters and draft picks before recording starting assets.';
  const seeds: object[] = [];
  const { app, calls } = stub({ TargetCommissionerPreview: async body =>
    answer(JSON.parse(body), [refusal], 'reason') });
  Object.assign(app, { TargetNativeSeed: async (body: object) => { seeds.push(body); return receipt(); } });
  const host = await mount(render(SeedForm, { league }), app);
  await type(control(host, 'Reason'), 'Starting the season');
  await settle();
  expect(host.textContent).toContain(refusal);
  expect(disabled(button(host, 'Record starting assets'))).toBe(true);
  expect(seeds).toEqual([]);
  expect(calls.sent).toEqual([]);
});

it('a seed preview refused before Go named the league shows the refusal, not a mismatch', async () => {
  const refusal = 'Hold the league rosters and settings before seeding.';
  const { app } = stub({ TargetCommissionerPreview: async body => ({
    ...answer(JSON.parse(body), [refusal], 'reason'), target: { league: '', name: '', version: 0 } }) });
  const host = await mount(render(SeedForm, { league }), app);
  await settle();
  expect(host.textContent).toContain(refusal);
  expect(host.textContent).not.toContain('does not match this league version');
  expect(disabled(button(host, 'Record starting assets'))).toBe(true);
});

it('changing a reviewed draft cancels its approval and changes its request key', async () => {
  const { host, app, calls } = await show('correct.roster');
  await fill(host, cases[0].fields);
  const old = previewBodies[previewBodies.length - 1];
  let release: (value: object) => void = () => {};
  app.TargetCommissionerPreview = body => new Promise(resolve => {
    const sent = JSON.parse(body);
    previewBodies.push(sent);
    release = resolve;
  });
  await type(control(host, 'Player'), '67890');
  expect(disabled(button(host, 'Record this change'))).toBe(true);
  expect(calls.sent).toEqual([]);
  const current = previewBodies[previewBodies.length - 1];
  expect(current.request).not.toBe(old.request);
  await act(async () => release(answer(current)));
  await settle();
  await fire(host);
  expect(calls.sent).toEqual([current]);
});

it('an act keeps its last preview on screen while the next one is read', async () => {
  const { host, app, calls } = await show('correct.roster');
  await fill(host, cases[0].fields);
  let release: (value: object) => void = () => {};
  app.TargetCommissionerPreview = body => new Promise(resolve => {
    previewBodies.push(JSON.parse(body));
    release = resolve;
  });
  await type(control(host, 'Reason'), 'The recorded facts were wrong twice');
  expect(host.textContent).toContain('Before value → After value');
  expect(host.textContent).toContain('Checking the change again');
  expect(disabled(button(host, 'Record this change'))).toBe(true);
  const current = previewBodies[previewBodies.length - 1];
  await act(async () => release(answer(current)));
  await settle();
  expect(host.textContent).not.toContain('Checking the change again');
  await fire(host);
  expect(calls.sent).toEqual([current]);
});

it('a form opened after a receipt does not show the earlier receipt', async () => {
  const { host, app } = await show('correct.roster');
  await fill(host, cases[0].fields);
  await fire(host);
  expect(host.textContent).toContain('Recorded in the league ledger.');
  await unmount();
  const again = await mount(
    render(CommissionerActForm, { definition: definition('correct.roster'), league }),
    app,
  );
  await settle();
  expect(again.textContent).not.toContain('Recorded in the league ledger.');
});

it('a mismatched preview version cannot confirm a frozen target', async () => {
  const { host, app, calls } = await show('correct.roster');
  app.TargetCommissionerPreview = async body => {
    const sent = JSON.parse(body);
    return answer({ ...sent, expected: sent.expected + 1 });
  };
  await fill(host, cases[0].fields);
  expect(host.textContent).toContain('The preview does not match this league version. Reopen the change.');
  expect(disabled(button(host, 'Record this change'))).toBe(true);
  expect(calls.sent).toEqual([]);
});

it('sends invitations and ballots private and every other act to the league, as Go requires', () => {
  // Go refuses the wrong visibility: governance events whose Private() is true,
  // and the private column of succession.go.
  const privateActs = ['commish.seat.invite', 'commish.seat.invite.revoke', 'commish.seat.invite.redeem',
    'commish.vote.cast'];
  const held = { league, revision: 3, principal: 'creator' } as Parameters<typeof actChange>[2];
  const sample = (kind: string): string | boolean | string[] => {
    if (kind === 'boolean') return true;
    if (kind === 'order' || kind === 'starters') return ['a'];
    if (['pick', 'scope', 'shape', 'scheduleGame'].includes(kind)) return '{}';
    if (kind === 'instant') return '2026-10-18T12:00';
    return ['number', 'money', 'decimal', 'bracketSlot'].includes(kind) ? '1' : 'x';
  };
  for (const def of actTable) {
    const values = Object.fromEntries(def.fields.map(field => [field.path, sample(field.kind)]));
    expect(actChange(def, values, held, 'reason')?.visibility, def.command)
      .toBe(privateActs.includes(def.command) ? 'private' : 'league');
  }
});
