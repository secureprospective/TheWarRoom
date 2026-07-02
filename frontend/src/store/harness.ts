import { create } from 'zustand';
import {
  ScoreRookies,
  RunValidationSuite,
  GetParams,
  SetParam,
  ScoreLeague,
  GetRankings,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Harness store. Per WF5 every IPC call lives here, never in a component: components read
// a slice and dispatch an action. This is the testing sandbox's single backend gateway.
interface HarnessState {
  rookies: main.RookiesResult | null;
  validation: main.ValidationResult | null;
  params: main.ParamsResult | null;
  rankings: main.RankingsResult | null;
  scoreReport: main.ScoreLeagueResult | null;
  scoring: boolean;
  loading: boolean;
  error: string;
  loadAll: () => Promise<void>;
  setParam: (key: string, value: number) => Promise<void>;
  loadRankings: () => Promise<void>;
  scoreLeague: () => Promise<void>;
}

export const useHarnessStore = create<HarnessState>((set, get) => ({
  rookies: null,
  validation: null,
  params: null,
  rankings: null,
  scoreReport: null,
  scoring: false,
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

  // loadRankings reads the persisted M1 board back from B6. Read-only — empty rows
  // means ScoreLeague has not run for the active config yet.
  loadRankings: async () => {
    set({ error: '' });
    try {
      const rankings = await GetRankings();
      set({ rankings, error: rankings.ok ? '' : rankings.error });
    } catch (e) {
      set({ error: String(e) });
    }
  },

  // scoreLeague runs the M1 orchestrator (fetch the labeled YTD proxy, score all 32
  // rosters, persist to B6 stamped with the active config), then re-reads the board.
  // The report (exclusions with reasons, zero-base count, skip-if-present) is kept
  // for display — an invisible exclusion is a silent lie.
  scoreLeague: async () => {
    set({ scoring: true, error: '' });
    try {
      const scoreReport = await ScoreLeague();
      set({ scoreReport, scoring: false, error: scoreReport.ok ? '' : scoreReport.error });
      if (scoreReport.ok) {
        await get().loadRankings();
      }
    } catch (e) {
      set({ scoring: false, error: String(e) });
    }
  },
}));
