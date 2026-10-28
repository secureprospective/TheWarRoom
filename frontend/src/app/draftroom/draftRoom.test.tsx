import { act } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, control, disabled, flush, holder, settle, type, unmount, type Element }
  from '../commissioner/commissionerDom';
import { leagueState } from '../commissioner/proxy';
import { surfaceActions } from '../commissioner/surfaceActions';
import { dialogOf } from '../commissioner/trayHarness';
import { at, openRoom, people, wire } from './draftHarness';
import { draftRoom } from './state';

let restore: (() => void) | undefined;
afterEach(async () => {
  await unmount();
  restore?.();
  vi.useRealTimers();
  vi.restoreAllMocks();
});
const room = async (over: Parameters<typeof openRoom>[0] = {}) => {
  const opened = await openRoom(over);
  restore = opened.restore;
  return opened;
};
const labels = (host: Element) => all(host, 'button').map(b => b.getAttribute('aria-label'));
const timer = (host: Element) => all(host, 'span').find(s => s.getAttribute('role') === 'timer')!;
const noon = '2027-07-14T17:30:00Z';

describe('The clock and its countdown', () => {
  it('says whose turn it is, what is left and when the pick closes, in the viewer’s own zone', async () => {
    const zone = (timeZone: string) => {
      const resolved = new Intl.DateTimeFormat().resolvedOptions();
      vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({ ...resolved, timeZone });
    };
    at(noon);
    zone('America/Los_Angeles');
    const { host } = await room();
    const text = host.textContent;
    expect(text).toContain('On the clock: Held team 1, round 1, pick 2 (originally Held team 2’s)');
    expect(timer(host).textContent).toBe('5 hours 30 minutes');
    const local = new Date(wire.running.draft.onClock.deadline).toLocaleString(undefined, {
      weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
      timeZoneName: 'short', timeZone: 'America/Los_Angeles' });
    expect(text).toContain(`The pick closes ${local}`);
    expect(local).toContain('PDT');
    expect(text).not.toContain('UTC');
    expect(text).toContain('This pick signs for $3,000,000 for 5 years.');
    await unmount();
    zone('Asia/Tokyo');
    const tokyo = await room();
    expect(tokyo.host.textContent).toContain('Jul 15');
    expect(tokyo.host.textContent).toContain('8:00 AM');
  });
  it('counts down every second and looks at the draft again the moment the clock runs out', async () => {
    vi.useFakeTimers();
    at('2027-07-14T22:59:57Z');
    const opened = await openRoom({ fakeTimers: true });
    restore = opened.restore;
    await flush();
    expect(timer(opened.host).textContent).toBe('3 seconds');
    const looked = opened.reads.length;
    at('2027-07-14T22:59:58Z');
    await act(async () => { vi.advanceTimersByTime(1000); });
    expect(timer(opened.host).textContent).toBe('2 seconds');
    expect(opened.reads).toHaveLength(looked);
    at('2027-07-14T23:00:00Z');
    await act(async () => { vi.advanceTimersByTime(1000); });
    await flush();
    expect(timer(opened.host).textContent).toBe('Time is up');
    expect(opened.reads).toHaveLength(looked + 1);
    await act(async () => { vi.advanceTimersByTime(1000); });
    expect(opened.reads).toHaveLength(looked + 1);
  });
  it('counts down to the start and looks at the draft again when the start arrives', async () => {
    vi.useFakeTimers();
    at('2027-07-14T22:59:58Z');
    const notStarted = { ...wire.running, draft: { ...wire.running.draft, started: false, onClock: null,
      start: '2027-07-14T23:00:00Z' } };
    const opened = await openRoom({ reading: notStarted, fakeTimers: true });
    restore = opened.restore;
    await flush();
    expect(opened.host.textContent).toContain('Starts in 2 seconds');
    const looked = opened.reads.length;
    at('2027-07-14T23:00:00Z');
    await act(async () => { vi.advanceTimersByTime(2000); });
    await flush();
    expect(opened.reads).toHaveLength(looked + 1);
  });
  it('shows a paused clock with the time it holds, and counts nothing down', async () => {
    at(noon);
    const { host } = await room({ reading: wire.paused });
    expect(timer(host).textContent).toBe('6 hours');
    expect(host.textContent).toContain('left, and the clock is paused');
    expect(host.textContent).not.toContain('The pick closes');
    expect(labels(host)).toContain('Resume the clock');
    expect(labels(host)).not.toContain('Pause the clock');
  });
  it('says why a draft cannot be ordered yet, in Go’s words, and shows no clock to count', async () => {
    const { host } = await room({ reading: wire.waiting });
    expect(host.textContent).toContain('The draft order needs the final standings of season 2026');
    expect(all(host, 'span').find(s => s.getAttribute('role') === 'timer')).toBeUndefined();
    expect(host.textContent).not.toContain('Starts in');
    expect(host.textContent).not.toContain('0001');
  });
});

