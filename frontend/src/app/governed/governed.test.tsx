import { afterEach, describe, expect, it, vi } from 'vitest';
import { all, button, click, mount, render, settle, unmount, type Element }
  from '../commissioner/commissionerDom';
import { parseGoverned } from './governed';
import { GovernedGlyph } from './GovernedGlyph';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

const governed = { entry: 'act.1', who: 'Local operator', when: '2027-05-03T04:00:00Z',
  change: 'Active roster size: 18 to 80 → 20 to 70, in force now' };
const inZone = (timeZone: string) => {
  const resolved = new Intl.DateTimeFormat().resolvedOptions();
  vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({ ...resolved, timeZone });
};
const marker = (host: Element) => button(host, 'Who set roster size, when, and with which change');
const detail = (host: Element) => all(host, 'dl')[0];

describe('The governed glyph', () => {
  it('opens to who set the value, when in the viewer’s own time, and with which change', async () => {
    inZone('America/Los_Angeles');
    const host = await mount(render(GovernedGlyph, { governed, subject: 'roster size' }));
    expect(detail(host)).toBeUndefined();
    expect(marker(host).getAttribute('aria-expanded')).toBe('false');
    await click(marker(host));
    await settle();
    const local = new Date(governed.when).toLocaleString(undefined, {
      weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
      timeZoneName: 'short', year: 'numeric', timeZone: 'America/Los_Angeles',
    });
    expect(marker(host).getAttribute('aria-expanded')).toBe('true');
    expect(detail(host).textContent).toContain('Set byLocal operator');
    expect(detail(host).textContent).toContain(local);
    expect(detail(host).textContent).toContain(`With this change${governed.change}`);
    expect(local).toContain('PDT');
    expect(local).not.toContain('UTC');
    await click(marker(host));
    expect(detail(host)).toBeUndefined();
  });
  it('reads the same instant in another viewer’s zone, never a fixed one', async () => {
    inZone('Asia/Tokyo');
    const host = await mount(render(GovernedGlyph, { governed, subject: 'roster size' }));
    await click(marker(host));
    await settle();
    expect(detail(host).textContent).toContain('May 3');
    expect(detail(host).textContent).toContain('1:00 PM');
  });
  it('is a labelled control with no census code or gravity word on it', async () => {
    const host = await mount(render(GovernedGlyph, { governed, subject: 'roster size' }));
    await click(marker(host));
    expect(host.textContent).not.toMatch(/CM-\d|G[123]\b/);
    expect(marker(host).getAttribute('aria-controls')).toBe(detail(host).getAttribute('id'));
  });
  it('refuses a reading that is not who, when and which change', () => {
    expect(parseGoverned(governed, 'x')).toEqual(governed);
    expect(() => parseGoverned({ ...governed, extra: 1 }, 'x')).toThrow('unexpected field');
    expect(() => parseGoverned({ ...governed, when: 'yesterday' }, 'x')).toThrow();
    expect(() => parseGoverned({ who: 'a' }, 'x')).toThrow();
  });
});
