import type { Provenance, Snapshot } from './contract';
import { parseSnapshot } from './parse';
import { TargetSnapshot } from '../../../wailsjs/go/main/App';

export interface Provider {
  snapshot(): Promise<Snapshot>;
}

// Wails injects window.go before the page script runs; a plain browser (dev, screenshots) has none.
function hasWails(): boolean {
  return typeof window !== 'undefined' && (window as { go?: unknown }).go !== undefined;
}

export class FixtureProvider implements Provider {
  async snapshot(): Promise<Snapshot> {
    const fixture = await import('./fixtures/snapshot.json');
    return parseSnapshot(fixture.default);
  }
}

function failedSnapshot(cause: unknown): Snapshot {
  const provenance = (): Provenance => ({
    source: 'mfl-mirror',
    kind: 'live',
    freshness: {
      state: 'fail',
      fetchedAt: '',
      note: `TargetSnapshot failed: ${String(cause)}`,
    },
  });
  return {
    league: { value: { season: 0, franchiseCount: 0 }, provenance: provenance() },
    franchises: { value: [], provenance: provenance() },
    rosters: { value: [], provenance: provenance() },
    players: { value: [], provenance: provenance() },
  };
}

export class LiveProvider implements Provider {
  async snapshot(): Promise<Snapshot> {
    try {
      if (!hasWails()) throw new Error('Wails runtime absent');
      return parseSnapshot(await TargetSnapshot());
    } catch (cause) {
      return failedSnapshot(cause);
    }
  }
}

export function selectProvider(): Provider {
  return hasWails() ? new LiveProvider() : new FixtureProvider();
}