describe('The board', () => {
  it('lists each pick with its owner, the rookie named, what became of it and its contract', async () => {
    at(noon);
    const { host } = await room();
    const board = all(host, 'table').find(t => t.getAttribute('aria-label') === 'Round 1')!;
    const rows = all(board, 'tr').map(r => r.textContent);
    expect(rows[1]).toContain('Held team 3');
    expect(rows[1]).toContain('Player, 5001');
    expect(rows[1]).toContain('Picked by the franchise');
    expect(rows[1]).toContain('$3,000,000 for 5 years');
    expect(rows[2]).toContain('Held team 1 (from Held team 2)');
    expect(rows[2]).toContain('On the clock');
    expect(rows[3]).toContain('Waiting for its turn');
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
  });
  it('shows another round on request', async () => {
    const { host } = await room();
    await type(control(host, 'Round to show'), '2');
    expect(all(host, 'table').map(t => t.getAttribute('aria-label'))).toEqual(['Round 2']);
  });
  it('shows what a closed draft did: the rookies who went to the pool and the picks issued', async () => {
    const { host } = await room({ reading: wire.closed });
    const text = host.textContent;
    expect(text).toContain('The rookie draft is closed.');
    expect(text).toContain('Closed');
    expect(text).toContain('automatically, at the last pick');
    for (const p of wire.closed.closed.pool) expect(text).toContain(p.name);
    expect(text).toContain('2028: 160 picks over 5 rounds, tradable now');
    expect(text).toContain('2029: 160 picks over 5 rounds, tradable now');
    expect(labels(host)).not.toContain('Close the draft');
    expect(labels(host)).not.toContain('Skip this pick');
    expect(labels(host)).toContain('Restore the draft to an earlier point');
  });
  it('says what failed, in Go’s words, and shows no board when the draft cannot be read', async () => {
    const failed = await room({ reading: () => { throw new Error('The draft could not be replayed.'); } });
    expect(failed.host.textContent).toContain('The draft could not be replayed.');
    expect(all(failed.host, 'table')).toHaveLength(0);
  });
});

const launch = async (host: Element, label: string) => {
  const opener = button(host, label);
  opener.focus();
  await click(opener);
  await settle();
  return dialogOf(host)!;
};
const confirm = async (panel: Element, label: string, reason = 'The pick was missed') => {
  await type(control(panel, 'Reason'), reason);
  await settle();
  await click(button(panel, label));
  await settle();
};

