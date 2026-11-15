import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

// The locked mock's table rule `.ct{width:100%}` also styles the panel captions (class ct), which is why
// they read left in the mock. The room carries that width on its captions.
describe('Control Room · caption alignment', () => {
  it("captions take the mock's width:100% so they sit left, as the mock renders them", () => {
    const mock = readFileSync(
      new URL('../../../../../docs/ui/mocks/control-room-mock-v1.html', import.meta.url),
      'utf8',
    );
    const css = readFileSync(new URL('./controlRoom.css', import.meta.url), 'utf8');
    expect(mock).toMatch(/\n\.ct\{width:100%;/);
    expect(css).toContain('.crx .cr-ph .ct{width:100%}');
  });
});

describe('Control Room · ledger sentences', () => {
  it('says machine timestamps as a date and time, and leaves other words alone', async () => {
    const { plainStamps, when } = await import('./room');
    const iso = '2026-11-11T15:28:08Z';
    expect(plainStamps(`Removed a player; effective ${iso}`))
      .toBe(`Removed a player; effective ${when(iso)}`);
    expect(plainStamps('Trade deadline: 24 Nov 2026')).toBe('Trade deadline: 24 Nov 2026');
  });
});
