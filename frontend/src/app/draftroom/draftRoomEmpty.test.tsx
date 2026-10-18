import { afterEach, expect, it, vi } from 'vitest';
import { unmount } from '../commissioner/commissionerDom';
import { openRoom } from './draftHarness';

let restore: (() => void) | undefined;
afterEach(async () => { await unmount(); restore?.(); vi.restoreAllMocks(); });

// This file's league state has never chosen a league, which is how a first-time install reads.
it('shows nothing in the Draft room where no app league has been started', async () => {
  const opened = await openRoom({ leagues: [] });
  restore = opened.restore;
  expect(opened.host.textContent).toBe('');
  expect(opened.reads).toHaveLength(0);
});
