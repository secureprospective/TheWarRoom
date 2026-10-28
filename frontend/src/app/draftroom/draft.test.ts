import { describe, expect, it } from 'vitest';
import { parseDraft } from './draft';
import { checkpointWords, contractWords, countdownWords, isOnClock, money, pickWords } from './draftWords';
import { changeOf, missing, noInputs, parseTrayAction, trayValue } from './trayAction';
import wire from './draft.wire.testdata.json';

const running = parseDraft(wire.running);

describe('The draft wire Go writes', () => {
  it('reads the running, paused and closed drafts strictly', () => {
    expect(running.season).toBe(2027);
    expect(running.onClock).toMatchObject({ round: 1, owner: 'franchise.01', paused: false });
    expect(parseDraft(wire.paused).onClock?.paused).toBe(true);
    const closed = parseDraft(wire.closed);
    expect(closed.closed?.automatic).toBe(true);
    expect(closed.closed?.pool).toHaveLength(5);
    expect(closed.closed?.issued.map(i => [i.season, i.picks, i.tradable]))
      .toEqual([[2028, 160, true], [2029, 160, true]]);
    expect(closed.onClock).toBeNull();
    expect(() => parseDraft({ ...wire.running, confidence: 0.4 })).toThrow('unexpected field');
    expect(() => parseDraft({ ...wire.running, season: 0 })).toThrow();
    const odd = { ...wire.running.draft, picks: [{ ...wire.running.draft.picks[0], how: 'luck' }] };
    expect(() => parseDraft({ ...wire.running, draft: odd })).toThrow('Picks[0].how');
    expect(() => parseDraft({ ...wire.running, nextOwnedPick: { round: 1 } })).toThrow();
  });
  it('names every rookie a pick mentions, and keeps round one’s words while it cannot be ordered', () => {
    for (const pick of running.picks) {
      if (pick.player !== '') expect(running.players[pick.player]).toBeDefined();
    }
    expect(running.available.map(p => p.id)).not.toContain('5001');
  });
});

describe('The words a GM reads', () => {
  it('counts down in the two units that matter, and says when time is up', () => {
    expect(countdownWords(5.5 * 3600_000)).toBe('5 hours 30 minutes');
    expect(countdownWords(3600_000)).toBe('1 hour');
    expect(countdownWords(90_500)).toBe('1 minute 31 seconds');
    expect(countdownWords(45_000)).toBe('45 seconds');
    expect(countdownWords(0)).toBe('Time is up');
    expect(countdownWords(-5000)).toBe('Time is up');
  });
  it('writes money and contracts from whole cents', () => {
    expect(money(300000000)).toBe('$3,000,000');
    expect(money(142000050)).toBe('$1,420,000.50');
    expect(contractWords({ salaryCents: 300000000, years: 5, rfaAfter: false }))
      .toBe('$3,000,000 for 5 years');
    expect(contractWords({ salaryCents: 142000000, years: 3, rfaAfter: true }))
      .toBe('$1,420,000 for 3 years, then restricted free agency');
  });
  it('says what became of each pick without a code', () => {
    const [made, onClock] = running.picks;
    expect(pickWords(running, made)).toBe('Picked by the franchise');
    expect(pickWords(running, onClock)).toBe('On the clock');
    expect(isOnClock(running, onClock)).toBe(true);
    const paused = parseDraft(wire.paused);
    expect(pickWords(paused, paused.picks[1])).toBe('On the clock, paused');
    const closed = parseDraft(wire.closed);
    expect(pickWords(closed, closed.picks[1])).toBe('Skipped: the clock ran out');
    const waiting = running.picks.find(p => p.player === '' && !p.skipped && !isOnClock(running, p))!;
    expect(pickWords(running, waiting)).toBe('Waiting for its turn');
    expect(checkpointWords(running.checkpoints[0])).toContain('Picked Player, 5001');
    const text = JSON.stringify(running.picks.map(p => pickWords(running, p)));
    expect(text).not.toMatch(/CM-\d|G[123]\b/);
  });
});

