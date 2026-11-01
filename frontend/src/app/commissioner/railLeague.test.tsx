// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import { LeagueContext, type LeagueState } from '../shell/leaguePicker';
import { LeagueRoom } from '../control/LeagueRoom';
import { leagueState } from './proxy';
import { stub } from './commissionerStub';
import jobsWire from './jobs.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';

it('follows the rail league, without a second picker or app readings on the mirror', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  const { app, calls } = stub({ TargetCommissionerLeagues: async () => [
    { id: 'other', name: 'Other league' }, { id: rulesWire.league, name: rulesWire.name },
  ], TargetCommissionerJobs: async () => jobsWire.fresh });
  Object.defineProperty(window, 'go', { configurable: true, value: { commissioner: { API: app } } });
  const leagues = [
    { id: 'mirror', name: 'Reference', teams: 32, app: false },
    { id: rulesWire.league, name: rulesWire.name, teams: 12, app: true },
    { id: 'other', name: 'Other league', teams: 12, app: true },
  ];
  const container = document.createElement('div');
  const root = createRoot(container);
  const show = async (id: string) => {
    const state: LeagueState = { leagues, selectedId: id, selected: leagues.find(l => l.id === id),
      loading: false, select: () => {} };
    await act(async () => { root.render(<LeagueContext.Provider value={state}>
      <LeagueRoom /></LeagueContext.Provider>); });
    for (let i = 0; i < 60 && (!container.querySelector('.move-mfl') ||
      !container.querySelector('.league-landing') || leagueState.peek().pending); i++)
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 20)); });
  };
  try {
    await show(rulesWire.league);
    expect(leagueState.peek().selected).toBe(rulesWire.league);
    expect(calls.rules).toContain(rulesWire.league);
    expect(calls.rules).not.toContain('other');
    expect(container.querySelector('.jobs-strip')).not.toBeNull();
    expect(container.querySelector('.control-search')).not.toBeNull();
    expect([...container.querySelectorAll('label')].some(l => l.textContent?.startsWith('App league')))
      .toBe(false);
    expect(container.querySelector('.mfl-reference')).toBeNull();
    expect(container.querySelector('.move-mfl')).not.toBeNull();
    await show('other');
    expect(calls.rules).toContain('other');
    const reads = calls.rules.length;
    await show('mirror');
    expect(leagueState.peek().selected).toBe('');
    expect(leagueState.peek().jobs).toBeUndefined();
    expect(calls.rules).toHaveLength(reads);
    expect(container.querySelector('.mfl-reference')).not.toBeNull();
    expect(container.querySelector('.move-mfl')).not.toBeNull();
  } finally {
    await act(async () => { root.unmount(); });
    delete (window as unknown as { go?: unknown }).go;
    vi.unstubAllGlobals();
  }
});
