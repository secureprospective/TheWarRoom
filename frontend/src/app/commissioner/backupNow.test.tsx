import { afterEach, describe, expect, it } from 'vitest';
import { act } from 'react';
import {
  all, button, click, control, mount, present, render, settle, type, unmount,
} from './commissionerDom';
import { stub } from './commissionerStub';
import { BackupNow } from './BackupNow';
import { backupState } from './backupState';

afterEach(unmount);

describe('Back up now', () => {
  it('asks for a sign-in before the first backup, then backs up', async () => {
    let hasAccount = false;
    const backups: string[] = [];
    let calls = 0;
    const { app } = stub({
      TargetCommissionerSeat: async () => ({ hasAccount }),
      TargetCommissionerSeatSignIn: async () => {
        hasAccount = true;
        return { hasAccount, username: 'commish' };
      },
      TargetBackupNow: async () => {
        calls++;
        if (!hasAccount) throw new Error('create a sign-in first');
        backups.push('manual-20261024T120000.000000000Z');
        return { set: backups[0], message: `Backup saved as ${backups[0]}.` };
      },
    });
    const host = await mount(render(BackupNow), app);
    await settle();
    expect(present(host, 'Before the first backup, create a sign-in.')).toBe(true);
    expect(all(host, 'button').some(b => b.textContent === 'Back up now')).toBe(false);
    // Asked from the command bar, it is still refused here, before Go is called.
    await act(async () => { await backupState.run(); });
    expect(calls).toBe(0);
    expect(present(host, 'Before the first backup, create a sign-in.')).toBe(true);
    await type(control(host, 'Username'), 'commish');
    await type(control(host, 'Password'), 'league-password');
    await act(async () => {
      (all(host, 'form')[0].props().onSubmit as (e: unknown) => void)({ preventDefault() {} });
    });
    await settle();
    await click(button(host, 'Back up now'));
    await settle();
    expect([calls, backups.length]).toEqual([1, 1]);
    expect(present(host, 'Backup saved as manual-20261024T120000.000000000Z.')).toBe(true);
  });
});
