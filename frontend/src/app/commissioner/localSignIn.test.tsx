import { afterEach, describe, expect, it, vi } from 'vitest';
import { act } from 'react';
import { all, control, mount, present, render, settle, type, unmount } from './commissionerDom';
import { stub } from './commissionerStub';
import { commands } from '../commands/registry';
import { LocalSignIn } from './LocalSignIn';
import { leagueState } from './proxy';

afterEach(async () => { await unmount(); vi.restoreAllMocks(); });

const submit = (host: Parameters<typeof all>[0]) => act(async () => {
  (all(host, 'form')[0].props().onSubmit as (e: unknown) => void)({ preventDefault() {} });
});

describe('League sign-in', () => {
  it('creates the first sign-in and then says this computer is linked', async () => {
    const sent: string[][] = [];
    const { app } = stub({
      TargetCommissionerSeat: async () => ({ hasAccount: false }),
      TargetCommissionerSeatSignIn: async (...args) => {
        sent.push(args);
        return { hasAccount: true, username: 'commish' };
      },
    });
    const changeSeat = vi.spyOn(leagueState, 'changeSeat').mockResolvedValue();
    const host = await mount(render(LocalSignIn), app);
    await settle();
    expect(present(host, 'Keep your league if you change computers')).toBe(true);
    await type(control(host, 'Username'), 'Commish');
    await type(control(host, 'Password'), 'league-password');
    await submit(host);
    await settle();
    expect(sent).toEqual([['Commish', 'league-password']]);
    // An accepted sign-in is a new seat: the old seat's readings end, then the league is reread.
    expect(changeSeat).toHaveBeenCalledTimes(1);
    expect(present(host, 'This computer is signed in as commish.')).toBe(true);
  });
  it('asks a restored copy for the original sign-in and clears a refused password', async () => {
    const { app } = stub({
      TargetCommissionerSeatSignIn: async () => {
        throw new Error('incorrect username or password');
      },
      TargetCommissionerSeat: async () => ({ hasAccount: true }),
    });
    const host = await mount(render(LocalSignIn), app);
    await settle();
    expect(present(host, 'Restored from another computer?')).toBe(true);
    await type(control(host, 'Username'), 'commish');
    await type(control(host, 'Password'), 'guessed');
    await submit(host);
    await settle();
    expect(present(host, 'incorrect username or password')).toBe(true);
    expect(control(host, 'Password').props().value).toBe('');
    expect(commands.history().at(-1)).toMatchObject({ status: 'failed',
      reason: 'incorrect username or password', args: { screen: 'local-sign-in' } });
    expect(JSON.stringify(commands.history())).not.toContain('guessed');
  });
});
