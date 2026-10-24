import { describe, expect, it } from 'vitest';
import { OFFLINE, plainBindings, plainFailure } from './plainFailure';

describe('plain failures', () => {
  it('shows the offline sentence without the Go context in front of it', async () => {
    const other = new Error('league: unexpected export shape');
    const go = { control: { API: {
      TargetSiteNews: async () => { throw new Error(`site news: startup: ${OFFLINE}`); },
      TargetBylaws: async () => { throw other; },
      TargetAccounting: async () => 'report',
    } } };
    plainBindings(go);
    await expect(go.control.API.TargetSiteNews()).rejects.toThrow(new Error(OFFLINE));
    await expect(go.control.API.TargetBylaws()).rejects.toBe(other);
    await expect(go.control.API.TargetAccounting()).resolves.toBe('report');
    expect(plainFailure('incorrect username or password')).toBe('incorrect username or password');
  });
});
