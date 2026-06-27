import { create } from 'zustand';
import {
  ScoreRookies,
  RunValidationSuite,
  GetParams,
  SetParam,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Harness store. Per WF5 every IPC call lives here, never in a component: components read
// a slice and dispatch an action. This is the testing sandbox's single backend gateway.
interface HarnessState {
  rookies: main.RookiesResult | null;
  validation: main.ValidationResult | null;
  params: main.ParamsResult | null;
  loading: boolean;
  error: string;
  loadAll: () => Promise<void>;
  setParam: (key: string, value: number) => Promise<void>;
}

export const useHarnessStore = create<HarnessState>((set, get) => ({
  rookies: null,
  validation: null,
  params: null,
  loading: false,
  error: '',

  // loadAll pulls every module's data in parallel. Called on mount and after a param
  // change so the rankings board reflects the new calibration.
  loadAll: async () => {
    set({ loading: true, error: '' });
    try {
      const [rookies, validation, params] = await Promise.all([
        ScoreRookies(),
        RunValidationSuite(),
        GetParams(),
      ]);
      set({ rookies, validation, params, loading: false });
    } catch (e) {
      set({ loading: false, error: String(e) });
    }
  },

  // setParam writes a live admin override then re-pulls so the operator sees the score
  // move — the sandbox's whole point (functional gate).
  setParam: async (key, value) => {
    set({ error: '' });
    const res = await SetParam(key, value);
    if (!res.ok) {
      set({ error: res.error });
      return;
    }
    await get().loadAll();
  },
}));
