import { create } from 'zustand';
import { GetCrosswalkReport, LoadCrosswalk } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Crosswalk store: the latest DynastyProcess load's match report. read() shows what this launch
// already loaded (ScoreLeague loads it too); load() fetches DynastyProcess again.
interface CrosswalkState {
  report: main.CrosswalkReport | null;
  loading: boolean;
  error: string;
  read: () => Promise<void>;
  load: () => Promise<void>;
}

export const useCrosswalkStore = create<CrosswalkState>((set) => ({
  report: null,
  loading: false,
  error: '',
  read: async () => {
    try {
      set({ report: await GetCrosswalkReport() });
    } catch (e) {
      set({ error: String(e) });
    }
  },
  load: async () => {
    set({ loading: true, error: '' });
    try {
      const report = await LoadCrosswalk();
      set({ report, loading: false, error: report.ok ? '' : report.error });
    } catch (e) {
      set({ loading: false, error: String(e) });
    }
  },
}));
