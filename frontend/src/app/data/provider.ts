import type {
  LineupReading, ClockReading, EnvelopeDemo, MFLKeyStatus, Provenance, Receipt, Snapshot, Sourced,
} from './contract';
import {
  DeleteMFLKey, MFLKeyStatus as ReadMFLKey, SetMFLKey,
  TargetClock, TargetDraftIR, TargetMoves, TargetSnapshot,
} from '../../../wailsjs/go/main/App';

interface ReadingProvider {
  mflKey(): Promise<MFLKeyStatus>;
  connectMFL(key: string): Promise<MFLKeyStatus>;
  forgetMFL(): Promise<MFLKeyStatus>;
  snapshot(): Promise<Snapshot>;
  clock(): Promise<Sourced<ClockReading>>;
  onClockChange(listener: () => void): () => void;
  onSeasonChange(listener: () => void): () => void;
  lineup(franchiseId: string): Promise<LineupReading>;
}

export interface DraftingProvider {
  kind: 'live';
  draftIR(franchiseId: string, playerId: string): Promise<Receipt>;
  moves(franchiseId: string): Promise<Receipt[]>;
}
export interface DemoProvider {
  kind: 'fixture';
  reason: string;
  demo(): Promise<EnvelopeDemo>;
}
export type Provider = ReadingProvider & (DraftingProvider | DemoProvider);

// Wails injects window.go before the page script runs; a plain browser (dev, screenshots) has none.
function hasWails(): boolean {
  return typeof window !== 'undefined' && (window as { go?: unknown }).go !== undefined;
}

export class FixtureProvider implements ReadingProvider, DemoProvider {
  readonly kind = 'fixture';
  readonly reason = 'Drafting needs the desktop app';
  async mflKey(): Promise<MFLKeyStatus> {
    return { state: 'absent', league: '', season: 0, detail: 'Connecting MFL needs the desktop app' };
  }
  async connectMFL(_key: string): Promise<MFLKeyStatus> {
    throw new Error('Connecting MFL needs the desktop app');
  }
  async forgetMFL(): Promise<MFLKeyStatus> {
    throw new Error('Connecting MFL needs the desktop app');
  }
  onClockChange(_listener: () => void): () => void {
    return () => {};
  }
  onSeasonChange(_listener: () => void): () => void {
    return () => {};
  }
  async lineup(_franchiseId: string): Promise<LineupReading> {
    throw new Error('Lineups need the desktop app');
  }
  async demo(): Promise<EnvelopeDemo> {
    const [{ parseEnvelopeDemo }, fixture] = await Promise.all([
      import('./parseEnvelope'), import('./fixtures/envelope-demo.json'),
    ]);
    return parseEnvelopeDemo(fixture.default);
  }

  async clock(): Promise<Sourced<ClockReading>> {
    try {
      const [{ parseClock }, fixture] = await Promise.all([
        import('./parse'), import('./fixtures/clock.json'),
      ]);
      return parseClock(fixture.default);
    } catch (cause) {
      return failedClock(cause, 'fixture');
    }
  }

  async snapshot(): Promise<Snapshot> {
    const [{ parseSnapshot }, fixture] = await Promise.all([
      import('./parse'), import('./fixtures/snapshot.json'),
    ]);
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

function onTargetChange(event: string, listener: () => void): () => void {
  let active = true;
  let unsubscribe: (() => void) | undefined;
  void import('../../../wailsjs/runtime/runtime').then(({ EventsOn }) => {
    if (!active) return;
    unsubscribe = EventsOn(event, listener);
    // An event emitted while the runtime chunk loaded was missed; catch up once.
    listener();
  }).catch(() => {
    // No runtime chunk means no change events; the initial read still reports availability.
  });
  return () => {
    active = false;
    unsubscribe?.();
  };
}

export class LiveProvider implements ReadingProvider, DraftingProvider {
  readonly kind = 'live';
  async mflKey(): Promise<MFLKeyStatus> {
    const [{ parseMFLKeyStatus }, status] = await Promise.all([import('./parse'), ReadMFLKey()]);
    return parseMFLKeyStatus(status);
  }
  async connectMFL(key: string): Promise<MFLKeyStatus> {
    const [{ parseMFLKeyStatus }, status] = await Promise.all([import('./parse'), SetMFLKey(key)]);
    return parseMFLKeyStatus(status);
  }
  async forgetMFL(): Promise<MFLKeyStatus> {
    const [{ parseMFLKeyStatus }, status] = await Promise.all([import('./parse'), DeleteMFLKey()]);
    return parseMFLKeyStatus(status);
  }
  onClockChange(listener: () => void): () => void {
    return onTargetChange('target:clock', listener);
  }
  onSeasonChange(listener: () => void): () => void {
    return onTargetChange('target:season', listener);
  }
  async lineup(franchiseId: string): Promise<LineupReading> {
    const [{ parseLineup }, reading] = await Promise.all([
      import('./parseLineup'),
      import('../../../wailsjs/go/main/App').then(({ TargetLineup }) => TargetLineup(franchiseId)),
    ]);
    return parseLineup(reading);
  }
  async draftIR(franchiseId: string, playerId: string): Promise<Receipt> {
    const [{ parseReceipt }, receipt] = await Promise.all([
      import('./parseEnvelope'), TargetDraftIR(franchiseId, playerId),
    ]);
    return parseReceipt(receipt);
  }
  async moves(franchiseId: string): Promise<Receipt[]> {
    const [{ parseReceipt }, receipts] = await Promise.all([
      import('./parseEnvelope'), TargetMoves(franchiseId),
    ]);
    if (!Array.isArray(receipts)) throw new Error('TargetMoves: expected array');
    return receipts.map(parseReceipt);
  }

  async clock(): Promise<Sourced<ClockReading>> {
    try {
      if (!hasWails()) throw new Error('Wails runtime absent');
        const [{ parseClock }, payload] = await Promise.all([
        import('./parse'), TargetClock(),
      ]);
      return parseClock(payload);
    } catch (cause) {
      return failedClock(cause, 'live');
    }
  }

  async snapshot(): Promise<Snapshot> {
    try {
      if (!hasWails()) throw new Error('Wails runtime absent');
        const [{ parseSnapshot }, payload] = await Promise.all([
        import('./parse'), TargetSnapshot(),
      ]);
      return parseSnapshot(payload);
    } catch (cause) {
      return failedSnapshot(cause);
    }
  }
}

export function selectProvider(): Provider {
  return hasWails() ? new LiveProvider() : new FixtureProvider();
}