describe('The commissioner’s tray', () => {
  it('is offered to a commissioner and never to an owner', async () => {
    const owner = await room({ people: people.owner });
    for (const label of ['Skip this pick', 'Pause the clock', 'Set the clock', 'Close the draft',
      'Restore the draft to an earlier point']) expect(labels(owner.host)).not.toContain(label);
    expect(labels(owner.host).filter(l => l?.startsWith('Change ') || l?.startsWith('Add an extra')))
      .toEqual([]);
    await act(async () => surfaceActions.run({ screen: 'draftroom', action: 'tray', value: 'skip' }));
    expect(dialogOf(owner.host)).toBeUndefined();
    await unmount();
    restore?.();
    const commissioner = await room();
    for (const label of ['Skip this pick', 'Pause the clock', 'Set the clock', 'Close the draft',
      'Restore the draft to an earlier point']) expect(labels(commissioner.host)).toContain(label);
  });
  it('skips the pick on the clock: Go’s preview, a typed reason, then exactly that change', async () => {
    at(noon);
    const { host, sent, reads } = await room();
    const panel = await launch(host, 'Skip this pick');
    expect(all(panel, 'h3').map(h => h.textContent)).toEqual(['Skip this pick']);
    expect(panel.textContent).toContain(wire.previews.skip.affects.words);
    expect(panel.textContent).not.toMatch(/CM-\d|G[123]\b/);
    expect(disabled(button(panel, 'Skip this pick'))).toBe(true);
    expect(sent).toHaveLength(0);
    const before = reads.length;
    await confirm(panel, 'Skip this pick');
    expect(sent).toEqual([expect.objectContaining({ command: 'draft.skip', league: 'native.rules',
      reason: 'The pick was missed', expected: wire.running.revision, effective: 'now',
      visibility: 'league',
      args: { draft: { season: 2027, act: { round: 1, original: 'franchise.02' } } } })]);
    expect(dialogOf(host)).toBeUndefined();
    expect(reads.length).toBeGreaterThan(before);
    expect(host.textContent).toContain('Recorded in the league ledger.');
  });
  it('warns before it swaps a pick already made, and sends the rookie it holds so it cannot retarget',
    async () => {
      at(noon);
      const { host, sent } = await room();
      const panel = await launch(host, 'Change round 1 pick 1, Held team 3');
      expect(panel.textContent)
        .toContain('Round 1, pick 1, Held team 3 — Picked by the franchise: Player, 5001');
      await type(control(panel, 'The rookie to give this pick'), '5004');
      await settle();
      expect(panel.textContent).toContain('This swaps the rookie a pick already made holds');
      await confirm(panel, 'Change this pick', 'Wrong rookie was entered');
      expect(sent[0].command).toBe('draft.change');
      expect(sent[0].args.draft.act).toEqual({ round: 1, original: 'franchise.03', player: '5004',
        replaces: '5001' });
    });
  it('uses player wording when assigning a startup pick', async () => {
    at(noon);
    const reading = { ...wire.running, startup: true };
    const { host } = await room({ reading });
    await act(async () => { draftRoom.choose(true); });
    await settle();
    const panel = await launch(host, 'Choose a player for round 1 pick 2, Held team 1');
    expect(panel.textContent).toContain('Choose a player.');
    expect(panel.textContent).not.toContain('rookie');
    expect(disabled(button(panel, 'Give this player to the pick'))).toBe(true);
    await type(control(panel, 'Player'), '5002');
    await settle();
    expect(panel.textContent).not.toContain('Choose a player.');
  });
  it('gives a rookie to the pick on the clock, and to nothing else without one chosen', async () => {
    at(noon);
    const { host, sent } = await room();
    const panel = await launch(host, 'Choose a rookie for round 1 pick 2, Held team 1');
    expect(disabled(button(panel, 'Give this rookie to the pick'))).toBe(true);
    await type(control(panel, 'Rookie'), '5002');
    await confirm(panel, 'Give this rookie to the pick', 'They are away');
    expect(sent[0].command).toBe('draft.assign');
    expect(sent[0].args.draft.act).toEqual({ round: 1, original: 'franchise.02', player: '5002' });
  });
  it('lets a commissioner give a rookie to a pick the clock skipped, and says it was skipped', async () => {
    at(noon);
    const [made, clockPick, ...rest] = wire.running.draft.picks;
    const skipped = { round: made.round, original: made.original, owner: made.owner, slot: made.slot,
      skipped: true, forfeited: false, madeAt: '2027-07-14T09:00:00-07:00', how: 'expired' };
    const picks = [skipped, clockPick, ...rest];
    const reading = { ...wire.running, draft: { ...wire.running.draft, picks } };
    const { host, sent } = await room({ reading });
    const row = all(host, 'tr').find(r => r.textContent.includes('Skipped: the clock ran out'))!;
    expect(row.textContent).toContain('Held team 3');
    expect(labels(host)).toContain('Choose a rookie for round 1 pick 1, Held team 3');
    const panel = await launch(host, 'Choose a rookie for round 1 pick 1, Held team 3');
    await type(control(panel, 'Rookie'), '5002');
    await confirm(panel, 'Give this rookie to the pick', 'The owner was away');
    expect(sent[0].args.draft.act).toEqual({ round: 1, original: 'franchise.03', player: '5002' });
  });
  it('adds an extra pick only when a person names its franchise and rookie', async () => {
    at(noon);
    const { host, sent } = await room();
    const panel = await launch(host, 'Add an extra pick after round 1 pick 2, Held team 1');
    await type(control(panel, 'Rookie'), '5003');
    await settle();
    expect(panel.textContent).toContain('Choose the franchise that gets the extra pick');
    expect(disabled(button(panel, 'Add this extra pick'))).toBe(true);
    await type(control(panel, 'Franchise that gets the extra pick'), 'franchise.05');
    await confirm(panel, 'Add this extra pick', 'Compensation for the missed pick');
    expect(sent[0].command).toBe('draft.insert');
    expect(sent[0].args.draft.act).toEqual({ round: 1, original: 'franchise.02', player: '5003',
      franchise: 'franchise.05' });
  });
  it('pauses and sets the clock, and resumes a paused one', async () => {
    at(noon);
    const first = await room();
    await confirm(await launch(first.host, 'Pause the clock'), 'Pause the clock', 'Technical problem');
    expect(first.sent[0].command).toBe('draft.timer');
    expect(first.sent[0].args.draft.act).toEqual({ timer: 'pause' });
    const panel = await launch(first.host, 'Set the clock');
    await type(control(panel, 'Minutes the pick on the clock has left'), '30');
    await confirm(panel, 'Set the clock', 'Speed it up');
    expect(first.sent[1].args.draft.act).toEqual({ timer: 'set', seconds: 1800 });
    await unmount();
    restore?.();
    const paused = await room({ reading: wire.paused });
    await confirm(await launch(paused.host, 'Resume the clock'), 'Resume the clock', 'Back again');
    expect(paused.sent[0].args.draft.act).toEqual({ timer: 'resume' });
  });
  it('restores the draft to a point it names, and says nothing is erased', async () => {
    at(noon);
    const { host, sent } = await room();
    const panel = await launch(host, 'Restore the draft to an earlier point');
    await type(control(panel, 'Restore the draft to how it stood after'), wire.checkpoint);
    await settle();
    expect(panel.textContent).toContain('set aside, not erased');
    await confirm(panel, 'Restore the draft', 'We picked in the wrong order');
    expect(sent[0].command).toBe('draft.revert');
    expect(sent[0].args.draft.act).toEqual({ to: wire.checkpoint });
  });
  it('closes the draft only on a held confirm, and sends it once the hold is complete', async () => {
    at(noon);
    const { host, sent } = await room();
    const panel = await launch(host, 'Close the draft');
    expect(panel.textContent).toContain('Every rookie nobody took goes to the free-agent pool');
    const confirmButton = button(panel, 'Close the draft');
    expect(confirmButton.props().onClick).toBeUndefined();
    await type(control(panel, 'Reason'), 'Everyone is done');
    await settle();
    vi.useFakeTimers();
    const target = holder(panel, 'Close the draft');
    const handler = target.props().onPointerDown as (event: object) => void;
    await act(async () => { handler({ button: 0 }); });
    await act(async () => { vi.advanceTimersByTime(300); });
    expect(sent).toHaveLength(0);
    await act(async () => { vi.advanceTimersByTime(500); });
    vi.useRealTimers();
    await settle();
    expect(sent).toHaveLength(1);
    expect(sent[0]).toMatchObject({ command: 'draft.archive', args: { draft: { season: 2027 } } });
    expect(sent[0].args.draft).not.toHaveProperty('act');
  });
  it('says Go’s refusal in its words, and keeps the tray open, when a change is refused', async () => {
    at(noon);
    const words = 'The draft changed while you were editing; read it again.';
    const { host } = await room({ execute: async () => { throw new Error(words); } });
    const panel = await launch(host, 'Skip this pick');
    await confirm(panel, 'Skip this pick');
    expect(dialogOf(host)).toBeDefined();
    expect(dialogOf(host)!.textContent).toContain(words);
  });
  it('keeps one request key for a retry of the same change', async () => {
    at(noon);
    const { host, sent } = await room({ execute: async body => {
      const request = (JSON.parse(body) as { request: string }).request;
      sent.push(JSON.parse(body));
      if (sent.length === 1) throw new Error(`Busy: ${request}`);
      return { entry: 'act.2', league: 'native.rules', revision: 9, recordedAt: '2027-07-14T18:00:00Z',
        message: 'Recorded in the league ledger.' };
    } });
    const panel = await launch(host, 'Skip this pick');
    await confirm(panel, 'Skip this pick');
    await click(button(dialogOf(host)!, 'Skip this pick'));
    await settle();
    expect(sent).toHaveLength(2);
    expect(sent[1].request).toBe(sent[0].request);
  });
});

