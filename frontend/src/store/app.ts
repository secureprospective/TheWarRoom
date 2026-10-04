import { create } from 'zustand';
import {
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

// The app store. Every IPC call lives here, never in a component: components read a slice and
// dispatch an action. It is the frontend's single backend gateway.
interface AppState {
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
  loadParams: () => Promise<void>;
  setParam: (key: string, position: string, value: number) => Promise<void>;
  loadRankings: () => Promise<void>;
  loadPowerRankings: (weight: number, aggMode: string) => Promise<void>;
  scoreLeague: () => Promise<void>;
}

export const useAppStore = create<AppState>((set, get) => ({
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

  // loadParams reads every calibration parameter with the value in effect. Called on mount and
  // after a param change.
  loadParams: async () => {
    set({ loading: true, error: '' });
    try {
      const params = await GetParams();
      set({ params, loading: false });
    } catch (e) {
      set({ loading: false, error: String(e) });
    }
  },

  // setParam writes an admin override, then re-reads the params so the console shows the value
  // in effect. The next Score League scores with it.
  setParam: async (key, position, value) => {
    set({ error: '' });
    const res = await SetParam(key, position, value);
    if (!res.ok) {
      set({ error: res.error });
      return;
    }
    await get().loadParams();
  },

  // loadRankings reads the latest board run back from history. Read-only — empty
  // rows means ScoreLeague has not run this season yet.
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

  // scoreLeague runs the M1 orchestrator (load the labeled YTD proxy into history,
  // score all 32 rosters, write a board run), then re-reads the board. The report
  // (the run, exclusions with reasons, zero-base count, unchanged) is kept
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
