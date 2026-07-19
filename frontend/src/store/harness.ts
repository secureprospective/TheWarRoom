import { create } from 'zustand';
import {
  ScoreRookies,
  RunValidationSuite,
  GetParams,
  SetParam,
  ScoreLeague,
  GetRankings,
  GetPowerRankings,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// powerReqSeq monotonically tags each GetPowerRankings call so a slow earlier
// response (e.g. weight 0.6 dispatched, then 0.8 dispatched and returning first)
// cannot overwrite a newer one's rows. Only the latest request commits.
let powerReqSeq = 0;

// Harness store. Per WF5 every IPC call lives here, never in a component: components read
// a slice and dispatch an action. This is the testing sandbox's single backend gateway.
interface HarnessState {
  rookies: main.RookiesResult | null;
  validation: main.ValidationResult | null;
  params: main.ParamsResult | null;
  rankings: main.RankingsResult | null;
  powerRankings: main.PowerRankingsResult | null;
  powerWeight: number; // scouting weight applied to the 60/40 blend (default 0.60)
  powerAgg: string; // scouting aggregation: 'sum' | 'topn'
  powerLoading: boolean;
  scoreReport: main.ScoreLeagueResult | null;
  scoring: boolean;
  loading: boolean;
  error: string;
  loadAll: () => Promise<void>;
  setParam: (key: string, value: number) => Promise<void>;
  loadRankings: () => Promise<void>;
  loadPowerRankings: (weight: number, aggMode: string) => Promise<void>;
  scoreLeague: () => Promise<void>;
}

export const useHarnessStore = create<HarnessState>((set, get) => ({
  rookies: null,
  validation: null,
  params: null,
  rankings: null,
  powerRankings: null,
  powerWeight: 0.6,
  powerAgg: 'sum',
  powerLoading: false,
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

  // loadPowerRankings pulls the M2 blended board for the given scouting weight. It
  // fetches live MFL standings server-side, so it is the one board with a real
  // network dependency — powerLoading gates the UI while it runs. The backend echoes
  // the CLAMPED weight it actually applied; we sync powerWeight to it so the slider
  // never drifts from the rows.
  loadPowerRankings: async (weight, aggMode) => {
    const seq = ++powerReqSeq;
    set({ powerLoading: true, error: '' });
    try {
      const powerRankings = await GetPowerRankings(weight, aggMode);
      if (seq !== powerReqSeq) return; // a newer request superseded this one — drop it
      set({
        powerRankings,
        powerWeight: powerRankings.ok ? powerRankings.weight : weight,
        powerAgg: powerRankings.ok ? powerRankings.aggMode : aggMode,
        powerLoading: false,
        error: powerRankings.ok ? '' : powerRankings.error,
      });
    } catch (e) {
      if (seq !== powerReqSeq) return;
      set({ powerLoading: false, error: String(e) });
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
