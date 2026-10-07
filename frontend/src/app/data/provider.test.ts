import { afterEach, describe, expect, it, vi } from 'vitest';
import fixture from './fixtures/snapshot.json';
import clockFixture from './fixtures/clock.json';
import { parseEnvelopeDemo } from './parse';
import { readFileSync } from 'node:fs';
import { FixtureProvider, LiveProvider, selectProvider } from './provider';

const originalWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
afterEach(() => {
  if (originalWindow) Object.defineProperty(globalThis, 'window', originalWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('provider selection', () => {
  it('chooses fixtures without Wails and returns parsed fixture provenance', async () => {
    vi.stubGlobal('window', {});
    const provider = selectProvider();
    expect(provider).toBeInstanceOf(FixtureProvider);
    const parsed = await provider.snapshot();
    for (const section of Object.values(parsed))
      expect(section.provenance.kind).toBe('fixture');
  });

  it('chooses live when window.go exists and calls the binding', async () => {
    const payload = JSON.parse(JSON.stringify(fixture));
    for (const section of Object.values(payload) as { provenance: { kind: string } }[])
      section.provenance.kind = 'live';
    const call = vi.fn().mockResolvedValue(payload);
    vi.stubGlobal('window', { go: { main: { App: { TargetSnapshot: call } } } });
    const provider = selectProvider();
    expect(provider).toBeInstanceOf(LiveProvider);
    expect((await provider.snapshot()).players.provenance.kind).toBe('live');
    expect(call).toHaveBeenCalledTimes(1);
  });

  it.each(['outage', 'corrupt payload'])(
    'never substitutes fixtures for a live %s',
    async (cause) => {
      const call =
        cause === 'outage'
          ? vi.fn().mockRejectedValue(new Error('MFL offline'))
          : vi.fn().mockResolvedValue({ players: [] });
      vi.stubGlobal('window', { go: { main: { App: { TargetSnapshot: call } } } });
      const parsed = await selectProvider().snapshot();
      for (const section of Object.values(parsed)) {
        expect(section.provenance.kind).toBe('live');
        expect(section.provenance.freshness.state).toBe('fail');
      }
      expect(parsed.players.value).toEqual([]);
      expect(parsed.franchises.value).toEqual([]);
      expect(parsed.league.value).toEqual({ season: 0, franchiseCount: 0 });
      expect(call).toHaveBeenCalledTimes(1);
    },
  );
});


describe('clock provider boundary', () => {
  it('loads and parses the real clock fixture dynamically', async () => {
    const clock = await new FixtureProvider().clock();
    expect(clock.provenance.kind).toBe('fixture');
    expect(clock.value.phase).toBe('OFFSEASON');
    expect(clock.value.deadlines).toEqual([]);
    expect(clock.value.windows).toHaveLength(7);
  });
  it('calls TargetClock and parses live provenance without waiting for snapshot', async () => {
    const payload = {
      ...clockFixture, provenance: { ...clockFixture.provenance, kind: 'live' },
    };
    const clock = vi.fn().mockResolvedValue(payload);
    const snapshot = vi.fn().mockImplementation(() => new Promise(() => undefined));
    vi.stubGlobal('window', { go: { main: { App: { TargetClock: clock, TargetSnapshot: snapshot } } } });
    const provider = selectProvider();
    void provider.snapshot();
    expect((await provider.clock()).provenance.kind).toBe('live');
    expect(clock).toHaveBeenCalledTimes(1);
    expect(snapshot).toHaveBeenCalledTimes(1);
  });
  it.each(['outage', 'corrupt payload', 'runtime absent'])(
    'shows a fail reading, never fixtures, on %s', async (cause) => {
      const call = cause === 'outage'
        ? vi.fn().mockRejectedValue(new Error('clock offline'))
        : vi.fn().mockResolvedValue({ value: { phase: 'SPRING' } });
      vi.stubGlobal('window', cause === 'runtime absent' ? {} : {
        go: { main: { App: { TargetClock: call } } },
      });
      const reading = await new LiveProvider().clock();
      expect(reading.provenance.kind).toBe('live');
      expect(reading.provenance.freshness.state).toBe('fail');
      expect(reading.provenance.freshness.note).toContain('TargetClock failed:');
      const error = cause === 'outage' ? 'clock offline'
        : cause === 'runtime absent' ? 'Wails runtime absent' : 'clock.';
      expect(reading.provenance.freshness.note).toContain(error);
      expect(reading.value.deadlines).toEqual([]);
      expect(reading.value.windows).toEqual([]);
    },
  );
});

describe('moves provider boundary', () => {
  it('calls and parses both generated bindings, preserving string player ids', async () => {
    const receipt = parseEnvelopeDemo(JSON.parse(readFileSync(
      'src/app/data/fixtures/envelope-demo.json', 'utf8',
    ))).receipt;
    const draft = vi.fn().mockResolvedValue(receipt);
    const moves = vi.fn().mockResolvedValue([receipt]);
    vi.stubGlobal('window', { go: { main: { App: { TargetDraftIR: draft, TargetMoves: moves } } } });
    const provider = new LiveProvider();
    expect(await provider.draftIR('0001', '11675')).toEqual(receipt);
    expect(await provider.moves('0001')).toEqual([receipt]);
    expect(draft).toHaveBeenCalledWith('0001', '11675');
    expect(moves).toHaveBeenCalledWith('0001');
  });
  it('propagates rejected calls and malformed receipts, never an empty list', async () => {
    const draft = vi.fn().mockRejectedValue(new Error('draft binding down'));
    const moves = vi.fn().mockRejectedValue(new Error('moves binding down'));
    vi.stubGlobal('window', { go: { main: { App: { TargetDraftIR: draft, TargetMoves: moves } } } });
    const provider = new LiveProvider();
    await expect(provider.draftIR('0001', '11675')).rejects.toThrow('draft binding down');
    await expect(provider.moves('0001')).rejects.toThrow('moves binding down');
    moves.mockResolvedValue([{}]);
    await expect(provider.moves('0001')).rejects.toThrow('receipt');
    moves.mockResolvedValue(null);
    await expect(provider.moves('0001')).rejects.toThrow('TargetMoves: expected array');
  });
});

describe('parallel parser and binding startup', () => {
  it('starts every live binding synchronously, before the parser import can settle', async () => {
    const receipt = parseEnvelopeDemo(JSON.parse(readFileSync(
      'src/app/data/fixtures/envelope-demo.json', 'utf8',
    ))).receipt;
    const snapshot = vi.fn().mockResolvedValue(fixture);
    const clock = vi.fn().mockResolvedValue(clockFixture);
    const draft = vi.fn().mockResolvedValue(receipt);
    const moves = vi.fn().mockResolvedValue([receipt]);
    vi.stubGlobal('window', { go: { main: { App: {
      TargetSnapshot: snapshot, TargetClock: clock, TargetDraftIR: draft, TargetMoves: moves,
    } } } });
    const provider = new LiveProvider();
    const pending = [
      provider.snapshot(), provider.clock(), provider.draftIR('0001', '11675'),
      provider.moves('0001'),
    ];
    for (const binding of [snapshot, clock, draft, moves]) {
      expect(binding).toHaveBeenCalledTimes(1);
    }
    await Promise.all(pending);
  });
});
