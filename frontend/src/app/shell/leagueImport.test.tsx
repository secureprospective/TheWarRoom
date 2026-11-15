// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import { LeagueContext, LeaguePicker, useLeagues, LEAGUE_KEY, type LeagueState } from './leaguePicker';
import { createMigration, migration } from '../control/migration';
import MoveMFL from '../control/MoveMFL';

it('rereads after import, selects the league and preserves it on refresh failure', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  const mirror = { id: 'mirror', name: 'Reference', teams: 32, app: false };
  const added = { id: 'league.new', name: 'Our imported league', teams: 12, app: true };
  let choices = [mirror];
  const read = vi.fn(async () => choices);
  let state: LeagueState;
  const flow = createMigration({ run: async command => {
    if (command.endsWith('prepare')) return { league: added.id, name: added.name, season: 2026,
      franchises: 12, players: 0, picks: 0, settings: 1, notes: [], problems: [], ready: true,
      preset: 'default32', waiting: [] };
    choices = [mirror, added];
    return [{ entry: 'import.1', league: added.id, revision: 1,
      recordedAt: '2026-10-31T12:00:00Z', message: 'Imported' }];
  }, open: async () => {} });
  const spy = vi.spyOn(migration, 'use').mockImplementation(flow.use);
  function Screen() {
    state = useLeagues(read);
    return <LeagueContext.Provider value={state}>
      <LeaguePicker leagues={state.leagues} selected={state.selected} onSelect={state.select}
        error={state.error} /><MoveMFL />
    </LeagueContext.Provider>;
  }
  const container = document.createElement('div');
  const root = createRoot(container);
  try {
    await act(async () => { root.render(<Screen />); });
    expect(read).toHaveBeenCalledTimes(1);
    expect(state!.selectedId).toBe('mirror');
    await act(async () => {
      flow.input('12345');
      await flow.prepare();
      flow.reason('Move our league');
      await flow.importLeague();
    });
    expect(read).toHaveBeenCalledTimes(2);
    expect(state!.selectedId).toBe(added.id);
    expect(localStorage.getItem(LEAGUE_KEY)).toBe(added.id);
    expect(container.querySelector('.league b')?.textContent).toBe(added.name);
    const switcher = [...container.querySelectorAll('button')].find(b => b.textContent === 'Switch league')!;
    await act(async () => { switcher.click(); });
    expect(container.textContent).toContain(added.name);
    expect(state!.leagues).toEqual([mirror, added]);
    read.mockRejectedValueOnce(new Error('offline'));
    await act(async () => { state!.refresh(added.id); });
    expect(state!.selectedId).toBe(added.id);
    expect(state!.leagues).toEqual([mirror, added]);
    expect(state!.error).toContain('offline');
    expect(flow.peek().done).toBe(added.id);
  } finally {
    await act(async () => { root.unmount(); });
    spy.mockRestore();
    localStorage.clear();
    vi.unstubAllGlobals();
  }
});
