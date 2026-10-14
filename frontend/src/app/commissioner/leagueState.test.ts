import jobsWire from './jobs.wire.testdata.json';
import controlsWire from './controls.wire.testdata.json';
import { describe, expect, it } from 'vitest';
import { createLeagueState, createProxy, formOf, staleWords, type Port } from './proxy';
import ledger from './wire.testdata.json';
import people from './people.wire.testdata.json';
import rules from './rules.wire.testdata.json';

const receipt = { entry: 'act.new', league: 'native.two', revision: 2, recordedAt: '2026-10-13T12:00:00Z',
  message: 'Recorded in the league ledger.' };
function port(over: Partial<Port> = {}) {
  const calls = { ledger: [] as string[], rules: [] as string[], people: [] as string[] };
  const base: Port = {
    execute: async () => receipt,
    ledger: async league => { calls.ledger.push(league); return { ...ledger, league }; },
    rules: async league => { calls.rules.push(league); return { ...rules, league }; },
    people: async league => { calls.people.push(league); return { ...people.owner, league }; },
    jobs: async () => jobsWire.fresh, controls: async () => controlsWire,
    leagues: async () => [{ id: 'native.one', name: 'One' }, { id: 'native.two', name: 'Two' }],
  };
  return { calls, state: createLeagueState(createProxy({ ...base, ...over })) };
}

describe('League state', () => {
  it('names the form a body belongs to from its command, and an undo by its entry', () => {
    expect(formOf('{"command":"commish.league.create"}')).toBe('create');
    expect(formOf('{"command":"commish.rules.set"}')).toBe('rule');
    expect(formOf('{"command":"commish.role.grant"}')).toBe('role');
    expect(formOf('{"command":"commish.role.revoke"}')).toBe('role');
    expect(formOf('{"command":"commish.correction.revert","args":{"entry":"act.7"}}')).toBe('undo:act.7');
    expect(formOf('not json')).toBe('other');
  });
  it('loads the ledger, rules and people for one league together', async () => {
    const { state, calls } = port();
    await state.choose();
    expect(state.peek().selected).toBe('native.one');
    expect([calls.ledger, calls.rules, calls.people]).toEqual(Array(3).fill(['native.one']));
    expect(state.peek().people?.league).toBe('native.one');
  });
  it('keeps the readings while the same league reloads, and drops them for another league', async () => {
    let release: () => void = () => undefined;
    const slow = new Promise<void>(resolve => { release = resolve; });
    let hold = false;
    const { state } = port({ ledger: async league => {
      if (hold) await slow;
      return { ...ledger, league };
    } });
    await state.choose('native.one');
    hold = true;
    const reload = state.choose();
    expect(state.peek().pending).toBe(true);
    expect(state.peek().rules?.league).toBe('native.one');
    release();
    await reload;
    hold = true;
    const other = state.choose('native.two');
    expect(state.peek().rules).toBeUndefined();
    release();
    await other;
    expect(state.peek().selected).toBe('native.two');
  });
  it('selects the league on a receipt and reloads all three readings for it', async () => {
    const { state, calls } = port();
    await state.choose('native.one');
    calls.ledger.length = calls.rules.length = calls.people.length = 0;
    await state.execute('{"command":"commish.league.create"}');
    expect(state.peek().selected).toBe('native.two');
    expect(state.peek().outcome).toMatchObject({ status: 'done', form: 'create', receipt });
    expect([calls.ledger, calls.rules, calls.people]).toEqual(Array(3).fill(['native.two']));
  });
  it('keeps a Go refusal word for word and reloads nothing', async () => {
    const words = 'Only the commissioner may make this change';
    const { state, calls } = port({ execute: async () => { throw new Error(words); } });
    await state.choose('native.one');
    calls.ledger.length = 0;
    await expect(state.execute('{"command":"commish.rules.set"}')).rejects.toThrow(words);
    expect(state.peek().outcome).toEqual({ status: 'refused', form: 'rule', message: words });
    expect(calls.ledger).toEqual([]);
  });
  it('treats a Wails string rejection like an error', async () => {
    const { state } = port({ execute: () => Promise.reject('Error: give a reason for this change') });
    await state.choose('native.one');
    await expect(state.execute('{}')).rejects.toThrow('give a reason for this change');
    expect(state.peek().outcome)
      .toMatchObject({ status: 'refused', message: 'give a reason for this change' });
  });
  it('turns a stale version into the plain sentence and reloads', async () => {
    const { state, calls } = port({ execute: async () => {
      throw new Error('League version is 9; reread before changing it (expected 3)');
    } });
    await state.choose('native.one');
    calls.rules.length = 0;
    await expect(state.execute('{"command":"commish.rules.set"}')).rejects.toThrow(staleWords);
    expect(state.peek().outcome).toMatchObject({ status: 'refused', message: staleWords });
    expect(calls.rules).toEqual(['native.one']);
  });
});

describe('Landing reading lifecycle', () => {
  it('reads controls without a league, and reloads jobs on each recorded receipt', async () => {
    let reads = 0;
    const { state } = port({ jobs: async league => { reads++; return { ...jobsWire.live, league }; } });
    await state.choose();
    expect(state.peek().jobs?.league).toBe('native.one');
    expect(state.peek().controls?.groups).toHaveLength(8);
    await state.execute('{}');
    expect(state.peek().jobs?.league).toBe('native.two');
    expect(reads).toBe(2);
    const empty = port({ leagues: async () => [] });
    await empty.state.choose();
    expect(empty.state.peek().jobs).toBeUndefined();
    expect(empty.state.peek().controls?.groups).toHaveLength(8);
  });
  it('keeps existing screens usable when jobs fail, without pretending jobs exist', async () => {
    const { state } = port({ jobs: async () => { throw new Error('No recorded calendar'); } });
    await state.choose();
    expect(state.peek().landingError).toContain('No recorded calendar');
    expect(state.peek().jobs).toBeUndefined();
    expect(state.peek().rules?.league).toBe('native.one');
    expect(state.peek().controls?.groups).toHaveLength(8);
    expect(state.peek().pending).toBe(false);
  });
});
