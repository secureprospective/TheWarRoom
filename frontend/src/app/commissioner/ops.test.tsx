import { act } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, control, check, mount, render, settle, type, unmount, type Element }
  from './commissionerDom';
import { stub } from './commissionerStub';
import LeagueLanding from './LeagueLanding';
import LeagueOps from './LeagueOps';
import { leagueState } from './proxy';
import { surfaceActions } from './surfaceActions';
import { active, dialogOf, modelFocusApis } from './trayHarness';
import controlsWire from './controls.wire.testdata.json';
import digestsWire from './digests.wire.testdata.json';
import jobsWire from './jobs.wire.testdata.json';
import ledgerWire from './wire.testdata.json';
import noticesWire from './notices.wire.testdata.json';
import peopleWire from './people.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';
import { parseDigests, parseNotes } from './notes';

let restoreFocus: () => void;
beforeEach(() => { leagueState.forget(); restoreFocus = modelFocusApis(); });
afterEach(async () => { await unmount(); vi.restoreAllMocks(); restoreFocus(); });

const league = rulesWire.league;
const ops = async (over: Parameters<typeof stub>[0] = {}) => {
  const { app, calls } = stub(over);
  const host = await mount(render(LeagueOps), app);
  await leagueState.choose(league);
  await settle();
  return { host, calls };
};
const sections = (host: Element) => all(host, 'section').map(s => s.getAttribute('aria-label'));
const section = (host: Element, label: string) =>
  all(host, 'section').find(s => s.getAttribute('aria-label') === label)!;

describe('League notices, What’s New, the mail log and the digest', () => {
  it('reads the Go-written wire strictly', () => {
    expect(parseNotes(noticesWire.commissioner).notices).toHaveLength(2);
    expect(parseDigests(digestsWire.creator).digests).toHaveLength(4);
    expect(() => parseNotes({ ...noticesWire.commissioner, extra: 1 })).toThrow('unexpected field');
    expect(() => parseNotes({ ...noticesWire.commissioner,
      notices: [{ ...noticesWire.commissioner.notices[0], kind: 'newsletter' }] })).toThrow();
    expect(() => parseDigests({ league: 'x', digests: [{ id: 'a', sentence: 's' }] })).toThrow();
  });
  it('shows the pinned notice first with a glyph that says who, when and which change', async () => {
    const { host } = await ops();
    const notices = section(host, 'League notices');
    const items = all(notices, 'li');
    expect(items.map(li => li.textContent)).toEqual([
      expect.stringContaining('Pinned Rookie draft is Saturday'),
      expect.stringContaining('Waivers run Tuesday'),
    ]);
    expect(items[0].getAttribute('data-pinned')).toBe('true');
    expect(items[1].getAttribute('data-pinned')).toBe('false');
    await click(button(items[0], 'Who set this note, when, and with which change'));
    expect(items[0].textContent)
      .toContain('Posted an official notice, pinned: “Rookie draft is Saturday”');
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
  });
  it('shows What’s New: league notes and the app’s release notes, which no one posts', async () => {
    const { host } = await ops({ TargetCommissionerNotices: async () => ({ ...noticesWire.commissioner,
      notes: [] }) });
    const news = section(host, 'What’s New').textContent;
    expect(news).toContain('No league notes have been posted.');
    expect(news).toContain('What changed in the app');
    for (const release of noticesWire.commissioner.releases) expect(news).toContain(release.title);
  });
  it('keeps a league note apart from the official notices', async () => {
    const { host } = await ops();
    expect(section(host, 'League notices').textContent).not.toContain('House rule: no trades in week 18');
    expect(section(host, 'What’s New').textContent).toContain('House rule: no trades in week 18');
  });
  it('says in plain words the app sends no mail, and lists every notice it recorded', async () => {
    const { host } = await ops({ TargetCommissionerNotices: async () => noticesWire.withdrawn });
    const log = section(host, 'Mail log').textContent;
    expect(log).toContain(noticesWire.withdrawn.mail.words);
    expect(noticesWire.withdrawn.mail.words).toContain('sends no email and no messages');
    expect(log).toContain('Rookie draft is Saturday');
    expect(log).toContain('Withdrawn');
    expect(log).toContain('Shown to owners');
    expect(section(host, 'League notices').textContent).not.toContain('Rookie draft is Saturday');
  });
  it('shows one sentence per digest with its time in the viewer’s own zone', async () => {
    const resolved = new Intl.DateTimeFormat().resolvedOptions();
    vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions')
      .mockReturnValue({ ...resolved, timeZone: 'America/Los_Angeles' });
    const { host } = await ops();
    const digest = section(host, 'Digest of changes');
    const lines = all(digest, 'li').map(li => li.textContent);
    expect(lines).toHaveLength(digestsWire.creator.digests.length);
    expect(lines[0]).toContain(digestsWire.creator.digests[0].sentence);
    expect(lines[0]).toContain('6:00 AM PDT');
    expect(lines[0]).not.toContain('UTC');
  });
  it('offers the post launchers only to a person Go lets post', async () => {
    const owner = await ops({ TargetCommissionerPeople: async () => peopleWire.owner });
    expect(all(owner.host, 'button').map(b => b.getAttribute('aria-label')))
      .not.toContain('Post an official notice');
    expect(owner.host.textContent).toContain('Rookie draft is Saturday');
    await unmount();
    const commissioner = await ops();
    expect(button(commissioner.host, 'Post an official notice')).toBeDefined();
    expect(button(commissioner.host, 'Post a league note')).toBeDefined();
  });
  it('offers Withdraw only where Go lets this person undo, and opens the undo for that entry', async () => {
    const [first, second] = noticesWire.commissioner.notices;
    const ledger = { ...ledgerWire, entries: [{ ...ledgerWire.entries[0], id: first.id, undo: 'league' },
      { ...ledgerWire.entries[0], id: second.id, undo: '' }] };
    const opened: string[] = [];
    const stop = surfaceActions.register('landing', args => { opened.push(`${args.action} ${args.value}`); });
    const { host } = await ops({ TargetCommissionerLedger: async () => ledger });
    const withdraw = all(host, 'button').filter(b => b.getAttribute('aria-label')?.startsWith('Withdraw'));
    expect(withdraw.map(b => b.getAttribute('aria-label'))).toEqual([`Withdraw: ${first.text}`]);
    await click(withdraw[0]);
    stop();
    expect(opened).toEqual([`tray undo ${first.id}`]);
  });
  it('says what it could not read, and still shows the rest of the league', async () => {
    const { host } = await ops({
      TargetCommissionerNotices: async () => { throw new Error('ledger busy'); } });
    expect(host.textContent).toContain('could not read the league notices (ledger busy)');
    expect(leagueState.peek().rules).toBeDefined();
    expect(section(host, 'Digest of changes')).toBeDefined();
  });
  it('answers the owner rows it replaces: commissioner news and the mail log', async () => {
    const { host } = await ops();
    expect(section(host, 'League notices').getAttribute('data-endpoint')).toBe('M-095');
    expect(section(host, 'Mail log').getAttribute('data-endpoint')).toBe('M-093');
  });
  it('lists only these four sections', async () => {
    const { host } = await ops();
    expect(sections(host).filter(Boolean)).toEqual(['League operations', 'League notices', 'What’s New',
      'Mail log', 'Digest of changes']);
  });
});

