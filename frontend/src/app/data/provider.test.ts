import { expect, it, vi } from 'vitest';
import { FixtureProvider, selectProvider } from './provider';

it('returns fixture provenance in the preview provider without a desktop transport', async () => {
  const provider = selectProvider();
  expect(provider).toBeInstanceOf(FixtureProvider);
  const parsed = await provider.snapshot();
  for (const key of ['league', 'franchises', 'rosters', 'players'] as const)
    expect(parsed[key].provenance.kind).toBe('fixture');
  for (const feed of Object.values(parsed.eligibility.feeds)) expect(feed.kind).toBe('fixture');
});

it('loads the real clock fixture and does not synthesize a season notification', async () => {
  const provider = new FixtureProvider();
  const listener = vi.fn();
  const stop = provider.onSeasonChange(listener);
  const clock = await provider.clock();
  stop();
  expect(clock.provenance.kind).toBe('fixture');
  expect(clock.value.phase).toBe('OFFSEASON');
  expect(clock.value.deadlines).toEqual([]);
  expect(clock.value.windows).toHaveLength(9);
  expect(listener).not.toHaveBeenCalled();
});