describe('The change the tray sends', () => {
  const clock = running.onClock!;
  it('names the pick, the rookie and the rookie it holds now, at the version the form read', () => {
    const ref = { round: 1, original: 'franchise.03', extra: 0 };
    const input = { ...noInputs, player: '5004' };
    expect(changeOf({ kind: 'change', ref }, running, input, ' Fix it ')).toEqual({
      league: 'native.seed', command: 'draft.change', reason: 'Fix it', expected: running.revision,
      args: { draft: { season: 2027, act: { round: 1, original: 'franchise.03', player: '5004',
        replaces: '5001' } } }, effective: 'now', visibility: 'league' });
    expect(changeOf({ kind: 'skip' }, running, noInputs, 'r')?.args).toEqual({ draft: { season: 2027,
      act: { round: clock.round, original: clock.original } } });
    expect(changeOf({ kind: 'set' }, running, { ...noInputs, minutes: '30' }, '')?.args)
      .toEqual({ draft: { season: 2027, act: { timer: 'set', seconds: 1800 } } });
    expect(changeOf({ kind: 'close' }, running, noInputs, '')?.args).toEqual({ draft: { season: 2027 } });
    expect(changeOf({ kind: 'restore' }, running, { ...noInputs, checkpoint: 'act.1' }, '')?.command)
      .toBe('draft.revert');
  });
  it('opens a startup draft, and names it on every change to it', () => {
    const startup = { ...running, startup: true };
    expect(changeOf({ kind: 'open' }, startup, noInputs, ' Start ')).toEqual({ league: 'native.seed',
      command: 'draft.startup.open', reason: 'Start', expected: running.revision,
      args: { draft: { season: 2027, startup: true } }, effective: 'now', visibility: 'league' });
    expect(changeOf({ kind: 'close' }, startup, noInputs, '')?.args)
      .toEqual({ draft: { season: 2027, startup: true } });
    expect(parseTrayAction(trayValue({ kind: 'open' }))).toEqual({ kind: 'open' });
  });
  it('is not whole until everything it needs is chosen, and says what is missing', () => {
    const ref = { round: 1, original: 'franchise.02', extra: 0 };
    expect(missing({ kind: 'assign', ref }, running, noInputs)).toBe('Choose a rookie.');
    expect(missing({ kind: 'insert', ref }, running, { ...noInputs, player: '5002' }))
      .toContain('franchise that gets the extra pick');
    for (const minutes of ['', '0', '1.5', 'x']) {
      expect(changeOf({ kind: 'set' }, running, { ...noInputs, minutes }, '')).toBeNull();
    }
    expect(changeOf({ kind: 'restore' }, running, noInputs, '')).toBeNull();
  });
  it('travels as the value of a launcher and back', () => {
    const ref = { round: 2, original: 'franchise.04', extra: 1 };
    for (const action of [{ kind: 'skip' }, { kind: 'close' }, { kind: 'assign', ref },
      { kind: 'change', ref }] as const) {
      expect(parseTrayAction(trayValue(action))).toEqual(action);
    }
    for (const bad of ['', 'skip 1', 'assign', 'assign 0 franchise.01 0', 'assign 1 franchise.01 -1',
      'assign x franchise.01 0', 'delete']) expect(parseTrayAction(bad)).toBeNull();
  });
});

it('sends the side of the startup draft for commissioner actions only when present', () => {
  const draft = parseDraft({ ...wire.running, startup: true, side: 'AFC' });
  expect(changeOf({ kind: 'pause' }, draft, noInputs, 'pause')?.args)
    .toEqual({ draft: { season: draft.season, startup: true, side: 'AFC', act: { timer: 'pause' } } });
  expect(changeOf({ kind: 'pause' }, running, noInputs, 'pause')?.args)
    .toEqual({ draft: { season: running.season, act: { timer: 'pause' } } });
  expect(changeOf({ kind: 'open' }, draft, noInputs, 'open')?.args)
    .toEqual({ draft: { season: draft.season, startup: true } });
});
