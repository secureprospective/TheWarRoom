import { describe, expect, it, vi } from 'vitest';
import { actTable } from '../commissioner/actTable';
import { situations } from '../control/room/guide';
import { filterSituations, followSituationRoute, situationCategories } from './CtrlSituationsV3';

describe('situationCategories', () => {
  it('starts with ALL and counts each category', () => {
    const counts = situationCategories(situations);
    expect(counts[0]).toEqual({ category: 'ALL', count: situations.length });
    const gamesCount = situations.filter(s => s[0] === 'GAMES').length;
    expect(counts.find(c => c.category === 'GAMES')).toEqual({ category: 'GAMES', count: gamesCount });
  });
});

describe('filterSituations', () => {
  it('returns all situations for ALL and filters by category otherwise', () => {
    expect(filterSituations('ALL', situations)).toHaveLength(situations.length);
    const trades = filterSituations('TRADES', situations);
    expect(trades.length).toBeGreaterThan(0);
    expect(trades.every(s => s[0] === 'TRADES')).toBe(true);
  });
});

describe('followSituationRoute', () => {
  it('dispatches tab, tool, unavailable, and unknown routes to room actions', () => {
    const navigateTo = vi.fn();
    const openTool = vi.fn();
    const showRoute = vi.fn();
    const c = { navigateTo, openTool, showRoute };

    followSituationRoute({ kind: 'tab', tab: 'rules' }, c);
    expect(navigateTo).toHaveBeenCalledWith('rules');

    const definition = actTable[0]!;
    followSituationRoute({ kind: 'tool', definition }, c);
    expect(openTool).toHaveBeenCalledWith(definition);

    followSituationRoute({ kind: 'unavailable', words: 'Not on server yet.' }, c);
    expect(showRoute).toHaveBeenCalledWith('Not on server yet.');

    followSituationRoute({ kind: 'unknown', route: 'foo.bar' }, c);
    expect(showRoute).toHaveBeenCalledWith('This situation route is not registered: foo.bar');
  });
});
