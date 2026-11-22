import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { controlAPI, readAlerts, readShellStatus, setLeagueWindow } from './api';

let before: PropertyDescriptor | undefined;
beforeEach(() => { before = Object.getOwnPropertyDescriptor(globalThis, 'window'); });
afterEach(() => {
  if (before) Object.defineProperty(globalThis, 'window', before);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('control adapter', () => {
  it('reaches the alert tray, status strip and league dates through window.go.control.API', async () => {
    const api = {
      TargetAlerts: vi.fn().mockResolvedValue({ alerts: [], unavailable: [] }),
      TargetShellStatus: vi.fn().mockResolvedValue({ health: 'ok', state: 'live', gameday: 'Week 5' }),
      TargetSetWindow: vi.fn().mockResolvedValue(undefined),
      TargetBylaws: vi.fn().mockResolvedValue([]),
    };
    vi.stubGlobal('window', { go: { control: { API: api } } });
    await expect(readAlerts('0001')).resolves.toEqual({ alerts: [], unavailable: [] });
    expect(api.TargetAlerts).toHaveBeenCalledWith('0001');
    await expect(readShellStatus()).resolves.toMatchObject({ gameday: 'Week 5' });
    await setLeagueWindow('cut_day', '2026-11-01T18:00:00Z');
    expect(api.TargetSetWindow).toHaveBeenCalledWith('cut_day', '2026-11-01T18:00:00Z');
    await controlAPI.bylaws();
    expect(api.TargetBylaws).toHaveBeenCalledTimes(1);
  });

  it('says the Control Room needs the desktop app when there is no window.go', async () => {
    vi.stubGlobal('window', {});
    await expect(controlAPI.settings()).rejects.toThrow('Control Room requires the desktop app');
  });
});
