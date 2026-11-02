import { describe, expect, it } from 'vitest';
import { all, mount, render, settle } from '../../commissioner/commissionerDom';
import { open } from '../../commissioner/commissionerStub';
import { stub } from '../../commissioner/commissionerStub';
import ControlRoom from './ControlRoom';
import { data, league, roomHooks, tab } from './roomTestKit';
import { seenAgo, signInOf } from './roomReads';

roomHooks();
const now = Date.parse('2026-11-02T12:00:00Z');

describe('Control Room · People 2FA and SEEN from the accounts store', () => {
  it('seenAgo says minutes, hours, days, or a dash for never', () => {
    expect(seenAgo(undefined, now)).toBe('—');
    expect(seenAgo('2026-11-02T11:15:00Z', now)).toBe('45m');
    expect(seenAgo('2026-11-02T06:00:00Z', now)).toBe('6h');
    expect(seenAgo('2026-10-27T12:00:00Z', now)).toBe('6d');
  });
  it('a seat shows its owner\'s 2FA and SEEN; an owner with no sign-in shows a dash and nothing', async () => {
    const s = stub({});
    const facts = [{ principal: 'owner-one', authenticator: true, lastSeen: new Date(Date.now() - 3 * 3_600_000).toISOString() }];
    expect(signInOf({ ...data, signIns: facts }, 'owner-one')?.authenticator).toBe(true);
    const host = await mount(render(ControlRoom, { league, initial: { ...data, signIns: facts } }), s.app);
    await open(league);
    await settle();
    await tab(host, 'PEOPLE');
    const rows = all(host, 'tr').filter(r => r.getAttribute('data-team'));
    const cells = (i: number) => all(rows[i]!, 'td').map(td => td.textContent);
    expect(cells(0).slice(3)).toEqual(['3h', '2FA']);
    expect(cells(1).slice(3)).toEqual(['—', '']);
  });
});
