import { describe, expect, it } from 'vitest';
import fixture from './fixtures/snapshot.json';
import { parseSnapshot } from './parse';

describe('snapshot boundary', () => {
  it('parses the real fixture with provenance on every section', () => {
    const parsed = parseSnapshot(fixture);
    for (const section of Object.values(parsed)) {
      expect(section.provenance.kind).toBe('fixture');
      expect(section.provenance.source).not.toBe('');
    }
    expect(parsed.franchises.value.length).toBeGreaterThan(0);
    expect(parsed.players.value.some((p) => p.name && p.team)).toBe(true);
    expect(parsed.players.value.some((p) => p.id.startsWith('0'))).toBe(true);
    expect(parsed.players.value.every((p) => typeof p.id === 'string')).toBe(true);
  });

  it('keeps canonical leading-zero player and franchise ids', () => {
    const copy = JSON.parse(JSON.stringify(fixture));
    copy.players.value[0].id = '0531';
    copy.franchises.value[0].id = '0001';
    const parsed = parseSnapshot(copy);
    expect(parsed.players.value[0].id).toBe('0531');
    expect(parsed.franchises.value[0].id).toBe('0001');
  });

  it.each([
    [
      'snapshot.players.value[0].id: expected string',
      (p: typeof fixture) => {
        (p.players.value[0] as unknown as { id: number }).id = 531;
      },
    ],
    [
      'snapshot.rosters.value: expected array',
      (p: typeof fixture) => {
        (p.rosters as unknown as { value: null }).value = null;
      },
    ],
    [
      'snapshot.league.provenance.freshness.state: expected live | stale | fail',
      (p: typeof fixture) => {
        p.league.provenance.freshness.state = 'fiction';
      },
    ],
    [
      'snapshot.players.value[0].birthdate: expected safe integer >= -9007199254740991',
      (p: typeof fixture) => {
        (p.players.value[0] as unknown as { birthdate: string }).birthdate = 'yesterday';
      },
    ],
    [
      'snapshot.players.value[0].isRookie: expected boolean',
      (p: typeof fixture) => {
        (p.players.value[0] as unknown as { isRookie: string }).isRookie = 'yes';
      },
    ],
    [
      'snapshot.players.value[0].position: expected QB | RB | WR | TE | K | DT | DE | LB | CB | S | FLAG',
      (p: typeof fixture) => {
        (p.players.value[0] as unknown as { position: string }).position = 'invented';
      },
    ],
  ])('rejects deliberate corruption: %s', (message, corrupt) => {
    const copy = JSON.parse(JSON.stringify(fixture));
    corrupt(copy);
    expect(() => parseSnapshot(copy)).toThrowError(message);
  });

  it('rejects unexpected fields and absent provenance precisely', () => {
    const copy = JSON.parse(JSON.stringify(fixture));
    copy.players.value[0].invented = true;
    expect(() => parseSnapshot(copy)).toThrowError(
      'snapshot.players.value[0].invented: unexpected field',
    );
    delete copy.players.value[0].invented;
    delete copy.players.provenance;
    expect(() => parseSnapshot(copy)).toThrowError(
      'snapshot.players.provenance: expected object',
    );
  });
});
