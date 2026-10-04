import { create } from 'zustand';
import { GetSignals, LoadSignals } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Signals store: source health, freshness and coverage (read), and a load of every due file
// (load). The app also loads signals in the background at launch.
interface SignalsState {
  view: main.SignalsView | null;
  loading: boolean;
  error: string;
  read: () => Promise<void>;
  load: () => Promise<void>;
}

export const useSignalsStore = create<SignalsState>((set, get) => ({
  view: null,
  loading: false,
  error: '',
  read: async () => {
    try {
      const view = await GetSignals();
      set({ view, error: view.ok ? '' : view.error });
    } catch (e) {
      set({ error: String(e) });
    }
  },
  load: async () => {
    set({ loading: true, error: '' });
    try {
      const rep = await LoadSignals();
      set({ loading: false, error: rep.ok ? '' : rep.error });
      await get().read();
    } catch (e) {
      set({ loading: false, error: String(e) });
    }
  },
}));
