import { act } from 'react';
import { commands } from '../commands/registry';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { button, click, control, mount, render, rerender, settle, type, unmount } from './commissionerDom';
import { LedgerFilters } from './LedgerFilters';
import LeagueLedger, { LedgerEntries } from './LeagueLedger';
import { stub } from './commissionerStub';
import { leagueState } from './proxy';
import { parseLedger, readLedgerFiltered, type Ledger } from './proxy';
import wire from './ledger-filters.wire.testdata.json';

const reading = parseLedger(wire.all);
const child = (reading: Ledger, filtered: boolean) => render(LedgerEntries, { reading, filtered });
afterEach(unmount);

describe('Ledger filters over Go readings', () => {
  it('keeps the active filters across a refresh of the commissioner readings', async () => {
    let calls = 0;
    let waiting = false;
    let release: () => void = () => undefined;
    const pause = new Promise<void>(resolve => { release = resolve; });
    const app = stub({
      TargetCommissionerLedger: async () => { if (waiting) await pause; return wire.all; },
      TargetCommissionerLedgerFiltered: async () => { calls++; return wire.rules; },
    }).app;
    const host = await mount(render(LeagueLedger), app);
    await leagueState.choose('native.rules');
    await settle();
    await type(control(host, 'Kind of change'), 'rules');
    await click(button(host, 'Find changes'));
    await settle();
    waiting = true;
    await click(button(host, 'Refresh ledger'));
    await settle();
    try {
      expect(control(host, 'Kind of change').props().value).toBe('rules');
    } finally {
      release();
      await settle();
    }
    expect(calls).toBe(2);
    expect(host.textContent).not.toContain(wire.setup.entries[0].what);
  });
  it('sends person, kind and viewer-local bounds and presents only Go’s returned entries', async () => {
    const calls: { league: string; filter: unknown }[] = [];
    const host = await mount(render(LedgerFilters, { reading, children: child }), {
      TargetCommissionerLedgerFiltered: async (league, body) => {
        calls.push({ league, filter: JSON.parse(body) });
        return wire.rules;
      },
    });
    await type(control(host, 'Person'), 'creator');
    await type(control(host, 'Kind of change'), 'rules');
    await type(control(host, 'From'), '2026-10-13T13:00');
    await type(control(host, 'Before'), '2026-10-13T15:00');
    expect(calls).toHaveLength(0);
    await click(button(host, 'Find changes'));
    await settle();
    expect(calls).toEqual([{ league: 'native.shell', filter: {
      person: 'creator', kind: 'rules', from: new Date('2026-10-13T13:00').toISOString(),
      before: new Date('2026-10-13T15:00').toISOString(),
    } }]);
    expect(host.textContent).toContain(wire.rules.entries[0].what);
    expect(host.textContent).not.toContain(wire.setup.entries[0].what);
    expect(host.textContent).not.toContain('League version');
    await click(button(host, 'Clear filters'));
    expect(host.textContent).toContain(wire.setup.entries[0].what);
    expect(control(host, 'Person').props().value).toBe('');
    expect(calls).toHaveLength(1);
  });
  it('does not reinterpret Go’s reading by kind or synthesize private owner rows', async () => {
    const host = await mount(render(LedgerFilters, { reading, children: child }), {
      TargetCommissionerLedgerFiltered: async () => wire.setup,
    });
    await type(control(host, 'Kind of change'), 'rules');
    await click(button(host, 'Find changes'));
    await settle();
    expect(host.textContent).toContain(wire.setup.entries[0].what);
    await rerender(render(LedgerFilters,
      { key: 'owner', reading: parseLedger(wire.owner), children: child }));
    expect(host.textContent).not.toContain('secret-deputy');
    expect(host.textContent).toContain('No changes are visible to you');
  });
  it('shows an empty filtered result and Go refusals without falling back to unfiltered rows', async () => {
    let failed = false;
    const words = 'The ledger start must be earlier than its end';
    const host = await mount(render(LedgerFilters, { reading, children: child }), {
      TargetCommissionerLedgerFiltered: async () => {
        if (failed) throw new Error(words);
        return wire.absent;
      },
    });
    await type(control(host, 'Person'), 'nobody');
    await click(button(host, 'Find changes'));
    await settle();
    expect(host.textContent).toContain('No changes match these filters.');
    expect(host.textContent).not.toContain(wire.setup.entries[0].what);
    failed = true;
    await type(control(host, 'Person'), 'anybody');
    await click(button(host, 'Find changes'));
    await settle();
    expect(host.textContent).toContain(words);
    expect(host.textContent).not.toContain(wire.setup.entries[0].what);
    expect(host.textContent).not.toContain('Finding league changes');
  });
  it('ignores an older answer and reloads the current filter when the league version changes', async () => {
    let finish: (reading: Ledger) => void = () => undefined;
    let calls = 0;
    const read: typeof readLedgerFiltered = async (_league, filter) => {
      calls++;
      if (filter.kind === 'rules') return new Promise(resolve => { finish = resolve; });
      return parseLedger(wire.people);
    };
    const props = { reading, children: child, read };
    const host = await mount(render(LedgerFilters, props));
    await type(control(host, 'Kind of change'), 'rules');
    await click(button(host, 'Find changes'));
    await type(control(host, 'Kind of change'), 'people');
    await click(button(host, 'Find changes'));
    await settle();
    finish(parseLedger(wire.rules));
    await settle();
    expect(host.textContent).toContain(wire.people.entries[0].what);
    expect(host.textContent).not.toContain(wire.rules.entries[0].what);
    await rerender(render(LedgerFilters, { ...props, reading: { ...reading, revision: 5 } }));
    await settle();
    expect(calls).toBe(3);
  });
});

it.each(['done', 'failed'] as const)('records ledger filter %s after its read settles', async status => {
  let finish!: (value: Ledger) => void;
  let fail!: (cause: Error) => void;
  const reason = 'The ledger is unavailable. Try again later.';
  const read = vi.fn(() => new Promise<Ledger>((resolve, reject) => { finish = resolve; fail = reject; }));
  const logged = vi.spyOn(console, 'error').mockImplementation(() => {});
  try {
    const host = await mount(render(LedgerFilters, { reading, children: child, read }));
    await type(control(host, 'Kind of change'), 'rules');
    const before = commands.history();
    let result: void | Promise<void>;
    await act(async () => {
      result = commands.dispatch('commissioner.surface', {
        screen: `ledger.${reading.league}`, action: 'find', value: '',
      });
    });
    expect(read).toHaveBeenCalledTimes(1);
    expect(commands.history()).toEqual(before);
    expect(host.textContent).toContain('Finding league changes');
    await act(async () => {
      if (status === 'done') finish(parseLedger(wire.rules));
      else fail(new Error(reason));
      await result;
    });
    expect(commands.history()[commands.history().length - 1]).toMatchObject({
      id: 'commissioner.surface', status,
      ...(status === 'failed' ? { reason } : {}) });
    if (status === 'failed') {
      expect(host.textContent).toContain(reason);
      expect(logged).toHaveBeenCalled();
    } else expect(host.textContent).toContain(wire.rules.entries[0].what);
  } finally { logged.mockRestore(); }
});
