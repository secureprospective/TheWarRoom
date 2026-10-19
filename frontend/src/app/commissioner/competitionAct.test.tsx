import { afterEach, expect, it } from 'vitest';
import { act } from 'react';
import { CommissionerActForm } from './CommissionerActTray';
import { actTable, actsFor } from './actTable';
import { all, button, click, control, disabled, holder, mount, render, settle, type, unmount }
  from './commissionerDom';
import { stub, type Sent } from './commissionerStub';
import { leagueState } from './proxy';
import wire from './competition.wire.testdata.json';

afterEach(async () => { await unmount(); leagueState.forget(); });
for (const entry of wire) {
  const raw = entry.request;
  it(`${raw.command}: sends its exact Raw body and presents the Go-authored preview`, async () => {
    const previews: Sent[] = [];
    const { app, calls } = stub({
      ...{
        TargetNativeAssets: async () => ({ league: raw.league, revision: raw.expected }),
        TargetNativeDirectory: async () => entry.directory,
        TargetNativeBracket: async () => entry.bracket,
        TargetCommissionerSchedule: async () => entry.schedule,
      },
      TargetCommissionerPreview: async body => {
        previews.push(JSON.parse(body) as Sent);
        return entry.preview;
      },
    });
    const definition = actTable.find(d => d.command === raw.command);
    expect(definition).toBeDefined();
    const host = await mount(render(CommissionerActForm,
      { league: raw.league, definition: definition! }), app);
    await settle();
    const bracket = raw.args.bracket;
    const schedule = raw.args.schedule;
    if (bracket) {
      await type(control(host, 'Season'), String(bracket.season));
      await settle();
      if (bracket.preset) await type(control(host, 'Bracket layout'), bracket.preset);
      if (bracket.name) {
        await type(control(host, 'Layout name'), bracket.name);
        await type(control(host, 'Layout to save'), JSON.stringify(bracket.shape));
      }
      if (bracket.node) await type(control(host, 'Round and place'), bracket.node);
      if (bracket.slot) await type(control(host, 'Playoff seed'), String(bracket.slot));
      if (bracket.game) await type(control(host, 'Playoff game'), bracket.game);
      if (bracket.team) await type(control(host, 'Team'), bracket.team);
    }
    if (schedule?.weeks) {
      await type(control(host, 'Regular-season weeks'), String(schedule.weeks));
      await type(control(host, 'Matchup pattern'), schedule.pattern!);
    }
    if (schedule?.games) {
      const game = schedule.games[0];
      const value = JSON.stringify({ week: schedule.week, id: game.id, published: false });
      expect(all(host, 'option').some(o => o.attributes.get('value') === value)).toBe(true);
      await type(control(host, 'Game and week'), value);
      await type(control(host, 'Home team'), game.home);
      await type(control(host, 'Away team'), game.away);
    }
    await type(control(host, 'Reason'), raw.reason!);
    await settle();
    const args = bracket ? { bracket: { ...bracket,
      ...(raw.command !== 'bracket.advance' ? { started: false } : {}) } }
      : schedule?.games ? { schedule: { ...schedule, published: false } } : raw.args;
    expect(previews.at(-1)).toMatchObject({ command: raw.command, league: raw.league,
      expected: raw.expected, effective: raw.effective, visibility: raw.visibility, args });
    expect(previews.at(-1)?.args).toEqual(args);
    for (const field of entry.preview.fields) {
      expect(host.textContent).toContain(field.before);
      expect(host.textContent).toContain(field.after);
    }
    expect(host.textContent).not.toContain('League version:');
    expect(disabled(button(host, 'Record this change'))).toBe(false);
    if (entry.preview.safeguard === 'hold') {
      await act(async () => {
        const wrap = holder(host, 'Record this change');
        (wrap.props().onKeyDown as (event: object) => void)({ key: 'Enter', preventDefault() {} });
        await new Promise(resolve => setTimeout(resolve, 730));
      });
    } else await click(button(host, 'Record this change'));
    await settle();
    expect(calls.sent).toEqual([previews.at(-1)]);
  });
}

it('offers every built schedule and bracket act under its control', () => {
  expect(actsFor('CM-015').map(d => d.command)).toEqual(
    ['schedule.generate', 'schedule.edit', 'schedule.publish']);
  expect(actsFor('CM-046').map(d => d.command)).toEqual(
    ['bracket.build', 'bracket.preset.save', 'bracket.edit', 'bracket.seed']);
  expect(actsFor('CM-047').map(d => d.command)).toEqual(['bracket.advance']);
});

it('names the held games, bracket places, seeds and teams instead of asking for ids', async () => {
  const entry = wire.find(e => e.request.command === 'bracket.edit')!;
  const raw = entry.request;
  const { app } = stub({ ...{
    TargetNativeAssets: async () => ({ league: raw.league, revision: raw.expected }),
    TargetNativeDirectory: async () => entry.directory,
    TargetNativeBracket: async () => entry.bracket,
  } });
  const definition = actTable.find(d => d.command === raw.command)!;
  const host = await mount(render(CommissionerActForm, { league: raw.league, definition }), app);
  await settle();
  expect(control(host, 'Round and place').tagName).toBe('SELECT');
  expect(control(host, 'Team').tagName).toBe('SELECT');
  const round = entry.bracket!.rounds[0];
  expect(host.textContent).toContain(round.name);
  expect(host.textContent).toContain(`seed ${round.games[0].home!.seed}`);
  expect(host.textContent).toContain(round.games[0].home!.name);
});

it('distinguishes published games from the draft and sends the publication flag', async () => {
  const entry = wire.find(e => e.request.command === 'schedule.edit')!;
  const raw = entry.request;
  const previews: Sent[] = [];
  const { app } = stub({ ...{
    TargetNativeAssets: async () => ({ league: raw.league, revision: raw.expected }),
    TargetNativeDirectory: async () => entry.directory,
    TargetCommissionerSchedule: async () => ({ ...entry.schedule,
      published: entry.schedule!.draft }),
  }, TargetCommissionerPreview: async body => {
    previews.push(JSON.parse(body) as Sent);
    return entry.preview;
  } });
  const definition = actTable.find(d => d.command === raw.command)!;
  const host = await mount(render(CommissionerActForm, { league: raw.league, definition }), app);
  await settle();
  const game = raw.args.schedule!.games![0];
  const value = JSON.stringify({ week: 1, id: game.id, published: true });
  expect(host.textContent).toContain('Published week 1:');
  expect(host.textContent).toContain('Draft week 1:');
  expect(control(host, 'Game and week').tagName).toBe('SELECT');
  await type(control(host, 'Game and week'), value);
  await type(control(host, 'Home team'), game.home);
  await type(control(host, 'Away team'), game.away);
  await type(control(host, 'Reason'), 'Correct the published pairing');
  await settle();
  expect(previews.at(-1)?.args).toEqual({ schedule: { week: 1, games: [game], published: true } });
});
