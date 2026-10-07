import type { ClockReading, Provenance, Snapshot, Sourced } from './contract';
import { parseClock, parseSnapshot } from './parse';
import { TargetClock, TargetSnapshot } from '../../../wailsjs/go/main/App';

export interface Provider {
  snapshot(): Promise<Snapshot>;
  clock(): Promise<Sourced<ClockReading>>;
}

// Wails injects window.go before the page script runs; a plain browser (dev, screenshots) has none.
function hasWails(): boolean {
  return typeof window !== 'undefined' && (window as { go?: unknown }).go !== undefined;
}

export class FixtureProvider implements Provider {
  async clock(): Promise<Sourced<ClockReading>> {
    try {
      const fixture = await import('./fixtures/clock.json');
      return parseClock(fixture.default);
    } catch (cause) {
      return failedClock(cause, 'fixture');
    }
  }

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

function failedClock(cause: unknown, kind: Provenance['kind']): Sourced<ClockReading> {
  const operation = kind === 'live' ? 'TargetClock' : 'Clock fixture';
  return {
    value: { season: 0, phase: 'OFFSEASON', deadlines: [], windows: [] },
    provenance: {
      source: 'phase-log+commissioner-calendar',
      kind,
      freshness: { state: 'fail', fetchedAt: '', note: `${operation} failed: ${String(cause)}` },
    },
  };
}

export class LiveProvider implements Provider {
  async clock(): Promise<Sourced<ClockReading>> {
    try {
      if (!hasWails()) throw new Error('Wails runtime absent');
      return parseClock(await TargetClock());
    } catch (cause) {
      return failedClock(cause, 'live');
    }
  }

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