describe('The owner’s own pick', () => {
  const mine = () => room({ people: people.owner });
  const choose = async (host: Element, player: string) => {
    await type(control(host, 'Rookie'), player);
    await settle();
  };
  it('is offered to the owner whose franchise is on the clock, with the contract it signs', async () => {
    at(noon);
    const { host } = await mine();
    expect(host.textContent).toContain('It is your pick');
    expect(host.textContent).toContain('This pick signs for $3,000,000 for 5 years.');
    expect(disabled(button(host, 'Make this pick'))).toBe(true);
    await choose(host, '5002');
    expect(disabled(button(host, 'Pick Player, 5002'))).toBe(false);
  });
  it('is not offered to a commissioner who owns no franchise, nor to an owner not on the clock', async () => {
    at(noon);
    const commissioner = await room();
    expect(commissioner.host.textContent).not.toContain('It is your pick');
    await unmount();
    restore?.();
    const other = await room({ people: { ...people.owner, yourFranchise: 'franchise.05' } });
    expect(other.host.textContent).not.toContain('It is your pick');
    expect(other.host.textContent).toContain('Held team 5’s next pick');
  });
  it('is not offered while the clock is paused or after the draft closes', async () => {
    at(noon);
    const paused = await room({ people: people.owner, reading: wire.paused });
    expect(paused.host.textContent).not.toContain('It is your pick');
    await unmount();
    restore?.();
    const closed = await room({ people: people.owner, reading: wire.closed });
    expect(closed.host.textContent).not.toContain('It is your pick');
  });
  it('sends the pick for the owner’s franchise at the version it read, and shows the answer', async () => {
    at(noon);
    const { host, picks, reads } = await mine();
    await choose(host, '5002');
    const before = reads.length;
    await click(button(host, 'Pick Player, 5002'));
    await settle();
    expect(picks).toEqual([{ league: 'native.rules', season: 2027, franchise: 'franchise.01',
      player: '5002', request: expect.stringMatching(/^[0-9a-f]{32}$/), expected: wire.running.revision }]);
    expect(host.textContent).toContain('Your pick is recorded.');
    expect(reads.length).toBeGreaterThan(before);
  });
  it('repeats the request key when a pick is sent again, and takes a new one for another', async () => {
    at(noon);
    const { host, picks } = await room({ people: people.owner, pick: async choice => {
      if (picks.length < 3) throw new Error('The league is busy; try again.');
      return { entry: 'act.3', league: choice.league, revision: 9, recordedAt: '2027-07-14T18:00:00Z',
        message: 'Your pick is recorded.' };
    } });
    await choose(host, '5002');
    await click(button(host, 'Pick Player, 5002'));
    await settle();
    expect(host.textContent).toContain('The league is busy; try again.');
    await click(button(host, 'Pick Player, 5002'));
    await settle();
    await choose(host, '5003');
    await click(button(host, 'Pick Player, 5003'));
    await settle();
    expect(picks.map(p => p.player)).toEqual(['5002', '5002', '5003']);
    expect(picks[1].request).toBe(picks[0].request);
    expect(picks[2].request).not.toBe(picks[0].request);
  });
  it('says Go’s refusal in its own words, and asks to read again when the league moved on', async () => {
    at(noon);
    const words = 'Only the owner of this franchise may make its picks';
    const { host } = await room({ people: people.owner, pick: async () => { throw new Error(words); } });
    await choose(host, '5002');
    await click(button(host, 'Pick Player, 5002'));
    await settle();
    expect(all(host, 'p').find(p => p.getAttribute('role') === 'alert')?.textContent).toBe(words);
    await unmount();
    restore?.();
    const stale = await room({ people: people.owner, pick: async () => {
      throw new Error('league version is 9; reread before changing it (expected 8)');
    } });
    await choose(stale.host, '5002');
    await click(button(stale.host, 'Pick Player, 5002'));
    await settle();
    expect(stale.host.textContent).toContain('The league changed while you were editing');
  });
});

describe('Which franchise is shown', () => {
  it('reads the draft for the franchise a person owns, else for the first, and lets a person look at another',
    async () => {
      const owner = await room({ people: people.owner });
      expect(owner.reads[0]).toMatchObject({ season: 0, franchise: 'franchise.01',
        at: '0001-01-01T00:00:00Z' });
      await unmount();
      restore?.();
      const commissioner = await room();
      expect(commissioner.reads[0].franchise).toBe(commissionerSeat());
      await type(control(commissioner.host, 'Franchise to view'), 'franchise.03');
      await settle();
      expect(commissioner.reads.at(-1)?.franchise).toBe('franchise.03');
    });
  it('reads the draft that is next whatever the season is, and never names a code on screen', async () => {
    const { host, reads } = await room();
    expect(reads.every(r => r.season === 0)).toBe(true);
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b|franchise\.\d\d/);
  });
});
const commissionerSeat = () => leagueState.peek().people?.seats[0]?.franchise;
