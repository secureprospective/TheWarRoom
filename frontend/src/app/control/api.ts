import type { control } from '../../../wailsjs/go/models';
import type * as Bindings from '../../../wailsjs/go/control/API';
import {
  DeleteMFLKey, GetParams, MFLKeyStatus, ResetParam, SetMFLKey, SetParam, TargetAlerts, TargetSetWindow,
  TargetShellStatus,
} from '../../../wailsjs/go/control/API';

// The only file that touches the control bindings. The Control Room reads say when the desktop
// app is missing; the key and parameter calls are guarded by their callers, as before.
function bindings(): typeof Bindings {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('Control Room requires the desktop app; no browser fixture is substituted.');
  return (window as unknown as { go: { control: { API: typeof Bindings } } }).go.control.API;
}

export const controlAPI = {
  settings: async (): Promise<unknown> => bindings().TargetLeagueSettings(),
  bylaws: async (): Promise<unknown> => bindings().TargetBylaws(),
  accounting: async (): Promise<unknown> => bindings().TargetAccounting(),
  news: async (): Promise<unknown> => bindings().TargetSiteNews(),
  openAccounting: async (): Promise<void> => bindings().TargetOpenMFLAccounting(),
};

export const readMFLKey = MFLKeyStatus;
export const setMFLKey = SetMFLKey;
export const deleteMFLKey = DeleteMFLKey;
export const getParams = GetParams;
export const setParam = SetParam;
export const resetParam = ResetParam;
// The alert tray, the status strip and league dates; their callers keep their own desktop checks.
export const readAlerts = TargetAlerts;
export const readShellStatus = TargetShellStatus;
export const setLeagueWindow = TargetSetWindow;

// ParamsResult is the admin parameters payload, as the Data and sources screen reads it.
export type ParamsResult = control.ParamsResult;
