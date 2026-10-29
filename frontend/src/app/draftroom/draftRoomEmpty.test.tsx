import { afterEach, expect, it, vi } from 'vitest';
import { unmount } from '../commissioner/commissionerDom';
import { openRoom } from './draftHarness';

let restore: (() => void) | undefined;
afterEach(async () => { await unmount(); restore?.(); vi.restoreAllMocks(); });

// This file's league state has never chosen a league, which is how a first-time install reads.
it('explains how to use the Draft room when no app league has been started', async () => {
  const opened = await openRoom({ leagues: [] });
  restore = opened.restore;
  expect(opened.host.textContent).toContain('No app league has been started.');
  expect(opened.host.textContent).toContain('Create a league in Control Room');
  expect(opened.reads).toHaveLength(0);
});
