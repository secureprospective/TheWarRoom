import { vi } from 'vitest';

// The command registry prefetches the HQ chunk; a load still pending at teardown fails the run.
vi.mock('../app/shell/FranchiseHQMount', async importOriginal => ({
  ...await importOriginal<typeof import('../app/shell/FranchiseHQMount')>(),
  prefetchHQ: () => Promise.resolve({ default: () => null }),
}));
