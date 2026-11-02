import { describe, expect, it } from 'vitest';
import { all, button, click, present, settle } from '../../commissioner/commissionerDom';
import { region, room, roomHooks } from './roomTestKit';


roomHooks();

describe('Control Room · Jobs', () => {
  it('lists Go’s jobs by state and opens a job’s first form from OPEN', async () => {
    const { host } = await room();
    expect(present(host, 'NEEDS YOU · 3')).toBe(true);
    expect(present(host, 'BLOCKED · 7')).toBe(true);
    await click(all(host, 'div').find(d => d.getAttribute('data-job') === 'setup')!);
    await click(button(host, 'OPEN'));
    await settle();
    expect(region(host, 'Generate a schedule')).toBeTruthy();
  });
  it('opens the stat queue card with the queued change and runs approve through its form', async () => {
    const { host } = await room();
    await click(all(host, 'button').find(b => b.getAttribute('data-launch') === 'stat')!);
    expect(present(host, 'Stat corrections · 1 waiting')).toBe(true);
    expect(present(host, 'J. Kincaid')).toBe(true);
    await click(button(host, 'APPROVE…'));
    await settle();
    expect(region(host, 'Accept a statistics change')).toBeTruthy();
  });
});
