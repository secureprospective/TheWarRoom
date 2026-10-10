import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import fixture from '../data/fixtures/snapshot.json';
import { parseEnvelopeDemo, parseSnapshot } from '../data/parse';
import { FixtureProvider, LiveProvider } from '../data/provider';
import { createCommands } from './registry';

const receipt = parseEnvelopeDemo(JSON.parse(readFileSync(
  'src/app/data/fixtures/envelope-demo.json', 'utf8',
))).receipt;
const snapshot = parseSnapshot(fixture);
const subject = { kind: 'player' as const, id: '11675', franchiseId: '0001' };
function setup(provider = new LiveProvider()) {
  const c = createCommands(() => undefined, provider);
  c.loadSnapshot(snapshot);
  c.dispatch('franchise.set', { franchiseId: subject.franchiseId });
  return c;
}
async function waitFor(assertion: () => void) {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      assertion();
      return;
    } catch (cause) {
      if (attempt === 99) throw cause;
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
  }
}
async function settled(c: ReturnType<typeof createCommands>) {
  await waitFor(() => expect(c.readMoves().drafting['0001:11675']).toBe(false));
}

describe.each([
  ['roster.ir', 'draftIR', 'Draft IR placement'],
  ['roster.taxi', 'draftTaxi', 'Draft move to taxi squad'],
] as const)('%s', (intent, method, label) => {
  const draftedReceipt = {
    ...receipt, spec: { ...receipt.spec, intent, expected: {
      ...receipt.spec.expected, rosterStatus: intent === 'roster.ir' ? 'IR' as const : 'TAXI_SQUAD' as const,
    } },
  };
  it('stores the receipt, clears pending and refuses a second fire', async () => {
    const provider = new LiveProvider();
    let resolve!: (value: typeof draftedReceipt) => void;
    const draft = vi.spyOn(provider, method).mockImplementation(() =>
      new Promise<typeof draftedReceipt>((done) => {
        resolve = done;
      }));
    const c = setup(provider);
    expect(c.registry[intent]).toMatchObject({
      label, gravity: 'G2', undo: 'reversible',
      roles: ['gm'], args: ['subject'],
    });
    const shellNotifications = vi.fn();
    c.subscribe(shellNotifications);
    c.dispatch(intent, { subject });
    expect(c.readMoves().drafting['0001:11675']).toBe(true);
    expect(c.draftReason(subject)).toBe('Drafting…');
    c.dispatch(intent, { subject });
    await waitFor(() => expect(draft).toHaveBeenCalledTimes(1));
    resolve(draftedReceipt);
    await settled(c);
    expect(c.readMoves().moves).toEqual([draftedReceipt]);
    expect(c.readMoves().draftErrors['0001:11675']).toBe('');
    expect(shellNotifications).not.toHaveBeenCalled();
  });
  it('stores rejection text for this subject without a receipt', async () => {
    const provider = new LiveProvider();
    vi.spyOn(provider, method).mockRejectedValue(new Error('desktop disconnected'));
    const c = setup(provider);
    c.dispatch(intent, { subject });
    await settled(c);
    expect(c.readMoves().draftErrors['0001:11675']).toContain('desktop disconnected');
    expect(c.readMoves().moves).toEqual([]);
  });
  it('denies another franchise before any call', () => {
    const provider = new LiveProvider();
    const draft = vi.spyOn(provider, method);
    const c = setup(provider);
    const other = { ...subject, franchiseId: '0002' };
    expect(c.draftReason(other)).toBe('Only players on my franchise');
    c.dispatch(intent, { subject: other });
    expect(draft).not.toHaveBeenCalled();
    expect(c.readMoves().drafting).toEqual({});
  });
  it('leaves roster membership checks to Go and displays its receipt', async () => {
    const provider = new LiveProvider();
    const blocked = {
      ...draftedReceipt, state: 'blocked' as const,
      audit: [{ ...draftedReceipt.audit[0], to: 'blocked' as const, note: 'Player is not on roster' }],
    };
    const draft = vi.spyOn(provider, method).mockResolvedValue(blocked);
    const c = setup(provider);
    const other = { ...subject, id: '99999' };
    expect(c.draftReason(other)).toBeUndefined();
    c.dispatch(intent, { subject: other });
    await waitFor(() => expect(c.readMoves().drafting['0001:99999']).toBe(false));
    expect(draft).toHaveBeenCalledWith('0001', '99999');
    expect(c.readMoves().moves).toEqual([blocked]);
  });
  it('declares fixture drafting unavailable and never starts pending', () => {
    const c = createCommands(() => undefined, new FixtureProvider());
    c.loadSnapshot(snapshot);
    expect(c.draftReason(subject)).toBe('Drafting needs the desktop app');
    c.dispatch(intent, { subject });
    expect(c.readMoves().drafting).toEqual({});
    expect(c.readMoves().moves).toEqual([]);
  });
  it('merges drafts made during a session load without duplicates', async () => {
    const provider = new LiveProvider();
    let resolve!: (value: typeof draftedReceipt[]) => void;
    vi.spyOn(provider, 'moves').mockImplementation(() => new Promise<typeof draftedReceipt[]>((done) => {
      resolve = done;
    }));
    vi.spyOn(provider, method).mockResolvedValue(draftedReceipt);
    const c = setup(provider);
    const loading = c.loadMoves('0001');
    await waitFor(() => expect(resolve).toBeTypeOf('function'));
    c.dispatch(intent, { subject });
    await settled(c);
    resolve([draftedReceipt]);
    await loading;
    expect(c.readMoves().moves).toEqual([draftedReceipt]);
    expect(c.readMoves().movesLoading['0001']).toBe(false);
  });
  it('keeps load errors visible rather than pretending the list is empty', async () => {
    const provider = new LiveProvider();
    vi.spyOn(provider, 'moves').mockRejectedValue(new Error('move log offline'));
    const c = setup(provider);
    await c.loadMoves('0001');
    expect(c.readMoves().movesErrors['0001']).toContain('move log offline');
  });
});

describe('receipt ordering at storage', () => {
  it('orders fractional Go timestamps numerically for both loads and drafts', async () => {
    const at = (id: string, timestamp: string) => ({
      ...receipt, correlationId: id,
      audit: receipt.audit.map((entry) => ({ ...entry, at: timestamp })),
    });
    const whole = at('whole', '2026-10-06T00:00:00Z');
    const fractional = at('fractional', '2026-10-06T00:00:00.5Z');
    const earlier = at('earlier', '2026-10-05T23:59:59Z');
    const provider = new LiveProvider();
    vi.spyOn(provider, 'moves').mockResolvedValue([whole, fractional]);
    vi.spyOn(provider, 'draftIR').mockResolvedValue(earlier);
    const c = setup(provider);
    await c.loadMoves('0001');
    expect(c.readMoves().moves).toEqual([fractional, whole]);
    c.dispatch('roster.ir', { subject });
    await settled(c);
    expect(c.readMoves().moves).toEqual([fractional, whole, earlier]);
  });
});
