import type {
  LineupCheck, LineupReading, ClockReading, EnvelopeDemo, MFLKeyStatus,
  AlertReading, WindowKind, Provenance, Receipt, Snapshot, Sourced, TradeReading,
} from './contract';
import { ruleDatedWindows } from '../clock/labels';
interface ReadingProvider {
  alerts(franchiseId: string): Promise<AlertReading>;
  trades(franchiseId: string): Promise<TradeReading>;
  draftTradeAccept(franchiseId: string, tradeId: string): Promise<Receipt>;
  checkLineup(franchiseId: string, starters: string[]): Promise<LineupCheck>;
  draftLineup(franchiseId: string, starters: string[]): Promise<Receipt>;
  handOff(correlationId: string): Promise<Receipt>;
  onMovesChange(listener: () => void): () => void;
  mflKey(): Promise<MFLKeyStatus>;
  connectMFL(key: string): Promise<MFLKeyStatus>;
  forgetMFL(): Promise<MFLKeyStatus>;
  snapshot(): Promise<Snapshot>;
  clock(): Promise<Sourced<ClockReading>>;
  setWindow(kind: WindowKind, at: string): Promise<void>;
  onClockChange(listener: () => void): () => void;
  onSeasonChange(listener: () => void): () => void;
  lineup(franchiseId: string): Promise<LineupReading>;
}

export interface DraftingProvider {
  kind: 'live';
  draftIR(franchiseId: string, playerId: string): Promise<Receipt>;
  draftTaxi(franchiseId: string, playerId: string): Promise<Receipt>;
  moves(franchiseId: string): Promise<Receipt[]>;
}
export interface DemoProvider {
  kind: 'fixture';
  reason: string;
  demo(): Promise<EnvelopeDemo>;
}
export type Provider = ReadingProvider & (DraftingProvider | DemoProvider);

// League dates can be rehearsed in fixtures; remote acts still require the desktop app.
const desktop = (what: string) => async (): Promise<never> => {
  throw new Error(`${what} the desktop app`);
};
const desktopLineup = desktop('Lineup changes need');
const desktopTrades = desktop('Trades need');
export class FixtureProvider implements ReadingProvider, DemoProvider {
  readonly kind = 'fixture';
  readonly reason = 'Drafting needs the desktop app';
  private heldClock?: Sourced<ClockReading>;
  private baseClock?: Sourced<ClockReading>;
  private clockListeners = new Set<() => void>();
  async setWindow(kind: WindowKind, at: string): Promise<void> {
    const reading = await this.clock();
    if (at && Number.isNaN(new Date(at).getTime())) throw new Error('Choose a valid date and time');
    if (reading.provenance.freshness.state === 'fail' || !this.baseClock) {
      throw new Error('League clock unavailable; reload before changing dates');
    }
    const base = this.baseClock.value.windows.find(window => window.kind === kind);
    if (!base) throw new Error('Unknown league window');
    if (!at && !reading.value.windows.find(window => window.kind === kind)?.moved) {
      throw new Error('This window already follows the rulebook; there is nothing to reset');
    }
    const windows = reading.value.windows.map(window => window.kind !== kind ? window : at
      ? { ...base, status: 'dated' as const, at, moved: true,
        reason: ruleDatedWindows.has(kind) ? `Moved by the commissioner · ${base.reason}`
          : 'Set by the commissioner' }
      : { ...base });
    const { grade } = await import('../clock/urgency');
    const now = Date.now();
    const deadlines = reading.value.deadlines.filter(deadline => !deadline.id.startsWith('window:'));
    for (const window of windows) {
      if (window.at && Date.parse(window.at) > now) deadlines.push({
        id: `window:${window.kind}`, label: window.kind, at: window.at,
        urgency: grade(window.at, now), pinned: Date.parse(window.at) - now <= 172_800_000,
        promoted: Date.parse(window.at) - now <= 604_800_000,
      });
    }
    deadlines.sort((a, b) => b.urgency.localeCompare(a.urgency)
      || (Date.parse(a.at ?? '') - Date.parse(b.at ?? '')) || a.id.localeCompare(b.id));
    this.heldClock = { ...reading, value: { ...reading.value, windows, deadlines } };
    this.clockListeners.forEach(listener => listener());
  }
  alerts: ReadingProvider['alerts'] = desktop('Alerts need');
  trades: ReadingProvider['trades'] = desktopTrades;
  draftTradeAccept: ReadingProvider['draftTradeAccept'] = desktopTrades;
  checkLineup: ReadingProvider['checkLineup'] = desktopLineup;
  draftLineup: ReadingProvider['draftLineup'] = desktopLineup;
  handOff: ReadingProvider['handOff'] = desktopLineup;
  onMovesChange(_listener: () => void): () => void {
    return () => {};
  }
  async mflKey(): Promise<MFLKeyStatus> {
    return { state: 'absent', league: '', season: 0, detail: 'Connecting MFL needs the desktop app' };
  }
  async connectMFL(_key: string): Promise<MFLKeyStatus> {
    throw new Error('Connecting MFL needs the desktop app');
  }
  async forgetMFL(): Promise<MFLKeyStatus> {
    throw new Error('Connecting MFL needs the desktop app');
  }
  onClockChange(listener: () => void): () => void {
    this.clockListeners.add(listener);
    return () => { this.clockListeners.delete(listener); };
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
      if (!this.baseClock) this.baseClock = parseClock(fixture.default);
      return this.heldClock ?? this.baseClock;
    } catch (cause) {
      return failedClock(cause);
    }
  }

  async snapshot(): Promise<Snapshot> {
    const [{ parseSnapshot }, fixture] = await Promise.all([
      import('./parse'), import('./fixtures/snapshot.json'),
    ]);
    return parseSnapshot(fixture.default);
  }
}

function failedClock(cause: unknown): Sourced<ClockReading> {
  return {
    value: { season: 0, phase: 'OFFSEASON', deadlines: [], windows: [] },
    provenance: {
      source: 'phase-log+commissioner-calendar', kind: 'fixture',
      freshness: { state: 'fail', fetchedAt: '', note: `Clock fixture failed: ${String(cause)}` },
    },
  };
}

const fixtureProvider = new FixtureProvider();
export function selectProvider(): Provider {
  return fixtureProvider;
}
