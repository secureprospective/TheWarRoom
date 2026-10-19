import { act } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, mount, render, settle, unmount } from './commissionerDom';
import { active, dialogOf, modelFocusApis } from './trayHarness';
import { stub } from './commissionerStub';
import { leagueState } from './proxy';
import { KeyInputElement } from '../shell/mflKeyTestDom';
import { commands } from '../commands/registry';
import { actsFor, liveDraftControls } from './actTable';
import { entriesOf, parseControls } from './controls';
import { surfaceActions } from './surfaceActions';
import LeagueLanding from './LeagueLanding';
import { JobsStrip } from './JobsStrip';
import { parseJobs } from './jobs';
import controlsWire from './controls.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';

let restore: () => void;
beforeEach(async () => {
  restore = modelFocusApis();
  Object.defineProperty(KeyInputElement.prototype, 'tabIndex', { configurable: true,
    get() { return -1; }, set() {} });
  await Promise.all([import('./CommissionerActTray'), import('./RuleTray'), import('./ActionTray'),
    import('./SeedTray'), import('./LeagueRules'), import('./LeaguePeople')]);
});
afterEach(async () => { await unmount(); restore(); vi.restoreAllMocks(); });
const inScope = (id: string) => actsFor(id).length > 0 || liveDraftControls.has(id) ||
  ['CM-021', 'CM-040'].includes(id);
const entries = entriesOf(parseControls(controlsWire)).filter(e => inScope(e.id));
async function landing(future: boolean) {
  const controls = { groups: controlsWire.groups.map(g => ({ ...g,
    controls: g.controls.map(e => ({ ...e, built: e.built || (future && inScope(e.id)) })) })) };
  const { app } = stub({ TargetCommissionerControls: async () => controls,
    TargetCommissionerJobs: async () => jobsWire.fresh });
  await mount(render(LeagueLanding), app);
  for (const g of leagueState.peek().controls?.groups ?? [])
    for (const e of g.controls) e.built = controls.groups.flatMap(g => g.controls)
      .find(c => c.id === e.id)?.built ?? false;
  await leagueState.choose();
  await unmount();
  const host = await mount(render(LeagueLanding), app);
  active(host);
  await settle();
  return host;
}
for (const future of [false, true]) describe(future ? 'Ready for Go built flags' : 'Go controls wire', () => {
  for (const entry of entries.filter(e => future || e.built))
    it(`${entry.name} opens its tray or the existing live Draft room`, async () => {
      const nav = vi.spyOn(commands, 'dispatch');
      const host = await landing(future);
      await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: entry.group }));
      await settle();
      button(host, `Open ${entry.name}`).focus();
      await click(button(host, `Open ${entry.name}`));
      await settle();
      if (liveDraftControls.has(entry.id)) {
        expect(nav).toHaveBeenCalledWith('nav.open', { node: 'trade', workspace: 'draft-room' });
      } else {
        expect(dialogOf(host)).toBeDefined();
        const tray = dialogOf(host)!;
        if (actsFor(entry.id).length) {
          expect(all(tray, 'option').map(o => o.textContent)).toContain(actsFor(entry.id)[0].label);
        } else expect(tray.textContent).toContain(entry.id === 'CM-021' ? 'Give or take a role' : 'Rule');
      }
      expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
    });
});
it('unbuilt action rows remain disabled and explain why', async () => {
  const host = await landing(false);
  await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: 'season' }));
  await settle();
  for (const entry of entries.filter(e => e.group === 'season' && !e.built)) {
    expect(button(host, `Open ${entry.name}`).attributes.has('disabled')).toBe(true);
  }
  expect(host.textContent).toContain('This action is not built yet.');
});
it('seed-blocked jobs open starting assets, not a rule or an unrelated control', async () => {
  const calls = vi.spyOn(surfaceActions, 'run');
  const jobs = parseJobs(jobsWire.fresh);
  const host = await mount(render(JobsStrip, { jobs }));
  const blocked = jobs.jobs.filter(j => j.words === 'Record starting assets first.');
  expect(blocked.length).toBeGreaterThan(0);
  for (const job of blocked) {
    const b = all(host, 'button').find(b => b.textContent.startsWith(job.name))!;
    await click(b);
    expect(calls).toHaveBeenLastCalledWith({ screen: 'landing', action: 'tray', value: 'seed' });
  }
});
it('live multi-control jobs prefer the matching act over their rule settings', async () => {
  const calls = vi.spyOn(surfaceActions, 'run');
  const jobs = parseJobs(jobsWire.fresh);
  const job = { ...jobs.jobs.find(j => j.id === 'dot_review')!,
    state: 'live' as const, words: 'Review trades' };
  const host = await mount(render(JobsStrip, { jobs: { ...jobs, jobs: [job] } }));
  await click(all(host, 'button')[0]);
  expect(calls).toHaveBeenLastCalledWith({ screen: 'landing', action: 'open', value: 'CM-037' });
});