const open = async (host: Element, launcher: string) => {
  const opener = button(host, launcher);
  opener.focus();
  await click(opener);
  await settle();
  return dialogOf(host)!;
};
const landing = async (over: Parameters<typeof stub>[0] = {}) => {
  await Promise.all([import('./LeagueOps'), import('./ActionTray')]);
  const { app, calls } = stub({ TargetCommissionerControls: async () => controlsWire,
    TargetCommissionerJobs: async () => jobsWire.fresh, ...over });
  const host = await mount(render(LeagueLanding), app);
  active(host);
  await settle();
  await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: 'ops' }));
  await settle();
  return { host, calls };
};

describe('Posting a notice or a league note', () => {
  it.each([
    { name: 'an official notice', launcher: 'Post an official notice', field: 'Notice',
      command: 'commish.notice.publish', confirm: 'Post this notice' },
    { name: 'a league note', launcher: 'Post a league note', field: 'League note',
      command: 'commish.whatsnew.post', confirm: 'Post this note' },
  ])('posts $name through the one tray, as written and pinned', async test => {
    const { host, calls } = await landing();
    expect(all(host, 'textarea')).toHaveLength(0);
    const panel = await open(host, test.launcher);
    expect(button(panel, test.confirm).attributes.has('disabled')).toBe(true);
    expect(panel.textContent).toContain('Write what owners should read.');
    await type(control(panel, test.field), '  Draft moved to Sunday  ');
    await check(control(panel, 'Pin to the top'), true);
    await settle();
    expect(panel.textContent).toContain('Not posted');
    expect(panel.textContent).toContain('The app sends no email and no messages.');
    expect(panel.textContent).not.toMatch(/CM-\d|G[123]\b/);
    await click(button(panel, test.confirm));
    await settle();
    expect(calls.sent).toHaveLength(1);
    expect(calls.sent[0]).toMatchObject({ command: test.command, league: peopleWire.commissioner.league,
      expected: peopleWire.commissioner.revision, effective: 'now',
      visibility: 'league', args: { note: { text: 'Draft moved to Sunday', pin: true } } });
  });
  it('sends the previewed words, so what Go described is what is posted', async () => {
    const previewed: string[] = [];
    const { app } = stub({ TargetCommissionerPreview: async body => {
      previewed.push(body);
      return stub().app.TargetCommissionerPreview!(body);
    } });
    const host = await mount(render(LeagueLanding), { ...app,
      TargetCommissionerControls: async () => controlsWire,
      TargetCommissionerJobs: async () => jobsWire.fresh });
    active(host);
    await settle();
    await act(async () => surfaceActions.run({ screen: 'landing', action: 'group', value: 'ops' }));
    await settle();
    await type(control(await open(host, 'Post an official notice'), 'Notice'), 'Waivers run Tuesday');
    await settle();
    expect(JSON.parse(previewed.at(-1)!)).toMatchObject({ command: 'commish.notice.publish',
      args: { note: { text: 'Waivers run Tuesday' } } });
    expect(JSON.parse(previewed.at(-1)!).args.note.pin).toBe(false);
  });
  it('shows Go’s refusal in its own words and keeps what was written', async () => {
    const { host } = await landing({ TargetCommissionerExecute: async () => {
      throw new Error('Keep the notice to 1000 characters');
    } });
    const panel = await open(host, 'Post an official notice');
    await type(control(panel, 'Notice'), 'x');
    await settle();
    await click(button(panel, 'Post this notice'));
    await settle();
    expect(panel.textContent).toContain('Keep the notice to 1000 characters');
    expect(control(panel, 'Notice').props().value).toBe('x');
  });
  it('opens nothing for a person Go does not let post', async () => {
    const { host } = await landing({ TargetCommissionerPeople: async () => peopleWire.owner });
    await act(async () => surfaceActions.run({ screen: 'landing', action: 'tray', value: 'notice' }));
    await settle();
    expect(dialogOf(host)).toBeUndefined();
  });
});

describe('The ledger and the league page keep up with notices', () => {
  it('finds posted notices by their own kind of change, and Go does the finding', async () => {
    const sent: { league: string; filter: unknown }[] = [];
    const { LedgerFilters } = await import('./LedgerFilters');
    const { parseLedger } = await import('./proxy');
    const ledger = parseLedger((await import('./wire.testdata.json')).default);
    const found = await mount(render(LedgerFilters, { reading: ledger, children: () => null }), {
      TargetCommissionerLedgerFiltered: async (l, body) => {
        sent.push({ league: l, filter: JSON.parse(body) });
        return (await import('./wire.testdata.json')).default;
      },
    });
    await type(control(found, 'Kind of change'), 'notices');
    await click(button(found, 'Find changes'));
    await settle();
    expect(sent).toEqual([{ league: ledger.league,
      filter: { person: '', kind: 'notices', from: '', before: '' } }]);
  });
  it('reads the notices and the digest again after a change is recorded', async () => {
    const reads = { notices: 0, digests: 0 };
    const { app } = stub({
      TargetCommissionerNotices: async () => { reads.notices++; return noticesWire.commissioner; },
      TargetCommissionerDigests: async () => { reads.digests++; return digestsWire.creator; },
    });
    await mount(render(LeagueOps), app);
    await leagueState.choose(league);
    await settle();
    expect(reads).toEqual({ notices: 1, digests: 1 });
    await leagueState.execute(JSON.stringify({ league, command: 'commish.notice.publish', args: {
      note: { text: 'Hi', pin: false } }, reason: '', expected: 4, request: 'r1', effective: 'now',
    visibility: 'league' }));
    await settle();
    expect(reads).toEqual({ notices: 2, digests: 2 });
  });
  it('marks the league operations rows available in the control list, with what they do not do', async () => {
    const { host } = await landing();
    const rows = all(host, 'li').filter(li => li.getAttribute('class') === 'control-row');
    const named = (name: string) => rows.find(li => li.textContent.startsWith(name))!;
    for (const name of ['Notifications', 'Official notices', "What's New and league notes",
      'Data refresh and cache'])
      expect(named(name).getAttribute('data-built'), name).toBe('true');
    expect(named('Engine parameters').getAttribute('data-built')).toBe('false');
    expect(named('Notifications').textContent).toContain('sends no email and no messages');
    expect(named('Notifications').textContent).toContain('Tell the league about transactions');
    expect(named('Notifications').textContent).not.toMatch(/CM-\d|G[123]\b/);
  });
});
